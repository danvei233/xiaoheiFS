package http

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// 复用 admin_path_validator_test.go 的 stubSettingsService：Get 返回配置的
// admin_path。真实管理路径固定为 myadmin。
func checkAdminPathHandlerWithPath(t *testing.T, configured string) *Handler {
	t.Helper()
	return &Handler{settingsSvc: &stubSettingsService{valueJSON: `"` + configured + `"`}}
}

func runCheckAdminPath(t *testing.T, h *Handler, body string) (int, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("POST", "/api/v1/check-admin-path", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.CheckAdminPath(c)
	var payload map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	return rec.Code, payload
}

// web_security F1：check-admin-path 不得成为任意字符串的枚举 oracle。
// 任意候选路径（含危险格式）除真实路径外一律 is_admin:false；危险格式
// （绝对 URL/协议相对/userinfo/多段）不进入比较、不泄露格式判定。
func TestCheckAdminPathNotAnOracle(t *testing.T) {
	h := checkAdminPathHandlerWithPath(t, "myadmin")

	// 真实路径首段返回 true
	code, payload := runCheckAdminPath(t, h, `{"path":"myadmin"}`)
	if code != 200 {
		t.Fatalf("real path: code=%d", code)
	}
	if payload["is_admin"] != true {
		t.Fatalf("real path should be is_admin=true, got %v", payload)
	}

	// 任意候选/危险格式全部 is_admin:false
	candidates := []string{
		// 任意候选
		"mypanel", "manage", "control", "admin1", "secret", "x", "a1b2c3",
		// 危险格式：不得进入比较、不得泄露格式判定
		"//evil.com", "https://evil.com", "javascript:alert(1)", "admin/login",
		"../admin", "admin\x00", "user:pass@host", "/", "myadmin/", "myadmin?x=1",
	}
	for _, path := range candidates {
		code, payload := runCheckAdminPath(t, h, `{"path":`+mustJSONString(path)+`}`)
		if code != 200 {
			t.Fatalf("candidate %q: unexpected code %d", path, code)
		}
		if payload["is_admin"] != false {
			t.Fatalf("candidate %q must be is_admin=false, got %v", path, payload)
		}
	}
}

func mustJSONString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}
