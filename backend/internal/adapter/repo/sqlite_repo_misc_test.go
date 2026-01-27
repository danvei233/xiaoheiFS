package repo_test

import (
	"context"
	"time"

	"testing"

	"xiaoheiplay/internal/domain"
)

func TestSQLiteRepo_APIsAndLogs(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	key := &domain.APIKey{Name: "key1", KeyHash: "hash1", Status: domain.APIKeyStatusActive, ScopesJSON: `["*"]`}
	if err := r.CreateAPIKey(ctx, key); err != nil {
		t.Fatalf("create api key: %v", err)
	}
	_, err := r.GetAPIKeyByHash(ctx, "hash1")
	if err != nil {
		t.Fatalf("get api key by hash: %v", err)
	}

	cycle := &domain.BillingCycle{Name: "monthly", Months: 1, Multiplier: 1, MinQty: 1, MaxQty: 12, Active: true, SortOrder: 1}
	if err := r.CreateBillingCycle(ctx, cycle); err != nil {
		t.Fatalf("create billing cycle: %v", err)
	}
	if _, err := r.GetBillingCycle(ctx, cycle.ID); err != nil {
		t.Fatalf("get billing cycle: %v", err)
	}
	cycle.Multiplier = 1.2
	if err := r.UpdateBillingCycle(ctx, *cycle); err != nil {
		t.Fatalf("update billing cycle: %v", err)
	}
	cycles, err := r.ListBillingCycles(ctx)
	if err != nil || len(cycles) == 0 {
		t.Fatalf("list billing cycles: %v", err)
	}

	uploadUser := domain.User{Username: "uploader", Email: "uploader@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := r.CreateUser(ctx, &uploadUser); err != nil {
		t.Fatalf("create upload user: %v", err)
	}
	upload := &domain.Upload{Name: "a.txt", Path: "/tmp/a.txt", URL: "/uploads/a.txt", Mime: "text/plain", Size: 1, UploaderID: uploadUser.ID}
	if err := r.CreateUpload(ctx, upload); err != nil {
		t.Fatalf("create upload: %v", err)
	}
	uploads, _, err := r.ListUploads(ctx, 10, 0)
	if err != nil || len(uploads) == 0 {
		t.Fatalf("list uploads: %v", err)
	}

	logEntry := domain.AdminAuditLog{AdminID: 1, Action: "create", TargetType: "order", TargetID: "1", DetailJSON: "{}"}
	if err := r.AddAuditLog(ctx, logEntry); err != nil {
		t.Fatalf("add audit log: %v", err)
	}
	logs, _, err := r.ListAuditLogs(ctx, 10, 0)
	if err != nil || len(logs) == 0 {
		t.Fatalf("list audit logs: %v", err)
	}

	syncLog := &domain.IntegrationSyncLog{Target: "automation", Mode: "manual", Status: "ok", Message: "done"}
	if err := r.CreateSyncLog(ctx, syncLog); err != nil {
		t.Fatalf("create sync log: %v", err)
	}
	syncLogs, _, err := r.ListSyncLogs(ctx, "", 10, 0)
	if err != nil || len(syncLogs) == 0 {
		t.Fatalf("list sync logs: %v", err)
	}
}

func TestSQLiteRepo_PaymentsTicketsAndInstances(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	user := domain.User{Username: "u1", Email: "u1@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := r.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	order := domain.Order{UserID: user.ID, OrderNo: "O-1", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "USD"}
	if err := r.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	payment := &domain.OrderPayment{
		OrderID:   order.ID,
		UserID:    user.ID,
		Method:    "custom",
		TradeNo:   "T-1",
		Amount:    1000,
		Currency:  "USD",
		Status:    domain.PaymentStatusPendingPayment,
		CreatedAt: time.Now(),
	}
	if err := r.CreatePayment(ctx, payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}
	reviewer := int64(1)
	if err := r.UpdatePaymentStatus(ctx, payment.ID, domain.PaymentStatusApproved, &reviewer, "ok"); err != nil {
		t.Fatalf("update payment status: %v", err)
	}
	payments, err := r.ListPaymentsByOrder(ctx, order.ID)
	if err != nil || len(payments) == 0 {
		t.Fatalf("list payments: %v", err)
	}

	items := []domain.OrderItem{
		{OrderID: order.ID, SpecJSON: "{}", Qty: 1, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, AutomationInstanceID: "", Action: "create", DurationMonths: 1},
	}
	if err := r.CreateOrderItems(ctx, items); err != nil {
		t.Fatalf("create order items: %v", err)
	}
	orderItemID := items[0].ID

	ticket := &domain.Ticket{
		UserID:  user.ID,
		Subject: "help",
		Status:  "open",
	}
	msg := &domain.TicketMessage{SenderID: user.ID, SenderRole: string(domain.UserRoleUser), SenderName: "u1", Content: "hello"}
	if err := r.CreateTicketWithDetails(ctx, ticket, msg, nil); err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, err := r.GetTicket(ctx, ticket.ID); err != nil {
		t.Fatalf("get ticket: %v", err)
	}

	expireAt := time.Now().Add(24 * time.Hour)
	inst := &domain.VPSInstance{
		UserID:      user.ID,
		OrderItemID: orderItemID,
		Name:        "vm-1",
		Status:      domain.VPSStatusRunning,
		ExpireAt:    &expireAt,
	}
	if err := r.CreateInstance(ctx, inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if err := r.UpdateInstanceExpireAt(ctx, inst.ID, time.Now().Add(48*time.Hour)); err != nil {
		t.Fatalf("update instance expire: %v", err)
	}
	inst.PanelURLCache = "http://panel"
	if err := r.UpdateInstanceLocal(ctx, *inst); err != nil {
		t.Fatalf("update instance local: %v", err)
	}
}
