package testutil

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestHelpers(t *testing.T) {
	_, repo := NewTestDB(t, false)

	user := CreateUser(t, repo, "user1", "user1@example.com", "pass")
	if user.ID == 0 {
		t.Fatalf("expected user id")
	}
	admin := CreateAdmin(t, repo, "admin1", "admin1@example.com", "pass", 1)
	if admin.ID == 0 {
		t.Fatalf("expected admin id")
	}
	token := IssueJWT(t, "secret", user.ID, "user", time.Minute)
	if token == "" {
		t.Fatalf("expected token")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	rec := DoJSON(t, mux, http.MethodPost, "/ping", map[string]any{"a": "b"}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("expected ok response")
	}

	if Itoa(42) != "42" {
		t.Fatalf("itoa mismatch")
	}
}

func TestSeedCatalog(t *testing.T) {
	_, repo := NewTestDB(t, false)
	seed := SeedCatalog(t, repo)
	if seed.Region.ID == 0 || seed.PlanGroup.ID == 0 || seed.Package.ID == 0 || seed.SystemImage.ID == 0 {
		t.Fatalf("seed catalog ids missing")
	}
}
