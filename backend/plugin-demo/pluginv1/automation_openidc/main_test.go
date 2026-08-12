package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math"
	"strings"
	"testing"

	pluginv1 "xiaoheiplay/plugin/v1"
)

// TestRedactSensitiveJSON covers H2/H3: request bodies that contain
// os_pass/vc_pass/password/api_key must not be written verbatim to plugin.log
// or embedded in the http_trace returned to the host.
func TestRedactSensitiveJSON(t *testing.T) {
	in := `{"vm_uuid":"vm-1","cpu_num":2,"os_pass":"super-secret","vc_pass":"vnc-secret","nic_all":{"nic0":{"nic_type":"nat"}}}`
	got := redactSensitiveJSON([]byte(in))
	if strings.Contains(got, "super-secret") || strings.Contains(got, "vnc-secret") {
		t.Fatalf("redactSensitiveJSON leaked password: %s", got)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("redactSensitiveJSON returned non-JSON: %q (%v)", got, err)
	}
	if v, _ := m["os_pass"].(string); v != "***" {
		t.Fatalf("os_pass = %q, want ***", v)
	}
	if v, _ := m["vc_pass"].(string); v != "***" {
		t.Fatalf("vc_pass = %q, want ***", v)
	}
	if v, _ := m["cpu_num"].(float64); v != 2 {
		t.Fatalf("cpu_num = %v, want 2 (non-secret fields preserved)", v)
	}
}

// TestRedactSensitiveJSON_NonJSON ensures a non-JSON body (e.g. form data) is
// never dumped raw — only its length is logged.
func TestRedactSensitiveJSON_NonJSON(t *testing.T) {
	got := redactSensitiveJSON([]byte("password=plaintext-secret&os_pass=x"))
	if strings.Contains(got, "plaintext-secret") || strings.Contains(got, "os_pass") {
		t.Fatalf("non-JSON body leaked: %s", got)
	}
}

// TestPositiveID covers L4: fnv64 must never produce negative ids, including
// the case where the raw 64-bit hash has the sign bit set (int64 conversion
// would previously produce MinInt64, and -v overflows back to a negative).
func TestPositiveID(t *testing.T) {
	cases := []struct {
		in   uint64
		want int64
	}{
		{0, 0},
		{math.MaxInt64, math.MaxInt64},
		{0x8000000000000000, 0}, // sign bit set -> masked to 0
		{0x8000000000000001, 1}, // sign bit set -> low bits preserved
		{math.MaxUint64, math.MaxInt64},
	}
	for _, c := range cases {
		if got := positiveID(c.in); got != c.want {
			t.Errorf("positiveID(%#x) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestFnv64_NonNegative ensures fnv64 returns a non-negative id for a variety
// of realistic host/vm name inputs.
func TestFnv64_NonNegative(t *testing.T) {
	inputs := []string{
		"hs-01", "hs-01/vm-uuid-1", "hz1/plan/standard",
		"", "a", "xiaohei-host-01/xiaohei-vm-02",
	}
	for _, in := range inputs {
		if got := fnv64(in); got < 0 {
			t.Fatalf("fnv64(%q) = %d, want >= 0", in, got)
		}
	}
}

// TestInit_DoesNotLogConfigJSON covers H2: Init must not write the full
// config_json (containing api_key) to plugin.log.
func TestInit_DoesNotLogConfigJSON(t *testing.T) {
	var buf bytes.Buffer
	old := pluginLog
	pluginLog = log.New(&buf, "[openidc-test] ", log.LstdFlags)
	defer func() { pluginLog = old }()

	s := &coreServer{}
	secret := "sk-0123456789abcdef"
	cfgJSON := `{"base_url":"http://127.0.0.1:1880","api_key":"` + secret + `","dry_run":false}`
	if _, err := s.Init(nil, &pluginv1.InitRequest{InstanceId: "inst-1", ConfigJson: cfgJSON}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	logged := buf.String()
	if strings.Contains(logged, secret) {
		t.Fatalf("Init logged the api_key to plugin.log:\n%s", logged)
	}
	if strings.Contains(logged, "config_json=") {
		t.Fatalf("Init logged raw config_json to plugin.log:\n%s", logged)
	}
}
