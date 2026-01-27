package main

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestEnsureAdminUser(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ensureAdminUser(repo, "root", "pass1234")

	user, err := repo.GetUserByUsernameOrEmail(context.Background(), "root")
	if err != nil {
		t.Fatalf("get admin: %v", err)
	}
	if user.Role != domain.UserRoleAdmin {
		t.Fatalf("expected admin role")
	}
}

func TestGetSettingValue(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	if err := repo.UpsertSetting(context.Background(), domain.Setting{Key: "site_name", ValueJSON: "Example"}); err != nil {
		t.Fatalf("upsert setting: %v", err)
	}
	if got := getSettingValue(repo, "site_name"); got != "Example" {
		t.Fatalf("unexpected setting value: %s", got)
	}
	if got := getSettingValue(repo, "missing"); got != "" {
		t.Fatalf("expected empty value")
	}
}
