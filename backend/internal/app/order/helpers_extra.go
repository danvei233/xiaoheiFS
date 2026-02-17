package order

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

var ErrProvisioning = domain.ErrProvisioning

type RefundPolicy struct {
	FullHours          int
	ProrateHours       int
	NoRefundHours      int
	FullDays           int
	ProrateDays        int
	NoRefundDays       int
	Curve              []RefundCurvePoint
	RequireApproval    bool
	AutoRefundOnDelete bool
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
	if min == -1 || max == -1 {
		if value != 0 {
			return ErrInvalidInput
		}
		return nil
	}
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

func mergeSpecInfo(existing string, info AutomationHostInfo) string {
	spec := map[string]any{}
	if existing != "" {
		if err := json.Unmarshal([]byte(existing), &spec); err != nil {
			spec = map[string]any{}
		}
	}
	if info.CPU > 0 {
		spec["cpu"] = info.CPU
	}
	if info.MemoryGB > 0 {
		spec["memory_gb"] = info.MemoryGB
	}
	if info.DiskGB > 0 {
		spec["disk_gb"] = info.DiskGB
	}
	if info.Bandwidth > 0 {
		spec["bandwidth_mbps"] = info.Bandwidth
	}
	if len(spec) == 0 {
		return ""
	}
	return mustJSON(spec)
}

func parseJSON(raw string) map[string]any {
	if raw == "" {
		return map[string]any{}
	}
	var out map[string]any
	_ = json.Unmarshal([]byte(raw), &out)
	if out == nil {
		out = map[string]any{}
	}
	return out
}

func getInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		return n
	default:
		return 0
	}
}

func toString(id int64) string {
	return fmt.Sprintf("%d", id)
}

func hashKey(raw string) string {
	sum := sha256Sum(raw)
	return fmt.Sprintf("%x", sum)
}

func sha256Sum(raw string) [32]byte {
	return sha256.Sum256([]byte(raw))
}

func refundElapsedRatio(inst domain.VPSInstance, now time.Time) float64 {
	if inst.ExpireAt == nil || inst.CreatedAt.IsZero() {
		return 1
	}
	total := inst.ExpireAt.Sub(inst.CreatedAt)
	if total <= 0 {
		return 1
	}
	if now.Before(inst.CreatedAt) {
		return 0
	}
	if !inst.ExpireAt.After(now) {
		return 1
	}
	ratio := now.Sub(inst.CreatedAt).Seconds() / total.Seconds()
	if ratio < 0 {
		return 0
	}
	if ratio > 1 {
		return 1
	}
	return ratio
}

func refundPeriodHours(inst domain.VPSInstance) (float64, bool) {
	if inst.ExpireAt == nil || inst.CreatedAt.IsZero() {
		return 0, false
	}
	total := inst.ExpireAt.Sub(inst.CreatedAt).Hours()
	if total <= 0 {
		return 0, false
	}
	return total, true
}

func refundRatioThreshold(totalHours float64, hours int, days int) float64 {
	if totalHours <= 0 {
		return 0
	}
	thresholdHours := float64(hours)
	if thresholdHours <= 0 && days > 0 {
		thresholdHours = float64(days) * 24
	}
	if thresholdHours <= 0 {
		return 0
	}
	ratio := thresholdHours / totalHours
	if ratio < 0 {
		return 0
	}
	if ratio > 1 {
		return 1
	}
	return ratio
}

func RenderTemplate(content string, vars any, html bool) string {
	return appshared.RenderTemplate(content, vars, html)
}

func IsHTMLContent(content string) bool {
	return appshared.IsHTMLContent(content)
}

func roundCents(v float64) int64 {
	return int64(math.Round(v))
}
