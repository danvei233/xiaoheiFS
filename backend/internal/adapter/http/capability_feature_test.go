package http

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"xiaoheiplay/internal/domain"
)

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

func TestApplyPackageCapabilityPolicy_ResizeRefundRequirePluginFeature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	h := &Handler{}
	inst := domain.VPSInstance{ID: 23, GoodsTypeID: 1}

	withoutFeatures := &VPSAutomationCapabilityDTO{Features: []string{"lifecycle"}}
	got := h.applyPackageCapabilityPolicy(ctx, inst, withoutFeatures)
	if featureAllowedByCapability(got, "resize", false) {
		t.Fatalf("expected resize disabled when plugin feature missing")
	}
	if featureAllowedByCapability(got, "refund", false) {
		t.Fatalf("expected refund disabled when plugin feature missing")
	}
	if got.NotSupportedReasons["resize"] == "" {
		t.Fatalf("expected resize reason when plugin feature missing")
	}
	if got.NotSupportedReasons["refund"] == "" {
		t.Fatalf("expected refund reason when plugin feature missing")
	}

	withFeatures := &VPSAutomationCapabilityDTO{Features: []string{"resize", "refund"}}
	got = h.applyPackageCapabilityPolicy(ctx, inst, withFeatures)
	if !featureAllowedByCapability(got, "resize", false) {
		t.Fatalf("expected resize enabled when plugin feature present")
	}
	if !featureAllowedByCapability(got, "refund", false) {
		t.Fatalf("expected refund enabled when plugin feature present")
	}
}
