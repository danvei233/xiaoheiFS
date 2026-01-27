package automation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"xiaoheiplay/internal/usecase"
)

func TestClient_HTTPFlows(t *testing.T) {
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
	mux.HandleFunc("/index.php/api/cloud/hostinfo", func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, map[string]any{
			"id":             123,
			"host_name":      "host-1",
			"state":          2,
			"cpu":            2,
			"memory":         4,
			"hard_disks":     40,
			"bandwidth":      10,
			"panel_password": "pp",
			"vnc_password":   "vv",
			"os_password":    "os",
			"remote_ip":      "1.1.1.1",
			"end_time":       "2025-01-02",
		})
	})
	mux.HandleFunc("/index.php/api/cloud/hostlist", func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, []map[string]any{{"id": 1, "host_name": "h", "ip": "2.2.2.2"}})
	})
	mux.HandleFunc("/index.php/api/cloud/elastic_update", func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, map[string]any{})
	})
	mux.HandleFunc("/index.php/api/cloud/renew", func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, map[string]any{})
	})
	for _, path := range []string{"/start", "/shutdown", "/reboot", "/lock", "/unlock", "/delete"} {
		mux.HandleFunc("/index.php/api/cloud"+path, func(w http.ResponseWriter, r *http.Request) {
			writeOK(w, map[string]any{})
		})
	}
	mux.HandleFunc("/index.php/api/cloud/panel", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/panel/abc")
		w.WriteHeader(http.StatusFound)
	})
	mux.HandleFunc("/index.php/api/cloud/mirror_image", func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, []map[string]any{{"id": 10, "name": "img", "type": "linux"}})
	})
	mux.HandleFunc("/index.php/api/cloud/area_list", func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, []map[string]any{{"id": 2, "area_name": "area", "state": 1}})
	})
	mux.HandleFunc("/index.php/api/cloud/line", func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, []map[string]any{{"id": 3, "line_api": "line-api", "area_id": 2, "state": 1}})
	})
	mux.HandleFunc("/index.php/api/cloud/product", func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, map[string]any{
			"p1": map[string]any{
				"id":           7,
				"product_name": "prod",
				"host_cpu":     2,
				"host_ram":     4,
				"host_data":    40,
				"bandwidth":    10,
				"nat_port_num": 20,
				"price":        "12.5",
			},
		})
	})
	mux.HandleFunc("/index.php/api/cloud/monitor", func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, map[string]any{
			"StorageStats": 10,
			"NetworkStats": [][]any{{0, 100, 200}},
			"CpuStats":     20,
			"MemoryStats":  30,
		})
	})
	mux.HandleFunc("/index.php/api/cloud/vnc_view", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "http://example.com/vnc")
		w.WriteHeader(http.StatusTemporaryRedirect)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.URL+"/index.php/api/cloud", "secret", time.Second)
	var lastLog httpLogEntry
	client.WithLogger(func(ctx context.Context, entry httpLogEntry) {
		lastLog = entry
	})

	ctx := context.Background()
	if res, err := client.CreateHost(ctx, usecase.AutomationCreateHostRequest{
		LineID:     1,
		OS:         "linux",
		CPU:        2,
		MemoryGB:   4,
		DiskGB:     40,
		Bandwidth:  10,
		ExpireTime: time.Now(),
	}); err != nil || res.HostID != 123 {
		t.Fatalf("create host: %v %v", res, err)
	}
	if info, err := client.GetHostInfo(ctx, 123); err != nil || info.HostID != 123 || info.RemoteIP == "" {
		t.Fatalf("host info: %v %v", info, err)
	}
	if items, err := client.ListHostSimple(ctx, ""); err != nil || len(items) != 1 {
		t.Fatalf("host list: %v %v", items, err)
	}
	if err := client.ElasticUpdate(ctx, usecase.AutomationElasticUpdateRequest{HostID: 1}); err != nil {
		t.Fatalf("elastic update: %v", err)
	}
	if err := client.RenewHost(ctx, 1, time.Now()); err != nil {
		t.Fatalf("renew: %v", err)
	}
	if err := client.StartHost(ctx, 1); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := client.ShutdownHost(ctx, 1); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := client.RebootHost(ctx, 1); err != nil {
		t.Fatalf("reboot: %v", err)
	}
	if err := client.LockHost(ctx, 1); err != nil {
		t.Fatalf("lock: %v", err)
	}
	if err := client.UnlockHost(ctx, 1); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if err := client.DeleteHost(ctx, 1); err != nil {
		t.Fatalf("delete: %v", err)
	}
	panelURL, err := client.GetPanelURL(ctx, "host-1", "pp")
	if err != nil || !strings.Contains(panelURL, "/panel/abc") {
		t.Fatalf("panel url: %v %v", panelURL, err)
	}
	if imgs, err := client.ListImages(ctx, 0); err != nil || len(imgs) != 1 {
		t.Fatalf("images: %v %v", imgs, err)
	}
	if areas, err := client.ListAreas(ctx); err != nil || len(areas) != 1 {
		t.Fatalf("areas: %v %v", areas, err)
	}
	if lines, err := client.ListLines(ctx); err != nil || len(lines) != 1 || lines[0].Name != "line-api" {
		t.Fatalf("lines: %v %v", lines, err)
	}
	if products, err := client.ListProducts(ctx, 0); err != nil || len(products) != 1 {
		t.Fatalf("products: %v %v", products, err)
	}
	if monitor, err := client.GetMonitor(ctx, 1); err != nil || monitor.CPUPercent == 0 || monitor.BytesOut == 0 {
		t.Fatalf("monitor: %v %v", monitor, err)
	}
	if vncURL, err := client.GetVNCURL(ctx, 1); err != nil || !strings.Contains(vncURL, "example.com") {
		t.Fatalf("vnc url: %v %v", vncURL, err)
	}
	if headers, ok := lastLog.Request["headers"].(map[string]string); ok {
		found := false
		for k, v := range headers {
			if strings.EqualFold(k, "apikey") {
				found = true
				if v != "***" {
					t.Fatalf("expected api key masked, got %q", v)
				}
			}
		}
		if !found {
			t.Fatalf("expected api key header")
		}
	}
}
