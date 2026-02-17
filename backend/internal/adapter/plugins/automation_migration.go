package plugins

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
)

func MigrateLegacyAutomationToPlugins(ctx context.Context, settings appports.SettingsRepository, goodsTypes appports.GoodsTypeRepository, mgr *Manager) error {
	if settings == nil || mgr == nil {
		return nil
	}
	if v := getSettingValue(ctx, settings, "automation_migrated_to_plugins"); strings.ToLower(v) == "true" {
		return nil
	}
	baseURL := strings.TrimSpace(getSettingValue(ctx, settings, "automation_base_url"))
	apiKey := strings.TrimSpace(getSettingValue(ctx, settings, "automation_api_key"))
	if baseURL == "" || apiKey == "" {
		return nil
	}
	enabled := strings.ToLower(getSettingValue(ctx, settings, "automation_enabled")) == "true"
	timeoutSec := strings.TrimSpace(getSettingValue(ctx, settings, "automation_timeout_sec"))
	retry := strings.TrimSpace(getSettingValue(ctx, settings, "automation_retry"))
	dryRun := strings.ToLower(getSettingValue(ctx, settings, "automation_dry_run")) == "true"

	// Ensure plugin files exist for the current platform; otherwise keep legacy behavior.
	pluginDir := filepath.Join(mgr.baseDir, "automation", "lightboat")
	manifest, err := ReadManifest(pluginDir)
	if err != nil {
		return nil
	}
	if _, err := ResolveEntry(pluginDir, manifest); err != nil {
		return nil
	}

	// Ensure default goods type exists and bound (best-effort).
	if goodsTypes != nil {
		if items, err := goodsTypes.ListGoodsTypes(ctx); err == nil && len(items) > 0 {
			// no-op
		} else {
			gt := &domain.GoodsType{
				Code:                 "lightboat_vps",
				Name:                 "轻舟VPS",
				Active:               true,
				SortOrder:            0,
				AutomationCategory:   "automation",
				AutomationPluginID:   "lightboat",
				AutomationInstanceID: DefaultInstanceID,
			}
			_ = goodsTypes.CreateGoodsType(ctx, gt)
		}
	}

	// Ensure plugin instance exists.
	if _, err := mgr.repo.GetPluginInstallation(ctx, "automation", "lightboat", DefaultInstanceID); err != nil {
		if _, err := mgr.CreateInstance(ctx, "automation", "lightboat", DefaultInstanceID); err != nil {
			return nil
		}
	}

	cfg := map[string]any{
		"base_url":    baseURL,
		"api_key":     apiKey,
		"timeout_sec": parseIntDefault(timeoutSec, 12),
		"retry":       parseIntDefault(retry, 1),
		"dry_run":     dryRun,
	}
	b, _ := json.Marshal(cfg)
	if err := mgr.UpdateConfigInstance(ctx, "automation", "lightboat", DefaultInstanceID, string(b)); err != nil {
		// If the plugin binary isn't built yet, do not block startup.
		return nil
	}
	if enabled {
		_ = mgr.EnableInstance(ctx, "automation", "lightboat", DefaultInstanceID)
	}
	_ = settings.UpsertSetting(ctx, domain.Setting{Key: "automation_migrated_to_plugins", ValueJSON: "true"})
	return nil
}

func getSettingValue(ctx context.Context, settings appports.SettingsRepository, key string) string {
	if settings == nil {
		return ""
	}
	s, err := settings.GetSetting(ctx, key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(s.ValueJSON)
}

func parseIntDefault(v string, def int) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	n := 0
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	if n <= 0 {
		return def
	}
	return n
}
