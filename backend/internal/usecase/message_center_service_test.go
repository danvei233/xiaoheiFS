package usecase_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestMessageCenterService_NotifyAndRead(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "m1", "m1@example.com", "pass")
	svc := usecase.NewMessageCenterService(repo, repo)

	if err := svc.NotifyUser(context.Background(), user.ID, "info", "title", "body"); err != nil {
		t.Fatalf("notify: %v", err)
	}
	count, err := svc.UnreadCount(context.Background(), user.ID)
	if err != nil || count == 0 {
		t.Fatalf("expected unread")
	}
	items, _, err := svc.List(context.Background(), user.ID, "", 10, 0)
	if err != nil || len(items) == 0 {
		t.Fatalf("expected items")
	}
	if err := svc.MarkRead(context.Background(), user.ID, items[0].ID); err != nil {
		t.Fatalf("mark read: %v", err)
	}
}
