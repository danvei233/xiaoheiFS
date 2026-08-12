package goodstype_test

import (
	"context"
	"testing"

	appgoodstype "xiaoheiplay/internal/app/goodstype"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestGoodsTypeService_CreateGetUpdate(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appgoodstype.NewService(repo, nil)
	ctx := context.Background()

	gt := domain.GoodsType{
		Name:                 "vps",
		AutomationPluginID:   "plug",
		AutomationInstanceID: "inst",
	}
	if err := svc.Create(ctx, &gt); err != nil {
		t.Fatalf("create: %v", err)
	}
	if gt.ID <= 0 {
		t.Fatalf("expected id assigned")
	}

	gt.Name = "vps-2"
	if err := svc.Update(ctx, gt); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := svc.Get(ctx, gt.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "vps-2" {
		t.Fatalf("expected updated name, got %q", got.Name)
	}
}

func TestGoodsTypeService_UpdateNonExistentReturnsNotFound(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appgoodstype.NewService(repo, nil)
	ctx := context.Background()

	gt := domain.GoodsType{
		ID:                   9999,
		Name:                 "ghost",
		AutomationCategory:   "automation",
		AutomationPluginID:   "plug",
		AutomationInstanceID: "inst",
	}
	if err := svc.Update(ctx, gt); err != domain.ErrNotFound {
		t.Fatalf("expected not found for missing id, got %v", err)
	}
}
