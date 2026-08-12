package plugins

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/pkg/pluginsign"
)

// buildGRPCPaymentPlugin builds the real gRPC payment plugin demo (payment_ezpay)
// so tests exercise the actual go-plugin AutoMTLS negotiation over the gRPC
// transport, not just the host-side config flags. The module root is four
// directories above the package dir.
func buildGRPCPaymentPlugin(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..", "..", "..")
	bin := filepath.Join(t.TempDir(), "payment_ezpay")
	cmd := exec.Command("go", "build", "-o", bin, "./plugin-demo/pluginv1/payment_ezpay")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build plugin-demo/pluginv1/payment_ezpay: %v\n%s", err, out)
	}
	return bin
}

// TestRuntimeStart_GRPCPluginAutoMTLS_E2E proves the P2-51 gRPC transport
// invariants against a real plugin process: the host spawns the plugin from the
// verified snapshot with AutoMTLS enabled, the plugin connects over the
// mutually-authenticated gRPC channel, and StopAll tears the child down.
func TestRuntimeStart_GRPCPluginAutoMTLS_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end plugin build in short mode")
	}
	bin := buildGRPCPaymentPlugin(t)

	base := t.TempDir()
	pluginDir := filepath.Join(base, "payment", "ezpay")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{
  "plugin_id": "ezpay",
  "name": "EZPay",
  "version": "1.0.0",
  "capabilities": { "payment": { "methods": ["alipay","wxpay","qqpay"] } }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	binData, err := os.ReadFile(bin)
	if err != nil {
		t.Fatalf("read plugin binary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), binData, 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := pluginsign.ContentDigest(pluginDir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "payment",
		PluginID:        "ezpay",
		InstanceID:      DefaultInstanceID,
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   digest,
	}}
	m := NewManager(base, repo, newTestCipher(t), nil)
	// Teardown must run even when an assertion below fails before the explicit
	// StopAll: go-plugin registers no automatic child reaper for non-Managed
	// clients, so an un-killed child is orphaned to PID 1 when the test exits.
	t.Cleanup(m.StopAll)
	// The gRPC child is spawned from the verified snapshot
	// <baseDir>/.snapshots/payment/ezpay/<digest>/, so its executable path is a
	// per-test marker we can assert against at the process level.
	marker := filepath.Join(base, snapshotRootName, "payment", "ezpay")

	manifest, err := m.EnsureRunning(context.Background(), "payment", "ezpay", DefaultInstanceID)
	if err != nil {
		t.Fatalf("EnsureRunning over mTLS gRPC channel: %v", err)
	}
	if manifest.GetPluginId() != "ezpay" {
		t.Fatalf("unexpected plugin id %q", manifest.GetPluginId())
	}
	rp, ok := m.runtime.GetRunning("payment", "ezpay", DefaultInstanceID)
	if !ok || rp == nil || rp.client == nil {
		t.Fatal("plugin not running after mTLS start")
	}
	if rp.client.Exited() {
		t.Fatal("plugin process exited immediately after mTLS start")
	}
	// Positive control: exactly one live plugin child (this test's unique temp
	// base, so no cross-test noise).
	if runtime.GOOS == "linux" {
		if pids := testutil.LiveProcessesByExe(marker); len(pids) != 1 {
			t.Fatalf("expected exactly one live gRPC plugin child while running, got %v", pids)
		}
	}
	m.StopAll()
	if !rp.client.Exited() {
		t.Fatal("StopAll must terminate the plugin process")
	}
	if runtime.GOOS == "linux" {
		testutil.WaitNoProcessByExe(t, marker, 5*time.Second)
	}
}
