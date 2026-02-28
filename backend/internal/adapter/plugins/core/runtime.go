package plugins

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"

	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type Runtime struct {
	baseDir string

	mu      sync.Mutex
	running map[string]*runningPlugin
}

type runningPlugin struct {
	mu sync.Mutex

	category   string
	pluginID   string
	instanceID string

	client     *plugin.Client
	core       pluginv1.CoreServiceClient
	sms        pluginv1.SmsServiceClient
	payment    pluginv1.PaymentServiceClient
	kyc        pluginv1.KycServiceClient
	automation pluginv1.AutomationServiceClient
	manifest   *pluginv1.Manifest

	lastHealth time.Time
	health     *pluginv1.HealthCheckResponse
	cancelHB   context.CancelFunc
}

func NewRuntime(baseDir string) *Runtime {
	return &Runtime{baseDir: baseDir, running: map[string]*runningPlugin{}}
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
	}

	return nil
}

func (r *Runtime) key(category, pluginID, instanceID string) string {
	return category + ":" + pluginID + ":" + instanceID
}

func (r *Runtime) Start(ctx context.Context, category, pluginID, instanceID, configJSON string) (*pluginv1.Manifest, error) {
	if category == "" || pluginID == "" || instanceID == "" {
		return nil, fmt.Errorf("invalid input")
	}
	k := r.key(category, pluginID, instanceID)

	r.mu.Lock()
	if existing := r.running[k]; existing != nil {
		r.mu.Unlock()
		return existing.manifest, nil
	}
	r.mu.Unlock()

	pluginDir := filepath.Join(r.baseDir, category, pluginID)
	manifestJSON, err := ReadManifest(pluginDir)
	if err != nil {
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

	client := plugin.NewClient(&plugin.ClientConfig{
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
		Cmd: cmd,
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
			return nil, fmt.Errorf("%s", initResp.Error)
		}
		return nil, fmt.Errorf("plugin init failed")
	}

	hbCtx, hbCancel := context.WithCancel(context.Background())
	rp := &runningPlugin{
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
	go rp.heartbeatLoop(hbCtx)

	r.mu.Lock()
	r.running[k] = rp
	r.mu.Unlock()

	return manifest, nil
}

func (r *Runtime) Stop(category, pluginID, instanceID string) {
	k := r.key(category, pluginID, instanceID)
	r.mu.Lock()
	rp := r.running[k]
	delete(r.running, k)
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

func (r *Runtime) GetRunning(category, pluginID, instanceID string) (*runningPlugin, bool) {
	k := r.key(category, pluginID, instanceID)
	r.mu.Lock()
	defer r.mu.Unlock()
	rp := r.running[k]
	return rp, rp != nil
}

func (p *runningPlugin) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if p.core == nil || p.manifest == nil {
				continue
			}
			cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			resp, err := p.core.Health(cctx, &pluginv1.HealthCheckRequest{InstanceId: p.instanceID})
			cancel()
			if err != nil {
				continue
			}
			p.mu.Lock()
			p.lastHealth = time.Now()
			p.health = resp
			p.mu.Unlock()
		}
	}
}
