package order

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
)

type RefundCurvePoint struct {
	// Percent represents the elapsed percentage of the current billing period (0-100).
	// For backward compatibility, JSON may also provide this value in the "hours" field.
	Percent int     `json:"percent"`
	Ratio   float64 `json:"ratio"`
}

func (p *RefundCurvePoint) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	p.Percent = 0
	if v, ok := raw["percent"]; ok {
		if n, ok := asInt(v); ok {
			p.Percent = n
		}
	} else if v, ok := raw["hours"]; ok { // legacy key name
		if n, ok := asInt(v); ok {
			p.Percent = n
		}
	}

	p.Ratio = 0
	if v, ok := raw["ratio"]; ok {
		switch val := v.(type) {
		case float64:
			p.Ratio = val
		case int:
			p.Ratio = float64(val)
		case int64:
			p.Ratio = float64(val)
		}
	}
	return nil
}

func asInt(v any) (int, bool) {
	switch val := v.(type) {
	case float64:
		return int(val), true
	case int:
		return val, true
	case int64:
		return int(val), true
	case json.Number:
		i, err := val.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}

func LoadRefundCurve(ctx context.Context, repo SettingsRepository) ([]RefundCurvePoint, bool) {
	return LoadRefundCurveByKey(ctx, repo, "refund_curve_json")
}

func LoadRefundCurveByKey(ctx context.Context, repo SettingsRepository, key string) ([]RefundCurvePoint, bool) {
	raw, ok := getSettingString(ctx, repo, key)
	if !ok || strings.TrimSpace(raw) == "" {
		return nil, false
	}
	var points []RefundCurvePoint
	if err := json.Unmarshal([]byte(raw), &points); err != nil {
		return nil, false
	}
	points = NormalizeRefundCurve(points)
	if len(points) == 0 {
		return nil, false
	}
	return points, true
}

func NormalizeRefundCurve(points []RefundCurvePoint) []RefundCurvePoint {
	if len(points) == 0 {
		return nil
	}
	seen := make(map[int]float64)
	order := make([]int, 0, len(points))
	for _, point := range points {
		if point.Percent < 0 {
			continue
		}
		ratio := clamp01(point.Ratio)
		if _, ok := seen[point.Percent]; !ok {
			order = append(order, point.Percent)
		}
		seen[point.Percent] = ratio
	}
	if len(order) == 0 {
		return nil
	}
	sort.Ints(order)
	out := make([]RefundCurvePoint, 0, len(order))
	for _, percent := range order {
		out = append(out, RefundCurvePoint{Percent: percent, Ratio: seen[percent]})
	}
	return out
}

func RefundCurveRatio(points []RefundCurvePoint, elapsedPercent float64) (float64, bool) {
	if len(points) == 0 {
		return 0, false
	}
	if elapsedPercent < 0 {
		elapsedPercent = 0
	}
	if elapsedPercent <= float64(points[0].Percent) {
		return clamp01(points[0].Ratio), true
	}
	for i := 1; i < len(points); i++ {
		if elapsedPercent <= float64(points[i].Percent) {
			prev := points[i-1]
			next := points[i]
			span := float64(next.Percent - prev.Percent)
			if span <= 0 {
				return clamp01(next.Ratio), true
			}
			t := (elapsedPercent - float64(prev.Percent)) / span
			ratio := prev.Ratio + t*(next.Ratio-prev.Ratio)
			return clamp01(ratio), true
		}
	}
	return clamp01(points[len(points)-1].Ratio), true
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
