package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/usecase"
)

type DynamicClient struct {
	settings       usecase.SettingsRepository
	fallbackBase   string
	fallbackAPIKey string
	autoLogs       usecase.AutomationLogRepository
}

type dynamicConfig struct {
	baseURL       string
	apiKey        string
	enabled       bool
	dryRun        bool
	timeout       time.Duration
	retry         int
	debugEnabled  bool
	retentionDays int
}

func NewDynamicClient(settings usecase.SettingsRepository, fallbackBase, fallbackAPIKey string, autoLogs usecase.AutomationLogRepository) *DynamicClient {
	return &DynamicClient{settings: settings, fallbackBase: fallbackBase, fallbackAPIKey: fallbackAPIKey, autoLogs: autoLogs}
}

func (d *DynamicClient) CreateHost(ctx context.Context, req usecase.AutomationCreateHostRequest) (usecase.AutomationCreateHostResult, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return usecase.AutomationCreateHostResult{}, fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return usecase.AutomationCreateHostResult{HostID: time.Now().Unix()}, nil
	}
	client := d.newClient(cfg)
	var out usecase.AutomationCreateHostResult
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.CreateHost(ctx, req)
		return err
	})
	return out, err
}

func (d *DynamicClient) GetHostInfo(ctx context.Context, hostID int64) (usecase.AutomationHostInfo, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return usecase.AutomationHostInfo{}, fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return usecase.AutomationHostInfo{HostID: hostID, HostName: fmt.Sprintf("dry-%d", hostID), State: 1}, nil
	}
	client := d.newClient(cfg)
	var out usecase.AutomationHostInfo
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.GetHostInfo(ctx, hostID)
		return err
	})
	return out, err
}

func (d *DynamicClient) ListHostSimple(ctx context.Context, searchTag string) ([]usecase.AutomationHostSimple, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return nil, fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return []usecase.AutomationHostSimple{}, nil
	}
	client := d.newClient(cfg)
	var out []usecase.AutomationHostSimple
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.ListHostSimple(ctx, searchTag)
		return err
	})
	return out, err
}

func (d *DynamicClient) ElasticUpdate(ctx context.Context, req usecase.AutomationElasticUpdateRequest) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.ElasticUpdate(ctx, req) })
}

func (d *DynamicClient) RenewHost(ctx context.Context, hostID int64, nextDueDate time.Time) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.RenewHost(ctx, hostID, nextDueDate) })
}

func (d *DynamicClient) LockHost(ctx context.Context, hostID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.LockHost(ctx, hostID) })
}

func (d *DynamicClient) UnlockHost(ctx context.Context, hostID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.UnlockHost(ctx, hostID) })
}

func (d *DynamicClient) DeleteHost(ctx context.Context, hostID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.DeleteHost(ctx, hostID) })
}

func (d *DynamicClient) StartHost(ctx context.Context, hostID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.StartHost(ctx, hostID) })
}

func (d *DynamicClient) ShutdownHost(ctx context.Context, hostID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.ShutdownHost(ctx, hostID) })
}

func (d *DynamicClient) RebootHost(ctx context.Context, hostID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.RebootHost(ctx, hostID) })
}

func (d *DynamicClient) ResetOS(ctx context.Context, hostID int64, templateID int64, password string) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.ResetOS(ctx, hostID, templateID, password) })
}

func (d *DynamicClient) ResetOSPassword(ctx context.Context, hostID int64, password string) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.ResetOSPassword(ctx, hostID, password) })
}

func (d *DynamicClient) ListSnapshots(ctx context.Context, hostID int64) ([]usecase.AutomationSnapshot, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return nil, fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return []usecase.AutomationSnapshot{}, nil
	}
	client := d.newClient(cfg)
	var out []usecase.AutomationSnapshot
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.ListSnapshots(ctx, hostID)
		return err
	})
	return out, err
}

func (d *DynamicClient) CreateSnapshot(ctx context.Context, hostID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.CreateSnapshot(ctx, hostID) })
}

func (d *DynamicClient) DeleteSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.DeleteSnapshot(ctx, hostID, snapshotID) })
}

func (d *DynamicClient) RestoreSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.RestoreSnapshot(ctx, hostID, snapshotID) })
}

func (d *DynamicClient) ListBackups(ctx context.Context, hostID int64) ([]usecase.AutomationBackup, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return nil, fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return []usecase.AutomationBackup{}, nil
	}
	client := d.newClient(cfg)
	var out []usecase.AutomationBackup
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.ListBackups(ctx, hostID)
		return err
	})
	return out, err
}

func (d *DynamicClient) CreateBackup(ctx context.Context, hostID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.CreateBackup(ctx, hostID) })
}

func (d *DynamicClient) DeleteBackup(ctx context.Context, hostID int64, backupID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.DeleteBackup(ctx, hostID, backupID) })
}

func (d *DynamicClient) RestoreBackup(ctx context.Context, hostID int64, backupID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.RestoreBackup(ctx, hostID, backupID) })
}

func (d *DynamicClient) ListFirewallRules(ctx context.Context, hostID int64) ([]usecase.AutomationFirewallRule, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return nil, fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return []usecase.AutomationFirewallRule{}, nil
	}
	client := d.newClient(cfg)
	var out []usecase.AutomationFirewallRule
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.ListFirewallRules(ctx, hostID)
		return err
	})
	return out, err
}

func (d *DynamicClient) AddFirewallRule(ctx context.Context, req usecase.AutomationFirewallRuleCreate) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.AddFirewallRule(ctx, req) })
}

func (d *DynamicClient) DeleteFirewallRule(ctx context.Context, hostID int64, ruleID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.DeleteFirewallRule(ctx, hostID, ruleID) })
}

func (d *DynamicClient) ListPortMappings(ctx context.Context, hostID int64) ([]usecase.AutomationPortMapping, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return nil, fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return []usecase.AutomationPortMapping{}, nil
	}
	client := d.newClient(cfg)
	var out []usecase.AutomationPortMapping
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.ListPortMappings(ctx, hostID)
		return err
	})
	return out, err
}

func (d *DynamicClient) AddPortMapping(ctx context.Context, req usecase.AutomationPortMappingCreate) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.AddPortMapping(ctx, req) })
}

func (d *DynamicClient) DeletePortMapping(ctx context.Context, hostID int64, mappingID int64) error {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return nil
	}
	client := d.newClient(cfg)
	return d.retry(cfg, func() error { return client.DeletePortMapping(ctx, hostID, mappingID) })
}

func (d *DynamicClient) FindPortCandidates(ctx context.Context, hostID int64, keywords string) ([]int64, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return nil, fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return []int64{}, nil
	}
	client := d.newClient(cfg)
	var out []int64
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.FindPortCandidates(ctx, hostID, keywords)
		return err
	})
	return out, err
}

func (d *DynamicClient) GetPanelURL(ctx context.Context, hostName string, panelPassword string) (string, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return "", fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return "", nil
	}
	client := d.newClient(cfg)
	var out string
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.GetPanelURL(ctx, hostName, panelPassword)
		return err
	})
	return out, err
}

func (d *DynamicClient) ListImages(ctx context.Context, lineID int64) ([]usecase.AutomationImage, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return nil, fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return []usecase.AutomationImage{}, nil
	}
	client := d.newClient(cfg)
	var out []usecase.AutomationImage
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.ListImages(ctx, lineID)
		return err
	})
	return out, err
}

func (d *DynamicClient) ListAreas(ctx context.Context) ([]usecase.AutomationArea, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return nil, fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return []usecase.AutomationArea{}, nil
	}
	client := d.newClient(cfg)
	var out []usecase.AutomationArea
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.ListAreas(ctx)
		return err
	})
	return out, err
}

func (d *DynamicClient) ListLines(ctx context.Context) ([]usecase.AutomationLine, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return nil, fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return []usecase.AutomationLine{}, nil
	}
	client := d.newClient(cfg)
	var out []usecase.AutomationLine
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.ListLines(ctx)
		return err
	})
	return out, err
}

func (d *DynamicClient) ListProducts(ctx context.Context, lineID int64) ([]usecase.AutomationProduct, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return nil, fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return []usecase.AutomationProduct{}, nil
	}
	client := d.newClient(cfg)
	var out []usecase.AutomationProduct
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.ListProducts(ctx, lineID)
		return err
	})
	return out, err
}

func (d *DynamicClient) GetMonitor(ctx context.Context, hostID int64) (usecase.AutomationMonitor, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return usecase.AutomationMonitor{}, fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return usecase.AutomationMonitor{}, nil
	}
	client := d.newClient(cfg)
	var out usecase.AutomationMonitor
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.GetMonitor(ctx, hostID)
		return err
	})
	return out, err
}

func (d *DynamicClient) GetVNCURL(ctx context.Context, hostID int64) (string, error) {
	cfg := d.loadConfig(ctx)
	if !cfg.enabled {
		return "", fmt.Errorf("automation disabled")
	}
	if cfg.dryRun {
		return "", nil
	}
	client := d.newClient(cfg)
	var out string
	err := d.retry(cfg, func() error {
		var err error
		out, err = client.GetVNCURL(ctx, hostID)
		return err
	})
	return out, err
}

func (d *DynamicClient) retry(cfg dynamicConfig, fn func() error) error {
	var err error
	for i := 0; i <= cfg.retry; i++ {
		err = fn()
		if err == nil {
			return nil
		}
	}
	return err
}

func (d *DynamicClient) newClient(cfg dynamicConfig) *Client {
	client := NewClient(cfg.baseURL, cfg.apiKey, cfg.timeout)
	if d.autoLogs == nil || !cfg.debugEnabled {
		return client
	}
	return client.WithLogger(func(ctx context.Context, entry httpLogEntry) {
		trace, _ := usecase.GetAutomationLogContext(ctx)
		if cfg.retentionDays > 0 {
			before := time.Now().AddDate(0, 0, -cfg.retentionDays)
			_ = d.autoLogs.PurgeAutomationLogs(ctx, before)
		}
		reqJSON, _ := json.Marshal(entry.Request)
		respJSON, _ := json.Marshal(entry.Response)
		logEntry := domain.AutomationLog{
			OrderID:      trace.OrderID,
			OrderItemID:  trace.OrderItemID,
			Action:       entry.Action,
			RequestJSON:  string(reqJSON),
			ResponseJSON: string(respJSON),
			Success:      entry.Success,
			Message:      entry.Message,
		}
		_ = d.autoLogs.CreateAutomationLog(ctx, &logEntry)
	})
}

func (d *DynamicClient) loadConfig(ctx context.Context) dynamicConfig {
	cfg := dynamicConfig{
		baseURL: d.fallbackBase,
		apiKey:  d.fallbackAPIKey,
		enabled: true,
		timeout: 12 * time.Second,
		retry:   0,
	}
	if d.settings == nil {
		return cfg
	}
	if setting, err := d.settings.GetSetting(ctx, "automation_base_url"); err == nil && setting.ValueJSON != "" {
		cfg.baseURL = setting.ValueJSON
	}
	if setting, err := d.settings.GetSetting(ctx, "automation_api_key"); err == nil && setting.ValueJSON != "" {
		cfg.apiKey = setting.ValueJSON
	}
	if setting, err := d.settings.GetSetting(ctx, "automation_enabled"); err == nil && setting.ValueJSON != "" {
		cfg.enabled = strings.ToLower(setting.ValueJSON) == "true"
	}
	if setting, err := d.settings.GetSetting(ctx, "automation_timeout_sec"); err == nil && setting.ValueJSON != "" {
		if v, err := strconv.Atoi(setting.ValueJSON); err == nil {
			cfg.timeout = time.Duration(v) * time.Second
		}
	}
	if setting, err := d.settings.GetSetting(ctx, "automation_retry"); err == nil && setting.ValueJSON != "" {
		if v, err := strconv.Atoi(setting.ValueJSON); err == nil {
			cfg.retry = v
		}
	}
	if setting, err := d.settings.GetSetting(ctx, "automation_dry_run"); err == nil && setting.ValueJSON != "" {
		cfg.dryRun = strings.ToLower(setting.ValueJSON) == "true"
	}
	if setting, err := d.settings.GetSetting(ctx, "debug_enabled"); err == nil && setting.ValueJSON != "" {
		cfg.debugEnabled = strings.ToLower(setting.ValueJSON) == "true"
	}
	if setting, err := d.settings.GetSetting(ctx, "automation_log_retention_days"); err == nil && setting.ValueJSON != "" {
		if v, err := strconv.Atoi(setting.ValueJSON); err == nil && v > 0 {
			cfg.retentionDays = v
		}
	}
	if cfg.baseURL == "" || cfg.apiKey == "" {
		cfg.enabled = false
	}
	return cfg
}
