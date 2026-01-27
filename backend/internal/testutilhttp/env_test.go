package testutilhttp

import (
	"net/http"
	"testing"

	"xiaoheiplay/internal/testutil"
)

func TestNewTestEnv(t *testing.T) {
	env := NewTestEnv(t, false)
	if env == nil || env.Router == nil || env.Repo == nil {
		t.Fatalf("env not initialized")
	}
	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/site/settings", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("site settings status: %d", rec.Code)
	}
}
