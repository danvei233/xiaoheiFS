package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"testing"
	"xiaoheiplay/internal/domain"
)

func TestContainsDisallowedHTML(t *testing.T) {
	out := sanitizeHTML("<script>alert(1)</script><p>ok</p>")
	if strings.Contains(strings.ToLower(out), "<script") {
		t.Fatalf("expected script to be removed, got %q", out)
	}
	if !strings.Contains(out, "ok") {
		t.Fatalf("expected content preserved, got %q", out)
	}
}

func TestBuildUploadName(t *testing.T) {
	name := buildUploadName("image/png")
	if !strings.HasSuffix(name, ".png") {
		t.Fatalf("expected .png suffix")
	}
}

func TestBuildPermissionTree(t *testing.T) {
	perms := []domain.Permission{
		{Code: "root", Name: "Root", SortOrder: 2},
		{Code: "child-a", Name: "Child A", ParentCode: "root", SortOrder: 2},
		{Code: "child-b", Name: "Child B", ParentCode: "root", SortOrder: 1},
		{Code: "other", Name: "Other", SortOrder: 1},
	}
	tree := buildPermissionTree(perms)
	if len(tree) != 2 {
		t.Fatalf("expected 2 roots")
	}
	if tree[0].Code != "other" {
		t.Fatalf("expected sorted roots")
	}
	if len(tree[1].Children) != 2 {
		t.Fatalf("expected children")
	}
	if tree[1].Children[0].Code != "child-b" {
		t.Fatalf("expected sorted children")
	}
}

func TestClientErrMessage_MasksInternalErrors(t *testing.T) {
	// Known domain sentinels keep their canonical text.
	if got := clientErrMessage(domain.ErrInvalidInput); got != domain.ErrInvalidInput.Error() {
		t.Fatalf("sentinel must keep its text, got %q", got)
	}
	// Wrapped sentinels are still recognized and keep their (safe) full text.
	wrapped := fmt.Errorf("create group: %w", domain.ErrNotFound)
	if got := clientErrMessage(wrapped); got != wrapped.Error() {
		t.Fatalf("wrapped sentinel must keep its safe text, got %q", got)
	}
	// Raw repository/driver errors must not leak internal details (P2-11).
	raw := errors.New("SQLITE_BUSY: database is locked (file=/var/lib/mysql/users.db)")
	if got := clientErrMessage(raw); got == raw.Error() {
		t.Fatalf("internal error must be masked, got %q", got)
	}
	if got := clientErrMessage(raw); got != domain.ErrRequestFailed.Error() {
		t.Fatalf("internal error must map to the generic message, got %q", got)
	}
}

func TestVerifyHMACAndHelpers(t *testing.T) {
	body := []byte("hello")
	secret := "s3cret"
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	signature := fmt.Sprintf("%x", mac.Sum(nil))
	if !verifyHMAC(body, secret, signature) {
		t.Fatalf("expected valid hmac")
	}
	if verifyHMAC(body, secret, "bad") {
		t.Fatalf("expected invalid hmac")
	}
	if parseHostIDLocal("123") != 123 {
		t.Fatalf("unexpected host id")
	}
	if !isDigits("12345") || isDigits("12a") {
		t.Fatalf("unexpected isDigits result")
	}
}
