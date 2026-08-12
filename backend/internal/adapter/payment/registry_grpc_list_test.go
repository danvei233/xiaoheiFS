package payment

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	plugins "xiaoheiplay/internal/adapter/plugins/core"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	pluginv1 "xiaoheiplay/plugin/v1"
)

// fakePluginManager is an in-process stand-in for *plugins.Manager so tests can
// exercise the gRPC provider listing without launching real plugin binaries.
type fakePluginManager struct {
	items    []plugins.ListItem
	clientOK bool
}

func (f *fakePluginManager) List(ctx context.Context) ([]plugins.ListItem, error) {
	return f.items, nil
}

func (f *fakePluginManager) GetPaymentClient(category, pluginID, instanceID string) (pluginv1.PaymentServiceClient, bool) {
	return nil, f.clientOK
}

type fakePluginPaymentMethodRepo struct {
	items []domain.PluginPaymentMethod
	err   error
}

func (f *fakePluginPaymentMethodRepo) ListPluginPaymentMethods(ctx context.Context, category, pluginID, instanceID string) ([]domain.PluginPaymentMethod, error) {
	return f.items, f.err
}

func (f *fakePluginPaymentMethodRepo) UpsertPluginPaymentMethod(ctx context.Context, m *domain.PluginPaymentMethod) error {
	return nil
}

func (f *fakePluginPaymentMethodRepo) DeletePluginPaymentMethod(ctx context.Context, category, pluginID, instanceID, method string) error {
	return nil
}

func providerKeys(providers []appshared.PaymentProvider) []string {
	keys := make([]string, 0, len(providers))
	for _, p := range providers {
		keys = append(keys, p.Key())
	}
	return keys
}

func containsKey(keys []string, want string) bool {
	for _, k := range keys {
		if k == want {
			return true
		}
	}
	return false
}

func ezpayListItem(t *testing.T, methods []string) plugins.ListItem {
	t.Helper()
	var m plugins.Manifest
	raw, err := json.Marshal(map[string]any{
		"plugin_id": "ezpay",
		"name":      "EZPay",
		"version":   "1.0.0",
		"capabilities": map[string]any{
			"payment": map[string]any{"methods": methods},
		},
	})
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	return plugins.ListItem{
		Category:     "payment",
		PluginID:     "ezpay",
		InstanceID:   plugins.DefaultInstanceID,
		Name:         "EZPay",
		Enabled:      true,
		Loaded:       true,
		Capabilities: m,
	}
}

// testGRPCRegistry builds a Registry whose gRPC manager advertises a loaded
// ezpay plugin with the given methods and whose method repo disables the given
// method names.
func testGRPCRegistry(t *testing.T, methods []string, disabled []string) *Registry {
	t.Helper()
	items := make([]domain.PluginPaymentMethod, 0, len(disabled))
	for _, d := range disabled {
		items = append(items, domain.PluginPaymentMethod{
			Category:   "payment",
			PluginID:   "ezpay",
			InstanceID: plugins.DefaultInstanceID,
			Method:     d,
			Enabled:    false,
		})
	}
	reg := NewRegistry(nil)
	reg.SetPluginPaymentMethodRepo(&fakePluginPaymentMethodRepo{items: items})
	reg.SetPluginManager(&fakePluginManager{items: []plugins.ListItem{ezpayListItem(t, methods)}, clientOK: true})
	return reg
}

// M1: includeDisabled=true must include gRPC payment methods (including ones
// disabled per-method), and includeDisabled=false must hide disabled ones.
func TestRegistry_ListProviders_IncludeDisabledListsGRPCMethods(t *testing.T) {
	reg := testGRPCRegistry(t, []string{"alipay", "wechat"}, []string{"alipay"})
	ctx := context.Background()

	all, err := reg.ListProviders(ctx, true)
	if err != nil {
		t.Fatalf("list providers (include disabled): %v", err)
	}
	keys := providerKeys(all)
	if !containsKey(keys, "ezpay.alipay") {
		t.Fatalf("ListProviders(includeDisabled=true) must include disabled gRPC method ezpay.alipay, got %v", keys)
	}
	if !containsKey(keys, "ezpay.wechat") {
		t.Fatalf("ListProviders(includeDisabled=true) missing enabled gRPC method ezpay.wechat, got %v", keys)
	}

	enabled, err := reg.ListProviders(ctx, false)
	if err != nil {
		t.Fatalf("list providers (enabled only): %v", err)
	}
	keys = providerKeys(enabled)
	if containsKey(keys, "ezpay.alipay") {
		t.Fatalf("ListProviders(includeDisabled=false) must hide disabled gRPC method ezpay.alipay, got %v", keys)
	}
	if !containsKey(keys, "ezpay.wechat") {
		t.Fatalf("ListProviders(includeDisabled=false) missing enabled gRPC method ezpay.wechat, got %v", keys)
	}
}

// L3: grpcProviderByKey must tolerate surrounding whitespace in manifest method
// names the same way grpcProviders does.
func TestRegistry_GetProvider_TrimsManifestMethodWhitespace(t *testing.T) {
	reg := testGRPCRegistry(t, []string{"  alipay  ", "wechat"}, nil)
	ctx := context.Background()

	p, err := reg.GetProvider(ctx, "ezpay.alipay")
	if err != nil {
		t.Fatalf("get provider for method with whitespace in manifest: %v", err)
	}
	if p.Key() != "ezpay.alipay" {
		t.Fatalf("unexpected provider key %q", p.Key())
	}
}

// L2: when the payment-method repo query fails, methods must fail closed
// (unreachable) instead of becoming reachable.
func TestRegistry_GetProvider_MethodRepoErrorFailsClosed(t *testing.T) {
	reg := NewRegistry(nil)
	reg.SetPluginPaymentMethodRepo(&fakePluginPaymentMethodRepo{err: errors.New("db unavailable")})
	reg.SetPluginManager(&fakePluginManager{items: []plugins.ListItem{ezpayListItem(t, []string{"alipay"})}, clientOK: true})
	ctx := context.Background()

	if _, err := reg.GetProvider(ctx, "ezpay.alipay"); err == nil {
		t.Fatalf("expected fail-closed error when method repo query fails")
	}
}

// L2: with an authoritative but empty method table, gRPC methods keep the
// existing fail-open default (a gRPC plugin is explicitly installed + enabled).
func TestRegistry_GetProvider_EmptyMethodTableKeepsMethodAvailable(t *testing.T) {
	reg := testGRPCRegistry(t, []string{"alipay", "wechat"}, nil)
	ctx := context.Background()

	p, err := reg.GetProvider(ctx, "ezpay.alipay")
	if err != nil {
		t.Fatalf("gRPC method with no method-table row should stay available, got %v", err)
	}
	if p.Key() != "ezpay.alipay" {
		t.Fatalf("unexpected provider key %q", p.Key())
	}
}
