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

func TestAdminFlow_E2E(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "adminflow", "adminflow@example.com", "pass", groupID)

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
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	token := loginResp.AccessToken

	regionID := createRegion(t, env, token)
	list := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/regions", nil, token)
	if list.Code != http.StatusOK {
		t.Fatalf("region list: %d", list.Code)
	}
	update := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/regions/"+testutil.Itoa(regionID), map[string]any{
		"name": "Region99-upd",
	}, token)
	if update.Code != http.StatusOK {
		t.Fatalf("region update: %d", update.Code)
	}

	lineID := createLine(t, env, token, regionID)
	list = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/lines", nil, token)
	if list.Code != http.StatusOK {
		t.Fatalf("line list: %d", list.Code)
	}
	deleteLine := testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/lines/"+testutil.Itoa(lineID), nil, token)
	if deleteLine.Code != http.StatusOK {
		t.Fatalf("line delete: %d", deleteLine.Code)
	}

	planGroupID := createPlanGroup(t, env, token, regionID)
	pkgID := createPackage(t, env, token, planGroupID)
	if pkgID == 0 {
		t.Fatalf("package id missing")
	}
	cycleID := createBillingCycle(t, env, token)
	if cycleID == 0 {
		t.Fatalf("billing cycle id missing")
	}
	imageID := createSystemImage(t, env, token)
	if imageID == 0 {
		t.Fatalf("system image id missing")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/permissions/sync", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("permissions sync: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/permissions/list", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("permissions list: %d", rec.Code)
	}
	if err := env.Repo.UpsertPermission(context.Background(), &domain.Permission{Code: "order.view", Name: "Order View", Category: "order"}); err != nil {
		t.Fatalf("seed permission: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/permissions/order.view", map[string]any{
		"name":     "Order View",
		"category": "order",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("permissions patch: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":   "site_name",
		"value": "AdminFlow",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("settings patch: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/profile/change-password", map[string]any{
		"old_password": "pass",
		"new_password": "pass1234",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("change password: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/packages/"+testutil.Itoa(pkgID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("package delete: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/plan-groups/"+testutil.Itoa(planGroupID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("plan group delete: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/billing-cycles/"+testutil.Itoa(cycleID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("billing cycle delete: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/system-images/"+testutil.Itoa(imageID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("system image delete: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/regions/"+testutil.Itoa(regionID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("region delete: %d", rec.Code)
	}
}

func ensureAdminGroup(t *testing.T, env *testutilhttp.Env) int64 {
	groups, err := env.Repo.ListPermissionGroups(context.Background())
	if err != nil || len(groups) == 0 {
		t.Fatalf("permission groups: %v", err)
	}
	for _, g := range groups {
		if g.PermissionsJSON == `["*"]` {
			return g.ID
		}
	}
	group := domain.PermissionGroup{Name: "all", PermissionsJSON: `["*"]`}
	if err := env.Repo.CreatePermissionGroup(context.Background(), &group); err != nil {
		t.Fatalf("create permission group: %v", err)
	}
	return group.ID
}

func createRegion(t *testing.T, env *testutilhttp.Env, token string) int64 {
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/regions", map[string]any{
		"code":   "area-99",
		"name":   "Region99",
		"active": true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("region create: %d", rec.Code)
	}
	var resp struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return resp.ID
}

func createLine(t *testing.T, env *testutilhttp.Env, token string, regionID int64) int64 {
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/lines", map[string]any{
		"region_id": regionID,
		"name":      "Line1",
		"line_id":   100,
		"unit_core": 1,
		"unit_mem":  1,
		"unit_disk": 1,
		"unit_bw":   1,
		"active":    true,
		"visible":   true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("line create: %d", rec.Code)
	}
	var resp struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return resp.ID
}

func createPlanGroup(t *testing.T, env *testutilhttp.Env, token string, regionID int64) int64 {
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/plan-groups", map[string]any{
		"region_id": regionID,
		"name":      "PlanGroup1",
		"line_id":   101,
		"unit_core": 1,
		"unit_mem":  1,
		"unit_disk": 1,
		"unit_bw":   1,
		"active":    true,
		"visible":   true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("plan group create: %d", rec.Code)
	}
	var resp struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return resp.ID
}

func createPackage(t *testing.T, env *testutilhttp.Env, token string, planGroupID int64) int64 {
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/packages", map[string]any{
		"plan_group_id":  planGroupID,
		"name":           "Pkg1",
		"cores":          2,
		"memory_gb":      4,
		"disk_gb":        40,
		"bandwidth_mbps": 10,
		"cpu_model":      "x",
		"monthly_price":  10,
		"port_num":       30,
		"active":         true,
		"visible":        true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("package create: %d", rec.Code)
	}
	var resp struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return resp.ID
}

func createBillingCycle(t *testing.T, env *testutilhttp.Env, token string) int64 {
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/billing-cycles", map[string]any{
		"name":       "monthly",
		"months":     1,
		"multiplier": 1.0,
		"min_qty":    1,
		"max_qty":    12,
		"active":     true,
		"sort_order": 1,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("billing cycle create: %d", rec.Code)
	}
	var resp struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return resp.ID
}

func createSystemImage(t *testing.T, env *testutilhttp.Env, token string) int64 {
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/system-images", map[string]any{
		"image_id": 1,
		"name":     "Ubuntu",
		"type":     "linux",
		"enabled":  true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("system image create: %d", rec.Code)
	}
	var resp struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return resp.ID
}
