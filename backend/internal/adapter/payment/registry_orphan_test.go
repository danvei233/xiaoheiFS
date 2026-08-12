package payment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/hashicorp/go-plugin"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// Process-level orphan regression guards.
//
// The legacy net-RPC payment plugin children are go-plugin processes that the
// host must kill explicitly (Kill() is the only reaper; go-plugin registers no
// automatic child reaper for non-Managed clients). A test or server path that
// exits without calling StopAll therefore orphans the child to PID 1. These
// tests scan /proc by executable path — the verified snapshot inode that the
// child is spawned from resolves to `<pluginDir>/.snapshots/payment/...` even
// when the exec was fd-bound (/proc/self/fd/N) — and assert no live plugin
// subprocess survives each lifecycle window.

// assertNoPluginSubprocesses fails the test if any live process still has an
// executable under the given snapshot marker. The /proc scan is Linux-only;
// on other platforms the check is a no-op.
func assertNoPluginSubprocesses(t *testing.T, marker string) {
	t.Helper()
	if runtime.GOOS != "linux" {
		return
	}
	testutil.WaitNoProcessByExe(t, marker, 5*time.Second)
}

// assertOnePluginSubprocess is the positive control: exactly one live child
// must be running under the marker. Skipped off-Linux where /proc is absent.
func assertOnePluginSubprocess(t *testing.T, marker string) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("process-level /proc scan is Linux-only")
	}
	if pids := testutil.LiveProcessesByExe(marker); len(pids) != 1 {
		t.Fatalf("expected exactly one live plugin child while loaded, got %v", pids)
	}
}

// TestRegistry_LegacyPluginEndToEnd_NoOrphanAfterStopAll is the process-level
// companion to TestRegistry_LegacyPluginEndToEnd_AutoMTLS: it proves the real
// plugin child is running while loaded (positive control for the /proc scan),
// that StopAll tears it down, and that a second StopAll leaves nothing behind.
func TestRegistry_LegacyPluginEndToEnd_NoOrphanAfterStopAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end plugin build in short mode")
	}
	bin := buildLegacyPaymentDemo(t)
	_, repo := testutil.NewTestDB(t, false)
	seedAnchoredPaymentPlugin(t, repo, bin, "demo_pay")

	reg := NewRegistry(repo)
	reg.SetPluginDir(filepath.Dir(bin))
	marker := filepath.Join(filepath.Dir(bin), paymentSnapshotRootName, "payment")

	if _, err := reg.ListProviders(context.Background(), true); err != nil {
		t.Fatalf("list providers: %v", err)
	}
	// Positive control: the scan must see exactly the child we spawned. The
	// marker embeds this test's unique temp dir, so a leftover from another
	// test cannot satisfy it.
	assertOnePluginSubprocess(t, marker)

	reg.StopAll()
	assertNoPluginSubprocesses(t, marker)

	reg.StopAll() // idempotent
	assertNoPluginSubprocesses(t, marker)
}

// TestRegistry_LegacyPluginEndToEnd_TeardownViaCleanup_NoOrphan proves the
// failure window cannot orphan the child: t.Cleanup — the exact mechanism that
// runs when an assertion fails before the explicit StopAll line — tears the
// plugin down. The body deliberately never calls StopAll; only the cleanup
// registered at the start does.
func TestRegistry_LegacyPluginEndToEnd_TeardownViaCleanup_NoOrphan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end plugin build in short mode")
	}
	bin := buildLegacyPaymentDemo(t)
	_, repo := testutil.NewTestDB(t, false)
	seedAnchoredPaymentPlugin(t, repo, bin, "demo_pay")

	reg := NewRegistry(repo)
	reg.SetPluginDir(filepath.Dir(bin))
	marker := filepath.Join(filepath.Dir(bin), paymentSnapshotRootName, "payment")

	// Assert after StopAll runs (cleanups are LIFO: StopAll is registered last).
	t.Cleanup(func() { assertNoPluginSubprocesses(t, marker) })
	t.Cleanup(reg.StopAll)

	if _, err := reg.ListProviders(context.Background(), true); err != nil {
		t.Fatalf("list providers: %v", err)
	}
	assertOnePluginSubprocess(t, marker)
	// No explicit StopAll in the body: the child must be torn down by the
	// cleanup that also runs on an assertion failure.
}

// TestRegistry_LegacyPluginEndToEnd_NoOrphanOnSettingsClear is the process-level
// guard for the settings-clearing window: emptying payment_plugins and loading
// again must kill the previously spawned child, not keep it alive.
func TestRegistry_LegacyPluginEndToEnd_NoOrphanOnSettingsClear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end plugin build in short mode")
	}
	bin := buildLegacyPaymentDemo(t)
	_, repo := testutil.NewTestDB(t, false)
	seedAnchoredPaymentPlugin(t, repo, bin, "demo_pay")

	reg := NewRegistry(repo)
	reg.SetPluginDir(filepath.Dir(bin))
	marker := filepath.Join(filepath.Dir(bin), paymentSnapshotRootName, "payment")
	t.Cleanup(reg.StopAll)

	if _, err := reg.ListProviders(context.Background(), true); err != nil {
		t.Fatalf("list providers: %v", err)
	}
	assertOnePluginSubprocess(t, marker)

	// Clear payment_plugins; the next load must reconcile and kill the child.
	if err := repo.UpsertSetting(context.Background(), domain.Setting{
		Key:       settingPaymentPlugins,
		ValueJSON: `[]`,
	}); err != nil {
		t.Fatalf("clear payment_plugins: %v", err)
	}
	if _, err := reg.ListProviders(context.Background(), true); err != nil {
		t.Fatalf("list providers after clear: %v", err)
	}
	assertNoPluginSubprocesses(t, marker)
}

// failingLegacyClient is a pluginClientHandle whose Client() always fails,
// modeling the Client() failure window: the spawn attempt dies and the host
// must kill the client and retain nothing.
type failingLegacyClient struct {
	cfg    *plugin.ClientConfig
	killed bool
}

func (c *failingLegacyClient) Client() (plugin.ClientProtocol, error) {
	return nil, errors.New("simulated client failure")
}

func (c *failingLegacyClient) Kill() { c.killed = true }

type failingClientFactory struct {
	clients []*failingLegacyClient
}

func (f *failingClientFactory) new(cfg *plugin.ClientConfig) pluginClientHandle {
	c := &failingLegacyClient{cfg: cfg}
	f.clients = append(f.clients, c)
	return c
}

// TestRegistry_LegacyClientFailureKillsAndRetainsNothing covers the Client()
// failure window: the failed client is killed, no provider is served from it,
// and StopAll afterwards has nothing to do (no retained client to leak).
func TestRegistry_LegacyClientFailureKillsAndRetainsNothing(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "legacy")
	if err := os.WriteFile(binPath, []byte("good-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, repo := testutil.NewTestDB(t, false)
	seedAnchoredPaymentPlugin(t, repo, binPath, "demo_pay")

	factory := &failingClientFactory{}
	reg := NewRegistry(repo)
	reg.SetPluginDir(dir)
	reg.newClient = factory.new

	providers, err := reg.ListProviders(context.Background(), true)
	if err != nil {
		t.Fatalf("list providers must tolerate a client failure (fault isolation): %v", err)
	}
	if containsKey(providerKeys(providers), "demo_pay") {
		t.Fatalf("client that failed to start must not be served, got %v", providerKeys(providers))
	}
	if len(factory.clients) != 1 {
		t.Fatalf("expected one spawned client, got %d", len(factory.clients))
	}
	if !factory.clients[0].killed {
		t.Fatal("a client whose Client() failed must be killed")
	}
	// Nothing retained: StopAll (and Close) must be no-ops and not panic.
	reg.StopAll()
	reg.Close()
}
