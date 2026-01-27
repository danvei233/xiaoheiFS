package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
)

type VPSService struct {
	vps        VPSRepository
	automation AutomationClient
	settings   SettingsRepository
}

func NewVPSService(vps VPSRepository, automation AutomationClient, settings SettingsRepository) *VPSService {
	return &VPSService{vps: vps, automation: automation, settings: settings}
}

func (s *VPSService) ListByUser(ctx context.Context, userID int64) ([]domain.VPSInstance, error) {
	return s.vps.ListInstancesByUser(ctx, userID)
}

func (s *VPSService) RefreshAll(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	offset := 0
	refreshed := 0
	for {
		items, total, err := s.vps.ListInstances(ctx, limit, offset)
		if err != nil {
			return refreshed, err
		}
		for _, inst := range items {
			if _, err := s.RefreshStatus(ctx, inst); err == nil {
				refreshed++
			}
		}
		offset += len(items)
		if offset >= total || len(items) == 0 {
			break
		}
	}
	return refreshed, nil
}

func (s *VPSService) Get(ctx context.Context, id int64, userID int64) (domain.VPSInstance, error) {
	inst, err := s.vps.GetInstance(ctx, id)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	if inst.UserID != userID {
		return domain.VPSInstance{}, ErrForbidden
	}
	return inst, nil
}

func (s *VPSService) RefreshStatus(ctx context.Context, inst domain.VPSInstance) (domain.VPSInstance, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return domain.VPSInstance{}, ErrInvalidInput
	}
	info, err := s.automation.GetHostInfo(ctx, hostID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	status := MapAutomationState(info.State)
	if err := s.vps.UpdateInstanceStatus(ctx, inst.ID, status, info.State); err != nil {
		return domain.VPSInstance{}, err
	}
	if info.ExpireAt != nil {
		_ = s.vps.UpdateInstanceExpireAt(ctx, inst.ID, *info.ExpireAt)
	}
	if info.RemoteIP != "" || info.PanelPassword != "" || info.VNCPassword != "" {
		_ = s.vps.UpdateInstanceAccessInfo(ctx, inst.ID, mergeAccessInfo(inst.AccessInfoJSON, info))
	}
	if info.CPU > 0 || info.MemoryGB > 0 || info.DiskGB > 0 || info.Bandwidth > 0 {
		merged := mergeSpecInfo(inst.SpecJSON, info)
		if merged != "" {
			_ = s.vps.UpdateInstanceSpec(ctx, inst.ID, merged)
		}
	}
	return s.vps.GetInstance(ctx, inst.ID)
}

func (s *VPSService) SetStatus(ctx context.Context, inst domain.VPSInstance, status domain.VPSStatus, automationState int) error {
	return s.vps.UpdateInstanceStatus(ctx, inst.ID, status, automationState)
}

func (s *VPSService) GetPanelURL(ctx context.Context, inst domain.VPSInstance) (string, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return "", ErrInvalidInput
	}
	info, err := s.automation.GetHostInfo(ctx, hostID)
	if err != nil {
		return "", err
	}
	url, err := s.automation.GetPanelURL(ctx, info.HostName, info.PanelPassword)
	if err != nil {
		return "", err
	}
	_ = s.vps.UpdateInstancePanelCache(ctx, inst.ID, url)
	return url, nil
}

func (s *VPSService) RenewNow(ctx context.Context, inst domain.VPSInstance, days int) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	if days <= 0 {
		days = 30
	}
	next := time.Now().Add(time.Duration(days) * 24 * time.Hour)
	if inst.ExpireAt != nil && inst.ExpireAt.After(time.Now()) {
		next = inst.ExpireAt.Add(time.Duration(days) * 24 * time.Hour)
	}
	if err := s.automation.RenewHost(ctx, hostID, next); err != nil {
		return err
	}
	if inst.AdminStatus != domain.VPSAdminStatusNormal || inst.Status == domain.VPSStatusExpiredLocked {
		_ = s.automation.UnlockHost(ctx, hostID)
	}
	return s.vps.UpdateInstanceExpireAt(ctx, inst.ID, next)
}

func (s *VPSService) Start(ctx context.Context, inst domain.VPSInstance) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	return s.automation.StartHost(ctx, hostID)
}

func (s *VPSService) Shutdown(ctx context.Context, inst domain.VPSInstance) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	return s.automation.ShutdownHost(ctx, hostID)
}

func (s *VPSService) Reboot(ctx context.Context, inst domain.VPSInstance) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	return s.automation.RebootHost(ctx, hostID)
}

func (s *VPSService) ResetOS(ctx context.Context, inst domain.VPSInstance, templateID int64, password string) error {
	if s.automation == nil {
		return ErrInvalidInput
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	if templateID <= 0 {
		return ErrInvalidInput
	}
	if password == "" && inst.AccessInfoJSON != "" {
		var existing map[string]any
		if err := json.Unmarshal([]byte(inst.AccessInfoJSON), &existing); err == nil {
			if v, ok := existing["os_password"]; ok {
				password = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
	}
	if password == "" {
		return ErrInvalidInput
	}
	if err := s.automation.ResetOS(ctx, hostID, templateID, password); err != nil {
		return err
	}
	_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, domain.VPSStatusReinstalling, 4)
	access := map[string]any{}
	if inst.AccessInfoJSON != "" {
		_ = json.Unmarshal([]byte(inst.AccessInfoJSON), &access)
	}
	access["os_password"] = password
	_ = s.vps.UpdateInstanceAccessInfo(ctx, inst.ID, mustJSON(access))
	return nil
}

func (s *VPSService) ResetOSPassword(ctx context.Context, inst domain.VPSInstance, password string) error {
	if s.automation == nil {
		return ErrInvalidInput
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return ErrInvalidInput
	}
	if err := s.automation.ResetOSPassword(ctx, hostID, password); err != nil {
		return err
	}
	access := map[string]any{}
	if inst.AccessInfoJSON != "" {
		_ = json.Unmarshal([]byte(inst.AccessInfoJSON), &access)
	}
	access["os_password"] = password
	_ = s.vps.UpdateInstanceAccessInfo(ctx, inst.ID, mustJSON(access))
	return nil
}

func (s *VPSService) ListSnapshots(ctx context.Context, inst domain.VPSInstance) ([]AutomationSnapshot, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return nil, ErrInvalidInput
	}
	return s.automation.ListSnapshots(ctx, hostID)
}

func (s *VPSService) CreateSnapshot(ctx context.Context, inst domain.VPSInstance) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	return s.automation.CreateSnapshot(ctx, hostID)
}

func (s *VPSService) DeleteSnapshot(ctx context.Context, inst domain.VPSInstance, snapshotID int64) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 || snapshotID <= 0 {
		return ErrInvalidInput
	}
	return s.automation.DeleteSnapshot(ctx, hostID, snapshotID)
}

func (s *VPSService) RestoreSnapshot(ctx context.Context, inst domain.VPSInstance, snapshotID int64) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 || snapshotID <= 0 {
		return ErrInvalidInput
	}
	return s.automation.RestoreSnapshot(ctx, hostID, snapshotID)
}

func (s *VPSService) ListBackups(ctx context.Context, inst domain.VPSInstance) ([]AutomationBackup, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return nil, ErrInvalidInput
	}
	return s.automation.ListBackups(ctx, hostID)
}

func (s *VPSService) CreateBackup(ctx context.Context, inst domain.VPSInstance) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	return s.automation.CreateBackup(ctx, hostID)
}

func (s *VPSService) DeleteBackup(ctx context.Context, inst domain.VPSInstance, backupID int64) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 || backupID <= 0 {
		return ErrInvalidInput
	}
	return s.automation.DeleteBackup(ctx, hostID, backupID)
}

func (s *VPSService) RestoreBackup(ctx context.Context, inst domain.VPSInstance, backupID int64) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 || backupID <= 0 {
		return ErrInvalidInput
	}
	return s.automation.RestoreBackup(ctx, hostID, backupID)
}

func (s *VPSService) ListFirewallRules(ctx context.Context, inst domain.VPSInstance) ([]AutomationFirewallRule, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return nil, ErrInvalidInput
	}
	return s.automation.ListFirewallRules(ctx, hostID)
}

func (s *VPSService) AddFirewallRule(ctx context.Context, inst domain.VPSInstance, req AutomationFirewallRuleCreate) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	req.HostID = hostID
	return s.automation.AddFirewallRule(ctx, req)
}

func (s *VPSService) DeleteFirewallRule(ctx context.Context, inst domain.VPSInstance, ruleID int64) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 || ruleID <= 0 {
		return ErrInvalidInput
	}
	return s.automation.DeleteFirewallRule(ctx, hostID, ruleID)
}

func (s *VPSService) ListPortMappings(ctx context.Context, inst domain.VPSInstance) ([]AutomationPortMapping, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return nil, ErrInvalidInput
	}
	return s.automation.ListPortMappings(ctx, hostID)
}

func (s *VPSService) AddPortMapping(ctx context.Context, inst domain.VPSInstance, req AutomationPortMappingCreate) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	req.HostID = hostID
	return s.automation.AddPortMapping(ctx, req)
}

func (s *VPSService) DeletePortMapping(ctx context.Context, inst domain.VPSInstance, mappingID int64) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 || mappingID <= 0 {
		return ErrInvalidInput
	}
	return s.automation.DeletePortMapping(ctx, hostID, mappingID)
}

func (s *VPSService) FindPortCandidates(ctx context.Context, inst domain.VPSInstance, keywords string) ([]int64, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return nil, ErrInvalidInput
	}
	return s.automation.FindPortCandidates(ctx, hostID, keywords)
}

func (s *VPSService) Monitor(ctx context.Context, inst domain.VPSInstance) (AutomationMonitor, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return AutomationMonitor{}, ErrInvalidInput
	}
	return s.automation.GetMonitor(ctx, hostID)
}

func mergeSpecInfo(existing string, info AutomationHostInfo) string {
	spec := map[string]any{}
	if existing != "" {
		if err := json.Unmarshal([]byte(existing), &spec); err != nil {
			spec = map[string]any{}
		}
	}
	if info.CPU > 0 {
		spec["cpu"] = info.CPU
	}
	if info.MemoryGB > 0 {
		spec["memory_gb"] = info.MemoryGB
	}
	if info.DiskGB > 0 {
		spec["disk_gb"] = info.DiskGB
	}
	if info.Bandwidth > 0 {
		spec["bandwidth_mbps"] = info.Bandwidth
	}
	if len(spec) == 0 {
		return ""
	}
	return mustJSON(spec)
}

func (s *VPSService) VNCURL(ctx context.Context, inst domain.VPSInstance) (string, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return "", ErrInvalidInput
	}
	return s.automation.GetVNCURL(ctx, hostID)
}

func (s *VPSService) EmergencyRenew(ctx context.Context, inst domain.VPSInstance) (domain.VPSInstance, error) {
	if s.settings == nil {
		return domain.VPSInstance{}, ErrInvalidInput
	}
	policy := loadEmergencyRenewPolicy(ctx, s.settings)
	if !policy.Enabled {
		return domain.VPSInstance{}, ErrForbidden
	}
	if !emergencyRenewInWindow(time.Now(), inst.ExpireAt, policy.WindowDays) {
		return domain.VPSInstance{}, ErrForbidden
	}
	if inst.LastEmergencyRenewAt != nil {
		if time.Since(*inst.LastEmergencyRenewAt) < time.Duration(policy.IntervalHours)*time.Hour {
			return domain.VPSInstance{}, ErrConflict
		}
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return domain.VPSInstance{}, ErrInvalidInput
	}
	next := time.Now().Add(time.Duration(policy.RenewDays) * 24 * time.Hour)
	if inst.ExpireAt != nil && inst.ExpireAt.After(time.Now()) {
		next = inst.ExpireAt.Add(time.Duration(policy.RenewDays) * 24 * time.Hour)
	}
	if err := s.automation.RenewHost(ctx, hostID, next); err != nil {
		return domain.VPSInstance{}, err
	}
	if inst.AdminStatus != domain.VPSAdminStatusNormal || inst.Status == domain.VPSStatusExpiredLocked {
		_ = s.automation.UnlockHost(ctx, hostID)
	}
	now := time.Now()
	_ = s.vps.UpdateInstanceExpireAt(ctx, inst.ID, next)
	_ = s.vps.UpdateInstanceEmergencyRenewAt(ctx, inst.ID, now)
	return s.vps.GetInstance(ctx, inst.ID)
}

func (s *VPSService) AutoDeleteExpired(ctx context.Context) error {
	if s.settings == nil || s.vps == nil || s.automation == nil {
		return nil
	}
	enabled, ok := getSettingBool(ctx, s.settings, "auto_delete_enabled")
	if !ok || !enabled {
		return nil
	}
	days := 0
	if v, ok := getSettingInt(ctx, s.settings, "auto_delete_days"); ok {
		days = v
	}
	if days < 0 {
		days = 0
	}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	items, err := s.vps.ListInstancesExpiring(ctx, cutoff)
	if err != nil {
		return err
	}
	for _, inst := range items {
		if inst.ExpireAt == nil || inst.ExpireAt.After(cutoff) {
			continue
		}
		hostID := parseHostID(inst.AutomationInstanceID)
		if hostID == 0 {
			continue
		}
		if err := s.automation.DeleteHost(ctx, hostID); err != nil {
			continue
		}
		_ = s.vps.DeleteInstance(ctx, inst.ID)
	}
	return nil
}
