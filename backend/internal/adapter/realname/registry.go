package realname

import "xiaoheiplay/internal/usecase"

type Registry struct {
	providers map[string]usecase.RealNameProvider
}

func NewRegistry() *Registry {
	reg := &Registry{providers: map[string]usecase.RealNameProvider{}}
	reg.Register(&IDCardCNProvider{})
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
