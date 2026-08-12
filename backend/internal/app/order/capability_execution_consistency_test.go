package order

import (
	"context"
	"testing"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type capabilityExecutionResolver struct {
	capability *appshared.PluginAutomationCapability
}

func (r capabilityExecutionResolver) ClientForGoodsType(context.Context, int64) (appshared.AutomationClient, error) {
	return nil, nil
}

func (r capabilityExecutionResolver) CapabilitiesForGoodsType(context.Context, int64) (*appshared.PluginAutomationCapability, error) {
	return r.capability, nil
}

func TestAutomationFeatureAllowed_UsesDisplayCapabilitySemantics(t *testing.T) {
	service := &OrderService{automation: capabilityExecutionResolver{
		capability: &appshared.PluginAutomationCapability{Features: []string{"refund"}},
	}}
	inst := domain.VPSInstance{GoodsTypeID: 4}

	allowed, err := service.automationFeatureAllowed(context.Background(), inst, "resize", true)
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Fatal("plugin manifest without resize must disable execution")
	}

	inst.AccessInfoJSON = `{"capabilities":{"automation":{"features":["resize"]}}}`
	allowed, err = service.automationFeatureAllowed(context.Background(), inst, "resize", true)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("explicit dynamic feature list must override the manifest baseline like the display path")
	}

	allowed, err = service.automationFeatureAllowed(context.Background(), inst, "resize", false)
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Fatal("policy disable must not be overridden by dynamic capability")
	}
}
