package email

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSender_Disabled(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	_ = repo.UpsertSetting(context.Background(), domain.Setting{Key: "smtp_enabled", ValueJSON: "false"})
	sender := NewSender(repo)
	if err := sender.Send(context.Background(), "a@example.com", "subj", "body"); err == nil {
		t.Fatalf("expected smtp disabled error")
	}
}

func TestIsHTMLContent(t *testing.T) {
	if !isHTMLContent("<html><body>hi</body></html>") {
		t.Fatalf("expected html content")
	}
	if isHTMLContent("plain text") {
		t.Fatalf("expected non-html")
	}
}
