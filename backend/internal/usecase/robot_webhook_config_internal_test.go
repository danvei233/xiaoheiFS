package usecase

import "testing"

func TestRobotWebhookConfig(t *testing.T) {
	raw := `[{"name":"bot","url":"http://example","enabled":true,"events":["order.paid"]}]`
	cfgs := ParseRobotWebhookConfigs(raw)
	if len(cfgs) != 1 {
		t.Fatalf("expected parsed configs")
	}
	if !cfgs[0].MatchesEvent("order.paid") {
		t.Fatalf("expected event match")
	}
	if cfgs[0].MatchesEvent("order.refund") {
		t.Fatalf("expected no match")
	}
	empty := ParseRobotWebhookConfigs(" ")
	if empty != nil {
		t.Fatalf("expected nil for empty input")
	}
}
