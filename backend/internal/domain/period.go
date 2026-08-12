package domain

import (
	"encoding/json"
	"strings"
	"time"
)

// CurrentBillingPeriod returns the billing period currently in effect for the
// instance. It prefers the current_period_start/current_period_end recorded in
// SpecJSON by the purchase, renewal and emergency-renew flows, falling back to
// the full CreatedAt..ExpireAt lifetime for legacy instances that predate the
// period bookkeeping. The boolean reports whether a usable period could be
// resolved.
//
// Both the order refund path and the walletorder refund path anchor their
// elapsed ratio to this period; without it, renewals (which extend ExpireAt
// without touching CreatedAt) would measure elapsed time from the original
// purchase and systematically under-refund.
func (v VPSInstance) CurrentBillingPeriod(now time.Time) (time.Time, time.Time, bool) {
	start, end, ok := currentBillingPeriodFromSpec(v.SpecJSON, v.ExpireAt)
	if ok {
		return start, end, true
	}
	if v.ExpireAt != nil && !v.CreatedAt.IsZero() && v.ExpireAt.After(now) {
		return v.CreatedAt, *v.ExpireAt, true
	}
	return time.Time{}, time.Time{}, false
}

// currentBillingPeriodFromSpec parses the period bookkeeping out of a VPS
// instance SpecJSON blob.
func currentBillingPeriodFromSpec(specJSON string, expireAt *time.Time) (time.Time, time.Time, bool) {
	if strings.TrimSpace(specJSON) == "" {
		return time.Time{}, time.Time{}, false
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(specJSON), &payload); err != nil {
		return time.Time{}, time.Time{}, false
	}
	startText := parsePeriodValue(payload, "current_period_start", "period_start", "last_renew_at", "purchase_at")
	endText := parsePeriodValue(payload, "current_period_end", "period_end")
	if endText == "" && expireAt != nil {
		endText = expireAt.UTC().Format(time.RFC3339)
	}
	if startText == "" || endText == "" {
		return time.Time{}, time.Time{}, false
	}
	start, err := time.Parse(time.RFC3339, startText)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	end, err := time.Parse(time.RFC3339, endText)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

// parsePeriodValue returns the first non-empty string value found for the
// given keys in a decoded SpecJSON payload.
func parsePeriodValue(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if raw, ok := payload[key]; ok {
			if val, ok := raw.(string); ok {
				return strings.TrimSpace(val)
			}
		}
	}
	return ""
}
