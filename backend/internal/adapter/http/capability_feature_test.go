package http

import "testing"

func TestFeatureAllowedByCapability(t *testing.T) {
	cap := &VPSAutomationCapabilityDTO{
		Features: []string{"upgrade", "firewall"},
	}
	if !featureAllowedByCapability(cap, "resize", false) {
		t.Fatalf("expected resize allowed by upgrade alias")
	}
	if featureAllowedByCapability(cap, "panel_login", false) {
		t.Fatalf("expected panel_login not allowed when feature missing")
	}
	if featureAllowedByCapability(cap, "refund", true) {
		t.Fatalf("expected refund not allowed when features set excludes refund")
	}
	if !featureAllowedByCapability(nil, "refund", true) {
		t.Fatalf("expected fallback when capability is nil")
	}
	cap.Features = append(cap.Features, "panel_login")
	if !featureAllowedByCapability(cap, "panel_login", false) {
		t.Fatalf("expected panel_login allowed when feature present")
	}
}
