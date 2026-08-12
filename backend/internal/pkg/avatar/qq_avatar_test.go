package avatar

import (
	"strings"
	"testing"
)

func TestGetQQAvatarURL(t *testing.T) {
	got := GetQQAvatarURL("123", 0)
	if got == "" || got == GetQQAvatarURL("123", 50) {
		t.Fatalf("expected default size applied")
	}
	if def := GetQQAvatarURLDefault("123"); def == "" {
		t.Fatalf("expected default url")
	}
}

func TestGetQQAvatarURLEscapesQQ(t *testing.T) {
	got := GetQQAvatarURL("1&x=y", 100)
	if strings.Contains(got, "nk=1&x=y") {
		t.Fatalf("expected qq to be URL-escaped, got %q", got)
	}
	if !strings.Contains(got, "1%26x%3Dy") {
		t.Fatalf("expected encoded qq in nk param, got %q", got)
	}
}
