package scheduledtask

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type TaskStrategy string

const (
	TaskStrategyInterval TaskStrategy = "interval"
	TaskStrategyDaily    TaskStrategy = "daily"
)

type ScheduledTaskConfig struct {
	Key         string       `json:"key"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Enabled     bool         `json:"enabled"`
	Strategy    TaskStrategy `json:"strategy"`
	IntervalSec int          `json:"interval_sec"`
	DailyAt     string       `json:"daily_at"`
	LastRunAt   *time.Time   `json:"last_run_at,omitempty"`
	NextRunAt   *time.Time   `json:"next_run_at,omitempty"`
	Running     bool         `json:"running"`
	LastStatus  string       `json:"last_status,omitempty"`
	LastError   string       `json:"last_error,omitempty"`
	LastElapsed int          `json:"last_elapsed_sec,omitempty"`
}

type ScheduledTaskUpdate = appshared.ScheduledTaskUpdate

type vpsTaskService interface {
	RefreshAll(ctx context.Context, limit int) (int, error)
	AutoDeleteExpired(ctx context.Context) error
	AutoLockExpired(ctx context.Context) error
}

type orderTaskService interface {
	ReconcileProvisioningOrders(ctx context.Context, limit int) (int, error)
	ProcessProvisionJobs(ctx context.Context, limit int) error
	ProcessResizeTasks(ctx context.Context, limit int) error
}

type notificationTaskService interface {
	SendExpireReminders(ctx context.Context) error
}

type realnameTaskService interface {
	PollPending(ctx context.Context, limit int) (int, error)
}

type userTierTaskService interface {
	ReconcileExpired(ctx context.Context, limit int) (int, error)
}

type integrationInventorySyncService interface {
	SyncAutomationInventoryForGoodsType(ctx context.Context, goodsTypeID int64) (int, error)
}

type logRetentionCleaner interface {
	Cleanup(ctx context.Context) (string, error)
}

type taskRuntime struct {
	lastRun     time.Time
	running     bool
	lastStatus  string
	lastError   string
	lastElapsed int
}

type Service struct {
	settings    appports.SettingsRepository
	vps         vpsTaskService
	orders      orderTaskService
	notify      notificationTaskService
	realname    realnameTaskService
	userTier    userTierTaskService
	integration integrationInventorySyncService
	logCleaner  logRetentionCleaner
	runs        appports.ScheduledTaskRunRepository
	mu          sync.Mutex
	runtime     map[string]*taskRuntime
}

func NewService(settings appports.SettingsRepository, vps vpsTaskService, orders orderTaskService, notify notificationTaskService, runs appports.ScheduledTaskRunRepository, realname ...realnameTaskService) *Service {
	var rn realnameTaskService
	if len(realname) > 0 {
		rn = realname[0]
	}
	return &Service{
		settings: settings,
		vps:      vps,
		orders:   orders,
		notify:   notify,
		realname: rn,
		runs:     runs,
		runtime:  make(map[string]*taskRuntime),
	}
}

func (s *Service) SetUserTierService(svc userTierTaskService) {
	s.userTier = svc
}

func (s *Service) SetIntegrationService(svc integrationInventorySyncService) {
	s.integration = svc
}

func (s *Service) SetLogRetentionCleaner(svc logRetentionCleaner) {
	s.logCleaner = svc
}

func (s *Service) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		s.runOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) ListTasks(ctx context.Context) ([]ScheduledTaskConfig, error) {
	defs := defaultTaskDefinitions()
	out := make([]ScheduledTaskConfig, 0, len(defs))
	for _, def := range defs {
		cfg := s.loadTaskConfig(ctx, def)
		last, next := s.computeRunTimes(cfg)
		s.mu.Lock()
		rt := s.ensureRuntime(cfg.Key)
		cfg.Running = rt.running
		cfg.LastStatus = rt.lastStatus
		cfg.LastError = rt.lastError
		cfg.LastElapsed = rt.lastElapsed
		s.mu.Unlock()
		cfg.LastRunAt = last
		cfg.NextRunAt = next
		out = append(out, cfg)
	}
	return out, nil
}

func (s *Service) ListTaskRuns(ctx context.Context, key string, limit int) ([]domain.ScheduledTaskRun, error) {
	if s.runs == nil {
		return nil, appshared.ErrInvalidInput
	}
	return s.runs.ListTaskRuns(ctx, key, limit)
}

func (s *Service) UpdateTask(ctx context.Context, key string, input ScheduledTaskUpdate) (ScheduledTaskConfig, error) {
	defs := defaultTaskDefinitions()
	def, ok := defs[key]
	if !ok {
		return ScheduledTaskConfig{}, appshared.ErrNotFound
	}
	cfg := s.loadTaskConfig(ctx, def)
	if input.Enabled != nil {
		cfg.Enabled = *input.Enabled
	}
	if input.Strategy != "" {
		cfg.Strategy = TaskStrategy(input.Strategy)
	}
	if input.IntervalSec != nil {
		cfg.IntervalSec = *input.IntervalSec
	}
	if input.DailyAt != nil {
		cfg.DailyAt = *input.DailyAt
	}
	if err := validateTaskConfig(cfg); err != nil {
		return ScheduledTaskConfig{}, err
	}
	if s.settings == nil {
		return ScheduledTaskConfig{}, appshared.ErrInvalidInput
	}
	raw, _ := json.Marshal(map[string]any{
		"enabled":      cfg.Enabled,
		"strategy":     cfg.Strategy,
		"interval_sec": cfg.IntervalSec,
		"daily_at":     cfg.DailyAt,
	})
	if err := s.settings.UpsertSetting(ctx, domain.Setting{Key: taskSettingKey(cfg.Key), ValueJSON: string(raw), UpdatedAt: time.Now()}); err != nil {
		return ScheduledTaskConfig{}, err
	}
	return cfg, nil
}

func (s *Service) runOnce(ctx context.Context) {
	defs := defaultTaskDefinitions()
	for _, def := range defs {
		cfg := s.loadTaskConfig(ctx, def)
		if !cfg.Enabled || !s.shouldRun(cfg) {
			continue
		}
		s.executeTask(ctx, cfg)
	}
}

func (s *Service) shouldRun(cfg ScheduledTaskConfig) bool {
	now := time.Now()
	s.mu.Lock()
	rt := s.ensureRuntime(cfg.Key)
	last := rt.lastRun
	s.mu.Unlock()
	if cfg.Strategy == TaskStrategyDaily {
		return shouldRunDaily(now, last, cfg.DailyAt)
	}
	interval := time.Duration(cfg.IntervalSec) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}
	if last.IsZero() {
		return true
	}
	return now.Sub(last) >= interval
}

func (s *Service) executeTask(ctx context.Context, cfg ScheduledTaskConfig) {
	s.mu.Lock()
	rt := s.ensureRuntime(cfg.Key)
	if rt.running {
		s.mu.Unlock()
		return
	}
	rt.running = true
	s.mu.Unlock()

	go func() {
		start := time.Now()
		run := &domain.ScheduledTaskRun{TaskKey: cfg.Key, Status: "running", StartedAt: start}
		if s.runs != nil {
			_ = s.runs.CreateTaskRun(ctx, run)
		}
		var runErr error
		defer func() {
			elapsed := int(time.Since(start).Seconds())
			if elapsed < 0 {
				elapsed = 0
			}
			s.mu.Lock()
			rt.running = false
			rt.lastRun = time.Now()
			rt.lastElapsed = elapsed
			s.mu.Unlock()
			status := "success"
			msg := ""
			if runErr != nil {
				status = "failed"
				msg = runErr.Error()
			}
			s.mu.Lock()
			rt.lastStatus = status
			rt.lastError = msg
			s.mu.Unlock()
			if s.runs != nil {
				finish := time.Now()
				run.Status = status
				run.Message = msg
				run.FinishedAt = &finish
				run.DurationSec = elapsed
				_ = s.runs.UpdateTaskRun(ctx, *run)
			}
		}()
		switch cfg.Key {
		case "vps_refresh":
			if s.vps != nil {
				_, runErr = s.vps.RefreshAll(ctx, 200)
				if runErr == nil && s.orders != nil {
					_, _ = s.orders.ReconcileProvisioningOrders(ctx, 50)
				}
			}
		case "order_provision_watchdog":
			if s.orders != nil {
				runErr = s.orders.ProcessProvisionJobs(ctx, 50)
			}
		case "resize_task_runner":
			if s.orders != nil {
				runErr = s.orders.ProcessResizeTasks(ctx, 50)
			}
		case "expire_reminder":
			if s.notify != nil {
				runErr = s.notify.SendExpireReminders(ctx)
			}
		case "vps_expire_cleanup":
			if s.vps != nil {
				runErr = s.vps.AutoDeleteExpired(ctx)
			}
		case "vps_expire_lock":
			if s.vps != nil {
				runErr = s.vps.AutoLockExpired(ctx)
			}
		case "plugin_schedule":
			if s.realname != nil {
				_, runErr = s.realname.PollPending(ctx, 200)
			}
		case "user_tier_expire_reconcile":
			if s.userTier != nil {
				_, runErr = s.userTier.ReconcileExpired(ctx, 500)
			}
		case "integration_inventory_sync":
			if s.integration != nil {
				_, runErr = s.integration.SyncAutomationInventoryForGoodsType(ctx, 0)
			}
		case "log_retention_cleanup":
			if s.logCleaner != nil {
				_, runErr = s.logCleaner.Cleanup(ctx)
			}
		}
	}()
}

func (s *Service) ensureRuntime(key string) *taskRuntime {
	if rt, ok := s.runtime[key]; ok {
		return rt
	}
	rt := &taskRuntime{}
	s.runtime[key] = rt
	return rt
}

func (s *Service) loadTaskConfig(ctx context.Context, def ScheduledTaskConfig) ScheduledTaskConfig {
	cfg := def
	if s.settings == nil {
		return cfg
	}
	setting, err := s.settings.GetSetting(ctx, taskSettingKey(def.Key))
	if err != nil || setting.ValueJSON == "" {
		return cfg
	}
	var raw struct {
		Enabled     *bool   `json:"enabled"`
		Strategy    *string `json:"strategy"`
		IntervalSec *int    `json:"interval_sec"`
		DailyAt     *string `json:"daily_at"`
	}
	if err := json.Unmarshal([]byte(setting.ValueJSON), &raw); err != nil {
		return cfg
	}
	if raw.Enabled != nil {
		cfg.Enabled = *raw.Enabled
	}
	if raw.Strategy != nil {
		cfg.Strategy = TaskStrategy(*raw.Strategy)
	}
	if raw.IntervalSec != nil {
		cfg.IntervalSec = *raw.IntervalSec
	}
	if raw.DailyAt != nil {
		cfg.DailyAt = *raw.DailyAt
	}
	return cfg
}

func (s *Service) computeRunTimes(cfg ScheduledTaskConfig) (*time.Time, *time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rt := s.ensureRuntime(cfg.Key)
	var last *time.Time
	var next *time.Time
	if !rt.lastRun.IsZero() {
		t := rt.lastRun
		last = &t
	}
	now := time.Now()
	if cfg.Strategy == TaskStrategyDaily {
		n := nextDailyRun(now, rt.lastRun, cfg.DailyAt)
		if !n.IsZero() {
			next = &n
		}
		return last, next
	}
	interval := time.Duration(cfg.IntervalSec) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}
	if rt.lastRun.IsZero() {
		n := now
		next = &n
	} else {
		n := rt.lastRun.Add(interval)
		next = &n
	}
	return last, next
}

func validateTaskConfig(cfg ScheduledTaskConfig) error {
	switch cfg.Strategy {
	case "", TaskStrategyInterval:
		if cfg.IntervalSec <= 0 {
			return appshared.ErrInvalidInput
		}
		return nil
	case TaskStrategyDaily:
		if cfg.DailyAt == "" {
			return appshared.ErrInvalidInput
		}
		_, err := time.Parse("15:04", cfg.DailyAt)
		return err
	default:
		return domain.ErrInvalidStrategy
	}
}

func shouldRunDaily(now, lastRun time.Time, dailyAt string) bool {
	hour, minute, ok := parseDailyAt(dailyAt)
	if !ok {
		return false
	}
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if now.Before(target) {
		return false
	}
	if lastRun.IsZero() {
		return true
	}
	return lastRun.Before(target)
}

func nextDailyRun(now, lastRun time.Time, dailyAt string) time.Time {
	hour, minute, ok := parseDailyAt(dailyAt)
	if !ok {
		return time.Time{}
	}
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if lastRun.IsZero() {
		if now.Before(target) {
			return target
		}
		return target.Add(24 * time.Hour)
	}
	if lastRun.Before(target) && now.Before(target) {
		return target
	}
	return target.Add(24 * time.Hour)
}

func parseDailyAt(value string) (int, int, bool) {
	if value == "" {
		return 0, 0, false
	}
	t, err := time.Parse("15:04", value)
	if err != nil {
		return 0, 0, false
	}
	return t.Hour(), t.Minute(), true
}

func taskSettingKey(key string) string {
	return "task." + key
}

func defaultTaskDefinitions() map[string]ScheduledTaskConfig {
	return map[string]ScheduledTaskConfig{
		"vps_refresh": {
			Key:         "vps_refresh",
			Name:        "VPS Auto Refresh",
			Description: "Periodically refresh automation data: status, expiry, and access info.",
			Enabled:     true,
			Strategy:    TaskStrategyInterval,
			IntervalSec: 300,
		},
		"order_provision_watchdog": {
			Key:         "order_provision_watchdog",
			Name:        "Order Provision Watchdog",
			Description: "Poll provision jobs and advance order status.",
			Enabled:     true,
			Strategy:    TaskStrategyInterval,
			IntervalSec: 5,
		},
		"resize_task_runner": {
			Key:         "resize_task_runner",
			Name:        "Resize Task Runner",
			Description: "Process pending resize tasks.",
			Enabled:     true,
			Strategy:    TaskStrategyInterval,
			IntervalSec: 30,
		},
		"expire_reminder": {
			Key:         "expire_reminder",
			Name:        "Expire Reminder",
			Description: "Send daily expiry reminders via email/notifications.",
			Enabled:     true,
			Strategy:    TaskStrategyDaily,
			DailyAt:     "09:00",
		},
		"vps_expire_cleanup": {
			Key:         "vps_expire_cleanup",
			Name:        "VPS Expire Cleanup",
			Description: "Auto delete expired VPS instances based on lifecycle settings.",
			Enabled:     true,
			Strategy:    TaskStrategyDaily,
			DailyAt:     "03:00",
		},
		"vps_expire_lock": {
			Key:         "vps_expire_lock",
			Name:        "VPS Expire Lock",
			Description: "Auto lock expired VPS instances that are not locked yet.",
			Enabled:     true,
			Strategy:    TaskStrategyInterval,
			IntervalSec: 300,
		},
		"plugin_schedule": {
			Key:         "plugin_schedule",
			Name:        "Plugin Schedule",
			Description: "Process scheduled tasks registered by plugins.",
			Enabled:     true,
			Strategy:    TaskStrategyInterval,
			IntervalSec: 20,
		},
		"user_tier_expire_reconcile": {
			Key:         "user_tier_expire_reconcile",
			Name:        "User Tier Expire Reconcile",
			Description: "Reconcile expired user tier memberships and re-run auto approval.",
			Enabled:     true,
			Strategy:    TaskStrategyInterval,
			IntervalSec: 60,
		},
		"integration_inventory_sync": {
			Key:         "integration_inventory_sync",
			Name:        "Integration Inventory Sync",
			Description: "Sync package inventory only, without touching structure or pricing.",
			Enabled:     false,
			Strategy:    TaskStrategyInterval,
			IntervalSec: 300,
		},
		"log_retention_cleanup": {
			Key:         "log_retention_cleanup",
			Name:        "Log Retention Cleanup",
			Description: "Purge expired logs by retention settings.",
			Enabled:     true,
			Strategy:    TaskStrategyDaily,
			DailyAt:     "03:30",
		},
	}
}
