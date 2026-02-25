package seed

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type planSeed struct {
	Name     string
	LineID   int64
	UnitCore int64
	UnitMem  int64
	UnitDisk int64
	UnitBW   int64
	Packages []pkgSeed
}

type pkgSeed struct {
	Name        string
	Cores       int
	MemoryGB    int
	DiskGB      int
	BandwidthMB int
	CPUModel    string
	Monthly     int64
}

func SeedIfEmpty(gdb *gorm.DB) error {
	var count int64
	if err := gdb.Table("regions").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaultLineID := getSettingInt(gdb, "default_line_id", 0)
	if defaultLineID < 0 {
		defaultLineID = 0
	}
	portNumDefault := getSettingInt(gdb, "default_port_num", 30)
	if portNumDefault <= 0 {
		portNumDefault = 30
	}

	plans := []planSeed{
		{
			Name:     "E5-2667 v2",
			LineID:   1,
			UnitCore: 500, UnitMem: 400, UnitDisk: 100, UnitBW: 1000,
			Packages: []pkgSeed{
				{Name: "2C4G 40G 10M", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2667 v2", Monthly: 1500},
				{Name: "4C8G 40G 10M", Cores: 4, MemoryGB: 8, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2667 v2", Monthly: 2000},
				{Name: "6C16G 40G 15M", Cores: 6, MemoryGB: 16, DiskGB: 40, BandwidthMB: 15, CPUModel: "E5-2667 v2", Monthly: 3000},
			},
		},
		{
			Name:     "E5-2697 v4",
			LineID:   3,
			UnitCore: 400, UnitMem: 400, UnitDisk: 100, UnitBW: 1000,
			Packages: []pkgSeed{
				{Name: "4C8G 40G 10M", Cores: 4, MemoryGB: 8, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2697 v4", Monthly: 1500},
				{Name: "8C16G 40G 10M", Cores: 8, MemoryGB: 16, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2697 v4", Monthly: 3000},
				{Name: "16C32G 40G 10M", Cores: 16, MemoryGB: 32, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2697 v4", Monthly: 7000},
			},
		},
		{
			Name:     "AMD R7 7840H",
			LineID:   4,
			UnitCore: 800, UnitMem: 600, UnitDisk: 100, UnitBW: 1000,
			Packages: []pkgSeed{
				{Name: "2C4G 40G 10M", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R7 7840H", Monthly: 4000},
				{Name: "4C8G 40G 10M", Cores: 4, MemoryGB: 8, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R7 7840H", Monthly: 7000},
				{Name: "8C24G 40G 10M", Cores: 8, MemoryGB: 24, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R7 7840H", Monthly: 17000},
			},
		},
	}

	systemImages := []systemImageSeedRow{
		{ImageID: 0, Name: "Ubuntu 22.04", Type: "linux", Enabled: 1},
		{ImageID: 0, Name: "Debian 12", Type: "linux", Enabled: 1},
		{ImageID: 0, Name: "Windows Server 2022", Type: "windows", Enabled: 1},
	}

	provisionBody := `<!DOCTYPE html><html><body><h2>VPS Provisioned</h2><p>Hi {{.user.username}},</p><p>Your VPS for order <strong>{{.order.no}}</strong> is now active.</p></body></html>`
	expireBody := `<!DOCTYPE html><html><body><h2>VPS Expiration Reminder</h2><p>Hi {{.user.username}},</p><p>Your VPS <strong>{{.vps.name}}</strong> will expire on <strong>{{.vps.expire_at}}</strong>.</p></body></html>`
	approvedBody := `<!DOCTYPE html><html><body><h2>Order Approved</h2><p>Hi {{.user.username}},</p><p>Your order <strong>{{.order.no}}</strong> has been approved.</p></body></html>`
	rejectedBody := `<!DOCTYPE html><html><body><h2>Order Rejected</h2><p>Hi {{.user.username}},</p><p>Your order <strong>{{.order.no}}</strong> has been rejected.</p></body></html>`
	passwordResetBody := `<!DOCTYPE html><html><body><h2>Password Reset</h2><p>Hi {{.user.username}},</p><p>Your reset token is: <strong>{{.token}}</strong></p></body></html>`

	permissionGroups := []permissionGroupSeedRow{
		{Name: "超级管理员", Description: "拥有所有权限", PermissionsJSON: `["*"]`},
		{Name: "运维管理员", Description: "负责VPS运维和订单审核", PermissionsJSON: `["user.list","user.view","order.list","order.view","order.approve","order.reject","vps.*","audit_log.view","scheduled_tasks.*"]`},
		{Name: "客服管理员", Description: "负责用户和订单查询", PermissionsJSON: `["user.list","user.view","order.list","order.view","vps.list","vps.view"]`},
		{Name: "财务管理员", Description: "负责订单审核和财务管理", PermissionsJSON: `["order.list","order.view","order.approve","order.reject","audit_log.view"]`},
	}

	billingCycles := []billingCycleSeedRow{
		{Name: "monthly", Months: 1, Multiplier: 1.0, MinQty: 1, MaxQty: 24, Active: 1, SortOrder: 1},
		{Name: "quarterly", Months: 3, Multiplier: 2.8, MinQty: 1, MaxQty: 12, Active: 1, SortOrder: 2},
		{Name: "yearly", Months: 12, Multiplier: 10.0, MinQty: 1, MaxQty: 5, Active: 1, SortOrder: 3},
	}

	return gdb.Transaction(func(tx *gorm.DB) error {
		region1 := regionSeedRow{GoodsTypeID: 0, Code: "area-1", Name: "晋中", Active: 1}
		if err := tx.Create(&region1).Error; err != nil {
			return err
		}
		if err := tx.Create(&regionSeedRow{GoodsTypeID: 0, Code: "area-2", Name: "宁波", Active: 0}).Error; err != nil {
			return err
		}

		for idx, plan := range plans {
			lineID := plan.LineID
			if lineID == 0 {
				lineID = int64(defaultLineID)
			}
			planRow := planGroupSeedRow{
				GoodsTypeID:       0,
				RegionID:          region1.ID,
				Name:              plan.Name,
				LineID:            lineID,
				UnitCore:          plan.UnitCore,
				UnitMem:           plan.UnitMem,
				UnitDisk:          plan.UnitDisk,
				UnitBW:            plan.UnitBW,
				AddCoreMin:        0,
				AddCoreMax:        0,
				AddCoreStep:       1,
				AddMemMin:         0,
				AddMemMax:         0,
				AddMemStep:        1,
				AddDiskMin:        0,
				AddDiskMax:        0,
				AddDiskStep:       1,
				AddBWMin:          0,
				AddBWMax:          0,
				AddBWStep:         1,
				Active:            1,
				Visible:           1,
				CapacityRemaining: -1,
				SortOrder:         idx,
			}
			if err := tx.Create(&planRow).Error; err != nil {
				return err
			}
			for pidx, pkg := range plan.Packages {
				pkgRow := packageSeedRow{
					GoodsTypeID:       0,
					PlanGroupID:       planRow.ID,
					ProductID:         0,
					Name:              pkg.Name,
					Cores:             pkg.Cores,
					MemoryGB:          pkg.MemoryGB,
					DiskGB:            pkg.DiskGB,
					BandwidthMbps:     pkg.BandwidthMB,
					CPUModel:          pkg.CPUModel,
					MonthlyPrice:      pkg.Monthly,
					PortNum:           portNumDefault,
					SortOrder:         pidx,
					Active:            1,
					Visible:           1,
					CapacityRemaining: -1,
				}
				if err := tx.Create(&pkgRow).Error; err != nil {
					return err
				}
			}
		}

		lineIDSet := map[int64]struct{}{}
		for _, plan := range plans {
			if plan.LineID > 0 {
				lineIDSet[plan.LineID] = struct{}{}
				continue
			}
			if defaultLineID > 0 {
				lineIDSet[int64(defaultLineID)] = struct{}{}
			}
		}

		for i := range systemImages {
			if err := tx.Create(&systemImages[i]).Error; err != nil {
				return err
			}
			for lineID := range lineIDSet {
				link := lineSystemImageSeedRow{LineID: lineID, SystemImageID: systemImages[i].ID}
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "line_id"}, {Name: "system_image_id"}},
					DoNothing: true,
				}).Create(&link).Error; err != nil {
					return err
				}
			}
		}

		emailTemplates := []emailTemplateSeedRow{
			{Name: "provision_success", Subject: "VPS Provisioned: Order {{.order.no}}", Body: provisionBody, Enabled: 1},
			{Name: "expire_reminder", Subject: "VPS Expiration Reminder: {{.vps.name}}", Body: expireBody, Enabled: 1},
			{Name: "order_approved", Subject: "Order Approved: {{.order.no}}", Body: approvedBody, Enabled: 1},
			{Name: "order_rejected", Subject: "Order Rejected: {{.order.no}}", Body: rejectedBody, Enabled: 1},
			{Name: "password_reset", Subject: "Password Reset", Body: passwordResetBody, Enabled: 1},
			{Name: "register_verify_code", Subject: "注册验证码", Body: "您好，您的注册验证码是：{{code}}，请在有效期内完成验证。", Enabled: 1},
			{Name: "login_ip_change_alert", Subject: "登录提醒", Body: "您的账号于 {{time}} 在 {{city}} 登录（IP：{{ip}}）。如非本人操作请立即修改密码。", Enabled: 1},
			{Name: "password_reset_verify_code", Subject: "找回密码验证码", Body: "您好，您正在进行找回密码操作，验证码：{{code}}，10分钟内有效。", Enabled: 1},
			{Name: "email_bind_verify_code", Subject: "邮箱绑定验证码", Body: "您的邮箱绑定验证码：{{code}}，10分钟内有效。", Enabled: 1},
			{Name: "email_change_alert_old_contact", Subject: "邮箱变更安全提醒", Body: "您的账号邮箱已于 {{time}} 从 {{old_email}} 修改为 {{new_email}}。如非本人操作，请立即修改密码并检查账号安全。", Enabled: 1},
		}
		for i := range emailTemplates {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoNothing: true,
			}).Create(&emailTemplates[i]).Error; err != nil {
				return err
			}
		}

		for i := range permissionGroups {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoNothing: true,
			}).Create(&permissionGroups[i]).Error; err != nil {
				return err
			}
		}

		for i := range billingCycles {
			if err := tx.Create(&billingCycles[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func EnsureSettings(gdb *gorm.DB) error {
	defaultPluginUploadPassword := generateSecureSeedPassword()
	settings := map[string]string{
		"default_line_id":                          "0",
		"default_port_num":                         "30",
		"payment_providers_enabled":                `{"approval":true,"balance":true}`,
		"payment_providers_config":                 `{}`,
		"payment_plugins":                          "[]",
		"payment_plugin_dir":                       "plugins/payment",
		"payment_plugin_upload_password":           defaultPluginUploadPassword,
		"robot_webhook_url":                        "",
		"robot_webhook_secret":                     "",
		"robot_webhook_enabled":                    "false",
		"realname_enabled":                         "false",
		"realname_provider":                        "idcard_cn",
		"smtp_host":                                "",
		"smtp_port":                                "",
		"smtp_user":                                "",
		"smtp_pass":                                "",
		"smtp_from":                                "",
		"smtp_enabled":                             "false",
		"sms_enabled":                              "true",
		"sms_plugin_id":                            "",
		"sms_instance_id":                          "default",
		"sms_default_template_id":                  "",
		"sms_provider_template_id":                 "",
		"email_enabled":                            "true",
		"email_expire_enabled":                     "true",
		"expire_reminder_days":                     "7",
		"emergency_renew_enabled":                  "true",
		"emergency_renew_window_days":              "7",
		"emergency_renew_days":                     "1",
		"emergency_renew_interval_hours":           "720",
		"auto_delete_enabled":                      "false",
		"auto_delete_days":                         "7",
		"refund_full_days":                         "1",
		"refund_prorate_days":                      "7",
		"refund_no_refund_days":                    "30",
		"refund_full_hours":                        "0",
		"refund_prorate_hours":                     "0",
		"refund_no_refund_hours":                   "0",
		"refund_curve_json":                        "[]",
		"refund_requires_approval":                 "true",
		"refund_on_admin_delete":                   "true",
		"resize_price_mode":                        "remaining",
		"resize_refund_ratio":                      "1",
		"resize_rounding":                          "round",
		"resize_min_charge":                        "0",
		"resize_min_refund":                        "0",
		"resize_charge_curve_json":                 "[]",
		"resize_refund_to_wallet":                  "true",
		"debug_enabled":                            "false",
		"automation_base_url":                      "",
		"automation_api_key":                       "",
		"automation_enabled":                       "true",
		"automation_timeout_sec":                   "12",
		"automation_retry":                         "0",
		"automation_dry_run":                       "false",
		"automation_log_retention_days":            "30",
		"audit_log_retention_days":                 "90",
		"integration_sync_log_retention_days":      "30",
		"scheduled_task_run_retention_days":        "14",
		"probe_status_event_retention_days":        "30",
		"probe_log_session_retention_days":         "7",
		"provision_watchdog_max_jobs":              "8",
		"provision_watchdog_max_minutes":           "20",
		"site_name":                                "Cloud Console",
		"site_url":                                 "",
		"logo_url":                                 "",
		"favicon_url":                              "",
		"site_description":                         "",
		"site_keywords":                            "",
		"company_name":                             "",
		"contact_phone":                            "",
		"contact_email":                            "",
		"contact_qq":                               "",
		"wechat_qrcode":                            "",
		"icp_number":                               "",
		"psbe_number":                              "",
		"maintenance_mode":                         "false",
		"maintenance_message":                      "We are under maintenance, please check back later.",
		"analytics_code":                           "",
		"site_logo":                                "",
		"site_icp":                                 "",
		"site_maintenance_mode":                    "false",
		"site_maintenance_message":                 "We are under maintenance, please check back later.",
		"auth_register_enabled":                    "true",
		"auth_password_min_len":                    "6",
		"auth_password_require_upper":              "false",
		"auth_password_require_lower":              "false",
		"auth_password_require_number":             "false",
		"auth_password_require_symbol":             "false",
		"auth_register_verify_type":                "none",
		"auth_register_email_required":             "true",
		"auth_register_verify_ttl_sec":             "600",
		"auth_register_captcha_enabled":            "true",
		"auth_captcha_provider":                    "image",
		"auth_geetest_captcha_id":                  "",
		"auth_geetest_captcha_key":                 "",
		"auth_geetest_api_server":                  "https://gcaptcha4.geetest.com",
		"auth_register_email_subject":              "Your verification code",
		"auth_register_email_body":                 "Your verification code is: {{code}}",
		"auth_register_sms_plugin_id":              "",
		"auth_register_sms_instance_id":            "default",
		"auth_register_sms_template_id":            "",
		"auth_login_captcha_enabled":               "false",
		"auth_login_rate_limit_enabled":            "true",
		"auth_login_rate_limit_window_sec":         "300",
		"auth_login_rate_limit_max_attempts":       "5",
		"auth_login_notify_enabled":                "true",
		"auth_login_notify_on_first_login":         "true",
		"auth_login_notify_on_ip_change":           "true",
		"auth_geoip_mmdb_path":                     "",
		"auth_password_reset_enabled":              "true",
		"auth_password_reset_verify_ttl_sec":       "600",
		"auth_sms_code_len":                        "6",
		"auth_sms_code_complexity":                 "digits",
		"auth_email_code_len":                      "6",
		"auth_email_code_complexity":               "alnum",
		"auth_captcha_code_len":                    "5",
		"auth_captcha_code_complexity":             "alnum",
		"auth_email_bind_enabled":                  "true",
		"auth_phone_bind_enabled":                  "true",
		"auth_contact_change_notify_old_enabled":   "true",
		"auth_contact_bind_verify_ttl_sec":         "600",
		"auth_bind_require_password_when_no_2fa":   "false",
		"auth_rebind_require_password_when_no_2fa": "true",
		"auth_2fa_enabled":                         "true",
		"auth_2fa_bind_enabled":                    "true",
		"auth_2fa_rebind_enabled":                  "true",
		"probe_heartbeat_interval_sec":             "20",
		"probe_snapshot_interval_sec":              "60",
		"probe_offline_grace_sec":                  "90",
		"probe_sla_window_days":                    "7",
		"probe_log_session_ttl_sec":                "600",
		"probe_log_chunk_max_bytes":                "16384",
		"probe_log_file_source":                    "file:logs",
	}

	rows := make([]settingSeedRow, 0, len(settings))
	for key, val := range settings {
		rows = append(rows, settingSeedRow{Key: key, ValueJSON: val, UpdatedAt: time.Now()})
	}
	if err := gdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoNothing: true,
	}).Create(&rows).Error; err != nil {
		return err
	}
	// Backfill historical empty values to avoid frontend falling back to display defaults.
	if err := ensureSettingNotBlank(gdb, "site_nav_items", `[]`); err != nil {
		return err
	}
	if err := ensureDefaultListSettingValues(gdb); err != nil {
		return err
	}
	if err := ensureDefaultScheduledTaskConfigs(gdb); err != nil {
		return err
	}
	if err := ensureDefaultRobotWebhooks(gdb); err != nil {
		return err
	}
	return EnsureMessageTemplateDefaults(gdb)
}

func generateSecureSeedPassword() string {
	secret := make([]byte, 48)
	if _, err := rand.Read(secret); err != nil {
		return "w6jW0pWj7Dq9Wq2BY8ahb0gNZXf3vLQ1-rg4r_4pKjv9Sm7ESe8B6y6M-R3qp9tH"
	}
	return base64.RawURLEncoding.EncodeToString(secret)
}

func ensureSettingNotBlank(gdb *gorm.DB, key, defaultValue string) error {
	var row settingSeedRow
	err := gdb.Where("`key` = ?", key).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return gdb.Create(&settingSeedRow{
				Key:       key,
				ValueJSON: defaultValue,
				UpdatedAt: time.Now(),
			}).Error
		}
		return err
	}
	if strings.TrimSpace(row.ValueJSON) != "" {
		return nil
	}
	return gdb.Model(&settingSeedRow{}).Where("`key` = ?", key).Updates(map[string]any{
		"value_json": defaultValue,
		"updated_at": time.Now(),
	}).Error
}

func EnsureMessageTemplateDefaults(gdb *gorm.DB) error {
	if gdb == nil {
		return nil
	}
	if err := ensureDefaultEmailTemplates(gdb); err != nil {
		return err
	}
	return ensureDefaultSMSTemplates(gdb)
}

func ensureDefaultEmailTemplates(gdb *gorm.DB) error {
	templates := []emailTemplateSeedRow{
		{Name: "provision_success", Subject: "VPS Provisioned: Order {{.order.no}}", Body: `<!DOCTYPE html><html><body><h2>VPS Provisioned</h2><p>Hi {{.user.username}},</p><p>Your VPS for order <strong>{{.order.no}}</strong> is now active.</p></body></html>`, Enabled: 1},
		{Name: "expire_reminder", Subject: "VPS Expiration Reminder: {{.vps.name}}", Body: `<!DOCTYPE html><html><body><h2>VPS Expiration Reminder</h2><p>Hi {{.user.username}},</p><p>Your VPS <strong>{{.vps.name}}</strong> will expire on <strong>{{.vps.expire_at}}</strong>.</p></body></html>`, Enabled: 1},
		{Name: "order_approved", Subject: "Order Approved: {{.order.no}}", Body: `<!DOCTYPE html><html><body><h2>Order Approved</h2><p>Hi {{.user.username}},</p><p>Your order <strong>{{.order.no}}</strong> has been approved.</p></body></html>`, Enabled: 1},
		{Name: "order_rejected", Subject: "Order Rejected: {{.order.no}}", Body: `<!DOCTYPE html><html><body><h2>Order Rejected</h2><p>Hi {{.user.username}},</p><p>Your order <strong>{{.order.no}}</strong> has been rejected.</p></body></html>`, Enabled: 1},
		{Name: "password_reset", Subject: "Password Reset", Body: `<!DOCTYPE html><html><body><h2>Password Reset</h2><p>Hi {{.user.username}},</p><p>Your reset token is: <strong>{{.token}}</strong></p></body></html>`, Enabled: 1},
		{Name: "register_verify_code", Subject: "注册验证码", Body: "您好，您的注册验证码是：{{code}}，请在有效期内完成验证。", Enabled: 1},
		{Name: "login_ip_change_alert", Subject: "登录提醒", Body: "您的账号于 {{time}} 在 {{city}} 登录（IP：{{ip}}）。如非本人操作请立即修改密码。", Enabled: 1},
		{Name: "password_reset_verify_code", Subject: "找回密码验证码", Body: "您好，您正在进行找回密码操作，验证码：{{code}}，10分钟内有效。", Enabled: 1},
		{Name: "email_bind_verify_code", Subject: "邮箱绑定验证码", Body: "您的邮箱绑定验证码：{{code}}，10分钟内有效。", Enabled: 1},
		{Name: "email_change_alert_old_contact", Subject: "邮箱变更安全提醒", Body: "您的账号邮箱已于 {{time}} 从 {{old_email}} 修改为 {{new_email}}。如非本人操作，请立即修改密码并检查账号安全。", Enabled: 1},
	}
	for i := range templates {
		if err := gdb.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).Create(&templates[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureDefaultSMSTemplates(gdb *gorm.DB) error {
	type smsTemplateSeed struct {
		ID        int64     `json:"id"`
		Name      string    `json:"name"`
		Content   string    `json:"content"`
		Enabled   bool      `json:"enabled"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	defaultItems := []smsTemplateSeed{
		{ID: 1, Name: "register_verify_code", Content: "【XXX】您正在注册XXX平台账号，验证码是：{{code}}，3分钟内有效，请及时输入。", Enabled: true},
		{ID: 2, Name: "login_ip_change_alert", Content: "【XXX】登录提醒：您的账号于 {{time}} 在 {{city}} 发生登录（IP：{{ip}}）。如为本人操作，请忽略本消息；如非本人操作，请立即修改密码并开启二次验证，确保账号安全。", Enabled: true},
		{ID: 3, Name: "password_reset_verify_code", Content: "【XXX】您好，您在XXX平台（APP）的账号正在进行找回密码操作，切勿将验证码泄露于他人，10分钟内有效。验证码：{{code}}。", Enabled: true},
		{ID: 4, Name: "phone_bind_verify_code", Content: "【XXX】手机绑定验证码：{{code}}，感谢您的支持！如非本人操作，请忽略本短信。", Enabled: true},
		{ID: 5, Name: "phone_change_alert_old_contact", Content: "【XXX】安全提醒：您的账号手机号已于 {{time}} 从 {{old_phone}} 修改为 {{new_phone}}。如非本人操作，请立即修改密码并联系管理员。", Enabled: true},
	}
	now := time.Now()
	for i := range defaultItems {
		defaultItems[i].CreatedAt = now
		defaultItems[i].UpdatedAt = now
	}

	var existing []smsTemplateStoreSeedRow
	if err := gdb.Order("id ASC").Find(&existing).Error; err != nil {
		return err
	}
	if len(existing) == 0 {
		var legacy settingSeedRow
		if err := gdb.Where("`key` = ?", "sms_templates_json").Take(&legacy).Error; err == nil {
			var parsed []smsTemplateSeed
			if json.Unmarshal([]byte(strings.TrimSpace(legacy.ValueJSON)), &parsed) == nil && len(parsed) > 0 {
				defaultItems = parsed
			}
		}
		rows := make([]smsTemplateStoreSeedRow, 0, len(defaultItems))
		for _, item := range defaultItems {
			name := strings.TrimSpace(item.Name)
			content := strings.TrimSpace(item.Content)
			if name == "" || content == "" {
				continue
			}
			row := smsTemplateStoreSeedRow{
				Name:      name,
				Content:   content,
				Enabled:   boolToInt(item.Enabled),
				CreatedAt: now,
				UpdatedAt: now,
			}
			if item.ID > 0 {
				row.ID = item.ID
			}
			if !item.CreatedAt.IsZero() {
				row.CreatedAt = item.CreatedAt
			}
			if !item.UpdatedAt.IsZero() {
				row.UpdatedAt = item.UpdatedAt
			}
			rows = append(rows, row)
		}
		if len(rows) > 0 {
			if err := gdb.Create(&rows).Error; err != nil {
				return err
			}
		}
		return gdb.Where("`key` = ?", "sms_templates_json").Delete(&settingSeedRow{}).Error
	}

	byName := map[string]bool{}
	maxID := int64(0)
	for _, row := range existing {
		byName[strings.TrimSpace(row.Name)] = true
		if row.ID > maxID {
			maxID = row.ID
		}
	}
	for _, item := range defaultItems {
		if byName[item.Name] {
			continue
		}
		maxID++
		if err := gdb.Create(&smsTemplateStoreSeedRow{
			ID:        maxID,
			Name:      item.Name,
			Content:   item.Content,
			Enabled:   boolToInt(item.Enabled),
			CreatedAt: now,
			UpdatedAt: now,
		}).Error; err != nil {
			return err
		}
	}
	return gdb.Where("`key` = ?", "sms_templates_json").Delete(&settingSeedRow{}).Error
}

func ensureDefaultListSettingValues(gdb *gorm.DB) error {
	defaults := map[string][]string{
		"auth_register_required_fields": {"username", "email", "password"},
		"auth_register_verify_channels": {"email", "sms"},
		"auth_login_notify_channels":    {"email"},
		"auth_password_reset_channels":  {"email"},
		"realname_block_actions":        {"purchase_vps"},
	}
	for key, fallback := range defaults {
		var count int64
		if err := gdb.Model(&settingListValueSeedRow{}).Where("setting_key = ?", key).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			_ = gdb.Where("`key` = ?", key).Delete(&settingSeedRow{}).Error
			continue
		}
		values := append([]string{}, fallback...)
		loadedFromLegacy := false
		var legacy settingSeedRow
		if err := gdb.Where("`key` = ?", key).Take(&legacy).Error; err == nil {
			var parsed []string
			if json.Unmarshal([]byte(strings.TrimSpace(legacy.ValueJSON)), &parsed) == nil {
				values = parsed
				loadedFromLegacy = true
			}
		}
		if !loadedFromLegacy && len(values) == 0 {
			values = append(values, fallback...)
		}
		rows := make([]settingListValueSeedRow, 0, len(values))
		uniq := map[string]struct{}{}
		for i, raw := range values {
			v := strings.TrimSpace(raw)
			if v == "" {
				continue
			}
			if _, ok := uniq[v]; ok {
				continue
			}
			uniq[v] = struct{}{}
			rows = append(rows, settingListValueSeedRow{SettingKey: key, Value: v, SortOrder: i})
		}
		if len(rows) > 0 {
			if err := gdb.Create(&rows).Error; err != nil {
				return err
			}
			_ = gdb.Where("`key` = ?", key).Delete(&settingSeedRow{}).Error
			continue
		}
		if loadedFromLegacy {
			continue
		}
		_ = gdb.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value_json", "updated_at"}),
		}).Create(&settingSeedRow{Key: key, ValueJSON: "[]", UpdatedAt: time.Now()}).Error
	}
	return nil
}

func ensureDefaultScheduledTaskConfigs(gdb *gorm.DB) error {
	defaults := map[string]scheduledTaskConfigSeedRow{
		"vps_refresh":              {TaskKey: "vps_refresh", Enabled: 1, Strategy: "interval", IntervalSec: 300, DailyAt: ""},
		"order_provision_watchdog": {TaskKey: "order_provision_watchdog", Enabled: 1, Strategy: "interval", IntervalSec: 5, DailyAt: ""},
		"log_retention_cleanup":    {TaskKey: "log_retention_cleanup", Enabled: 1, Strategy: "daily", IntervalSec: 60, DailyAt: "03:30"},
		"expire_reminder":          {TaskKey: "expire_reminder", Enabled: 1, Strategy: "daily", IntervalSec: 60, DailyAt: "09:00"},
		"vps_expire_cleanup":       {TaskKey: "vps_expire_cleanup", Enabled: 1, Strategy: "daily", IntervalSec: 60, DailyAt: "03:00"},
	}
	for taskKey, def := range defaults {
		var count int64
		if err := gdb.Model(&scheduledTaskConfigSeedRow{}).Where("task_key = ?", taskKey).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			_ = gdb.Where("`key` = ?", "task."+taskKey).Delete(&settingSeedRow{}).Error
			continue
		}
		row := def
		var legacy settingSeedRow
		if err := gdb.Where("`key` = ?", "task."+taskKey).Take(&legacy).Error; err == nil {
			var payload struct {
				Enabled     *bool   `json:"enabled"`
				Strategy    *string `json:"strategy"`
				IntervalSec *int    `json:"interval_sec"`
				DailyAt     *string `json:"daily_at"`
			}
			if json.Unmarshal([]byte(strings.TrimSpace(legacy.ValueJSON)), &payload) == nil {
				if payload.Enabled != nil {
					row.Enabled = boolToInt(*payload.Enabled)
				}
				if payload.Strategy != nil && strings.TrimSpace(*payload.Strategy) != "" {
					row.Strategy = strings.TrimSpace(*payload.Strategy)
				}
				if payload.IntervalSec != nil && *payload.IntervalSec > 0 {
					row.IntervalSec = *payload.IntervalSec
				}
				if payload.DailyAt != nil {
					row.DailyAt = strings.TrimSpace(*payload.DailyAt)
				}
			}
		}
		if err := gdb.Create(&row).Error; err != nil {
			return err
		}
		_ = gdb.Where("`key` = ?", "task."+taskKey).Delete(&settingSeedRow{}).Error
	}
	return nil
}

func ensureDefaultRobotWebhooks(gdb *gorm.DB) error {
	var count int64
	if err := gdb.Model(&robotWebhookSeedRow{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		_ = gdb.Where("`key` = ?", "robot_webhooks").Delete(&settingSeedRow{}).Error
		return nil
	}
	var legacy settingSeedRow
	if err := gdb.Where("`key` = ?", "robot_webhooks").Take(&legacy).Error; err != nil {
		return nil
	}
	type webhookItem struct {
		Name    string   `json:"name"`
		URL     string   `json:"url"`
		Secret  string   `json:"secret"`
		Enabled bool     `json:"enabled"`
		Events  []string `json:"events"`
	}
	var items []webhookItem
	if err := json.Unmarshal([]byte(strings.TrimSpace(legacy.ValueJSON)), &items); err != nil {
		return nil
	}
	rows := make([]robotWebhookSeedRow, 0, len(items))
	for i, item := range items {
		if strings.TrimSpace(item.URL) == "" {
			continue
		}
		eventsRaw, _ := json.Marshal(item.Events)
		rows = append(rows, robotWebhookSeedRow{
			Name:       strings.TrimSpace(item.Name),
			URL:        strings.TrimSpace(item.URL),
			Secret:     strings.TrimSpace(item.Secret),
			Enabled:    boolToInt(item.Enabled),
			EventsJSON: string(eventsRaw),
			SortOrder:  i,
		})
	}
	if len(rows) > 0 {
		if err := gdb.Create(&rows).Error; err != nil {
			return err
		}
	}
	return gdb.Where("`key` = ?", "robot_webhooks").Delete(&settingSeedRow{}).Error
}

func EnsurePermissionDefaults(gdb *gorm.DB) error {
	if err := gdb.Model(&permissionSeedRow{}).
		Where("friendly_name IS NULL OR friendly_name = ''").
		Update("friendly_name", gorm.Expr("name")).Error; err != nil {
		return err
	}
	return gdb.Model(&permissionSeedRow{}).
		Where("parent_code IS NULL").
		Update("parent_code", "").Error
}

func EnsurePermissionGroups(gdb *gorm.DB) error {
	superAdminPerms := `["*"]`
	opsAdminPerms := `["dashboard.overview","dashboard.revenue","dashboard.vps_status","user.list","user.view","order.list","order.view","order.approve","order.reject","vps.*","audit_log.view","scheduled_tasks.*"]`
	csAdminPerms := `["dashboard.overview","dashboard.revenue","user.list","user.view","order.list","order.view","vps.list","vps.view"]`
	financeAdminPerms := `["dashboard.overview","dashboard.revenue","order.list","order.view","order.approve","order.reject","audit_log.view"]`

	groups := []permissionGroupSeedRow{
		{Name: "超级管理员", Description: "拥有所有权限", PermissionsJSON: superAdminPerms},
		{Name: "运维管理员", Description: "负责VPS运维和订单审核", PermissionsJSON: opsAdminPerms},
		{Name: "客服管理员", Description: "负责用户和订单查询", PermissionsJSON: csAdminPerms},
		{Name: "财务管理员", Description: "负责订单审核和财务管理", PermissionsJSON: financeAdminPerms},
	}

	for i := range groups {
		group := groups[i]
		if err := gdb.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "name"}},
			DoUpdates: clause.Assignments(map[string]any{
				"description":      group.Description,
				"permissions_json": group.PermissionsJSON,
				"updated_at":       time.Now(),
			}),
		}).Create(&group).Error; err != nil {
			return err
		}
		if err := syncPermissionGroupPermissionRows(gdb, group.ID, group.PermissionsJSON); err != nil {
			return err
		}
	}

	var allGroups []permissionGroupSeedRow
	if err := gdb.Order("id ASC").Find(&allGroups).Error; err != nil {
		return err
	}
	if len(allGroups) == 0 {
		return nil
	}

	superAdminGroupID := int64(0)
	for _, group := range allGroups {
		var perms []string
		if json.Unmarshal([]byte(group.PermissionsJSON), &perms) != nil {
			continue
		}
		for _, p := range perms {
			if strings.TrimSpace(p) == "*" {
				superAdminGroupID = group.ID
				break
			}
		}
		if superAdminGroupID > 0 {
			break
		}
	}
	if superAdminGroupID == 0 {
		for _, group := range allGroups {
			name := strings.TrimSpace(group.Name)
			if name == "超级管理员" || strings.Contains(strings.ToLower(name), "super") {
				superAdminGroupID = group.ID
				break
			}
		}
	}
	if superAdminGroupID == 0 {
		superAdminGroupID = allGroups[0].ID
	}

	if err := gdb.Model(&userSeedRow{}).
		Where("role = ? AND (permission_group_id IS NULL OR permission_group_id = 0)", "admin").
		Update("permission_group_id", superAdminGroupID).Error; err != nil {
		return err
	}

	var primaryAdmin userSeedRow
	if err := gdb.Where("role = ?", "admin").Order("id ASC").Take(&primaryAdmin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return gdb.Model(&userSeedRow{}).
		Where("id = ?", primaryAdmin.ID).
		Update("permission_group_id", superAdminGroupID).Error
}

func syncPermissionGroupPermissionRows(gdb *gorm.DB, groupID int64, permissionsJSON string) error {
	var perms []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(permissionsJSON)), &perms); err != nil {
		return err
	}
	rows := make([]permissionGroupPermissionSeedRow, 0, len(perms))
	uniq := map[string]struct{}{}
	for _, raw := range perms {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if _, ok := uniq[v]; ok {
			continue
		}
		uniq[v] = struct{}{}
		rows = append(rows, permissionGroupPermissionSeedRow{
			PermissionGroupID: groupID,
			PermissionCode:    v,
		})
	}
	return gdb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("permission_group_id = ?", groupID).Delete(&permissionGroupPermissionSeedRow{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Create(&rows).Error
	})
}

func EnsureCMSDefaults(gdb *gorm.DB) error {
	categories := []cmsCategorySeedRow{
		{Key: "tutorials", Name: "教程", Lang: "zh-CN", SortOrder: 1, Visible: 1},
		{Key: "docs", Name: "文档", Lang: "zh-CN", SortOrder: 2, Visible: 1},
		{Key: "announcements", Name: "公告", Lang: "zh-CN", SortOrder: 3, Visible: 1},
		{Key: "activities", Name: "活动", Lang: "zh-CN", SortOrder: 4, Visible: 1},
		{Key: "tutorials", Name: "Tutorials", Lang: "en-US", SortOrder: 1, Visible: 1},
		{Key: "docs", Name: "Docs", Lang: "en-US", SortOrder: 2, Visible: 1},
		{Key: "announcements", Name: "Announcements", Lang: "en-US", SortOrder: 3, Visible: 1},
		{Key: "activities", Name: "Activities", Lang: "en-US", SortOrder: 4, Visible: 1},
	}
	if err := gdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}, {Name: "lang"}},
		DoNothing: true,
	}).Create(&categories).Error; err != nil {
		return err
	}

	var blocksCount int64
	if err := gdb.Model(&cmsBlockSeedRow{}).Count(&blocksCount).Error; err != nil {
		return err
	}
	if blocksCount > 0 {
		return nil
	}

	blocks := []cmsBlockSeedRow{
		{Page: "home", Type: "hero", Title: "Build on Cloud", Subtitle: "Fast and stable", ContentJSON: `{"buttons":[{"text":"Get Started","url":"/register"}]}`, Lang: "en-US", Visible: 1, SortOrder: 1},
		{Page: "home", Type: "hero", Title: "云上构建", Subtitle: "稳定高效", ContentJSON: `{"buttons":[{"text":"立即开始","url":"/register"}]}`, Lang: "zh-CN", Visible: 1, SortOrder: 1},
		{Page: "products", Type: "intro", Title: "Products", Subtitle: "Core cloud services", ContentJSON: `{"items":["vps","storage","cdn"]}`, Lang: "en-US", Visible: 1, SortOrder: 1},
		{Page: "products", Type: "intro", Title: "产品", Subtitle: "核心云服务", ContentJSON: `{"items":["vps","storage","cdn"]}`, Lang: "zh-CN", Visible: 1, SortOrder: 1},
	}
	return gdb.Create(&blocks).Error
}

func ensurePermissionInGroup(gdb *gorm.DB, groupName string, permission string) error {
	var group permissionGroupSeedRow
	if err := gdb.Where("name = ?", groupName).First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var perms []string
	if group.PermissionsJSON != "" {
		if err := json.Unmarshal([]byte(group.PermissionsJSON), &perms); err != nil {
			return err
		}
	}
	for _, p := range perms {
		if permissionCovers(p, permission) {
			return nil
		}
	}
	perms = append(perms, permission)
	b, err := json.Marshal(perms)
	if err != nil {
		return err
	}
	if err := gdb.Model(&permissionGroupSeedRow{}).
		Where("id = ?", group.ID).
		Update("permissions_json", string(b)).Error; err != nil {
		return err
	}
	return syncPermissionGroupPermissionRows(gdb, group.ID, string(b))
}

func permissionCovers(entry string, permission string) bool {
	entry = strings.TrimSpace(entry)
	permission = strings.TrimSpace(permission)
	if entry == "" || permission == "" {
		return false
	}
	if entry == "*" || entry == permission {
		return true
	}
	if strings.HasSuffix(entry, ".*") {
		prefix := strings.TrimSuffix(entry, ".*")
		return strings.HasPrefix(permission, prefix+".")
	}
	return false
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func getSettingInt(gdb *gorm.DB, key string, fallback int) int {
	var setting settingSeedRow
	if err := gdb.Where("`key` = ?", key).Take(&setting).Error; err != nil {
		return fallback
	}
	val, err := strconv.Atoi(setting.ValueJSON)
	if err != nil {
		return fallback
	}
	return val
}

func SeedIfEmptyGorm(gdb *gorm.DB) error    { return SeedIfEmpty(gdb) }
func EnsureSettingsGorm(gdb *gorm.DB) error { return EnsureSettings(gdb) }
func EnsurePermissionDefaultsGorm(gdb *gorm.DB) error {
	return EnsurePermissionDefaults(gdb)
}
func EnsurePermissionGroupsGorm(gdb *gorm.DB) error { return EnsurePermissionGroups(gdb) }
func EnsureCMSDefaultsGorm(gdb *gorm.DB) error      { return EnsureCMSDefaults(gdb) }

type regionSeedRow struct {
	ID          int64 `gorm:"column:id;primaryKey;autoIncrement"`
	GoodsTypeID int64 `gorm:"column:goods_type_id"`
	Code        string
	Name        string
	Active      int
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (regionSeedRow) TableName() string { return "regions" }

type planGroupSeedRow struct {
	ID                int64 `gorm:"column:id;primaryKey;autoIncrement"`
	GoodsTypeID       int64 `gorm:"column:goods_type_id"`
	RegionID          int64 `gorm:"column:region_id"`
	Name              string
	LineID            int64 `gorm:"column:line_id"`
	UnitCore          int64 `gorm:"column:unit_core"`
	UnitMem           int64 `gorm:"column:unit_mem"`
	UnitDisk          int64 `gorm:"column:unit_disk"`
	UnitBW            int64 `gorm:"column:unit_bw"`
	AddCoreMin        int   `gorm:"column:add_core_min"`
	AddCoreMax        int   `gorm:"column:add_core_max"`
	AddCoreStep       int   `gorm:"column:add_core_step"`
	AddMemMin         int   `gorm:"column:add_mem_min"`
	AddMemMax         int   `gorm:"column:add_mem_max"`
	AddMemStep        int   `gorm:"column:add_mem_step"`
	AddDiskMin        int   `gorm:"column:add_disk_min"`
	AddDiskMax        int   `gorm:"column:add_disk_max"`
	AddDiskStep       int   `gorm:"column:add_disk_step"`
	AddBWMin          int   `gorm:"column:add_bw_min"`
	AddBWMax          int   `gorm:"column:add_bw_max"`
	AddBWStep         int   `gorm:"column:add_bw_step"`
	Active            int
	Visible           int
	CapacityRemaining int       `gorm:"column:capacity_remaining"`
	SortOrder         int       `gorm:"column:sort_order"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (planGroupSeedRow) TableName() string { return "plan_groups" }

type packageSeedRow struct {
	ID                int64 `gorm:"column:id;primaryKey;autoIncrement"`
	GoodsTypeID       int64 `gorm:"column:goods_type_id"`
	PlanGroupID       int64 `gorm:"column:plan_group_id"`
	ProductID         int64 `gorm:"column:product_id"`
	Name              string
	Cores             int
	MemoryGB          int    `gorm:"column:memory_gb"`
	DiskGB            int    `gorm:"column:disk_gb"`
	BandwidthMbps     int    `gorm:"column:bandwidth_mbps"`
	CPUModel          string `gorm:"column:cpu_model"`
	MonthlyPrice      int64  `gorm:"column:monthly_price"`
	PortNum           int    `gorm:"column:port_num"`
	SortOrder         int    `gorm:"column:sort_order"`
	Active            int
	Visible           int
	CapacityRemaining int       `gorm:"column:capacity_remaining"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (packageSeedRow) TableName() string { return "packages" }

type systemImageSeedRow struct {
	ID        int64 `gorm:"column:id;primaryKey;autoIncrement"`
	ImageID   int64 `gorm:"column:image_id"`
	Name      string
	Type      string
	Enabled   int
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (systemImageSeedRow) TableName() string { return "system_images" }

type lineSystemImageSeedRow struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement"`
	LineID        int64     `gorm:"column:line_id"`
	SystemImageID int64     `gorm:"column:system_image_id"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (lineSystemImageSeedRow) TableName() string { return "line_system_images" }

type emailTemplateSeedRow struct {
	ID        int64 `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string
	Subject   string
	Body      string
	Enabled   int
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (emailTemplateSeedRow) TableName() string { return "email_templates" }

type permissionGroupSeedRow struct {
	ID              int64 `gorm:"column:id;primaryKey;autoIncrement"`
	Name            string
	Description     string
	PermissionsJSON string `gorm:"column:permissions_json"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (permissionGroupSeedRow) TableName() string { return "permission_groups" }

type permissionGroupPermissionSeedRow struct {
	ID                int64     `gorm:"column:id;primaryKey;autoIncrement"`
	PermissionGroupID int64     `gorm:"column:permission_group_id"`
	PermissionCode    string    `gorm:"column:permission_code"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

func (permissionGroupPermissionSeedRow) TableName() string { return "permission_group_permissions" }

type billingCycleSeedRow struct {
	ID         int64 `gorm:"column:id;primaryKey;autoIncrement"`
	Name       string
	Months     int
	Multiplier float64
	MinQty     int `gorm:"column:min_qty"`
	MaxQty     int `gorm:"column:max_qty"`
	Active     int
	SortOrder  int       `gorm:"column:sort_order"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (billingCycleSeedRow) TableName() string { return "billing_cycles" }

type settingSeedRow struct {
	Key       string `gorm:"column:key;primaryKey"`
	ValueJSON string `gorm:"column:value_json"`
	UpdatedAt time.Time
}

func (settingSeedRow) TableName() string { return "settings" }

type smsTemplateStoreSeedRow struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string    `gorm:"column:name"`
	Content   string    `gorm:"column:content"`
	Enabled   int       `gorm:"column:enabled"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (smsTemplateStoreSeedRow) TableName() string { return "sms_templates" }

type settingListValueSeedRow struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	SettingKey string    `gorm:"column:setting_key"`
	Value      string    `gorm:"column:value"`
	SortOrder  int       `gorm:"column:sort_order"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (settingListValueSeedRow) TableName() string { return "setting_list_values" }

type scheduledTaskConfigSeedRow struct {
	TaskKey     string    `gorm:"column:task_key;primaryKey"`
	Enabled     int       `gorm:"column:enabled"`
	Strategy    string    `gorm:"column:strategy"`
	IntervalSec int       `gorm:"column:interval_sec"`
	DailyAt     string    `gorm:"column:daily_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (scheduledTaskConfigSeedRow) TableName() string { return "scheduled_task_configs" }

type robotWebhookSeedRow struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name       string    `gorm:"column:name"`
	URL        string    `gorm:"column:url"`
	Secret     string    `gorm:"column:secret"`
	Enabled    int       `gorm:"column:enabled"`
	EventsJSON string    `gorm:"column:events_json"`
	SortOrder  int       `gorm:"column:sort_order"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (robotWebhookSeedRow) TableName() string { return "robot_webhooks" }

type permissionSeedRow struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement"`
	FriendlyName string `gorm:"column:friendly_name"`
	Name         string
	ParentCode   string `gorm:"column:parent_code"`
}

func (permissionSeedRow) TableName() string { return "permissions" }

type userSeedRow struct {
	ID                int64 `gorm:"column:id;primaryKey;autoIncrement"`
	Role              string
	PermissionGroupID *int64 `gorm:"column:permission_group_id"`
}

func (userSeedRow) TableName() string { return "users" }

type cmsCategorySeedRow struct {
	ID        int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Key       string `gorm:"column:key"`
	Name      string
	Lang      string
	SortOrder int `gorm:"column:sort_order"`
	Visible   int
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (cmsCategorySeedRow) TableName() string { return "cms_categories" }

type cmsBlockSeedRow struct {
	ID          int64 `gorm:"column:id;primaryKey;autoIncrement"`
	Page        string
	Type        string
	Title       string
	Subtitle    string
	ContentJSON string `gorm:"type:longtext;column:content_json"`
	CustomHTML  string `gorm:"type:longtext;column:custom_html"`
	Lang        string
	Visible     int
	SortOrder   int       `gorm:"column:sort_order"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (cmsBlockSeedRow) TableName() string { return "cms_blocks" }
