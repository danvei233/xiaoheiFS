package http_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

// TestRealNameVerifyResponseNeverLeaksPendingToken verifies that a pending KYC
// verification stores its continuation token encrypted in provider_state (never
// in Reason) and the API response contains neither the token nor a
// provider_state field.
func TestRealNameVerifyResponseNeverLeaksPendingToken(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)

	const pendingToken = "tok-super-secret-42"
	env.RealnameReg.Register(&testutil.FakeRealNameProvider{
		KeyVal:  "pending_provider",
		NameVal: "Pending",
		OK:      false,
		Reason:  "pending:" + pendingToken,
	})
	ctx := context.Background()
	if err := env.Repo.UpsertSetting(ctx, domain.Setting{Key: "realname_enabled", ValueJSON: "true"}); err != nil {
		t.Fatalf("enable realname: %v", err)
	}
	if err := env.Repo.UpsertSetting(ctx, domain.Setting{Key: "realname_provider", ValueJSON: "pending_provider"}); err != nil {
		t.Fatalf("set provider: %v", err)
	}

	user := testutil.CreateUser(t, env.Repo, "realnametok", "realnametok@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/realname/verify", map[string]any{
		"real_name": "Alice",
		"id_number": "11010519491231002X",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify: code=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, pendingToken) {
		t.Fatalf("API response leaked pending token: %s", body)
	}
	if strings.Contains(body, "provider_state") {
		t.Fatalf("API response leaked provider_state field: %s", body)
	}
	if strings.Contains(body, "tok-super-secret") {
		t.Fatalf("API response contains token fragments: %s", body)
	}
	// The DB record must hold the token encrypted, not in Reason.
	record, err := env.Repo.GetLatestRealNameVerification(ctx, user.ID)
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if strings.Contains(record.Reason, pendingToken) {
		t.Fatalf("token persisted in Reason: %q", record.Reason)
	}
	if record.ProviderState == "" {
		t.Fatalf("expected encrypted provider state to be persisted")
	}
	if strings.Contains(record.ProviderState, pendingToken) {
		t.Fatalf("token persisted in provider_state plaintext")
	}
}
