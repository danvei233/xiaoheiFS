package http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_AdminCatalogOps(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "admincat", "admincat@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	regionID := createRegion(t, env, token)
	planGroupID := createPlanGroup(t, env, token, regionID)
	planGroup, err := env.Repo.GetPlanGroup(context.Background(), planGroupID)
	if err != nil {
		t.Fatalf("get plan group: %v", err)
	}
	lineID := planGroup.LineID

	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/plan-groups/"+testutil.Itoa(planGroupID), map[string]any{
		"name": "PlanGroup2",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("plan group update: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/lines/"+testutil.Itoa(planGroupID), map[string]any{
		"name": "Line2",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("line update: %d", rec.Code)
	}

	imageID := createSystemImage(t, env, token)
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/lines/"+testutil.Itoa(planGroupID)+"/system-images", map[string]any{
		"image_ids": []int64{imageID},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("line system images: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/system-images?line_id="+testutil.Itoa(lineID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("system images list: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/system-images/"+testutil.Itoa(imageID), map[string]any{
		"name":    "Image2",
		"type":    "linux",
		"enabled": true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("system image update: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/system-images/sync?line_id="+testutil.Itoa(lineID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("system image sync: %d", rec.Code)
	}

	pkgID := createPackage(t, env, token, planGroupID)
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/packages?plan_group_id="+testutil.Itoa(planGroupID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("package list: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/packages/"+testutil.Itoa(pkgID), map[string]any{
		"name": "Pkg2",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("package update: %d", rec.Code)
	}

	cycleID := createBillingCycle(t, env, token)
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/billing-cycles", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("billing cycles list: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/billing-cycles/"+testutil.Itoa(cycleID), map[string]any{
		"name":       "monthly",
		"months":     1,
		"multiplier": 1.2,
		"min_qty":    1,
		"max_qty":    12,
		"active":     true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("billing cycle update: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/packages/bulk-delete", map[string]any{
		"ids": []int64{pkgID},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("package bulk delete: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/billing-cycles/bulk-delete", map[string]any{
		"ids": []int64{cycleID},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("billing cycle bulk delete: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/system-images/bulk-delete", map[string]any{
		"ids": []int64{imageID},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("system image bulk delete: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/plan-groups/bulk-delete", map[string]any{
		"ids": []int64{planGroupID},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("plan group bulk delete: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/regions/bulk-delete", map[string]any{
		"ids": []int64{regionID},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("region bulk delete: %d", rec.Code)
	}
}
