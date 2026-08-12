package plugins

import (
	"encoding/json"
	"os"
	"sort"
	"testing"
)

// schemaFile is the subset of a JSON Schema object that the static plugin
// config schemas use and that the metadata tests care about.
type schemaFile struct {
	Properties map[string]json.RawMessage `json:"properties"`
	Required   []string                   `json:"required"`
}

func loadSchema(t *testing.T, rel string) schemaFile {
	t.Helper()
	b, err := os.ReadFile(rel)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	var s schemaFile
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("parse %s: %v", rel, err)
	}
	return s
}

func loadUISchema(t *testing.T, rel string) map[string]json.RawMessage {
	t.Helper()
	b, err := os.ReadFile(rel)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	var u map[string]json.RawMessage
	if err := json.Unmarshal(b, &u); err != nil {
		t.Fatalf("parse %s: %v", rel, err)
	}
	return u
}

// prop decodes a single schema property into a plain map for assertions.
func prop(t *testing.T, s schemaFile, name string) map[string]any {
	t.Helper()
	raw, ok := s.Properties[name]
	if !ok {
		t.Fatalf("property %q missing", name)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse property %q: %v", name, err)
	}
	return m
}

func num(t *testing.T, m map[string]any, key string) float64 {
	t.Helper()
	v, ok := m[key].(float64)
	if !ok {
		t.Fatalf("property key %q is not a number", key)
	}
	return v
}

func strSlice(t *testing.T, m map[string]any, key string) []string {
	t.Helper()
	raw, ok := m[key].([]any)
	if !ok {
		t.Fatalf("property key %q is not an array", key)
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		if !ok {
			t.Fatalf("property key %q contains a non-string element", key)
		}
		out = append(out, s)
	}
	return out
}

func setEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aa := append([]string(nil), a...)
	bb := append([]string(nil), b...)
	sort.Strings(aa)
	sort.Strings(bb)
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}
