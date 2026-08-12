package plugins

import (
	"context"
	"encoding/json"
	"fmt"
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

	// Ensure default goods type exists and bound (best-effort). A read error must
	// not be turned into a write: only create the default when the list succeeds
	// and is empty.
	if goodsTypes != nil {
		if items, err := goodsTypes.ListGoodsTypes(ctx); err == nil && len(items) == 0 {
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

	timeout := parseIntDefault(timeoutSec, 12)
	if timeout <= 0 {
		// timeout_sec schema minimum is 1; 0/negative legacy values mean "use default".
		timeout = 12
	}
	cfg := map[string]any{
		"base_url":    baseURL,
		"api_key":     apiKey,
		"timeout_sec": timeout,
		"retry":       parseIntDefault(retry, 1),
		"dry_run":     dryRun,
	}
	b, _ := json.Marshal(cfg)
	if err := mgr.UpdateConfigInstance(ctx, "automation", "lightboat", DefaultInstanceID, string(b)); err != nil {
		// If the plugin binary isn't built yet, do not block startup.
		return nil
	}
	if enabled {
		// When a trusted official key is configured, never AUTO-ENABLE a plugin
		// whose on-disk content is not authenticated by it. CreateInstance above
		// self-certifies whatever is on disk, so a malicious pre-placed
		// automation/lightboat directory would otherwise be verified-by-itself and
		// started here with no admin interaction. Keep it disabled and surface the
		// condition so the operator explicitly re-imports/authorizes it.
		if mgr.officialKeys != nil {
			status, verr := VerifySignature(pluginDir, mgr.officialKeys)
			if verr != nil || status != domain.PluginSignatureOfficial {
				return fmt.Errorf("refusing to auto-enable unverified automation plugin (signature status=%q): re-import/authorize it explicitly", status)
			}
		}
		if err := mgr.EnableInstance(ctx, "automation", "lightboat", DefaultInstanceID); err != nil {
			// Do not mark migration complete when the plugin cannot be enabled;
			// otherwise automation stays silently disabled and never retries.
			return err
		}
	}
	if err := settings.UpsertSetting(ctx, domain.Setting{Key: "automation_migrated_to_plugins", ValueJSON: "true"}); err != nil {
		return err
	}
	// The api_key now lives encrypted in the plugin instance config; drop the
	// legacy plaintext copy so the secret does not linger in settings.
	_ = settings.UpsertSetting(ctx, domain.Setting{Key: "automation_api_key", ValueJSON: ""})
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
	return n
}
