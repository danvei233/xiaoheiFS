package http

import (
	"encoding/json"
	"testing"
	"time"
	"xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func TestDTO_Mappings(t *testing.T) {
	now := time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)
	permGroupID := int64(7)
	reviewerID := int64(8)
	verifiedAt := now.Add(2 * time.Hour)

	user := domain.User{
		ID:                101,
		Username:          "alice",
		Email:             "alice@example.com",
		QQ:                "12345",
		Phone:             "18800001111",
		Bio:               "bio",
		Intro:             "intro",
		Avatar:            "avatars/a.png",
		PermissionGroupID: &permGroupID,
		Role:              domain.UserRoleUser,
		Status:            domain.UserStatusActive,
		CreatedAt:         now,
		UpdatedAt:         now.Add(time.Hour),
	}
	userDTO := toUserDTO(user)
	if userDTO.ID != user.ID || userDTO.Username != user.Username || userDTO.Role != string(user.Role) {
		t.Fatalf("user dto mismatch: %+v", userDTO)
	}
	users := toUserDTOs([]domain.User{user})
	if len(users) != 1 || users[0].Email != user.Email {
		t.Fatalf("user dto list mismatch: %+v", users)
	}

	order := domain.Order{
		ID:             501,
		UserID:         user.ID,
		OrderNo:        "ORD-501",
		Status:         domain.OrderStatusPendingPayment,
		TotalAmount:    9950,
		Currency:       "CNY",
		IdempotencyKey: "idem-1",
		PendingReason:  "need_review",
		CreatedAt:      now,
		UpdatedAt:      now.Add(5 * time.Minute),
	}
	orderDTO := toOrderDTO(order)
	if orderDTO.ID != order.ID || orderDTO.CanReview != true || orderDTO.Status != string(order.Status) {
		t.Fatalf("order dto mismatch: %+v", orderDTO)
	}
	if got := toOrderDTOs([]domain.Order{order}); len(got) != 1 || got[0].OrderNo != order.OrderNo {
		t.Fatalf("order dto list mismatch: %+v", got)
	}
	reviewOrder := order
	reviewOrder.Status = domain.OrderStatusPendingReview
	reviewDTO := toOrderDTO(reviewOrder)
	if !reviewDTO.CanReview {
		t.Fatalf("expected pending review order to be reviewable")
	}

	payment := domain.OrderPayment{
		ID:             900,
		OrderID:        order.ID,
		UserID:         user.ID,
		Method:         "manual",
		Amount:         9950,
		Currency:       "CNY",
		TradeNo:        "TRADE-900",
		Note:           "note",
		ScreenshotURL:  "https://img.local/1.png",
		Status:         domain.PaymentStatusPendingReview,
		IdempotencyKey: "pay-idem",
		ReviewedBy:     &reviewerID,
		ReviewReason:   "ok",
		CreatedAt:      now,
		UpdatedAt:      now.Add(10 * time.Minute),
	}
	payDTO := toOrderPaymentDTO(payment)
	if payDTO.ID != payment.ID || payDTO.Method != payment.Method || payDTO.Status != string(payment.Status) {
		t.Fatalf("payment dto mismatch: %+v", payDTO)
	}
	if got := toOrderPaymentDTOs([]domain.OrderPayment{payment}); len(got) != 1 || got[0].TradeNo != payment.TradeNo {
		t.Fatalf("payment dto list mismatch: %+v", got)
	}

	event := domain.OrderEvent{
		ID:        1001,
		OrderID:   order.ID,
		Seq:       2,
		Type:      "created",
		DataJSON:  `{"stage":"init"}`,
		CreatedAt: now,
	}
	evDTO := toOrderEventDTO(event)
	if evDTO.ID != event.ID || evDTO.Type != event.Type || string(evDTO.Data) == "" {
		t.Fatalf("order event dto mismatch: %+v", evDTO)
	}
	if got := toOrderEventDTOs([]domain.OrderEvent{event}); len(got) != 1 || got[0].Seq != event.Seq {
		t.Fatalf("order event dto list mismatch: %+v", got)
	}

	providers := []shared.PaymentProviderInfo{{
		Key:        "fake",
		Name:       "FakePay",
		Enabled:    true,
		SchemaJSON: `{"fields":[]}`,
		ConfigJSON: `{"k":"v"}`,
	}}
	providerDTOs := toPaymentProviderDTOs(providers)
	if len(providerDTOs) != 1 || providerDTOs[0].Key != "fake" || !providerDTOs[0].Enabled {
		t.Fatalf("provider dto list mismatch: %+v", providerDTOs)
	}

	methods := []shared.PaymentMethodInfo{{
		Key:        "balance",
		Name:       "Balance",
		SchemaJSON: `{}`,
		ConfigJSON: `{}`,
		Balance:    1250,
	}}
	methodDTOs := toPaymentMethodDTOs(methods)
	if len(methodDTOs) != 1 || methodDTOs[0].Key != "balance" || methodDTOs[0].Balance != 12.5 {
		t.Fatalf("method dto list mismatch: %+v", methodDTOs)
	}

	walletTx := domain.WalletTransaction{
		ID:        700,
		UserID:    user.ID,
		Amount:    2050,
		Type:      "credit",
		RefType:   "seed",
		RefID:     1,
		Note:      "init",
		CreatedAt: now,
	}
	txDTOs := toWalletTransactionDTOs([]domain.WalletTransaction{walletTx})
	if len(txDTOs) != 1 || txDTOs[0].Amount != centsToFloat(walletTx.Amount) {
		t.Fatalf("wallet tx dto mismatch: %+v", txDTOs)
	}

	walletOrder := domain.WalletOrder{
		ID:           800,
		UserID:       user.ID,
		Type:         domain.WalletOrderRecharge,
		Amount:       10000,
		Currency:     "CNY",
		Status:       domain.WalletOrderPendingReview,
		Note:         "recharge",
		MetaJSON:     `{"channel":"bank"}`,
		ReviewedBy:   &reviewerID,
		ReviewReason: "ok",
		CreatedAt:    now,
		UpdatedAt:    now.Add(time.Minute),
	}
	orderDTOs := toWalletOrderDTOs([]domain.WalletOrder{walletOrder})
	if len(orderDTOs) != 1 || orderDTOs[0].Status != string(walletOrder.Status) {
		t.Fatalf("wallet order dto mismatch: %+v", orderDTOs)
	}
	if orderDTOs[0].Meta["channel"] != "bank" {
		t.Fatalf("wallet order meta mismatch: %+v", orderDTOs[0].Meta)
	}

	realname := domain.RealNameVerification{
		ID:         300,
		UserID:     user.ID,
		RealName:   "Alice",
		IDNumber:   "1234567890123456",
		Status:     "verified",
		Provider:   "fake",
		Reason:     "",
		CreatedAt:  now,
		VerifiedAt: &verifiedAt,
	}
	realDTO := toRealNameVerificationDTO(realname)
	if realDTO.ID != realname.ID || realDTO.IDNumber != "1234****3456" {
		t.Fatalf("realname dto mismatch: %+v", realDTO)
	}

	audit := domain.AdminAuditLog{
		ID:         400,
		AdminID:    reviewerID,
		Action:     "order.create",
		TargetType: "order",
		TargetID:   "501",
		DetailJSON: `{"ip":"127.0.0.1"}`,
		CreatedAt:  now,
	}
	auditDTO := toAdminAuditLogDTO(audit)
	if auditDTO.ID != audit.ID || auditDTO.TargetID != audit.TargetID || string(auditDTO.Detail) == "" {
		t.Fatalf("audit dto mismatch: %+v", auditDTO)
	}

	apiKey := domain.APIKey{
		ID:         600,
		Name:       "key-1",
		KeyHash:    "hash",
		Status:     domain.APIKeyStatusActive,
		ScopesJSON: `["order.view","order.edit"]`,
		CreatedAt:  now,
		UpdatedAt:  now.Add(time.Minute),
	}
	keyDTO := toAPIKeyDTO(apiKey)
	if keyDTO.ID != apiKey.ID || keyDTO.Status != string(apiKey.Status) || len(keyDTO.Scopes) != 2 {
		t.Fatalf("apikey dto mismatch: %+v", keyDTO)
	}

	setting := domain.Setting{
		Key:       "site_name",
		ValueJSON: "demo",
		UpdatedAt: now,
	}
	setDTO := toSettingDTO(setting)
	if setDTO.Key != setting.Key || setDTO.Value != setting.ValueJSON {
		t.Fatalf("setting dto mismatch: %+v", setDTO)
	}

	tmpl := domain.EmailTemplate{
		ID:        701,
		Name:      "welcome",
		Subject:   "Hello",
		Body:      "body",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now.Add(time.Minute),
	}
	tmplDTO := toEmailTemplateDTO(tmpl)
	if tmplDTO.ID != tmpl.ID || tmplDTO.Name != tmpl.Name || !tmplDTO.Enabled {
		t.Fatalf("email template dto mismatch: %+v", tmplDTO)
	}

	syncLog := domain.IntegrationSyncLog{
		ID:        702,
		Target:    "automation",
		Mode:      "merge",
		Status:    "ok",
		Message:   "synced",
		CreatedAt: now,
	}
	syncDTO := toIntegrationSyncLogDTO(syncLog)
	if syncDTO.ID != syncLog.ID || syncDTO.Target != syncLog.Target {
		t.Fatalf("sync log dto mismatch: %+v", syncDTO)
	}

	autoLog := domain.AutomationLog{
		ID:           703,
		OrderID:      order.ID,
		OrderItemID:  1,
		Action:       "create",
		RequestJSON:  `{"req":"data"}`,
		ResponseJSON: `{"ok":true}`,
		Success:      true,
		Message:      "ok",
		CreatedAt:    now,
	}
	autoDTO := toAutomationLogDTO(autoLog)
	if autoDTO.ID != autoLog.ID || !autoDTO.Success || string(autoDTO.RequestJSON) == "" {
		t.Fatalf("automation log dto mismatch: %+v", autoDTO)
	}

	if _, err := json.Marshal(autoDTO.RequestJSON); err != nil {
		t.Fatalf("automation log raw json: %v", err)
	}
}

func TestDTO_ServerStatus_CompatFields(t *testing.T) {
	status := shared.ServerStatus{
		Hostname:        "h",
		CPUUsagePercent: 12.34,
		MemUsedPercent:  56.78,
		DiskUsedPercent: 90.12,
	}
	dto := toServerStatusDTO(status)
	if dto.MemUsagePercent != dto.MemUsedPercent {
		t.Fatalf("mem percent mismatch: used=%v usage=%v", dto.MemUsedPercent, dto.MemUsagePercent)
	}
	if dto.DiskUsagePercent != dto.DiskUsedPercent {
		t.Fatalf("disk percent mismatch: used=%v usage=%v", dto.DiskUsedPercent, dto.DiskUsagePercent)
	}
	b, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal server status dto: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal server status dto: %v", err)
	}
	if _, ok := m["mem_used_percent"]; !ok {
		t.Fatalf("expected mem_used_percent in json")
	}
	if _, ok := m["mem_usage_percent"]; !ok {
		t.Fatalf("expected mem_usage_percent in json")
	}
	if _, ok := m["disk_used_percent"]; !ok {
		t.Fatalf("expected disk_used_percent in json")
	}
	if _, ok := m["disk_usage_percent"]; !ok {
		t.Fatalf("expected disk_usage_percent in json")
	}
}

func TestOrderItemDTO_ResizeSpecAmountsConvertedToYuan(t *testing.T) {
	item := domain.OrderItem{
		ID:      1,
		OrderID: 2,
		Action:  "resize",
		SpecJSON: `{
			"current_monthly": 1500,
			"target_monthly": 2000,
			"charge_amount": 500,
			"refund_amount": 0
		}`,
	}
	dto := toOrderItemDTO(item)
	var spec map[string]any
	if err := json.Unmarshal(dto.Spec, &spec); err != nil {
		t.Fatalf("unmarshal spec: %v", err)
	}
	if spec["current_monthly"] != 15.0 {
		t.Fatalf("expected current_monthly 15.0, got %v", spec["current_monthly"])
	}
	if spec["target_monthly"] != 20.0 {
		t.Fatalf("expected target_monthly 20.0, got %v", spec["target_monthly"])
	}
	if spec["charge_amount"] != 5.0 {
		t.Fatalf("expected charge_amount 5.0, got %v", spec["charge_amount"])
	}
}
