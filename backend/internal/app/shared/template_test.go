package shared

import (
	"strings"
	"testing"
)

func TestRenderTemplateTextBranchStripsCRLF(t *testing.T) {
	out := RenderTemplate("Hello {{.Name}}\r\nInjected: cc@evil.com", map[string]any{"Name": "Alice"}, false)
	if strings.Contains(out, "\r") || strings.Contains(out, "\n") {
		t.Fatalf("text template output must not contain CR/LF: %q", out)
	}
}

func TestRenderTemplateHTMLBranchEscapes(t *testing.T) {
	out := RenderTemplate("<b>{{.Name}}</b>", map[string]any{"Name": "<script>alert(1)</script>"}, true)
	if strings.Contains(out, "<script>") {
		t.Fatalf("html template output must escape user data: %q", out)
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("expected escaped output, got %q", out)
	}
}

func TestSanitizePlainTextStripsTags(t *testing.T) {
	got := SanitizePlainText("<script>alert(1)</script>hello")
	if strings.Contains(got, "<script>") {
		t.Fatalf("tags not stripped: %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Fatalf("text content lost: %q", got)
	}
}

func TestSanitizePlainTextStripsBidiAndZeroWidth(t *testing.T) {
	// U+202E right-to-left override, U+2066 left-to-right isolate,
	// U+200B zero-width space, U+FEFF BOM.
	input := "good\u202Eevil\u2066hidden\u200B\uFEFFtext"
	got := SanitizePlainText(input)
	for _, bad := range []rune{0x202E, 0x2066, 0x200B, 0xFEFF} {
		if strings.ContainsRune(got, bad) {
			t.Fatalf("format control %U survived sanitization: %q", bad, got)
		}
	}
	if !strings.Contains(got, "good") || !strings.Contains(got, "text") {
		t.Fatalf("text content lost: %q", got)
	}
}
