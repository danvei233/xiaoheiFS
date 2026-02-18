package automation

import (
	"fmt"
	"strings"
	"testing"
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
