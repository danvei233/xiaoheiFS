package automation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/usecase"
)

type fakeSettings struct {
	values map[string]string
}

func (f *fakeSettings) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	if v, ok := f.values[key]; ok {
		return domain.Setting{Key: key, ValueJSON: v}, nil
	}
	return domain.Setting{}, usecase.ErrNotFound
}

func (f *fakeSettings) UpsertSetting(ctx context.Context, setting domain.Setting) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[setting.Key] = setting.ValueJSON
	return nil
}

func (f *fakeSettings) ListSettings(ctx context.Context) ([]domain.Setting, error) {
	return nil, nil
}

func (f *fakeSettings) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	return nil, nil
}

func (f *fakeSettings) GetEmailTemplate(ctx context.Context, id int64) (domain.EmailTemplate, error) {
	return domain.EmailTemplate{}, usecase.ErrNotFound
}

func (f *fakeSettings) UpsertEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error {
	return nil
}

func (f *fakeSettings) DeleteEmailTemplate(ctx context.Context, id int64) error {
	return nil
}

type fakeAutoLogs struct {
	createCount int
	purgeCount  int
}

func (f *fakeAutoLogs) CreateAutomationLog(ctx context.Context, log *domain.AutomationLog) error {
	f.createCount++
	return nil
}

func (f *fakeAutoLogs) ListAutomationLogs(ctx context.Context, orderID int64, limit, offset int) ([]domain.AutomationLog, int, error) {
	return nil, 0, nil
}

func (f *fakeAutoLogs) PurgeAutomationLogs(ctx context.Context, before time.Time) error {
	f.purgeCount++
	return nil
}

func TestDynamicClient_Disabled(t *testing.T) {
	settings := &fakeSettings{values: map[string]string{
		"automation_enabled": "false",
	}}
	client := NewDynamicClient(settings, "http://example", "k", nil)
	if _, err := client.CreateHost(context.Background(), usecase.AutomationCreateHostRequest{}); err == nil {
		t.Fatalf("expected disabled error")
	}
}

func TestDynamicClient_DryRun(t *testing.T) {
	settings := &fakeSettings{values: map[string]string{
		"automation_enabled": "true",
		"automation_dry_run": "true",
	}}
	client := NewDynamicClient(settings, "http://example", "k", nil)
	res, err := client.CreateHost(context.Background(), usecase.AutomationCreateHostRequest{})
	if err != nil || res.HostID == 0 {
		t.Fatalf("dry run create: %v %v", res, err)
	}
	info, err := client.GetHostInfo(context.Background(), 10)
	if err != nil || info.HostID != 10 {
		t.Fatalf("dry run info: %v %v", info, err)
	}
	if err := client.RenewHost(context.Background(), 10, time.Now()); err != nil {
		t.Fatalf("dry run renew: %v", err)
	}
	if images, err := client.ListImages(context.Background(), 0); err != nil || images == nil {
		t.Fatalf("dry run images: %v %v", images, err)
	}
}

func TestDynamicClient_Enabled(t *testing.T) {
	mux := http.NewServeMux()
	writeOK := func(w http.ResponseWriter, data any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 1,
			"msg":  "ok",
			"data": data,
		})
	}
	mux.HandleFunc("/index.php/api/cloud/create_host", func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, map[string]any{"host_id": 123})
	})
	mux.HandleFunc("/index.php/api/cloud/renew", func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, map[string]any{})
	})
	mux.HandleFunc("/index.php/api/cloud/panel", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/panel/abc")
		w.WriteHeader(http.StatusFound)
	})
	mux.HandleFunc("/index.php/api/cloud/mirror_image", func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, []map[string]any{{"id": 1, "name": "img", "type": "linux"}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	settings := &fakeSettings{values: map[string]string{
		"automation_enabled":            "true",
		"automation_base_url":           server.URL,
		"automation_api_key":            "k",
		"automation_timeout_sec":        "1",
		"automation_retry":              "0",
		"debug_enabled":                 "true",
		"automation_log_retention_days": "1",
	}}
	logs := &fakeAutoLogs{}
	client := NewDynamicClient(settings, "", "", logs)

	res, err := client.CreateHost(context.Background(), usecase.AutomationCreateHostRequest{LineID: 1, OS: "linux", CPU: 1, MemoryGB: 1, DiskGB: 10, Bandwidth: 1, ExpireTime: time.Now()})
	if err != nil || res.HostID != 123 {
		t.Fatalf("create host: %v %v", res, err)
	}
	if err := client.RenewHost(context.Background(), 1, time.Now()); err != nil {
		t.Fatalf("renew host: %v", err)
	}
	if url, err := client.GetPanelURL(context.Background(), "host", "pwd"); err != nil || url == "" {
		t.Fatalf("panel url: %v %v", url, err)
	}
	if images, err := client.ListImages(context.Background(), 0); err != nil || len(images) != 1 {
		t.Fatalf("images: %v %v", images, err)
	}
	if logs.createCount == 0 || logs.purgeCount == 0 {
		t.Fatalf("expected automation logs")
	}
}
