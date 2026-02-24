package order

import (
	"encoding/json"
	"strings"

	"xiaoheiplay/internal/domain"
)

func isResizeAllowed(inst domain.VPSInstance, fallback bool) bool {
	return isAutomationFeatureAllowed(inst, "resize", fallback)
}

func isRefundAllowed(inst domain.VPSInstance, fallback bool) bool {
	return isAutomationFeatureAllowed(inst, "refund", fallback)
}

func isAutomationFeatureAllowed(inst domain.VPSInstance, feature string, fallback bool) bool {
	raw := strings.TrimSpace(inst.AccessInfoJSON)
	if raw == "" {
		return fallback
	}
	var envelope struct {
		Capabilities struct {
			Automation json.RawMessage `json:"automation"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil || len(envelope.Capabilities.Automation) == 0 {
		return fallback
	}
	var payloadMap map[string]json.RawMessage
	if err := json.Unmarshal(envelope.Capabilities.Automation, &payloadMap); err != nil {
		return fallback
	}
	var payload struct {
		Features         []string `json:"features"`
		AddFeatures      []string `json:"add_features"`
		RemoveFeatures   []string `json:"remove_features"`
		DisabledFeatures []string `json:"disabled_features"`
		DenyFeatures     []string `json:"deny_features"`
	}
	if err := json.Unmarshal(envelope.Capabilities.Automation, &payload); err != nil {
		return fallback
	}
	_, hasFeatures := payloadMap["features"]
	allowed := fallback
	if hasFeatures {
		allowed = featureSliceContains(payload.Features, feature)
	}
	if featureSliceContains(payload.AddFeatures, feature) {
		allowed = true
	}
	if featureSliceContains(payload.RemoveFeatures, feature) ||
		featureSliceContains(payload.DisabledFeatures, feature) ||
		featureSliceContains(payload.DenyFeatures, feature) {
		allowed = false
	}
	return allowed
}

func featureSliceContains(items []string, feature string) bool {
	canonical := normalizeFeatureKey(feature)
	if canonical == "" {
		return false
	}
	for _, item := range items {
		if normalizeFeatureKey(item) == canonical {
			return true
		}
	}
	return false
}

func normalizeFeatureKey(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "upgrade", "downgrade":
		return "resize"
	case "refund_request":
		return "refund"
	default:
		return v
	}
}
