package http

import (
	"net/http"
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
	// The handler now derives its request context from the HTTP request, so the
	// test must supply one.
	ctx.Request = &http.Request{}
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

func TestResolveVPSCapabilitiesFromSnapshot_PackageOverridesGoodsType(t *testing.T) {
	disabled := false
	enabled := true
	inst := domain.VPSInstance{
		ID: 9, PackageID: 30, GoodsTypeID: 4,
		AccessInfoJSON: `{"capabilities":{"automation":{"features":["resize","refund"]}}}`,
	}
	got := resolveVPSCapabilitiesFromSnapshot(
		inst,
		map[int64]domain.GoodsType{},
		nil,
		map[string]packageCapabilityPolicy{"30": {ResizeEnabled: &disabled}},
		map[string]packageCapabilityPolicy{"4": {ResizeEnabled: &enabled, RefundEnabled: &disabled}},
		nil,
		map[string]domain.Setting{"resize_enabled": {ValueJSON: "true"}, "refund_enabled": {ValueJSON: "true"}},
	)
	if got == nil || got.Automation == nil {
		t.Fatal("expected automation capability")
	}
	if featureAllowedByCapability(got.Automation, "resize", false) {
		t.Fatalf("package disable must override goods type enable: %+v", got.Automation)
	}
	if featureAllowedByCapability(got.Automation, "refund", false) {
		t.Fatalf("package nil must inherit goods type disable: %+v", got.Automation)
	}
	if got.Automation.NotSupportedReasons["resize"] != "套餐未开启升降配" {
		t.Fatalf("unexpected resize reason: %+v", got.Automation.NotSupportedReasons)
	}
}
