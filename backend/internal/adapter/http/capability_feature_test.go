package http

import "testing"

func TestFeatureAllowedByCapability(t *testing.T) {
	cap := &VPSAutomationCapabilityDTO{
		Features: []string{"upgrade", "firewall"},
	}
	if !featureAllowedByCapability(cap, "resize", false) {
		t.Fatalf("expected resize allowed by upgrade alias")
	}
	if featureAllowedByCapability(cap, "refund", true) {
		t.Fatalf("expected refund not allowed when features set excludes refund")
	}
	if !featureAllowedByCapability(nil, "refund", true) {
		t.Fatalf("expected fallback when capability is nil")
	}
}
