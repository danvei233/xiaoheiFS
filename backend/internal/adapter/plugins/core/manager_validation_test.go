package plugins

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
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

	// The stored config is encrypted; the admin read path must redact the
	// automation api_key (m068 F1) while keeping the non-secret fields.
	stored, err := m.repo.GetPluginInstallation(context.Background(), "automation", "lightboat", "default")
	if err != nil {
		t.Fatalf("get stored installation: %v", err)
	}
	if stored.ConfigCipher == "" || stored.ConfigCipher == `{"base_url":"https://api.example.com","api_key":"secret","timeout_sec":10}` {
		t.Fatalf("automation config is not stored encrypted: %q", stored.ConfigCipher)
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
	if apiKey, ok := cfg["api_key"].(string); !ok || apiKey != "" {
		t.Fatalf("automation api_key must be redacted on read, got %#v", cfg["api_key"])
	}
}

// TestAutomationConfigUpdatePreservesAPIKeyWhenOmitted is the regression for
// the silent api_key wipe: a PATCH that updates base_url (or any non-secret
// field) WITHOUT resubmitting api_key must keep the stored key. The handler
// reads the redacted config (api_key="") and writes it back, so the merge must
// treat the empty api_key as "keep old value" — exactly like plugin-managed
// secrets.
func TestAutomationConfigUpdatePreservesAPIKeyWhenOmitted(t *testing.T) {
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	cipher, err := cryptox.NewAESGCM(key)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	initial, err := cipher.EncryptToString([]byte(`{"base_url":"https://api.example.com","api_key":"SECRET","timeout_sec":10,"retry":0,"dry_run":false}`))
	if err != nil {
		t.Fatalf("encrypt initial config: %v", err)
	}
	repo := &fakePluginInstallationRepo{
		inst: domain.PluginInstallation{
			Category:     "automation",
			PluginID:     "lightboat",
			InstanceID:   "default",
			Enabled:      false,
			ConfigCipher: initial,
		},
	}
	m := NewManager(t.TempDir(), repo, cipher, nil)

	// Mimic the handler: the read-back config has api_key redacted to "", and the
	// PATCH only changes base_url without supplying a new key.
	if err := m.UpdateConfigInstance(context.Background(), "automation", "lightboat", "default",
		`{"base_url":"https://new.example.com","api_key":"","timeout_sec":10,"retry":0,"dry_run":false}`); err != nil {
		t.Fatalf("update config: %v", err)
	}

	stored, err := m.repo.GetPluginInstallation(context.Background(), "automation", "lightboat", "default")
	if err != nil {
		t.Fatalf("get stored installation: %v", err)
	}
	plain, err := m.decryptConfig(stored.ConfigCipher)
	if err != nil {
		t.Fatalf("decrypt stored config: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(plain), &cfg); err != nil {
		t.Fatalf("invalid stored config json: %v", err)
	}
	if cfg["base_url"] != "https://new.example.com" {
		t.Fatalf("base_url not updated: %v", cfg["base_url"])
	}
	if apiKey, ok := cfg["api_key"].(string); !ok || apiKey != "SECRET" {
		t.Fatalf("api_key was lost when omitted from the update, got %#v", cfg["api_key"])
	}
}

// TestAutomationConfigUpdateReplacesAPIKeyWhenProvided confirms the merge does
// not block an explicit key rotation: a non-empty api_key in the update is
// written through to the stored config.
func TestAutomationConfigUpdateReplacesAPIKeyWhenProvided(t *testing.T) {
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	cipher, err := cryptox.NewAESGCM(key)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	initial, err := cipher.EncryptToString([]byte(`{"base_url":"https://api.example.com","api_key":"OLD","timeout_sec":10,"retry":0,"dry_run":false}`))
	if err != nil {
		t.Fatalf("encrypt initial config: %v", err)
	}
	repo := &fakePluginInstallationRepo{
		inst: domain.PluginInstallation{
			Category:     "automation",
			PluginID:     "lightboat",
			InstanceID:   "default",
			Enabled:      false,
			ConfigCipher: initial,
		},
	}
	m := NewManager(t.TempDir(), repo, cipher, nil)

	if err := m.UpdateConfigInstance(context.Background(), "automation", "lightboat", "default",
		`{"base_url":"https://api.example.com","api_key":"NEW","timeout_sec":10,"retry":0,"dry_run":false}`); err != nil {
		t.Fatalf("update config: %v", err)
	}
	stored, err := m.repo.GetPluginInstallation(context.Background(), "automation", "lightboat", "default")
	if err != nil {
		t.Fatalf("get stored installation: %v", err)
	}
	plain, err := m.decryptConfig(stored.ConfigCipher)
	if err != nil {
		t.Fatalf("decrypt stored config: %v", err)
	}
	if !strings.Contains(plain, `"api_key":"NEW"`) {
		t.Fatalf("rotated api_key not stored: %s", plain)
	}
}

func TestAutomationGetConfigFallsBackWhenCipherKeyChanged(t *testing.T) {
	oldKey := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	oldCipher, err := cryptox.NewAESGCM(oldKey)
	if err != nil {
		t.Fatalf("new old cipher: %v", err)
	}
	ct, err := oldCipher.EncryptToString([]byte(`{"base_url":"https://old.example.com","api_key":"old"}`))
	if err != nil {
		t.Fatalf("encrypt old config: %v", err)
	}

	newKey := base64.RawURLEncoding.EncodeToString([]byte("fedcba9876543210fedcba9876543210"))
	newCipher, err := cryptox.NewAESGCM(newKey)
	if err != nil {
		t.Fatalf("new new cipher: %v", err)
	}
	repo := &fakePluginInstallationRepo{
		inst: domain.PluginInstallation{
			Category:     "automation",
			PluginID:     "lightboat",
			InstanceID:   "default",
			Enabled:      false,
			ConfigCipher: ct,
		},
	}
	m := NewManager(t.TempDir(), repo, newCipher, nil)

	// Old ciphertext cannot be decrypted with the new key. We should still
	// return an editable empty config so admin can recover by saving new values.
	got, err := m.GetConfigInstance(context.Background(), "automation", "lightboat", "default")
	if err != nil {
		t.Fatalf("get config with mismatched key: %v", err)
	}
	if got != "{}" {
		t.Fatalf("expected empty config fallback, got: %s", got)
	}
}

func TestValidatePluginConfigJSONStrict(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "object", raw: `{"a":1}`, wantErr: false},
		{name: "array", raw: `[1,2,3]`, wantErr: true},
		{name: "scalar string", raw: `"abc"`, wantErr: true},
		{name: "double encoded object", raw: `"{\"a\":1}"`, wantErr: true},
		{name: "invalid json", raw: `{`, wantErr: true},
	}
	for _, tc := range cases {
		err := validatePluginConfigJSONStrict(tc.raw)
		if tc.wantErr && err == nil {
			t.Fatalf("%s: expected error", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
	}
}
