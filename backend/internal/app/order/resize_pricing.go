package order

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/money"
)

type ResizePricingPolicy struct {
	ChargeMode     string
	RefundRatio    float64
	Rounding       string
	MinCharge      int64
	MinRefund      int64
	RefundToWallet bool
	RefundCurve    []RefundCurvePoint
}

type ResizeQuote = appshared.ResizeQuote

func (s *OrderService) quoteResize(ctx context.Context, inst domain.VPSInstance, spec *CartSpec, targetPackageID int64, resetAddons bool) (ResizeQuote, CartSpec, error) {
	if s.catalog == nil {
		return ResizeQuote{}, CartSpec{}, ErrInvalidInput
	}
	policy := s.resizePricingPolicy(ctx)
	currentPkg, err := s.catalog.GetPackage(ctx, inst.PackageID)
	if err != nil {
		return ResizeQuote{}, CartSpec{}, err
	}
	plan, err := s.catalog.GetPlanGroup(ctx, currentPkg.PlanGroupID)
	if err != nil {
		return ResizeQuote{}, CartSpec{}, err
	}

	currentSpec := parseCartSpecJSON(inst.SpecJSON)
	currentBase := currentPkg.Monthly
	currentUnitCore := plan.UnitCore
	currentUnitMem := plan.UnitMem
	currentUnitDisk := plan.UnitDisk
	currentUnitBW := plan.UnitBW
	if s.pricer != nil && inst.UserID > 0 {
		if pricing, _, err := s.pricer.ResolvePackagePricing(ctx, inst.UserID, currentPkg.ID); err == nil {
			currentBase = pricing.MonthlyPrice
			currentUnitCore = pricing.UnitCore
			currentUnitMem = pricing.UnitMem
			currentUnitDisk = pricing.UnitDisk
			currentUnitBW = pricing.UnitBW
		}
	}
	currentAddon := int64(currentSpec.AddCores)*currentUnitCore +
		int64(currentSpec.AddMemGB)*currentUnitMem +
		int64(currentSpec.AddDiskGB)*currentUnitDisk +
		int64(currentSpec.AddBWMbps)*currentUnitBW
	currentMonthly := currentBase + currentAddon

	quote := ResizeQuote{
		RefundToWallet: policy.RefundToWallet,
		CurrentMonthly: currentMonthly,
	}

	if currentSpec.DurationMonths == 0 && inst.OrderItemID > 0 && s.items != nil {
		if item, err := s.items.GetOrderItem(ctx, inst.OrderItemID); err == nil && item.DurationMonths > 0 {
			currentSpec.DurationMonths = item.DurationMonths
		}
	}

	targetPkg := currentPkg
	if targetPackageID > 0 {
		nextPkg, err := s.catalog.GetPackage(ctx, targetPackageID)
		if err != nil {
			return ResizeQuote{}, CartSpec{}, err
		}
		if nextPkg.PlanGroupID != currentPkg.PlanGroupID {
			return ResizeQuote{}, CartSpec{}, ErrInvalidInput
		}
		targetPkg = nextPkg
	}
	quote.TargetPackageID = targetPkg.ID

	targetSpec := currentSpec
	if resetAddons {
		targetSpec = CartSpec{}
	}
	if spec != nil {
		targetSpec = *spec
	}

	if err := normalizeCartSpec(&targetSpec); err != nil {
		return ResizeQuote{}, CartSpec{}, err
	}
	if err := validateAddonSpec(targetSpec, plan); err != nil {
		return ResizeQuote{}, CartSpec{}, err
	}

	if targetPkg.ID != currentPkg.ID && targetPkg.ProductID != 0 && targetPkg.ProductID == currentPkg.ProductID {
		return ResizeQuote{}, CartSpec{}, ErrResizeSamePlan
	}

	targetBase := targetPkg.Monthly
	targetUnitCore := plan.UnitCore
	targetUnitMem := plan.UnitMem
	targetUnitDisk := plan.UnitDisk
	targetUnitBW := plan.UnitBW
	if s.pricer != nil && inst.UserID > 0 {
		if pricing, _, err := s.pricer.ResolvePackagePricing(ctx, inst.UserID, targetPkg.ID); err == nil {
			targetBase = pricing.MonthlyPrice
			targetUnitCore = pricing.UnitCore
			targetUnitMem = pricing.UnitMem
			targetUnitDisk = pricing.UnitDisk
			targetUnitBW = pricing.UnitBW
		}
	}
	targetAddon := int64(targetSpec.AddCores)*targetUnitCore +
		int64(targetSpec.AddMemGB)*targetUnitMem +
		int64(targetSpec.AddDiskGB)*targetUnitDisk +
		int64(targetSpec.AddBWMbps)*targetUnitBW
	targetMonthly := targetBase + targetAddon
	quote.TargetCPU = targetPkg.Cores + targetSpec.AddCores
	quote.TargetMemGB = targetPkg.MemoryGB + targetSpec.AddMemGB
	quote.TargetDiskGB = targetPkg.DiskGB + targetSpec.AddDiskGB
	quote.TargetBWMbps = targetPkg.BandwidthMB + targetSpec.AddBWMbps
	quote.TargetMonthly = targetMonthly

	currentCPU := currentPkg.Cores + currentSpec.AddCores
	currentMem := currentPkg.MemoryGB + currentSpec.AddMemGB
	currentDisk := currentPkg.DiskGB + currentSpec.AddDiskGB
	currentBW := currentPkg.BandwidthMB + currentSpec.AddBWMbps
	quote.CurrentPackageID = currentPkg.ID
	quote.CurrentCPU = currentCPU
	quote.CurrentMemGB = currentMem
	quote.CurrentDiskGB = currentDisk
	quote.CurrentBWMbps = currentBW
	if quote.TargetDiskGB < currentDisk {
		return ResizeQuote{}, CartSpec{}, ErrInvalidInput
	}
	if currentCPU == quote.TargetCPU && currentMem == quote.TargetMemGB && currentDisk == quote.TargetDiskGB && currentBW == quote.TargetBWMbps {
		return ResizeQuote{}, CartSpec{}, ErrResizeSamePlan
	}

	if _, _, ok := currentPeriod(inst, time.Now()); !ok {
		// Pricing needs a resolvable billing period. Without one the proration
		// falls back to 0, silently turning the resize into a free upgrade.
		return ResizeQuote{}, CartSpec{}, ErrInvalidInput
	}
	charge := resizeProration(currentMonthly, targetMonthly, inst, time.Now(), policy.Rounding)
	if charge > 0 {
		quote.ChargeAmount = charge
		if policy.MinCharge > 0 && quote.ChargeAmount < policy.MinCharge {
			quote.ChargeAmount = policy.MinCharge
		}
	} else if charge < 0 {
		refund := int64(math.Round(float64(-charge) * policy.RefundRatio))
		if policy.MinRefund > 0 && refund > 0 && refund < policy.MinRefund {
			refund = 0
		}
		if refund <= 0 || !policy.RefundToWallet {
			// No automatic wallet refund is due (ratio 0, below min_refund,
			// or refund_to_wallet=false). Store a plain free resize instead of
			// a negative ChargeAmount paired with RefundAmount=0: the
			// approval-time refund fallback treats `RefundAmount<=0 &&
			// ChargeAmount<0` as "credit the full negative diff", which would
			// bypass the configured ratio/min_refund entirely.
			quote.RefundAmount = 0
			quote.ChargeAmount = 0
		} else {
			quote.ChargeAmount = charge
			quote.RefundAmount = refund
			quote.RefundToWallet = true
		}
	}
	return quote, targetSpec, nil
}

func (s *OrderService) resizePricingPolicy(ctx context.Context) ResizePricingPolicy {
	policy := ResizePricingPolicy{
		ChargeMode:     "remaining",
		RefundRatio:    1,
		Rounding:       "round",
		MinCharge:      0,
		MinRefund:      0,
		RefundToWallet: true,
	}
	if s.settings == nil {
		return policy
	}
	if v, ok := getSettingString(ctx, s.settings, "resize_price_mode"); ok {
		policy.ChargeMode = v
	}
	if v, ok := getSettingFloat(ctx, s.settings, "resize_refund_ratio"); ok {
		policy.RefundRatio = v
	}
	if v, ok := getSettingString(ctx, s.settings, "resize_rounding"); ok {
		policy.Rounding = v
	}
	if v, ok := getSettingCents(ctx, s.settings, "resize_min_charge"); ok {
		policy.MinCharge = v
	}
	if v, ok := getSettingCents(ctx, s.settings, "resize_min_refund"); ok {
		policy.MinRefund = v
	}
	if v, ok := getSettingBool(ctx, s.settings, "resize_refund_to_wallet"); ok {
		policy.RefundToWallet = v
	}
	if curve, ok := LoadRefundCurve(ctx, s.settings); ok {
		policy.RefundCurve = curve
	}
	if policy.RefundRatio < 0 {
		policy.RefundRatio = 0
	}
	if policy.RefundRatio > 1 {
		policy.RefundRatio = 1
	}
	return policy
}

func applyResizeCharge(policy ResizePricingPolicy, base int64, inst domain.VPSInstance, durationMonths int) int64 {
	value := money.ProrateCents(base, remainingNanos(inst, time.Now()), periodNanos(inst, time.Now()))
	return applyRounding(value, policy.Rounding)
}

func applyResizeRefund(policy ResizePricingPolicy, base int64, inst domain.VPSInstance, durationMonths int) int64 {
	value := money.ProrateCents(base, remainingNanos(inst, time.Now()), periodNanos(inst, time.Now()))
	return applyRounding(value, policy.Rounding)
}

func resizeProration(currentMonthly, targetMonthly int64, inst domain.VPSInstance, now time.Time, mode string) int64 {
	diffMonthly := targetMonthly - currentMonthly
	if diffMonthly == 0 {
		return 0
	}
	start, end, ok := currentPeriod(inst, now)
	if !ok {
		return 0
	}
	total := end.Sub(start).Nanoseconds()
	remaining := end.Sub(now).Nanoseconds()
	if total <= 0 || remaining <= 0 {
		return 0
	}
	// Monthly prices are denominated per calendar month, but the current
	// period spans the whole billing cycle (1..N months). Dividing by the
	// full cycle here would under-charge multi-month cycles by a factor of
	// the cycle length (e.g. diff*remaining/total where total=12mo). Scale
	// the denominator down to a single billing month so the result is
	// diff × remaining months.
	months := cycleMonths(start, end)
	perMonth := total / int64(months)
	if perMonth <= 0 {
		return 0
	}
	return prorateWithMode(diffMonthly, remaining, perMonth, mode)
}

// cycleMonths returns the number of billing months spanned by the current
// period, estimated from the period length with a 30-day month. The period is
// always whole calendar months in practice (set at purchase/renewal), so the
// estimate is exact for real data and degrades gracefully for legacy periods.
func cycleMonths(start, end time.Time) int {
	total := end.Sub(start)
	if total <= 0 {
		return 1
	}
	months := int(math.Round(total.Hours() / 720.0))
	if months < 1 {
		months = 1
	}
	return months
}

func remainingRatio(inst domain.VPSInstance, now time.Time) float64 {
	start, end, ok := currentPeriod(inst, now)
	if !ok {
		return 0
	}
	total := end.Sub(start)
	if total <= 0 {
		return 0
	}
	remaining := end.Sub(now)
	if remaining <= 0 {
		return 0
	}
	ratio := remaining.Seconds() / total.Seconds()
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return ratio
}

func currentPeriod(inst domain.VPSInstance, now time.Time) (time.Time, time.Time, bool) {
	return inst.CurrentBillingPeriod(now)
}

func applyRounding(value int64, mode string) int64 {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "ceil", "floor", "round", "":
		return value
	default:
		return value
	}
}

func addonPrice(plan domain.PlanGroup, spec CartSpec) int64 {
	return int64(spec.AddCores)*plan.UnitCore +
		int64(spec.AddMemGB)*plan.UnitMem +
		int64(spec.AddDiskGB)*plan.UnitDisk +
		int64(spec.AddBWMbps)*plan.UnitBW
}

func parseCartSpecJSON(specJSON string) CartSpec {
	if specJSON == "" {
		return CartSpec{}
	}
	var spec CartSpec
	_ = json.Unmarshal([]byte(specJSON), &spec)
	return spec
}

func getSettingFloat(ctx context.Context, repo SettingsRepository, key string) (float64, bool) {
	if repo == nil {
		return 0, false
	}
	setting, err := repo.GetSetting(ctx, key)
	if err != nil {
		return 0, false
	}
	val, err := strconv.ParseFloat(strings.TrimSpace(setting.ValueJSON), 64)
	if err != nil {
		return 0, false
	}
	if math.IsNaN(val) {
		// NaN is not a valid ratio: reject it so the caller falls back to its
		// default instead of propagating a value that breaks float→int64 casts.
		return 0, false
	}
	return val, true
}

func getSettingCents(ctx context.Context, repo SettingsRepository, key string) (int64, bool) {
	if repo == nil {
		return 0, false
	}
	setting, err := repo.GetSetting(ctx, key)
	if err != nil {
		return 0, false
	}
	val, err := money.ParseNumberStringToCents(strings.TrimSpace(setting.ValueJSON))
	if err != nil {
		return 0, false
	}
	return val, true
}

func prorateWithMode(cents int64, remainNanos int64, totalNanos int64, mode string) int64 {
	if totalNanos == 0 {
		return 0
	}
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "ceil", "floor":
		num := big.NewInt(cents)
		num.Mul(num, big.NewInt(remainNanos))
		den := big.NewInt(totalNanos)
		quot, rem := new(big.Int).QuoRem(num, den, new(big.Int))
		if rem.Sign() == 0 {
			return quot.Int64()
		}
		if strings.EqualFold(mode, "ceil") {
			if num.Sign() > 0 {
				quot.Add(quot, big.NewInt(1))
			}
			return quot.Int64()
		}
		if num.Sign() < 0 {
			quot.Sub(quot, big.NewInt(1))
		}
		return quot.Int64()
	default:
		return money.ProrateCents(cents, remainNanos, totalNanos)
	}
}

func remainingNanos(inst domain.VPSInstance, now time.Time) int64 {
	_, end, ok := currentPeriod(inst, now)
	if !ok {
		return 0
	}
	remaining := end.Sub(now)
	if remaining <= 0 {
		return 0
	}
	return remaining.Nanoseconds()
}

func periodNanos(inst domain.VPSInstance, now time.Time) int64 {
	start, end, ok := currentPeriod(inst, now)
	if !ok {
		return 0
	}
	total := end.Sub(start)
	if total <= 0 {
		return 0
	}
	return total.Nanoseconds()
}

func getSettingString(ctx context.Context, repo SettingsRepository, key string) (string, bool) {
	if repo == nil {
		return "", false
	}
	setting, err := repo.GetSetting(ctx, key)
	if err != nil {
		return "", false
	}
	val := strings.TrimSpace(setting.ValueJSON)
	if val == "" {
		return "", false
	}
	return val, true
}
