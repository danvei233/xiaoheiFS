package http

import (
	"math"
	"testing"

	"xiaoheiplay/internal/pkg/money"
)

// TestParseAmountCents_RejectsInt64Overflow locks the F6 fix: the int/int64
// branches must not multiply x100 on raw int64 (which wraps to a bogus positive
// value) — oversized values must be rejected exactly like the string path.
func TestParseAmountCents_RejectsInt64Overflow(t *testing.T) {
	cases := []struct {
		name  string
		value any
	}{
		{"max-int64-hundredths", int64(math.MaxInt64)},
		{"max-int64/100+1", int64(math.MaxInt64/100 + 1)},
		{"int", int(math.MaxInt64)},
		{"float64-huge", float64(math.MaxInt64)},
		{"float64-inf", math.Inf(1)},
		{"float64-nan", math.NaN()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseAmountCents(tc.value); err != money.ErrInvalidAmount {
				t.Fatalf("parseAmountCents(%v) = err %v, want ErrInvalidAmount", tc.value, err)
			}
		})
	}
}

func TestParseAmountCents_AcceptableValues(t *testing.T) {
	cases := []struct {
		value any
		want  int64
	}{
		{int64(1234), 123400},
		{int(1234), 123400},
		{float64(12.34), 1234},
		{float32(12.34), 1234},
		{"12.345", 1235}, // string path rounds half up
	}
	for _, tc := range cases {
		got, err := parseAmountCents(tc.value)
		if err != nil {
			t.Fatalf("parseAmountCents(%v) err = %v, want nil", tc.value, err)
		}
		if got != tc.want {
			t.Fatalf("parseAmountCents(%v) = %d, want %d", tc.value, got, tc.want)
		}
	}
}
