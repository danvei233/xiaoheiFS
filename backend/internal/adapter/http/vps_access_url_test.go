package http

import "testing"

// web_url_content F6：面板/VNC 的 URL 来自自动化插件，会直接 302 到 Location。
// 插件被攻陷/第三方返回 javascript:/data:/协议相对时必须 fail-closed。
func TestIsSafeVPSAccessURL(t *testing.T) {
	allowed := []string{
		"https://panel.example.com:8888/",
		"http://192.168.1.100:8888/panel",
		"http://localhost:8080/panel",
		"https://vps.example.com/vnc?token=abc",
	}
	for _, raw := range allowed {
		if !isSafeVPSAccessURL(raw) {
			t.Fatalf("expected %q to be allowed", raw)
		}
	}

	rejected := []string{
		"",
		"javascript:alert(1)",
		"JaVaScRiPt:alert(1)",
		"data:text/html,<script>x</script>",
		"file:///etc/passwd",
		"//evil.example.com/steal",
		"https://user:pass@panel.example.com/",
		"https://",
		"ws://panel.example.com/socket",
		"vnc://panel.example.com/session",
	}
	for _, raw := range rejected {
		if isSafeVPSAccessURL(raw) {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}
