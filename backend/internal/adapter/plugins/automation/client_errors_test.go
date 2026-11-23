package automation

import (
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

func TestMapRPCBusinessError_KeepsTransientCodesRetryable(t *testing.T) {
	for _, code := range []codes.Code{codes.DeadlineExceeded, codes.Unavailable} {
		err := status.Error(code, "context deadline exceeded")
		got := mapRPCBusinessError(err)
		if !errors.Is(got, appshared.ErrAutomationTransient) {
			t.Fatalf("code %s: expected transient sentinel, got %v", code, got)
		}
		if !strings.Contains(got.Error(), code.String()) {
			t.Fatalf("code %s: original error text lost: %v", code, got)
		}
	}
}

func TestMapRPCBusinessError_DoesNotMarkBusinessErrorTransient(t *testing.T) {
	err := status.Error(codes.FailedPrecondition, "镜像文件不存在")
	got := mapRPCBusinessError(err)
	if errors.Is(got, appshared.ErrAutomationTransient) {
		t.Fatalf("business error must not be marked transient: %v", got)
	}
}
