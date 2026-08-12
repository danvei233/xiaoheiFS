package payment

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestRegistry_PluginDirScan(t *testing.T) {
	dir := t.TempDir()
	pluginPath := filepath.Join(dir, "demo.exe")
	if err := os.WriteFile(pluginPath, []byte("demo"), 0o755); err != nil {
		t.Fatalf("write plugin: %v", err)
	}

	specs, err := scanPluginDir(dir)
	if err != nil {
		t.Fatalf("scan plugin dir: %v", err)
	}
	if len(specs) == 0 {
		t.Fatalf("expected plugin specs")
	}
	if pluginSpecHash(specs) == "" {
		t.Fatalf("expected plugin spec hash")
	}

	reg := NewRegistry(nil)
	reg.SetPluginDir(dir)
	ctx, cancel := context.WithCancel(context.Background())
	if err := reg.StartWatcher(ctx, dir); err != nil {
		cancel()
		t.Fatalf("start watcher: %v", err)
	}
	cancel()
	time.Sleep(10 * time.Millisecond)

	if err := reg.refreshDirSpecs(); err != nil {
		t.Fatalf("refresh dir specs: %v", err)
	}
	if _, err := reg.ensurePlugins(nil); err != nil {
		t.Fatalf("ensure plugins: %v", err)
	}
}

// TestRegistry_DoesNotAutoSpawnUnlistedDirExecutable guards the trust boundary
// for the legacy net-RPC payment plugin dir: a raw executable manually dropped
// into the plugin dir must NEVER be auto-loaded/spawned. Only plugins explicitly
// listed in the payment_plugins setting are execution sources. This is the
// regression for the residual "StartWatcher must not auto-execute unverified
// files" item.
func TestRegistry_DoesNotAutoSpawnUnlistedDirExecutable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "evil.exe"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write dropped executable: %v", err)
	}

	_, repo := testutil.NewTestDB(t, false)
	// payment_plugins is seeded to [] — nothing has explicitly authorized the
	// dropped file.
	reg := NewRegistry(repo)
	reg.SetPluginDir(dir)
	ctx := context.Background()
	if err := reg.refreshDirSpecs(); err != nil {
		t.Fatalf("refresh dir specs: %v", err)
	}

	specs := reg.loadPluginSpecs(ctx)
	if len(specs) != 0 {
		t.Fatalf("unlisted dir executable must not be auto-loaded as a plugin spec: %v", specs)
	}
	state, err := reg.ensurePlugins(specs)
	if err != nil {
		t.Fatalf("ensure plugins: %v", err)
	}
	if len(state.providers) != 0 {
		t.Fatalf("unlisted dir executable was spawned: %v", state.providers)
	}
}

// TestRegistry_RefusesUnanchoredConfiguredPlugin confirms the payment_plugins
// setting is no longer a blind execution source: an entry that does not carry a
// content-digest anchor is refused (never spawned). An operator who configures a
// legacy net-RPC plugin must explicitly pin the sha256 of the executable they
// authorize; anything else fails closed.
func TestRegistry_RefusesUnanchoredConfiguredPlugin(t *testing.T) {
	dir := t.TempDir()
	pluginPath := filepath.Join(dir, "demo")
	if err := os.WriteFile(pluginPath, []byte("demo-binary"), 0o755); err != nil {
		t.Fatalf("write plugin: %v", err)
	}
	_, repo := testutil.NewTestDB(t, false)
	if err := repo.UpsertSetting(context.Background(), domain.Setting{
		Key:       settingPaymentPlugins,
		ValueJSON: `[{"key":"configured","path":"` + pluginPath + `"}]`,
	}); err != nil {
		t.Fatalf("seed payment_plugins: %v", err)
	}

	reg := NewRegistry(repo)
	specs := reg.loadPluginSpecs(context.Background())
	if len(specs) != 0 {
		t.Fatalf("unanchored payment_plugins entry must be refused, got: %v", specs)
	}
	state, err := reg.ensurePlugins(specs)
	if err != nil {
		t.Fatalf("ensure plugins: %v", err)
	}
	if len(state.providers) != 0 {
		t.Fatalf("unanchored plugin was spawned: %v", state.providers)
	}
}

// TestRegistry_LoadsAnchoredConfiguredPlugin confirms a payment_plugins entry
// that carries a digest matching the current file bytes is still authorized to
// load (the legacy mechanism remains available to operators who pin the digest).
func TestRegistry_LoadsAnchoredConfiguredPlugin(t *testing.T) {
	dir := t.TempDir()
	pluginPath := filepath.Join(dir, "demo")
	if err := os.WriteFile(pluginPath, []byte("demo-binary"), 0o755); err != nil {
		t.Fatalf("write plugin: %v", err)
	}
	sum, err := fileSHA256(pluginPath)
	if err != nil {
		t.Fatalf("hash plugin: %v", err)
	}
	_, repo := testutil.NewTestDB(t, false)
	if err := repo.UpsertSetting(context.Background(), domain.Setting{
		Key:       settingPaymentPlugins,
		ValueJSON: `[{"key":"configured","path":"` + pluginPath + `","digest":"` + sum + `"}]`,
	}); err != nil {
		t.Fatalf("seed payment_plugins: %v", err)
	}

	reg := NewRegistry(repo)
	specs := reg.loadPluginSpecs(context.Background())
	if len(specs) != 1 || specs[0].Key != "configured" || specs[0].Digest != sum {
		t.Fatalf("anchored payment_plugins entry was not loaded: %v", specs)
	}
}

// TestRegistry_RefusesTamperedAnchoredPlugin confirms the anchor is enforced at
// load time: a payment_plugins entry whose executable was replaced on disk after
// the digest was pinned is refused, so a file-write attacker cannot swap an
// anchored executable for arbitrary code and have it spawned.
func TestRegistry_RefusesTamperedAnchoredPlugin(t *testing.T) {
	dir := t.TempDir()
	pluginPath := filepath.Join(dir, "demo")
	if err := os.WriteFile(pluginPath, []byte("demo-binary"), 0o755); err != nil {
		t.Fatalf("write plugin: %v", err)
	}
	sum, err := fileSHA256(pluginPath)
	if err != nil {
		t.Fatalf("hash plugin: %v", err)
	}
	_, repo := testutil.NewTestDB(t, false)
	if err := repo.UpsertSetting(context.Background(), domain.Setting{
		Key:       settingPaymentPlugins,
		ValueJSON: `[{"key":"configured","path":"` + pluginPath + `","digest":"` + sum + `"}]`,
	}); err != nil {
		t.Fatalf("seed payment_plugins: %v", err)
	}

	// Replace the executable after it was anchored.
	if err := os.WriteFile(pluginPath, []byte("attacker-payload"), 0o755); err != nil {
		t.Fatalf("tamper plugin: %v", err)
	}

	reg := NewRegistry(repo)
	specs := reg.loadPluginSpecs(context.Background())
	if len(specs) != 0 {
		t.Fatalf("tampered anchored plugin must be refused, got: %v", specs)
	}
	state, err := reg.ensurePlugins(specs)
	if err != nil {
		t.Fatalf("ensure plugins: %v", err)
	}
	if len(state.providers) != 0 {
		t.Fatalf("tampered anchored plugin was spawned: %v", state.providers)
	}
}
