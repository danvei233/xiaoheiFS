package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
)

func TestContainsDisallowedHTML(t *testing.T) {
	if !containsDisallowedHTML("<script>alert(1)</script>") {
		t.Fatalf("expected disallowed html")
	}
	if containsDisallowedHTML("<p>ok</p>") {
		t.Fatalf("expected allowed html")
	}
}

func TestBuildUploadName(t *testing.T) {
	name := buildUploadName("image.png")
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
