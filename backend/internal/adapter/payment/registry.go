package payment

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/go-plugin"

	plugins "xiaoheiplay/internal/adapter/plugins/core"
	paymentplugin "xiaoheiplay/internal/adapter/plugins/payment"
	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/safego"
)

const (
	settingPaymentEnabled = "payment_providers_enabled"
	settingPaymentConfig  = "payment_providers_config"
	settingPaymentScene   = "payment_providers_scene_enabled"
	settingPaymentPlugins = "payment_plugins"

	// paymentSnapshotRootName is the hidden root under the legacy plugin dir that
	// holds verified content-addressed copies of authorized executables.
	paymentSnapshotRootName = ".snapshots"
)

// legacyExecSupported reports whether a legacy net-RPC payment plugin can be
// spawned from the verified inode handle. The open-handle exec relies on
// /proc/self/fd/<n>, which is Linux-only; on other platforms the legacy plugin
// is skipped with a logged reason (fail-closed) instead of degrading to a
// path-based exec that would reopen the hash-to-exec TOCTOU window. It is a
// package-level var so tests can stub the platform decision.
var legacyExecSupported = runtime.GOOS == "linux"

type providerMeta struct {
	key            string
	defaultEnabled bool
	defaultConfig  string
	factory        func() appshared.PaymentProvider
}

type Registry struct {
	settings    appports.SettingsRepository
	pluginDir   string
	dirSpecs    []pluginSpec
	mu          sync.Mutex
	plugins     pluginState
	grpcPlugins grpcPluginManager
	methodRepo  appports.PluginPaymentMethodRepository

	// newClient constructs the go-plugin client for a legacy net-RPC provider.
	// It is a field so tests can substitute a fake client and capture the exec
	// target without spawning a real plugin process.
	newClient func(cfg *plugin.ClientConfig) pluginClientHandle

	// beforeSpawn, when non-nil, runs immediately before a legacy plugin process
	// is spawned. Tests use it to widen the digest-check-to-spawn window and
	// prove the exec target is the verified snapshot, not the original path.
	beforeSpawn func()
}

// pluginClientHandle is the subset of *plugin.Client the registry relies on.
// Kill() blocks until the plugin process has fully exited (go-plugin's contract),
// so StopAll and the settings-clearing reload wait for every legacy child before
// returning.
type pluginClientHandle interface {
	Client() (plugin.ClientProtocol, error)
	Kill()
}

type pluginState struct {
	hash      string
	clients   map[string]pluginClientHandle
	providers map[string]appshared.PaymentProvider
	// handles pins the open verified-snapshot file descriptors that each legacy
	// plugin was spawned from, keyed by spec.Key. They stay open for the life of
	// the loaded client and are closed when the client is replaced or torn down.
	handles map[string]*os.File
}

type pluginSpec struct {
	Key  string `json:"key"`
	Path string `json:"path"`
	// Digest is the sha256 hex of the executable file, recorded by an operator
	// when authorizing this legacy net-RPC plugin. Entries without a digest, or
	// whose file no longer hashes to it, are refused at load time: an unverified
	// executable must never be spawned (the legacy raw-upload route that once
	// populated these paths has been removed).
	Digest string `json:"digest"`
}

func NewRegistry(settings appports.SettingsRepository) *Registry {
	r := &Registry{settings: settings}
	r.newClient = func(cfg *plugin.ClientConfig) pluginClientHandle { return plugin.NewClient(cfg) }
	return r
}

func (r *Registry) SetPluginManager(mgr grpcPluginManager) {
	r.grpcPlugins = mgr
}

func (r *Registry) SetPluginPaymentMethodRepo(repo appports.PluginPaymentMethodRepository) {
	r.methodRepo = repo
}

func (r *Registry) SetPluginDir(dir string) {
	r.pluginDir = strings.TrimSpace(dir)
}

func (r *Registry) StartWatcher(ctx context.Context, dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}
	r.pluginDir = dir
	if err := r.refreshDirSpecs(); err != nil {
		return err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := watcher.Add(dir); err != nil {
		_ = watcher.Close()
		return err
	}
	safego.Go("legacy payment plugin watcher", func() {
		defer watcher.Close()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				safego.Run("legacy payment plugin refresh", func() { _ = r.refreshDirSpecs() })
			case <-watcher.Events:
				safego.Run("legacy payment plugin refresh", func() { _ = r.refreshDirSpecs() })
			case <-watcher.Errors:
			}
		}
	})
	return nil
}

func (r *Registry) ListProviders(ctx context.Context, includeDisabled bool) ([]appshared.PaymentProvider, error) {
	enabledMap, configMap, err := r.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	providers := make([]appshared.PaymentProvider, 0)
	for _, meta := range r.builtins() {
		enabled := meta.defaultEnabled
		if val, ok := enabledMap[meta.key]; ok {
			enabled = val
		}
		if !enabled && !includeDisabled {
			continue
		}
		provider := meta.factory()
		cfg := meta.defaultConfig
		if val, ok := configMap[meta.key]; ok && len(val) > 0 {
			cfg = string(val)
		}
		if cfg != "" {
			if configurable, ok := provider.(appshared.ConfigurablePaymentProvider); ok {
				_ = configurable.SetConfig(cfg)
			}
		}
		providers = append(providers, provider)
	}
	pluginProviders, err := r.loadPluginProviders(ctx, includeDisabled, enabledMap, configMap)
	if err != nil {
		// A broken legacy net-RPC plugin must not take down the whole provider
		// list (builtins and gRPC providers stay usable); log and continue.
		log.Printf("payment registry: list providers: load net-rpc plugins: %v", err)
	} else {
		providers = append(providers, pluginProviders...)
	}
	if r.grpcPlugins != nil {
		providers = append(providers, r.grpcProviders(ctx, includeDisabled)...)
	}
	sort.SliceStable(providers, func(i, j int) bool {
		return providers[i].Key() < providers[j].Key()
	})
	return providers, nil
}

func (r *Registry) GetProvider(ctx context.Context, key string) (appshared.PaymentProvider, error) {
	enabledMap, configMap, err := r.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	for _, meta := range r.builtins() {
		if meta.key != key {
			continue
		}
		enabled := meta.defaultEnabled
		if val, ok := enabledMap[key]; ok {
			enabled = val
		}
		if !enabled {
			return nil, appshared.ErrForbidden
		}
		provider := meta.factory()
		cfg := meta.defaultConfig
		if val, ok := configMap[key]; ok && len(val) > 0 {
			cfg = string(val)
		}
		if cfg != "" {
			if configurable, ok := provider.(appshared.ConfigurablePaymentProvider); ok {
				_ = configurable.SetConfig(cfg)
			}
		}
		return provider, nil
	}
	pluginProviders, err := r.loadPluginProviders(ctx, true, enabledMap, configMap)
	if err != nil {
		// Same fault isolation as ListProviders: a broken legacy net-RPC plugin
		// must not prevent looking up builtin or gRPC providers.
		log.Printf("payment registry: get provider %q: load net-rpc plugins: %v", key, err)
		pluginProviders = nil
	}
	for _, provider := range pluginProviders {
		if provider.Key() != key {
			continue
		}
		enabled := false
		if val, ok := enabledMap[key]; ok {
			enabled = val
		}
		if !enabled {
			return nil, appshared.ErrForbidden
		}
		return provider, nil
	}
	if r.grpcPlugins != nil {
		// Persistent settings are authoritative for gRPC plugin methods too: a
		// method explicitly disabled in payment_providers_enabled must not become
		// reachable just because a process for it happens to be loaded (P3-23).
		if val, ok := enabledMap[key]; ok && !val {
			return nil, appshared.ErrForbidden
		}
		if disabled, known := r.grpcPaymentMethodDisabled(ctx, key); known && disabled {
			return nil, appshared.ErrForbidden
		}
		if p := r.grpcProviderByKey(ctx, key); p != nil {
			return p, nil
		}
	}
	return nil, appshared.ErrNotFound
}

func (r *Registry) grpcPaymentMethodDisabled(ctx context.Context, key string) (disabled bool, known bool) {
	parts := strings.SplitN(strings.TrimSpace(key), ".", 2)
	if len(parts) != 2 {
		return false, false
	}
	pluginID := strings.TrimSpace(parts[0])
	method := strings.TrimSpace(parts[1])
	if pluginID == "" || method == "" {
		return false, false
	}
	items, err := r.grpcPlugins.List(ctx)
	if err != nil {
		return false, false
	}
	for _, it := range items {
		if !it.Enabled || it.InstanceID != plugins.DefaultInstanceID || it.PluginID != pluginID || it.Capabilities.Capabilities.Payment == nil {
			continue
		}
		found := false
		for _, m := range it.Capabilities.Capabilities.Payment.Methods {
			if m == method {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		known = true
		enabledMap, authoritative := r.pluginPaymentMethodEnabledMap(ctx, it.Category, it.PluginID, it.InstanceID)
		if !authoritative {
			// Payment-method enablement state could not be loaded; fail closed
			// rather than risking a method that was disabled becoming reachable.
			return true, true
		}
		if ok, exists := enabledMap[method]; exists {
			return !ok, true
		}
		return false, true
	}
	return false, false
}

func (r *Registry) GetProviderConfig(ctx context.Context, key string) (string, bool, error) {
	enabledMap, configMap, err := r.loadSettings(ctx)
	if err != nil {
		return "", false, err
	}
	defaultConfig := ""
	defaultEnabled := false
	isBuiltin := false
	for _, meta := range r.builtins() {
		if meta.key == key {
			defaultConfig = meta.defaultConfig
			defaultEnabled = meta.defaultEnabled
			isBuiltin = true
			break
		}
	}
	if !isBuiltin {
		defaultEnabled = false
	}
	enabled := defaultEnabled
	// Persistent settings are the authority. In particular a provider explicitly
	// disabled in payment_providers_enabled must stay disabled even when a gRPC
	// plugin for the same key is currently loaded: a running process must never
	// flip a settings-disabled provider back to enabled.
	if val, ok := enabledMap[key]; ok {
		enabled = val
	} else if !isBuiltin && r.grpcPlugins != nil {
		// gRPC plugin payment methods have no payment_providers_enabled entry;
		// their enablement comes from the persisted install + per-method table,
		// never from whether a process happens to be loaded.
		if disabled, known := r.grpcPaymentMethodDisabled(ctx, key); known {
			enabled = !disabled
		}
	}
	cfg := defaultConfig
	if val, ok := configMap[key]; ok && len(val) > 0 {
		cfg = string(val)
	}
	return cfg, enabled, nil
}

func (r *Registry) UpdateProviderConfig(ctx context.Context, key string, enabled bool, configJSON string) error {
	if r.settings == nil {
		return appshared.ErrInvalidInput
	}
	if r.grpcPlugins != nil && strings.Contains(key, ".") {
		return appshared.ErrInvalidInput
	}
	enabledMap, configMap, err := r.loadSettings(ctx)
	if err != nil {
		return err
	}
	enabledMap[key] = enabled
	if configJSON != "" {
		configMap[key] = json.RawMessage(configJSON)
	} else {
		delete(configMap, key)
	}
	if err := r.upsertJSON(ctx, settingPaymentEnabled, enabledMap); err != nil {
		return err
	}
	return r.upsertJSON(ctx, settingPaymentConfig, configMap)
}

func (r *Registry) GetProviderSceneEnabled(ctx context.Context, key, scene string) (bool, error) {
	if strings.TrimSpace(key) == "" || strings.TrimSpace(scene) == "" {
		return false, appshared.ErrInvalidInput
	}
	sceneMap, err := r.loadSceneSettings(ctx)
	if err != nil {
		return false, err
	}
	providerMap, ok := sceneMap[key]
	if !ok {
		return true, nil
	}
	enabled, ok := providerMap[scene]
	if !ok {
		return true, nil
	}
	return enabled, nil
}

func (r *Registry) UpdateProviderSceneEnabled(ctx context.Context, key, scene string, enabled bool) error {
	if strings.TrimSpace(key) == "" || strings.TrimSpace(scene) == "" {
		return appshared.ErrInvalidInput
	}
	if r.settings == nil {
		return appshared.ErrInvalidInput
	}
	sceneMap, err := r.loadSceneSettings(ctx)
	if err != nil {
		return err
	}
	providerMap, ok := sceneMap[key]
	if !ok || providerMap == nil {
		providerMap = map[string]bool{}
	}
	providerMap[scene] = enabled
	sceneMap[key] = providerMap
	return r.upsertJSON(ctx, settingPaymentScene, sceneMap)
}

func (r *Registry) builtins() []providerMeta {
	return []providerMeta{
		{
			key:            "approval",
			defaultEnabled: true,
			factory: func() appshared.PaymentProvider {
				return newApprovalProvider()
			},
		},
		{
			key:            "balance",
			defaultEnabled: true,
			factory: func() appshared.PaymentProvider {
				return newBalanceProvider()
			},
		},
	}
}

func (r *Registry) loadSettings(ctx context.Context) (map[string]bool, map[string]json.RawMessage, error) {
	enabledMap := map[string]bool{}
	configMap := map[string]json.RawMessage{}
	if r.settings == nil {
		return enabledMap, configMap, nil
	}
	enabledMap = loadBoolMap(ctx, r.settings, settingPaymentEnabled)
	configMap = loadRawMap(ctx, r.settings, settingPaymentConfig)
	return enabledMap, configMap, nil
}

func (r *Registry) loadSceneSettings(ctx context.Context) (map[string]map[string]bool, error) {
	if r.settings == nil {
		return map[string]map[string]bool{}, nil
	}
	setting, err := r.settings.GetSetting(ctx, settingPaymentScene)
	if err != nil {
		if errors.Is(err, appshared.ErrNotFound) {
			return map[string]map[string]bool{}, nil
		}
		return map[string]map[string]bool{}, err
	}
	if setting.ValueJSON == "" {
		return map[string]map[string]bool{}, nil
	}
	var out map[string]map[string]bool
	if err := json.Unmarshal([]byte(setting.ValueJSON), &out); err != nil {
		return map[string]map[string]bool{}, err
	}
	if out == nil {
		return map[string]map[string]bool{}, nil
	}
	return out, nil
}

func (r *Registry) upsertJSON(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.settings.UpsertSetting(ctx, domain.Setting{Key: key, ValueJSON: string(raw)})
}

func loadBoolMap(ctx context.Context, repo appports.SettingsRepository, key string) map[string]bool {
	setting, err := repo.GetSetting(ctx, key)
	if err != nil || setting.ValueJSON == "" {
		return map[string]bool{}
	}
	var out map[string]bool
	if err := json.Unmarshal([]byte(setting.ValueJSON), &out); err != nil {
		return map[string]bool{}
	}
	return out
}

func loadRawMap(ctx context.Context, repo appports.SettingsRepository, key string) map[string]json.RawMessage {
	setting, err := repo.GetSetting(ctx, key)
	if err != nil || setting.ValueJSON == "" {
		return map[string]json.RawMessage{}
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal([]byte(setting.ValueJSON), &out); err != nil {
		return map[string]json.RawMessage{}
	}
	return out
}

func (r *Registry) loadPluginProviders(ctx context.Context, includeDisabled bool, enabledMap map[string]bool, configMap map[string]json.RawMessage) ([]appshared.PaymentProvider, error) {
	specs := r.loadPluginSpecs(ctx)
	// Always reconcile the running set with the persisted specs — even an empty
	// spec list must tear down clients that were previously spawned, otherwise
	// clearing payment_plugins would leak the child processes.
	state, err := r.ensurePlugins(specs)
	if err != nil {
		return nil, err
	}
	out := make([]appshared.PaymentProvider, 0, len(state.providers))
	for key, provider := range state.providers {
		enabled := false
		if val, ok := enabledMap[key]; ok {
			enabled = val
		}
		if !enabled && !includeDisabled {
			continue
		}
		if cfg, ok := configMap[key]; ok && len(cfg) > 0 {
			if configurable, ok := provider.(appshared.ConfigurablePaymentProvider); ok {
				_ = configurable.SetConfig(string(cfg))
			}
		}
		out = append(out, provider)
	}
	return out, nil
}

func (r *Registry) loadPluginSpecs(ctx context.Context) []pluginSpec {
	// Only plugins EXPLICITLY listed in the payment_plugins setting are ever
	// spawned, and only when each entry carries a content-digest anchor that
	// still matches the file on disk. A raw executable manually dropped into the
	// plugin dir must not be auto-executed, and a settings entry pointing at an
	// unanchored (or since-modified) executable must be refused rather than
	// spawned: neither has passed package/signature/content-digest verification,
	// and the legacy raw-upload route that once populated that dir has been
	// removed. Dir scanning remains only as a discovery aid (StartWatcher still
	// watches and lists what is present); it is never an execution source.
	return r.verifiedPluginSpecs(ctx)
}

// verifiedPluginSpecs loads the payment_plugins setting and drops every entry
// that is not content-digest anchored: the entry must carry a digest (sha256 of
// the executable file) and the file must still hash to it. Unanchored or
// modified entries are logged and skipped — never spawned — so an operator who
// configures a legacy net-RPC plugin must explicitly pin the digest of the
// executable they are authorizing, and any later on-disk modification is
// detected instead of executed.
func (r *Registry) verifiedPluginSpecs(ctx context.Context) []pluginSpec {
	specs := r.loadSettingsSpecs(ctx)
	if len(specs) == 0 {
		return nil
	}
	out := make([]pluginSpec, 0, len(specs))
	for _, spec := range specs {
		if strings.TrimSpace(spec.Path) == "" {
			log.Printf("payment registry: skipping payment_plugins entry %q with empty path", spec.Key)
			continue
		}
		digest := strings.TrimSpace(spec.Digest)
		if digest == "" {
			log.Printf("payment registry: refusing unanchored payment_plugins entry %q (path %s): add a sha256 digest to authorize it", spec.Key, spec.Path)
			continue
		}
		sum, err := fileSHA256(spec.Path)
		if err != nil {
			log.Printf("payment registry: refusing payment_plugins entry %q: %v", spec.Key, err)
			continue
		}
		if !strings.EqualFold(sum, digest) {
			log.Printf("payment registry: refusing payment_plugins entry %q: file digest changed since it was anchored", spec.Key)
			continue
		}
		out = append(out, spec)
	}
	return out
}

// fileSHA256 returns the lowercase hex sha256 of a file's contents.
func fileSHA256(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum[:]), nil
}

// paymentSnapshotRoot returns the directory holding verified snapshots for the
// legacy plugin dir, or "" when no plugin dir is configured.
func (r *Registry) paymentSnapshotRoot() string {
	if strings.TrimSpace(r.pluginDir) == "" {
		return ""
	}
	return filepath.Join(r.pluginDir, paymentSnapshotRootName, "payment")
}

// unixSocketConfig pins go-plugin control sockets to a dedicated 0700 directory
// under the legacy plugin dir instead of the shared system TMPDIR, so the
// sockets are never reachable through a world-traversable temp location. The
// mode is enforced even when the directory pre-exists with looser permissions.
func (r *Registry) unixSocketConfig() *plugin.UnixSocketConfig {
	dir := filepath.Join(r.pluginDir, ".sockets")
	_ = os.MkdirAll(dir, 0o700)
	_ = os.Chmod(dir, 0o700)
	return &plugin.UnixSocketConfig{TempDir: dir}
}

// verifiedPluginBinarySnapshot returns the path AND an open handle to an
// immutable, content-addressed copy of the legacy plugin executable whose bytes
// are verified to hash to spec.Digest. ensurePlugins executes the plugin from
// that copy — never from spec.Path — so a swap of the original between the digest
// check and the spawn cannot change what is executed. The returned handle pins
// the exact inode that was verified: on Linux the child is spawned via
// /proc/self/fd/<n> with that handle, so even a same-user replacement of the
// snapshot path in the window between verification and spawn cannot change what
// is executed — the executed bytes ARE the verified bytes. On reuse the existing
// copy is re-verified the same way; a tampered copy is refused rather than
// executed.
//
// On platforms without /proc/self/fd the handle is unavailable and this returns
// a nil handle: ensurePlugins then refuses the entry (fail-closed) instead of
// falling back to a path-based exec, which would reopen the hash-to-exec TOCTOU
// window (P2-51). The guard is the legacyExecSupported package var.
func (r *Registry) verifiedPluginBinarySnapshot(spec pluginSpec) (string, *os.File, error) {
	if !legacyExecSupported {
		return "", nil, nil
	}
	root := r.paymentSnapshotRoot()
	if root == "" {
		return "", nil, fmt.Errorf("payment plugin dir not configured")
	}
	dir := filepath.Join(root, spec.Key)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", nil, err
	}
	final := filepath.Join(dir, strings.ToLower(strings.TrimSpace(spec.Digest)))

	// Reuse path: open the existing copy, re-hash the OPEN HANDLE's bytes, and
	// keep the handle so the inode actually executed is the one just verified. A
	// same-user replacement of the snapshot is detected instead of executed; if
	// the path is swapped after we open it, the handle still references the
	// original (verified) inode.
	if f, err := os.Open(final); err == nil {
		sum, hErr := hashFile(f)
		if hErr != nil {
			_ = f.Close()
			return "", nil, hErr
		}
		if !strings.EqualFold(sum, spec.Digest) {
			_ = f.Close()
			return "", nil, fmt.Errorf("verified snapshot tampered: content no longer matches its digest anchor")
		}
		return final, f, nil
	}

	// Fresh copy: write to a temp file, re-hash the exact bytes written, open the
	// verified temp (pinning its inode), then atomically rename into the
	// content-addressed location. Even if spec.Path is swapped while the copy is
	// in flight, the temp bytes must still hash to the anchored digest or they are
	// discarded and the entry is refused.
	tmp, err := os.CreateTemp(dir, ".snap-*")
	if err != nil {
		return "", nil, err
	}
	tmpPath := tmp.Name()
	abort := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	src, err := os.Open(spec.Path)
	if err != nil {
		abort()
		return "", nil, err
	}
	if _, err := io.Copy(tmp, src); err != nil {
		_ = src.Close()
		abort()
		return "", nil, err
	}
	if err := src.Close(); err != nil {
		abort()
		return "", nil, err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", nil, err
	}
	sum, err := fileSHA256(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", nil, err
	}
	if !strings.EqualFold(sum, spec.Digest) {
		_ = os.Remove(tmpPath)
		return "", nil, fmt.Errorf("executable digest changed since it was anchored")
	}
	if err := os.Chmod(tmpPath, 0o500); err != nil {
		_ = os.Remove(tmpPath)
		return "", nil, err
	}
	// Open the verified temp BEFORE renaming it into place: the handle then
	// references the exact inode whose bytes hashed to the anchor, regardless of
	// any later rename or path swap.
	f, err := os.Open(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", nil, err
	}
	if err := os.Rename(tmpPath, final); err != nil {
		// A concurrent caller may have won the race; re-verify its copy (via a
		// fresh handle) before reusing it.
		if f2, err2 := os.Open(final); err2 == nil {
			sum2, hErr2 := hashFile(f2)
			if hErr2 == nil && strings.EqualFold(sum2, spec.Digest) {
				_ = os.Remove(tmpPath)
				_ = f.Close()
				return final, f2, nil
			}
			_ = f2.Close()
		}
		_ = os.Remove(tmpPath)
		_ = f.Close()
		return "", nil, err
	}
	return final, f, nil
}

// hashFile returns the lowercase hex sha256 of the current contents of the open
// handle, rewinding first so a handle can be re-hashed after previous reads.
func hashFile(f *os.File) (string, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// execFromVerifiedHandle builds an exec.Cmd whose executable is the open handle
// to the verified snapshot inode, so the bytes actually executed are exactly the
// bytes that were verified even if the snapshot path is swapped at spawn time
// (hash-to-exec TOCTOU). The first ExtraFiles entry is fd 3 in the child;
// execve of /proc/self/fd/3 resolves through the inherited open file
// description. execFromVerifiedHandle is Linux-only (/proc/self/fd); there is no
// path-based fallback — ensurePlugins refuses to spawn legacy net-RPC plugins on
// platforms without a verified handle (the legacyExecSupported guard).
func execFromVerifiedHandle(snapPath string, f *os.File) *exec.Cmd {
	// snapPath is unused by the fd-bound path but kept for documentation of the
	// verified snapshot that the handle pins.
	_ = snapPath
	cmd := exec.Command("/proc/self/fd/3")
	cmd.ExtraFiles = []*os.File{f}
	return cmd
}

// StopAll terminates every legacy net-RPC plugin client and waits for each child
// process to exit (go-plugin Kill blocks until the process is gone). It is
// invoked on application shutdown so legacy payment plugin children are never
// orphaned, and it also tears down clients whose settings were cleared (the next
// load would otherwise reuse the stale set). Open verified-snapshot handles are
// closed so no fd is retained.
func (r *Registry) StopAll() {
	r.mu.Lock()
	clients := r.plugins.clients
	handles := r.plugins.handles
	r.plugins = pluginState{}
	r.mu.Unlock()
	for key, client := range clients {
		log.Printf("payment registry: stopping legacy net-rpc plugin %q", key)
		client.Kill()
	}
	for _, f := range handles {
		_ = f.Close()
	}
}

// Close is an alias for StopAll so the registry satisfies a common shutdown
// lifecycle interface.
func (r *Registry) Close() {
	r.StopAll()
}

// isReservedProviderKey reports whether a plugin-reported key collides with a
// built-in provider key that can never be overridden by a plugin.
func isReservedProviderKey(key string) bool {
	switch key {
	case "approval", "balance":
		return true
	}
	return false
}

func (r *Registry) ensurePlugins(specs []pluginSpec) (pluginState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	hash := pluginSpecHash(specs)
	if r.plugins.hash == hash && r.plugins.providers != nil {
		return r.plugins, nil
	}
	for _, client := range r.plugins.clients {
		client.Kill()
	}
	for _, f := range r.plugins.handles {
		_ = f.Close()
	}
	clients := map[string]pluginClientHandle{}
	providers := map[string]appshared.PaymentProvider{}
	handles := map[string]*os.File{}
	fail := func(err error) (pluginState, error) {
		for _, c := range clients {
			c.Kill()
		}
		for _, f := range handles {
			_ = f.Close()
		}
		r.plugins = pluginState{}
		return pluginState{}, err
	}
	for _, spec := range specs {
		if spec.Path == "" {
			continue
		}
		// Bind the spawn to a verified immutable snapshot of the executable, not
		// to spec.Path directly: the digest check in verifiedPluginSpecs runs
		// earlier, and executing the original path would be a hash-to-exec TOCTOU
		// (a same-user swap between the check and the spawn would run attacker
		// bytes). The snapshot copy is re-verified on every reuse and the child is
		// spawned from the OPEN HANDLE of the verified inode (Linux), so even a
		// swap of the snapshot path between verification and spawn cannot change
		// what is executed.
		snapPath, snapFile, err := r.verifiedPluginBinarySnapshot(spec)
		if err != nil {
			log.Printf("payment registry: refusing payment_plugins entry %q: %v", spec.Key, err)
			continue
		}
		if snapFile == nil {
			// P2-51 fail-closed: without an open verified inode handle (Linux
			// /proc/self/fd) a path-based exec would retain the hash-to-exec
			// TOCTOU window. Refuse to spawn the legacy net-rpc plugin on this
			// platform instead of degrading to an unverifiable exec.
			log.Printf("payment registry: refusing payment_plugins entry %q on this platform: verified inode handle unavailable (legacy exec disabled)", spec.Key)
			continue
		}
		handles[spec.Key] = snapFile
		if r.beforeSpawn != nil {
			r.beforeSpawn()
		}
		client := r.newClient(&plugin.ClientConfig{
			HandshakeConfig: paymentplugin.Handshake,
			Plugins: map[string]plugin.Plugin{
				paymentplugin.ProviderPluginName: &paymentplugin.ProviderPlugin{},
			},
			Cmd:              execFromVerifiedHandle(snapPath, snapFile),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
			// AutoMTLS: the host and the spawned plugin exchange one-time certs,
			// so the host only talks to the exact process it launched and that
			// process only accepts the host. Control sockets are pinned to a
			// dedicated 0700 directory instead of the shared TMPDIR.
			AutoMTLS:         true,
			UnixSocketConfig: r.unixSocketConfig(),
		})
		rpcClient, err := client.Client()
		if err != nil {
			client.Kill()
			return fail(err)
		}
		raw, err := rpcClient.Dispense(paymentplugin.ProviderPluginName)
		if err != nil {
			client.Kill()
			return fail(err)
		}
		provider, ok := raw.(paymentplugin.Provider)
		if !ok {
			client.Kill()
			return fail(fmt.Errorf("invalid payment provider plugin"))
		}
		wrapped := &pluginProvider{provider: provider}
		key := wrapped.Key()
		if key == "" || isReservedProviderKey(key) {
			// An empty or reserved key can never be served (builtins win in
			// GetProvider); dropping it prevents namespace pollution and leaks.
			client.Kill()
			if f, ok := handles[spec.Key]; ok {
				_ = f.Close()
				delete(handles, spec.Key)
			}
			log.Printf("payment registry: skipping plugin %q with unusable provider key %q", spec.Key, key)
			continue
		}
		if existing, ok := clients[key]; ok {
			// Duplicate self-reported keys: keep the newest plugin but kill the
			// replaced client so the earlier process does not leak.
			existing.Kill()
			log.Printf("payment registry: duplicate provider key %q, replacing previously loaded plugin", key)
		}
		providers[key] = wrapped
		clients[key] = client
	}
	r.plugins = pluginState{hash: hash, clients: clients, providers: providers, handles: handles}
	return r.plugins, nil
}

func pluginSpecHash(specs []pluginSpec) string {
	items := make([]string, 0, len(specs))
	for _, spec := range specs {
		items = append(items, spec.Key+"="+spec.Path+"="+spec.Digest)
	}
	sort.Strings(items)
	return strings.Join(items, ";")
}

func (r *Registry) loadSettingsSpecs(ctx context.Context) []pluginSpec {
	if r.settings == nil {
		return nil
	}
	setting, err := r.settings.GetSetting(ctx, settingPaymentPlugins)
	if err != nil || setting.ValueJSON == "" {
		return nil
	}
	var specs []pluginSpec
	if err := json.Unmarshal([]byte(setting.ValueJSON), &specs); err != nil {
		return nil
	}
	return specs
}

func (r *Registry) refreshDirSpecs() error {
	if r.pluginDir == "" {
		return nil
	}
	specs, err := scanPluginDir(r.pluginDir)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.dirSpecs = specs
	r.mu.Unlock()
	return nil
}

func scanPluginDir(dir string) ([]pluginSpec, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var specs []pluginSpec
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := filepath.Join(dir, name)
		if !isExecutable(path) {
			continue
		}
		key := strings.TrimSuffix(name, filepath.Ext(name))
		if key == "" {
			continue
		}
		specs = append(specs, pluginSpec{Key: key, Path: path})
	}
	return specs, nil
}

func isExecutable(path string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Ext(path), ".exe")
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0o111 != 0
}

type pluginProvider struct {
	provider paymentplugin.Provider
}

func (p *pluginProvider) Key() string {
	return p.provider.Key()
}

func (p *pluginProvider) Name() string {
	return p.provider.Name()
}

func (p *pluginProvider) SchemaJSON() string {
	return p.provider.SchemaJSON()
}

func (p *pluginProvider) SetConfig(configJSON string) error {
	return p.provider.SetConfig(configJSON)
}

func (p *pluginProvider) CreatePayment(ctx context.Context, req appshared.PaymentCreateRequest) (appshared.PaymentCreateResult, error) {
	// The legacy net-RPC Provider interface has no ctx, so honor the caller's
	// deadline/cancellation here; without a deadline dispatch inline to avoid a
	// goroutine on the hot path.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		return p.provider.CreatePayment(req)
	}
	if err := ctx.Err(); err != nil {
		return appshared.PaymentCreateResult{}, err
	}
	type callResult struct {
		res appshared.PaymentCreateResult
		err error
	}
	ch := make(chan callResult, 1)
	go func() {
		out := callResult{}
		if safego.Run("legacy payment create", func() { out.res, out.err = p.provider.CreatePayment(req) }) {
			out.err = domain.ErrServiceUnavailable
		}
		ch <- out
	}()
	select {
	case <-ctx.Done():
		return appshared.PaymentCreateResult{}, ctx.Err()
	case out := <-ch:
		return out.res, out.err
	}
}

func (p *pluginProvider) VerifyNotify(ctx context.Context, req appshared.RawHTTPRequest) (appshared.PaymentNotifyResult, error) {
	// Versioned contract: the plugin receives the full original HTTP callback
	// (method/path/raw query/headers/body). A plugin that only implements the
	// retired v1 params callback is refused (fail-closed) — signature material
	// in JSON/multipart bodies or headers must never be lossily reduced to a
	// params map.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		return p.provider.VerifyNotifyV2(req)
	}
	if err := ctx.Err(); err != nil {
		return appshared.PaymentNotifyResult{}, err
	}
	type callResult struct {
		res appshared.PaymentNotifyResult
		err error
	}
	ch := make(chan callResult, 1)
	go func() {
		out := callResult{}
		if safego.Run("legacy payment notify verification", func() { out.res, out.err = p.provider.VerifyNotifyV2(req) }) {
			out.err = domain.ErrServiceUnavailable
		}
		ch <- out
	}()
	select {
	case <-ctx.Done():
		return appshared.PaymentNotifyResult{}, ctx.Err()
	case out := <-ch:
		return out.res, out.err
	}
}
