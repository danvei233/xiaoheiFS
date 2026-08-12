package main

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	apppluginadmin "xiaoheiplay/internal/app/pluginadmin"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestResolvePluginUploadDir(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := apppluginadmin.NewService(nil, nil, repo)
	if got := svc.ResolveUploadDir(context.Background(), ""); got != "plugins/payment" {
		t.Fatalf("unexpected default upload dir: %s", got)
	}
	if err := repo.UpsertSetting(context.Background(), domain.Setting{Key: "payment_plugin_dir", ValueJSON: "plugins/custom"}); err != nil {
		t.Fatalf("upsert setting: %v", err)
	}
	if got := svc.ResolveUploadDir(context.Background(), ""); got != "plugins/custom" {
		t.Fatalf("unexpected setting upload dir: %s", got)
	}
}

func TestApplyGinModeDefaultsToRelease(t *testing.T) {
	t.Setenv("GIN_MODE", "")
	applyGinMode()
	// gin.Mode() is a string; "release" is the expected default.
	if gin.Mode() != gin.ReleaseMode {
		t.Fatalf("expected release mode by default, got %q", gin.Mode())
	}
}

func TestApplyGinModeHonorsDebugOverride(t *testing.T) {
	t.Setenv("GIN_MODE", "debug")
	applyGinMode()
	if gin.Mode() != gin.DebugMode {
		t.Fatalf("expected debug mode honored, got %q", gin.Mode())
	}
}
