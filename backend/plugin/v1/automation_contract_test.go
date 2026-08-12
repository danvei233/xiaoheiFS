package pluginv1_test

import (
	"os"
	"regexp"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	pluginv1 "xiaoheiplay/plugin/v1"
)

// TestAutomationServiceDescMatchesProtoContract guards against .proto edits that
// are not regenerated into the *_pb.go files (second-round audit finding H-1:
// automation.proto declared UpsertPackage/DeletePackage that were missing from
// the generated ServiceDesc, so the catalog write-back RPCs did not exist on
// the wire at all). Every `rpc` declared in automation.proto must be present in
// the ServiceDesc that RegisterAutomationServiceServer actually registers.
func TestAutomationServiceDescMatchesProtoContract(t *testing.T) {
	src, err := os.ReadFile("automation.proto")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^\s*rpc\s+(\w+)\s*\(`)
	matches := re.FindAllSubmatch(src, -1)
	if len(matches) == 0 {
		t.Fatal("no rpc declarations found in automation.proto")
	}
	want := map[string]bool{}
	for _, m := range matches {
		want[string(m[1])] = true
	}

	got := map[string]bool{}
	for _, m := range pluginv1.AutomationService_ServiceDesc.Methods {
		got[m.MethodName] = true
	}
	if len(got) != len(want) {
		t.Errorf("AutomationService_ServiceDesc has %d methods, want %d", len(got), len(want))
	}
	for name := range want {
		if !got[name] {
			t.Errorf("AutomationService_ServiceDesc missing RPC %q declared in automation.proto", name)
		}
	}
}

// TestAutomationWriteBackMessagesRegistered verifies the message types that back
// the catalog write-back RPCs are compiled into the descriptor registry.
func TestAutomationWriteBackMessagesRegistered(t *testing.T) {
	for _, name := range []string{
		"plugin.v1.UpsertPackageRequest",
		"plugin.v1.DeletePackageRequest",
	} {
		if _, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(name)); err != nil {
			t.Errorf("message %s not registered: %v", name, err)
		}
	}
}

// TestUpsertPackageRequestWireEncoding locks the field numbers/types of
// UpsertPackageRequest (automation.proto:126-135):
// line_id=1 int64, package_name=2 string, cpu=3 int32, ..., gpu_mem_gb=8 int32.
func TestUpsertPackageRequestWireEncoding(t *testing.T) {
	req := &pluginv1.UpsertPackageRequest{LineId: 7, PackageName: "plan-1", Cpu: 2}
	b, err := proto.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x08, 0x07, 0x12, 0x06, 'p', 'l', 'a', 'n', '-', '1', 0x18, 0x02}
	if string(b) != string(want) {
		t.Fatalf("wire bytes = %x, want %x", b, want)
	}
	var got pluginv1.UpsertPackageRequest
	if err := proto.Unmarshal(want, &got); err != nil {
		t.Fatal(err)
	}
	if got.GetLineId() != 7 || got.GetPackageName() != "plan-1" || got.GetCpu() != 2 {
		t.Fatalf("round trip mismatch: %+v", &got)
	}
}

// TestDeletePackageRequestWireEncoding locks the field numbers/types of
// DeletePackageRequest (automation.proto:138-141):
// line_id=1 int64, package_name=2 string.
func TestDeletePackageRequestWireEncoding(t *testing.T) {
	req := &pluginv1.DeletePackageRequest{LineId: 3, PackageName: "x"}
	b, err := proto.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var got pluginv1.DeletePackageRequest
	if err := proto.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.GetLineId() != 3 || got.GetPackageName() != "x" {
		t.Fatalf("round trip mismatch: %+v", &got)
	}
}
