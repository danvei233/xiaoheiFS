package http_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

// web_auth_surface F6：设置列表绝不回显密钥明文（掩码 "********"），提交掩码/
// 空值保留原值不覆盖。
func TestHandlers_AdminSettings_SecretsMaskedAndPreserved(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	token := issueSettingsAdminToken(t, env)

	// 写入一个真实密钥
	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":   "smtp_pass",
		"value": "super-secret-password",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 to seed smtp_pass, got %d: %s", rec.Code, rec.Body.String())
	}

	// 列表接口不得返回明文
	listRec := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/settings?limit=500", nil, token)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for settings list, got %d", listRec.Code)
	}
	body := listRec.Body.String()
	if strings.Contains(body, "super-secret-password") {
		t.Fatalf("settings list must not leak smtp_pass plaintext")
	}
	var list struct {
		Items []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode settings list: %v", err)
	}
	found := false
	for _, item := range list.Items {
		if item.Key == "smtp_pass" {
			found = true
			if item.Value != "********" {
				t.Fatalf("expected masked smtp_pass %q, got %q", "********", item.Value)
			}
		}
	}
	if !found {
		t.Fatalf("smtp_pass should be present in settings list (masked)")
	}

	// 提交掩码值 → 保留原值
	maskRec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":   "smtp_pass",
		"value": "********",
	}, token)
	if maskRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for masked smtp_pass submit, got %d", maskRec.Code)
	}
	rec2 := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/settings?limit=500", nil, token)
	if strings.Contains(rec2.Body.String(), "super-secret-password") {
		t.Fatalf("masked submit must keep the original secret (not overwrite with mask)")
	}

	// 提交空值 → 保留原值
	emptyRec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":   "smtp_pass",
		"value": "",
	}, token)
	if emptyRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty smtp_pass submit, got %d", emptyRec.Code)
	}

	// 提交新值 → 真正更新
	updateRec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":   "smtp_pass",
		"value": "brand-new-password",
	}, token)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for new smtp_pass submit, got %d", updateRec.Code)
	}
	rec3 := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/settings?limit=500", nil, token)
	if strings.Contains(rec3.Body.String(), "super-secret-password") {
		t.Fatalf("old secret must be gone after update")
	}
	if strings.Contains(rec3.Body.String(), "brand-new-password") {
		t.Fatalf("new secret must still be masked in the list")
	}
}

// web_auth_surface F6：SMTP 专用配置接口同样不回显明文，掩码提交保留原值。
func TestHandlers_AdminSMTPConfig_SecretMaskedAndPreserved(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	token := issueSettingsAdminToken(t, env)

	seed := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":   "smtp_pass",
		"value": "smtp-real-secret",
	}, token)
	if seed.Code != http.StatusOK {
		t.Fatalf("seed smtp_pass: %d %s", seed.Code, seed.Body.String())
	}

	getRec := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/integrations/smtp", nil, token)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for smtp config, got %d", getRec.Code)
	}
	var cfg struct {
		Pass string `json:"pass"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("decode smtp config: %v", err)
	}
	if cfg.Pass != "********" {
		t.Fatalf("expected masked smtp pass, got %q", cfg.Pass)
	}

	// 掩码提交保留原值
	maskUpdate := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/integrations/smtp", map[string]any{
		"host":    "smtp.example.com",
		"port":    "587",
		"user":    "mailer",
		"pass":    "********",
		"from":    "no-reply@example.com",
		"enabled": true,
	}, token)
	if maskUpdate.Code != http.StatusOK {
		t.Fatalf("expected 200 for masked smtp update, got %d", maskUpdate.Code)
	}
	getRec2 := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/integrations/smtp", nil, token)
	if strings.Contains(getRec2.Body.String(), "smtp-real-secret") {
		t.Fatalf("masked smtp update must keep the original secret")
	}
}
