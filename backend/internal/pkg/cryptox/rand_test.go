package cryptox

import (
	"strings"
	"testing"
)

// TestRandomAlphabet_Uniform verifies F5 regression: the alphabet draw must be
// uniform (no modulo bias). A chi-square goodness-of-fit over a large sample
// must stay below the critical value for the alphabet size. The old
// byte%len(charset) implementation fails this test (62-char max deviation
// ~+22%); the crypto/rand.Int implementation passes it.
func TestRandomAlphabet_Uniform(t *testing.T) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const samples = 200_000
	const n = 1 // single char per draw; we aggregate per alphabet index

	counts := make([]int, len([]rune(alphabet)))
	for i := 0; i < samples; i++ {
		s, err := RandomAlphabet(alphabet, n)
		if err != nil {
			t.Fatalf("RandomAlphabet: %v", err)
		}
		idx := strings.IndexRune(alphabet, []rune(s)[0])
		if idx < 0 {
			t.Fatalf("unexpected rune in output: %q", s)
		}
		counts[idx]++
	}

	expected := float64(samples) / float64(len(counts))
	var chi2 float64
	for _, c := range counts {
		d := float64(c) - expected
		chi2 += (d * d) / expected
	}
	// Critical value for alpha=0.001 with k-1=61 degrees of freedom is ~99.3;
	// the biased implementation typically exceeds it by a wide margin.
	threshold := 120.0
	if chi2 > threshold {
		t.Fatalf("character distribution is not uniform: chi2=%.2f (threshold %.0f), counts=%v", chi2, threshold, counts)
	}
}

func TestRandomAlphabet_Errors(t *testing.T) {
	if _, err := RandomAlphabet("", 8); err == nil {
		t.Fatal("empty alphabet must error")
	}
	if _, err := RandomAlphabet("abc", -1); err == nil {
		t.Fatal("negative length must error")
	}
	if s, err := RandomAlphabet("abc", 0); err != nil || s != "" {
		t.Fatalf("zero length must return empty, got %q err=%v", s, err)
	}
	if s, err := RandomAlphabet("ab", 10); err != nil || len([]rune(s)) != 10 {
		t.Fatalf("length 10 draw: got %q err=%v", s, err)
	}
}
