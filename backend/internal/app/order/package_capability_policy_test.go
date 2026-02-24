package order

import (
	"context"
	"testing"
)

func TestLoadPackageCapabilityPolicy(t *testing.T) {
	repo := &fakeSettingRepo{values: map[string]string{
		packageCapabilitiesSettingKey: `{"100":{"resize_enabled":false},"200":{"refund_enabled":true}}`,
	}}

	got := loadPackageCapabilityPolicy(context.Background(), repo, 100)
	if got.ResizeEnabled == nil || *got.ResizeEnabled {
		t.Fatalf("expected package 100 resize_enabled=false, got %+v", got)
	}
	if got.RefundEnabled != nil {
		t.Fatalf("expected package 100 refund_enabled=nil, got %+v", got)
	}

	got = loadPackageCapabilityPolicy(context.Background(), repo, 200)
	if got.RefundEnabled == nil || !*got.RefundEnabled {
		t.Fatalf("expected package 200 refund_enabled=true, got %+v", got)
	}

	got = loadPackageCapabilityPolicy(context.Background(), repo, 999)
	if got.ResizeEnabled != nil || got.RefundEnabled != nil {
		t.Fatalf("expected empty policy for unknown package, got %+v", got)
	}
}
