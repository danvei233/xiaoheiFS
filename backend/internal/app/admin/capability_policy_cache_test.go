package admin

import (
	"context"
	"sync"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
)

type capabilityPolicyCacheRepo struct {
	mu    sync.Mutex
	items map[domain.CapabilityPolicyScope]map[int64]domain.CapabilityPolicy
	lists int
}

func (r *capabilityPolicyCacheRepo) ListCapabilityPolicies(_ context.Context, scope domain.CapabilityPolicyScope) (map[int64]domain.CapabilityPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lists++
	out := map[int64]domain.CapabilityPolicy{}
	for id, item := range r.items[scope] {
		out[id] = item
	}
	return out, nil
}

func (r *capabilityPolicyCacheRepo) GetCapabilityPolicy(context.Context, domain.CapabilityPolicyScope, int64) (domain.CapabilityPolicy, error) {
	return domain.CapabilityPolicy{}, domain.ErrNotFound
}

func (r *capabilityPolicyCacheRepo) UpsertCapabilityPolicy(_ context.Context, scope domain.CapabilityPolicyScope, itemID int64, policy domain.CapabilityPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.items[scope] == nil {
		r.items[scope] = map[int64]domain.CapabilityPolicy{}
	}
	r.items[scope][itemID] = policy
	return nil
}

func TestCapabilityPolicyDisplayCache_InvalidatesAndConvergesAcrossInstances(t *testing.T) {
	enabled := true
	disabled := false
	repository := &capabilityPolicyCacheRepo{items: map[domain.CapabilityPolicyScope]map[int64]domain.CapabilityPolicy{
		domain.CapabilityPolicyScopePackage: {7: {ResizeEnabled: &enabled}},
	}}
	newService := func() *Service {
		return &Service{capabilityPolicy: repository, capabilityCache: map[domain.CapabilityPolicyScope]capabilityPolicyCacheEntry{}}
	}
	instanceA := newService()
	instanceB := newService()
	ctx := context.Background()

	first, err := instanceA.ListCapabilityPolicies(ctx, domain.CapabilityPolicyScopePackage)
	if err != nil || first[7].ResizeEnabled == nil || !*first[7].ResizeEnabled {
		t.Fatalf("initial list: %+v err=%v", first, err)
	}
	first[7] = domain.CapabilityPolicy{ResizeEnabled: &disabled}
	second, err := instanceA.ListCapabilityPolicies(ctx, domain.CapabilityPolicyScopePackage)
	if err != nil || second[7].ResizeEnabled == nil || !*second[7].ResizeEnabled {
		t.Fatalf("cached copy was mutated by caller: %+v err=%v", second, err)
	}
	if repository.lists != 1 {
		t.Fatalf("expected one repository list, got %d", repository.lists)
	}

	if err := instanceB.UpsertCapabilityPolicy(ctx, domain.CapabilityPolicyScopePackage, 7, domain.CapabilityPolicy{ResizeEnabled: &disabled}); err != nil {
		t.Fatal(err)
	}
	instanceA.capabilityMu.Lock()
	entry := instanceA.capabilityCache[domain.CapabilityPolicyScopePackage]
	entry.expiresAt = time.Now().Add(-time.Second)
	instanceA.capabilityCache[domain.CapabilityPolicyScopePackage] = entry
	instanceA.capabilityMu.Unlock()
	converged, err := instanceA.ListCapabilityPolicies(ctx, domain.CapabilityPolicyScopePackage)
	if err != nil || converged[7].ResizeEnabled == nil || *converged[7].ResizeEnabled {
		t.Fatalf("cache did not converge after ttl: %+v err=%v", converged, err)
	}
}
