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

func TestHandlers_TicketAndNotifications(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "tuser", "tuser@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/tickets", map[string]any{
		"subject": "help",
		"content": "need support",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("ticket create code: %d", rec.Code)
	}
	var ticketResp struct {
		Ticket struct {
			ID int64 `json:"id"`
		} `json:"ticket"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &ticketResp); err != nil {
		t.Fatalf("decode ticket: %v", err)
	}

	other := testutil.CreateUser(t, env.Repo, "other", "other@example.com", "pass")
	otherToken := testutil.IssueJWT(t, env.JWTSecret, other.ID, "user", time.Hour)
	// P3-10: another user's ticket must be indistinguishable from a missing
	// ticket (404), never a 403 existence oracle.
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/tickets/"+testutil.Itoa(ticketResp.Ticket.ID), nil, otherToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for another user's ticket, got %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/tickets/"+testutil.Itoa(ticketResp.Ticket.ID)+"/messages", map[string]any{
		"content": "hijack",
	}, otherToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for replying to another user's ticket, got %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/tickets/"+testutil.Itoa(ticketResp.Ticket.ID)+"/close", nil, otherToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for closing another user's ticket, got %d", rec.Code)
	}
	// A genuinely missing ticket also returns 404.
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/tickets/999999", nil, otherToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a missing ticket, got %d", rec.Code)
	}

	n := domain.Notification{UserID: user.ID, Type: "info", Title: "hi", Content: "msg"}
	if err := env.Repo.CreateNotification(context.Background(), &n); err != nil {
		t.Fatalf("create notification: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/notifications", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("notifications code: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/notifications/unread-count", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("unread count code: %d", rec.Code)
	}
}

func TestHandlers_WalletRechargeWithdraw(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "wuser", "wuser@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/wallet/recharge", map[string]any{
		"amount":   10,
		"currency": "CNY",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet recharge code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/wallet/withdraw", map[string]any{
		"amount":   10,
		"currency": "CNY",
	}, token)
	if rec.Code != http.StatusConflict {
		t.Fatalf("wallet withdraw code: %d", rec.Code)
	}
}
