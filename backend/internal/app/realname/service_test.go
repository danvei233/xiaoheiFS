package realname

import "testing"

func TestParsePendingReason_ExtendedPendingFaceFormat(t *testing.T) {
	provider, token, ok := parsePendingReason("pending_face:baidu:tok123:extra_payload")
	if !ok {
		t.Fatalf("expected parse success")
	}
	if provider != "baidu" {
		t.Fatalf("unexpected provider: %q", provider)
	}
	if token != "tok123" {
		t.Fatalf("unexpected token: %q", token)
	}
}

