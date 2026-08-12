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

func TestHandlers_UserVPSListFiltersBeforePagination(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "user-vps-filter", "user-vps-filter@example.com", "pass")
	other := testutil.CreateUser(t, env.Repo, "user-vps-other", "user-vps-other@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	now := time.Now()
	create := func(userID int64, name, region string, status domain.VPSStatus, automationState int, expireAt time.Time) {
		inst := domain.VPSInstance{
			UserID: userID, AutomationInstanceID: name, Name: name, Region: region,
			Status: status, AutomationState: automationState, AdminStatus: domain.VPSAdminStatusNormal,
			ExpireAt: &expireAt, SpecJSON: "{}",
		}
		if err := env.Repo.CreateInstance(context.Background(), &inst); err != nil {
			t.Fatalf("create VPS %s: %v", name, err)
		}
	}
	create(user.ID, "older-match-a", "Tokyo", domain.VPSStatusRunning, 2, now.Add(48*time.Hour))
	create(user.ID, "older-match-b", "Tokyo", domain.VPSStatusRunning, 2, now.Add(72*time.Hour))
	create(user.ID, "newer-other", "Singapore", domain.VPSStatusRunning, 2, now.Add(30*24*time.Hour))
	create(user.ID, "failed-instance", "Tokyo", domain.VPSStatusFailed, 11, now.Add(48*time.Hour))
	create(user.ID, "deleting-instance", "Tokyo", domain.VPSStatusDeleting, 12, now.Add(48*time.Hour))
	create(user.ID, "expired-locked-instance", "Tokyo", domain.VPSStatusLocked, 10, now.Add(-time.Hour))
	create(other.ID, "other-match", "Tokyo", domain.VPSStatusRunning, 2, now.Add(24*time.Hour))
	create(other.ID, "other-expired-locked", "Tokyo", domain.VPSStatusLocked, 10, now.Add(-time.Hour))

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps?keyword=older-match-a&region=Tokyo&expire_days=7&limit=1", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list filtered VPS: %d %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode VPS list: %v", err)
	}
	if got.Total != 1 || len(got.Items) != 1 || got.Items[0].Name != "older-match-a" {
		t.Fatalf("filter must run before pagination and stay user-scoped: %+v", got)
	}

	// A second matching row proves count is computed before LIMIT/OFFSET, not
	// from the current page. The second page must contain the other match.
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps?keyword=older-match&region=Tokyo&expire_days=7&limit=1&offset=1", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("second filtered VPS page: %d %s", rec.Code, rec.Body.String())
	}
	got = struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
		Total int `json:"total"`
	}{}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode second VPS page: %v", err)
	}
	if got.Total != 2 || len(got.Items) != 1 || got.Items[0].Name != "older-match-a" {
		// IDs are descending, so older-match-a (inserted before b) is the
		// second row. The important contract is total=2 and one row here.
		t.Fatalf("filter must run before pagination: %+v", got)
	}

	for _, status := range []string{"failed", "deleting"} {
		rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps?status="+status, nil, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("frontend status %q must be accepted, got %d", status, rec.Code)
		}
		var statusResult struct {
			Items []struct {
				Name string `json:"name"`
			} `json:"items"`
			Total int `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &statusResult); err != nil {
			t.Fatalf("decode status %q: %v", status, err)
		}
		if statusResult.Total != 1 || len(statusResult.Items) != 1 || statusResult.Items[0].Name != status+"-instance" {
			t.Fatalf("status filter %q must use canonical domain status: %+v", status, statusResult)
		}
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps?status=expired_locked&limit=1", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expired status filter: %d %s", rec.Code, rec.Body.String())
	}
	var expiredResult struct {
		Items []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &expiredResult); err != nil {
		t.Fatalf("decode expired status: %v", err)
	}
	if expiredResult.Total != 1 || len(expiredResult.Items) != 1 || expiredResult.Items[0].Name != "expired-locked-instance" || expiredResult.Items[0].Status != string(domain.VPSStatusExpiredLocked) {
		t.Fatalf("expired_locked must include expired locked rows: %+v", expiredResult)
	}
}
