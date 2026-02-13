package usecase

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

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

type ResizeQuote struct {
	ChargeAmount    int64
	RefundAmount    int64
	RefundToWallet  bool
	TargetPackageID int64
	TargetCPU       int
	TargetMemGB     int
	TargetDiskGB    int
	TargetBWMbps    int
	TargetMonthly   int64
	CurrentMonthly  int64
}

func (q ResizeQuote) ToPayload(vpsID int64, spec CartSpec) map[string]any {
	payload := map[string]any{
		"vps_id":            vpsID,
		"spec":              spec,
		"target_package_id": q.TargetPackageID,
		"target_cpu":        q.TargetCPU,
		"target_mem_gb":     q.TargetMemGB,
		"target_disk_gb":    q.TargetDiskGB,
		"target_bw_mbps":    q.TargetBWMbps,
		"charge_amount":     q.ChargeAmount,
		"refund_amount":     q.RefundAmount,
		"refund_to_wallet":  q.RefundToWallet,
	}
	return payload
}

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
	currentAddon := addonPrice(plan, currentSpec)
	currentMonthly := currentPkg.Monthly + currentAddon

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
		quote.TargetPackageID = nextPkg.ID
	}

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

	targetAddon := addonPrice(plan, targetSpec)
	targetMonthly := targetPkg.Monthly + targetAddon
	quote.TargetCPU = targetPkg.Cores + targetSpec.AddCores
	quote.TargetMemGB = targetPkg.MemoryGB + targetSpec.AddMemGB
	quote.TargetDiskGB = targetPkg.DiskGB + targetSpec.AddDiskGB
	quote.TargetBWMbps = targetPkg.BandwidthMB + targetSpec.AddBWMbps
	quote.TargetMonthly = targetMonthly

	currentCPU := currentPkg.Cores + currentSpec.AddCores
	currentMem := currentPkg.MemoryGB + currentSpec.AddMemGB
	currentDisk := currentPkg.DiskGB + currentSpec.AddDiskGB
	currentBW := currentPkg.BandwidthMB + currentSpec.AddBWMbps
	if quote.TargetDiskGB < currentDisk {
		return ResizeQuote{}, CartSpec{}, ErrInvalidInput
	}
	if currentCPU == quote.TargetCPU && currentMem == quote.TargetMemGB && currentDisk == quote.TargetDiskGB && currentBW == quote.TargetBWMbps {
		return ResizeQuote{}, CartSpec{}, ErrResizeSamePlan
	}

	charge := resizeProration(currentMonthly, targetMonthly, inst, time.Now(), policy.Rounding)
	if charge > 0 {
		quote.ChargeAmount = charge
		if policy.MinCharge > 0 && quote.ChargeAmount < policy.MinCharge {
			quote.ChargeAmount = policy.MinCharge
		}
	} else if charge < 0 {
		quote.ChargeAmount = charge
		refund := int64(math.Round(float64(-charge) * policy.RefundRatio))
		if policy.MinRefund > 0 && refund > 0 && refund < policy.MinRefund {
			refund = 0
		}
		quote.RefundAmount = refund
		if refund > 0 {
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
	return prorateWithMode(diffMonthly, remaining, total, mode)
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
	start, end, ok := currentPeriodFromSpec(inst.SpecJSON, inst.ExpireAt)
	if ok {
		return start, end, true
	}
	if inst.ExpireAt != nil && !inst.CreatedAt.IsZero() && inst.ExpireAt.After(now) {
		return inst.CreatedAt, *inst.ExpireAt, true
	}
	return time.Time{}, time.Time{}, false
}

func currentPeriodFromSpec(specJSON string, expireAt *time.Time) (time.Time, time.Time, bool) {
	if strings.TrimSpace(specJSON) == "" {
		return time.Time{}, time.Time{}, false
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(specJSON), &payload); err != nil {
		return time.Time{}, time.Time{}, false
	}
	startText := parsePeriodValue(payload, "current_period_start", "period_start", "last_renew_at", "purchase_at")
	endText := parsePeriodValue(payload, "current_period_end", "period_end")
	if endText == "" && expireAt != nil {
		endText = expireAt.UTC().Format(time.RFC3339)
	}
	if startText == "" || endText == "" {
		return time.Time{}, time.Time{}, false
	}
	start, err := time.Parse(time.RFC3339, startText)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	end, err := time.Parse(time.RFC3339, endText)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func parsePeriodValue(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if raw, ok := payload[key]; ok {
			switch val := raw.(type) {
			case string:
				return strings.TrimSpace(val)
			}
		}
	}
	return ""
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
