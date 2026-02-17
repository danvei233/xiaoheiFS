package repo_test

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSQLiteRepo_ListUsersByRoleStatus(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	admin := domain.User{
		Username:     "admin1",
		Email:        "admin1@example.com",
		PasswordHash: "hash",
		Role:         domain.UserRoleAdmin,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	user := domain.User{
		Username:     "user1",
		Email:        "user1@example.com",
		PasswordHash: "hash",
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusDisabled,
	}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	items, total, err := repo.ListUsersByRoleStatus(ctx, string(domain.UserRoleUser), "", 10, 0)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("list users by role: %v %d %d", err, total, len(items))
	}
	items, total, err = repo.ListUsersByRoleStatus(ctx, string(domain.UserRoleUser), string(domain.UserStatusDisabled), 10, 0)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("list users by status: %v %d %d", err, total, len(items))
	}
	items, total, err = repo.ListUsersByRoleStatus(ctx, string(domain.UserRoleAdmin), string(domain.UserStatusActive), 10, 0)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("list admins: %v %d %d", err, total, len(items))
	}
}

func TestSQLiteRepo_ListOrdersPaymentsFilters(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := testutil.CreateUser(t, repo, "buyer", "buyer@example.com", "pass")
	order1 := domain.Order{UserID: user.ID, OrderNo: "ORD-F1", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	order2 := domain.Order{UserID: user.ID, OrderNo: "ORD-F2", Status: domain.OrderStatusApproved, TotalAmount: 2000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order1); err != nil {
		t.Fatalf("create order1: %v", err)
	}
	if err := repo.CreateOrder(ctx, &order2); err != nil {
		t.Fatalf("create order2: %v", err)
	}
	if err := repo.UpdateOrderStatus(ctx, order2.ID, domain.OrderStatusApproved); err != nil {
		t.Fatalf("update order2: %v", err)
	}

	items, total, err := repo.ListOrders(ctx, shared.OrderFilter{UserID: user.ID}, 10, 0)
	if err != nil || total != 2 || len(items) != 2 {
		t.Fatalf("list orders: %v %d %d", err, total, len(items))
	}
	items, total, err = repo.ListOrders(ctx, shared.OrderFilter{Status: string(domain.OrderStatusApproved), UserID: user.ID}, 10, 0)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("list orders by status: %v %d %d", err, total, len(items))
	}
	items, _, err = repo.ListOrders(ctx, shared.OrderFilter{UserID: user.ID}, 1, 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("list orders paging: %v %d", err, len(items))
	}

	payment1 := domain.OrderPayment{OrderID: order1.ID, UserID: user.ID, Method: "fake", Amount: 1000, Currency: "CNY", TradeNo: "TN-F1", Status: domain.PaymentStatusPendingPayment}
	payment2 := domain.OrderPayment{OrderID: order2.ID, UserID: user.ID, Method: "fake", Amount: 2000, Currency: "CNY", TradeNo: "TN-F2", Status: domain.PaymentStatusApproved}
	if err := repo.CreatePayment(ctx, &payment1); err != nil {
		t.Fatalf("create payment1: %v", err)
	}
	if err := repo.CreatePayment(ctx, &payment2); err != nil {
		t.Fatalf("create payment2: %v", err)
	}

	pays, total, err := repo.ListPayments(ctx, shared.PaymentFilter{Status: string(domain.PaymentStatusApproved)}, 10, 0)
	if err != nil || total != 1 || len(pays) != 1 {
		t.Fatalf("list payments by status: %v %d %d", err, total, len(pays))
	}
}

func TestSQLiteRepo_NotificationsAndTickets(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := testutil.CreateUser(t, repo, "notify", "notify@example.com", "pass")
	unread := domain.Notification{UserID: user.ID, Type: "info", Title: "t1", Content: "c1"}
	read := domain.Notification{UserID: user.ID, Type: "info", Title: "t2", Content: "c2", ReadAt: timePtr(time.Now())}
	if err := repo.CreateNotification(ctx, &unread); err != nil {
		t.Fatalf("create unread: %v", err)
	}
	if err := repo.CreateNotification(ctx, &read); err != nil {
		t.Fatalf("create read: %v", err)
	}

	filter := shared.NotificationFilter{UserID: &user.ID, Status: "unread", Limit: 10}
	items, total, err := repo.ListNotifications(ctx, filter)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("list unread: %v %d %d", err, total, len(items))
	}
	filter.Status = "read"
	items, total, err = repo.ListNotifications(ctx, filter)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("list read: %v %d %d", err, total, len(items))
	}
	if count, err := repo.CountUnread(ctx, user.ID); err != nil || count != 1 {
		t.Fatalf("count unread: %v %d", err, count)
	}
	if err := repo.MarkAllRead(ctx, user.ID); err != nil {
		t.Fatalf("mark all read: %v", err)
	}
	if count, err := repo.CountUnread(ctx, user.ID); err != nil || count != 0 {
		t.Fatalf("count unread after: %v %d", err, count)
	}

	ticket := domain.Ticket{UserID: user.ID, Subject: "Help", Status: "open"}
	msg := domain.TicketMessage{SenderID: user.ID, SenderRole: "user", Content: "hello"}
	if err := repo.CreateTicketWithDetails(ctx, &ticket, &msg, nil); err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	list, total, err := repo.ListTickets(ctx, shared.TicketFilter{UserID: &user.ID, Status: "open", Limit: 10})
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("list tickets: %v %d %d", err, total, len(list))
	}
	if err := repo.DeleteTicket(ctx, ticket.ID); err != nil {
		t.Fatalf("delete ticket: %v", err)
	}
}

func TestSQLiteRepo_RenewOrderAndInstances(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := testutil.CreateUser(t, repo, "renew", "renew@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-R1", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:  order.ID,
		Amount:   1000,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "renew",
		SpecJSON: `{"vps_id":123}`,
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	if ok, err := repo.HasPendingRenewOrder(ctx, user.ID, 123); err != nil || !ok {
		t.Fatalf("pending renew: %v %v", err, ok)
	}
	if ok, err := repo.HasPendingRenewOrder(ctx, user.ID, 999); err != nil || ok {
		t.Fatalf("pending renew other: %v %v", err, ok)
	}
	order2 := domain.Order{UserID: user.ID, OrderNo: "ORD-R2", Status: domain.OrderStatusProvisioning, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order2); err != nil {
		t.Fatalf("create order2: %v", err)
	}
	item2 := domain.OrderItem{
		OrderID:  order2.ID,
		Amount:   1000,
		Status:   domain.OrderItemStatusProvisioning,
		Action:   "resize",
		SpecJSON: `{"vps_id":123}`,
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{item2}); err != nil {
		t.Fatalf("create item2: %v", err)
	}
	if ok, err := repo.HasPendingRenewOrder(ctx, user.ID, 123); err != nil || !ok {
		t.Fatalf("pending renew conflict with resize: %v %v", err, ok)
	}

	itemsList, err := repo.ListOrderItems(ctx, order.ID)
	if err != nil || len(itemsList) == 0 {
		t.Fatalf("list order items: %v", err)
	}
	inst := domain.VPSInstance{
		UserID:      user.ID,
		OrderItemID: itemsList[0].ID,
		Name:        "inst1",
		Status:      domain.VPSStatusRunning,
		SpecJSON:    "{}",
	}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	items, total, err := repo.ListInstances(ctx, 1, 0)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("list instances: %v %d %d", err, total, len(items))
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
