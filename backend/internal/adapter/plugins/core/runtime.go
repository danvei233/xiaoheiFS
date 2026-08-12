package plugins

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"xiaoheiplay/internal/pkg/safego"
	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

const defaultStartTimeout = 30 * time.Second

// maxGRPCMessageBytes bounds the host-side gRPC client recv/send sizes. The
// plugin-side server (pkg/pluginsdk) enforces the same limit, so a plugin
// response larger than this is rejected at the source; this closes the
// host-side OOM vector for plugins that do not use the SDK (m040 M-1 residual).
const maxGRPCMessageBytes = 64 << 20 // 64 MiB

// socketRootName is the hidden root under the plugin base dir where go-plugin
// control sockets are placed. The leading dot keeps it out of scanDiskPlugins
// and the bootstrap scan, and its 0700 mode keeps the sockets owner-only.
const socketRootName = ".plugin-sockets"

// boundedGRPCDialOptions caps per-message sizes on the host gRPC client.
var boundedGRPCDialOptions = []grpc.DialOption{
	grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(maxGRPCMessageBytes),
		grpc.MaxCallSendMsgSize(maxGRPCMessageBytes),
	),
}

type Runtime struct {
	baseDir string

	mu       sync.Mutex
	running  map[string]*runningPlugin
	keyLocks map[string]*keyLock
	// lastKill records when a plugin key was torn down by the health loop
	// (process exited or became unresponsive). Start uses it as a cooldown so a
	// broken plugin cannot be respawned in a tight loop (restart storm). Guarded
	// by mu.
	lastKill map[string]time.Time

	// newClient constructs the plugin client. It is a field so tests can
	// substitute a fake client without spawning a real plugin process.
	newClient func(cfg *plugin.ClientConfig) clientHandle

	// heartbeatInterval and healthTimeout bound the health-check loop. They are
	// fields so tests can shorten them; production uses the defaults.
	heartbeatInterval time.Duration
	healthTimeout     time.Duration

	// maxHealthFailures is the number of consecutive failed health checks after
	// which an alive-but-unresponsive plugin is killed and removed. It is a
	// field so tests can lower it; production uses the default.
	maxHealthFailures int
	// healthRestartBackoff is the minimum delay between a health-triggered kill
	// and a fresh spawn for the same plugin key.
	healthRestartBackoff time.Duration
}

// keyLock serializes Start/Stop for a single plugin key so a key can never be
// spawned or torn down twice concurrently (which would leak a plugin process or
// resurrect a disabled instance).
type keyLock struct {
	mu   sync.Mutex
	refs int
}

// clientHandle is the subset of *plugin.Client that Runtime relies on. It lets
// tests substitute a fake client and inspect the ClientConfig.
type clientHandle interface {
	Client() (plugin.ClientProtocol, error)
	Kill()
	Exited() bool
}

type runningPlugin struct {
	mu sync.Mutex

	runtime    *Runtime
	category   string
	pluginID   string
	instanceID string

	client     clientHandle
	core       pluginv1.CoreServiceClient
	sms        pluginv1.SmsServiceClient
	payment    pluginv1.PaymentServiceClient
	kyc        pluginv1.KycServiceClient
	automation pluginv1.AutomationServiceClient
	manifest   *pluginv1.Manifest

	lastHealth time.Time
	health     *pluginv1.HealthCheckResponse
	cancelHB   context.CancelFunc

	// consecutiveHealthFailures counts heartbeat failures in a row. When it
	// reaches the runtime's threshold the plugin is treated as a hung process
	// and torn down so the next EnsureRunning/Start can restart it.
	consecutiveHealthFailures int
}

func NewRuntime(baseDir string) *Runtime {
	return &Runtime{
		baseDir:              baseDir,
		running:              map[string]*runningPlugin{},
		keyLocks:             map[string]*keyLock{},
		lastKill:             map[string]time.Time{},
		newClient:            func(cfg *plugin.ClientConfig) clientHandle { return plugin.NewClient(cfg) },
		heartbeatInterval:    10 * time.Second,
		healthTimeout:        2 * time.Second,
		maxHealthFailures:    3,
		healthRestartBackoff: 5 * time.Second,
	}
}

// unixSocketConfig pins go-plugin control sockets to a dedicated 0700 directory
// under the plugin base dir instead of the shared system TMPDIR, so plugin
// sockets are never reachable through a world-traversable temp location. It
// returns nil when there is no base dir (the caller then uses go-plugin's
// default, which is still a per-plugin 0700 temp dir).
func (r *Runtime) unixSocketConfig() *plugin.UnixSocketConfig {
	if strings.TrimSpace(r.baseDir) == "" {
		return nil
	}
	dir := filepath.Join(r.baseDir, socketRootName)
	_ = os.MkdirAll(dir, 0o700)
	// Enforce the mode even when the directory pre-exists from an earlier run
	// with a looser umask (P2-51): a world-traversable socket dir would let other
	// users reach the control socket despite the mutual-TLS peer auth.
	_ = os.Chmod(dir, 0o700)
	return &plugin.UnixSocketConfig{TempDir: dir}
}

// lockKey returns a function that releases the per-key lock. A plugin key may
// only be started or stopped by one caller at a time; the refcount reaps the
// lock entry once nobody is waiting on it.
func (r *Runtime) lockKey(k string) func() {
	r.mu.Lock()
	kl := r.keyLocks[k]
	if kl == nil {
		kl = &keyLock{}
		r.keyLocks[k] = kl
	}
	kl.refs++
	r.mu.Unlock()

	kl.mu.Lock()
	return func() {
		kl.mu.Unlock()
		r.mu.Lock()
		kl.refs--
		if kl.refs == 0 {
			delete(r.keyLocks, k)
		}
		r.mu.Unlock()
	}
}

func automationFeatureFromString(s string) (pluginv1.AutomationFeature, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "catalog_sync":
		return pluginv1.AutomationFeature_AUTOMATION_FEATURE_CATALOG_SYNC, true
	case "lifecycle":
		return pluginv1.AutomationFeature_AUTOMATION_FEATURE_LIFECYCLE, true
	case "port_mapping":
		return pluginv1.AutomationFeature_AUTOMATION_FEATURE_PORT_MAPPING, true
	case "backup":
		return pluginv1.AutomationFeature_AUTOMATION_FEATURE_BACKUP, true
	case "snapshot":
		return pluginv1.AutomationFeature_AUTOMATION_FEATURE_SNAPSHOT, true
	case "firewall":
		return pluginv1.AutomationFeature_AUTOMATION_FEATURE_FIREWALL, true
	default:
		return pluginv1.AutomationFeature_AUTOMATION_FEATURE_UNSPECIFIED, false
	}
}

func normalizeAutomationNotSupportedReasons(m map[string]string) map[int32]string {
	if len(m) == 0 {
		return nil
	}
	out := map[int32]string{}
	for k, v := range m {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		// allow either numeric enum value ("3") or feature name ("port_mapping")
		if i, err := strconv.Atoi(key); err == nil {
			out[int32(i)] = v
			continue
		}
		if f, ok := automationFeatureFromString(key); ok {
			out[int32(f)] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func validateManifestConsistency(jsonM Manifest, grpcM *pluginv1.Manifest) error {
	if grpcM == nil {
		return fmt.Errorf("invalid manifest")
	}
	jPluginID := strings.TrimSpace(jsonM.PluginID)
	jName := strings.TrimSpace(jsonM.Name)
	jVersion := strings.TrimSpace(jsonM.Version)

	gPluginID := strings.TrimSpace(grpcM.GetPluginId())
	gName := strings.TrimSpace(grpcM.GetName())
	gVersion := strings.TrimSpace(grpcM.GetVersion())

	if gPluginID != jPluginID {
		return fmt.Errorf("manifest mismatch: plugin_id (json=%q grpc=%q)", jPluginID, gPluginID)
	}
	if gName != jName {
		return fmt.Errorf("manifest mismatch: name (json=%q grpc=%q)", jName, gName)
	}
	if gVersion != jVersion {
		return fmt.Errorf("manifest mismatch: version (json=%q grpc=%q)", jVersion, gVersion)
	}

	// sms
	if (jsonM.Capabilities.SMS != nil) != (grpcM.Sms != nil) {
		return fmt.Errorf("manifest mismatch: sms capability presence")
	}
	if jsonM.Capabilities.SMS != nil && grpcM.Sms != nil {
		if grpcM.Sms.GetSend() != jsonM.Capabilities.SMS.Send {
			return fmt.Errorf("manifest mismatch: sms.send")
		}
	}

	// payment
	if (jsonM.Capabilities.Payment != nil) != (grpcM.Payment != nil) {
		return fmt.Errorf("manifest mismatch: payment capability presence")
	}
	if jsonM.Capabilities.Payment != nil && grpcM.Payment != nil {
		jm := append([]string{}, jsonM.Capabilities.Payment.Methods...)
		gm := append([]string{}, grpcM.Payment.GetMethods()...)
		sort.Strings(jm)
		sort.Strings(gm)
		if !slices.Equal(jm, gm) {
			return fmt.Errorf("manifest mismatch: payment.methods")
		}
	}

	// kyc
	if (jsonM.Capabilities.KYC != nil) != (grpcM.Kyc != nil) {
		return fmt.Errorf("manifest mismatch: kyc capability presence")
	}
	if jsonM.Capabilities.KYC != nil && grpcM.Kyc != nil {
		if grpcM.Kyc.GetStart() != jsonM.Capabilities.KYC.Start || grpcM.Kyc.GetQueryResult() != jsonM.Capabilities.KYC.QueryResult {
			return fmt.Errorf("manifest mismatch: kyc flags")
		}
	}

	// automation
	if (jsonM.Capabilities.Automation != nil) != (grpcM.Automation != nil) {
		return fmt.Errorf("manifest mismatch: automation capability presence")
	}
	if jsonM.Capabilities.Automation != nil && grpcM.Automation != nil {
		want := map[pluginv1.AutomationFeature]bool{}
		for _, s := range jsonM.Capabilities.Automation.Features {
			f, ok := automationFeatureFromString(s)
			if !ok {
				// Allow forward-compatible string features that are not present in
				// the current protobuf enum (e.g. ui-only capability tags).
				continue
			}
			want[f] = true
		}
		got := map[pluginv1.AutomationFeature]bool{}
		for _, f := range grpcM.Automation.GetFeatures() {
			got[f] = true
		}
		for f := range got {
			if !want[f] {
				return fmt.Errorf("manifest mismatch: automation.features")
			}
		}
		jReasons := normalizeAutomationNotSupportedReasons(jsonM.Capabilities.Automation.NotSupportedReason)
		gReasons := grpcM.Automation.GetNotSupportedReasons()
		if len(jReasons) != len(gReasons) {
			return fmt.Errorf("manifest mismatch: automation.not_supported_reasons")
		}
		for k, v := range jReasons {
			if gReasons[k] != v {
				return fmt.Errorf("manifest mismatch: automation.not_supported_reasons")
			}
		}
		if grpcM.Automation.GetCatalogReadonly() != jsonM.Capabilities.Automation.CatalogReadonly {
			return fmt.Errorf("manifest mismatch: automation.catalog_readonly")
		}
	}

	return nil
}

func (r *Runtime) key(category, pluginID, instanceID string) string {
	return category + ":" + pluginID + ":" + instanceID
}

// Start spawns (or returns the existing) plugin process for the key. The last
// variadic argument, when non-empty, is an immutable verified snapshot directory
// that the plugin is executed from: the caller (Manager) has verified the digest
// and signature of that exact directory, so a path swap in the live plugin dir
// between verification and spawn cannot change what is executed. When omitted,
// the plugin is run from baseDir/<category>/<pluginID> (used by tests and by
// callers that do not require the verified-snapshot guarantee).
func (r *Runtime) Start(ctx context.Context, category, pluginID, instanceID, configJSON string, runDir ...string) (*pluginv1.Manifest, error) {
	if category == "" || pluginID == "" || instanceID == "" {
		return nil, fmt.Errorf("invalid input")
	}
	k := r.key(category, pluginID, instanceID)

	// Serialize start/stop per key. The check-then-spawn-then-write sequence is
	// atomic with respect to other Start/Stop calls for the same key, so two
	// concurrent Starts can never each spawn a plugin process (one would be
	// orphaned), and an in-flight Start can never resurrect an instance after a
	// concurrent Stop.
	unlock := r.lockKey(k)
	defer unlock()

	r.mu.Lock()
	if existing := r.running[k]; existing != nil {
		if existing.client == nil || !existing.client.Exited() {
			r.mu.Unlock()
			return existing.manifest, nil
		}
		// The plugin process died underneath us: drop the stale entry and
		// restart so callers get a fresh process instead of a dead client.
		delete(r.running, k)
		// Record the tear-down time so the respawn below obeys the same cooldown
		// as a health-loop kill. Otherwise a crash-looping plugin that dies before
		// the heartbeat ever sees it would be respawned on every EnsureRunning
		// call — a spawn storm (each new process runs the on-disk binary).
		r.lastKill[k] = time.Now()
		r.mu.Unlock()
		if existing.cancelHB != nil {
			existing.cancelHB()
		}
		if existing.client != nil {
			existing.client.Kill()
		}
	} else {
		r.mu.Unlock()
	}

	// Circuit breaker: if the health loop recently killed this key (the process
	// exited or became unresponsive), do not respawn within the cooldown window.
	// This prevents a broken plugin from being restarted in a tight loop.
	r.mu.Lock()
	lastKill, recentlyKilled := r.lastKill[k]
	r.mu.Unlock()
	if recentlyKilled {
		backoff := r.healthRestartBackoff
		if backoff <= 0 {
			backoff = 5 * time.Second
		}
		if time.Since(lastKill) < backoff {
			return nil, fmt.Errorf("plugin %s is cooling down after a health failure; retry shortly", k)
		}
	}

	// Run from the verified snapshot when the caller provides one, otherwise from
	// the live plugin dir. Once a snapshot is in use it is immutable (read-only),
	// so a path swap in the live dir after this point cannot change the executed
	// payload.
	pluginDir := filepath.Join(r.baseDir, category, pluginID)
	if len(runDir) > 0 && strings.TrimSpace(runDir[0]) != "" {
		pluginDir = strings.TrimSpace(runDir[0])
	}
	manifestJSON, err := ReadManifest(pluginDir)
	if err != nil {
		return nil, err
	}
	// Runtime backstop: a test_only plugin must never be launched in a
	// production process, even if it slipped past packaging or was installed
	// before this check existed.
	if err := manifestJSON.RejectTestOnly(false); err != nil {
		return nil, err
	}
	entry, err := ResolveEntry(pluginDir, manifestJSON)
	if err != nil {
		if len(entry.SupportedPlatforms) > 0 {
			return nil, fmt.Errorf("%s", "unsupported platform "+entry.Platform+", supported: "+strings.Join(entry.SupportedPlatforms, ", "))
		}
		return nil, err
	}

	cmd := exec.Command(entry.EntryPath)
	cmd.Dir = pluginDir

	client := r.newClient(&plugin.ClientConfig{
		HandshakeConfig: pluginsdk.Handshake,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
		Plugins: map[string]plugin.Plugin{
			pluginsdk.PluginKeyCore:       &pluginsdk.CoreGRPCPlugin{},
			pluginsdk.PluginKeySMS:        &pluginsdk.SmsGRPCPlugin{},
			pluginsdk.PluginKeyPayment:    &pluginsdk.PaymentGRPCPlugin{},
			pluginsdk.PluginKeyKYC:        &pluginsdk.KycGRPCPlugin{},
			pluginsdk.PluginKeyAutomation: &pluginsdk.AutomationGRPCPlugin{},
		},
		Cmd:             cmd,
		StartTimeout:    defaultStartTimeout,
		GRPCDialOptions: boundedGRPCDialOptions,
		// AutoMTLS (P2-51): the host and the spawned plugin exchange one-time
		// certs, so the host only talks to the exact process it launched and that
		// process only accepts the host. Control sockets are pinned to a dedicated
		// 0700 directory instead of the shared system TMPDIR.
		AutoMTLS:         true,
		UnixSocketConfig: r.unixSocketConfig(),
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, err
	}
	rawCore, err := rpcClient.Dispense(pluginsdk.PluginKeyCore)
	if err != nil {
		client.Kill()
		return nil, err
	}
	core, ok := rawCore.(pluginv1.CoreServiceClient)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("invalid core client")
	}

	ctxm, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	manifest, err := core.GetManifest(ctxm, &pluginv1.Empty{})
	if err != nil {
		client.Kill()
		return nil, err
	}
	if manifest.GetPluginId() == "" {
		client.Kill()
		return nil, fmt.Errorf("invalid manifest")
	}
	if err := validateManifestConsistency(manifestJSON, manifest); err != nil {
		client.Kill()
		return nil, err
	}

	var sms pluginv1.SmsServiceClient
	var payment pluginv1.PaymentServiceClient
	var kyc pluginv1.KycServiceClient
	var automation pluginv1.AutomationServiceClient

	if manifest.Sms != nil {
		raw, err := rpcClient.Dispense(pluginsdk.PluginKeySMS)
		if err != nil {
			client.Kill()
			return nil, err
		}
		c, ok := raw.(pluginv1.SmsServiceClient)
		if !ok {
			client.Kill()
			return nil, fmt.Errorf("invalid sms client")
		}
		sms = c
	}
	if manifest.Payment != nil {
		raw, err := rpcClient.Dispense(pluginsdk.PluginKeyPayment)
		if err != nil {
			client.Kill()
			return nil, err
		}
		c, ok := raw.(pluginv1.PaymentServiceClient)
		if !ok {
			client.Kill()
			return nil, fmt.Errorf("invalid payment client")
		}
		payment = c
	}
	if manifest.Kyc != nil {
		raw, err := rpcClient.Dispense(pluginsdk.PluginKeyKYC)
		if err != nil {
			client.Kill()
			return nil, err
		}
		c, ok := raw.(pluginv1.KycServiceClient)
		if !ok {
			client.Kill()
			return nil, fmt.Errorf("invalid kyc client")
		}
		kyc = c
	}
	if manifest.Automation != nil {
		raw, err := rpcClient.Dispense(pluginsdk.PluginKeyAutomation)
		if err != nil {
			client.Kill()
			return nil, err
		}
		c, ok := raw.(pluginv1.AutomationServiceClient)
		if !ok {
			client.Kill()
			return nil, fmt.Errorf("invalid automation client")
		}
		automation = c
	}

	ctxi, cancelInit := context.WithTimeout(ctx, 10*time.Second)
	defer cancelInit()
	initResp, err := core.Init(ctxi, &pluginv1.InitRequest{InstanceId: instanceID, ConfigJson: configJSON})
	if err != nil {
		client.Kill()
		return nil, err
	}
	if initResp != nil && !initResp.Ok {
		client.Kill()
		if initResp.Error != "" {
			return nil, fmt.Errorf("%s", SanitizePluginMessage(initResp.Error))
		}
		return nil, fmt.Errorf("plugin init failed")
	}

	hbCtx, hbCancel := context.WithCancel(context.Background())
	rp := &runningPlugin{
		runtime:    r,
		category:   category,
		pluginID:   pluginID,
		instanceID: instanceID,
		client:     client,
		core:       core,
		sms:        sms,
		payment:    payment,
		kyc:        kyc,
		automation: automation,
		manifest:   manifest,
		cancelHB:   hbCancel,
		health:     nil,
		lastHealth: time.Time{},
	}
	r.mu.Lock()
	r.running[k] = rp
	r.mu.Unlock()
	go func() {
		if safego.Run("plugin heartbeat", func() { rp.heartbeatLoop(hbCtx) }) {
			r.terminateUnhealthy(rp)
		}
	}()

	return manifest, nil
}

func (r *Runtime) Stop(category, pluginID, instanceID string) {
	k := r.key(category, pluginID, instanceID)
	unlock := r.lockKey(k)
	defer unlock()
	r.mu.Lock()
	rp := r.running[k]
	delete(r.running, k)
	// An explicit stop clears any health-triggered cooldown so an operator who
	// disables/re-enables (or restarts) the plugin can start it immediately.
	delete(r.lastKill, k)
	r.mu.Unlock()
	if rp == nil {
		return
	}
	if rp.cancelHB != nil {
		rp.cancelHB()
	}
	if rp.client != nil {
		rp.client.Kill()
	}
}

// remove drops a specific running entry from the map (used when the heartbeat
// detects that the plugin process has exited). The identity guard prevents an
// old heartbeat from removing a freshly restarted instance. The removal time is
// recorded so a subsequent Start obeys the restart cooldown.
func (r *Runtime) remove(category, pluginID, instanceID string, rp *runningPlugin) {
	k := r.key(category, pluginID, instanceID)
	r.mu.Lock()
	if r.running[k] == rp {
		delete(r.running, k)
		r.lastKill[k] = time.Now()
	}
	r.mu.Unlock()
	if rp.cancelHB != nil {
		rp.cancelHB()
	}
	if rp.client != nil {
		rp.client.Kill()
	}
}

// terminateUnhealthy kills an alive-but-unresponsive plugin process after the
// consecutive-failure threshold is reached and removes it from the running set
// so the next EnsureRunning/Start can start a fresh instance (subject to the
// restart cooldown).
func (r *Runtime) terminateUnhealthy(p *runningPlugin) {
	k := r.key(p.category, p.pluginID, p.instanceID)
	r.mu.Lock()
	if r.running[k] == p {
		delete(r.running, k)
		r.lastKill[k] = time.Now()
	}
	r.mu.Unlock()
	if p.cancelHB != nil {
		p.cancelHB()
	}
	if p.client != nil {
		p.client.Kill()
	}
}

func (r *Runtime) GetRunning(category, pluginID, instanceID string) (*runningPlugin, bool) {
	k := r.key(category, pluginID, instanceID)
	r.mu.Lock()
	defer r.mu.Unlock()
	rp := r.running[k]
	return rp, rp != nil
}

// StopAll tears down every running plugin. It is used on application shutdown
// (P2-58) so plugin child processes are killed instead of being orphaned.
func (r *Runtime) StopAll() {
	r.mu.Lock()
	keys := make([]string, 0, len(r.running))
	for k := range r.running {
		keys = append(keys, k)
	}
	r.mu.Unlock()
	for _, k := range keys {
		parts := strings.SplitN(k, ":", 3)
		if len(parts) != 3 {
			continue
		}
		r.Stop(parts[0], parts[1], parts[2])
	}
}

func (p *runningPlugin) heartbeatLoop(ctx context.Context) {
	interval := p.runtime.heartbeatInterval
	if interval <= 0 {
		interval = 10 * time.Second
	}
	timeout := p.runtime.healthTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	maxFailures := p.runtime.maxHealthFailures
	if maxFailures <= 0 {
		maxFailures = 3
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if p.core == nil || p.manifest == nil {
				continue
			}
			cctx, cancel := context.WithTimeout(ctx, timeout)
			resp, err := p.core.Health(cctx, &pluginv1.HealthCheckRequest{InstanceId: p.instanceID})
			cancel()
			if err != nil {
				if p.client != nil && p.client.Exited() {
					// The plugin process is gone. Remove it from the running set
					// so a later EnsureRunning can start a fresh instance
					// instead of handing callers a dead client forever.
					p.runtime.remove(p.category, p.pluginID, p.instanceID, p)
					return
				}
				// Process is alive but unresponsive. Do not report the last
				// successful health as current; after a consecutive-failure
				// threshold the plugin is treated as hung, killed and removed so
				// the next EnsureRunning can restart it (with a cooldown).
				p.mu.Lock()
				p.consecutiveHealthFailures++
				failures := p.consecutiveHealthFailures
				p.health = nil
				p.mu.Unlock()
				if failures >= maxFailures {
					p.runtime.terminateUnhealthy(p)
					return
				}
				continue
			}
			p.mu.Lock()
			p.consecutiveHealthFailures = 0
			p.lastHealth = time.Now()
			p.health = resp
			p.mu.Unlock()
		}
	}
}
