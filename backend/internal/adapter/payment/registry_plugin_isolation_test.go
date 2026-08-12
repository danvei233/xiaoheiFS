package payment

import (
	"context"
	"testing"

	plugins "xiaoheiplay/internal/adapter/plugins/core"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func seedBrokenNetRPCPluginSpec(t *testing.T, repo interface {
	UpsertSetting(ctx context.Context, s domain.Setting) error
}) {
	t.Helper()
	if err := repo.UpsertSetting(context.Background(), domain.Setting{
		Key:       settingPaymentPlugins,
		ValueJSON: `[{"key":"bad","path":"/nonexistent/plugin-binary"}]`,
	}); err != nil {
		t.Fatalf("seed payment_plugins: %v", err)
	}
}

// M4: a broken legacy net-RPC plugin spec must not take down the whole provider
// list; builtins stay listed.
func TestRegistry_ListProviders_BadNetRPCPluginSpecIsolated(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seedBrokenNetRPCPluginSpec(t, repo)
	ctx := context.Background()

	reg := NewRegistry(repo)
	providers, err := reg.ListProviders(ctx, true)
	if err != nil {
		t.Fatalf("ListProviders must tolerate a broken net-rpc plugin spec, got %v", err)
	}
	keys := providerKeys(providers)
	if !containsKey(keys, "approval") || !containsKey(keys, "balance") {
		t.Fatalf("builtins must remain listed, got %v", keys)
	}
}

// M4: GetProvider for a gRPC key must not be blocked by a broken net-RPC plugin
// spec that fails while loading.
func TestRegistry_GetProvider_BadNetRPCPluginDoesNotBlockGRPC(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seedBrokenNetRPCPluginSpec(t, repo)
	ctx := context.Background()

	reg := NewRegistry(repo)
	reg.SetPluginPaymentMethodRepo(&fakePluginPaymentMethodRepo{})
	reg.SetPluginManager(&fakePluginManager{items: []plugins.ListItem{ezpayListItem(t, []string{"alipay", "wechat"})}, clientOK: true})

	p, err := reg.GetProvider(ctx, "ezpay.wechat")
	if err != nil {
		t.Fatalf("GetProvider must not be blocked by a broken net-rpc plugin, got %v", err)
	}
	if p.Key() != "ezpay.wechat" {
		t.Fatalf("unexpected provider key %q", p.Key())
	}
}

// M2: reserved builtin keys can never be served by a plugin.
func TestIsReservedProviderKey(t *testing.T) {
	for _, k := range []string{"approval", "balance"} {
		if !isReservedProviderKey(k) {
			t.Fatalf("expected %q to be a reserved provider key", k)
		}
	}
	for _, k := range []string{"alipay", "ezpay.alipay", "", "wechat"} {
		if isReservedProviderKey(k) {
			t.Fatalf("did not expect %q to be a reserved provider key", k)
		}
	}
}
