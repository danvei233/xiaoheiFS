package payment

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/testutil"
)

// buildLegacyPaymentDemo builds the real net-RPC payment plugin demo binary so
// tests exercise the actual go-plugin AutoMTLS negotiation, not just the client
// config flags. The module root is three directories above the package dir.
func buildLegacyPaymentDemo(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..", "..")
	bin := filepath.Join(t.TempDir(), "payment_demo")
	cmd := exec.Command("go", "build", "-o", bin, "./plugin-demo/payment_demo")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build plugin-demo/payment_demo: %v\n%s", err, out)
	}
	return bin
}

// TestRegistry_LegacyPluginEndToEnd_AutoMTLS proves the P2-51 transport
// invariants against a real plugin process: the legacy net-RPC client is
// spawned from the verified snapshot with AutoMTLS enabled, the plugin connects
// over the mutually-authenticated channel, and StopAll tears the child down.
func TestRegistry_LegacyPluginEndToEnd_AutoMTLS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end plugin build in short mode")
	}
	bin := buildLegacyPaymentDemo(t)
	_, repo := testutil.NewTestDB(t, false)
	seedAnchoredPaymentPlugin(t, repo, bin, "demo_pay")

	reg := NewRegistry(repo)
	reg.SetPluginDir(filepath.Dir(bin))
	// Teardown must run even when an assertion below fails before the explicit
	// StopAll: go-plugin registers no automatic child reaper for non-Managed
	// clients, so an un-killed child is orphaned to PID 1 when the test exits.
	t.Cleanup(reg.StopAll)

	providers, err := reg.ListProviders(context.Background(), true)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if !containsKey(providerKeys(providers), "demo_pay") {
		t.Fatalf("expected demo_pay provider over the mTLS channel, got %v", providerKeys(providers))
	}

	// Exercise the RPC over the AutoMTLS channel: a real call proves the
	// mutually-authenticated transport actually negotiated.
	var p appshared.PaymentProvider
	for _, pr := range providers {
		if pr.Key() == "demo_pay" {
			p = pr
			break
		}
	}
	if p == nil {
		t.Fatal("demo_pay provider not returned by ListProviders")
	}
	res, err := p.CreatePayment(context.Background(), appshared.PaymentCreateRequest{
		OrderID: 42,
		UserID:  7,
	})
	if err != nil {
		t.Fatalf("create payment over mTLS channel: %v", err)
	}
	if !strings.Contains(res.PayURL, "trade_no=demo-42-7") {
		t.Fatalf("unexpected pay url %q from real plugin", res.PayURL)
	}

	// StopAll kills the child process and blocks until it exits; a double StopAll
	// is a no-op (the registry no longer references the clients).
	reg.StopAll()
	reg.StopAll()
}
