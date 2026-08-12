package walletorder_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	appwalletorder "xiaoheiplay/internal/app/walletorder"
	"xiaoheiplay/internal/domain"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/testutil"
)

type fakeAuditRepo struct {
	logs []domain.AdminAuditLog
}

func (f *fakeAuditRepo) AddAuditLog(ctx context.Context, log domain.AdminAuditLog) error {
	f.logs = append(f.logs, log)
	return nil
}

func (f *fakeAuditRepo) ListAuditLogs(ctx context.Context, filter appshared.AuditLogFilter, limit, offset int) ([]domain.AdminAuditLog, int, error) {
	return f.logs, len(f.logs), nil
}

// failingDeleteAutomation returns a provider-style error that embeds a signed
// URL, an internal IP and a token — exactly what must never be persisted.
type failingDeleteAutomation struct {
	*testutil.FakeAutomationClient
}

func (f *failingDeleteAutomation) DeleteHost(context.Context, int64) error {
	if f.FakeAutomationClient.DeleteErr != nil {
		return f.FakeAutomationClient.DeleteErr
	}
	return nil
}

// TestWalletOrderService_RefundCleanupFailurePersistsStableCode locks the secret
// hygiene contract: when DeleteHost fails, the wallet-order meta and audit log
// must persist only the centralized error code and a sanitized one-line summary,
// never the raw provider error text (which can carry URLs, tokens and topology).
func TestWalletOrderService_RefundCleanupFailurePersistsStableCode(t *testing.T) {
	settings := &fakeSettingsRepo{values: map[string]string{}}
	wallets := &fakeWalletRepo{}
	orders := &fakeWalletOrderRepo{wallets: wallets}
	item := domain.OrderItem{ID: 40, Amount: 10000, Status: domain.OrderItemStatusActive, Action: "create", SpecJSON: "{}"}
	inst := domain.VPSInstance{
		ID: 5, UserID: 1, OrderItemID: item.ID, AutomationInstanceID: "100", GoodsTypeID: 1,
		CreatedAt: time.Now(),
	}
	expire := time.Now().Add(24 * time.Hour)
	inst.ExpireAt = &expire

	secretToken := "sk-live-deadbeef0123456789abcdef"
	internalIP := "10.0.0.15"
	// The provider error embeds a signed callback URL and an internal address.
	deleteErr := &providerDeleteError{msg: "delete host 100 failed: POST http://" + internalIP + ":8080/v1/hosts/100/delete?token=" + secretToken}
	deleteClient := &failingDeleteAutomation{&testutil.FakeAutomationClient{DeleteErr: deleteErr}}

	vps := &fakeVPSRepo{inst: inst}
	items := &fakeOrderItemRepo{item: item}
	audit := &fakeAuditRepo{}
	svc := appwalletorder.NewService(orders, wallets, settings, vps, items, &testutil.FakeAutomationResolver{Client: deleteClient}, audit)

	refund := domain.WalletOrder{
		UserID: 1, Type: domain.WalletOrderRefund, Amount: 10000, Currency: "CNY",
		Status: domain.WalletOrderPendingReview, Note: "refund",
		MetaJSON: `{"delete_on_approve": true, "vps_id": 5}`,
	}
	if err := orders.CreateWalletOrder(context.Background(), &refund); err != nil {
		t.Fatalf("create refund order: %v", err)
	}

	if _, _, err := svc.Approve(context.Background(), 7, refund.ID); err != nil {
		t.Fatalf("approve refund: %v", err)
	}

	stored, err := orders.GetWalletOrder(context.Background(), refund.ID)
	if err != nil {
		t.Fatalf("get wallet order: %v", err)
	}
	meta := map[string]any{}
	if err := json.Unmarshal([]byte(stored.MetaJSON), &meta); err != nil {
		t.Fatalf("decode meta: %v", err)
	}
	if meta["delete_pending"] != true {
		t.Fatalf("expected delete_pending, got %v", meta)
	}
	code, _ := meta["delete_error_code"].(string)
	if code == "" {
		t.Fatalf("missing delete_error_code in meta: %v", meta)
	}
	summary, _ := meta["delete_error"].(string)
	if summary == "" {
		t.Fatalf("missing sanitized delete_error summary in meta: %v", meta)
	}
	metaJSON := stored.MetaJSON
	for _, secret := range []string{secretToken, internalIP, "sk-live-deadbeef"} {
		if strings.Contains(metaJSON, secret) {
			t.Fatalf("secret %q leaked into wallet-order meta: %s", secret, metaJSON)
		}
	}

	// Audit must also be sanitized.
	var foundAudit bool
	for _, l := range audit.logs {
		if l.Action == "wallet_order.refund_cleanup_failed" {
			foundAudit = true
			detail := l.DetailJSON
			for _, secret := range []string{secretToken, internalIP, "sk-live-deadbeef"} {
				if strings.Contains(detail, secret) {
					t.Fatalf("secret %q leaked into audit log: %s", secret, detail)
				}
			}
			if !strings.Contains(detail, code) && !strings.Contains(detail, "cleanup") {
				t.Fatalf("audit detail does not carry the stable code/summary: %s", detail)
			}
		}
	}
	if !foundAudit {
		t.Fatalf("no refund_cleanup_failed audit entry written")
	}
}

type providerDeleteError struct {
	msg string
}

func (e *providerDeleteError) Error() string { return e.msg }
