package seed

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
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

func SeedIfEmpty(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM regions`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaultLineID := getSettingInt(db, "default_line_id", 0)
	if defaultLineID < 0 {
		defaultLineID = 0
	}
	portNumDefault := getSettingInt(db, "default_port_num", 30)
	if portNumDefault <= 0 {
		portNumDefault = 30
	}

	plans := []planSeed{
		{
			Name:     "E5-2667 v2",
			LineID:   1,
			UnitCore: 500, UnitMem: 400, UnitDisk: 100, UnitBW: 1000,
			Packages: []pkgSeed{
				{Name: "2核4G 50G 10M 3.6GHz", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2667 v2", Monthly: 1500},
				{Name: "4核8G 50G 10M 3.6GHz", Cores: 4, MemoryGB: 8, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2667 v2", Monthly: 2000},
				{Name: "4核12G 50G 15M 3.6GHz", Cores: 4, MemoryGB: 12, DiskGB: 40, BandwidthMB: 15, CPUModel: "E5-2667 v2", Monthly: 2500},
				{Name: "6核16G 50G 15M 3.6GHz", Cores: 6, MemoryGB: 16, DiskGB: 40, BandwidthMB: 15, CPUModel: "E5-2667 v2", Monthly: 3000},
				{Name: "8核24G 50G 20M 3.6GHz", Cores: 8, MemoryGB: 24, DiskGB: 40, BandwidthMB: 20, CPUModel: "E5-2667 v2", Monthly: 4000},
				{Name: "8核32G 50G 20M 3.6GHz", Cores: 8, MemoryGB: 32, DiskGB: 40, BandwidthMB: 20, CPUModel: "E5-2667 v2", Monthly: 7000},
				{Name: "10核36G 50G 20M 3.6GHz", Cores: 10, MemoryGB: 36, DiskGB: 40, BandwidthMB: 20, CPUModel: "E5-2667 v2", Monthly: 8500},
			},
		},
		{
			Name:     "E5-2697 v4",
			LineID:   3,
			UnitCore: 400, UnitMem: 400, UnitDisk: 100, UnitBW: 1000,
			Packages: []pkgSeed{
				{Name: "4核4G", Cores: 4, MemoryGB: 4, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2697 v4", Monthly: 1000},
				{Name: "4核8G", Cores: 4, MemoryGB: 8, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2697 v4", Monthly: 1500},
				{Name: "6核12G", Cores: 6, MemoryGB: 12, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2697 v4", Monthly: 2200},
				{Name: "8核16G", Cores: 8, MemoryGB: 16, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2697 v4", Monthly: 3000},
				{Name: "12核24G", Cores: 12, MemoryGB: 24, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2697 v4", Monthly: 4000},
				{Name: "16核32G", Cores: 16, MemoryGB: 32, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2697 v4", Monthly: 7000},
				{Name: "16核36G", Cores: 16, MemoryGB: 36, DiskGB: 40, BandwidthMB: 10, CPUModel: "E5-2697 v4", Monthly: 8000},
			},
		},
		{
			Name:     "AMD R7 7840H",
			LineID:   4,
			UnitCore: 800, UnitMem: 600, UnitDisk: 100, UnitBW: 1000,
			Packages: []pkgSeed{
				{Name: "2核4G", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R7 7840H", Monthly: 4000},
				{Name: "4核8G", Cores: 4, MemoryGB: 8, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R7 7840H", Monthly: 7000},
				{Name: "4核12G", Cores: 4, MemoryGB: 12, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R7 7840H", Monthly: 9000},
				{Name: "4核16G", Cores: 4, MemoryGB: 16, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R7 7840H", Monthly: 11000},
				{Name: "6核18G", Cores: 6, MemoryGB: 18, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R7 7840H", Monthly: 13000},
				{Name: "8核24G", Cores: 8, MemoryGB: 24, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R7 7840H", Monthly: 17000},
				{Name: "8核32G", Cores: 8, MemoryGB: 32, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R7 7840H", Monthly: 22000},
			},
		},
		{
			Name:     "AMD R9 9950X",
			LineID:   5,
			UnitCore: 1200, UnitMem: 800, UnitDisk: 100, UnitBW: 1000,
			Packages: []pkgSeed{
				{Name: "2核4G", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R9 9950X", Monthly: 6000},
				{Name: "4核8G", Cores: 4, MemoryGB: 8, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R9 9950X", Monthly: 9000},
				{Name: "4核12G", Cores: 4, MemoryGB: 12, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R9 9950X", Monthly: 12000},
				{Name: "4核16G", Cores: 4, MemoryGB: 16, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R9 9950X", Monthly: 14000},
				{Name: "6核18G", Cores: 6, MemoryGB: 18, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R9 9950X", Monthly: 17000},
				{Name: "8核24G", Cores: 8, MemoryGB: 24, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R9 9950X", Monthly: 23000},
				{Name: "12核28G", Cores: 12, MemoryGB: 28, DiskGB: 40, BandwidthMB: 10, CPUModel: "AMD R9 9950X", Monthly: 34000},
			},
		},
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO regions(code,name,active) VALUES (?,?,?)`, "area-1", "晋中", 1)
	if err != nil {
		return err
	}
	shanxiID, _ := res.LastInsertId()

	_, err = tx.Exec(`INSERT INTO regions(code,name,active) VALUES (?,?,?)`, "area-2", "宁波", 0)
	if err != nil {
		return err
	}

	for idx, plan := range plans {
		lineID := plan.LineID
		if lineID == 0 {
			lineID = int64(defaultLineID)
		}
		res, err := tx.Exec(`INSERT INTO plan_groups(region_id,name,line_id,unit_core,unit_mem,unit_disk,unit_bw,add_core_min,add_core_max,add_core_step,add_mem_min,add_mem_max,add_mem_step,add_disk_min,add_disk_max,add_disk_step,add_bw_min,add_bw_max,add_bw_step,active,visible,capacity_remaining,sort_order) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			shanxiID, plan.Name, lineID, plan.UnitCore, plan.UnitMem, plan.UnitDisk, plan.UnitBW,
			0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1,
			1, 1, -1, idx)
		if err != nil {
			return err
		}
		planID, _ := res.LastInsertId()
		for pidx, pkg := range plan.Packages {
			_, err = tx.Exec(`INSERT INTO packages(plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				planID, 0, pkg.Name, pkg.Cores, pkg.MemoryGB, pkg.DiskGB, pkg.BandwidthMB, pkg.CPUModel, pkg.Monthly, portNumDefault, pidx, 1, 1, -1)
			if err != nil {
				return err
			}
		}
	}

	systemImages := []struct {
		Name string
		Type string
	}{
		{Name: "Ubuntu 22.04", Type: "linux"},
		{Name: "Debian 12", Type: "linux"},
		{Name: "Windows Server 2022", Type: "windows"},
	}
	systemImageIDs := make([]int64, 0, len(systemImages))
	for _, img := range systemImages {
		res, err := tx.Exec(`INSERT INTO system_images(image_id,name,type,enabled) VALUES (?,?,?,?)`, 0, img.Name, img.Type, 1)
		if err != nil {
			return err
		}
		if id, _ := res.LastInsertId(); id > 0 {
			systemImageIDs = append(systemImageIDs, id)
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
	for lineID := range lineIDSet {
		for _, imageID := range systemImageIDs {
			if _, err := tx.Exec(`INSERT INTO line_system_images(line_id, system_image_id) VALUES (?,?)`, lineID, imageID); err != nil {
				return err
			}
		}
	}

	provisionBody := `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; color: #222; line-height: 1.6;">
  <h2>VPS Provisioned</h2>
  <p>Hi {{.user.username}},</p>
  <p>Your VPS for order <strong>{{.order.no}}</strong> is now active.</p>
  <p>You can log in to the control panel to manage your instance.</p>
  <hr>
  <p>If you have any questions, reply to this email.</p>
</body>
</html>`
	expireBody := `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; color: #222; line-height: 1.6;">
  <h2>VPS Expiration Reminder</h2>
  <p>Hi {{.user.username}},</p>
  <p>Your VPS <strong>{{.vps.name}}</strong> will expire on <strong>{{.vps.expire_at}}</strong>.</p>
  <p>Please renew in time to avoid service interruption.</p>
</body>
</html>`
	approvedBody := `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; color: #222; line-height: 1.6;">
  <h2>Order Approved</h2>
  <p>Hi {{.user.username}},</p>
  <p>Your order <strong>{{.order.no}}</strong> has been approved.</p>
  <p>{{.message}}</p>
</body>
</html>`
	rejectedBody := `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; color: #222; line-height: 1.6;">
  <h2>Order Rejected</h2>
  <p>Hi {{.user.username}},</p>
  <p>Your order <strong>{{.order.no}}</strong> has been rejected.</p>
  <p>Reason: {{.message}}</p>
</body>
</html>`
	_, _ = tx.Exec(`INSERT INTO email_templates(name,subject,body,enabled) VALUES (?,?,?,?)`, "provision_success", "VPS Provisioned: Order {{.order.no}}", provisionBody, 1)
	_, _ = tx.Exec(`INSERT INTO email_templates(name,subject,body,enabled) VALUES (?,?,?,?)`, "expire_reminder", "VPS Expiration Reminder: {{.vps.name}}", expireBody, 1)
	_, _ = tx.Exec(`INSERT INTO email_templates(name,subject,body,enabled) VALUES (?,?,?,?)`, "order_approved", "Order Approved: {{.order.no}}", approvedBody, 1)
	_, _ = tx.Exec(`INSERT INTO email_templates(name,subject,body,enabled) VALUES (?,?,?,?)`, "order_rejected", "Order Rejected: {{.order.no}}", rejectedBody, 1)

	passwordResetBody := `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; color: #222; line-height: 1.6;">
  <h2>Password Reset</h2>
  <p>Hi {{.user.username}},</p>
  <p>You have requested to reset your password.</p>
  <p>Your reset token is: <strong>{{.token}}</strong></p>
  <p>Please use this token to reset your password. The token will expire in 24 hours.</p>
  <p>If you did not request this, please ignore this email.</p>
</body>
</html>`
	_, _ = tx.Exec(`INSERT INTO email_templates(name,subject,body,enabled) VALUES (?,?,?,?)`, "password_reset", "Password Reset", passwordResetBody, 1)

	superAdminPerms := `["*"]`
	opsAdminPerms := `["user.list","user.view","order.list","order.view","order.approve","order.reject","vps.*","audit_log.view","scheduled_tasks.*"]`
	csAdminPerms := `["user.list","user.view","order.list","order.view","vps.list","vps.view"]`
	financeAdminPerms := `["order.list","order.view","order.approve","order.reject","audit_log.view"]`

	_, _ = tx.Exec(`INSERT INTO permission_groups(name,description,permissions_json) VALUES (?,?,?)`, "超级管理员", "拥有所有权限", superAdminPerms)
	_, _ = tx.Exec(`INSERT INTO permission_groups(name,description,permissions_json) VALUES (?,?,?)`, "运维管理员", "负责VPS运维和订单审核", opsAdminPerms)
	_, _ = tx.Exec(`INSERT INTO permission_groups(name,description,permissions_json) VALUES (?,?,?)`, "客服管理员", "负责用户和订单查询", csAdminPerms)
	_, _ = tx.Exec(`INSERT INTO permission_groups(name,description,permissions_json) VALUES (?,?,?)`, "财务管理员", "负责订单审核和财务管理", financeAdminPerms)

	_, _ = tx.Exec(`INSERT INTO billing_cycles(name,months,multiplier,min_qty,max_qty,active,sort_order) VALUES (?,?,?,?,?,?,?)`, "monthly", 1, 1.0, 1, 24, 1, 1)
	_, _ = tx.Exec(`INSERT INTO billing_cycles(name,months,multiplier,min_qty,max_qty,active,sort_order) VALUES (?,?,?,?,?,?,?)`, "quarterly", 3, 2.8, 1, 12, 1, 2)
	_, _ = tx.Exec(`INSERT INTO billing_cycles(name,months,multiplier,min_qty,max_qty,active,sort_order) VALUES (?,?,?,?,?,?,?)`, "yearly", 12, 10.0, 1, 5, 1, 3)

	return tx.Commit()
}

func EnsureSettings(db *sql.DB, dialect string) error {
	settings := map[string]string{
		"default_line_id":                "0",
		"default_port_num":               "30",
		"payment_providers_enabled":      `{"approval":true,"balance":true,"custom":true,"yipay":false}`,
		"payment_providers_config":       `{"custom":{"pay_url":"","instructions":""},"yipay":{"base_url":"https://pays.org.cn/submit.php","pid":"","key":"","pay_type":"","notify_url":"","return_url":"","sign_type":"MD5"}}`,
		"payment_plugins":                "[]",
		"payment_plugin_dir":             "plugins/payment",
		"payment_plugin_upload_password": "qweasd123456",
		"robot_webhook_url":              "",
		"robot_webhook_secret":           "",
		"robot_webhook_enabled":          "false",
		"robot_webhooks":                 "[]",
		"realname_enabled":               "false",
		"realname_provider":              "idcard_cn",
		"realname_block_actions":         `["purchase_vps"]`,
		"smtp_host":                      "",
		"smtp_port":                      "",
		"smtp_user":                      "",
		"smtp_pass":                      "",
		"smtp_from":                      "",
		"smtp_enabled":                   "false",
		"email_enabled":                  "true",
		"email_expire_enabled":           "true",
		"expire_reminder_days":           "7",
		"emergency_renew_enabled":        "true",
		"emergency_renew_window_days":    "7",
		"emergency_renew_days":           "1",
		"emergency_renew_interval_hours": "720",
		"auto_delete_enabled":            "false",
		"auto_delete_days":               "7",
		"refund_full_days":               "1",
		"refund_prorate_days":            "7",
		"refund_no_refund_days":          "30",
		"refund_full_hours":              "0",
		"refund_prorate_hours":           "0",
		"refund_no_refund_hours":         "0",
		"refund_curve_json":              "[]",
		"refund_requires_approval":       "true",
		"refund_on_admin_delete":         "true",
		"resize_price_mode":              "remaining",
		"resize_refund_ratio":            "1",
		"resize_rounding":                "round",
		"resize_min_charge":              "0",
		"resize_min_refund":              "0",
		"resize_charge_curve_json":       "[]",
		"resize_refund_to_wallet":        "true",
		"debug_enabled":                  "false",
		"automation_base_url":            "",
		"automation_api_key":             "",
		"automation_enabled":             "true",
		"automation_timeout_sec":         "12",
		"automation_retry":               "0",
		"automation_dry_run":             "false",
		"automation_log_retention_days":  "0",
		"task.vps_refresh":               `{"enabled":true,"strategy":"interval","interval_sec":300}`,
		"task.order_provision_watchdog":  `{"enabled":true,"strategy":"interval","interval_sec":5}`,
		"provision_watchdog_max_jobs":    "8",
		"provision_watchdog_max_minutes": "20",
		"task.expire_reminder":           `{"enabled":true,"strategy":"daily","daily_at":"09:00"}`,
		"task.vps_expire_cleanup":        `{"enabled":true,"strategy":"daily","daily_at":"03:00"}`,
		"site_name":                      "Cloud Console",
		"site_url":                       "",
		"logo_url":                       "",
		"favicon_url":                    "",
		"site_description":               "",
		"site_keywords":                  "",
		"company_name":                   "",
		"contact_phone":                  "",
		"contact_email":                  "",
		"contact_qq":                     "",
		"wechat_qrcode":                  "",
		"icp_number":                     "",
		"psbe_number":                    "",
		"maintenance_mode":               "false",
		"maintenance_message":            "We are under maintenance, please check back later.",
		"analytics_code":                 "",
		"site_logo":                      "",
		"site_icp":                       "",
		"site_maintenance_mode":          "false",
		"site_maintenance_message":       "We are under maintenance, please check back later.",
		"site_nav_items":                 `[{"label":"产品","url":"/products","target":"_self","lang":"zh-CN"},{"label":"活动","url":"/activities","target":"_self","lang":"zh-CN"},{"label":"文档","url":"/docs","target":"_self","lang":"zh-CN"},{"label":"Products","url":"/products","target":"_self","lang":"en-US"},{"label":"Activities","url":"/activities","target":"_self","lang":"en-US"},{"label":"Docs","url":"/docs","target":"_self","lang":"en-US"}]`,
	}
	for key, val := range settings {
		if strings.EqualFold(strings.TrimSpace(dialect), "mysql") {
			_, _ = db.Exec("INSERT INTO settings(`key`,value_json,updated_at) VALUES (?,?,CURRENT_TIMESTAMP) ON DUPLICATE KEY UPDATE `key`=`key`", key, val)
		} else {
			_, _ = db.Exec(`INSERT INTO settings(key,value_json,updated_at) VALUES (?,?,CURRENT_TIMESTAMP) ON CONFLICT(key) DO NOTHING`, key, val)
		}
	}
	return nil
}

func EnsurePermissionDefaults(db *sql.DB, _ string) error {
	_, err := db.Exec(`UPDATE permissions SET friendly_name = name WHERE friendly_name IS NULL OR friendly_name = ''`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE permissions SET parent_code = '' WHERE parent_code IS NULL`)
	return err
}

func EnsurePermissionGroups(db *sql.DB, dialect string) error {
	superAdminPerms := `["*"]`
	opsAdminPerms := `["dashboard.overview","dashboard.revenue","dashboard.vps_status","user.list","user.view","order.list","order.view","order.approve","order.reject","vps.*","audit_log.view","scheduled_tasks.*"]`
	csAdminPerms := `["dashboard.overview","dashboard.revenue","user.list","user.view","order.list","order.view","vps.list","vps.view"]`
	financeAdminPerms := `["dashboard.overview","dashboard.revenue","order.list","order.view","order.approve","order.reject","audit_log.view"]`

	insertSQL := `INSERT INTO permission_groups(name,description,permissions_json) VALUES (?,?,?) ON CONFLICT(name) DO NOTHING`
	if strings.EqualFold(strings.TrimSpace(dialect), "mysql") {
		insertSQL = `INSERT INTO permission_groups(name,description,permissions_json) VALUES (?,?,?) ON DUPLICATE KEY UPDATE name=name`
	}

	_, err := db.Exec(insertSQL, "超级管理员", "拥有所有权限", superAdminPerms)
	if err != nil {
		return err
	}
	_, err = db.Exec(insertSQL, "运维管理员", "负责VPS运维和订单审核", opsAdminPerms)
	if err != nil {
		return err
	}
	_, err = db.Exec(insertSQL, "客服管理员", "负责用户和订单查询", csAdminPerms)
	if err != nil {
		return err
	}
	_, err = db.Exec(insertSQL, "财务管理员", "负责订单审核和财务管理", financeAdminPerms)
	if err != nil {
		return err
	}

	for _, groupName := range []string{"运维管理员", "客服管理员", "财务管理员"} {
		if err := ensurePermissionInGroup(db, groupName, "dashboard.revenue"); err != nil {
			return err
		}
	}

	_, err = db.Exec(`UPDATE users SET permission_group_id = 1 WHERE role = 'admin' AND (permission_group_id IS NULL OR permission_group_id = 0)`)
	return err
}

func EnsureCMSDefaults(db *sql.DB, dialect string) error {
	categories := []struct {
		Key   string
		Name  string
		Lang  string
		Order int
	}{
		{"tutorials", "教程", "zh-CN", 1},
		{"docs", "文档", "zh-CN", 2},
		{"announcements", "公告", "zh-CN", 3},
		{"activities", "活动", "zh-CN", 4},
		{"tutorials", "Tutorials", "en-US", 1},
		{"docs", "Docs", "en-US", 2},
		{"announcements", "Announcements", "en-US", 3},
		{"activities", "Activities", "en-US", 4},
	}

	var insertCategorySQL string
	if strings.EqualFold(strings.TrimSpace(dialect), "mysql") {
		insertCategorySQL = "INSERT INTO cms_categories(`key`,name,lang,sort_order,visible) VALUES (?,?,?,?,1) ON DUPLICATE KEY UPDATE `key`=`key`"
	} else {
		insertCategorySQL = `INSERT INTO cms_categories(key,name,lang,sort_order,visible) VALUES (?,?,?,?,1) ON CONFLICT(key,lang) DO NOTHING`
	}

	for _, item := range categories {
		_, _ = db.Exec(insertCategorySQL, item.Key, item.Name, item.Lang, item.Order)
	}

	var blocksCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM cms_blocks`).Scan(&blocksCount); err != nil {
		return err
	}
	if blocksCount > 0 {
		return nil
	}

	blocks := []struct {
		Page     string
		Type     string
		Title    string
		Subtitle string
		Content  string
		Lang     string
		Order    int
	}{
		{
			"home",
			"hero_3d",
			"构建未来 从云端开始",
			"",
			`{"badge":"企业级云基础设施","title_lines":["构建未来","从云端开始"],"description_lines":["新一代云基础设施，专为现代应用构建","弹性计算 · 全球部署 · 智能运维"],"buttons":[{"text":"立即开始","url":"/console","type":"primary","size":"large"},{"text":"浏览产品","url":"/products","type":"default","size":"large"}],"trust_badges":["ISO 27001","SOC 2","等保三级"],"card1_icon":"⚡","card1_label":"实例状态","card1_value":"运行中","card2_icon":"📊","card2_label":"CPU 使用率","card2_value":"45%","card2_suffix":"%","card3_icon":"💾","card3_label":"存储空间","card3_value":"2.4 TB","ring_value":"52847","ring_label":"活跃实例"}`,
			"zh-CN",
			1,
		},
		{
			"home",
			"stats_bar",
			"实时统计",
			"",
			`{"stats":[{"icon":"cloud","value":99.99,"unit":"%","label":"服务可用性","gradient":"background: linear-gradient(135deg, #1677ff, #4096ff)"},{"icon":"zap","value":30,"unit":"秒","label":"平均部署时间","gradient":"background: linear-gradient(135deg, #52c41a, #73d13d)"},{"icon":"globe","value":30,"unit":"+","label":"全球数据中心","gradient":"background: linear-gradient(135deg, #722ed1, #9254de)"},{"icon":"headphones","value":15,"unit":"分钟","label":"工单响应时间","gradient":"background: linear-gradient(135deg, #faad14, #ffc53d)"}]}`,
			"zh-CN",
			2,
		},
		{
			"home",
			"product_cards",
			"一站式云计算解决方案",
			"提供完整的云产品矩阵，满足不同场景的业务需求",
			`{"products":[{"name":"云服务器","emoji":"☁️","desc":"高性能计算实例，弹性伸缩","link":"/products/ecs","gradient":"background: linear-gradient(135deg, #1677ff, #4096ff)","features":["秒级交付","弹性扩容","多种配置","自动备份"]},{"name":"对象存储","emoji":"🗄️","desc":"安全稳定的云端存储服务","link":"/products/oss","gradient":"background: linear-gradient(135deg, #52c41a, #73d13d)","features":["99.99%可靠","无限容量","CDN加速","数据加密"]},{"name":"CDN加速","emoji":"🚀","desc":"全球节点，极速访问体验","link":"/products/cdn","gradient":"background: linear-gradient(135deg, #722ed1, #9254de)","features":["全球覆盖","智能调度","HTTPS支持","实时监控"]},{"name":"云数据库","emoji":"🗃️","desc":"高性能数据库托管服务","link":"/products/rds","gradient":"background: linear-gradient(135deg, #faad14, #ffc53d)","features":["自动备份","主从复制","性能监控","弹性扩展"]},{"name":"容器服务","emoji":"🐳","desc":"Kubernetes 容器编排","link":"/products/k8s","gradient":"background: linear-gradient(135deg, #eb2f96, #f759ab)","features":["一键部署","自动扩缩","服务网格","DevOps"]},{"name":"负载均衡","emoji":"⚖️","desc":"流量分发，保障高可用","link":"/products/slb","gradient":"background: linear-gradient(135deg, #13c2c2, #36cfc9)","features":["多种算法","健康检查","会话保持","DDoS防护"]}]}`,
			"zh-CN",
			3,
		},
		{
			"home",
			"feature_metrics",
			"为什么选择小黑云",
			"企业级技术实力，助力业务快速增长",
			`{"features":[{"icon":"shield","title":"安全可靠","desc":"多层安全防护体系，通过多项国际认证，保障数据安全","gradient":"background: linear-gradient(135deg, #1677ff, #4096ff)","metrics":[{"value":"99.99%","label":"可用性"},{"value":"7x24","label":"监控"}]},{"icon":"zap","title":"极速性能","desc":"最新硬件配置，优化网络架构，提供卓越性能体验","gradient":"background: linear-gradient(135deg, #52c41a, #73d13d)","metrics":[{"value":"30s","label":"交付"},{"value":"10Gbps","label":"带宽"}]},{"icon":"globe","title":"全球覆盖","desc":"30+数据中心遍布全球，BGP多线接入，就近访问","gradient":"background: linear-gradient(135deg, #722ed1, #9254de)","metrics":[{"value":"30+","label":"节点"},{"value":"100+","label":"国家"}]},{"icon":"headphones","title":"专业支持","desc":"专业技术团队7x24小时在线，快速响应解决问题","gradient":"background: linear-gradient(135deg, #faad14, #ffc53d)","metrics":[{"value":"15min","label":"响应"},{"value":"99%","label":"满意度"}]}]}`,
			"zh-CN",
			4,
		},
		{
			"home",
			"solutions_tabs",
			"为各行各业提供云端动力",
			"",
			`{"solutions":[{"icon":"🛒","name":"电商","title":"电商行业解决方案","desc":"应对大流量挑战，保障购物高峰期稳定运行","items":["弹性应对促销高峰","高并发架构设计","CDN加速访问","实时数据分析"],"cards":[{"icon":"📈","title":"流量承载","value":"10x+"},{"icon":"⚡","title":"页面加载","value":"<1s"}]},{"icon":"🎮","name":"游戏","title":"游戏行业解决方案","desc":"低延迟、高并发，提供流畅游戏体验","items":["全球节点部署","智能路由调度","实时语音同步","反外挂防护"],"cards":[{"icon":"🌐","title":"全球部署","value":"30+"},{"icon":"⚡","title":"网络延迟","value":"<20ms"}]},{"icon":"💰","name":"金融","title":"金融行业解决方案","desc":"安全合规，满足金融行业严苛要求","items":["等保三级认证","多重加密防护","异地容灾备份","审计日志"],"cards":[{"icon":"🔒","title":"安全等级","value":"等保三级"},{"icon":"✅","title":"合规认证","value":"10+"}]}]}`,
			"zh-CN",
			5,
		},
		{
			"home",
			"customers",
			"值得信赖的云服务伙伴",
			"",
			`{"logos":[{"text":"LOGO 1"},{"text":"LOGO 2"},{"text":"LOGO 3"},{"text":"LOGO 4"},{"text":"LOGO 5"},{"text":"LOGO 6"},{"text":"LOGO 7"},{"text":"LOGO 8"},{"text":"LOGO 9"},{"text":"LOGO 10"},{"text":"LOGO 11"},{"text":"LOGO 12"}],"stats":[{"value":100000,"label":"企业用户"},{"value":500000,"label":"云服务器"},{"value":99,"label":"客户满意度"}]}`,
			"zh-CN",
			6,
		},
		{
			"home",
			"cta_gift",
			"新用户注册即送",
			"",
			`{"badge":"限时优惠","title":"新用户注册即送","currency":"¥","amount":"500","unit":"体验金","desc":"注册即可领取，用于体验全系列产品","gradient":"background: linear-gradient(135deg, #1677ff 0%, #722ed1 100%)","buttons":[{"text":"立即注册","url":"/register","type":"primary","size":"large"},{"text":"了解规则","url":"/docs","type":"secondary","size":"large"}]}`,
			"zh-CN",
			7,
		},
		{
			"home",
			"footer_links",
			"Footer",
			"",
			`{"brand_name":"小黑云","brand_desc":"专业的云计算服务提供商，为企业提供稳定、安全、高效的云基础设施","social_links":[{"href":"#"},{"href":"#"},{"href":"#"},{"href":"#"}],"groups":[{"title":"产品","links":[{"text":"云服务器","href":"/products/ecs"},{"text":"对象存储","href":"/products/oss"},{"text":"CDN加速","href":"/products/cdn"},{"text":"云数据库","href":"/products/rds"}]},{"title":"解决方案","links":[{"text":"电商解决方案","href":"/solutions/ecommerce"},{"text":"游戏解决方案","href":"/solutions/game"},{"text":"金融解决方案","href":"/solutions/finance"},{"text":"视频解决方案","href":"/solutions/video"}]},{"title":"支持","links":[{"text":"开发文档","href":"/docs"},{"text":"API参考","href":"/api"},{"text":"SDK下载","href":"/sdk"},{"text":"工单系统","href":"/tickets"}]},{"title":"关于","links":[{"text":"关于我们","href":"/about"},{"text":"新闻动态","href":"/news"},{"text":"加入我们","href":"/careers"},{"text":"联系我们","href":"/contact"}]}],"legal_links":[{"text":"隐私政策","href":"#"},{"text":"服务条款","href":"#"},{"text":"备案信息","href":"#"}],"badges":["可信云","等保三级","ISO 27001"],"copyright":"2024 小黑云. All rights reserved."}`,
			"zh-CN",
			8,
		},
		{
			"products",
			"products_hero",
			"产品与解决方案",
			"",
			`{"badge":"云基础设施","title":"产品与解决方案","description":"面向企业与开发者的一站式云基础设施能力，助力业务快速上云","buttons":[{"text":"立即体验","url":"/console/buy"},{"text":"查看文档","url":"/docs"}],"features":[{"icon":"cloud","text":"弹性计算","color":"linear-gradient(135deg, #667eea 0%, #764ba2 100%)"},{"icon":"server","text":"高性能","color":"linear-gradient(135deg, #f093fb 0%, #f5576c 100%)"},{"icon":"shield","text":"安全可靠","color":"linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)"}]}`,
			"zh-CN",
			1,
		},
		{
			"products",
			"products_core",
			"核心产品",
			"稳定、安全、高效的云计算服务",
			`{"products":[{"icon":"server","name":"云服务器","desc":"提供高性能、高可靠的弹性计算服务，支持多种配置选择","color":"linear-gradient(135deg, #667eea 0%, #764ba2 100%)","link":"/console/buy","features":["弹性伸缩","按需付费","高性能计算"]},{"icon":"database","name":"云数据库","desc":"稳定可靠的在线数据库服务，支持多种数据库引擎","color":"linear-gradient(135deg, #f093fb 0%, #f5576c 100%)","link":"/console/buy","features":["自动备份","高可用","监控告警"]},{"icon":"zap","name":"CDN 加速","desc":"全球加速的内容分发网络，提升用户访问速度","color":"linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)","link":"/console/buy","features":["全球节点","智能调度","安全防护"]},{"icon":"cloud","name":"对象存储","desc":"安全、稳定、高效的云端存储服务","color":"linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)","link":"/console/buy","features":["无限容量","高可靠性","数据安全"]}]}`,
			"zh-CN",
			2,
		},
		{
			"products",
			"products_why",
			"为什么选择小黑云",
			"专业的技术实力与完善的服务体系",
			`{"items":[{"icon":"🚀","title":"快速部署","desc":"秒级创建云服务器，快速部署您的应用"},{"icon":"🔒","title":"安全可靠","desc":"多重安全防护，保障您的数据安全"},{"icon":"⚡","title":"高性能","desc":"采用最新硬件，提供卓越的计算性能"},{"icon":"🌍","title":"全球覆盖","desc":"多个数据中心，覆盖全球主要地区"},{"icon":"💰","title":"按需付费","desc":"灵活的计费方式，降低运营成本"},{"icon":"🛟","title":"7x24支持","desc":"专业技术团队，随时为您提供帮助"}]}`,
			"zh-CN",
			3,
		},
		{
			"products",
			"products_cta",
			"准备好开始了吗？",
			"",
			`{"title":"准备好开始了吗？","desc":"立即注册，免费体验我们的云服务","buttons":[{"text":"免费注册","url":"/register"},{"text":"购买产品","url":"/console/buy"}]}`,
			"zh-CN",
			4,
		},
	}
	for _, block := range blocks {
		_, _ = db.Exec(`INSERT INTO cms_blocks(page,type,title,subtitle,content_json,custom_html,lang,visible,sort_order) VALUES (?,?,?,?,?,'',?,1,?)`, block.Page, block.Type, block.Title, block.Subtitle, block.Content, block.Lang, block.Order)
	}
	return nil
}

func ensurePermissionInGroup(db *sql.DB, groupName string, permission string) error {
	var raw string
	if err := db.QueryRow(`SELECT permissions_json FROM permission_groups WHERE name = ?`, groupName).Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	var perms []string
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &perms); err != nil {
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
	_, err = db.Exec(`UPDATE permission_groups SET permissions_json = ? WHERE name = ?`, string(b), groupName)
	return err
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

func getSettingInt(db *sql.DB, key string, fallback int) int {
	var raw string
	if err := db.QueryRow(`SELECT value_json FROM settings WHERE key = ?`, key).Scan(&raw); err != nil {
		return fallback
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return val
}
