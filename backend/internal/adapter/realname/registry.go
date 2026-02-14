package realname

import "xiaoheiplay/internal/usecase"

type Registry struct {
	providers map[string]usecase.RealNameProvider
}

func NewRegistry(settings ...usecase.SettingsRepository) *Registry {
	var settingRepo usecase.SettingsRepository
	if len(settings) > 0 {
		settingRepo = settings[0]
	}
	reg := &Registry{providers: map[string]usecase.RealNameProvider{}}
	reg.Register(&IDCardCNProvider{})
	reg.Register(NewMangzhuRealNameProvider(settingRepo))
	return reg
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
	return nil, usecase.ErrNotFound
}

func (r *Registry) ListProviders() []usecase.RealNameProvider {
	out := make([]usecase.RealNameProvider, 0, len(r.providers))
	for _, provider := range r.providers {
		out = append(out, provider)
	}
	return out
}
