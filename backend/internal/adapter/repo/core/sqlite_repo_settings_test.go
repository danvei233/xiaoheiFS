package repo_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"xiaoheiplay/internal/adapter/repo/core"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
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

// TestSQLiteRepo_ClaimSettingIfAbsentIsAtomicAcrossConnections validates the
// cross-host install serialization point (plugin_install F4): the DB-side
// completion marker is claimed with a unique-key insert, so two processes (two
// independent connections/instances) sharing one database cannot both win the
// claim even when their install.lock files do not conflict (each host has its
// own lock). Exactly one claim must win; later claims must lose.
func TestSQLiteRepo_ClaimSettingIfAbsentIsAtomicAcrossConnections(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	connA, err := db.Open(config.Config{DBType: "sqlite", DBPath: dbPath})
	if err != nil {
		t.Fatalf("open A: %v", err)
	}
	t.Cleanup(func() { _ = connA.SQL.Close() })
	if err := repo.Migrate(connA.Gorm); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// A second independent connection to the same file models the second host.
	connB, err := db.Open(config.Config{DBType: "sqlite", DBPath: dbPath})
	if err != nil {
		t.Fatalf("open B: %v", err)
	}
	t.Cleanup(func() { _ = connB.SQL.Close() })

	repoA := repo.NewGormRepo(connA.Gorm)
	repoB := repo.NewGormRepo(connB.Gorm)

	type claimResult struct {
		ok  bool
		err error
	}
	start := make(chan struct{})
	results := make([]claimResult, 2)
	var wg sync.WaitGroup
	for i, r := range []*repo.GormRepo{repoA, repoB} {
		wg.Add(1)
		go func(idx int, r *repo.GormRepo) {
			defer wg.Done()
			<-start
			ok, err := r.ClaimSettingIfAbsent(context.Background(), "install_completed_at", "claimed")
			results[idx] = claimResult{ok: ok, err: err}
		}(i, r)
	}
	close(start)
	wg.Wait()

	winners := 0
	for _, res := range results {
		if res.err != nil {
			t.Fatalf("claim error: %v", res.err)
		}
		if res.ok {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("expected exactly one claim to win, got %d", winners)
	}

	// A subsequent claim on either connection must also lose.
	if ok, err := repoA.ClaimSettingIfAbsent(context.Background(), "install_completed_at", "claimed"); err != nil || ok {
		t.Fatalf("expected a later claim to lose (ok=%v err=%v)", ok, err)
	}
}

func TestSQLiteRepo_RejectsInvalidNormalizedSettingJSON(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	err := repo.UpsertSetting(ctx, domain.Setting{
		Key:       "auth_register_required_fields",
		ValueJSON: "not-json-array",
	})
	if err == nil {
		t.Fatalf("expected error for invalid normalized setting json")
	}
}

// TestSQLiteRepo_UpsertSettingsAtomic verifies the batch settings write is
// all-or-nothing: when the k-th key fails validation, the earlier keys are not
// left half-written.
func TestSQLiteRepo_UpsertSettingsAtomic(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	if err := repo.UpsertSetting(ctx, domain.Setting{Key: "site_name", ValueJSON: "old"}); err != nil {
		t.Fatalf("seed site_name: %v", err)
	}

	// The second item targets a normalized list setting with invalid JSON, which
	// must fail inside the shared transaction and roll back the first item.
	err := repo.UpsertSettings(ctx, []domain.Setting{
		{Key: "site_name", ValueJSON: "new"},
		{Key: "auth_register_required_fields", ValueJSON: "not-a-json-array"},
	})
	if err == nil {
		t.Fatalf("expected batch upsert to fail on the invalid list setting")
	}
	got, gerr := repo.GetSetting(ctx, "site_name")
	if gerr != nil {
		t.Fatalf("get site_name: %v", gerr)
	}
	if got.ValueJSON != "old" {
		t.Fatalf("site_name = %q, want %q (batch must roll back earlier keys)", got.ValueJSON, "old")
	}
}

// TestSQLiteRepo_UpsertSettingsAllPersist verifies a valid batch persists every
// key.
func TestSQLiteRepo_UpsertSettingsAllPersist(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	items := []domain.Setting{
		{Key: "site_name", ValueJSON: "A"},
		{Key: "site_title", ValueJSON: "B"},
	}
	if err := repo.UpsertSettings(ctx, items); err != nil {
		t.Fatalf("batch upsert: %v", err)
	}
	for _, item := range items {
		got, err := repo.GetSetting(ctx, item.Key)
		if err != nil {
			t.Fatalf("get %s: %v", item.Key, err)
		}
		if got.ValueJSON != item.ValueJSON {
			t.Fatalf("%s = %q, want %q", item.Key, got.ValueJSON, item.ValueJSON)
		}
	}
}
