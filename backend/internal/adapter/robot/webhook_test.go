package robot

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestWebhookNotifier_NoWebhooks(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	notifier := NewWebhookNotifier(repo)
	if err := notifier.NotifyOrderEvent(context.Background(), domain.OrderEvent{OrderID: 1, Seq: 1, Type: "order.pending_review", DataJSON: `{}`, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
