package repo_test

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSQLiteRepo_SettingsAndEmailTemplates(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	if err := repo.UpsertSetting(ctx, domain.Setting{Key: "site_name", ValueJSON: "Test"}); err != nil {
		t.Fatalf("upsert setting: %v", err)
	}
	if _, err := repo.GetSetting(ctx, "site_name"); err != nil {
		t.Fatalf("get setting: %v", err)
	}
	if list, err := repo.ListSettings(ctx); err != nil || len(list) == 0 {
		t.Fatalf("list settings: %v", err)
	}

	tmpl := domain.EmailTemplate{Name: "welcome", Subject: "Hi", Body: "Body", Enabled: true}
	if err := repo.UpsertEmailTemplate(ctx, &tmpl); err != nil {
		t.Fatalf("upsert template: %v", err)
	}
	if _, err := repo.GetEmailTemplate(ctx, tmpl.ID); err != nil {
		t.Fatalf("get template: %v", err)
	}
	if list, err := repo.ListEmailTemplates(ctx); err != nil || len(list) == 0 {
		t.Fatalf("list templates: %v", err)
	}
	if err := repo.DeleteEmailTemplate(ctx, tmpl.ID); err != nil {
		t.Fatalf("delete template: %v", err)
	}
}

func TestSQLiteRepo_APIKeysAndPasswordReset(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	key := domain.APIKey{Name: "k1", KeyHash: "hash-key", Status: domain.APIKeyStatusActive, ScopesJSON: `["*"]`}
	if err := repo.CreateAPIKey(ctx, &key); err != nil {
		t.Fatalf("create api key: %v", err)
	}
	if list, _, err := repo.ListAPIKeys(ctx, 10, 0); err != nil || len(list) == 0 {
		t.Fatalf("list api keys: %v", err)
	}
	if err := repo.UpdateAPIKeyStatus(ctx, key.ID, domain.APIKeyStatusDisabled); err != nil {
		t.Fatalf("update api key: %v", err)
	}
	if err := repo.TouchAPIKey(ctx, key.ID); err != nil {
		t.Fatalf("touch api key: %v", err)
	}

	user := testutil.CreateUser(t, repo, "reset", "reset@example.com", "pass")
	token := domain.PasswordResetToken{UserID: user.ID, Token: "token-1", ExpiresAt: time.Now().Add(time.Hour)}
	if err := repo.CreatePasswordResetToken(ctx, &token); err != nil {
		t.Fatalf("create token: %v", err)
	}
	if _, err := repo.GetPasswordResetToken(ctx, token.Token); err != nil {
		t.Fatalf("get token: %v", err)
	}
	if err := repo.MarkPasswordResetTokenUsed(ctx, token.ID); err != nil {
		t.Fatalf("mark used: %v", err)
	}
	expired := domain.PasswordResetToken{UserID: user.ID, Token: "token-expired", ExpiresAt: time.Now().Add(-time.Hour)}
	if err := repo.CreatePasswordResetToken(ctx, &expired); err != nil {
		t.Fatalf("create expired: %v", err)
	}
	if err := repo.DeleteExpiredTokens(ctx); err != nil {
		t.Fatalf("delete expired: %v", err)
	}
}
