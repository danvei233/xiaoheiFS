package money

import (
	"math"
	"testing"
)

func TestParseAmountToCentsBasic(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"0", 0},
		{"0.10", 10},
		{"1.00", 100},
		{"1.234", 123},
		{"1.235", 124},   // half-up
		{"1.999", 200},   // rounds to the next yuan
		{"-1.999", -200}, // negative half-up away from zero
		{"+1.50", 150},
		{" 1.5 ", 150},
		{".5", 50},
		{"92233720368547758.00", 9223372036854775800}, // max representable value
		{"92233720368547758.07", 9223372036854775807}, // exactly MaxInt64
	}
	for _, c := range cases {
		got, err := ParseAmountToCents(c.in)
		if err != nil {
			t.Errorf("ParseAmountToCents(%q): unexpected error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseAmountToCents(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestParseAmountToCentsOverflow(t *testing.T) {
	// Each of these overflows int64 cents (some silently wrap today).
	overflows := []string{
		"92233720368547758.08", // rounds to MaxInt64+1
		"92233720368547758.99",
		"99999999999999999.99",
		"200000000000000000.00",
		"184467440737095516.00",
		"-92233720368547758.99",
	}
	for _, in := range overflows {
		if _, err := ParseAmountToCents(in); err == nil {
			t.Errorf("ParseAmountToCents(%q): expected overflow error, got nil", in)
		}
	}
}

func TestParseAmountToCentsInvalid(t *testing.T) {
	invalid := []string{
		"",
		"-",
		"+",
		"abc",
		"1.2.3",
		"1..2",
		"1a",
		"1.2x",
	}
	for _, in := range invalid {
		if _, err := ParseAmountToCents(in); err == nil {
			t.Errorf("ParseAmountToCents(%q): expected error, got nil", in)
		}
	}
}

func TestFormatCents(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0.00"},
		{5, "0.05"},
		{150, "1.50"},
		{-150, "-1.50"},
		{1000, "10.00"},
		{math.MinInt64, "-92233720368547758.08"},
		{math.MaxInt64, "92233720368547758.07"},
	}
	for _, c := range cases {
		if got := FormatCents(c.in); got != c.want {
			t.Errorf("FormatCents(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestProrateCentsOverflowClamps(t *testing.T) {
	// remain > total forces the quotient beyond int64; it must clamp, not wrap.
	if got := ProrateCents(math.MaxInt64, 2, 1); got != math.MaxInt64 {
		t.Errorf("ProrateCents(MaxInt64,2,1) = %d, want MaxInt64", got)
	}
	if got := ProrateCents(math.MinInt64, 2, 1); got != math.MinInt64 {
		t.Errorf("ProrateCents(MinInt64,2,1) = %d, want MinInt64", got)
	}
}
