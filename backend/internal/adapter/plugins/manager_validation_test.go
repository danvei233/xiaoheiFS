package plugins

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
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
	wrapped := errors.New("wrap: " + err.Error())
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
