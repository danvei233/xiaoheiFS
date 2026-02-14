package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

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

type ScheduledTaskUpdate struct {
	Enabled     *bool        `json:"enabled"`
	Strategy    TaskStrategy `json:"strategy"`
	IntervalSec *int         `json:"interval_sec"`
	DailyAt     *string      `json:"daily_at"`
}

type taskRuntime struct {
	lastRun     time.Time
	running     bool
	lastStatus  string
	lastError   string
	lastElapsed int
}

type ScheduledTaskService struct {
	settings SettingsRepository
	vps      *VPSService
	orders   *OrderService
	notify   *NotificationService
	realname *RealNameService
	runs     ScheduledTaskRunRepository
	mu       sync.Mutex
	runtime  map[string]*taskRuntime
}

func NewScheduledTaskService(settings SettingsRepository, vps *VPSService, orders *OrderService, notify *NotificationService, runs ScheduledTaskRunRepository, realname ...*RealNameService) *ScheduledTaskService {
	var rn *RealNameService
	if len(realname) > 0 {
		rn = realname[0]
	}
	return &ScheduledTaskService{
		settings: settings,
		vps:      vps,
		orders:   orders,
		notify:   notify,
		realname: rn,
		runs:     runs,
		runtime:  make(map[string]*taskRuntime),
	}
}

func (s *ScheduledTaskService) Start(ctx context.Context) {
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

func (s *ScheduledTaskService) ListTasks(ctx context.Context) ([]ScheduledTaskConfig, error) {
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

func (s *ScheduledTaskService) ListTaskRuns(ctx context.Context, key string, limit int) ([]domain.ScheduledTaskRun, error) {
	if s.runs == nil {
		return nil, ErrInvalidInput
	}
	return s.runs.ListTaskRuns(ctx, key, limit)
}

func (s *ScheduledTaskService) UpdateTask(ctx context.Context, key string, input ScheduledTaskUpdate) (ScheduledTaskConfig, error) {
	defs := defaultTaskDefinitions()
	def, ok := defs[key]
	if !ok {
		return ScheduledTaskConfig{}, ErrNotFound
	}
	cfg := s.loadTaskConfig(ctx, def)
	if input.Enabled != nil {
		cfg.Enabled = *input.Enabled
	}
	if input.Strategy != "" {
		cfg.Strategy = input.Strategy
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
		return ScheduledTaskConfig{}, ErrInvalidInput
	}
	raw := mustJSON(map[string]any{
		"enabled":      cfg.Enabled,
		"strategy":     cfg.Strategy,
		"interval_sec": cfg.IntervalSec,
		"daily_at":     cfg.DailyAt,
	})
	if err := s.settings.UpsertSetting(ctx, domain.Setting{Key: taskSettingKey(cfg.Key), ValueJSON: raw, UpdatedAt: time.Now()}); err != nil {
		return ScheduledTaskConfig{}, err
	}
	return cfg, nil
}

func (s *ScheduledTaskService) runOnce(ctx context.Context) {
	defs := defaultTaskDefinitions()
	for _, def := range defs {
		cfg := s.loadTaskConfig(ctx, def)
		if !cfg.Enabled {
			continue
		}
		if !s.shouldRun(cfg) {
			continue
		}
		s.executeTask(ctx, cfg)
	}
}

func (s *ScheduledTaskService) shouldRun(cfg ScheduledTaskConfig) bool {
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

func (s *ScheduledTaskService) executeTask(ctx context.Context, cfg ScheduledTaskConfig) {
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
		run := &domain.ScheduledTaskRun{
			TaskKey:   cfg.Key,
			Status:    "running",
			StartedAt: start,
		}
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
		case "realname_mangzhu_poll":
			if s.realname != nil {
				_, runErr = s.realname.PollPending(ctx, 200)
			}
		}
	}()
}

func (s *ScheduledTaskService) ensureRuntime(key string) *taskRuntime {
	if rt, ok := s.runtime[key]; ok {
		return rt
	}
	rt := &taskRuntime{}
	s.runtime[key] = rt
	return rt
}

func (s *ScheduledTaskService) loadTaskConfig(ctx context.Context, def ScheduledTaskConfig) ScheduledTaskConfig {
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

func (s *ScheduledTaskService) computeRunTimes(cfg ScheduledTaskConfig) (*time.Time, *time.Time) {
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
			return ErrInvalidInput
		}
		return nil
	case TaskStrategyDaily:
		if cfg.DailyAt == "" {
			return ErrInvalidInput
		}
		_, err := time.Parse("15:04", cfg.DailyAt)
		return err
	default:
		return errors.New("invalid strategy")
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
		"realname_mangzhu_poll": {
			Key:         "realname_mangzhu_poll",
			Name:        "Realname Mangzhu Poll",
			Description: "Poll pending Mangzhu face verification records and update status.",
			Enabled:     true,
			Strategy:    TaskStrategyInterval,
			IntervalSec: 20,
		},
	}
}
