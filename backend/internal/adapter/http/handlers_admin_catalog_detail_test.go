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

func TestHandlers_AdminCatalogListsAndUpdates(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "admincat2", "admincat2@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	regionID := createRegion(t, env, token)
	planGroupID := createPlanGroup(t, env, token, regionID)
	imageID := createSystemImage(t, env, token)
	pkgID := createPackage(t, env, token, planGroupID)

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/system-images", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("system images list: %d", rec.Code)
	}
	var images struct {
		Items []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &images)
	if len(images.Items) == 0 || images.Items[0].ID == 0 {
		t.Fatalf("system images list empty")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/system-images/"+testutil.Itoa(imageID), map[string]any{
		"name":    "ImageUpdated",
		"type":    "linux",
		"enabled": true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("system image update: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/system-images/"+testutil.Itoa(imageID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("system image delete: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/packages?plan_group_id="+testutil.Itoa(planGroupID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("packages list: %d", rec.Code)
	}
	var packages struct {
		Items []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &packages)
	if len(packages.Items) == 0 || packages.Items[0].ID == 0 {
		t.Fatalf("packages list empty")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/packages/"+testutil.Itoa(pkgID), map[string]any{
		"name": "PkgUpdated",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("package update: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/packages/"+testutil.Itoa(pkgID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("package delete: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/regions", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("regions list: %d", rec.Code)
	}
}

func TestHandlers_AdminRealNameRecordsList(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "adminrn", "adminrn@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	user := testutil.CreateUser(t, env.Repo, "rnuser", "rnuser@example.com", "pass")
	verifiedAt := time.Now().UTC()
	if err := env.Repo.CreateRealNameVerification(context.Background(), &domain.RealNameVerification{
		UserID:     user.ID,
		RealName:   "Tester",
		IDNumber:   "1234567890123456",
		Status:     "verified",
		Provider:   "fake",
		Reason:     "",
		VerifiedAt: &verifiedAt,
	}); err != nil {
		t.Fatalf("create realname: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/realname/records", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("realname records: %d", rec.Code)
	}
	var records struct {
		Items []struct {
			ID       int64  `json:"id"`
			IDNumber string `json:"id_number"`
			Status   string `json:"status"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &records)
	if records.Total == 0 || len(records.Items) == 0 {
		t.Fatalf("realname records empty")
	}
	if records.Items[0].IDNumber != "1234****3456" || records.Items[0].Status != "verified" {
		t.Fatalf("realname record fields mismatch")
	}
}
