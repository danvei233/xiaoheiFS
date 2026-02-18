package http_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_CMSPublicAndAdmin(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, true)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "admincms", "admincms@example.com", "pass", groupID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/cms/categories", map[string]any{
		"key":        "news",
		"name":       "News",
		"lang":       "zh-CN",
		"sort_order": 1,
		"visible":    true,
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("create category: %d", rec.Code)
	}
	var cat struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &cat)

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/cms/categories/"+testutil.Itoa(cat.ID), map[string]any{
		"name": "News2",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("update category: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/cms/categories", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("list categories: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/cms/posts", map[string]any{
		"category_id":  cat.ID,
		"title":        "Post",
		"slug":         "post-1",
		"summary":      "sum",
		"content_html": "body",
		"lang":         "zh-CN",
		"status":       "published",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("create post: %d", rec.Code)
	}
	var post struct {
		ID   int64  `json:"id"`
		Slug string `json:"slug"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &post)

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/cms/posts/"+testutil.Itoa(post.ID), map[string]any{
		"title": "Post2",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("update post: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/cms/posts", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("list posts: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/cms/blocks", map[string]any{
		"page":         "home",
		"type":         "text",
		"title":        "Hero",
		"content_json": "{}",
		"lang":         "zh-CN",
		"visible":      true,
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("create block: %d", rec.Code)
	}
	var block struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &block)

	rec2 := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/cms/blocks", map[string]any{
		"page":         "api",
		"type":         "text",
		"title":        "Bad",
		"content_json": "{}",
		"lang":         "zh-CN",
		"visible":      true,
	}, adminToken)
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("expected reserved page to be rejected: %d", rec2.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/cms/blocks/"+testutil.Itoa(block.ID), map[string]any{
		"title": "Hero2",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("update block: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/cms/blocks", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("list blocks: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/cms/blocks/"+testutil.Itoa(block.ID), nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete block: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/cms/posts?lang=zh-CN", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("public posts: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/cms/posts/"+post.Slug, nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("public post detail: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/cms/posts/"+testutil.Itoa(post.ID), nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete post: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/cms/blocks?page=home&lang=zh-CN", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("public blocks: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/cms/categories/"+testutil.Itoa(cat.ID), nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete category: %d", rec.Code)
	}
}
