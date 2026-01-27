package http_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_TicketResourceDTO(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "ticketres", "ticketres@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/tickets", map[string]any{
		"subject": "Need help",
		"content": "details",
		"resources": []map[string]any{
			{"resource_type": "vps", "resource_id": 123, "resource_name": "node-1"},
		},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("ticket create: %d", rec.Code)
	}
	var resp struct {
		Ticket struct {
			ID int64 `json:"id"`
		} `json:"ticket"`
		Resources []struct {
			ResourceType string `json:"resource_type"`
			ResourceID   int64  `json:"resource_id"`
			ResourceName string `json:"resource_name"`
		} `json:"resources"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Ticket.ID == 0 || len(resp.Resources) != 1 {
		t.Fatalf("ticket resources missing")
	}
	if resp.Resources[0].ResourceType != "vps" || resp.Resources[0].ResourceID != 123 || resp.Resources[0].ResourceName != "node-1" {
		t.Fatalf("ticket resource fields mismatch")
	}
}
