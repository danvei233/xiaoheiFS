package order

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
)

type fakeBillingRepo struct {
	cycles map[int64]domain.BillingCycle
}

func (f *fakeBillingRepo) ListBillingCycles(ctx context.Context) ([]domain.BillingCycle, error) {
	out := make([]domain.BillingCycle, 0, len(f.cycles))
	for _, c := range f.cycles {
		out = append(out, c)
	}
	return out, nil
}
func (f *fakeBillingRepo) GetBillingCycle(ctx context.Context, id int64) (domain.BillingCycle, error) {
	return f.cycles[id], nil
}
func (f *fakeBillingRepo) CreateBillingCycle(ctx context.Context, cycle *domain.BillingCycle) error { return nil }
func (f *fakeBillingRepo) UpdateBillingCycle(ctx context.Context, cycle domain.BillingCycle) error  { return nil }
func (f *fakeBillingRepo) DeleteBillingCycle(ctx context.Context, id int64) error                   { return nil }

// 跨层 golden vectors（cross-module residual "订单价格预览"）。
//
// 前端 `utils/pricing.ts` 的 priceBreakdownToCents 必须与这里逐项整数舍入
// 语义一致：
//
//	golden 1: base=0.05 元, addon core=0.05/mem=0.05, multiplier=0.5（小数倍率）
//	          base=round(5*0.5)=3, core=round(5*0.5)=3, mem=3 → total=9 分
//	          （浮点聚合 (0.05+0.05+0.05)*0.5=0.075 → 7.5 分，会错 1~2 分）
//	golden 2: base=9.99 元, core=0.50×2, disk=0.25×1, multiplier=3
//	          base=round(999*3)=2997, core=round(100*3)=300, disk=round(25*3)=75
//	          → total=3372 分
func TestPriceBreakdownGoldenVectors(t *testing.T) {
	svc := &OrderService{billing: &fakeBillingRepo{cycles: map[int64]domain.BillingCycle{
		1: {ID: 1, Multiplier: 0.5, Months: 1, Active: true},
		2: {ID: 2, Multiplier: 3, Months: 3, Active: true},
	}}}
	ctx := context.Background()

	plan1 := domain.PlanGroup{
		UnitCore: 5, UnitMem: 5,
		AddCoreMax: 16, AddMemMax: 16, AddDiskMax: 2000, AddBWMax: 1000,
		AddCoreStep: 1, AddMemStep: 1, AddDiskStep: 1, AddBWStep: 1,
	}
	// golden 1：小数倍率 + 多 addon
	total1, base1, core1, mem1, _, _, _, err := svc.priceBreakdownForPackage(ctx, 0, domain.Package{Monthly: 5}, plan1, CartSpec{AddCores: 1, AddMemGB: 1, BillingCycleID: 1, CycleQty: 1})
	if err != nil {
		t.Fatalf("golden1: %v", err)
	}
	if base1 != 3 || core1 != 3 || mem1 != 3 || total1 != 9 {
		t.Fatalf("golden1 mismatch: base=%d core=%d mem=%d total=%d, want 3/3/3/9", base1, core1, mem1, total1)
	}

	plan2 := domain.PlanGroup{
		UnitCore: 50, UnitMem: 0, UnitDisk: 25, UnitBW: 0,
		AddCoreMax: 16, AddMemMax: 16, AddDiskMax: 2000, AddBWMax: 1000,
		AddCoreStep: 1, AddMemStep: 1, AddDiskStep: 1, AddBWStep: 1,
	}
	// golden 2：整数倍率
	total2, base2, core2, _, disk2, _, _, err := svc.priceBreakdownForPackage(ctx, 0, domain.Package{Monthly: 999}, plan2, CartSpec{AddCores: 2, AddDiskGB: 1, BillingCycleID: 2, CycleQty: 1})
	if err != nil {
		t.Fatalf("golden2: %v", err)
	}
	if base2 != 2997 || core2 != 300 || disk2 != 75 || total2 != 3372 {
		t.Fatalf("golden2 mismatch: base=%d core=%d disk=%d total=%d, want 2997/300/75/3372", base2, core2, disk2, total2)
	}
}
