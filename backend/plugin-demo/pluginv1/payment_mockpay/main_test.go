package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestEnsureCloudflaredBinary_NoAutoDownload covers M7: the plugin must not
// silently download and execute a remote cloudflared binary. When cloudflared
// is not on PATH it must fail fast with guidance, never reach out to the
// network.
func TestEnsureCloudflaredBinary_NoAutoDownload(t *testing.T) {
	old := cloudflaredLookPath
	cloudflaredLookPath = func(string) (string, error) { return "", fmt.Errorf("not found") }
	defer func() { cloudflaredLookPath = old }()

	if _, err := ensureCloudflaredBinary(); err == nil {
		t.Fatalf("expected an error when cloudflared is not installed")
	} else if !strings.Contains(err.Error(), "MOCKPAY_PUBLIC_BASE_URL") {
		t.Fatalf("error should direct the operator to MOCKPAY_PUBLIC_BASE_URL: %v", err)
	}
}

// TestEnsureCloudflaredBinary_UsesPATH verifies the explicit PATH install is
// still honored (cloudflared installed by the operator is fine).
func TestEnsureCloudflaredBinary_UsesPATH(t *testing.T) {
	old := cloudflaredLookPath
	cloudflaredLookPath = func(name string) (string, error) { return "/usr/local/bin/" + name, nil }
	defer func() { cloudflaredLookPath = old }()

	bin, err := ensureCloudflaredBinary()
	if err != nil {
		t.Fatalf("ensureCloudflaredBinary: %v", err)
	}
	if bin != "/usr/local/bin/cloudflared" {
		t.Fatalf("ensureCloudflaredBinary = %q", bin)
	}
}

// TestEnsurePublicBaseURL_UsesEnv verifies MOCKPAY_PUBLIC_BASE_URL is honored
// without touching the network.
func TestEnsurePublicBaseURL_UsesEnv(t *testing.T) {
	oldEnv := os.Getenv("MOCKPAY_PUBLIC_BASE_URL")
	defer os.Setenv("MOCKPAY_PUBLIC_BASE_URL", oldEnv)
	os.Setenv("MOCKPAY_PUBLIC_BASE_URL", "https://mock.example/")

	got, err := ensurePublicBaseURL("http://127.0.0.1:1234")
	if err != nil {
		t.Fatalf("ensurePublicBaseURL: %v", err)
	}
	if got != "https://mock.example" {
		t.Fatalf("ensurePublicBaseURL = %q, want https://mock.example", got)
	}

	// Invalid env value must error instead of downloading.
	os.Setenv("MOCKPAY_PUBLIC_BASE_URL", "javascript:alert(1)")
	if _, err := ensurePublicBaseURL("http://127.0.0.1:1234"); err == nil {
		t.Fatalf("expected error for invalid MOCKPAY_PUBLIC_BASE_URL")
	}
}
