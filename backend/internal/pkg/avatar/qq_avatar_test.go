package avatar

import "testing"

func TestGetQQAvatarURL(t *testing.T) {
	got := GetQQAvatarURL("123", 0)
	if got == "" || got == GetQQAvatarURL("123", 50) {
		t.Fatalf("expected default size applied")
	}
	if def := GetQQAvatarURLDefault("123"); def == "" {
		t.Fatalf("expected default url")
	}
}
