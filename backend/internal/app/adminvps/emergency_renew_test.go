package adminvps

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// TestEmergencyRenewInWindow_NonPositiveWindowNeverAllowed locks P3-27: a
// window_days of 0 or less must never open the emergency renew window for every
// instance — it must deny instead.
func TestEmergencyRenewInWindow_NonPositiveWindowNeverAllowed(t *testing.T) {
	now := time.Now()
	expire := now.Add(48 * time.Hour)

	for _, window := range []int{0, -1, -7} {
		if emergencyRenewInWindow(now, &expire, window) {
			t.Fatalf("window_days=%d must never open the emergency renew window", window)
		}
	}

	// A positive window inside the window must still allow.
	if !emergencyRenewInWindow(now, &expire, 7) {
		t.Fatalf("window_days=7 with expiry 48h out must be inside the window")
	}
	// A positive window outside the window must deny.
	if emergencyRenewInWindow(now, &expire, 1) {
		t.Fatalf("window_days=1 with expiry 48h out must be outside the window")
	}
	// A nil expiry is never inside the window.
	if emergencyRenewInWindow(now, nil, 7) {
		t.Fatalf("nil expiry must never be inside the window")
	}
}

// TestLoadEmergencyRenewPolicy_ClampsNonPositiveWindow locks the P3-27 clamp:
// the policy loader must never surface a window of 0/negative (which would
// translate into "always allowed"); it clamps to at least 1.
func TestLoadEmergencyRenewPolicy_ClampsNonPositiveWindow(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	for _, raw := range []string{"0", "-3"} {
		if err := repo.UpsertSetting(ctx, domain.Setting{Key: "emergency_renew_window_days", ValueJSON: raw}); err != nil {
			t.Fatalf("upsert window_days=%s: %v", raw, err)
		}
		policy := loadEmergencyRenewPolicy(ctx, repo)
		if policy.WindowDays <= 0 {
			t.Fatalf("window_days=%s must be clamped to a positive value, got %d", raw, policy.WindowDays)
		}
		if policy.WindowDays != 1 {
			t.Fatalf("window_days=%s must clamp to 1, got %d", raw, policy.WindowDays)
		}
	}
}
