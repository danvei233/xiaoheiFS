package usecase

import (
	"context"
	"math"

	"xiaoheiplay/internal/domain"
)

type CartService struct {
	cart    CartRepository
	catalog CatalogRepository
	billing BillingCycleRepository
}

type CartSpec struct {
	AddCores       int   `json:"add_cores"`
	AddMemGB       int   `json:"add_mem_gb"`
	AddDiskGB      int   `json:"add_disk_gb"`
	AddBWMbps      int   `json:"add_bw_mbps"`
	BillingCycleID int64 `json:"billing_cycle_id"`
	CycleQty       int   `json:"cycle_qty"`
	DurationMonths int   `json:"duration_months"`
}

func NewCartService(cart CartRepository, catalog CatalogRepository, billing BillingCycleRepository) *CartService {
	return &CartService{cart: cart, catalog: catalog, billing: billing}
}

func (s *CartService) List(ctx context.Context, userID int64) ([]domain.CartItem, error) {
	return s.cart.ListCartItems(ctx, userID)
}

func (s *CartService) Add(ctx context.Context, userID int64, packageID int64, systemID int64, spec CartSpec, qty int) (domain.CartItem, error) {
	if qty <= 0 {
		qty = 1
	}
	if err := normalizeCartSpec(&spec); err != nil {
		return domain.CartItem{}, err
	}
	pkg, err := s.catalog.GetPackage(ctx, packageID)
	if err != nil {
		return domain.CartItem{}, err
	}
	plan, err := s.catalog.GetPlanGroup(ctx, pkg.PlanGroupID)
	if err != nil {
		return domain.CartItem{}, err
	}
	if err := validateAddonSpec(spec, plan); err != nil {
		return domain.CartItem{}, err
	}
	months, multiplier, err := s.resolveBilling(ctx, spec)
	if err != nil {
		return domain.CartItem{}, err
	}
	spec.DurationMonths = months
	baseMonthly := pkg.Monthly
	addonMonthly := int64(spec.AddCores)*plan.UnitCore + int64(spec.AddMemGB)*plan.UnitMem + int64(spec.AddDiskGB)*plan.UnitDisk + int64(spec.AddBWMbps)*plan.UnitBW
	unitAmount := int64(math.Round(float64(baseMonthly+addonMonthly) * multiplier))
	specJSON := mustJSON(spec)
	item := domain.CartItem{
		UserID:    userID,
		PackageID: packageID,
		SystemID:  systemID,
		SpecJSON:  specJSON,
		Qty:       qty,
		Amount:    unitAmount * int64(qty),
	}
	if err := s.cart.AddCartItem(ctx, &item); err != nil {
		return domain.CartItem{}, err
	}
	return item, nil
}

func (s *CartService) Update(ctx context.Context, userID int64, itemID int64, spec CartSpec, qty int) (domain.CartItem, error) {
	if qty <= 0 {
		qty = 1
	}
	if err := normalizeCartSpec(&spec); err != nil {
		return domain.CartItem{}, err
	}
	items, err := s.cart.ListCartItems(ctx, userID)
	if err != nil {
		return domain.CartItem{}, err
	}
	var target domain.CartItem
	for _, it := range items {
		if it.ID == itemID {
			target = it
			break
		}
	}
	if target.ID == 0 {
		return domain.CartItem{}, ErrNotFound
	}
	pkg, err := s.catalog.GetPackage(ctx, target.PackageID)
	if err != nil {
		return domain.CartItem{}, err
	}
	plan, err := s.catalog.GetPlanGroup(ctx, pkg.PlanGroupID)
	if err != nil {
		return domain.CartItem{}, err
	}
	if err := validateAddonSpec(spec, plan); err != nil {
		return domain.CartItem{}, err
	}
	months, multiplier, err := s.resolveBilling(ctx, spec)
	if err != nil {
		return domain.CartItem{}, err
	}
	spec.DurationMonths = months
	baseMonthly := pkg.Monthly
	addonMonthly := int64(spec.AddCores)*plan.UnitCore + int64(spec.AddMemGB)*plan.UnitMem + int64(spec.AddDiskGB)*plan.UnitDisk + int64(spec.AddBWMbps)*plan.UnitBW
	unitAmount := int64(math.Round(float64(baseMonthly+addonMonthly) * multiplier))
	updated := domain.CartItem{
		ID:        itemID,
		UserID:    userID,
		PackageID: target.PackageID,
		SystemID:  target.SystemID,
		SpecJSON:  mustJSON(spec),
		Qty:       qty,
		Amount:    unitAmount * int64(qty),
	}
	if err := s.cart.UpdateCartItem(ctx, updated); err != nil {
		return domain.CartItem{}, err
	}
	return updated, nil
}

func (s *CartService) Remove(ctx context.Context, userID int64, itemID int64) error {
	return s.cart.DeleteCartItem(ctx, itemID, userID)
}

func (s *CartService) Clear(ctx context.Context, userID int64) error {
	return s.cart.ClearCart(ctx, userID)
}

func (s *CartService) resolveBilling(ctx context.Context, spec CartSpec) (int, float64, error) {
	months := 1
	multiplier := 1.0
	if spec.BillingCycleID == 0 || s.billing == nil {
		return months, multiplier, nil
	}
	cycle, err := s.billing.GetBillingCycle(ctx, spec.BillingCycleID)
	if err != nil {
		return 0, 0, err
	}
	if !cycle.Active {
		return 0, 0, ErrInvalidInput
	}
	qty := spec.CycleQty
	if qty <= 0 {
		qty = 1
	}
	if cycle.MinQty > 0 && qty < cycle.MinQty {
		return 0, 0, ErrInvalidInput
	}
	if cycle.MaxQty > 0 && qty > cycle.MaxQty {
		return 0, 0, ErrInvalidInput
	}
	months = cycle.Months * qty
	multiplier = cycle.Multiplier * float64(qty)
	return months, multiplier, nil
}

func validateAddonSpec(spec CartSpec, plan domain.PlanGroup) error {
	if err := validateAddonValue(spec.AddCores, plan.AddCoreMin, plan.AddCoreMax, plan.AddCoreStep); err != nil {
		return err
	}
	if err := validateAddonValue(spec.AddMemGB, plan.AddMemMin, plan.AddMemMax, plan.AddMemStep); err != nil {
		return err
	}
	if err := validateAddonValue(spec.AddDiskGB, plan.AddDiskMin, plan.AddDiskMax, plan.AddDiskStep); err != nil {
		return err
	}
	if err := validateAddonValue(spec.AddBWMbps, plan.AddBWMin, plan.AddBWMax, plan.AddBWStep); err != nil {
		return err
	}
	return nil
}

func validateAddonValue(value, min, max, step int) error {
	if value == 0 {
		return nil
	}
	if min > 0 && value < min {
		return ErrInvalidInput
	}
	if max > 0 && value > max {
		return ErrInvalidInput
	}
	if step <= 0 {
		step = 1
	}
	if value%step != 0 {
		return ErrInvalidInput
	}
	return nil
}

func normalizeCartSpec(spec *CartSpec) error {
	if spec.AddCores < 0 || spec.AddMemGB < 0 || spec.AddDiskGB < 0 || spec.AddBWMbps < 0 {
		return ErrInvalidInput
	}
	if spec.CycleQty < 0 {
		return ErrInvalidInput
	}
	return nil
}
