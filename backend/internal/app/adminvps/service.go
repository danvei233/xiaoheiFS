package adminvps

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type (
	AdminVPSCreateInput            = appshared.AdminVPSCreateInput
	AdminVPSUpdateInput            = appshared.AdminVPSUpdateInput
	AutomationCreateHostRequest    = appshared.AutomationCreateHostRequest
	AutomationElasticUpdateRequest = appshared.AutomationElasticUpdateRequest
)

type messageCenter interface {
	NotifyUser(ctx context.Context, userID int64, typ, title, content string) error
}

type Service struct {
	vps        appports.VPSRepository
	automation appports.AutomationClientResolver
	settings   appports.SettingsRepository
	audit      appports.AuditRepository
	users      appports.UserRepository
	messages   messageCenter
}

func NewService(vps appports.VPSRepository, automation appports.AutomationClientResolver, settings appports.SettingsRepository, audit appports.AuditRepository, users appports.UserRepository, messages messageCenter) *Service {
	return &Service{vps: vps, automation: automation, settings: settings, audit: audit, users: users, messages: messages}
}

func (s *Service) Get(ctx context.Context, vpsID int64) (domain.VPSInstance, error) {
	return s.vps.GetInstance(ctx, vpsID)
}

func (s *Service) Create(ctx context.Context, adminID int64, input AdminVPSCreateInput) (domain.VPSInstance, error) {
	if input.UserID == 0 || input.Name == "" {
		return domain.VPSInstance{}, appshared.ErrInvalidInput
	}
	if s.users != nil {
		if _, err := s.users.GetUserByID(ctx, input.UserID); err != nil {
			return domain.VPSInstance{}, appshared.ErrNotFound
		}
	}
	status := input.Status
	if status == "" {
		status = domain.VPSStatusUnknown
	}
	adminStatus := input.AdminStatus
	if adminStatus == "" {
		adminStatus = domain.VPSAdminStatusNormal
	}
	if input.Provision {
		if s.automation == nil {
			return domain.VPSInstance{}, appshared.ErrInvalidInput
		}
		if input.GoodsTypeID <= 0 {
			return domain.VPSInstance{}, appshared.ErrInvalidInput
		}
		if input.LineID <= 0 || input.OS == "" || input.CPU <= 0 || input.MemoryGB <= 0 || input.DiskGB <= 0 || input.BandwidthMB <= 0 {
			return domain.VPSInstance{}, appshared.ErrInvalidInput
		}
		cli, err := s.automation.ClientForGoodsType(ctx, input.GoodsTypeID)
		if err != nil {
			return domain.VPSInstance{}, err
		}
		expireAt := input.ExpireAt
		if expireAt == nil {
			t := time.Now().AddDate(0, 1, 0)
			expireAt = &t
		}
		portNum := input.PortNum
		if portNum <= 0 {
			portNum = 30
		}
		sysPwd := randomToken(12)
		vncPwd := randomToken(8)
		req := AutomationCreateHostRequest{
			LineID:     input.LineID,
			OS:         input.OS,
			CPU:        input.CPU,
			MemoryGB:   input.MemoryGB,
			DiskGB:     input.DiskGB,
			Bandwidth:  input.BandwidthMB,
			PortNum:    portNum,
			ExpireTime: *expireAt,
			HostName:   input.Name,
			SysPwd:     sysPwd,
			VNCPwd:     vncPwd,
		}
		res, err := cli.CreateHost(ctx, req)
		if err != nil {
			return domain.VPSInstance{}, err
		}
		hostID := res.HostID
		if hostID == 0 {
			if hosts, err := cli.ListHostSimple(ctx, input.Name); err == nil {
				for _, host := range hosts {
					if host.HostName == input.Name {
						hostID = host.ID
						break
					}
				}
			}
		}
		if hostID == 0 {
			return domain.VPSInstance{}, fmt.Errorf("host id not found")
		}
		input.AutomationInstanceID = fmt.Sprintf("%d", hostID)
		input.Status = domain.VPSStatusUnknown
		input.AutomationState = 0
		if info, err := cli.GetHostInfo(ctx, hostID); err == nil {
			input.Status = mapAutomationState(info.State)
			input.AutomationState = info.State
			if info.HostName != "" {
				input.Name = info.HostName
			}
			if info.ExpireAt != nil {
				input.ExpireAt = info.ExpireAt
			} else {
				input.ExpireAt = expireAt
			}
			input.AccessInfoJSON = mustJSON(map[string]any{
				"remote_ip":      info.RemoteIP,
				"panel_password": info.PanelPassword,
				"vnc_password":   info.VNCPassword,
				"os_password":    sysPwd,
			})
		} else if input.AccessInfoJSON == "" {
			input.AccessInfoJSON = mustJSON(map[string]any{
				"vnc_password": vncPwd,
				"os_password":  sysPwd,
			})
		}
	}
	if !input.Provision && input.GoodsTypeID > 0 && strings.TrimSpace(input.Name) != "" && strings.TrimSpace(input.AutomationInstanceID) == "" && s.automation != nil {
		cli, err := s.automation.ClientForGoodsType(ctx, input.GoodsTypeID)
		if err == nil {
			if hosts, listErr := cli.ListHostSimple(ctx, input.Name); listErr == nil {
				hostID := int64(0)
				for _, host := range hosts {
					if strings.EqualFold(strings.TrimSpace(host.HostName), strings.TrimSpace(input.Name)) {
						hostID = host.ID
						break
					}
				}
				if hostID == 0 && len(hosts) > 0 {
					hostID = hosts[0].ID
				}
				if hostID > 0 {
					input.AutomationInstanceID = fmt.Sprintf("%d", hostID)
					if info, infoErr := cli.GetHostInfo(ctx, hostID); infoErr == nil {
						input.Status = mapAutomationState(info.State)
						input.AutomationState = info.State
						if info.HostName != "" {
							input.Name = info.HostName
						}
						if info.ExpireAt != nil {
							input.ExpireAt = info.ExpireAt
						}
						if input.CPU <= 0 {
							input.CPU = info.CPU
						}
						if input.MemoryGB <= 0 {
							input.MemoryGB = info.MemoryGB
						}
						if input.DiskGB <= 0 {
							input.DiskGB = info.DiskGB
						}
						if input.BandwidthMB <= 0 {
							input.BandwidthMB = info.Bandwidth
						}
						if input.AccessInfoJSON == "" {
							input.AccessInfoJSON = mustJSON(map[string]any{
								"remote_ip":      info.RemoteIP,
								"panel_password": info.PanelPassword,
								"vnc_password":   info.VNCPassword,
								"os_password":    info.OSPassword,
							})
						}
					}
				}
			}
		}
	}
	status = input.Status
	if status == "" {
		status = domain.VPSStatusUnknown
	}
	inst := domain.VPSInstance{
		UserID:               input.UserID,
		OrderItemID:          input.OrderItemID,
		AutomationInstanceID: input.AutomationInstanceID,
		GoodsTypeID:          input.GoodsTypeID,
		Name:                 input.Name,
		Region:               input.Region,
		RegionID:             input.RegionID,
		LineID:               input.LineID,
		PackageID:            input.PackageID,
		PackageName:          input.PackageName,
		CPU:                  input.CPU,
		MemoryGB:             input.MemoryGB,
		DiskGB:               input.DiskGB,
		BandwidthMB:          input.BandwidthMB,
		PortNum:              input.PortNum,
		MonthlyPrice:         input.MonthlyPrice,
		SpecJSON:             input.SpecJSON,
		SystemID:             input.SystemID,
		Status:               status,
		AutomationState:      input.AutomationState,
		AdminStatus:          adminStatus,
		ExpireAt:             input.ExpireAt,
		PanelURLCache:        input.PanelURLCache,
		AccessInfoJSON:       input.AccessInfoJSON,
	}
	if err := s.vps.CreateInstance(ctx, &inst); err != nil {
		return domain.VPSInstance{}, err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "vps.create", TargetType: "vps", TargetID: fmt.Sprintf("%d", inst.ID), DetailJSON: mustJSON(map[string]any{"user_id": input.UserID})})
	}
	return s.vps.GetInstance(ctx, inst.ID)
}

func (s *Service) Refresh(ctx context.Context, adminID int64, vpsID int64) (domain.VPSInstance, error) {
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return domain.VPSInstance{}, appshared.ErrInvalidInput
	}
	cli, err := s.automation.ClientForGoodsType(ctx, inst.GoodsTypeID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	info, err := cli.GetHostInfo(ctx, hostID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	status := mapAutomationState(info.State)
	_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, status, info.State)
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "vps.refresh", TargetType: "vps", TargetID: fmt.Sprintf("%d", inst.ID), DetailJSON: "{}"})
	}
	return s.vps.GetInstance(ctx, inst.ID)
}

func (s *Service) SetAdminStatus(ctx context.Context, adminID int64, vpsID int64, status domain.VPSAdminStatus, reason string) error {
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return err
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.automation.ClientForGoodsType(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	switch status {
	case domain.VPSAdminStatusNormal:
		if err := cli.UnlockHost(ctx, hostID); err != nil {
			return err
		}
	case domain.VPSAdminStatusAbuse, domain.VPSAdminStatusFraud, domain.VPSAdminStatusLocked:
		if err := cli.LockHost(ctx, hostID); err != nil {
			return err
		}
	default:
		return appshared.ErrInvalidInput
	}
	if err := s.vps.UpdateInstanceAdminStatus(ctx, inst.ID, status); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "vps.admin_status", TargetType: "vps", TargetID: fmt.Sprintf("%d", inst.ID), DetailJSON: mustJSON(map[string]any{"status": status, "reason": reason})})
	}
	return nil
}

func (s *Service) EmergencyRenew(ctx context.Context, adminID int64, vpsID int64) (domain.VPSInstance, error) {
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return domain.VPSInstance{}, err
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
	cli, err := s.automation.ClientForGoodsType(ctx, inst.GoodsTypeID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	now := time.Now()
	expire := now.Add(time.Duration(policy.RenewDays) * 24 * time.Hour)
	if err := cli.UnlockHost(ctx, hostID); err != nil {
		return domain.VPSInstance{}, err
	}
	if err := cli.RenewHost(ctx, hostID, expire); err != nil {
		return domain.VPSInstance{}, err
	}
	if err := s.vps.UpdateInstanceExpireAt(ctx, inst.ID, expire); err != nil {
		return domain.VPSInstance{}, err
	}
	_ = s.vps.UpdateInstanceEmergencyRenewAt(ctx, inst.ID, now)
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "vps.emergency_renew", TargetType: "vps", TargetID: fmt.Sprintf("%d", inst.ID), DetailJSON: mustJSON(map[string]any{"days": policy.RenewDays})})
	}
	return s.vps.GetInstance(ctx, inst.ID)
}

func (s *Service) Resize(ctx context.Context, adminID int64, vpsID int64, req AutomationElasticUpdateRequest, specJSON string) error {
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return err
	}
	req.HostID = parseHostID(inst.AutomationInstanceID)
	if req.HostID == 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.automation.ClientForGoodsType(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	if err := cli.ElasticUpdate(ctx, req); err != nil {
		return err
	}
	if specJSON != "" {
		_ = s.vps.UpdateInstanceSpec(ctx, inst.ID, specJSON)
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "vps.resize", TargetType: "vps", TargetID: fmt.Sprintf("%d", inst.ID), DetailJSON: mustJSON(map[string]any{"spec": specJSON})})
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, adminID int64, vpsID int64) error {
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return err
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.automation.ClientForGoodsType(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	if err := cli.DeleteHost(ctx, hostID); err != nil {
		return err
	}
	if err := s.vps.DeleteInstance(ctx, inst.ID); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "vps.delete", TargetType: "vps", TargetID: fmt.Sprintf("%d", inst.ID), DetailJSON: "{}"})
	}
	if s.messages != nil {
		_ = s.messages.NotifyUser(ctx, inst.UserID, "vps_destroyed", "VPS Destroyed", "Your VPS "+inst.Name+" has been destroyed.")
	}
	return nil
}

func (s *Service) UpdateExpireAt(ctx context.Context, adminID int64, vpsID int64, expireAt time.Time) (domain.VPSInstance, error) {
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return domain.VPSInstance{}, appshared.ErrInvalidInput
	}
	cli, err := s.automation.ClientForGoodsType(ctx, inst.GoodsTypeID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	if err := cli.RenewHost(ctx, hostID, expireAt); err != nil {
		return domain.VPSInstance{}, err
	}
	if err := s.vps.UpdateInstanceExpireAt(ctx, inst.ID, expireAt); err != nil {
		return domain.VPSInstance{}, err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "vps.update_expire", TargetType: "vps", TargetID: fmt.Sprintf("%d", inst.ID), DetailJSON: mustJSON(map[string]any{"expire_at": expireAt})})
	}
	return s.vps.GetInstance(ctx, inst.ID)
}

func (s *Service) Update(ctx context.Context, adminID int64, vpsID int64, input AdminVPSUpdateInput) (domain.VPSInstance, error) {
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	if input.PackageID != nil {
		inst.PackageID = *input.PackageID
	}
	if input.PackageName != nil {
		inst.PackageName = strings.TrimSpace(*input.PackageName)
	}
	if input.MonthlyPrice != nil {
		inst.MonthlyPrice = *input.MonthlyPrice
	}
	if input.SystemID != nil {
		inst.SystemID = *input.SystemID
	}
	if input.SpecJSON != nil {
		inst.SpecJSON = *input.SpecJSON
	}
	if input.PanelURLCache != nil {
		inst.PanelURLCache = strings.TrimSpace(*input.PanelURLCache)
	}
	if input.AccessInfoJSON != nil {
		inst.AccessInfoJSON = *input.AccessInfoJSON
	}
	if input.Status != nil {
		inst.Status = *input.Status
	}
	if input.AdminStatus != nil {
		inst.AdminStatus = *input.AdminStatus
	}
	if input.CPU != nil {
		inst.CPU = *input.CPU
	}
	if input.MemoryGB != nil {
		inst.MemoryGB = *input.MemoryGB
	}
	if input.DiskGB != nil {
		inst.DiskGB = *input.DiskGB
	}
	if input.BandwidthMB != nil {
		inst.BandwidthMB = *input.BandwidthMB
	}
	if input.PortNum != nil {
		inst.PortNum = *input.PortNum
	}
	if strings.TrimSpace(input.SyncMode) == "automation" {
		if s.automation == nil {
			return domain.VPSInstance{}, appshared.ErrInvalidInput
		}
		hostID := parseHostID(inst.AutomationInstanceID)
		if hostID == 0 {
			return domain.VPSInstance{}, appshared.ErrInvalidInput
		}
		cli, err := s.automation.ClientForGoodsType(ctx, inst.GoodsTypeID)
		if err != nil {
			return domain.VPSInstance{}, err
		}
		req := AutomationElasticUpdateRequest{HostID: hostID}
		if input.CPU != nil {
			req.CPU = input.CPU
		}
		if input.MemoryGB != nil {
			req.MemoryGB = input.MemoryGB
		}
		if input.DiskGB != nil {
			req.DiskGB = input.DiskGB
		}
		if input.BandwidthMB != nil {
			req.Bandwidth = input.BandwidthMB
		}
		if input.PortNum != nil {
			req.PortNum = input.PortNum
		}
		if req.CPU != nil || req.MemoryGB != nil || req.DiskGB != nil || req.Bandwidth != nil || req.PortNum != nil {
			if err := cli.ElasticUpdate(ctx, req); err != nil {
				return domain.VPSInstance{}, err
			}
		}
		if input.AdminStatus != nil {
			if err := s.SetAdminStatus(ctx, adminID, inst.ID, *input.AdminStatus, "sync"); err != nil {
				return domain.VPSInstance{}, err
			}
		}
	}
	if err := s.vps.UpdateInstanceLocal(ctx, inst); err != nil {
		return domain.VPSInstance{}, err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "vps.update", TargetType: "vps", TargetID: fmt.Sprintf("%d", inst.ID), DetailJSON: mustJSON(map[string]any{"sync_mode": input.SyncMode})})
	}
	return s.vps.GetInstance(ctx, inst.ID)
}

type emergencyRenewPolicy struct {
	Enabled       bool
	WindowDays    int
	RenewDays     int
	IntervalHours int
}

func loadEmergencyRenewPolicy(ctx context.Context, settings appports.SettingsRepository) emergencyRenewPolicy {
	policy := emergencyRenewPolicy{
		Enabled:       true,
		WindowDays:    7,
		RenewDays:     1,
		IntervalHours: 720,
	}
	if settings == nil {
		return policy
	}
	if v, ok := getSettingBool(ctx, settings, "emergency_renew_enabled"); ok {
		policy.Enabled = v
	}
	if v, ok := getSettingInt(ctx, settings, "emergency_renew_window_days"); ok {
		policy.WindowDays = v
	}
	if v, ok := getSettingInt(ctx, settings, "emergency_renew_days"); ok {
		policy.RenewDays = v
	}
	if v, ok := getSettingInt(ctx, settings, "emergency_renew_interval_hours"); ok {
		policy.IntervalHours = v
	}
	if policy.WindowDays < 0 {
		policy.WindowDays = 0
	}
	if policy.RenewDays <= 0 {
		policy.RenewDays = 1
	}
	if policy.IntervalHours <= 0 {
		policy.IntervalHours = 24
	}
	return policy
}

func emergencyRenewInWindow(now time.Time, expireAt *time.Time, windowDays int) bool {
	if expireAt == nil {
		return false
	}
	if now.After(*expireAt) {
		return false
	}
	if windowDays <= 0 {
		return true
	}
	windowStart := expireAt.Add(-time.Duration(windowDays) * 24 * time.Hour)
	return !now.Before(windowStart)
}

func getSettingInt(ctx context.Context, repo appports.SettingsRepository, key string) (int, bool) {
	if repo == nil {
		return 0, false
	}
	setting, err := repo.GetSetting(ctx, key)
	if err != nil {
		return 0, false
	}
	val, err := strconv.Atoi(strings.TrimSpace(setting.ValueJSON))
	if err != nil {
		return 0, false
	}
	return val, true
}

func getSettingBool(ctx context.Context, repo appports.SettingsRepository, key string) (bool, bool) {
	if repo == nil {
		return false, false
	}
	setting, err := repo.GetSetting(ctx, key)
	if err != nil {
		return false, false
	}
	raw := strings.TrimSpace(setting.ValueJSON)
	if raw == "" {
		return false, false
	}
	switch strings.ToLower(raw) {
	case "true", "1", "yes":
		return true, true
	case "false", "0", "no":
		return false, true
	default:
		return false, false
	}
}

func parseHostID(v string) int64 {
	id, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	return id
}

func mapAutomationState(state int) domain.VPSStatus {
	switch state {
	case 0, 1, 13:
		return domain.VPSStatusProvisioning
	case 2:
		return domain.VPSStatusRunning
	case 3:
		return domain.VPSStatusStopped
	case 4:
		return domain.VPSStatusReinstalling
	case 5:
		return domain.VPSStatusReinstallFailed
	case 10:
		return domain.VPSStatusLocked
	default:
		return domain.VPSStatusUnknown
	}
}

func randomToken(n int) string {
	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	for i := range buf {
		buf[i] = letters[int(buf[i])%len(letters)]
	}
	return string(buf)
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
