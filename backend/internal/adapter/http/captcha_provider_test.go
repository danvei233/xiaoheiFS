package http

import (
	"fmt"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
)

// TestValidateGeeTestGenTime covers every freshness-gate branch (P3-20):
// missing, non-numeric, zero, negative, future and expired values must all be
// rejected, and boundary values that are exactly at the limits must be
// accepted.
func TestValidateGeeTestGenTime(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	cases := []struct {
		name    string
		genTime string
		wantErr bool
	}{
		{"missing", "", true},
		{"whitespace", "   ", true},
		{"non-numeric", "abc", true},
		{"zero", "0", true},
		{"negative", "-1", true},
		{"fractional", "1700000000.5", true},

		{"future within skew", fmt.Sprintf("%d", now.Add(2*time.Minute).Unix()), false},
		{"future just past skew", fmt.Sprintf("%d", now.Add(2*time.Minute+time.Second).Unix()), true},
		{"expired at limit", fmt.Sprintf("%d", now.Add(-10*time.Minute).Unix()), false},
		{"expired just past limit", fmt.Sprintf("%d", now.Add(-10*time.Minute-time.Second).Unix()), true},
		{"now", fmt.Sprintf("%d", now.Unix()), false},
		{"older but valid", fmt.Sprintf("%d", now.Add(-5*time.Minute).Unix()), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateGeeTestGenTime(tc.genTime, now)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %q", tc.genTime)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected success for %q, got %v", tc.genTime, err)
			}
			if tc.wantErr && err != domain.ErrCaptchaFailed {
				t.Fatalf("expected ErrCaptchaFailed, got %v", err)
			}
		})
	}
}
