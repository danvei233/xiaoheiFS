package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"xiaoheiplay/internal/adapter/automation"
	"xiaoheiplay/internal/usecase"
	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type config struct {
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key"`
	TimeoutSec int    `json:"timeout_sec"`
	Retry      int    `json:"retry"`
	DryRun     bool   `json:"dry_run"`
}

type coreServer struct {
	pluginv1.UnimplementedCoreServiceServer

	cfg       config
	instance  string
	updatedAt time.Time
}

func (s *coreServer) GetManifest(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.Manifest, error) {
	_ = ctx
	return &pluginv1.Manifest{
		PluginId:    "lightboat",
		Name:        "Lightboat Automation (Built-in)",
		Version:     "0.1.0",
		Description: "Built-in Lightboat/Qingzhou automation plugin (catalog + lifecycle + port/backup/snapshot/firewall).",
		Automation: &pluginv1.AutomationCapability{
			Features: []pluginv1.AutomationFeature{
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_CATALOG_SYNC,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_LIFECYCLE,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_PORT_MAPPING,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_BACKUP,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_SNAPSHOT,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_FIREWALL,
			},
			NotSupportedReasons: map[int32]string{},
		},
	}, nil
}

func (s *coreServer) GetConfigSchema(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ConfigSchema, error) {
	_ = ctx
	return &pluginv1.ConfigSchema{
		JsonSchema: `{
  "title": "Lightboat Automation",
  "type": "object",
  "properties": {
    "base_url": { "type": "string", "title": "Base URL", "description": "e.g. https://panel.example.com/index.php/api/cloud" },
    "api_key": { "type": "string", "title": "API Key", "format": "password" },
    "timeout_sec": { "type": "integer", "title": "Timeout (sec)", "default": 12, "minimum": 1, "maximum": 60 },
    "retry": { "type": "integer", "title": "Retry", "default": 1, "minimum": 0, "maximum": 5 },
    "dry_run": { "type": "boolean", "title": "Dry Run", "default": false }
  },
  "required": ["base_url","api_key"]
}`,
		UiSchema: `{
  "api_key": { "ui:widget": "password", "ui:help": "留空表示不修改（由宿主处理）" }
}`,
	}, nil
}

func (s *coreServer) ValidateConfig(ctx context.Context, req *pluginv1.ValidateConfigRequest) (*pluginv1.ValidateConfigResponse, error) {
	_ = ctx
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "invalid json"}, nil
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.APIKey) == "" {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "base_url/api_key required"}, nil
	}
	if cfg.TimeoutSec < 0 || cfg.TimeoutSec > 60 {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "timeout_sec out of range"}, nil
	}
	if cfg.Retry < 0 || cfg.Retry > 5 {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "retry out of range"}, nil
	}
	return &pluginv1.ValidateConfigResponse{Ok: true}, nil
}

func (s *coreServer) Init(ctx context.Context, req *pluginv1.InitRequest) (*pluginv1.InitResponse, error) {
	if strings.TrimSpace(req.GetConfigJson()) != "" {
		var cfg config
		if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
			return &pluginv1.InitResponse{Ok: false, Error: "invalid config"}, nil
		}
		s.cfg = cfg
	}
	s.instance = req.GetInstanceId()
	s.updatedAt = time.Now()
	return &pluginv1.InitResponse{Ok: true}, nil
}

func (s *coreServer) ReloadConfig(ctx context.Context, req *pluginv1.ReloadConfigRequest) (*pluginv1.ReloadConfigResponse, error) {
	_ = ctx
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		return &pluginv1.ReloadConfigResponse{Ok: false, Error: "invalid config"}, nil
	}
	s.cfg = cfg
	s.updatedAt = time.Now()
	return &pluginv1.ReloadConfigResponse{Ok: true}, nil
}

func (s *coreServer) Health(ctx context.Context, req *pluginv1.HealthCheckRequest) (*pluginv1.HealthCheckResponse, error) {
	_ = ctx
	msg := "ok"
	if req.GetInstanceId() == "" || s.instance == "" {
		msg = "not initialized"
	}
	return &pluginv1.HealthCheckResponse{
		Status:     pluginv1.HealthStatus_HEALTH_STATUS_OK,
		Message:    msg,
		UnixMillis: time.Now().UnixMilli(),
	}, nil
}

func (s *coreServer) newClient() (*automation.Client, error) {
	cfg := s.cfg
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("missing config")
	}
	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	return automation.NewClient(cfg.BaseURL, cfg.APIKey, timeout), nil
}

func (s *coreServer) newClientWithTrace() (*automation.Client, *automation.HTTPLogEntry, error) {
	client, err := s.newClient()
	if err != nil {
		return nil, nil, err
	}
	var last automation.HTTPLogEntry
	client.WithLogger(func(_ context.Context, entry automation.HTTPLogEntry) {
		last = entry
	})
	return client, &last, nil
}

func (s *coreServer) retry(fn func() error) error {
	if s.cfg.Retry < 0 {
		return fn()
	}
	var err error
	for i := 0; i <= s.cfg.Retry; i++ {
		err = fn()
		if err == nil {
			return nil
		}
	}
	return err
}

func wrapHTTPTraceErr(err error, last *automation.HTTPLogEntry) error {
	if err == nil {
		return nil
	}
	if last == nil || strings.TrimSpace(last.Action) == "" {
		return err
	}
	trace := map[string]any{
		"action":   last.Action,
		"request":  last.Request,
		"response": last.Response,
		"success":  last.Success,
		"message":  last.Message,
	}
	msg := extractTraceMessage(trace)
	if strings.TrimSpace(msg) == "" {
		msg = err.Error()
	}
	raw, marshalErr := json.Marshal(trace)
	if marshalErr != nil {
		return errors.New(msg)
	}
	return fmt.Errorf("%s | http_trace=%s", msg, string(raw))
}

func extractTraceMessage(trace map[string]any) string {
	if trace == nil {
		return ""
	}
	if resp, ok := trace["response"].(map[string]any); ok {
		if bodyJSON, ok := resp["body_json"].(map[string]any); ok {
			if msg, ok := bodyJSON["msg"].(string); ok && strings.TrimSpace(msg) != "" {
				return msg
			}
		}
	}
	if msg, ok := trace["message"].(string); ok {
		return strings.TrimSpace(msg)
	}
	return ""
}

type automationServer struct {
	pluginv1.UnimplementedAutomationServiceServer
	core *coreServer
}

func (a *automationServer) ListAreas(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ListAreasResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var items []usecase.AutomationArea
	err = a.core.retry(func() error {
		var callErr error
		items, callErr = c.ListAreas(ctx)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationArea, 0, len(items))
	for _, it := range items {
		out = append(out, &pluginv1.AutomationArea{Id: it.ID, Name: it.Name, State: int32(it.State)})
	}
	return &pluginv1.ListAreasResponse{Items: out}, nil
}

func (a *automationServer) ListLines(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ListLinesResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var items []usecase.AutomationLine
	err = a.core.retry(func() error {
		var callErr error
		items, callErr = c.ListLines(ctx)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationLine, 0, len(items))
	for _, it := range items {
		out = append(out, &pluginv1.AutomationLine{Id: it.ID, Name: it.Name, AreaId: it.AreaID, State: int32(it.State)})
	}
	return &pluginv1.ListLinesResponse{Items: out}, nil
}

func (a *automationServer) ListPackages(ctx context.Context, req *pluginv1.ListPackagesRequest) (*pluginv1.ListPackagesResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var items []usecase.AutomationProduct
	err = a.core.retry(func() error {
		var callErr error
		items, callErr = c.ListProducts(ctx, req.GetLineId())
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationPackage, 0, len(items))
	for _, it := range items {
		out = append(out, &pluginv1.AutomationPackage{
			Id:            it.ID,
			Name:          it.Name,
			Cpu:           int32(it.CPU),
			MemoryGb:      int32(it.MemoryGB),
			DiskGb:        int32(it.DiskGB),
			BandwidthMbps: int32(it.Bandwidth),
			PortNum:       int32(it.PortNum),
			MonthlyPrice:  it.Price,
		})
	}
	return &pluginv1.ListPackagesResponse{Items: out}, nil
}

func (a *automationServer) ListImages(ctx context.Context, req *pluginv1.ListImagesRequest) (*pluginv1.ListImagesResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var items []usecase.AutomationImage
	err = a.core.retry(func() error {
		var callErr error
		items, callErr = c.ListImages(ctx, req.GetLineId())
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationImage, 0, len(items))
	for _, it := range items {
		out = append(out, &pluginv1.AutomationImage{Id: it.ImageID, Name: it.Name, Type: it.Type})
	}
	return &pluginv1.ListImagesResponse{Items: out}, nil
}

func (a *automationServer) CreateInstance(ctx context.Context, req *pluginv1.CreateInstanceRequest) (*pluginv1.CreateInstanceResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.CreateInstanceResponse{InstanceId: time.Now().Unix()}, nil
	}
	expire := time.Now().AddDate(0, 1, 0)
	if req.GetExpireAtUnix() > 0 {
		expire = time.Unix(req.GetExpireAtUnix(), 0)
	}
	r := usecase.AutomationCreateHostRequest{
		LineID:     req.GetLineId(),
		OS:         req.GetOs(),
		CPU:        int(req.GetCpu()),
		MemoryGB:   int(req.GetMemoryGb()),
		DiskGB:     int(req.GetDiskGb()),
		Bandwidth:  int(req.GetBandwidthMbps()),
		ExpireTime: expire,
		HostName:   req.GetName(),
		SysPwd:     req.GetPassword(),
		VNCPwd:     req.GetVncPassword(),
		PortNum:    int(req.GetPortNum()),
	}
	res, err := c.CreateHost(ctx, r)
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.CreateInstanceResponse{InstanceId: res.HostID}, nil
}

func (a *automationServer) GetInstance(ctx context.Context, req *pluginv1.GetInstanceRequest) (*pluginv1.GetInstanceResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var info usecase.AutomationHostInfo
	err = a.core.retry(func() error {
		var callErr error
		info, callErr = c.GetHostInfo(ctx, req.GetInstanceId())
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	expire := int64(0)
	if info.ExpireAt != nil {
		expire = info.ExpireAt.Unix()
	}
	return &pluginv1.GetInstanceResponse{
		Instance: &pluginv1.AutomationInstance{
			Id:            info.HostID,
			Name:          info.HostName,
			State:         int32(info.State),
			Cpu:           int32(info.CPU),
			MemoryGb:      int32(info.MemoryGB),
			DiskGb:        int32(info.DiskGB),
			BandwidthMbps: int32(info.Bandwidth),
			RemoteIp:      info.RemoteIP,
			PanelPassword: info.PanelPassword,
			VncPassword:   info.VNCPassword,
			OsPassword:    info.OSPassword,
			ExpireAtUnix:  expire,
		},
	}, nil
}

func (a *automationServer) ListInstancesSimple(ctx context.Context, req *pluginv1.ListInstancesSimpleRequest) (*pluginv1.ListInstancesSimpleResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var items []usecase.AutomationHostSimple
	err = a.core.retry(func() error {
		var callErr error
		items, callErr = c.ListHostSimple(ctx, req.GetSearchTag())
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationInstanceSimple, 0, len(items))
	for _, it := range items {
		out = append(out, &pluginv1.AutomationInstanceSimple{Id: it.ID, Name: it.HostName, Ip: it.IP})
	}
	return &pluginv1.ListInstancesSimpleResponse{Items: out}, nil
}

func (a *automationServer) Start(ctx context.Context, req *pluginv1.StartRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.StartHost(ctx, req.GetInstanceId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) Shutdown(ctx context.Context, req *pluginv1.ShutdownRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.ShutdownHost(ctx, req.GetInstanceId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) Reboot(ctx context.Context, req *pluginv1.RebootRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.RebootHost(ctx, req.GetInstanceId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) Rebuild(ctx context.Context, req *pluginv1.RebuildRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.ResetOS(ctx, req.GetInstanceId(), req.GetImageId(), req.GetPassword())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) ResetPassword(ctx context.Context, req *pluginv1.ResetPasswordRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.ResetOSPassword(ctx, req.GetInstanceId(), req.GetPassword())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) ElasticUpdate(ctx context.Context, req *pluginv1.ElasticUpdateRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	r := usecase.AutomationElasticUpdateRequest{HostID: req.GetInstanceId()}
	if req.Cpu != nil {
		v := int(req.GetCpu())
		r.CPU = &v
	}
	if req.MemoryGb != nil {
		v := int(req.GetMemoryGb())
		r.MemoryGB = &v
	}
	if req.DiskGb != nil {
		v := int(req.GetDiskGb())
		r.DiskGB = &v
	}
	if req.BandwidthMbps != nil {
		v := int(req.GetBandwidthMbps())
		r.Bandwidth = &v
	}
	if req.PortNum != nil {
		v := int(req.GetPortNum())
		r.PortNum = &v
	}
	err = c.ElasticUpdate(ctx, r)
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) Lock(ctx context.Context, req *pluginv1.LockRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.LockHost(ctx, req.GetInstanceId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) Unlock(ctx context.Context, req *pluginv1.UnlockRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.UnlockHost(ctx, req.GetInstanceId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) Renew(ctx context.Context, req *pluginv1.RenewRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	next := time.Unix(req.GetNextDueAtUnix(), 0)
	err = c.RenewHost(ctx, req.GetInstanceId(), next)
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) Destroy(ctx context.Context, req *pluginv1.DestroyRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.DeleteHost(ctx, req.GetInstanceId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) GetPanelURL(ctx context.Context, req *pluginv1.GetPanelURLRequest) (*pluginv1.GetPanelURLResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var url string
	err = a.core.retry(func() error {
		var callErr error
		url, callErr = c.GetPanelURL(ctx, req.GetInstanceName(), req.GetPanelPassword())
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.GetPanelURLResponse{Url: url}, nil
}

func (a *automationServer) GetVNCURL(ctx context.Context, req *pluginv1.GetVNCURLRequest) (*pluginv1.GetVNCURLResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var url string
	err = a.core.retry(func() error {
		var callErr error
		url, callErr = c.GetVNCURL(ctx, req.GetInstanceId())
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.GetVNCURLResponse{Url: url}, nil
}

func (a *automationServer) GetMonitor(ctx context.Context, req *pluginv1.GetMonitorRequest) (*pluginv1.GetMonitorResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var mon usecase.AutomationMonitor
	err = a.core.retry(func() error {
		var callErr error
		mon, callErr = c.GetMonitor(ctx, req.GetInstanceId())
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	raw := map[string]any{
		"CpuStats":     float64(mon.CPUPercent),
		"MemoryStats":  float64(mon.MemoryPercent),
		"StorageStats": float64(mon.StoragePercent),
		"NetworkStats": map[string]any{
			"BytesSentPersec":     mon.BytesOut,
			"BytesReceivedPersec": mon.BytesIn,
		},
	}
	b, _ := json.Marshal(raw)
	return &pluginv1.GetMonitorResponse{RawJson: string(b)}, nil
}

func (a *automationServer) ListPortMappings(ctx context.Context, req *pluginv1.ListPortMappingsRequest) (*pluginv1.ListPortMappingsResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var items []usecase.AutomationPortMapping
	err = a.core.retry(func() error {
		var callErr error
		items, callErr = c.ListPortMappings(ctx, req.GetInstanceId())
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationPortMapping, 0, len(items))
	for _, it := range items {
		id := toInt64(it["id"])
		if id == 0 {
			id = toInt64(it["ID"])
		}
		out = append(out, &pluginv1.AutomationPortMapping{
			Id:    id,
			Name:  toString(it["name"]),
			Sport: toString(it["sport"]),
			Dport: toInt64(it["dport"]),
		})
	}
	return &pluginv1.ListPortMappingsResponse{Items: out}, nil
}

func (a *automationServer) AddPortMapping(ctx context.Context, req *pluginv1.AddPortMappingRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.AddPortMapping(ctx, usecase.AutomationPortMappingCreate{
		HostID: req.GetInstanceId(),
		Name:   req.GetName(),
		Sport:  req.GetSport(),
		Dport:  req.GetDport(),
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) DeletePortMapping(ctx context.Context, req *pluginv1.DeletePortMappingRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.DeletePortMapping(ctx, req.GetInstanceId(), req.GetMappingId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) FindPortCandidates(ctx context.Context, req *pluginv1.FindPortCandidatesRequest) (*pluginv1.FindPortCandidatesResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var ports []int64
	err = a.core.retry(func() error {
		var callErr error
		ports, callErr = c.FindPortCandidates(ctx, req.GetInstanceId(), req.GetKeywords())
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.FindPortCandidatesResponse{Ports: ports}, nil
}

func (a *automationServer) ListBackups(ctx context.Context, req *pluginv1.ListBackupsRequest) (*pluginv1.ListBackupsResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var items []usecase.AutomationBackup
	err = a.core.retry(func() error {
		var callErr error
		items, callErr = c.ListBackups(ctx, req.GetInstanceId())
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationBackup, 0, len(items))
	for _, it := range items {
		out = append(out, &pluginv1.AutomationBackup{
			Id:            toInt64(it["id"]),
			Name:          fmt.Sprint(it["name"]),
			CreatedAtUnix: toTimeUnix(it["created_at"], it["created_at_unix"]),
		})
	}
	return &pluginv1.ListBackupsResponse{Items: out}, nil
}

func (a *automationServer) CreateBackup(ctx context.Context, req *pluginv1.CreateBackupRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.CreateBackup(ctx, req.GetInstanceId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) DeleteBackup(ctx context.Context, req *pluginv1.DeleteBackupRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.DeleteBackup(ctx, req.GetInstanceId(), req.GetBackupId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) RestoreBackup(ctx context.Context, req *pluginv1.RestoreBackupRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.RestoreBackup(ctx, req.GetInstanceId(), req.GetBackupId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) ListSnapshots(ctx context.Context, req *pluginv1.ListSnapshotsRequest) (*pluginv1.ListSnapshotsResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var items []usecase.AutomationSnapshot
	err = a.core.retry(func() error {
		var callErr error
		items, callErr = c.ListSnapshots(ctx, req.GetInstanceId())
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationSnapshot, 0, len(items))
	for _, it := range items {
		out = append(out, &pluginv1.AutomationSnapshot{
			Id:            toInt64(it["id"]),
			Name:          fmt.Sprint(it["name"]),
			CreatedAtUnix: toTimeUnix(it["created_at"], it["created_at_unix"]),
		})
	}
	return &pluginv1.ListSnapshotsResponse{Items: out}, nil
}

func (a *automationServer) CreateSnapshot(ctx context.Context, req *pluginv1.CreateSnapshotRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.CreateSnapshot(ctx, req.GetInstanceId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) DeleteSnapshot(ctx context.Context, req *pluginv1.DeleteSnapshotRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.DeleteSnapshot(ctx, req.GetInstanceId(), req.GetSnapshotId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) RestoreSnapshot(ctx context.Context, req *pluginv1.RestoreSnapshotRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.RestoreSnapshot(ctx, req.GetInstanceId(), req.GetSnapshotId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) ListFirewallRules(ctx context.Context, req *pluginv1.ListFirewallRulesRequest) (*pluginv1.ListFirewallRulesResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var items []usecase.AutomationFirewallRule
	err = a.core.retry(func() error {
		var callErr error
		items, callErr = c.ListFirewallRules(ctx, req.GetInstanceId())
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationFirewallRule, 0, len(items))
	for _, it := range items {
		out = append(out, &pluginv1.AutomationFirewallRule{
			Id:        toInt64(it["id"]),
			Direction: toString(it["direction"]),
			Protocol:  toString(it["protocol"]),
			Method:    toString(it["method"]),
			Port:      toString(it["port"]),
			Ip:        toString(it["ip"]),
		})
	}
	return &pluginv1.ListFirewallRulesResponse{Items: out}, nil
}

func (a *automationServer) AddFirewallRule(ctx context.Context, req *pluginv1.AddFirewallRuleRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.AddFirewallRule(ctx, usecase.AutomationFirewallRuleCreate{
		HostID:    req.GetInstanceId(),
		Direction: req.GetDirection(),
		Protocol:  req.GetProtocol(),
		Method:    req.GetMethod(),
		Port:      req.GetPort(),
		IP:        req.GetIp(),
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func (a *automationServer) DeleteFirewallRule(ctx context.Context, req *pluginv1.DeleteFirewallRuleRequest) (*pluginv1.Empty, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	if a.core.cfg.DryRun {
		return &pluginv1.Empty{}, nil
	}
	err = c.DeleteFirewallRule(ctx, req.GetInstanceId(), req.GetRuleId())
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.Empty{}, nil
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case json.Number:
		i, _ := t.Int64()
		return i
	default:
		i, _ := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(v)), 10, 64)
		return i
	}
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		if t == "<nil>" || t == "null" {
			return ""
		}
		return t
	case []byte:
		return strings.TrimSpace(string(t))
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "<nil>" || s == "null" {
			return ""
		}
		return s
	}
}

func toTimeUnix(v1 any, v2 any) int64 {
	if v := toInt64(v2); v > 0 {
		// support milliseconds
		if v > 1_000_000_000_000 {
			return v / 1000
		}
		return v
	}
	s := strings.TrimSpace(fmt.Sprint(v1))
	if s == "" || s == "<nil>" {
		return 0
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil && i > 0 {
		if i > 1_000_000_000_000 {
			return i / 1000
		}
		return i
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Unix()
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.Unix()
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Unix()
	}
	return 0
}

func main() {
	core := &coreServer{}
	auto := &automationServer{core: core}
	pluginsdk.Serve(map[string]pluginsdk.Plugin{
		pluginsdk.PluginKeyCore:       &pluginsdk.CoreGRPCPlugin{Impl: core},
		pluginsdk.PluginKeyAutomation: &pluginsdk.AutomationGRPCPlugin{Impl: auto},
	})
}
