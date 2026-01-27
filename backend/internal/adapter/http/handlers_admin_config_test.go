package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_AdminConfigEndpoints(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, true)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "admincfg", "admincfg@example.com", "pass", groupID)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": admin.Username,
		"password": "pass",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin login: %d", rec.Code)
	}
	var loginResp struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &loginResp)
	token := loginResp.AccessToken

	if err := env.Repo.UpsertPermission(context.Background(), &domain.Permission{Code: "order.view", Name: "Order View", Category: "order"}); err != nil {
		t.Fatalf("seed permission: %v", err)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/permissions", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("permissions tree: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/permissions/order.view", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("permission detail: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/permission-groups", map[string]any{
		"name":        "ops",
		"description": "ops team",
		"permissions": []string{"order.view"},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("create permission group: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/permission-groups", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list permission groups: %d", rec.Code)
	}
	var group struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &group)
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/permission-groups/"+testutil.Itoa(group.ID), map[string]any{
		"name":        "ops-updated",
		"description": "ops team updated",
		"permissions": []string{"order.view"},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("update permission group: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/permission-groups/"+testutil.Itoa(group.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete permission group: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/profile", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin profile: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/profile", map[string]any{
		"email": "admincfg+1@example.com",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin profile update: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/admins", map[string]any{
		"username":            "opsadmin",
		"email":               "opsadmin@example.com",
		"password":            "pass",
		"permission_group_id": groupID,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin create: %d", rec.Code)
	}
	var adminResp struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &adminResp)
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/admins?status=active", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin list: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/admins/"+testutil.Itoa(adminResp.ID), map[string]any{
		"username":            "opsadmin2",
		"email":               "opsadmin2@example.com",
		"permission_group_id": groupID,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin update: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/admins/"+testutil.Itoa(adminResp.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin delete: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/integrations/robot", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("robot config: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/integrations/robot", map[string]any{
		"url":     "http://example.com/robot",
		"secret":  "secret",
		"enabled": true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("robot config update: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/realname/config", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("realname config: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/realname/config", map[string]any{
		"enabled":       true,
		"provider":      "fake",
		"block_actions": []string{"purchase_vps"},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("realname config update: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/realname/providers", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("realname providers: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/realname/records", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("realname records: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/integrations/smtp", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("smtp config: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/integrations/smtp", map[string]any{
		"host":    "smtp.example.com",
		"port":    "25",
		"user":    "user",
		"pass":    "pass",
		"from":    "noreply@example.com",
		"enabled": false,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("smtp config update: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/integrations/smtp/test", map[string]any{
		"to":      "test@example.com",
		"subject": "hello",
		"body":    "test",
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("smtp test expected error: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/email-templates", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("email templates list: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/email-templates", map[string]any{
		"name":    "custom",
		"subject": "hello",
		"body":    "<p>hi</p>",
		"enabled": true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("email template create: %d", rec.Code)
	}
	var tmpl struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &tmpl)
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/email-templates/"+testutil.Itoa(tmpl.ID), map[string]any{
		"name":    "custom",
		"subject": "hello2",
		"body":    "<p>hi2</p>",
		"enabled": true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("email template update: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/email-templates/"+testutil.Itoa(tmpl.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("email template delete: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/dashboard/overview", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard overview: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/dashboard/revenue", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard revenue: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/dashboard/vps-status", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard vps status: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/site/settings", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("site settings: %d", rec.Code)
	}
}
