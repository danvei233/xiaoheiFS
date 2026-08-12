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

// web_api_contract F5：管理端用户列表 keyword 服务端过滤（此前前端全量 500
// 条/页拉取后客户端过滤，用户量大时每次搜索 = ~100 个串行请求）。
func TestHandlers_AdminUsers_KeywordFilter(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "ufadmin", "ufadmin@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	createUser := func(username, email string) {
		u := domain.User{Username: username, Email: email, PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
		if err := env.Repo.CreateUser(context.Background(), &u); err != nil {
			t.Fatalf("create user %s: %v", username, err)
		}
	}
	createUser("alpha-user", "alpha@example.com")
	createUser("beta-user", "beta@example.com")
	createUser("gamma-user", "gamma@example.com")

	// keyword=beta 只返回 beta-user
	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/users?keyword=beta&limit=50", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list users: %d", rec.Code)
	}
	var list struct {
		Items []struct {
			Username string `json:"username"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode users: %v", err)
	}
	if list.Total != 1 || len(list.Items) != 1 {
		t.Fatalf("keyword=beta expected 1 user, got total=%d items=%d", list.Total, len(list.Items))
	}
	if list.Items[0].Username != "beta-user" {
		t.Fatalf("expected beta-user, got %s", list.Items[0].Username)
	}

	// 无过滤返回全部 3 个普通用户（不含管理员）
	rec2 := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/users?limit=50", nil, token)
	var list2 struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &list2); err != nil {
		t.Fatalf("decode users: %v", err)
	}
	if list2.Total != 3 {
		t.Fatalf("no filter expected 3 regular users (admins excluded), got total=%d", list2.Total)
	}
}
