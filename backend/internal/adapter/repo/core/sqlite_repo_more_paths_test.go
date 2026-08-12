package repo_test

import (
	"context"
	"testing"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
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
	// An in-flight provider session (pending_payment) must block hard deletion:
	// the order's external payment can arrive at any moment and would be stranded
	// with no record/refund. The order is deletable once the session is rejected.
	if err := r.DeleteOrder(ctx, order.ID); err != appshared.ErrConflict {
		t.Fatalf("delete order with in-flight provider session should conflict, got %v", err)
	}
	if err := r.UpdatePaymentStatus(ctx, payment.ID, domain.PaymentStatusRejected, nil, "test cleanup"); err != nil {
		t.Fatalf("reject payment: %v", err)
	}
	if err := r.DeleteOrder(ctx, order.ID); err != nil {
		t.Fatalf("delete order: %v", err)
	}
}

func TestSQLiteRepo_GetPaymentByTradeNoNormalizesEmpty(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	if _, err := r.GetPaymentByTradeNo(ctx, ""); err != appshared.ErrNotFound {
		t.Fatalf("expected ErrNotFound for empty trade no, got %v", err)
	}
	if _, err := r.GetPaymentByTradeNo(ctx, "   "); err != appshared.ErrNotFound {
		t.Fatalf("expected ErrNotFound for blank trade no, got %v", err)
	}
}

func TestSQLiteRepo_KeywordSearchEscapesLikeWildcards(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	user := domain.User{Username: "kwuser", Email: "kwuser@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := r.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	mkTicket := func(subject string) *domain.Ticket {
		ticket := &domain.Ticket{UserID: user.ID, Subject: subject, Status: "open"}
		msg := &domain.TicketMessage{SenderID: user.ID, SenderRole: string(domain.UserRoleUser), SenderName: "u", Content: "hi"}
		if err := r.CreateTicketWithDetails(ctx, ticket, msg, nil); err != nil {
			t.Fatalf("create ticket %q: %v", subject, err)
		}
		return ticket
	}
	mkTicket("discount 100% today")
	mkTicket("discount 100off today")

	// "%" in the keyword must match the literal percent, not the wildcard.
	filter := appshared.TicketFilter{Keyword: "100%", Limit: 10}
	rows, total, err := r.ListTickets(ctx, filter)
	if err != nil {
		t.Fatalf("list tickets by keyword: %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("expected only the literal '100%%' ticket, got total=%d len=%d", total, len(rows))
	}
	if rows[0].Subject != "discount 100% today" {
		t.Fatalf("unexpected matched subject: %q", rows[0].Subject)
	}

	node1 := &domain.ProbeNode{Name: "node_1", AgentID: "agent_1", SecretHash: "h1", Status: domain.ProbeStatusOffline, TagsJSON: "[]"}
	node2 := &domain.ProbeNode{Name: "node1", AgentID: "agent1", SecretHash: "h2", Status: domain.ProbeStatusOffline, TagsJSON: "[]"}
	if err := r.CreateProbeNode(ctx, node1); err != nil {
		t.Fatalf("create probe node 1: %v", err)
	}
	if err := r.CreateProbeNode(ctx, node2); err != nil {
		t.Fatalf("create probe node 2: %v", err)
	}

	// "_" in the keyword must match a literal underscore, not a single char.
	// Unescaped, %_% would match every non-empty name (both nodes); escaped,
	// only node_1/agent_1 actually contains an underscore.
	nodes, total, err := r.ListProbeNodes(ctx, appshared.ProbeNodeFilter{Keyword: "_"}, 10, 0)
	if err != nil {
		t.Fatalf("list probe nodes by keyword: %v", err)
	}
	if total != 1 || len(nodes) != 1 {
		t.Fatalf("expected only the literal '_' probe node, got total=%d len=%d", total, len(nodes))
	}
	if nodes[0].Name != "node_1" {
		t.Fatalf("unexpected matched probe node: %q", nodes[0].Name)
	}
}

func TestSQLiteRepo_AddTicketMessageIsAtomicAndTracksLastReply(t *testing.T) {
	rawDB, r := newTestRepo(t)
	ctx := context.Background()

	// Half-write protection: adding a message to a non-existent ticket must
	// fail and not leave an orphan message row behind.
	orphan := &domain.TicketMessage{TicketID: 999999, SenderID: 1, SenderRole: string(domain.UserRoleUser), SenderName: "u", Content: "ghost"}
	err := r.AddTicketMessage(ctx, orphan)
	if err == nil {
		t.Fatalf("expected AddTicketMessage to fail for missing ticket")
	}
	var msgCount int
	if err := rawDB.QueryRow("SELECT COUNT(1) FROM ticket_messages WHERE ticket_id = ?", 999999).Scan(&msgCount); err != nil {
		t.Fatalf("count orphan messages: %v", err)
	}
	if msgCount != 0 {
		t.Fatalf("expected no orphan message rows, got %d", msgCount)
	}

	// last_reply_* must match the newest message row, not a later wall-clock time.
	user := domain.User{Username: "tmsg", Email: "tmsg@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := r.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	ticket := &domain.Ticket{UserID: user.ID, Subject: "atomic", Status: "open"}
	msg := &domain.TicketMessage{SenderID: user.ID, SenderRole: string(domain.UserRoleUser), SenderName: "u", Content: "first"}
	if err := r.CreateTicketWithDetails(ctx, ticket, msg, nil); err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	msg2 := &domain.TicketMessage{TicketID: ticket.ID, SenderID: user.ID, SenderRole: string(domain.UserRoleUser), SenderName: "u", Content: "second"}
	if err := r.AddTicketMessage(ctx, msg2); err != nil {
		t.Fatalf("add ticket message: %v", err)
	}
	got, err := r.GetTicket(ctx, ticket.ID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if got.LastReplyAt == nil || !got.LastReplyAt.Equal(msg2.CreatedAt) {
		t.Fatalf("expected last_reply_at to equal message created_at %v, got %v", msg2.CreatedAt, got.LastReplyAt)
	}
	if got.LastReplyBy == nil || *got.LastReplyBy != msg2.SenderID {
		t.Fatalf("expected last_reply_by = %d, got %v", msg2.SenderID, got.LastReplyBy)
	}
	if got.LastReplyRole != string(domain.UserRoleUser) {
		t.Fatalf("unexpected last_reply_role: %q", got.LastReplyRole)
	}
}

func TestSQLiteRepo_UpsertPushTokenRejectsEmptyToken(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	bad := &domain.PushToken{UserID: 1, Platform: "android", Token: "", DeviceID: "dev-1"}
	if err := r.UpsertPushToken(ctx, bad); err == nil {
		t.Fatalf("expected empty push token to be rejected")
	} else if err != appshared.ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	good := &domain.PushToken{UserID: 1, Platform: "android", Token: "tok-1", DeviceID: "dev-1"}
	if err := r.UpsertPushToken(ctx, good); err != nil {
		t.Fatalf("upsert valid push token: %v", err)
	}
	tokens, err := r.ListPushTokensByUserIDs(ctx, []int64{1})
	if err != nil || len(tokens) != 1 || tokens[0].Token != "tok-1" {
		t.Fatalf("expected exactly one valid push token, got %v err=%v", tokens, err)
	}
}

func TestSQLiteRepo_MarkProbeEnrollTokenUsedIsIdempotent(t *testing.T) {
	rawDB, r := newTestRepo(t)
	ctx := context.Background()

	node := &domain.ProbeNode{Name: "probe-1", AgentID: "agent-1", SecretHash: "h1", Status: domain.ProbeStatusOffline, TagsJSON: "[]"}
	if err := r.CreateProbeNode(ctx, node); err != nil {
		t.Fatalf("create probe node: %v", err)
	}
	tok := &domain.ProbeEnrollToken{ProbeID: node.ID, TokenHash: "hash-token-1", ExpiresAt: time.Now().Add(time.Hour)}
	if err := r.CreateProbeEnrollToken(ctx, tok); err != nil {
		t.Fatalf("create enroll token: %v", err)
	}

	first := time.Now().Add(-time.Minute)
	if err := r.MarkProbeEnrollTokenUsed(ctx, tok.ID, first); err != nil {
		t.Fatalf("mark token used: %v", err)
	}
	// A second consumption attempt (e.g. a concurrent Enroll that passed the
	// validity check earlier) must not clobber the original used_at.
	if err := r.MarkProbeEnrollTokenUsed(ctx, tok.ID, time.Now()); err != nil {
		t.Fatalf("mark token used again: %v", err)
	}

	var usedAt *time.Time
	if err := rawDB.QueryRow("SELECT used_at FROM probe_enroll_tokens WHERE id = ?", tok.ID).Scan(&usedAt); err != nil {
		t.Fatalf("read used_at: %v", err)
	}
	if usedAt == nil {
		t.Fatalf("expected used_at set after mark")
	}
	if !usedAt.Equal(first) {
		t.Fatalf("second mark clobbered used_at: got %v want %v", usedAt, first)
	}
	if _, err := r.GetValidProbeEnrollTokenByHash(ctx, "hash-token-1", time.Now()); err == nil {
		t.Fatalf("used token must not be returned as valid")
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
