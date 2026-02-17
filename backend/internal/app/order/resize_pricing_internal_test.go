package order

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
)

func TestResizePricingHelpers(t *testing.T) {
	inst := domain.VPSInstance{
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}
	if ratio := remainingRatio(inst, time.Now()); ratio != 0 {
		t.Fatalf("expected ratio 0 without period")
	}
	expire := time.Now().Add(24 * time.Hour)
	inst.SpecJSON = mustJSON(map[string]any{
		"current_period_start": time.Now().Add(-12 * time.Hour).UTC().Format(time.RFC3339),
		"current_period_end":   expire.UTC().Format(time.RFC3339),
	})
	inst.ExpireAt = &expire
	if ratio := remainingRatio(inst, time.Now()); ratio <= 0 || ratio >= 1 {
		t.Fatalf("expected ratio between 0 and 1")
	}
	if got := applyRounding(123, "floor"); got != 123 {
		t.Fatalf("floor rounding: %v", got)
	}
	if got := applyRounding(124, "ceil"); got != 124 {
		t.Fatalf("ceil rounding: %v", got)
	}
	spec := parseCartSpecJSON(`{"add_cores":1}`)
	if spec.AddCores != 1 {
		t.Fatalf("expected spec parsed")
	}
	plan := domain.PlanGroup{UnitCore: 100, UnitMem: 200, UnitDisk: 300, UnitBW: 400}
	price := addonPrice(plan, CartSpec{AddCores: 1, AddMemGB: 1, AddDiskGB: 1, AddBWMbps: 1})
	if price != 1000 {
		t.Fatalf("unexpected addon price: %v", price)
	}
}

func TestResizePricingPolicySettings(t *testing.T) {
	repo := &fakeSettingRepo{values: map[string]string{
		"resize_price_mode":       "remaining",
		"resize_refund_ratio":     "0.5",
		"resize_rounding":         "ceil",
		"resize_min_charge":       "1",
		"resize_min_refund":       "2",
		"resize_refund_to_wallet": "false",
	}}
	svc := &OrderService{settings: repo}
	policy := svc.resizePricingPolicy(context.Background())
	if policy.RefundRatio != 0.5 || policy.MinCharge != 100 || policy.MinRefund != 200 || policy.RefundToWallet {
		t.Fatalf("unexpected policy: %+v", policy)
	}
}
