package plugins

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"xiaoheiplay/internal/domain"
)

func TestMapRPCErrorNil(t *testing.T) {
	if err := MapRPCError(nil, "sms plugin"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestMapRPCErrorUnavailableCollapsesDetail(t *testing.T) {
	st := status.New(codes.Unavailable, "dial tcp 169.254.169.254:80: connect: connection refused")
	err := MapRPCError(st.Err(), "sms plugin")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.Contains(msg, "169.254.169.254") {
		t.Fatalf("unavailable error leaked internal address: %q", msg)
	}
	if !strings.Contains(msg, "unavailable") {
		t.Fatalf("expected unavailable classification, got %q", msg)
	}
}

func TestMapRPCErrorInvalidArgumentRedactsSecret(t *testing.T) {
	st := status.New(codes.InvalidArgument, "api key fixture_secret_token_1234567890 invalid")
	err := MapRPCError(st.Err(), "sms plugin")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput wrap, got %T", err)
	}
	if strings.Contains(err.Error(), "fixture_secret_token_1234567890") {
		t.Fatalf("invalid-argument error leaked secret token: %q", err.Error())
	}
}

func TestMapRPCErrorFailedPreconditionRedactsURL(t *testing.T) {
	st := status.New(codes.FailedPrecondition, "webhook https://internal.example.com:8443/hook failed")
	err := MapRPCError(st.Err(), "payment plugin")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "internal.example.com") {
		t.Fatalf("failed-precondition error leaked internal URL: %q", err.Error())
	}
}

func TestMapRPCErrorDefaultRedactsPath(t *testing.T) {
	st := status.New(codes.Internal, "read /var/lib/xiaohei/secret.key: permission denied")
	err := MapRPCError(st.Err(), "kyc plugin")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "/var/lib/") {
		t.Fatalf("default error leaked absolute path: %q", err.Error())
	}
}

func TestSanitizePluginMessagePreservesPlainHint(t *testing.T) {
	got := SanitizePluginMessage("template not found")
	if got != "template not found" {
		t.Fatalf("plain message should pass through unchanged, got %q", got)
	}
}
