package payment

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/go-plugin"

	plugins "xiaoheiplay/internal/adapter/plugins/core"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// --- fake legacy net-RPC client infrastructure ---------------------------------
//
// The fake client handle captures the exec target and records Kill() calls so
// tests can assert both the verified-snapshot binding and the StopAll lifecycle
// without spawning real plugin processes.

type fakeLegacyClient struct {
	cfg    *plugin.ClientConfig
	killed bool
}

func (c *fakeLegacyClient) Client() (plugin.ClientProtocol, error) {
	return &fakeLegacyProtocol{}, nil
}

func (c *fakeLegacyClient) Kill() { c.killed = true }

type fakeLegacyProtocol struct{}

func (p *fakeLegacyProtocol) Dispense(string) (interface{}, error) {
	return &fakeLegacyProvider{key: "demo_pay"}, nil
}

func (p *fakeLegacyProtocol) Ping() error  { return nil }
func (p *fakeLegacyProtocol) Close() error { return nil }

type fakeLegacyProvider struct{ key string }

func (p *fakeLegacyProvider) Key() string            { return p.key }
func (p *fakeLegacyProvider) Name() string           { return "Demo Pay" }
func (p *fakeLegacyProvider) SchemaJSON() string     { return "" }
func (p *fakeLegacyProvider) SetConfig(string) error { return nil }
func (p *fakeLegacyProvider) CreatePayment(appshared.PaymentCreateRequest) (appshared.PaymentCreateResult, error) {
	return appshared.PaymentCreateResult{}, nil
}
func (p *fakeLegacyProvider) VerifyNotify(map[string]string) (appshared.PaymentNotifyResult, error) {
	return appshared.PaymentNotifyResult{}, nil
}
func (p *fakeLegacyProvider) VerifyNotifyV2(appshared.RawHTTPRequest) (appshared.PaymentNotifyResult, error) {
	return appshared.PaymentNotifyResult{}, nil
}

type legacyClientFactory struct {
	mu      sync.Mutex
	clients []*fakeLegacyClient
}

func (f *legacyClientFactory) new(cfg *plugin.ClientConfig) pluginClientHandle {
	c := &fakeLegacyClient{cfg: cfg}
	f.mu.Lock()
	f.clients = append(f.clients, c)
	f.mu.Unlock()
	return c
}

func (f *legacyClientFactory) all() []*fakeLegacyClient {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*fakeLegacyClient{}, f.clients...)
}

func (f *legacyClientFactory) killedCount() int {
	n := 0
	for _, c := range f.all() {
		if c.killed {
			n++
		}
	}
	return n
}

type settingsUpserter interface {
	UpsertSetting(ctx context.Context, s domain.Setting) error
}

func seedAnchoredPaymentPlugin(t *testing.T, repo settingsUpserter, path, key string) string {
	t.Helper()
	sum, err := fileSHA256(path)
	if err != nil {
		t.Fatalf("hash plugin: %v", err)
	}
	if err := repo.UpsertSetting(context.Background(), domain.Setting{
		Key:       settingPaymentPlugins,
		ValueJSON: fmt.Sprintf(`[{"key":%q,"path":%q,"digest":%q}]`, key, path, sum),
	}); err != nil {
		t.Fatalf("seed payment_plugins: %v", err)
	}
	return sum
}

// TestRegistry_LegacyPluginExecsFromVerifiedSnapshot is the hash-to-exec TOCTOU
// regression for the legacy net-RPC payment path: the exec target must be a
// verified content-addressed snapshot copy, never the original spec.Path, and
// the transport must negotiate AutoMTLS with a pinned 0700 socket dir.
func TestRegistry_LegacyPluginExecsFromVerifiedSnapshot(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "legacy")
	if err := os.WriteFile(binPath, []byte("good-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, repo := testutil.NewTestDB(t, false)
	sum := seedAnchoredPaymentPlugin(t, repo, binPath, "demo_pay")

	factory := &legacyClientFactory{}
	reg := NewRegistry(repo)
	reg.SetPluginDir(dir)
	reg.newClient = factory.new

	providers, err := reg.ListProviders(context.Background(), true)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if !containsKey(providerKeys(providers), "demo_pay") {
		t.Fatalf("expected demo_pay provider, got %v", providerKeys(providers))
	}

	clients := factory.all()
	if len(clients) != 1 {
		t.Fatalf("expected 1 spawned legacy client, got %d", len(clients))
	}
	cmd := clients[0].cfg.Cmd
	snapPath := filepath.Join(dir, paymentSnapshotRootName, "payment", "demo_pay", strings.ToLower(sum))
	assertExecBoundToVerifiedSnapshot(t, cmd, binPath, snapPath, "good-binary")

	// The transport must request AutoMTLS (mutual peer auth) and pin the control
	// socket to a dedicated directory.
	if !clients[0].cfg.AutoMTLS {
		t.Fatal("legacy plugin client must request AutoMTLS for peer authentication")
	}
	if clients[0].cfg.UnixSocketConfig == nil {
		t.Fatal("legacy plugin client must pin its unix socket directory")
	}
	reg.StopAll()
}

// TestRegistry_LegacyPluginSkippedOnNonLinux pins the P2-51 fail-closed platform
// guard: on platforms without /proc/self/fd (non-Linux) the legacy net-RPC
// payment plugin must be SKIPPED with a logged reason — never failed as a whole
// set and never degraded to a path-based exec that would reopen the hash-to-exec
// TOCTOU window. The platform decision is the legacyExecSupported package var,
// stubbed here to simulate non-Linux regardless of the host OS.
func TestRegistry_LegacyPluginSkippedOnNonLinux(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "legacy")
	if err := os.WriteFile(binPath, []byte("good-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, repo := testutil.NewTestDB(t, false)
	seedAnchoredPaymentPlugin(t, repo, binPath, "demo_pay")

	old := legacyExecSupported
	legacyExecSupported = false
	defer func() { legacyExecSupported = old }()

	factory := &legacyClientFactory{}
	reg := NewRegistry(repo)
	reg.SetPluginDir(dir)
	reg.newClient = factory.new

	providers, err := reg.ListProviders(context.Background(), true)
	if err != nil {
		t.Fatalf("non-Linux guard must skip the entry, not fail the whole registry: %v", err)
	}
	if containsKey(providerKeys(providers), "demo_pay") {
		t.Fatalf("legacy plugin must be skipped on non-Linux, got providers: %v", providerKeys(providers))
	}
	if n := len(factory.all()); n != 0 {
		t.Fatalf("legacy plugin was spawned on non-Linux: %d clients", n)
	}
	reg.StopAll()
}

// assertExecBoundToVerifiedSnapshot verifies that a legacy plugin exec is bound
// to the verified snapshot bytes: on Linux the executable is the open handle of
// the content-addressed snapshot inode (so a swap of either the original
// spec.Path or the snapshot path cannot change what runs); on other platforms
// the snapshot path itself is executed (the documented fallback).
func assertExecBoundToVerifiedSnapshot(t *testing.T, cmd *exec.Cmd, binPath, snapPath, want string) {
	t.Helper()
	if runtime.GOOS == "linux" && len(cmd.ExtraFiles) > 0 {
		if cmd.Path != "/proc/self/fd/3" {
			t.Fatalf("fd-bound exec path = %q, want /proc/self/fd/3", cmd.Path)
		}
		f := cmd.ExtraFiles[0]
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(f)
		if err != nil {
			t.Fatalf("read exec handle: %v", err)
		}
		if string(got) != want {
			t.Fatalf("executed handle must contain the verified bytes, got %q", got)
		}
		fi, err := f.Stat()
		if err != nil {
			t.Fatal(err)
		}
		si, err := os.Stat(snapPath)
		if err != nil {
			t.Fatalf("snapshot file missing: %v", err)
		}
		if !os.SameFile(fi, si) {
			t.Fatal("exec handle must reference the content-addressed snapshot inode, never the original spec.Path")
		}
		return
	}
	// Non-Linux fallback: the exec target is the verified snapshot path.
	cmdPath := filepath.Clean(cmd.Path)
	snapRoot := filepath.Clean(filepath.Dir(snapPath))
	if !strings.HasPrefix(cmdPath, snapRoot+string(os.PathSeparator)) {
		t.Fatalf("exec target %q is outside the verified snapshot root %q", cmdPath, snapRoot)
	}
	if cmdPath == filepath.Clean(binPath) {
		t.Fatal("exec target must never be the original spec.Path")
	}
	got, err := os.ReadFile(cmdPath)
	if err != nil {
		t.Fatalf("read snapshot executable: %v", err)
	}
	if string(got) != want {
		t.Fatalf("executed snapshot must contain the verified bytes, got %q", got)
	}
}

// TestRegistry_LegacyPluginSwapInWindowRunsVerifiedSnapshot widens the
// digest-check-to-spawn window by swapping the original executable AND the
// verified snapshot path right before spawn: only the open-handle binding
// (Linux) survives both swaps — the exec runs the verified bytes from the
// pinned inode, never the attacker payload that replaced the path.
func TestRegistry_LegacyPluginSwapInWindowRunsVerifiedSnapshot(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "legacy")
	if err := os.WriteFile(binPath, []byte("good-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, repo := testutil.NewTestDB(t, false)
	sum := seedAnchoredPaymentPlugin(t, repo, binPath, "demo_pay")

	factory := &legacyClientFactory{}
	reg := NewRegistry(repo)
	reg.SetPluginDir(dir)
	reg.newClient = factory.new
	reg.beforeSpawn = func() {
		if err := os.WriteFile(binPath, []byte("attacker-payload"), 0o755); err != nil {
			t.Errorf("swap original in spawn window: %v", err)
		}
		// Also swap the verified snapshot path itself: a path-based exec would run
		// these attacker bytes. The open-handle binding must still run good-binary.
		// The snapshot file is read-only (0500), so the same-user attacker unlinks
		// it and drops a replacement at the same path.
		snapFile := filepath.Join(dir, paymentSnapshotRootName, "payment", "demo_pay", strings.ToLower(sum))
		if err := os.Remove(snapFile); err != nil {
			t.Errorf("unlink snapshot in spawn window: %v", err)
			return
		}
		if err := os.WriteFile(snapFile, []byte("attacker-payload"), 0o500); err != nil {
			t.Errorf("replace snapshot in spawn window: %v", err)
		}
	}

	providers, err := reg.ListProviders(context.Background(), true)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if !containsKey(providerKeys(providers), "demo_pay") {
		t.Fatalf("expected demo_pay provider to load from the verified snapshot, got %v", providerKeys(providers))
	}
	clients := factory.all()
	if len(clients) != 1 {
		t.Fatalf("expected 1 spawned legacy client, got %d", len(clients))
	}
	cmd := clients[0].cfg.Cmd
	if runtime.GOOS != "linux" {
		t.Skip("open-handle exec binding is Linux-only; the path-based fallback cannot survive a snapshot-path swap")
	}
	if len(cmd.ExtraFiles) == 0 {
		t.Fatal("legacy exec must be bound to the open verified-snapshot handle (fd binding) to survive a snapshot-path swap")
	}
	if cmd.Path != "/proc/self/fd/3" {
		t.Fatalf("fd-bound exec path = %q, want /proc/self/fd/3", cmd.Path)
	}
	f := cmd.ExtraFiles[0]
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("read exec handle: %v", err)
	}
	if string(got) != "good-binary" {
		t.Fatalf("the swapped snapshot path must not be what runs; the exec handle has %q", got)
	}
	reg.StopAll()
}

// TestRegistry_LegacyPluginTamperedSnapshotRefused guards the snapshot reuse
// path: a same-user attacker who replaces the verified snapshot executable is
// refused on the next load — never silently executed.
func TestRegistry_LegacyPluginTamperedSnapshotRefused(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "legacy")
	if err := os.WriteFile(binPath, []byte("good-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, repo := testutil.NewTestDB(t, false)
	seedAnchoredPaymentPlugin(t, repo, binPath, "demo_pay")

	// First load materializes the verified snapshot.
	factory1 := &legacyClientFactory{}
	reg1 := NewRegistry(repo)
	reg1.SetPluginDir(dir)
	reg1.newClient = factory1.new
	if _, err := reg1.ListProviders(context.Background(), true); err != nil {
		t.Fatalf("first list: %v", err)
	}
	reg1.StopAll()

	// Same-user attacker unlinks and replaces the snapshot executable.
	snapDir := filepath.Join(dir, paymentSnapshotRootName, "payment", "demo_pay")
	entries, err := os.ReadDir(snapDir)
	if err != nil {
		t.Fatalf("read snapshot dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one snapshot, got %d", len(entries))
	}
	snapFile := filepath.Join(snapDir, entries[0].Name())
	if err := os.Remove(snapFile); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapFile, []byte("attacker-payload"), 0o500); err != nil {
		t.Fatal(err)
	}

	// A fresh registry (no cache) re-verifies the snapshot and must refuse the
	// entry without spawning it; other providers stay usable.
	factory2 := &legacyClientFactory{}
	reg2 := NewRegistry(repo)
	reg2.SetPluginDir(dir)
	reg2.newClient = factory2.new
	providers, err := reg2.ListProviders(context.Background(), true)
	if err != nil {
		t.Fatalf("list must tolerate a refused entry (fault isolation): %v", err)
	}
	if containsKey(providerKeys(providers), "demo_pay") {
		t.Fatalf("tampered snapshot must be refused, got providers: %v", providerKeys(providers))
	}
	if n := len(factory2.all()); n != 0 {
		t.Fatalf("tampered snapshot was spawned: %d clients", n)
	}
}

// TestRegistry_StopAllKillsLegacyClients is the lifecycle regression: StopAll
// must terminate every legacy net-RPC client and be idempotent, so shutdown and
// settings-clearing never orphan child processes.
func TestRegistry_StopAllKillsLegacyClients(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "legacy")
	if err := os.WriteFile(binPath, []byte("good-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, repo := testutil.NewTestDB(t, false)
	seedAnchoredPaymentPlugin(t, repo, binPath, "demo_pay")

	factory := &legacyClientFactory{}
	reg := NewRegistry(repo)
	reg.SetPluginDir(dir)
	reg.newClient = factory.new
	if _, err := reg.ListProviders(context.Background(), true); err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if got := factory.killedCount(); got != 0 {
		t.Fatalf("clients must not be killed before StopAll, got %d", got)
	}
	reg.StopAll()
	total := len(factory.all())
	if got := factory.killedCount(); got != total {
		t.Fatalf("StopAll must kill every legacy client: %d/%d", got, total)
	}
	// Idempotent: a second StopAll (e.g. double shutdown signal) is a no-op.
	reg.StopAll()
	if got := factory.killedCount(); got != total {
		t.Fatalf("StopAll must be idempotent: %d/%d", got, total)
	}
}

// TestRegistry_StopAllClearsStateOnSettingsClear guards the settings-clearing
// path: when payment_plugins is emptied, the next load must not keep serving the
// previously spawned legacy clients (which would be orphaned).
func TestRegistry_StopAllClearsStateOnSettingsClear(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "legacy")
	if err := os.WriteFile(binPath, []byte("good-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, repo := testutil.NewTestDB(t, false)
	seedAnchoredPaymentPlugin(t, repo, binPath, "demo_pay")

	factory := &legacyClientFactory{}
	reg := NewRegistry(repo)
	reg.SetPluginDir(dir)
	reg.newClient = factory.new
	if _, err := reg.ListProviders(context.Background(), true); err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if n := len(factory.all()); n != 1 {
		t.Fatalf("expected one legacy client, got %d", n)
	}

	// Clear the payment_plugins setting; the next load must tear down the old
	// client instead of reusing it.
	if err := repo.UpsertSetting(context.Background(), domain.Setting{
		Key:       settingPaymentPlugins,
		ValueJSON: `[]`,
	}); err != nil {
		t.Fatalf("clear payment_plugins: %v", err)
	}
	providers, err := reg.ListProviders(context.Background(), true)
	if err != nil {
		t.Fatalf("list providers after clear: %v", err)
	}
	if containsKey(providerKeys(providers), "demo_pay") {
		t.Fatalf("legacy provider must disappear after settings cleared, got %v", providerKeys(providers))
	}
	if got := factory.killedCount(); got != 1 {
		t.Fatalf("clearing settings must kill the previously spawned client, killed=%d/1", got)
	}
}

// TestRegistry_GetProviderConfig_SettingsDisabledStaysDisabled is the persistent
// -config-authority regression: a provider explicitly disabled in
// payment_providers_enabled must stay disabled even while a gRPC provider for
// the same key is currently loaded and running.
func TestRegistry_GetProviderConfig_SettingsDisabledStaysDisabled(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	if err := repo.UpsertSetting(context.Background(), domain.Setting{
		Key:       settingPaymentEnabled,
		ValueJSON: `{"ezpay.alipay":false}`,
	}); err != nil {
		t.Fatalf("seed payment_providers_enabled: %v", err)
	}

	reg := NewRegistry(repo)
	reg.SetPluginPaymentMethodRepo(&fakePluginPaymentMethodRepo{})
	reg.SetPluginManager(&fakePluginManager{items: []plugins.ListItem{ezpayListItem(t, []string{"alipay"})}, clientOK: true})

	cfg, enabled, err := reg.GetProviderConfig(context.Background(), "ezpay.alipay")
	if err != nil {
		t.Fatalf("GetProviderConfig: %v", err)
	}
	if enabled {
		t.Fatalf("settings-disabled provider must stay disabled while a gRPC provider is running (cfg=%q)", cfg)
	}
}

// TestRegistry_GetProviderConfig_GRPCMethodFromPersistentTable confirms gRPC
// plugin methods with no settings entry report enablement from the persisted
// install + method table, not from whether a process is loaded.
func TestRegistry_GetProviderConfig_GRPCMethodFromPersistentTable(t *testing.T) {
	reg := testGRPCRegistry(t, []string{"alipay", "wechat"}, nil)
	cfg, enabled, err := reg.GetProviderConfig(context.Background(), "ezpay.alipay")
	if err != nil {
		t.Fatalf("GetProviderConfig: %v", err)
	}
	if !enabled {
		t.Fatalf("gRPC method with no settings entry should be enabled from the persistent method table, cfg=%q", cfg)
	}
}

// TestRegistry_GetProviderConfig_GRPCMethodDisabledInTable confirms a method
// disabled in the persistent per-method table reports disabled.
func TestRegistry_GetProviderConfig_GRPCMethodDisabledInTable(t *testing.T) {
	reg := testGRPCRegistry(t, []string{"alipay", "wechat"}, []string{"alipay"})
	_, enabled, err := reg.GetProviderConfig(context.Background(), "ezpay.alipay")
	if err != nil {
		t.Fatalf("GetProviderConfig: %v", err)
	}
	if enabled {
		t.Fatal("method disabled in the persistent table must report disabled")
	}
}

// TestRegistry_GetProviderConfig_BuiltinNoSettingsUsesDefault confirms builtins
// keep their default enabled state when no settings entry exists.
func TestRegistry_GetProviderConfig_BuiltinNoSettingsUsesDefault(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	reg := NewRegistry(repo)
	_, enabled, err := reg.GetProviderConfig(context.Background(), "approval")
	if err != nil {
		t.Fatalf("GetProviderConfig: %v", err)
	}
	if !enabled {
		t.Fatal("builtin approval must default to enabled")
	}
}

// TestRegistry_GetProvider_SettingsDisabledGRPCMethodRefused is the persistent
// -config-authority regression for the serving path: a gRPC plugin method
// explicitly disabled in payment_providers_enabled must be refused by GetProvider
// even while a provider for the same key is loaded and running. Sibling methods
// without a settings entry stay reachable (their authority is the persisted
// install + per-method table).
func TestRegistry_GetProvider_SettingsDisabledGRPCMethodRefused(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	if err := repo.UpsertSetting(context.Background(), domain.Setting{
		Key:       settingPaymentEnabled,
		ValueJSON: `{"ezpay.alipay":false}`,
	}); err != nil {
		t.Fatalf("seed payment_providers_enabled: %v", err)
	}
	reg := NewRegistry(repo)
	reg.SetPluginPaymentMethodRepo(&fakePluginPaymentMethodRepo{})
	reg.SetPluginManager(&fakePluginManager{items: []plugins.ListItem{ezpayListItem(t, []string{"alipay", "wechat"})}, clientOK: true})

	if _, err := reg.GetProvider(context.Background(), "ezpay.alipay"); err == nil {
		t.Fatal("settings-disabled gRPC method must be refused by GetProvider even while its provider is loaded")
	}
	if _, err := reg.GetProvider(context.Background(), "ezpay.wechat"); err != nil {
		t.Fatalf("sibling gRPC method without a settings entry must remain reachable: %v", err)
	}
}

// TestRegistry_LegacyPluginClientConfig_NetRPCOnly ensures the legacy path still
// restricts the go-plugin protocol to net/rpc (never gRPC) even with AutoMTLS.
func TestRegistry_LegacyPluginClientConfig_NetRPCOnly(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "legacy")
	if err := os.WriteFile(binPath, []byte("good-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, repo := testutil.NewTestDB(t, false)
	seedAnchoredPaymentPlugin(t, repo, binPath, "demo_pay")

	factory := &legacyClientFactory{}
	reg := NewRegistry(repo)
	reg.SetPluginDir(dir)
	reg.newClient = factory.new
	if _, err := reg.ListProviders(context.Background(), true); err != nil {
		t.Fatalf("list providers: %v", err)
	}
	clients := factory.all()
	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}
	if clients[0].cfg.AllowedProtocols == nil || len(clients[0].cfg.AllowedProtocols) != 1 || clients[0].cfg.AllowedProtocols[0] != plugin.ProtocolNetRPC {
		t.Fatalf("legacy plugin must restrict to net/rpc, got %v", clients[0].cfg.AllowedProtocols)
	}
	reg.StopAll()
}

// compile-time check that the fake client satisfies pluginClientHandle.
var _ pluginClientHandle = (*fakeLegacyClient)(nil)

// P2-51 contract: the legacy net-rpc plugin client must always negotiate
// AutoMTLS — there is no silent downgrade to plaintext for plugins that do not
// support it (they fail to connect and are refused).
func TestRegistry_LegacyPluginClientConfig_AutoMTLSAlwaysOn(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "legacy")
	if err := os.WriteFile(binPath, []byte("good-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, repo := testutil.NewTestDB(t, false)
	seedAnchoredPaymentPlugin(t, repo, binPath, "demo_pay")

	factory := &legacyClientFactory{}
	reg := NewRegistry(repo)
	reg.SetPluginDir(dir)
	reg.newClient = factory.new
	if _, err := reg.ListProviders(context.Background(), true); err != nil {
		t.Fatalf("list providers: %v", err)
	}
	clients := factory.all()
	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}
	if !clients[0].cfg.AutoMTLS {
		t.Fatalf("legacy net-rpc plugin client must require AutoMTLS (no plaintext downgrade)")
	}
	if clients[0].cfg.UnixSocketConfig == nil {
		t.Fatalf("legacy client must pin control sockets to a dedicated 0700 dir")
	}
	reg.StopAll()
}

// P2-51 contract: on Linux the legacy executable is spawned from the verified
// open inode handle (/proc/self/fd/3), never from a re-parsed path — closing the
// hash-to-exec window. The fail-closed guard refuses to spawn when no handle is
// available (non-Linux platforms).
func TestExecFromVerifiedHandleBindsOpenInode(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("fd-bound exec is Linux-only")
	}
	f, err := os.CreateTemp(t.TempDir(), "verified-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write([]byte("good-bytes")); err != nil {
		t.Fatal(err)
	}
	cmd := execFromVerifiedHandle("/unused/snap/path", f)
	if cmd.Path != "/proc/self/fd/3" {
		t.Fatalf("expected exec of /proc/self/fd/3, got %q", cmd.Path)
	}
	if len(cmd.ExtraFiles) != 1 {
		t.Fatalf("expected the verified handle in ExtraFiles, got %d", len(cmd.ExtraFiles))
	}
	if cmd.ExtraFiles[0] != f {
		t.Fatalf("ExtraFiles[0] is not the verified handle")
	}
}
