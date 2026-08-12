package automation

import (
	"strings"
	"testing"

	pluginv1 "xiaoheiplay/plugin/v1"
)

func TestEnsureOpOK_RejectsSilentFailure(t *testing.T) {
	if err := ensureOpOK(&pluginv1.OperationResult{Ok: true}); err != nil {
		t.Fatalf("expected success for Ok=true, got: %v", err)
	}
	// A modern plugin that explicitly reports failure without error details
	// must not be treated as success: destructive ops (destroy/delete/resize/
	// renew) would otherwise be silently dropped.
	if err := ensureOpOK(&pluginv1.OperationResult{Ok: false}); err == nil {
		t.Fatalf("expected error for Ok=false without error details")
	}
	if err := ensureOpOK(&pluginv1.OperationResult{}); err == nil {
		t.Fatalf("expected error for all-zero OperationResult")
	}
	err := ensureOpOK(&pluginv1.OperationResult{Ok: false, ErrorCode: "E1", ErrorMessage: "boom"})
	if err == nil {
		t.Fatalf("expected error for detailed failure")
	}
	if !strings.Contains(err.Error(), "boom") || !strings.Contains(err.Error(), "E1") {
		t.Fatalf("expected error to carry message and code, got: %v", err)
	}
}

func TestEnsureOpOK_EmptyLegacySuccess(t *testing.T) {
	// The Empty-based response path (legacy plugins) keeps treating a success
	// status as success.
	if err := ensureOpOK(&pluginv1.Empty{}); err != nil {
		t.Fatalf("expected success for empty Empty, got: %v", err)
	}
	if err := ensureOpOK(&pluginv1.Empty{Status: "ok"}); err != nil {
		t.Fatalf("expected success for Empty status=ok, got: %v", err)
	}
	if err := ensureOpOK(&pluginv1.Empty{Status: "failed", Msg: "nope"}); err == nil {
		t.Fatalf("expected error for Empty failed status")
	}
}
