package order

import (
	"context"
	"testing"
)

func TestLoadCapabilityPolicy(t *testing.T) {
	repo := &fakeSettingRepo{values: map[string]string{
		goodsTypeCapabilitiesSettingKey: `{"10":{"refund_enabled":false},"20":{"resize_enabled":true}}`,
	}}

	got := loadCapabilityPolicy(context.Background(), repo, 100, 10)
	if got.ResizeEnabled != nil {
		t.Fatalf("expected resize_enabled unset for goods-type 10, got %+v", got)
	}
	if got.RefundEnabled == nil || *got.RefundEnabled {
		t.Fatalf("expected goods-type 10 refund_enabled=false, got %+v", got)
	}

	got = loadCapabilityPolicy(context.Background(), repo, 200, 20)
	if got.ResizeEnabled == nil || !*got.ResizeEnabled {
		t.Fatalf("expected goods-type 20 resize_enabled=true, got %+v", got)
	}
	if got.RefundEnabled != nil {
		t.Fatalf("expected refund_enabled unset for goods-type 20, got %+v", got)
	}

	got = loadCapabilityPolicy(context.Background(), repo, 999, 999)
	if got.ResizeEnabled != nil || got.RefundEnabled != nil {
		t.Fatalf("expected empty policy for unknown package/goods_type, got %+v", got)
	}
}
