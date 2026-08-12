package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

// cross-module residual：钱包流水 status/type 服务端过滤 + 真实分页。
// 前端此前整批拉 500 条后本地切片，分页是伪装的。
func TestHandlers_WalletOrders_ServerSideFilterAndPagination(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "wallet-filter", "wallet-filter@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	// 用 repo 直接造 3 条不同 status 的充值单
	mk := func(status domain.WalletOrderStatus) {
		order := domain.WalletOrder{UserID: user.ID, Type: domain.WalletOrderRecharge, Amount: 1000, Currency: "CNY", Status: status, Note: "seed"}
		if err := env.Repo.CreateWalletOrder(context.Background(), &order); err != nil {
			t.Fatalf("create wallet order: %v", err)
		}
	}
	mk(domain.WalletOrderApproved)
	mk(domain.WalletOrderPendingReview)
	mk(domain.WalletOrderApproved)

	// status=approved → 2 条
	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/wallet/orders?status=approved&limit=10", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet orders: %d", rec.Code)
	}
	var list struct {
		Items []struct {
			Status string `json:"status"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode wallet orders: %v", err)
	}
	if list.Total != 2 {
		t.Fatalf("status=approved expected 2, got total=%d", list.Total)
	}

	// 分页：limit=1 offset=0 → total 仍为 3，items 只有 1 条
	rec2 := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/wallet/orders?limit=1&offset=0", nil, token)
	var list2 struct {
		Items []json.RawMessage `json:"items"`
		Total int               `json:"total"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &list2); err != nil {
		t.Fatalf("decode wallet orders: %v", err)
	}
	if list2.Total != 3 {
		t.Fatalf("pagination total expected 3, got %d", list2.Total)
	}
	if len(list2.Items) != 1 {
		t.Fatalf("pagination limit=1 expected 1 item, got %d", len(list2.Items))
	}
}

func TestHandlers_WalletOrderSummaryUsesDatabaseAggregates(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "wallet-summary", "wallet-summary@example.com", "pass")
	other := testutil.CreateUser(t, env.Repo, "wallet-summary-other", "wallet-summary-other@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	seed := func(userID int64, typ domain.WalletOrderType, status domain.WalletOrderStatus, amount int64) {
		order := domain.WalletOrder{UserID: userID, Type: typ, Amount: amount, Currency: "CNY", Status: status, Note: "seed"}
		if err := env.Repo.CreateWalletOrder(context.Background(), &order); err != nil {
			t.Fatalf("create wallet order: %v", err)
		}
	}
	seed(user.ID, domain.WalletOrderRecharge, domain.WalletOrderApproved, 1250)
	seed(user.ID, domain.WalletOrderWithdraw, domain.WalletOrderApproved, 400)
	seed(user.ID, domain.WalletOrderRecharge, domain.WalletOrderPendingReview, 9999)
	seed(other.ID, domain.WalletOrderRecharge, domain.WalletOrderApproved, 500000)

	// Place more than the old 10,000-row frontend cap behind the API. A SQL
	// aggregate must include all of them without loading or page-looping.
	tx, err := env.SQLDB.Begin()
	if err != nil {
		t.Fatalf("begin bulk seed: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO wallet_orders (user_id, type, amount, currency, status, note, meta_json, created_at, updated_at) VALUES (?, 'recharge', 1, 'CNY', 'pending_review', 'bulk', '{}', ?, ?)`)
	if err != nil {
		t.Fatalf("prepare bulk seed: %v", err)
	}
	now := time.Now()
	for i := 0; i < 10001; i++ {
		if _, err := stmt.Exec(user.ID, now, now); err != nil {
			t.Fatalf("bulk seed row %d: %v", i, err)
		}
	}
	if err := stmt.Close(); err != nil {
		t.Fatalf("close bulk seed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit bulk seed: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/wallet/orders/summary", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet order summary: %d %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Recharge float64 `json:"recharge"`
		Withdraw float64 `json:"withdraw"`
		Pending  int64   `json:"pending"`
		Total    int64   `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode wallet summary: %v", err)
	}
	if got.Recharge != 12.5 || got.Withdraw != 4 || got.Pending != 10002 || got.Total != 10004 {
		t.Fatalf("unexpected wallet summary: %+v", got)
	}
}
