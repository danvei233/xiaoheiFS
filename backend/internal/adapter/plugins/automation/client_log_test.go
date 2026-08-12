package automation

import (
	"errors"
	"gorm.io/gorm"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type captureAutomationLogs struct {
	last *domain.AutomationLog
}

type countingSettings struct {
	*fakeSettings
	getCalls map[string]int
}

func (c *countingSettings) UpdateSettingAtomic(ctx context.Context, key string, mutate func(current string) (string, error)) error {
	current, err := c.GetSetting(ctx, key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			current = domain.Setting{}
		} else {
			return err
		}
	}
	next, err := mutate(current.ValueJSON)
	if err != nil {
		return err
	}
	return c.UpsertSetting(ctx, domain.Setting{Key: key, ValueJSON: next})
}
func (c *countingSettings) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	if c.getCalls == nil {
		c.getCalls = map[string]int{}
	}
	c.getCalls[key]++
	return c.fakeSettings.GetSetting(ctx, key)
}

type countingAutomationLogs struct {
	captureAutomationLogs
	purgeCalls  int
	createCalls int
}

func (c *countingAutomationLogs) PurgeAutomationLogs(ctx context.Context, before time.Time) error {
	c.purgeCalls++
	return nil
}

func (c *countingAutomationLogs) CreateAutomationLog(ctx context.Context, log *domain.AutomationLog) error {
	c.createCalls++
	return c.captureAutomationLogs.CreateAutomationLog(ctx, log)
}

type fakeSettings struct {
	values map[string]string
}

func (f *fakeSettings) UpdateSettingAtomic(ctx context.Context, key string, mutate func(current string) (string, error)) error {
	current, err := f.GetSetting(ctx, key)
	if err != nil {
		current = domain.Setting{}
	}
	next, err := mutate(current.ValueJSON)
	if err != nil {
		return err
	}
	return f.UpsertSetting(ctx, domain.Setting{Key: key, ValueJSON: next})
}

func (f *fakeSettings) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	_ = ctx
	if f != nil && f.values != nil {
		if v, ok := f.values[key]; ok {
			return domain.Setting{Key: key, ValueJSON: v}, nil
		}
	}
	return domain.Setting{}, appshared.ErrNotFound
}

func (f *fakeSettings) UpsertSetting(ctx context.Context, setting domain.Setting) error {
	_ = ctx
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[setting.Key] = setting.ValueJSON
	return nil
}

func (f *fakeSettings) UpsertSettings(ctx context.Context, settings []domain.Setting) error {
	for _, s := range settings {
		if err := f.UpsertSetting(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeSettings) ListSettings(ctx context.Context) ([]domain.Setting, error) {
	_ = ctx
	out := make([]domain.Setting, 0, len(f.values))
	for k, v := range f.values {
		out = append(out, domain.Setting{Key: k, ValueJSON: v})
	}
	return out, nil
}

func (f *fakeSettings) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	_ = ctx
	return nil, nil
}

func (f *fakeSettings) GetEmailTemplate(ctx context.Context, id int64) (domain.EmailTemplate, error) {
	_ = ctx
	_ = id
	return domain.EmailTemplate{}, appshared.ErrNotFound
}

func (f *fakeSettings) UpsertEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error {
	_ = ctx
	_ = tmpl
	return nil
}

func (f *fakeSettings) DeleteEmailTemplate(ctx context.Context, id int64) error {
	_ = ctx
	_ = id
	return nil
}

func (c *captureAutomationLogs) CreateAutomationLog(ctx context.Context, log *domain.AutomationLog) error {
	if log == nil {
		return nil
	}
	cloned := *log
	c.last = &cloned
	return nil
}

func (c *captureAutomationLogs) ListAutomationLogs(ctx context.Context, orderID int64, limit, offset int) ([]domain.AutomationLog, int, error) {
	return nil, 0, nil
}

func (c *captureAutomationLogs) PurgeAutomationLogs(ctx context.Context, before time.Time) error {
	return nil
}

func parseRequestPayload(t *testing.T, raw string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode request payload: %v", err)
	}
	return payload
}

func toStringMap(v any) map[string]string {
	out := map[string]string{}
	switch m := v.(type) {
	case map[string]any:
		for key, val := range m {
			out[key] = stringify(val)
		}
	case map[string]string:
		for key, val := range m {
			out[key] = val
		}
	}
	return out
}

func stringify(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func TestPluginInstanceClient_LogRPC_RecordsGRPCConnectionMeta(t *testing.T) {
	logs := &captureAutomationLogs{}
	settings := &fakeSettings{values: map[string]string{"debug_enabled": "true"}}
	client := NewPluginInstanceClient(nil, "demo_plugin", "line-a", settings, logs)

	client.logRPC(
		context.Background(),
		"automation.ListImages",
		map[string]any{"line_id": 4},
		map[string]any{"items": []any{}},
		12*time.Millisecond,
		nil,
	)

	if logs.last == nil {
		t.Fatalf("expected log entry")
	}
	req := parseRequestPayload(t, logs.last.RequestJSON)
	if got := stringify(req["method"]); got != "GRPC" {
		t.Fatalf("unexpected method: %q", got)
	}
	wantURL := "grpc://automation/demo_plugin/line-a/automation.ListImages"
	if got := stringify(req["url"]); got != wantURL {
		t.Fatalf("unexpected grpc url: %q", got)
	}
	headers := toStringMap(req["headers"])
	if headers["x-plugin-id"] != "demo_plugin" || headers["x-plugin-instance-id"] != "line-a" {
		t.Fatalf("missing plugin headers: %#v", headers)
	}
	if headers["x-transport"] != "grpc" {
		t.Fatalf("missing transport header: %#v", headers)
	}
}

func TestPluginInstanceClient_LogRPC_PreservesHTTPTraceAndGRPCMeta(t *testing.T) {
	logs := &captureAutomationLogs{}
	settings := &fakeSettings{values: map[string]string{"debug_enabled": "true"}}
	client := NewPluginInstanceClient(nil, "demo_plugin", "line-a", settings, logs)

	traceRaw, _ := json.Marshal(map[string]any{
		"action": "GET /index.php/api/cloud/mirror_image",
		"request": map[string]any{
			"method": "GET",
			"url":    "http://upstream/index.php/api/cloud/mirror_image?line_id=4",
			"body":   "",
			"headers": map[string]any{
				"apikey": "***",
			},
		},
		"response": map[string]any{
			"status": 200,
			"body":   `{"code":1}`,
			"format": "json",
		},
		"message": "upstream ok",
	})
	err := fmt.Errorf("%s", "rpc failed http_trace="+base64.StdEncoding.EncodeToString(traceRaw))

	client.logRPC(
		context.Background(),
		"automation.ListImages",
		map[string]any{"line_id": 4},
		nil,
		14*time.Millisecond,
		err,
	)

	if logs.last == nil {
		t.Fatalf("expected log entry")
	}
	if logs.last.Action != "GET /index.php/api/cloud/mirror_image" {
		t.Fatalf("unexpected action: %q", logs.last.Action)
	}
	req := parseRequestPayload(t, logs.last.RequestJSON)
	if got := stringify(req["method"]); got != "GET" {
		t.Fatalf("unexpected method from trace: %q", got)
	}
	headers := toStringMap(req["headers"])
	if headers["x-plugin-id"] != "demo_plugin" || headers["x-plugin-instance-id"] != "line-a" {
		t.Fatalf("grpc context should be preserved in headers: %#v", headers)
	}
	if headers["apikey"] != "***" {
		t.Fatalf("http trace headers should be preserved: %#v", headers)
	}
}

func TestPluginInstanceClient_LogRPC_RedactsProvisioningPasswords(t *testing.T) {
	logs := &captureAutomationLogs{}
	settings := &fakeSettings{values: map[string]string{"debug_enabled": "true"}}
	client := NewPluginInstanceClient(nil, "demo_plugin", "line-a", settings, logs)

	client.logRPC(
		context.Background(),
		"automation.CreateInstance",
		map[string]any{
			"password":     "system-secret",
			"vnc_password": "vnc-secret",
			"nested_json":  `{"os_password":"nested-system-secret","vnc_pwd":"nested-vnc-secret"}`,
		},
		map[string]any{"password": "response-secret"},
		time.Millisecond,
		nil,
	)

	if logs.last == nil {
		t.Fatalf("expected log entry")
	}
	serialized := logs.last.RequestJSON + logs.last.ResponseJSON
	for _, secret := range []string{"system-secret", "vnc-secret", "nested-system-secret", "nested-vnc-secret", "response-secret"} {
		if strings.Contains(serialized, secret) {
			t.Fatalf("automation log contains secret %q: %s", secret, serialized)
		}
	}
}

func TestPluginInstanceClient_LogRPC_RemovesSecretsFromHTTPTraceMessage(t *testing.T) {
	logs := &captureAutomationLogs{}
	settings := &fakeSettings{values: map[string]string{"debug_enabled": "true"}}
	client := NewPluginInstanceClient(nil, "demo_plugin", "line-a", settings, logs)
	traceRaw, _ := json.Marshal(map[string]any{
		"request": map[string]any{"body": map[string]any{"password": "trace-system-secret", "vnc_password": "trace-vnc-secret"}},
	})
	err := fmt.Errorf("%s", "rpc failed http_trace="+base64.StdEncoding.EncodeToString(traceRaw))

	client.logRPC(context.Background(), "automation.CreateInstance", nil, nil, time.Millisecond, err)

	if logs.last == nil {
		t.Fatalf("expected log entry")
	}
	serialized := logs.last.RequestJSON + logs.last.ResponseJSON + logs.last.Message
	if strings.Contains(serialized, "trace-system-secret") || strings.Contains(serialized, "trace-vnc-secret") {
		t.Fatalf("automation log contains HTTP trace secret: %s", serialized)
	}
}

func TestPluginInstanceClient_LogRPC_RedactsCredentialKeys(t *testing.T) {
	logs := &captureAutomationLogs{}
	settings := &fakeSettings{values: map[string]string{"debug_enabled": "true"}}
	client := NewPluginInstanceClient(nil, "demo_plugin", "line-a", settings, logs)
	traceRaw, _ := json.Marshal(map[string]any{
		"request": map[string]any{
			"headers": map[string]any{
				"authorization": "Bearer abc123",
				"x-api-key":     "key-123",
			},
			"body": map[string]any{
				"access_token":  "access-token-secret",
				"refresh_token": "refresh-token-secret",
				"auth_token":    "auth-token-secret",
				"jwt":           "jwt-secret",
				"private_key":   "private-key-secret",
			},
		},
		"response": map[string]any{
			"body_json": map[string]any{
				"id_token": "id-token-secret",
			},
		},
	})
	err := fmt.Errorf("%s", "rpc failed http_trace="+base64.StdEncoding.EncodeToString(traceRaw))

	client.logRPC(context.Background(), "automation.CreateInstance", nil, nil, time.Millisecond, err)

	if logs.last == nil {
		t.Fatalf("expected log entry")
	}
	serialized := logs.last.RequestJSON + logs.last.ResponseJSON + logs.last.Message
	for _, secret := range []string{"abc123", "key-123", "access-token-secret", "refresh-token-secret", "auth-token-secret", "jwt-secret", "private-key-secret", "id-token-secret"} {
		if strings.Contains(serialized, secret) {
			t.Fatalf("automation log contains secret %q: %s", secret, serialized)
		}
	}
}

func TestPluginInstanceClient_LogRPC_SuccessPathSkipsWritesAndReadsOnlyDebug(t *testing.T) {
	settings := &countingSettings{fakeSettings: &fakeSettings{values: map[string]string{}}}
	logs := &countingAutomationLogs{}
	client := NewPluginInstanceClient(nil, "demo_plugin", "line-a", settings, logs)

	client.logRPC(context.Background(), "automation.GetMonitor", nil, nil, time.Millisecond, nil)

	if logs.purgeCalls != 0 || logs.createCalls != 0 {
		t.Fatalf("expected no DB writes on success with debug disabled, purge=%d create=%d", logs.purgeCalls, logs.createCalls)
	}
	if got := settings.getCalls["debug_enabled"]; got != 1 {
		t.Fatalf("expected exactly one debug_enabled read, got %d", got)
	}
	if _, ok := settings.getCalls["automation_log_retention_days"]; ok {
		t.Fatalf("retention setting should not be read on the success fast path")
	}
}

// TestPluginInstanceClient_LogRPC_ScrubsTraceMessageSecrets guards the trace
// message leak (runtime F3 #1): a plugin trace whose message embeds a credential
// (e.g. "Authorization: Bearer SECRET") must not be persisted verbatim into the
// automation_log Message/body fields.
func TestPluginInstanceClient_LogRPC_ScrubsTraceMessageSecrets(t *testing.T) {
	logs := &captureAutomationLogs{}
	settings := &fakeSettings{values: map[string]string{"debug_enabled": "true"}}
	client := NewPluginInstanceClient(nil, "demo_plugin", "line-a", settings, logs)
	traceRaw, _ := json.Marshal(map[string]any{
		"message": "Authorization: Bearer SUPERSECRETTOKEN1234567890",
	})
	err := fmt.Errorf("%s", "rpc failed http_trace="+base64.StdEncoding.EncodeToString(traceRaw))

	client.logRPC(context.Background(), "automation.CreateInstance", nil, nil, time.Millisecond, err)

	if logs.last == nil {
		t.Fatalf("expected log entry")
	}
	if strings.Contains(logs.last.Message, "SUPERSECRETTOKEN1234567890") {
		t.Fatalf("trace message credential persisted: %s", logs.last.Message)
	}
}

// TestPluginInstanceClient_LogRPC_DoesNotPersistUnparseableTrace guards the
// parse-failure leak (runtime F3 #2): an error carrying an http_trace= marker
// whose payload is not valid base64/JSON must never be persisted verbatim —
// the raw base64 request/response trace would otherwise land in the log.
func TestPluginInstanceClient_LogRPC_DoesNotPersistUnparseableTrace(t *testing.T) {
	logs := &captureAutomationLogs{}
	settings := &fakeSettings{values: map[string]string{"debug_enabled": "true"}}
	client := NewPluginInstanceClient(nil, "demo_plugin", "line-a", settings, logs)
	payload := base64.StdEncoding.EncodeToString([]byte("{not-json"))
	err := fmt.Errorf("%s", "upstream failed http_trace="+payload)

	client.logRPC(context.Background(), "automation.CreateInstance", nil, nil, time.Millisecond, err)

	if logs.last == nil {
		t.Fatalf("expected log entry")
	}
	serialized := logs.last.RequestJSON + logs.last.ResponseJSON + logs.last.Message
	if strings.Contains(serialized, "http_trace=") || strings.Contains(serialized, payload) {
		t.Fatalf("unparseable http_trace payload persisted: %s", serialized)
	}
}

// TestPluginInstanceClient_LogRPC_ScrubsFormBodySecrets guards the form-body
// leak (P0-13 residual): a trace request body sent as
// application/x-www-form-urlencoded (e.g. sys_pwd=SECRET&vnc_pwd=SECRET) has no
// "?" marker, so the URL-query scrubber would pass it through unmasked. The
// form-body scrubber must mask the sensitive keys before the body is persisted.
func TestPluginInstanceClient_LogRPC_ScrubsFormBodySecrets(t *testing.T) {
	logs := &captureAutomationLogs{}
	settings := &fakeSettings{values: map[string]string{"debug_enabled": "true"}}
	client := NewPluginInstanceClient(nil, "demo_plugin", "line-a", settings, logs)

	traceRaw, _ := json.Marshal(map[string]any{
		"request": map[string]any{
			"method": "POST",
			"url":    "https://panel.example.com/index.php/api/cloud/create_host",
			// form-urlencoded body: no "?" present, secrets must still be masked.
			"body": "line_id=7&os=linux&cpu=2&sys_pwd=sys-form-secret&vnc_pwd=vnc-form-secret&panel_password=panel-form-secret",
		},
	})
	err := fmt.Errorf("%s", "rpc failed http_trace="+base64.StdEncoding.EncodeToString(traceRaw))

	client.logRPC(context.Background(), "automation.CreateInstance", nil, nil, time.Millisecond, err)

	if logs.last == nil {
		t.Fatalf("expected log entry")
	}
	serialized := logs.last.RequestJSON + logs.last.ResponseJSON + logs.last.Message
	for _, secret := range []string{"sys-form-secret", "vnc-form-secret", "panel-form-secret"} {
		if strings.Contains(serialized, secret) {
			t.Fatalf("form body secret %q persisted unmasked: %s", secret, serialized)
		}
	}
	req := parseRequestPayload(t, logs.last.RequestJSON)
	body := stringify(req["body"])
	// url.Values.Encode() percent-encodes the mask: *** becomes %2A%2A%2A.
	for _, key := range []string{"sys_pwd", "vnc_pwd", "panel_password"} {
		if !strings.Contains(body, key+"=%2A%2A%2A") {
			t.Fatalf("form body key %q not masked: %q", key, body)
		}
	}
}

// TestSanitizePayload_ScrubsFormBodySecrets exercises sanitizePayload directly
// on a form-urlencoded body string, independent of the http_trace plumbing.
func TestSanitizePayload_ScrubsFormBodySecrets(t *testing.T) {
	in := "sys_pwd=SECRET&vnc_pwd=SECRET&line_id=7"
	got := sanitizePayload(in)
	out, ok := got.(string)
	if !ok {
		t.Fatalf("expected string result, got %T", got)
	}
	if strings.Contains(out, "SECRET") {
		t.Fatalf("form body secret persisted unmasked: %q", out)
	}
	// url.Values.Encode() percent-encodes the mask: *** becomes %2A%2A%2A.
	if !strings.Contains(out, "sys_pwd=%2A%2A%2A") || !strings.Contains(out, "vnc_pwd=%2A%2A%2A") {
		t.Fatalf("expected sys_pwd/vnc_pwd masked: %q", out)
	}
	if !strings.Contains(out, "line_id=7") {
		t.Fatalf("non-sensitive form key must survive: %q", out)
	}
}

// TestSanitizePayload_LeavesNonFormStringsUntouched ensures the form-body
// scrubber does not mangle plain text or non-sensitive form bodies.
func TestSanitizePayload_LeavesNonFormStringsUntouched(t *testing.T) {
	for _, in := range []string{
		"plain text without equals",
		"line_id=7&os=linux",
		"https://panel.example.com/path",
		"",
	} {
		if got := sanitizePayload(in); got != in {
			t.Fatalf("sanitizePayload(%q) altered a non-sensitive string: %q", in, got)
		}
	}
}

// TestPluginInstanceClient_LogRPC_ScrubsURLQuerySecrets guards the URL-query
// leak (runtime F3 #3): a trace request URL such as
// https://api.example.com?token=SECRET must be persisted with the token value
// masked, not verbatim.
func TestPluginInstanceClient_LogRPC_ScrubsURLQuerySecrets(t *testing.T) {
	logs := &captureAutomationLogs{}
	settings := &fakeSettings{values: map[string]string{"debug_enabled": "true"}}
	client := NewPluginInstanceClient(nil, "demo_plugin", "line-a", settings, logs)
	traceRaw, _ := json.Marshal(map[string]any{
		"request": map[string]any{
			"method": "GET",
			"url":    "https://api.example.com?token=SECRETQUERYTOKEN",
		},
	})
	err := fmt.Errorf("%s", "rpc failed http_trace="+base64.StdEncoding.EncodeToString(traceRaw))

	client.logRPC(context.Background(), "automation.CreateInstance", nil, nil, time.Millisecond, err)

	if logs.last == nil {
		t.Fatalf("expected log entry")
	}
	if strings.Contains(logs.last.RequestJSON, "SECRETQUERYTOKEN") {
		t.Fatalf("URL query secret persisted: %s", logs.last.RequestJSON)
	}
}

func TestExtractHTTPTrace_CapsOversizedTrace(t *testing.T) {
	huge := strings.Repeat("A", maxHTTPTraceBytes+1)
	err := fmt.Errorf("%s http_trace=%s", "upstream failed", `{"request":{"body":"`+huge+`"}}`)

	req, resp, action, msg, ok := extractHTTPTrace(err)
	if ok {
		t.Fatalf("expected oversized trace to be rejected, got action=%q msg=%q", action, msg)
	}
	if req != nil || resp != nil {
		t.Fatalf("expected nil trace payloads for rejected trace")
	}
}
