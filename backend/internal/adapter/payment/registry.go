package payment

import (
	"context"
	"encoding/json"
	"errors"
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

	paymentplugin "xiaoheiplay/internal/adapter/payment/plugin"
	plugins "xiaoheiplay/internal/adapter/plugins"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/usecase"
)

const (
	settingPaymentEnabled = "payment_providers_enabled"
	settingPaymentConfig  = "payment_providers_config"
	settingPaymentPlugins = "payment_plugins"
)

type providerMeta struct {
	key            string
	defaultEnabled bool
	defaultConfig  string
	factory        func() usecase.PaymentProvider
}

type Registry struct {
	settings    usecase.SettingsRepository
	pluginDir   string
	dirSpecs    []pluginSpec
	mu          sync.Mutex
	plugins     pluginState
	grpcPlugins *plugins.Manager
	methodRepo  usecase.PluginPaymentMethodRepository
}

type pluginState struct {
	hash      string
	clients   map[string]*plugin.Client
	providers map[string]usecase.PaymentProvider
}

type pluginSpec struct {
	Key  string `json:"key"`
	Path string `json:"path"`
}

func NewRegistry(settings usecase.SettingsRepository) *Registry {
	return &Registry{settings: settings}
}

func (r *Registry) SetPluginManager(mgr *plugins.Manager) {
	r.grpcPlugins = mgr
}

func (r *Registry) SetPluginPaymentMethodRepo(repo usecase.PluginPaymentMethodRepository) {
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
	go func() {
		defer watcher.Close()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = r.refreshDirSpecs()
			case <-watcher.Events:
				_ = r.refreshDirSpecs()
			case <-watcher.Errors:
			}
		}
	}()
	return nil
}

func (r *Registry) ListProviders(ctx context.Context, includeDisabled bool) ([]usecase.PaymentProvider, error) {
	enabledMap, configMap, err := r.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	providers := make([]usecase.PaymentProvider, 0)
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
			if configurable, ok := provider.(usecase.ConfigurablePaymentProvider); ok {
				_ = configurable.SetConfig(cfg)
			}
		}
		providers = append(providers, provider)
	}
	pluginProviders, err := r.loadPluginProviders(ctx, includeDisabled, enabledMap, configMap)
	if err != nil {
		return nil, err
	}
	providers = append(providers, pluginProviders...)
	if !includeDisabled && r.grpcPlugins != nil {
		providers = append(providers, r.grpcProviders(ctx)...)
	}
	sort.SliceStable(providers, func(i, j int) bool {
		return providers[i].Key() < providers[j].Key()
	})
	return providers, nil
}

func (r *Registry) GetProvider(ctx context.Context, key string) (usecase.PaymentProvider, error) {
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
			return nil, usecase.ErrForbidden
		}
		provider := meta.factory()
		cfg := meta.defaultConfig
		if val, ok := configMap[key]; ok && len(val) > 0 {
			cfg = string(val)
		}
		if cfg != "" {
			if configurable, ok := provider.(usecase.ConfigurablePaymentProvider); ok {
				_ = configurable.SetConfig(cfg)
			}
		}
		return provider, nil
	}
	pluginProviders, err := r.loadPluginProviders(ctx, true, enabledMap, configMap)
	if err != nil {
		return nil, err
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
			return nil, usecase.ErrForbidden
		}
		return provider, nil
	}
	if r.grpcPlugins != nil {
		if disabled, known := r.grpcPaymentMethodDisabled(ctx, key); known && disabled {
			return nil, usecase.ErrForbidden
		}
		if p := r.grpcProviderByKey(ctx, key); p != nil {
			return p, nil
		}
	}
	return nil, usecase.ErrNotFound
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
		enabledMap := r.pluginPaymentMethodEnabledMap(ctx, it.Category, it.PluginID, it.InstanceID)
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
	if val, ok := enabledMap[key]; ok {
		enabled = val
	}
	cfg := defaultConfig
	if val, ok := configMap[key]; ok && len(val) > 0 {
		cfg = string(val)
	}
	if !enabled && r.grpcPlugins != nil {
		if p := r.grpcProviderByKey(ctx, key); p != nil {
			enabled = true
			cfg = ""
		}
	}
	return cfg, enabled, nil
}

func (r *Registry) UpdateProviderConfig(ctx context.Context, key string, enabled bool, configJSON string) error {
	if r.settings == nil {
		return usecase.ErrInvalidInput
	}
	if r.grpcPlugins != nil && strings.Contains(key, ".") {
		return usecase.ErrInvalidInput
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

func (r *Registry) builtins() []providerMeta {
	return []providerMeta{
		{
			key:            "approval",
			defaultEnabled: true,
			factory: func() usecase.PaymentProvider {
				return newApprovalProvider()
			},
		},
		{
			key:            "balance",
			defaultEnabled: true,
			factory: func() usecase.PaymentProvider {
				return newBalanceProvider()
			},
		},
		{
			key:            "yipay",
			defaultEnabled: false,
			defaultConfig:  `{"base_url":"https://pays.org.cn/submit.php","pid":"","key":"","pay_type":"","notify_url":"","return_url":"","sign_type":"MD5"}`,
			factory: func() usecase.PaymentProvider {
				return newYipayProvider()
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

func (r *Registry) upsertJSON(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.settings.UpsertSetting(ctx, domain.Setting{Key: key, ValueJSON: string(raw)})
}

func loadBoolMap(ctx context.Context, repo usecase.SettingsRepository, key string) map[string]bool {
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

func loadRawMap(ctx context.Context, repo usecase.SettingsRepository, key string) map[string]json.RawMessage {
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

func (r *Registry) loadPluginProviders(ctx context.Context, includeDisabled bool, enabledMap map[string]bool, configMap map[string]json.RawMessage) ([]usecase.PaymentProvider, error) {
	specs := r.loadPluginSpecs(ctx)
	if len(specs) == 0 {
		return nil, nil
	}
	state, err := r.ensurePlugins(specs)
	if err != nil {
		return nil, err
	}
	out := make([]usecase.PaymentProvider, 0, len(state.providers))
	for key, provider := range state.providers {
		enabled := false
		if val, ok := enabledMap[key]; ok {
			enabled = val
		}
		if !enabled && !includeDisabled {
			continue
		}
		if cfg, ok := configMap[key]; ok && len(cfg) > 0 {
			if configurable, ok := provider.(usecase.ConfigurablePaymentProvider); ok {
				_ = configurable.SetConfig(string(cfg))
			}
		}
		out = append(out, provider)
	}
	return out, nil
}

func (r *Registry) loadPluginSpecs(ctx context.Context) []pluginSpec {
	specs := r.loadSettingsSpecs(ctx)
	dirSpecs := r.copyDirSpecs()
	if r.pluginDir != "" {
		if err := r.refreshDirSpecs(); err == nil {
			dirSpecs = r.copyDirSpecs()
		}
	}
	return mergePluginSpecs(specs, dirSpecs)
}

func (r *Registry) ensurePlugins(specs []pluginSpec) (*pluginState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	hash := pluginSpecHash(specs)
	if r.plugins.hash == hash && r.plugins.providers != nil {
		return &r.plugins, nil
	}
	for _, client := range r.plugins.clients {
		client.Kill()
	}
	clients := map[string]*plugin.Client{}
	providers := map[string]usecase.PaymentProvider{}
	for _, spec := range specs {
		if spec.Path == "" {
			continue
		}
		client := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig: paymentplugin.Handshake,
			Plugins: map[string]plugin.Plugin{
				paymentplugin.ProviderPluginName: &paymentplugin.ProviderPlugin{},
			},
			Cmd:              exec.Command(spec.Path),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
		})
		rpcClient, err := client.Client()
		if err != nil {
			client.Kill()
			return nil, err
		}
		raw, err := rpcClient.Dispense(paymentplugin.ProviderPluginName)
		if err != nil {
			client.Kill()
			return nil, err
		}
		provider, ok := raw.(paymentplugin.Provider)
		if !ok {
			client.Kill()
			return nil, errors.New("invalid payment provider plugin")
		}
		wrapped := &pluginProvider{provider: provider}
		providers[wrapped.Key()] = wrapped
		clients[wrapped.Key()] = client
	}
	r.plugins = pluginState{hash: hash, clients: clients, providers: providers}
	return &r.plugins, nil
}

func pluginSpecHash(specs []pluginSpec) string {
	items := make([]string, 0, len(specs))
	for _, spec := range specs {
		items = append(items, spec.Key+"="+spec.Path)
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

func (r *Registry) copyDirSpecs() []pluginSpec {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.dirSpecs) == 0 {
		return nil
	}
	out := make([]pluginSpec, len(r.dirSpecs))
	copy(out, r.dirSpecs)
	return out
}

func mergePluginSpecs(specs []pluginSpec, discovered []pluginSpec) []pluginSpec {
	out := make([]pluginSpec, 0, len(specs)+len(discovered))
	seen := map[string]struct{}{}
	for _, spec := range specs {
		key := strings.TrimSpace(spec.Key)
		if key == "" {
			key = strings.TrimSuffix(filepath.Base(spec.Path), filepath.Ext(spec.Path))
		}
		if key == "" {
			continue
		}
		spec.Key = key
		out = append(out, spec)
		seen[key] = struct{}{}
	}
	for _, spec := range discovered {
		if _, ok := seen[spec.Key]; ok {
			continue
		}
		out = append(out, spec)
	}
	return out
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

func (p *pluginProvider) CreatePayment(ctx context.Context, req usecase.PaymentCreateRequest) (usecase.PaymentCreateResult, error) {
	return p.provider.CreatePayment(req)
}

func (p *pluginProvider) VerifyNotify(ctx context.Context, req usecase.RawHTTPRequest) (usecase.PaymentNotifyResult, error) {
	return p.provider.VerifyNotify(rawToParams(req))
}
