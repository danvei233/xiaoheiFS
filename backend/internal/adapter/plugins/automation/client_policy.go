package automation

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// P2-43: the automation RPC client must not use one fixed 12s timeout with no
// retry policy. Actions are categorized so:
//   - idempotent reads get a bounded number of automatic retries on transient
//     errors;
//   - long-running operations (create/provision, backup/snapshot create/restore,
//     resize, rebuild) get a generous timeout;
//   - destructive RPCs (destroy, delete, reset, reboot, lock...) are NEVER
//     retried automatically — a blind retry could double-execute a side effect.
//
// The per-plugin instance config may override the timeout via `timeout_sec`
// (the same field the plugin config schemas expose) and the read-retry count via
// `retry` / `retry_count`.

type actionCategory int

const (
	catRead actionCategory = iota
	catLongOp
	catDestructive
	catDefault
)

type callPolicy struct {
	category   actionCategory
	timeout    time.Duration
	maxRetries int // automatic retries for idempotent reads only
}

const (
	defaultRPCTimeout = 30 * time.Second
	longOpRPCTimeout  = 120 * time.Second
	maxReadRetries    = 2
)

// idempotent read actions may be safely retried: repeating them cannot create a
// side effect.
var readActions = map[string]bool{
	"automation.GetInstance":         true,
	"automation.ListInstancesSimple": true,
	"automation.ListSnapshots":       true,
	"automation.ListBackups":         true,
	"automation.ListFirewallRules":   true,
	"automation.ListPortMappings":    true,
	"automation.FindPortCandidates":  true,
	"automation.ListAreas":           true,
	"automation.ListImages":          true,
	"automation.ListLines":           true,
	"automation.ListPackages":        true,
	"automation.GetMonitor":          true,
	"automation.GetPanelURL":         true,
	"automation.GetVNCURL":           true,
}

// long operations may take far longer than a single RPC round-trip.
var longOpActions = map[string]bool{
	"automation.CreateInstance":  true,
	"automation.ElasticUpdate":   true,
	"automation.CreateSnapshot":  true,
	"automation.CreateBackup":    true,
	"automation.RestoreSnapshot": true,
	"automation.RestoreBackup":   true,
	"automation.Rebuild":         true,
}

// destructive RPCs must never be auto-retried.
var destructiveActions = map[string]bool{
	"automation.Destroy":            true,
	"automation.DeleteSnapshot":     true,
	"automation.DeleteBackup":       true,
	"automation.DeleteFirewallRule": true,
	"automation.DeletePortMapping":  true,
	"automation.ResetPassword":      true,
	"automation.Reboot":             true,
	"automation.Shutdown":           true,
	"automation.Start":              true,
	"automation.Lock":               true,
	"automation.Unlock":             true,
	"automation.Renew":              true,
}

func basePolicyFor(action string) callPolicy {
	action = strings.TrimSpace(action)
	switch {
	case readActions[action]:
		return callPolicy{category: catRead, timeout: defaultRPCTimeout, maxRetries: maxReadRetries}
	case longOpActions[action]:
		return callPolicy{category: catLongOp, timeout: longOpRPCTimeout, maxRetries: 0}
	case destructiveActions[action]:
		return callPolicy{category: catDestructive, timeout: defaultRPCTimeout, maxRetries: 0}
	default:
		return callPolicy{category: catDefault, timeout: defaultRPCTimeout, maxRetries: 0}
	}
}

// instanceConfigOverride reads optional timeout_sec / retry from the plugin
// instance config (the same fields exposed by the automation config schemas).
func (c *PluginInstanceClient) instanceConfigOverride(ctx context.Context) (timeout time.Duration, retries int) {
	if c.mgr == nil {
		return 0, 0
	}
	cfgJSON, err := c.mgr.GetConfigInstance(ctx, "automation", c.pluginID, c.instanceID)
	if err != nil || strings.TrimSpace(cfgJSON) == "" {
		return 0, 0
	}
	var cfg struct {
		TimeoutSec int `json:"timeout_sec"`
		Retry      int `json:"retry"`
		RetryCount int `json:"retry_count"`
	}
	if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
		return 0, 0
	}
	if cfg.TimeoutSec > 0 {
		timeout = time.Duration(cfg.TimeoutSec) * time.Second
	}
	retries = cfg.Retry
	if retries == 0 {
		retries = cfg.RetryCount
	}
	return timeout, retries
}

// policyFor resolves the effective policy for an action, applying the per-plugin
// instance config overrides.
func (c *PluginInstanceClient) policyFor(ctx context.Context, action string) callPolicy {
	p := basePolicyFor(action)
	timeoutOverride, retryOverride := c.instanceConfigOverride(ctx)
	if timeoutOverride > 0 {
		p.timeout = timeoutOverride
	}
	if retryOverride > 0 && p.category == catRead {
		p.maxRetries = retryOverride
	}
	return p
}

// isTransientAutomationError reports whether an RPC error is worth a bounded
// retry for an idempotent read (transport/busy errors), never for a
// business/certified failure.
func isTransientAutomationError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "unavailable") ||
		strings.Contains(msg, "context canceled")
}
