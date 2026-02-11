package seed

import (
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
	settings := map[string]string{
		"default_line_id":                    "0",
		"default_port_num":                   "30",
		"payment_providers_enabled":          `{"approval":true,"balance":true,"custom":true,"yipay":false}`,
		"payment_providers_config":           `{"custom":{"pay_url":"","instructions":""},"yipay":{"base_url":"https://pays.org.cn/submit.php","pid":"","key":"","pay_type":"","notify_url":"","return_url":"","sign_type":"MD5"}}`,
		"payment_plugins":                    "[]",
		"payment_plugin_dir":                 "plugins/payment",
		"payment_plugin_upload_password":     "qweasd123456",
		"robot_webhook_url":                  "",
		"robot_webhook_secret":               "",
		"robot_webhook_enabled":              "false",
		"robot_webhooks":                     "[]",
		"realname_enabled":                   "false",
		"realname_provider":                  "idcard_cn",
		"realname_block_actions":             `["purchase_vps"]`,
		"smtp_host":                          "",
		"smtp_port":                          "",
		"smtp_user":                          "",
		"smtp_pass":                          "",
		"smtp_from":                          "",
		"smtp_enabled":                       "false",
		"email_enabled":                      "true",
		"email_expire_enabled":               "true",
		"expire_reminder_days":               "7",
		"emergency_renew_enabled":            "true",
		"emergency_renew_window_days":        "7",
		"emergency_renew_days":               "1",
		"emergency_renew_interval_hours":     "720",
		"auto_delete_enabled":                "false",
		"auto_delete_days":                   "7",
		"refund_full_days":                   "1",
		"refund_prorate_days":                "7",
		"refund_no_refund_days":              "30",
		"refund_full_hours":                  "0",
		"refund_prorate_hours":               "0",
		"refund_no_refund_hours":             "0",
		"refund_curve_json":                  "[]",
		"refund_requires_approval":           "true",
		"refund_on_admin_delete":             "true",
		"resize_price_mode":                  "remaining",
		"resize_refund_ratio":                "1",
		"resize_rounding":                    "round",
		"resize_min_charge":                  "0",
		"resize_min_refund":                  "0",
		"resize_charge_curve_json":           "[]",
		"resize_refund_to_wallet":            "true",
		"debug_enabled":                      "false",
		"automation_base_url":                "",
		"automation_api_key":                 "",
		"automation_enabled":                 "true",
		"automation_timeout_sec":             "12",
		"automation_retry":                   "0",
		"automation_dry_run":                 "false",
		"automation_log_retention_days":      "0",
		"task.vps_refresh":                   `{"enabled":true,"strategy":"interval","interval_sec":300}`,
		"task.order_provision_watchdog":      `{"enabled":true,"strategy":"interval","interval_sec":5}`,
		"provision_watchdog_max_jobs":        "8",
		"provision_watchdog_max_minutes":     "20",
		"task.expire_reminder":               `{"enabled":true,"strategy":"daily","daily_at":"09:00"}`,
		"task.vps_expire_cleanup":            `{"enabled":true,"strategy":"daily","daily_at":"03:00"}`,
		"site_name":                          "Cloud Console",
		"site_url":                           "",
		"logo_url":                           "",
		"favicon_url":                        "",
		"site_description":                   "",
		"site_keywords":                      "",
		"company_name":                       "",
		"contact_phone":                      "",
		"contact_email":                      "",
		"contact_qq":                         "",
		"wechat_qrcode":                      "",
		"icp_number":                         "",
		"psbe_number":                        "",
		"maintenance_mode":                   "false",
		"maintenance_message":                "We are under maintenance, please check back later.",
		"analytics_code":                     "",
		"site_logo":                          "",
		"site_icp":                           "",
		"site_maintenance_mode":              "false",
		"site_maintenance_message":           "We are under maintenance, please check back later.",
		"site_nav_items":                     `[]`,
		"auth_register_enabled":              "true",
		"auth_register_required_fields":      `["username","email","password"]`,
		"auth_password_min_len":              "6",
		"auth_password_require_upper":        "false",
		"auth_password_require_lower":        "false",
		"auth_password_require_number":       "false",
		"auth_password_require_symbol":       "false",
		"auth_register_verify_type":          "none",
		"auth_register_verify_ttl_sec":       "600",
		"auth_register_captcha_enabled":      "true",
		"auth_register_email_subject":        "Your verification code",
		"auth_register_email_body":           "Your verification code is: {{code}}",
		"auth_register_sms_plugin_id":        "",
		"auth_register_sms_instance_id":      "default",
		"auth_register_sms_template_id":      "",
		"auth_login_captcha_enabled":         "false",
		"auth_login_rate_limit_enabled":      "true",
		"auth_login_rate_limit_window_sec":   "300",
		"auth_login_rate_limit_max_attempts": "5",
	}

	rows := make([]settingSeedRow, 0, len(settings))
	for key, val := range settings {
		rows = append(rows, settingSeedRow{Key: key, ValueJSON: val, UpdatedAt: time.Now()})
	}
	return gdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoNothing: true,
	}).Create(&rows).Error
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
	return gdb.Model(&permissionGroupSeedRow{}).
		Where("id = ?", group.ID).
		Update("permissions_json", string(b)).Error
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
