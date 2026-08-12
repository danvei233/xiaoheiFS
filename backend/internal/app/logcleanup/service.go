package logcleanup

import (
	"context"
	"strconv"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
)

type auditLogPurger interface {
	PurgeAuditLogs(ctx context.Context, before time.Time) error
}

type automationLogPurger interface {
	PurgeAutomationLogs(ctx context.Context, before time.Time) error
}

type integrationLogPurger interface {
	PurgeSyncLogs(ctx context.Context, before time.Time) error
}

type taskRunPurger interface {
	PurgeTaskRuns(ctx context.Context, before time.Time) error
}

type probeStatusEventPurger interface {
	DeleteProbeStatusEventsBefore(ctx context.Context, before time.Time) error
}

type probeLogSessionPurger interface {
	PurgeProbeLogSessions(ctx context.Context, before time.Time) error
}

type rateLimitPurger interface {
	CleanupRateLimits(ctx context.Context, before time.Time) (int64, error)
}

type Service struct {
	settings      appports.SettingsRepository
	audit         auditLogPurger
	automation    automationLogPurger
	integration   integrationLogPurger
	taskRuns      taskRunPurger
	probeEvents   probeStatusEventPurger
	probeSessions probeLogSessionPurger
	rateLimits    rateLimitPurger
}

func NewService(
	settings appports.SettingsRepository,
	audit auditLogPurger,
	automation automationLogPurger,
	integration integrationLogPurger,
	taskRuns taskRunPurger,
	probeEvents probeStatusEventPurger,
	probeSessions probeLogSessionPurger,
	rateLimits rateLimitPurger,
) *Service {
	return &Service{
		settings:      settings,
		audit:         audit,
		automation:    automation,
		integration:   integration,
		taskRuns:      taskRuns,
		probeEvents:   probeEvents,
		probeSessions: probeSessions,
		rateLimits:    rateLimits,
	}
}

func (s *Service) Cleanup(ctx context.Context) (string, error) {
	now := time.Now()
	parts := make([]string, 0, 6)

	run := func(settingKey string, fallbackDays int, label string, fn func(before time.Time) error) error {
		if fn == nil {
			return nil
		}
		days := s.settingDays(ctx, settingKey, fallbackDays)
		if days <= 0 {
			return nil
		}
		before := now.AddDate(0, 0, -days)
		if err := fn(before); err != nil {
			return err
		}
		parts = append(parts, label+":"+strconv.Itoa(days)+"d")
		return nil
	}

	var automationFn func(before time.Time) error
	if s.automation != nil {
		automationFn = func(before time.Time) error { return s.automation.PurgeAutomationLogs(ctx, before) }
	}
	if err := run("automation_log_retention_days", 30, "automation", automationFn); err != nil {
		return strings.Join(parts, ","), err
	}
	var auditFn func(before time.Time) error
	if s.audit != nil {
		auditFn = func(before time.Time) error { return s.audit.PurgeAuditLogs(ctx, before) }
	}
	if err := run("audit_log_retention_days", 90, "audit", auditFn); err != nil {
		return strings.Join(parts, ","), err
	}
	var syncFn func(before time.Time) error
	if s.integration != nil {
		syncFn = func(before time.Time) error { return s.integration.PurgeSyncLogs(ctx, before) }
	}
	if err := run("integration_sync_log_retention_days", 30, "sync", syncFn); err != nil {
		return strings.Join(parts, ","), err
	}
	var taskRunFn func(before time.Time) error
	if s.taskRuns != nil {
		taskRunFn = func(before time.Time) error { return s.taskRuns.PurgeTaskRuns(ctx, before) }
	}
	if err := run("scheduled_task_run_retention_days", 14, "task_run", taskRunFn); err != nil {
		return strings.Join(parts, ","), err
	}
	var probeEventFn func(before time.Time) error
	if s.probeEvents != nil {
		probeEventFn = func(before time.Time) error { return s.probeEvents.DeleteProbeStatusEventsBefore(ctx, before) }
	}
	// Default retention covers the full SLA query window (365d) plus margin.
	// scanOffline no longer deletes status events, so this is the only purge path.
	if err := run("probe_status_event_retention_days", 366, "probe_event", probeEventFn); err != nil {
		return strings.Join(parts, ","), err
	}
	var probeSessionFn func(before time.Time) error
	if s.probeSessions != nil {
		probeSessionFn = func(before time.Time) error { return s.probeSessions.PurgeProbeLogSessions(ctx, before) }
	}
	if err := run("probe_log_session_retention_days", 7, "probe_session", probeSessionFn); err != nil {
		return strings.Join(parts, ","), err
	}
	var rateLimitFn func(before time.Time) error
	if s.rateLimits != nil {
		rateLimitFn = func(before time.Time) error {
			_, err := s.rateLimits.CleanupRateLimits(ctx, before)
			return err
		}
	}
	if err := run("rate_limit_retention_days", 1, "rate_limit", rateLimitFn); err != nil {
		return strings.Join(parts, ","), err
	}

	return strings.Join(parts, ","), nil
}

func (s *Service) settingDays(ctx context.Context, key string, fallback int) int {
	if s == nil || s.settings == nil || strings.TrimSpace(key) == "" {
		return fallback
	}
	item, err := s.settings.GetSetting(ctx, key)
	if err != nil {
		return fallback
	}
	v, err := strconv.Atoi(strings.TrimSpace(item.ValueJSON))
	if err != nil || v <= 0 {
		return fallback
	}
	if v > 3650 {
		return 3650
	}
	return v
}
