package plugins

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fmt"
	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/pkg/pluginsign"
)

const pluginsBootstrappedSettingKey = "plugins_bootstrapped"

type DiscoverItem struct {
	Category        string                       `json:"category"`
	PluginID        string                       `json:"plugin_id"`
	Name            string                       `json:"name"`
	Version         string                       `json:"version"`
	SignatureStatus domain.PluginSignatureStatus `json:"signature_status"`
	Entry           EntryInfo                    `json:"entry"`
}

func (m *Manager) BootstrapFromDisk(ctx context.Context, settings appports.SettingsRepository) error {
	if m.repo == nil {
		return fmt.Errorf("plugin repo missing")
	}

	existing, err := m.repo.ListPluginInstallations(ctx)
	if err != nil {
		return err
	}
	existingMap := map[string]domain.PluginInstallation{}
	for _, inst := range existing {
		existingMap[inst.Category+":"+inst.PluginID] = inst
	}

	bootstrapped := false
	if settings != nil {
		if s, err := settings.GetSetting(ctx, pluginsBootstrappedSettingKey); err == nil {
			if v, ok := parseBoolSetting(s.ValueJSON); ok {
				bootstrapped = v
			}
		}
	}
	if len(existing) == 0 {
		bootstrapped = false
	}

	// Pre-digest installations (created before the content-digest anchor existed)
	// must never silently ratify the current disk content: if the plugin was
	// tampered before the upgrade, backfilling would enshrine the attack payload
	// as the trusted digest. Fail closed instead:
	//   - A plugin whose on-disk content verifies as OFFICIAL against the current
	//     trusted keys is anchored from those verified bytes (the signature is the
	//     authenticator, so the backfill is safe).
	//   - Any other pre-digest plugin is kept disabled and surfaced so an operator
	//     must explicitly re-import/authorize it (ImportFromDisk pins a fresh
	//     digest after the admin second factor for untrusted content).
	var errs []error
	for _, inst := range existingMap {
		if inst.Category == "" || inst.PluginID == "" {
			continue
		}
		pluginDir := filepath.Join(m.baseDir, inst.Category, inst.PluginID)
		if inst.ContentDigest == "" {
			// Pre-digest installation: only content authenticated by a current
			// trusted official key may be anchored from the verified bytes; any
			// other pre-digest plugin stays disabled (see comment above).
			if m.officialKeys != nil {
				if status, verr := VerifySignature(pluginDir, m.officialKeys); verr == nil && status == domain.PluginSignatureOfficial {
					digest, derr := pluginsign.ContentDigest(pluginDir)
					if derr != nil {
						errs = append(errs, fmt.Errorf("bootstrap digest %s/%s: %w", inst.Category, inst.PluginID, derr))
						continue
					}
					inst.ContentDigest = digest
					if uerr := m.repo.UpsertPluginInstallation(ctx, &inst); uerr != nil {
						errs = append(errs, fmt.Errorf("bootstrap digest persist %s/%s: %w", inst.Category, inst.PluginID, uerr))
					}
					continue
				}
			}
			// No trusted anchor. Keep the installation disabled so it can never
			// auto-start, and do not write the current disk bytes as the trusted
			// digest. Surface the condition so the operator re-imports explicitly.
			if inst.Enabled {
				inst.Enabled = false
				if uerr := m.repo.UpsertPluginInstallation(ctx, &inst); uerr != nil {
					errs = append(errs, fmt.Errorf("bootstrap disable %s/%s: %w", inst.Category, inst.PluginID, uerr))
					continue
				}
			}
			log.Printf("plugin %s/%s has no recorded content digest and is not authenticated by a trusted key; kept disabled — re-import it explicitly to authorize its current content",
				inst.Category, inst.PluginID)
			continue
		}

		// Digest-anchored installation: a legitimate upgrade (image rebuild,
		// operator refresh) changes the on-disk bytes, which would otherwise be
		// indistinguishable from tampering at start time. Only content that a
		// current trusted official key authenticates may be re-anchored; anything
		// else fails closed — the recorded digest stays pinned, the plugin is
		// disabled so it can never auto-start with unverified bytes, and the
		// condition is surfaced for an explicit operator re-import.
		digest, derr := pluginsign.ContentDigest(pluginDir)
		if derr != nil {
			errs = append(errs, fmt.Errorf("bootstrap digest %s/%s: %w", inst.Category, inst.PluginID, derr))
			continue
		}
		if digest == inst.ContentDigest {
			continue
		}
		if m.officialKeys != nil {
			if status, verr := VerifySignature(pluginDir, m.officialKeys); verr == nil && status == domain.PluginSignatureOfficial {
				inst.ContentDigest = digest
				inst.SignatureStatus = status
				if uerr := m.repo.UpsertPluginInstallation(ctx, &inst); uerr != nil {
					errs = append(errs, fmt.Errorf("bootstrap re-anchor %s/%s: %w", inst.Category, inst.PluginID, uerr))
				}
				continue
			}
		}
		if inst.Enabled {
			inst.Enabled = false
			if uerr := m.repo.UpsertPluginInstallation(ctx, &inst); uerr != nil {
				errs = append(errs, fmt.Errorf("bootstrap disable %s/%s: %w", inst.Category, inst.PluginID, uerr))
				continue
			}
		}
		log.Printf("plugin %s/%s content changed on disk and is not authenticated by a trusted key; kept disabled and digest pinned — re-import it explicitly to authorize the new content",
			inst.Category, inst.PluginID)
	}

	found, err := scanDiskPlugins(m.baseDir)
	if err != nil {
		return err
	}

	// First bootstrap: import everything (enabled=false, config empty).
	if !bootstrapped {
		for _, p := range found {
			key := p.Category + ":" + p.PluginID
			if _, ok := existingMap[key]; ok {
				continue
			}
			sigStatus, err := VerifySignature(p.Dir, m.officialKeys)
			if err != nil {
				errs = append(errs, fmt.Errorf("bootstrap verify %s/%s: %w", p.Category, p.PluginID, err))
				continue
			}
			// With a trusted official key configured, only official plugins are
			// auto-imported by the first bootstrap. Untrusted on-disk content is
			// surfaced via DiscoverOnDisk and requires an explicit operator import
			// (which enforces the admin second factor), so pre-placing a malicious
			// directory cannot get it silently into the installation list.
			if m.officialKeys != nil && sigStatus != domain.PluginSignatureOfficial {
				continue
			}
			_, _ = ResolveEntry(p.Dir, p.Manifest) // parse existence (for operator visibility via list/discover)
			digest, derr := pluginsign.ContentDigest(p.Dir)
			if derr != nil {
				errs = append(errs, fmt.Errorf("bootstrap digest %s/%s: %w", p.Category, p.PluginID, derr))
				continue
			}
			inst := domain.PluginInstallation{
				Category:        p.Category,
				PluginID:        p.PluginID,
				InstanceID:      DefaultInstanceID,
				Enabled:         false,
				SignatureStatus: sigStatus,
				ContentDigest:   digest,
				ConfigCipher:    "",
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			if err := m.repo.UpsertPluginInstallation(ctx, &inst); err != nil {
				errs = append(errs, fmt.Errorf("bootstrap import %s/%s: %w", p.Category, p.PluginID, err))
				continue
			}
		}
		// Mark the scan complete even when some imports failed. A persistently
		// broken plugin on disk must not keep the system in first-bootstrap mode
		// forever: while the marker stays unset, every restart re-imports ALL
		// untrusted plugins found on disk, so the "subsequent startup only
		// auto-imports official plugins" filter never engages. Failed plugins
		// remain un-imported and are surfaced for explicit operator action via
		// DiscoverOnDisk/ImportFromDisk. The errors are still returned so the
		// caller can log/alert on them.
		if settings != nil {
			_ = settings.UpsertSetting(ctx, domain.Setting{Key: pluginsBootstrappedSettingKey, ValueJSON: "true", UpdatedAt: time.Now()})
		}
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
		return nil
	}

	// Subsequent startup: auto-import ONLY new official plugins found on disk.
	for _, p := range found {
		key := p.Category + ":" + p.PluginID
		if _, ok := existingMap[key]; ok {
			continue
		}
		sigStatus, err := VerifySignature(p.Dir, m.officialKeys)
		if err != nil {
			errs = append(errs, fmt.Errorf("bootstrap verify %s/%s: %w", p.Category, p.PluginID, err))
			continue
		}
		if sigStatus != domain.PluginSignatureOfficial {
			// P2-65: never skip a new image-upgrade plugin silently — an operator
			// must be able to see that an unsigned bundle was not auto-imported
			// and take action (sign it or import manually).
			log.Printf("plugins/bootstrap: skipping new plugin %s on disk: signature status %q (not official)", key, sigStatus)
			continue
		}
		digest, derr := pluginsign.ContentDigest(p.Dir)
		if derr != nil {
			errs = append(errs, fmt.Errorf("bootstrap digest %s/%s: %w", p.Category, p.PluginID, derr))
			continue
		}
		inst := domain.PluginInstallation{
			Category:        p.Category,
			PluginID:        p.PluginID,
			InstanceID:      DefaultInstanceID,
			Enabled:         false,
			SignatureStatus: sigStatus,
			ContentDigest:   digest,
			ConfigCipher:    "",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := m.repo.UpsertPluginInstallation(ctx, &inst); err != nil {
			errs = append(errs, fmt.Errorf("bootstrap import %s/%s: %w", p.Category, p.PluginID, err))
			continue
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (m *Manager) DiscoverOnDisk(ctx context.Context) ([]DiscoverItem, error) {
	if m.repo == nil {
		return nil, fmt.Errorf("plugin repo missing")
	}
	existing, err := m.repo.ListPluginInstallations(ctx)
	if err != nil {
		return nil, err
	}
	existingMap := map[string]struct{}{}
	for _, inst := range existing {
		existingMap[inst.Category+":"+inst.PluginID] = struct{}{}
	}

	found, err := scanDiskPlugins(m.baseDir)
	if err != nil {
		return nil, err
	}

	out := make([]DiscoverItem, 0, len(found))
	for _, p := range found {
		if _, ok := existingMap[p.Category+":"+p.PluginID]; ok {
			continue
		}
		sigStatus, sigErr := VerifySignature(p.Dir, m.officialKeys)
		if sigErr != nil {
			// A plugin that cannot be verified is surfaced as untrusted so the
			// operator can see it, rather than disappearing from the listing.
			sigStatus = domain.PluginSignatureUntrusted
		}
		entry, err := ResolveEntry(p.Dir, p.Manifest)
		if err != nil {
			// keep entry info (supported platforms) for UI
		}
		out = append(out, DiscoverItem{
			Category:        p.Category,
			PluginID:        p.PluginID,
			Name:            p.Manifest.Name,
			Version:         p.Manifest.Version,
			SignatureStatus: sigStatus,
			Entry:           entry,
		})
	}
	return out, nil
}

// ImportPreview describes an on-disk plugin inspected for import: the signature
// status (used by the handler to require a second factor for untrusted content)
// and the content digest that authorization is bound to.
type ImportPreview struct {
	Category        string
	PluginID        string
	SignatureStatus domain.PluginSignatureStatus
	ContentDigest   string
}

// PreviewImportFromDisk inspects an on-disk plugin and returns its signature
// status and content digest WITHOUT committing anything. The handler uses this
// to decide whether a second factor is required; the final ImportFromDisk
// re-inspects the directory and refuses if the digest changed, so the trust
// decision is bound to the exact directory that gets imported.
func (m *Manager) PreviewImportFromDisk(ctx context.Context, category, pluginID string) (ImportPreview, error) {
	if m.repo == nil {
		return ImportPreview{}, fmt.Errorf("plugin repo missing")
	}
	preview, _, err := m.inspectImportTarget(ctx, category, pluginID)
	return preview, err
}

// inspectImportTarget reads the on-disk plugin exactly once and returns both the
// preview (for the trust gate) and the parsed manifest. ImportFromDisk re-runs
// it and requires the digest to be unchanged, which binds the admin's second
// factor to the specific directory being committed.
func (m *Manager) inspectImportTarget(ctx context.Context, category, pluginID string) (ImportPreview, Manifest, error) {
	category = strings.TrimSpace(category)
	pluginID = strings.TrimSpace(pluginID)
	if category == "" || pluginID == "" {
		return ImportPreview{}, Manifest{}, fmt.Errorf("invalid plugin")
	}
	pluginDir := filepath.Join(m.baseDir, category, pluginID)
	if _, err := os.Stat(pluginDir); err != nil {
		return ImportPreview{}, Manifest{}, fmt.Errorf("plugin dir not found")
	}

	manifest, err := ReadManifest(pluginDir)
	if err != nil {
		return ImportPreview{}, Manifest{}, err
	}
	if manifest.PluginID != pluginID {
		return ImportPreview{}, Manifest{}, fmt.Errorf("manifest plugin_id mismatch")
	}

	entry, err := ResolveEntry(pluginDir, manifest)
	if err != nil {
		if len(entry.SupportedPlatforms) > 0 {
			return ImportPreview{}, Manifest{}, fmt.Errorf("%s", "unsupported platform "+entry.Platform+", supported: "+strings.Join(entry.SupportedPlatforms, ", "))
		}
		return ImportPreview{}, Manifest{}, err
	}
	_ = entry // resolved above to validate availability

	sigStatus, err := VerifySignature(pluginDir, m.officialKeys)
	if err != nil {
		return ImportPreview{}, Manifest{}, err
	}
	digest, err := pluginsign.ContentDigest(pluginDir)
	if err != nil {
		return ImportPreview{}, Manifest{}, err
	}
	return ImportPreview{
		Category:        category,
		PluginID:        pluginID,
		SignatureStatus: sigStatus,
		ContentDigest:   digest,
	}, manifest, nil
}

// ImportFromDisk commits an on-disk plugin. expectedDigest, when non-empty, is
// the digest from the PreviewImportFromDisk that an admin's trust decision was
// bound to; the directory is re-inspected and, if its content digest changed
// since the preview, the import is refused as a conflict so a directory swap
// cannot turn an authorized import into a different plugin.
func (m *Manager) ImportFromDisk(ctx context.Context, category, pluginID, expectedDigest string) (domain.PluginInstallation, error) {
	if m.repo == nil {
		return domain.PluginInstallation{}, fmt.Errorf("plugin repo missing")
	}
	category = strings.TrimSpace(category)
	pluginID = strings.TrimSpace(pluginID)
	unlock := m.lockOp(category, pluginID, DefaultInstanceID)
	defer unlock()
	preview, _, err := m.inspectImportTarget(ctx, category, pluginID)
	if err != nil {
		return domain.PluginInstallation{}, err
	}
	if strings.TrimSpace(expectedDigest) != "" && preview.ContentDigest != strings.TrimSpace(expectedDigest) {
		return domain.PluginInstallation{}, fmt.Errorf("plugin directory changed during import; retry")
	}
	category, pluginID = preview.Category, preview.PluginID
	sigStatus, digest := preview.SignatureStatus, preview.ContentDigest

	// Upsert without touching enabled/config (new import defaults to disabled/empty).
	const instanceID = "default"
	if inst, err := m.repo.GetPluginInstallation(ctx, category, pluginID, instanceID); err == nil {
		inst.SignatureStatus = sigStatus
		inst.ContentDigest = digest
		if err := m.repo.UpsertPluginInstallation(ctx, &inst); err != nil {
			return domain.PluginInstallation{}, err
		}
		return m.repo.GetPluginInstallation(ctx, category, pluginID, instanceID)
	}

	inst := domain.PluginInstallation{
		Category:        category,
		PluginID:        pluginID,
		InstanceID:      instanceID,
		Enabled:         false,
		SignatureStatus: sigStatus,
		ContentDigest:   digest,
		ConfigCipher:    "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := m.repo.UpsertPluginInstallation(ctx, &inst); err != nil {
		return domain.PluginInstallation{}, err
	}
	return m.repo.GetPluginInstallation(ctx, category, pluginID, instanceID)
}

type diskPlugin struct {
	Category string
	PluginID string
	Dir      string
	Manifest Manifest
}

func scanDiskPlugins(baseDir string) ([]diskPlugin, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return nil, fmt.Errorf("missing base dir")
	}
	st, err := os.Stat(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("plugins base dir is not a directory")
	}

	var out []diskPlugin
	cats, _ := os.ReadDir(baseDir)
	for _, c := range cats {
		if !c.IsDir() {
			continue
		}
		category := strings.TrimSpace(c.Name())
		if category == "" || strings.HasPrefix(category, ".") {
			continue
		}
		catDir := filepath.Join(baseDir, category)
		plugins, _ := os.ReadDir(catDir)
		for _, p := range plugins {
			if !p.IsDir() {
				continue
			}
			pluginID := strings.TrimSpace(p.Name())
			if pluginID == "" || strings.HasPrefix(pluginID, ".") {
				continue
			}
			dir := filepath.Join(catDir, pluginID)
			if _, err := os.Stat(filepath.Join(dir, "manifest.json")); err != nil {
				continue
			}
			m, err := ReadManifest(dir)
			if err != nil {
				continue
			}
			if m.PluginID != pluginID {
				continue
			}
			out = append(out, diskPlugin{Category: category, PluginID: pluginID, Dir: dir, Manifest: m})
		}
	}
	return out, nil
}

func parseBoolSetting(v string) (bool, bool) {
	s := strings.TrimSpace(v)
	s = strings.Trim(s, "\"")
	switch strings.ToLower(s) {
	case "true", "1", "yes", "y":
		return true, true
	case "false", "0", "no", "n":
		return false, true
	default:
		return false, false
	}
}
