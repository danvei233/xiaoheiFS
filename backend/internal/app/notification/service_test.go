package notification_test

import (
	"context"
	"testing"
	"time"
	appmessage "xiaoheiplay/internal/app/message"
	appnotification "xiaoheiplay/internal/app/notification"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestNotificationService_SendExpireReminders(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	email := &testutil.FakeEmailSender{}
	msg := appmessage.NewService(repo, repo)
	svc := appnotification.NewService(repo, repo, repo, email, msg)

	user := testutil.CreateUser(t, repo, "n1", "n1@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-NOTIFY-1", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}"}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)
	exp := time.Now().Add(24 * time.Hour)
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "1",
		Name:                 "vm",
		SystemID:             1,
		Status:               domain.VPSStatusUnknown,
		SpecJSON:             "{}",
		ExpireAt:             &exp,
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create vps: %v", err)
	}

	if err := svc.SendExpireReminders(context.Background()); err != nil {
		t.Fatalf("send reminders: %v", err)
	}
	if len(email.Sends) == 0 {
		t.Fatalf("expected email sent")
	}
}
