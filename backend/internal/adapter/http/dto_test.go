package http

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	apppayment "xiaoheiplay/internal/app/payment"
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
		Key:          "balance",
		Name:         "Balance",
		Instructions: "Use wallet balance",
		Notice:       "Immediate",
		Balance:      1250,
	}}
	methodDTOs := toPaymentMethodDTOs(methods)
	if len(methodDTOs) != 1 || methodDTOs[0].Key != "balance" || methodDTOs[0].Balance != 12.5 || methodDTOs[0].Instructions == "" || methodDTOs[0].Notice == "" {
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
	realDTO := toRealNameVerificationDTO(realname, "")
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

func TestToRealNameVerificationDTO_ParsePendingFaceRedirectURL(t *testing.T) {
	redirect := "https://e.mangzhuyun.cn/face?token=abc"
	encoded := base64.RawURLEncoding.EncodeToString([]byte(redirect))
	record := domain.RealNameVerification{
		ID:       1,
		UserID:   2,
		RealName: "Alice",
		IDNumber: "11010519491231002X",
		Status:   "pending",
		Provider: "plugin/mangzhu_realname/default",
		Reason:   "pending_face:baidu:token123:" + encoded,
	}
	dto := toRealNameVerificationDTO(record, parsePendingRedirectURL(record.Reason))
	if dto.RedirectURL != redirect {
		t.Fatalf("redirect_url parse failed: got=%q", dto.RedirectURL)
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

func TestParseRawJSON_DoubleEncodedObject(t *testing.T) {
	raw := parseRawJSON("\"{\\\"cpu\\\":4,\\\"memory_gb\\\":8}\"")
	var decoded string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal decoded raw string: %v", err)
	}
	if decoded != "{\"cpu\":4,\"memory_gb\":8}" {
		t.Fatalf("expected untouched nested json string, got %q", decoded)
	}
}

func TestParseRawJSON_PlainObject(t *testing.T) {
	raw := parseRawJSON("{\"cpu\":2}")
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal plain raw: %v", err)
	}
	if decoded["cpu"] != float64(2) {
		t.Fatalf("expected cpu=2, got %v", decoded["cpu"])
	}
}

func TestParseRawJSON_EscapedObjectWithoutOuterQuotes(t *testing.T) {
	raw := parseRawJSON("{\\\"cpu\\\":6,\\\"memory_gb\\\":12}")
	var decoded string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal escaped raw string: %v", err)
	}
	if decoded != "{\\\"cpu\\\":6,\\\"memory_gb\\\":12}" {
		t.Fatalf("expected raw escaped payload unchanged, got %q", decoded)
	}
}

func TestToSettingDTO_LeavesDoubleEncodedJSONValueUntouched(t *testing.T) {
	dto := toSettingDTO(domain.Setting{
		Key:       "site_nav_items",
		ValueJSON: "\"[{\\\"label\\\":\\\"产品\\\",\\\"url\\\":\\\"/products\\\"}]\"",
	})
	if dto.Value != "\"[{\\\"label\\\":\\\"产品\\\",\\\"url\\\":\\\"/products\\\"}]\"" {
		t.Fatalf("expected untouched double-encoded value, got=%q", dto.Value)
	}
}

func TestToSettingDTO_LeavesPlainTextUntouched(t *testing.T) {
	input := "hello-world"
	dto := toSettingDTO(domain.Setting{Key: "site_name", ValueJSON: input})
	if dto.Value != input {
		t.Fatalf("plain text should remain unchanged: got=%q want=%q", dto.Value, input)
	}
}

// TestRequestDTO_BindingValidation guards the request-contract validation tags on
// catalog/email DTOs. These DTOs are bound via bindJSON (gin ShouldBindJSON with
// `binding` tags); empty/negative required fields must be rejected instead of
// being persisted verbatim by the catalog/email services.
func TestRequestDTO_BindingValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name    string
		payload any
		body    string
	}{
		{name: "region empty name", payload: &RegionDTO{}, body: `{"code":"","name":""}`},
		{name: "plan group empty name", payload: &PlanGroupDTO{}, body: `{"region_id":1,"name":""}`},
		{name: "plan group negative unit_core", payload: &PlanGroupDTO{}, body: `{"region_id":1,"name":"x","unit_core":-5}`},
		{name: "plan group negative unit_bw", payload: &PlanGroupDTO{}, body: `{"region_id":1,"name":"x","unit_bw":-1}`},
		{name: "package empty name", payload: &PackageDTO{}, body: `{"plan_group_id":1,"name":""}`},
		{name: "package negative monthly_price", payload: &PackageDTO{}, body: `{"plan_group_id":1,"name":"x","monthly_price":-1}`},
		{name: "billing cycle empty name", payload: &BillingCycleDTO{}, body: `{"name":"","months":1,"multiplier":1}`},
		{name: "billing cycle zero months", payload: &BillingCycleDTO{}, body: `{"name":"x","months":0,"multiplier":1}`},
		{name: "billing cycle negative multiplier", payload: &BillingCycleDTO{}, body: `{"name":"x","months":1,"multiplier":-1}`},
		{name: "system image empty name", payload: &SystemImageDTO{}, body: `{"name":""}`},
		{name: "email template empty subject", payload: &EmailTemplateDTO{}, body: `{"name":"x","subject":""}`},
		{name: "email template empty name", payload: &EmailTemplateDTO{}, body: `{"name":"","subject":"x"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			c.Request.Header.Set("Content-Type", "application/json")
			if err := bindJSON(c, tc.payload); err == nil {
				t.Fatalf("expected validation error for body %s", tc.body)
			}
		})
	}
}

// TestParseAnyInt64_FloatRoundsNotTruncates guards against float64 spec values
// being truncated (9.99 -> 9) instead of rounded (9.99 -> 10) in order-item specs.
func TestParseAnyInt64_FloatRoundsNotTruncates(t *testing.T) {
	got, ok := parseAnyInt64(float64(9.99))
	if !ok {
		t.Fatal("parseAnyInt64(9.99) returned ok=false")
	}
	if got != 10 {
		t.Fatalf("parseAnyInt64(9.99) = %d, want 10 (must round, not truncate)", got)
	}
}

// TestMaskIDNumber_MasksShortIDs guards that short ID numbers (e.g. passport-style)
// are masked instead of returned in full.
func TestMaskIDNumber_MasksShortIDs(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "ABCD", want: "****"},
		{in: "AB12345", want: "AB****"},
		{in: "1234567890123456", want: "1234****3456"},
	}
	for _, tc := range cases {
		if got := maskIDNumber(tc.in); got != tc.want {
			t.Fatalf("maskIDNumber(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestPaymentProviderDTO_MasksSecrets locks P2-13: the admin providers list
// must not echo payment-channel secrets (merchant keys / app secrets) in the
// config JSON; secret-keyed fields are masked, and the update path preserves
// the previously stored secret when the masked literal is submitted back.
func TestPaymentProviderDTO_MasksSecrets(t *testing.T) {
	providers := []shared.PaymentProviderInfo{{
		Key:        "ezpay",
		Name:       "EZPay",
		Enabled:    true,
		SchemaJSON: `{"fields":[]}`,
		ConfigJSON: `{"merchant_id":"MID-1","merchant_key":"REAL-SECRET","app_secret":"APP-SECRET","notify_url":"https://x/p"}`,
	}}
	dtos := toPaymentProviderDTOs(providers)
	if len(dtos) != 1 {
		t.Fatalf("provider dto list mismatch: %+v", dtos)
	}
	cfg := dtos[0].ConfigJSON
	if strings.Contains(cfg, "REAL-SECRET") || strings.Contains(cfg, "APP-SECRET") {
		t.Fatalf("provider list leaked a secret: %s", cfg)
	}
	if !strings.Contains(cfg, apppayment.ConfigSecretMask) {
		t.Fatalf("expected a masked secret in provider config: %s", cfg)
	}
	// Non-secret values are preserved.
	if !strings.Contains(cfg, "MID-1") || !strings.Contains(cfg, "https://x/p") {
		t.Fatalf("non-secret config fields were dropped: %s", cfg)
	}

	// The masked config round-trips without overwriting the stored secret.
	existing := `{"merchant_id":"MID-1","merchant_key":"REAL-SECRET","app_secret":"APP-SECRET","notify_url":"https://x/p"}`
	merged, err := apppayment.MergeConfigMask(existing, cfg)
	if err != nil {
		t.Fatalf("merge masked config: %v", err)
	}
	if !strings.Contains(merged, "REAL-SECRET") || !strings.Contains(merged, "APP-SECRET") {
		t.Fatalf("masked round-trip overwrote the stored secret: %s", merged)
	}

	nested := toPaymentProviderDTO(shared.PaymentProviderInfo{
		ConfigJSON: `{"credentials":{"api_key":"NESTED-SECRET"},"accounts":[{"id":"primary","private_key":"ARRAY-SECRET","name":"visible"}]}`,
	}).ConfigJSON
	if strings.Contains(nested, "NESTED-SECRET") || strings.Contains(nested, "ARRAY-SECRET") {
		t.Fatalf("nested provider credentials leaked: %s", nested)
	}
	nestedMerged, err := apppayment.MergeConfigMask(
		`{"credentials":{"api_key":"NESTED-SECRET"},"accounts":[{"id":"primary","private_key":"ARRAY-SECRET","name":"visible"}]}`,
		nested,
	)
	if err != nil {
		t.Fatalf("merge nested config: %v", err)
	}
	if !strings.Contains(nestedMerged, "NESTED-SECRET") || !strings.Contains(nestedMerged, "ARRAY-SECRET") {
		t.Fatalf("nested masked round-trip overwrote stored credentials: %s", nestedMerged)
	}

	camelAndSchema := toPaymentProviderDTO(shared.PaymentProviderInfo{
		SchemaJSON: `{"type":"object","properties":{"opaque":{"type":"string","x-secret":true},"passwordField":{"type":"string","format":"password"}}}`,
		ConfigJSON: `{"privateKey":"CAMEL-PRIVATE","clientSecret":"CAMEL-CLIENT","accessToken":"CAMEL-TOKEN","opaque":"SCHEMA-SECRET","passwordField":"FORMAT-SECRET","label":"visible"}`,
	}).ConfigJSON
	for _, secret := range []string{"CAMEL-PRIVATE", "CAMEL-CLIENT", "CAMEL-TOKEN", "SCHEMA-SECRET", "FORMAT-SECRET"} {
		if strings.Contains(camelAndSchema, secret) {
			t.Fatalf("provider schema/camelCase credential leaked (%s): %s", secret, camelAndSchema)
		}
	}
	if !strings.Contains(camelAndSchema, `"label":"visible"`) {
		t.Fatalf("non-secret provider field was lost: %s", camelAndSchema)
	}

	reordered, err := apppayment.MergeConfigMask(
		`{"accounts":[{"id":"a","privateKey":"SECRET-A"},{"id":"b","privateKey":"SECRET-B"}]}`,
		`{"accounts":[{"id":"b","privateKey":"********"},{"id":"a","privateKey":"********"}]}`,
	)
	if err != nil {
		t.Fatalf("merge reordered config: %v", err)
	}
	if !strings.Contains(reordered, `"id":"b","privateKey":"SECRET-B"`) ||
		!strings.Contains(reordered, `"id":"a","privateKey":"SECRET-A"`) {
		t.Fatalf("reordered account array restored credentials by index: %s", reordered)
	}

	_, err = apppayment.MergeConfigMask(
		`{"accounts":[{"name":"A","privateKey":"SECRET-A"},{"name":"B","privateKey":"SECRET-B"}]}`,
		`{"accounts":[{"name":"B","privateKey":"********"},{"name":"A","privateKey":"********"}]}`,
	)
	if !errors.Is(err, domain.ErrPaymentConfigMaskedSecretUnmatched) {
		t.Fatalf("anonymous reordered accounts must fail closed, got %v", err)
	}
	_, err = apppayment.MergeConfigMask(
		`{"accounts":[{"id":"dup","privateKey":"SECRET-A"},{"id":"dup","privateKey":"SECRET-B"}]}`,
		`{"accounts":[{"id":"dup","privateKey":"********"}]}`,
	)
	if !errors.Is(err, domain.ErrPaymentConfigMaskedSecretUnmatched) {
		t.Fatalf("duplicate account identities must fail closed, got %v", err)
	}

	topArray := toPaymentProviderDTO(shared.PaymentProviderInfo{ConfigJSON: `[{"secret":"TOP-SECRET","label":"visible"}]`}).ConfigJSON
	if strings.Contains(topArray, "TOP-SECRET") || !strings.Contains(topArray, apppayment.ConfigSecretMask) {
		t.Fatalf("top-level array leaked a secret: %s", topArray)
	}
	composed := toPaymentProviderDTO(shared.PaymentProviderInfo{
		SchemaJSON: `{"type":"object","$defs":{"credential":{"type":"string","x-secret":true}},"properties":{"custom":{"$ref":"#/$defs/credential"}},"allOf":[{"properties":{"other":{"type":"string","format":"password"}}}]}`,
		ConfigJSON: `{"custom":"REF-SECRET","other":"COMPOSED-SECRET","webhookSecret":"HOOK-SECRET","label":"visible"}`,
	}).ConfigJSON
	for _, secret := range []string{"REF-SECRET", "COMPOSED-SECRET", "HOOK-SECRET"} {
		if strings.Contains(composed, secret) {
			t.Fatalf("composed-schema secret leaked (%s): %s", secret, composed)
		}
	}

	arraySecrets := toPaymentProviderDTO(shared.PaymentProviderInfo{
		SchemaJSON: `{"type":"object","properties":{"keys":{"type":"array","items":{"type":"string","format":"password"}}}}`,
		ConfigJSON: `{"keys":["ARRAY-SECRET-A","ARRAY-SECRET-B"]}`,
	}).ConfigJSON
	if strings.Contains(arraySecrets, "ARRAY-SECRET") || strings.Count(arraySecrets, apppayment.ConfigSecretMask) != 2 {
		t.Fatalf("schema-declared array secrets leaked: %s", arraySecrets)
	}

	wildcardSecrets := toPaymentProviderDTO(shared.PaymentProviderInfo{
		SchemaJSON: `{"type":"object","properties":{"tenants":{"type":"object","additionalProperties":{"type":"object","additionalProperties":{"type":"string","x-secret":true}}}}}`,
		ConfigJSON: `{"tenants":{"acme":{"primary":"MULTI-WILDCARD-SECRET"}}}`,
	}).ConfigJSON
	if strings.Contains(wildcardSecrets, "MULTI-WILDCARD-SECRET") {
		t.Fatalf("multi-wildcard schema secret leaked: %s", wildcardSecrets)
	}

	recursiveSchema := `{"$defs":{"node":{"type":"object","properties":{"label":{"type":"string"},"secret":{"type":"string","x-secret":true},"children":{"type":"array","items":{"$ref":"#/$defs/node"}}}}},"type":"object","properties":{"root":{"$ref":"#/$defs/node"}}}`
	recursive := toPaymentProviderDTO(shared.PaymentProviderInfo{
		SchemaJSON: recursiveSchema,
		ConfigJSON: `{"root":{"label":"visible","secret":"ROOT-SECRET","children":[{"label":"child","secret":"CHILD-SECRET"}]}}`,
	}).ConfigJSON
	if strings.Contains(recursive, "ROOT-SECRET") || strings.Contains(recursive, "CHILD-SECRET") {
		t.Fatalf("recursive-schema secret leaked: %s", recursive)
	}
}
