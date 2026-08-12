package main

import (
	"context"
	"encoding/json"
	"testing"

	pluginv1 "xiaoheiplay/plugin/v1"
)

// TestGetConfigSchema_TimeoutMatchesHostLongOpBudget guards the P3 fix: the
// plugin's HTTP client default timeout must be at least the host long-op RPC
// budget (120s), and the config schema must allow configuring that value — a
// 12s default with a 60s cap would abort long create/provision calls the host
// is still waiting on.
func TestGetConfigSchema_TimeoutMatchesHostLongOpBudget(t *testing.T) {
	schema, err := (&coreServer{}).GetConfigSchema(context.Background(), &pluginv1.Empty{})
	if err != nil {
		t.Fatalf("get config schema: %v", err)
	}
	var parsed struct {
		Properties struct {
			TimeoutSec struct {
				Default int `json:"default"`
				Maximum int `json:"maximum"`
			} `json:"timeout_sec"`
		} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(schema.JsonSchema), &parsed); err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	if parsed.Properties.TimeoutSec.Default < 120 {
		t.Fatalf("timeout_sec default %d < host long-op budget 120s", parsed.Properties.TimeoutSec.Default)
	}
	if parsed.Properties.TimeoutSec.Maximum < 120 {
		t.Fatalf("timeout_sec maximum %d < host long-op budget 120s", parsed.Properties.TimeoutSec.Maximum)
	}
}
