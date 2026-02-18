package repo_test

import (
	"context"
	"time"

	"testing"

	"xiaoheiplay/internal/domain"
)

func TestSQLiteRepo_CartOrdersPayments(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	user := domain.User{Username: "cartuser", Email: "cartuser@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := r.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	item := &domain.CartItem{UserID: user.ID, PackageID: 1, SystemID: 1, SpecJSON: "{}", Qty: 1, Amount: 990}
	if err := r.AddCartItem(ctx, item); err != nil {
		t.Fatalf("add cart item: %v", err)
	}
	item.Qty = 2
	item.Amount = 1980
	if err := r.UpdateCartItem(ctx, *item); err != nil {
		t.Fatalf("update cart item: %v", err)
	}
	if err := r.DeleteCartItem(ctx, item.ID, user.ID); err != nil {
		t.Fatalf("delete cart item: %v", err)
	}
	item2 := &domain.CartItem{UserID: user.ID, PackageID: 2, SystemID: 2, SpecJSON: "{}", Qty: 1, Amount: 500}
	if err := r.AddCartItem(ctx, item2); err != nil {
		t.Fatalf("add cart item2: %v", err)
	}
	if err := r.ClearCart(ctx, user.ID); err != nil {
		t.Fatalf("clear cart: %v", err)
	}

	order := domain.Order{UserID: user.ID, OrderNo: "O-2", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "USD", IdempotencyKey: "idem-1"}
	if err := r.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if _, err := r.GetOrder(ctx, order.ID); err != nil {
		t.Fatalf("get order: %v", err)
	}
	if _, err := r.GetOrderByNo(ctx, order.OrderNo); err != nil {
		t.Fatalf("get order by no: %v", err)
	}
	if _, err := r.GetOrderByIdempotencyKey(ctx, user.ID, "idem-1"); err != nil {
		t.Fatalf("get order by idempotency: %v", err)
	}
	order.Status = domain.OrderStatusApproved
	if err := r.UpdateOrderMeta(ctx, order); err != nil {
		t.Fatalf("update order meta: %v", err)
	}

	items := []domain.OrderItem{
		{OrderID: order.ID, SpecJSON: "{}", Qty: 1, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, AutomationInstanceID: "", Action: "create", DurationMonths: 1},
	}
	if err := r.CreateOrderItems(ctx, items); err != nil {
		t.Fatalf("create order items: %v", err)
	}
	if _, err := r.GetOrderItem(ctx, items[0].ID); err != nil {
		t.Fatalf("get order item: %v", err)
	}
	if err := r.UpdateOrderItemStatus(ctx, items[0].ID, domain.OrderItemStatusApproved); err != nil {
		t.Fatalf("update order item status: %v", err)
	}
	if err := r.UpdateOrderItemAutomation(ctx, items[0].ID, "auto-1"); err != nil {
		t.Fatalf("update order item automation: %v", err)
	}

	payment := &domain.OrderPayment{
		OrderID:        order.ID,
		UserID:         user.ID,
		Method:         "custom",
		Amount:         1000,
		Currency:       "USD",
		TradeNo:        "T-2",
		Status:         domain.PaymentStatusPendingPayment,
		IdempotencyKey: "pay-1",
		CreatedAt:      time.Now(),
	}
	if err := r.CreatePayment(ctx, payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if _, err := r.GetPaymentByTradeNo(ctx, "T-2"); err != nil {
		t.Fatalf("get payment by trade no: %v", err)
	}
	if _, err := r.GetPaymentByIdempotencyKey(ctx, user.ID, "pay-1"); err != nil {
		t.Fatalf("get payment by idempotency: %v", err)
	}
	if err := r.DeleteOrder(ctx, order.ID); err != nil {
		t.Fatalf("delete order: %v", err)
	}
}

func TestSQLiteRepo_TicketsNotificationsRealname(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	user := domain.User{Username: "ticketuser", Email: "ticketuser@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := r.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	ticket := &domain.Ticket{UserID: user.ID, Subject: "help", Status: "open"}
	msg := &domain.TicketMessage{SenderID: user.ID, SenderRole: string(domain.UserRoleUser), SenderName: "u1", Content: "hello"}
	res := []domain.TicketResource{{ResourceType: "order", ResourceID: 1, ResourceName: "O-1"}}
	if err := r.CreateTicketWithDetails(ctx, ticket, msg, res); err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	msg2 := &domain.TicketMessage{TicketID: ticket.ID, SenderID: user.ID, SenderRole: string(domain.UserRoleUser), SenderName: "u1", Content: "follow-up"}
	if err := r.AddTicketMessage(ctx, msg2); err != nil {
		t.Fatalf("add ticket message: %v", err)
	}
	if _, err := r.ListTicketMessages(ctx, ticket.ID); err != nil {
		t.Fatalf("list ticket messages: %v", err)
	}
	if _, err := r.ListTicketResources(ctx, ticket.ID); err != nil {
		t.Fatalf("list ticket resources: %v", err)
	}
	ticket.Status = "closed"
	if err := r.UpdateTicket(ctx, *ticket); err != nil {
		t.Fatalf("update ticket: %v", err)
	}

	note := &domain.Notification{UserID: user.ID, Type: "info", Title: "hello", Content: "world"}
	if err := r.CreateNotification(ctx, note); err != nil {
		t.Fatalf("create notification: %v", err)
	}
	if err := r.MarkNotificationRead(ctx, user.ID, note.ID); err != nil {
		t.Fatalf("mark notification read: %v", err)
	}

	verifiedAt := time.Now()
	record := &domain.RealNameVerification{UserID: user.ID, RealName: "Alice", IDNumber: "ABC123456", Status: "verified", Provider: "fake", Reason: "", VerifiedAt: &verifiedAt}
	if err := r.CreateRealNameVerification(ctx, record); err != nil {
		t.Fatalf("create realname: %v", err)
	}
	if _, err := r.GetLatestRealNameVerification(ctx, user.ID); err != nil {
		t.Fatalf("get latest realname: %v", err)
	}
}
