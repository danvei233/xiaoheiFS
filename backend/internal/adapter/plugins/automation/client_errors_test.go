package automation

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	appshared "xiaoheiplay/internal/app/shared"
)

func TestMapRPCBusinessError_PreservesHTTPTrace(t *testing.T) {
	raw := `rpc error: code = Unknown desc = {"msg":"upstream failed"} | http_trace={"request":{"url":"http://upstream/index.php/api/cloud/mirror_image"}}`
	err := fmt.Errorf("%s", raw)

	got := mapRPCBusinessError(err)
	if got == nil {
		t.Fatalf("expected error")
	}
	if got.Error() != raw {
		t.Fatalf("trace should be preserved, got: %s", got.Error())
	}
	if !strings.Contains(got.Error(), "http_trace=") {
		t.Fatalf("missing trace marker: %s", got.Error())
	}
}

func TestMapRPCBusinessError_ExtractsMessageWithoutTrace(t *testing.T) {
	err := fmt.Errorf(`rpc error: code = Unknown desc = {"msg":"line_id 错误"}`)

	got := mapRPCBusinessError(err)
	if got == nil {
		t.Fatalf("expected error")
	}
	if got.Error() != "line_id 错误" {
		t.Fatalf("unexpected mapped message: %s", got.Error())
	}
}

func TestMapRPCBusinessError_PreservesErrNotSupportedWithJSONMessage(t *testing.T) {
	// mapUnimplemented wraps Unimplemented into ErrNotSupported first; mapping
	// must not flatten the wrapper away when the message contains a JSON blob.
	raw := status.Error(codes.Unimplemented, `{"msg":"endpoint moved"}`)

	got := mapRPCBusinessError(mapUnimplemented(raw))
	if got == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(got, appshared.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported to be preserved, got: %v", got)
	}
}

// TestSanitizeReturnError_StripsHTTPTraceAndScrubsDetails guards the exec_race
// F3 path: the error returned to callers (and from there into user-visible
// HTTP/SSE responses) must not carry the full http_trace= base64 payload or
// upstream URLs/IPs/paths/tokens, even though logRPC keeps them for the audit
// log.
func TestSanitizeReturnError_StripsHTTPTraceAndScrubsDetails(t *testing.T) {
	err := fmt.Errorf("upstream call failed http_trace=%s", base64.StdEncoding.EncodeToString([]byte(`{"request":{"url":"https://10.0.0.5:8443/internal/api?token=LEAKYTOKEN123456789"}}`)))

	got := sanitizeReturnError(err)
	if got == nil {
		t.Fatalf("expected error")
	}
	msg := got.Error()
	if strings.Contains(msg, "http_trace=") {
		t.Fatalf("http_trace leaked to caller: %s", msg)
	}
	if strings.Contains(msg, "10.0.0.5") || strings.Contains(msg, "8443") {
		t.Fatalf("internal IP/port leaked to caller: %s", msg)
	}
	if strings.Contains(msg, "LEAKYTOKEN123456789") {
		t.Fatalf("token leaked to caller: %s", msg)
	}
	if !strings.Contains(msg, "upstream call failed") {
		t.Fatalf("expected the readable hint to survive, got: %s", msg)
	}
}

// TestSanitizeReturnError_PreservesSentinel ensures scrubbing an
// ErrInvalidInput-wrapped error keeps the sentinel chain so downstream
// errors.Is checks (e.g. createHostFailureIsDefinitive) still match.
func TestSanitizeReturnError_PreservesSentinel(t *testing.T) {
	err := status.Error(codes.InvalidArgument, `{"msg":"bad cpu"}`)
	mapped := mapRPCBusinessError(err)

	got := sanitizeReturnError(mapped)
	if got == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(got, appshared.ErrInvalidInput) {
		t.Fatalf("ErrInvalidInput sentinel lost after sanitization: %v", got)
	}
}

// TestSanitizeReturnError_NoScrubReturnsSameInstance confirms a clean error is
// returned unchanged (no unnecessary re-wrap).
func TestSanitizeReturnError_NoScrubReturnsSameInstance(t *testing.T) {
	err := fmt.Errorf("line_id invalid")
	if got := sanitizeReturnError(err); got != err {
		t.Fatalf("expected the same error instance for already-clean text, got %v", got)
	}
}

func TestMapRPCBusinessError_MapsInvalidArgumentToErrInvalidInput(t *testing.T) {
	raw := status.Error(codes.InvalidArgument, `{"msg":"bad cpu"}`)

	got := mapRPCBusinessError(raw)
	if got == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(got, appshared.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", got)
	}
	if !strings.Contains(got.Error(), "bad cpu") {
		t.Fatalf("expected extracted message preserved, got: %v", got.Error())
	}

	plain := status.Error(codes.InvalidArgument, "bad cpu")
	gotPlain := mapRPCBusinessError(plain)
	if gotPlain == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(gotPlain, appshared.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for plain message, got: %v", gotPlain)
	}
}
