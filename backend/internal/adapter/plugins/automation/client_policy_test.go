package automation

import (
	"testing"
	"time"
)

// P2-43 contract: the client must not use a single fixed 12s timeout.
func TestActionPolicyCategories(t *testing.T) {
	cases := []struct {
		action     string
		wantCat    actionCategory
		wantRetry  int
	}{
		{"automation.GetInstance", catRead, 2},
		{"automation.ListSnapshots", catRead, 2},
		{"automation.GetMonitor", catRead, 2},
		{"automation.CreateBackup", catLongOp, 0},
		{"automation.CreateSnapshot", catLongOp, 0},
		{"automation.Rebuild", catLongOp, 0},
		{"automation.Destroy", catDestructive, 0},
		{"automation.DeleteSnapshot", catDestructive, 0},
		{"automation.ResetPassword", catDestructive, 0},
		{"automation.UnknownAction", catDefault, 0},
	}
	for _, tc := range cases {
		p := basePolicyFor(tc.action)
		if p.category != tc.wantCat {
			t.Errorf("%s: category = %d, want %d", tc.action, p.category, tc.wantCat)
		}
		if p.maxRetries != tc.wantRetry {
			t.Errorf("%s: maxRetries = %d, want %d", tc.action, p.maxRetries, tc.wantRetry)
		}
	}
}

// Long operations get a strictly larger timeout than the default read timeout.
func TestLongOpTimeoutExceedsDefault(t *testing.T) {
	read := basePolicyFor("automation.GetInstance")
	longOp := basePolicyFor("automation.CreateBackup")
	if longOp.timeout <= read.timeout {
		t.Fatalf("long-op timeout %s must exceed read timeout %s", longOp.timeout, read.timeout)
	}
}

// Destructive RPCs and long operations must never auto-retry.
func TestDestructiveAndLongOpNeverRetry(t *testing.T) {
	for _, action := range []string{"automation.Destroy", "automation.DeleteBackup", "automation.Reboot", "automation.CreateBackup"} {
		p := basePolicyFor(action)
		if p.maxRetries != 0 {
			t.Errorf("%s must not auto-retry, got maxRetries=%d", action, p.maxRetries)
		}
	}
}

// The fixed 12s default must be gone: the read timeout is the policy default,
// and long ops use a larger value (no literal 12s anywhere in the policy).
func TestNoFixedTwelveSecondDefault(t *testing.T) {
	if defaultRPCTimeout == 12*time.Second {
		t.Fatalf("default RPC timeout must not be the hardcoded 12s")
	}
	if longOpRPCTimeout <= 12*time.Second {
		t.Fatalf("long-op RPC timeout must be larger than the old 12s")
	}
}
