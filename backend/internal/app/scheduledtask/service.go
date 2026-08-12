package scheduledtask

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/safego"
)

// taskLeaseDuration bounds the DB execution lease for a scheduled task. The
// lease is renewed by a heartbeat while the task runs, so it only matters on
// crash: after lease_until a crashed worker's lease can be reclaimed.
const taskLeaseDuration = 2 * time.Minute

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
	// leaseFor is the DB execution lease duration. It defaults to
	// taskLeaseDuration and is shortened in tests so lease-loss paths can be
	// exercised without waiting minutes.
	leaseFor time.Duration
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
		leaseFor: taskLeaseDuration,
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
	safego.Run("scheduled task state hydrate", func() { s.hydrateLastRuns(ctx) })
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		safego.Run("scheduled task coordinator tick", func() { s.runOnce(ctx) })
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// hydrateLastRuns restores the in-memory last-run timestamps from persisted
// runs so that after a restart daily tasks do not re-trigger on the same day.
// Runs left "running" by a crashed worker are ignored: their lease has expired
// (or will expire), so the task must be allowed to run again.
func (s *Service) hydrateLastRuns(ctx context.Context) {
	if s.runs == nil {
		return
	}
	for _, def := range defaultTaskDefinitions() {
		runs, err := s.runs.ListTaskRuns(ctx, def.Key, 1)
		if err != nil || len(runs) == 0 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(runs[0].Status), "running") {
			continue
		}
		last := runs[0].StartedAt
		s.mu.Lock()
		rt := s.ensureRuntime(def.Key)
		rt.lastRun = last
		rt.lastStatus = runs[0].Status
		rt.lastError = runs[0].Message
		s.mu.Unlock()
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
	interval := intervalDuration(cfg)
	if last.IsZero() {
		return true
	}
	return now.Sub(last) >= interval
}

// shouldRunUnderLease re-evaluates whether the task still needs to run in this
// period while the caller holds the DB execution lease, using the persisted run
// history rather than the process-local lastRun. It returns true when the task
// must run.
//
// A replica's in-memory lastRun is only hydrated at startup, so after another
// instance completes a run in this period the replica would still see a stale
// lastRun and want to run again. Because this check runs under the lease, the
// decision is serialized: exactly one instance reaches it per period. A run a
// crashed worker left "running" does not suppress a retry (its side effects
// were never completed). A query failure fails closed (returns false) so a
// task is never run twice because the history could not be read.
func (s *Service) shouldRunUnderLease(ctx context.Context, cfg ScheduledTaskConfig) (bool, error) {
	runs, err := s.runs.ListTaskRuns(ctx, cfg.Key, 1)
	if err != nil {
		return false, err
	}
	if len(runs) == 0 {
		return true, nil
	}
	last := runs[0]
	if strings.EqualFold(strings.TrimSpace(last.Status), "running") {
		return true, nil
	}
	now := time.Now()
	if cfg.Strategy == TaskStrategyDaily {
		hour, minute, ok := parseDailyAt(cfg.DailyAt)
		if !ok {
			return false, nil
		}
		target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
		// Skip when the latest completed run started at/after today's target.
		return last.StartedAt.Before(target), nil
	}
	interval := intervalDuration(cfg)
	return now.Sub(last.StartedAt) >= interval, nil
}

func (s *Service) executeTask(ctx context.Context, cfg ScheduledTaskConfig) {
	s.mu.Lock()
	rt := s.ensureRuntime(cfg.Key)
	if rt.running {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	// Multi-instance exclusivity: only the DB lease winner may run the task.
	// A nil runs repo falls back to process-local behaviour (tests / single node).
	var lease domain.ScheduledTaskLease
	var leaseAcquired bool
	if s.runs != nil {
		owner := s.instanceOwner()
		var err error
		lease, leaseAcquired, err = s.runs.AcquireTaskLease(ctx, cfg.Key, owner, s.leaseFor)
		if err != nil {
			return
		}
		if !leaseAcquired {
			return // another instance owns the lease
		}
		// Per-period dedup while holding the lease: the in-memory lastRun is
		// only hydrated at startup, so a replica that started before this period
		// would otherwise re-run a task another instance already completed in
		// the same period. Re-checking against the persisted run history under
		// the lease closes that window without a check-then-act race.
		needsRun, err := s.shouldRunUnderLease(ctx, cfg)
		if err != nil {
			_ = s.runs.ReleaseTaskLease(ctx, cfg.Key, s.instanceOwner(), lease.FencingToken)
			return
		}
		if !needsRun {
			_ = s.runs.ReleaseTaskLease(ctx, cfg.Key, s.instanceOwner(), lease.FencingToken)
			return
		}
	}

	s.mu.Lock()
	if rt.running {
		s.mu.Unlock()
		if s.runs != nil {
			_ = s.runs.ReleaseTaskLease(ctx, cfg.Key, s.instanceOwner(), lease.FencingToken)
		}
		return
	}
	rt.running = true
	s.mu.Unlock()

	safego.Go("scheduled task worker", func() {
		start := time.Now()
		run := &domain.ScheduledTaskRun{TaskKey: cfg.Key, Status: "running", StartedAt: start}
		if s.runs != nil {
			_ = s.runs.CreateTaskRun(ctx, run)
		}
		// The task runs on its own cancellable context. Heartbeat renewal keeps
		// the lease owned; when the lease is lost (another instance reclaimed it
		// after a crash, or the DB rejects a renew), the context is cancelled so
		// external side effects stop as soon as the task next observes the
		// context, instead of running to completion and only being blocked at the
		// fenced final status write. The run row is still written (and fenced)
		// below.
		taskCtx, cancelTask := context.WithCancel(ctx)
		defer cancelTask()
		stopHeartbeat := make(chan struct{})
		defer close(stopHeartbeat)
		if s.runs != nil {
			go func() {
				if safego.Run("scheduled task lease heartbeat", func() {
					ticker := time.NewTicker(s.leaseFor / 3)
					defer ticker.Stop()
					for {
						select {
						case <-stopHeartbeat:
							return
						case <-ticker.C:
							ok, _ := s.runs.RenewTaskLease(ctx, cfg.Key, s.instanceOwner(), lease.FencingToken, s.leaseFor)
							if !ok {
								cancelTask()
								return
							}
						}
					}
				}) {
					cancelTask()
				}
			}()
		}
		var runErr error
		var panicVal any
		defer func() {
			if p := recover(); p != nil {
				panicVal = p
				safego.LogPanic("scheduled task execution", p)
				runErr = domain.ErrTaskPanic
			}
			elapsed := int(time.Since(start).Seconds())
			if elapsed < 0 {
				elapsed = 0
			}
			status := "success"
			msg := ""
			if runErr != nil {
				status = "failed"
				msg = runErr.Error()
			}
			if s.runs != nil {
				finish := time.Now()
				run.Status = status
				run.Message = msg
				run.FinishedAt = &finish
				run.DurationSec = elapsed
				// Fenced write: no-op when this worker lost the lease.
				_, _ = s.runs.CompleteTaskRun(ctx, *run, cfg.Key, s.instanceOwner(), lease.FencingToken)
				_ = s.runs.ReleaseTaskLease(ctx, cfg.Key, s.instanceOwner(), lease.FencingToken)
			}
			s.mu.Lock()
			rt.running = false
			rt.lastRun = time.Now()
			rt.lastElapsed = elapsed
			rt.lastStatus = status
			rt.lastError = msg
			s.mu.Unlock()
		}()
		_ = panicVal
		switch cfg.Key {
		case "vps_refresh":
			if s.vps != nil {
				_, runErr = s.vps.RefreshAll(taskCtx, 200)
				if runErr == nil && s.orders != nil {
					_, _ = s.orders.ReconcileProvisioningOrders(taskCtx, 50)
				}
			}
		case "order_provision_watchdog":
			if s.orders != nil {
				runErr = s.orders.ProcessProvisionJobs(taskCtx, 50)
			}
		case "resize_task_runner":
			if s.orders != nil {
				runErr = s.orders.ProcessResizeTasks(taskCtx, 50)
			}
		case "expire_reminder":
			if s.notify != nil {
				runErr = s.notify.SendExpireReminders(taskCtx)
			}
		case "vps_expire_cleanup":
			if s.vps != nil {
				runErr = s.vps.AutoDeleteExpired(taskCtx)
			}
		case "vps_expire_lock":
			if s.vps != nil {
				runErr = s.vps.AutoLockExpired(taskCtx)
			}
		case "plugin_schedule":
			if s.realname != nil {
				_, runErr = s.realname.PollPending(taskCtx, 200)
			}
		case "user_tier_expire_reconcile":
			if s.userTier != nil {
				_, runErr = s.userTier.ReconcileExpired(taskCtx, 500)
			}
		case "integration_inventory_sync":
			if s.integration != nil {
				_, runErr = s.integration.SyncAutomationInventoryForGoodsType(taskCtx, 0)
			}
		case "log_retention_cleanup":
			if s.logCleaner != nil {
				_, runErr = s.logCleaner.Cleanup(taskCtx)
			}
		}
	})
}

// instanceOwner identifies this server instance for lease ownership. It is
// stable per process, so restarts are distinguishable from concurrent replicas.
func (s *Service) instanceOwner() string {
	host, _ := os.Hostname()
	if strings.TrimSpace(host) == "" {
		host = "unknown"
	}
	return "scheduledtask:" + strings.TrimSpace(host) + ":" + strconv.Itoa(os.Getpid())
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

// maxIntervalSec is the largest IntervalSec value that fits in a time.Duration.
const maxIntervalSec = int64(^uint64(0)>>1) / int64(time.Second)

// intervalDuration converts an interval_sec setting into a duration, rejecting
// overflow and non-positive values by falling back to 60s.
func intervalDuration(cfg ScheduledTaskConfig) time.Duration {
	if cfg.IntervalSec <= 0 {
		return 60 * time.Second
	}
	if int64(cfg.IntervalSec) > maxIntervalSec {
		return 60 * time.Second
	}
	return time.Duration(cfg.IntervalSec) * time.Second
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
	interval := intervalDuration(cfg)
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
		if int64(cfg.IntervalSec) > maxIntervalSec {
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
