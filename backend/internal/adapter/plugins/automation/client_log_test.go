package automation

import (
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

type fakeSettings struct {
	values map[string]string
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
