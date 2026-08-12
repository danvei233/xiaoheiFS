package order

import (
	"context"
	"testing"
)

func TestLoadCapabilityPolicy(t *testing.T) {
	repo := &fakeSettingRepo{values: map[string]string{
		packageCapabilitiesSettingKey:   `{"100":{"resize_enabled":false}}`,
		goodsTypeCapabilitiesSettingKey: `{"10":{"refund_enabled":false},"20":{"resize_enabled":true}}`,
	}}

	got, err := loadCapabilityPolicy(context.Background(), nil, repo, 100, 10)
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if got.ResizeEnabled == nil || *got.ResizeEnabled {
		t.Fatalf("expected package resize_enabled=false to override goods type, got %+v", got)
	}
	if got.RefundEnabled == nil || *got.RefundEnabled {
		t.Fatalf("expected package to inherit goods-type refund_enabled=false, got %+v", got)
	}

	got, err = loadCapabilityPolicy(context.Background(), nil, repo, 200, 20)
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if got.ResizeEnabled == nil || !*got.ResizeEnabled {
		t.Fatalf("expected goods-type 20 resize_enabled=true, got %+v", got)
	}
	if got.RefundEnabled != nil {
		t.Fatalf("expected refund_enabled unset for goods-type 20, got %+v", got)
	}

	got, err = loadCapabilityPolicy(context.Background(), nil, repo, 999, 999)
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if got.ResizeEnabled != nil || got.RefundEnabled != nil {
		t.Fatalf("expected empty policy for unknown package/goods_type, got %+v", got)
	}
}
