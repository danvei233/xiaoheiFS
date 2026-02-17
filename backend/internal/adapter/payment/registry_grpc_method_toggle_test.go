package payment

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"xiaoheiplay/internal/adapter/plugins"
	"xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/cryptox"
	"xiaoheiplay/internal/testutil"
)

func TestRegistry_GRPCPaymentMethodToggle_Disable(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, "plugins")
	dir := filepath.Join(baseDir, "payment", "ezpay")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{
  "plugin_id": "ezpay",
  "name": "EZPay",
  "version": "1.0.0",
  "capabilities": { "payment": { "methods": ["alipay","wechat","qq"] } }
}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if err := repo.UpsertPluginInstallation(ctx, &domain.PluginInstallation{
		Category:   "payment",
		PluginID:   "ezpay",
		InstanceID: plugins.DefaultInstanceID,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("upsert plugin installation: %v", err)
	}

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	cipher, err := cryptox.NewAESGCM(base64.RawURLEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	pluginMgr := plugins.NewManager(baseDir, repo, cipher, nil)

	reg := NewRegistry(repo)
	reg.SetPluginManager(pluginMgr)
	reg.SetPluginPaymentMethodRepo(repo)

	if err := repo.UpsertPluginPaymentMethod(ctx, &domain.PluginPaymentMethod{
		Category:   "payment",
		PluginID:   "ezpay",
		InstanceID: plugins.DefaultInstanceID,
		Method:     "alipay",
		Enabled:    false,
	}); err != nil {
		t.Fatalf("upsert payment method: %v", err)
	}

	if _, err := reg.GetProvider(ctx, "ezpay.alipay"); err != shared.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}
