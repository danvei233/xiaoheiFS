package plugins

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/cryptox"
)

func TestMissingRequiredConfigFields(t *testing.T) {
	schema := `{
		"type":"object",
		"properties":{
			"base_url":{"type":"string"},
			"api_key":{"type":"string"}
		},
		"required":["base_url","api_key"]
	}`

	got := missingRequiredConfigFields(schema, `{}`)
	want := []string{"api_key", "base_url"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("missing fields mismatch: got=%v want=%v", got, want)
	}

	got = missingRequiredConfigFields(schema, `{"base_url":"https://example.com","api_key":""}`)
	want = []string{"api_key"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("missing fields mismatch after partial config: got=%v want=%v", got, want)
	}
}

func TestParseMissingFieldsFromError(t *testing.T) {
	got := parseMissingFieldsFromError("base_url/api_key required")
	want := []string{"api_key", "base_url"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parse missing fields mismatch: got=%v want=%v", got, want)
	}
}

func TestAsConfigValidationError(t *testing.T) {
	err := &ConfigValidationError{Code: "missing_required_config", Message: "base_url required"}
	wrapped := fmt.Errorf("%s", "wrap: "+err.Error())
	if _, ok := AsConfigValidationError(wrapped); ok {
		t.Fatal("expected false for non-wrapped error type")
	}
	wrappedTyped := fmt.Errorf("wrapped: %w", err)
	if cfgErr, ok := AsConfigValidationError(wrappedTyped); !ok || cfgErr == nil || cfgErr.Code != "missing_required_config" {
		t.Fatalf("expected wrapped config validation error, got ok=%v err=%v", ok, cfgErr)
	}
	if cfgErr, ok := AsConfigValidationError(err); !ok || cfgErr == nil || cfgErr.Code != "missing_required_config" {
		t.Fatalf("expected config validation error, got ok=%v err=%v", ok, cfgErr)
	}
}

type fakePluginInstallationRepo struct {
	inst domain.PluginInstallation
}

func (f *fakePluginInstallationRepo) UpsertPluginInstallation(_ context.Context, inst *domain.PluginInstallation) error {
	if inst == nil {
		return fmt.Errorf("nil installation")
	}
	f.inst = *inst
	return nil
}

func (f *fakePluginInstallationRepo) GetPluginInstallation(_ context.Context, category, pluginID, instanceID string) (domain.PluginInstallation, error) {
	if f.inst.Category == category && f.inst.PluginID == pluginID && f.inst.InstanceID == instanceID {
		return f.inst, nil
	}
	return domain.PluginInstallation{}, fmt.Errorf("not found")
}

func (f *fakePluginInstallationRepo) ListPluginInstallations(context.Context) ([]domain.PluginInstallation, error) {
	return []domain.PluginInstallation{f.inst}, nil
}

func (f *fakePluginInstallationRepo) DeletePluginInstallation(context.Context, string, string, string) error {
	return nil
}

func TestAutomationConfigSchemaIsBuiltIn(t *testing.T) {
	m := NewManager(t.TempDir(), nil, nil, nil)
	schemaJSON, uiJSON, err := m.GetConfigSchemaInstance(context.Background(), "automation", "lightboat", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uiJSON != "{}" {
		t.Fatalf("unexpected ui schema: %q", uiJSON)
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		t.Fatalf("invalid schema json: %v", err)
	}
	if schema["type"] != "object" {
		t.Fatalf("unexpected schema type: %v", schema["type"])
	}
	props, _ := schema["properties"].(map[string]any)
	if _, ok := props["base_url"]; !ok {
		t.Fatalf("schema missing base_url: %v", props)
	}
	if _, ok := props["api_key"]; !ok {
		t.Fatalf("schema missing api_key: %v", props)
	}
}

func TestAutomationUpdateConfigBypassesPluginValidation(t *testing.T) {
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	cipher, err := cryptox.NewAESGCM(key)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}

	repo := &fakePluginInstallationRepo{
		inst: domain.PluginInstallation{
			Category:   "automation",
			PluginID:   "lightboat",
			InstanceID: "default",
			Enabled:    false,
		},
	}
	m := NewManager(t.TempDir(), repo, cipher, nil)

	if err := m.UpdateConfigInstance(context.Background(), "automation", "lightboat", "default", `{"base_url":"https://api.example.com","api_key":"secret","timeout_sec":10}`); err != nil {
		t.Fatalf("update config: %v", err)
	}

	got, err := m.GetConfigInstance(context.Background(), "automation", "lightboat", "default")
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(got), &cfg); err != nil {
		t.Fatalf("invalid config json: %v", err)
	}
	if cfg["base_url"] != "https://api.example.com" {
		t.Fatalf("unexpected base_url: %v", cfg["base_url"])
	}
	if cfg["api_key"] != "secret" {
		t.Fatalf("unexpected api_key: %v", cfg["api_key"])
	}
}
