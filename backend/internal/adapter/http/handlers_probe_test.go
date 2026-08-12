package http

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseProbeIDFromBearer_RejectsWrongTokenType(t *testing.T) {
	h := &Handler{jwtSecret: []byte("test-secret")}
	// Sign with type=user (not probe_access) — must be rejected.
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"probe_id": 5,
		"role":     "probe",
		"type":     "user",
		"iss":      jwtIssuer,
		"aud":      jwtAudience,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	raw, err := tok.SignedString(h.jwtSecret)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := h.parseProbeIDFromBearer("Bearer " + raw); err == nil {
		t.Fatalf("expected token with wrong type claim to be rejected")
	}
}

func TestParseProbeIDFromBearer_RejectsWrongRole(t *testing.T) {
	h := &Handler{jwtSecret: []byte("test-secret")}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"probe_id": 5,
		"role":     "admin",
		"type":     "probe_access",
		"iss":      jwtIssuer,
		"aud":      jwtAudience,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	raw, _ := tok.SignedString(h.jwtSecret)
	if _, err := h.parseProbeIDFromBearer("Bearer " + raw); err == nil {
		t.Fatalf("expected token with wrong role claim to be rejected")
	}
}

func TestParseProbeIDFromBearer_AcceptsValidToken(t *testing.T) {
	h := &Handler{jwtSecret: []byte("test-secret")}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"probe_id": 5,
		"role":     "probe",
		"type":     "probe_access",
		"iss":      jwtIssuer,
		"aud":      jwtAudience,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	raw, _ := tok.SignedString(h.jwtSecret)
	id, err := h.parseProbeIDFromBearer("Bearer " + raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 5 {
		t.Fatalf("expected probe id 5, got %d", id)
	}
}

func TestParseTimeOrNow_ClampsFutureTimestamp(t *testing.T) {
	future := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
	got := parseTimeOrNow(future)
	if got.After(time.Now()) {
		t.Fatalf("future timestamp not clamped: %s", got.Format(time.RFC3339))
	}
}

func TestParseTimeOrNow_ClampsExtremePastTimestamp(t *testing.T) {
	past := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
	got := parseTimeOrNow(past)
	if got.Before(time.Now().Add(-24 * time.Hour)) {
		t.Fatalf("extreme past timestamp not clamped: %s", got.Format(time.RFC3339))
	}
}

func TestParseTimeOrNow_PreservesRecentTimestamp(t *testing.T) {
	recent := time.Now().Add(-5 * time.Minute)
	got := parseTimeOrNow(recent.Format(time.RFC3339))
	if got.Before(time.Now().Add(-10 * time.Minute)) {
		t.Fatalf("recent timestamp was dropped: %s", got.Format(time.RFC3339))
	}
}

func TestClampInt(t *testing.T) {
	cases := []struct {
		v, min, max, want int
	}{
		{5, 1, 10, 5},
		{0, 1, 10, 1},
		{100, 1, 10, 10},
		{3, 3, 3, 3},
	}
	for _, tc := range cases {
		if got := clampInt(tc.v, tc.min, tc.max); got != tc.want {
			t.Fatalf("clampInt(%d,%d,%d) = %d, want %d", tc.v, tc.min, tc.max, got, tc.want)
		}
	}
}

func TestProbeRandomTokenUsesAlphabet(t *testing.T) {
	tok := probeRandomToken(32)
	if len(tok) != 32 {
		t.Fatalf("unexpected length %d", len(tok))
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, r := range tok {
		if !strings.ContainsRune(alphabet, r) {
			t.Fatalf("token char outside alphabet: %q", r)
		}
	}
}
