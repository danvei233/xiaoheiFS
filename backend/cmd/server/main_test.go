package main

import (
	"context"
	"testing"

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
