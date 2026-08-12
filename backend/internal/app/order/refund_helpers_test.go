package order

import (
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/money"
)

// TestCalculateRefundAmount_UsesCurrentBillingPeriodAfterRenewal verifies that
// the elapsed ratio is measured against the current billing period (written to
// SpecJSON on renewal) rather than the instance creation time. Renewals extend
// ExpireAt without touching CreatedAt, so a lifetime-based ratio skews refunds.
//
// Layout: created 90 days ago, renewed 30 days ago for a 60-day period, so the
// current period is [now-30d, now+30d] and elapsed is ~50%.
// Curve [{0,1},{100,0}] then refunds ~50%, not the lifetime-derived 25%.
func TestCalculateRefundAmount_UsesCurrentBillingPeriodAfterRenewal(t *testing.T) {
	now := time.Now()
	created := now.Add(-90 * 24 * time.Hour)
	start := now.Add(-30 * 24 * time.Hour)
	end := now.Add(30 * 24 * time.Hour)
	inst := domain.VPSInstance{
		CreatedAt: created,
		ExpireAt:  &end,
		SpecJSON:  setCurrentPeriod("{}", start, end),
	}
	policy := RefundPolicy{Curve: []RefundCurvePoint{
		{Percent: 0, Ratio: 1},
		{Percent: 100, Ratio: 0},
	}}
	got := calculateRefundAmountForAmount(inst, 10000, policy)
	if got != 5000 {
		t.Fatalf("expected 5000 (50%% of current billing period), got %d", got)
	}
}

// TestCalculateRefundAmount_FullRefundOnRenewalDay verifies that a refund filed
// on the day of renewal is treated as ~0% elapsed of the fresh billing period,
// even though the instance itself was created months earlier.
func TestCalculateRefundAmount_FullRefundOnRenewalDay(t *testing.T) {
	now := time.Now()
	created := now.Add(-120 * 24 * time.Hour)
	start := now
	end := now.Add(30 * 24 * time.Hour)
	inst := domain.VPSInstance{
		CreatedAt: created,
		ExpireAt:  &end,
		SpecJSON:  setCurrentPeriod("{}", start, end),
	}
	policy := RefundPolicy{Curve: []RefundCurvePoint{
		{Percent: 0, Ratio: 1},
		{Percent: 100, Ratio: 0},
	}}
	got := calculateRefundAmountForAmount(inst, 10000, policy)
	if got != 10000 {
		t.Fatalf("expected full refund 10000 on renewal day, got %d", got)
	}
}

// TestCalculateRefundAmount_ProrateMatchesProrateCents locks the prorate branch
// to integer-exact proration (money.ProrateCents) instead of float64 rounding.
// With a 30-day period, 10 days elapsed and a 30-day prorate window the refund
// is 20/30 of the base amount.
func TestCalculateRefundAmount_ProrateMatchesProrateCents(t *testing.T) {
	now := time.Now()
	start := now.Add(-10 * 24 * time.Hour)
	end := now.Add(20 * 24 * time.Hour)
	inst := domain.VPSInstance{
		CreatedAt: start,
		ExpireAt:  &end,
		SpecJSON:  setCurrentPeriod("{}", start, end),
	}
	policy := RefundPolicy{ProrateDays: 30}
	want := money.ProrateCents(10000, int64(20*24*time.Hour), int64(30*24*time.Hour))
	if want != 6667 {
		t.Fatalf("sanity: expected 6667, got %d", want)
	}
	got := calculateRefundAmountForAmount(inst, 10000, policy)
	if got != want {
		t.Fatalf("expected integer prorate %d, got %d", want, got)
	}
}

// TestCalculateRefundAmount_DayPolicyUsesCurrentPeriod verifies the day/hour
// refund windows (full/prorate/no-refund) are applied against the current
// billing period. Without the period fix, a 95-day-old instance 5 days into a
// 60-day renewal would be treated as ~79% elapsed and get 0 refund; relative to
// the renewed period it is 8% elapsed and falls inside the 7-day prorate window.
func TestCalculateRefundAmount_DayPolicyUsesCurrentPeriod(t *testing.T) {
	now := time.Now()
	created := now.Add(-95 * 24 * time.Hour)
	start := now.Add(-5 * 24 * time.Hour)
	end := now.Add(55 * 24 * time.Hour)
	inst := domain.VPSInstance{
		CreatedAt: created,
		ExpireAt:  &end,
		SpecJSON:  setCurrentPeriod("{}", start, end),
	}
	policy := RefundPolicy{FullDays: 1, ProrateDays: 7, NoRefundDays: 30}
	got := calculateRefundAmountForAmount(inst, 10000, policy)
	want := money.ProrateCents(10000, int64(2*24*time.Hour), int64(7*24*time.Hour))
	if got != want {
		t.Fatalf("expected prorate %d, got %d", want, got)
	}
}

// TestCalculateRefundAmount_LegacyAgeFallbackExpired verifies an already-expired
// instance never refunds, even when it has no usable billing period.
func TestCalculateRefundAmount_LegacyAgeFallbackExpired(t *testing.T) {
	now := time.Now()
	expired := now.Add(-1 * time.Hour)
	inst := domain.VPSInstance{
		CreatedAt: now.Add(-31 * 24 * time.Hour),
		ExpireAt:  &expired,
		SpecJSON:  "{}",
	}
	policy := RefundPolicy{}
	if got := calculateRefundAmountForAmount(inst, 10000, policy); got != 0 {
		t.Fatalf("expected 0 for expired instance, got %d", got)
	}
}
