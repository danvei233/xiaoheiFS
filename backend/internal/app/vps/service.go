package vps

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type (
	AutomationClient             = appshared.AutomationClient
	AutomationHostInfo           = appshared.AutomationHostInfo
	AutomationSnapshot           = appshared.AutomationSnapshot
	AutomationBackup             = appshared.AutomationBackup
	AutomationFirewallRule       = appshared.AutomationFirewallRule
	AutomationFirewallRuleCreate = appshared.AutomationFirewallRuleCreate
	AutomationPortMapping        = appshared.AutomationPortMapping
	AutomationPortMappingCreate  = appshared.AutomationPortMappingCreate
	AutomationMonitor            = appshared.AutomationMonitor
)

type Service struct {
	vps        appports.VPSRepository
	automation appports.AutomationClientResolver
	settings   appports.SettingsRepository
}

func NewService(vps appports.VPSRepository, automation appports.AutomationClientResolver, settings appports.SettingsRepository) *Service {
	return &Service{vps: vps, automation: automation, settings: settings}
}

func (s *Service) client(ctx context.Context, goodsTypeID int64) (AutomationClient, error) {
	if s.automation == nil {
		return nil, appshared.ErrInvalidInput
	}
	return s.automation.ClientForGoodsType(ctx, goodsTypeID)
}

func (s *Service) ListByUser(ctx context.Context, userID int64) ([]domain.VPSInstance, error) {
	return s.vps.ListInstancesByUser(ctx, userID)
}

func (s *Service) RefreshAll(ctx context.Context, limit int) (int, error) {
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

func (s *Service) Get(ctx context.Context, id int64, userID int64) (domain.VPSInstance, error) {
	inst, err := s.vps.GetInstance(ctx, id)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	if inst.UserID != userID {
		return domain.VPSInstance{}, appshared.ErrForbidden
	}
	return inst, nil
}

func (s *Service) UpdateLocalSystemID(ctx context.Context, inst domain.VPSInstance, systemID int64) error {
	if inst.ID <= 0 || systemID <= 0 {
		return appshared.ErrInvalidInput
	}
	latest, err := s.vps.GetInstance(ctx, inst.ID)
	if err != nil {
		return err
	}
	latest.SystemID = systemID
	return s.vps.UpdateInstanceLocal(ctx, latest)
}

func (s *Service) RefreshStatus(ctx context.Context, inst domain.VPSInstance) (domain.VPSInstance, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return domain.VPSInstance{}, appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	info, err := cli.GetHostInfo(ctx, hostID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	status := MapAutomationState(info.State)
	if err := s.vps.UpdateInstanceStatus(ctx, inst.ID, status, info.State); err != nil {
		return domain.VPSInstance{}, err
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

func (s *Service) SetStatus(ctx context.Context, inst domain.VPSInstance, status domain.VPSStatus, automationState int) error {
	return s.vps.UpdateInstanceStatus(ctx, inst.ID, status, automationState)
}

func (s *Service) GetPanelURL(ctx context.Context, inst domain.VPSInstance) (string, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return "", appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return "", err
	}
	info, err := cli.GetHostInfo(ctx, hostID)
	if err != nil {
		return "", err
	}
	url, err := cli.GetPanelURL(ctx, info.HostName, info.PanelPassword)
	if err != nil {
		return "", err
	}
	_ = s.vps.UpdateInstancePanelCache(ctx, inst.ID, url)
	return url, nil
}

func (s *Service) RenewNow(ctx context.Context, inst domain.VPSInstance, days int) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	if days <= 0 {
		days = 30
	}
	next := time.Now().Add(time.Duration(days) * 24 * time.Hour)
	if inst.ExpireAt != nil && inst.ExpireAt.After(time.Now()) {
		next = inst.ExpireAt.Add(time.Duration(days) * 24 * time.Hour)
	}
	if err := cli.RenewHost(ctx, hostID, next); err != nil {
		return err
	}
	if inst.AdminStatus != domain.VPSAdminStatusNormal || inst.Status == domain.VPSStatusExpiredLocked {
		_ = cli.UnlockHost(ctx, hostID)
	}
	return s.vps.UpdateInstanceExpireAt(ctx, inst.ID, next)
}

func (s *Service) Start(ctx context.Context, inst domain.VPSInstance) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	return cli.StartHost(ctx, hostID)
}

func (s *Service) Shutdown(ctx context.Context, inst domain.VPSInstance) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	return cli.ShutdownHost(ctx, hostID)
}

func (s *Service) Reboot(ctx context.Context, inst domain.VPSInstance) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	return cli.RebootHost(ctx, hostID)
}

func (s *Service) ResetOS(ctx context.Context, inst domain.VPSInstance, templateID int64, password string) error {
	if s.automation == nil {
		return appshared.ErrInvalidInput
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return appshared.ErrInvalidInput
	}
	if templateID <= 0 {
		return appshared.ErrInvalidInput
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
		return appshared.ErrInvalidInput
	}
	validatedPassword, validateErr := trimAndValidateRequired(password, maxLenPassword)
	if validateErr != nil {
		return appshared.ErrInvalidInput
	}
	password = validatedPassword
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	if err := cli.ResetOS(ctx, hostID, templateID, password); err != nil {
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

func (s *Service) ResetOSPassword(ctx context.Context, inst domain.VPSInstance, password string) error {
	if s.automation == nil {
		return appshared.ErrInvalidInput
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return appshared.ErrInvalidInput
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return appshared.ErrInvalidInput
	}
	validatedPassword, validateErr := trimAndValidateRequired(password, maxLenPassword)
	if validateErr != nil {
		return appshared.ErrInvalidInput
	}
	password = validatedPassword
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	if err := cli.ResetOSPassword(ctx, hostID, password); err != nil {
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

func (s *Service) ListSnapshots(ctx context.Context, inst domain.VPSInstance) ([]AutomationSnapshot, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return nil, appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return nil, err
	}
	return cli.ListSnapshots(ctx, hostID)
}

func (s *Service) CreateSnapshot(ctx context.Context, inst domain.VPSInstance) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	return cli.CreateSnapshot(ctx, hostID)
}

func (s *Service) DeleteSnapshot(ctx context.Context, inst domain.VPSInstance, snapshotID int64) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 || snapshotID <= 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	return cli.DeleteSnapshot(ctx, hostID, snapshotID)
}

func (s *Service) RestoreSnapshot(ctx context.Context, inst domain.VPSInstance, snapshotID int64) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 || snapshotID <= 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	return cli.RestoreSnapshot(ctx, hostID, snapshotID)
}

func (s *Service) ListBackups(ctx context.Context, inst domain.VPSInstance) ([]AutomationBackup, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return nil, appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return nil, err
	}
	return cli.ListBackups(ctx, hostID)
}

func (s *Service) CreateBackup(ctx context.Context, inst domain.VPSInstance) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	return cli.CreateBackup(ctx, hostID)
}

func (s *Service) DeleteBackup(ctx context.Context, inst domain.VPSInstance, backupID int64) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 || backupID <= 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	return cli.DeleteBackup(ctx, hostID, backupID)
}

func (s *Service) RestoreBackup(ctx context.Context, inst domain.VPSInstance, backupID int64) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 || backupID <= 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	return cli.RestoreBackup(ctx, hostID, backupID)
}

func (s *Service) ListFirewallRules(ctx context.Context, inst domain.VPSInstance) ([]AutomationFirewallRule, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return nil, appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return nil, err
	}
	return cli.ListFirewallRules(ctx, hostID)
}

func (s *Service) AddFirewallRule(ctx context.Context, inst domain.VPSInstance, req AutomationFirewallRuleCreate) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return appshared.ErrInvalidInput
	}
	req.HostID = hostID
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	return cli.AddFirewallRule(ctx, req)
}

func (s *Service) DeleteFirewallRule(ctx context.Context, inst domain.VPSInstance, ruleID int64) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 || ruleID <= 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	return cli.DeleteFirewallRule(ctx, hostID, ruleID)
}

func (s *Service) ListPortMappings(ctx context.Context, inst domain.VPSInstance) ([]AutomationPortMapping, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return nil, appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return nil, err
	}
	return cli.ListPortMappings(ctx, hostID)
}

func (s *Service) AddPortMapping(ctx context.Context, inst domain.VPSInstance, req AutomationPortMappingCreate) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return appshared.ErrInvalidInput
	}
	if req.Name != "" {
		name, err := trimAndValidateOptional(req.Name, maxLenPortMappingName)
		if err != nil {
			return appshared.ErrInvalidInput
		}
		req.Name = name
	}
	req.HostID = hostID
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	return cli.AddPortMapping(ctx, req)
}

func (s *Service) DeletePortMapping(ctx context.Context, inst domain.VPSInstance, mappingID int64) error {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 || mappingID <= 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	return cli.DeletePortMapping(ctx, hostID, mappingID)
}

func (s *Service) FindPortCandidates(ctx context.Context, inst domain.VPSInstance, keywords string) ([]int64, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return nil, appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return nil, err
	}
	return cli.FindPortCandidates(ctx, hostID, keywords)
}

func (s *Service) Monitor(ctx context.Context, inst domain.VPSInstance) (AutomationMonitor, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return AutomationMonitor{}, appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return AutomationMonitor{}, err
	}
	return cli.GetMonitor(ctx, hostID)
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

func (s *Service) VNCURL(ctx context.Context, inst domain.VPSInstance) (string, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return "", appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return "", err
	}
	return cli.GetVNCURL(ctx, hostID)
}

func (s *Service) EmergencyRenew(ctx context.Context, inst domain.VPSInstance) (domain.VPSInstance, error) {
	if s.settings == nil {
		return domain.VPSInstance{}, appshared.ErrInvalidInput
	}
	policy := loadEmergencyRenewPolicy(ctx, s.settings)
	if !policy.Enabled {
		return domain.VPSInstance{}, appshared.ErrForbidden
	}
	if !emergencyRenewInWindow(time.Now(), inst.ExpireAt, policy.WindowDays) {
		return domain.VPSInstance{}, appshared.ErrForbidden
	}
	if inst.LastEmergencyRenewAt != nil {
		if time.Since(*inst.LastEmergencyRenewAt) < time.Duration(policy.IntervalHours)*time.Hour {
			return domain.VPSInstance{}, appshared.ErrConflict
		}
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return domain.VPSInstance{}, appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	next := time.Now().Add(time.Duration(policy.RenewDays) * 24 * time.Hour)
	if inst.ExpireAt != nil && inst.ExpireAt.After(time.Now()) {
		next = inst.ExpireAt.Add(time.Duration(policy.RenewDays) * 24 * time.Hour)
	}
	if err := cli.RenewHost(ctx, hostID, next); err != nil {
		return domain.VPSInstance{}, err
	}
	if inst.AdminStatus != domain.VPSAdminStatusNormal || inst.Status == domain.VPSStatusExpiredLocked {
		_ = cli.UnlockHost(ctx, hostID)
	}
	now := time.Now()
	_ = s.vps.UpdateInstanceExpireAt(ctx, inst.ID, next)
	_ = s.vps.UpdateInstanceEmergencyRenewAt(ctx, inst.ID, now)
	return s.vps.GetInstance(ctx, inst.ID)
}

func (s *Service) AutoDeleteExpired(ctx context.Context) error {
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
		cli, err := s.client(ctx, inst.GoodsTypeID)
		if err != nil {
			continue
		}
		if err := cli.DeleteHost(ctx, hostID); err != nil {
			continue
		}
		_ = s.vps.DeleteInstance(ctx, inst.ID)
	}
	return nil
}

func (s *Service) AutoLockExpired(ctx context.Context) error {
	if s.vps == nil || s.automation == nil {
		return nil
	}
	now := time.Now()
	items, err := s.vps.ListInstancesExpiring(ctx, now)
	if err != nil {
		return err
	}
	for _, inst := range items {
		if inst.ExpireAt == nil || inst.ExpireAt.After(now) {
			continue
		}
		if inst.Status == domain.VPSStatusLocked || inst.Status == domain.VPSStatusExpiredLocked {
			continue
		}
		if inst.AdminStatus != "" && inst.AdminStatus != domain.VPSAdminStatusNormal {
			continue
		}
		hostID := parseHostID(inst.AutomationInstanceID)
		if hostID == 0 {
			continue
		}
		cli, err := s.client(ctx, inst.GoodsTypeID)
		if err != nil {
			continue
		}
		if err := cli.LockHost(ctx, hostID); err != nil {
			continue
		}
		_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, domain.VPSStatusExpiredLocked, 10)
		_ = s.vps.UpdateInstanceAdminStatus(ctx, inst.ID, domain.VPSAdminStatusLocked)
	}
	return nil
}
