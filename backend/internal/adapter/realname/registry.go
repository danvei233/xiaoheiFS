package realname

import (
	"context"
	"strings"

	plugins "xiaoheiplay/internal/adapter/plugins"
	"xiaoheiplay/internal/usecase"
)

type Registry struct {
	providers map[string]usecase.RealNameProvider
	pluginMgr *plugins.Manager
}

func NewRegistry(settings ...usecase.SettingsRepository) *Registry {
	var settingRepo usecase.SettingsRepository
	if len(settings) > 0 {
		settingRepo = settings[0]
	}
	reg := &Registry{providers: map[string]usecase.RealNameProvider{}}
	reg.Register(&IDCardCNProvider{})
	_ = settingRepo
	return reg
}

func (r *Registry) SetPluginManager(mgr *plugins.Manager) {
	r.pluginMgr = mgr
}

func (r *Registry) Register(provider usecase.RealNameProvider) {
	if provider == nil {
		return
	}
	r.providers[provider.Key()] = provider
}

func (r *Registry) GetProvider(key string) (usecase.RealNameProvider, error) {
	if provider, ok := r.providers[key]; ok {
		return provider, nil
	}
	if r.pluginMgr != nil {
		if pluginID, instanceID, ok := parsePluginProviderKey(key); ok {
			if p := r.pluginProviderByID(context.Background(), pluginID, instanceID); p != nil {
				return p, nil
			}
		}
	}
	return nil, usecase.ErrNotFound
}

func (r *Registry) ListProviders() []usecase.RealNameProvider {
	out := make([]usecase.RealNameProvider, 0, len(r.providers))
	for _, provider := range r.providers {
		out = append(out, provider)
	}
	if r.pluginMgr != nil {
		out = append(out, r.pluginProviders(context.Background())...)
	}
	return out
}

func (r *Registry) pluginProviders(ctx context.Context) []usecase.RealNameProvider {
	items, err := r.pluginMgr.List(ctx)
	if err != nil {
		return nil
	}
	out := make([]usecase.RealNameProvider, 0)
	for _, it := range items {
		if strings.TrimSpace(it.Category) != "kyc" || !it.Enabled || !it.Loaded {
			continue
		}
		if it.Capabilities.Capabilities.KYC == nil {
			continue
		}
		if _, ok := r.pluginMgr.GetKYCClient(it.Category, it.PluginID, it.InstanceID); !ok {
			continue
		}
		out = append(out, &kycPluginProvider{
			mgr:        r.pluginMgr,
			pluginID:   it.PluginID,
			instanceID: it.InstanceID,
			name:       it.Name,
			canQuery:   it.Capabilities.Capabilities.KYC.QueryResult,
		})
	}
	return out
}

func (r *Registry) pluginProviderByID(ctx context.Context, pluginID, instanceID string) usecase.RealNameProvider {
	pluginID = strings.TrimSpace(pluginID)
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = plugins.DefaultInstanceID
	}
	if pluginID == "" {
		return nil
	}
	items, err := r.pluginMgr.List(ctx)
	if err != nil {
		return nil
	}
	for _, it := range items {
		if strings.TrimSpace(it.Category) != "kyc" || !it.Enabled || !it.Loaded {
			continue
		}
		if it.PluginID != pluginID || it.InstanceID != instanceID {
			continue
		}
		if it.Capabilities.Capabilities.KYC == nil {
			return nil
		}
		if _, ok := r.pluginMgr.GetKYCClient(it.Category, it.PluginID, it.InstanceID); !ok {
			return nil
		}
		return &kycPluginProvider{
			mgr:        r.pluginMgr,
			pluginID:   it.PluginID,
			instanceID: it.InstanceID,
			name:       it.Name,
			canQuery:   it.Capabilities.Capabilities.KYC.QueryResult,
		}
	}
	return nil
}
