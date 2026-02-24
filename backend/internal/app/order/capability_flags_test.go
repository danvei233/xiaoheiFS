package order

import (
	"testing"

	"xiaoheiplay/internal/domain"
)

func TestIsAutomationFeatureAllowed(t *testing.T) {
	inst := domain.VPSInstance{
		AccessInfoJSON: `{
			"capabilities": {
				"automation": {
					"features": ["upgrade", "refund_request"],
					"disabled_features": ["refund"]
				}
			}
		}`,
	}
	if !isResizeAllowed(inst, false) {
		t.Fatalf("expected resize to be allowed by upgrade alias")
	}
	if isRefundAllowed(inst, true) {
		t.Fatalf("expected refund to be disabled by disabled_features")
	}
}

func TestIsAutomationFeatureAllowedFallback(t *testing.T) {
	inst := domain.VPSInstance{AccessInfoJSON: ""}
	if !isResizeAllowed(inst, true) {
		t.Fatalf("expected fallback true when capabilities missing")
	}
	if isRefundAllowed(inst, false) {
		t.Fatalf("expected fallback false when capabilities missing")
	}
}
