package repo

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

func migrateSQLite(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			qq TEXT,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS captchas (
			id TEXT PRIMARY KEY,
			code_hash TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS regions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS plan_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			region_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			line_id INTEGER NOT NULL DEFAULT 0,
			unit_core INTEGER NOT NULL,
			unit_mem INTEGER NOT NULL,
			unit_disk INTEGER NOT NULL,
			unit_bw INTEGER NOT NULL,
			add_core_min INTEGER NOT NULL DEFAULT 0,
			add_core_max INTEGER NOT NULL DEFAULT 0,
			add_core_step INTEGER NOT NULL DEFAULT 1,
			add_mem_min INTEGER NOT NULL DEFAULT 0,
			add_mem_max INTEGER NOT NULL DEFAULT 0,
			add_mem_step INTEGER NOT NULL DEFAULT 1,
			add_disk_min INTEGER NOT NULL DEFAULT 0,
			add_disk_max INTEGER NOT NULL DEFAULT 0,
			add_disk_step INTEGER NOT NULL DEFAULT 1,
			add_bw_min INTEGER NOT NULL DEFAULT 0,
			add_bw_max INTEGER NOT NULL DEFAULT 0,
			add_bw_step INTEGER NOT NULL DEFAULT 1,
			active INTEGER NOT NULL DEFAULT 1,
			visible INTEGER NOT NULL DEFAULT 1,
			capacity_remaining INTEGER NOT NULL DEFAULT -1,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(region_id) REFERENCES regions(id)
		);`,
		`CREATE TABLE IF NOT EXISTS packages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plan_group_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL DEFAULT 0,
			name TEXT NOT NULL,
			cores INTEGER NOT NULL,
			memory_gb INTEGER NOT NULL,
			disk_gb INTEGER NOT NULL,
			bandwidth_mbps INTEGER NOT NULL,
			cpu_model TEXT NOT NULL,
			monthly_price INTEGER NOT NULL,
			port_num INTEGER NOT NULL DEFAULT 30,
			sort_order INTEGER NOT NULL DEFAULT 0,
			active INTEGER NOT NULL DEFAULT 1,
			visible INTEGER NOT NULL DEFAULT 1,
			capacity_remaining INTEGER NOT NULL DEFAULT -1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(plan_group_id) REFERENCES plan_groups(id)
		);`,
		`CREATE TABLE IF NOT EXISTS system_images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			image_id INTEGER NOT NULL DEFAULT 0,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS line_system_images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			line_id INTEGER NOT NULL,
			system_image_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(system_image_id) REFERENCES system_images(id)
		);`,
		`CREATE TABLE IF NOT EXISTS cart_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			package_id INTEGER NOT NULL,
			system_id INTEGER NOT NULL,
			spec_json TEXT NOT NULL,
			qty INTEGER NOT NULL DEFAULT 1,
			amount INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			order_no TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'pending_payment',
			total_amount INTEGER NOT NULL,
			currency TEXT NOT NULL,
			idempotency_key TEXT,
			pending_reason TEXT,
			approved_by INTEGER,
			approved_at DATETIME,
			rejected_reason TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS order_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			package_id INTEGER,
			system_id INTEGER,
			spec_json TEXT NOT NULL,
			qty INTEGER NOT NULL DEFAULT 1,
			amount INTEGER NOT NULL,
			status TEXT NOT NULL,
			automation_instance_id TEXT,
			action TEXT NOT NULL DEFAULT 'create',
			duration_months INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(order_id) REFERENCES orders(id)
		);`,
		`CREATE TABLE IF NOT EXISTS vps_instances (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			order_item_id INTEGER NOT NULL,
			automation_instance_id TEXT NOT NULL,
			name TEXT NOT NULL,
			region TEXT,
			region_id INTEGER NOT NULL DEFAULT 0,
			line_id INTEGER NOT NULL DEFAULT 0,
			package_id INTEGER NOT NULL DEFAULT 0,
			package_name TEXT NOT NULL DEFAULT '',
			cpu INTEGER NOT NULL DEFAULT 0,
			memory_gb INTEGER NOT NULL DEFAULT 0,
			disk_gb INTEGER NOT NULL DEFAULT 0,
			bandwidth_mbps INTEGER NOT NULL DEFAULT 0,
			port_num INTEGER NOT NULL DEFAULT 0,
			monthly_price INTEGER NOT NULL DEFAULT 0,
			spec_json TEXT NOT NULL,
			system_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			automation_state INTEGER NOT NULL DEFAULT 0,
			admin_status TEXT NOT NULL DEFAULT 'normal',
			expire_at DATETIME,
			panel_url_cache TEXT,
			access_info_json TEXT,
			last_emergency_renew_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(order_item_id) REFERENCES order_items(id)
		);`,
		`CREATE TABLE IF NOT EXISTS order_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			seq INTEGER NOT NULL,
			type TEXT NOT NULL,
			data_json TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(order_id) REFERENCES orders(id)
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_order_events_seq ON order_events(order_id, seq);`,
		`CREATE TABLE IF NOT EXISTS admin_audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			admin_id INTEGER NOT NULL,
			action TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			detail_json TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			key_hash TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL,
			scopes_json TEXT NOT NULL,
			permission_group_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_used_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value_json TEXT NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS email_templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			subject TEXT NOT NULL,
			body TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS order_payments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			method TEXT NOT NULL,
			amount INTEGER NOT NULL,
			currency TEXT NOT NULL,
			trade_no TEXT NOT NULL,
			note TEXT,
			screenshot_url TEXT,
			status TEXT NOT NULL,
			idempotency_key TEXT,
			reviewed_by INTEGER,
			review_reason TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(order_id) REFERENCES orders(id),
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_order_payments_trade_no ON order_payments(trade_no);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_order_payments_idem ON order_payments(order_id, idempotency_key);`,
		`CREATE TABLE IF NOT EXISTS billing_cycles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			months INTEGER NOT NULL,
			multiplier REAL NOT NULL,
			min_qty INTEGER NOT NULL DEFAULT 1,
			max_qty INTEGER NOT NULL DEFAULT 36,
			active INTEGER NOT NULL DEFAULT 1,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS automation_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			order_item_id INTEGER NOT NULL,
			action TEXT NOT NULL,
			request_json TEXT NOT NULL,
			response_json TEXT NOT NULL,
			success INTEGER NOT NULL DEFAULT 0,
			message TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS provision_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			order_item_id INTEGER NOT NULL,
			host_id INTEGER NOT NULL,
			host_name TEXT NOT NULL,
			status TEXT NOT NULL,
			attempts INTEGER NOT NULL DEFAULT 0,
			next_run_at DATETIME NOT NULL,
			last_error TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_provision_jobs_item ON provision_jobs(order_item_id);`,
		`CREATE TABLE IF NOT EXISTS resize_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			vps_id INTEGER NOT NULL,
			order_id INTEGER NOT NULL,
			order_item_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			scheduled_at DATETIME,
			started_at DATETIME,
			finished_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_resize_tasks_vps ON resize_tasks(vps_id);`,
		`CREATE INDEX IF NOT EXISTS idx_resize_tasks_status ON resize_tasks(status);`,
		`CREATE TABLE IF NOT EXISTS integration_sync_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target TEXT NOT NULL,
			mode TEXT NOT NULL,
			status TEXT NOT NULL,
			message TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS permission_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			permissions_json TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			used INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS permissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			friendly_name TEXT,
			category TEXT NOT NULL,
			parent_code TEXT,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS cms_categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL,
			name TEXT NOT NULL,
			lang TEXT NOT NULL DEFAULT 'zh-CN',
			sort_order INTEGER NOT NULL DEFAULT 0,
			visible INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(key, lang)
		);`,
		`CREATE TABLE IF NOT EXISTS cms_posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			category_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			summary TEXT NOT NULL DEFAULT '',
			content_html TEXT NOT NULL,
			cover_url TEXT NOT NULL DEFAULT '',
			lang TEXT NOT NULL DEFAULT 'zh-CN',
			status TEXT NOT NULL DEFAULT 'draft',
			pinned INTEGER NOT NULL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0,
			published_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(category_id) REFERENCES cms_categories(id)
		);`,
		`CREATE TABLE IF NOT EXISTS cms_blocks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			page TEXT NOT NULL,
			type TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			subtitle TEXT NOT NULL DEFAULT '',
			content_json TEXT NOT NULL DEFAULT '',
			custom_html TEXT NOT NULL DEFAULT '',
			lang TEXT NOT NULL DEFAULT 'zh-CN',
			visible INTEGER NOT NULL DEFAULT 1,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS uploads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			url TEXT NOT NULL,
			mime TEXT NOT NULL,
			size INTEGER NOT NULL,
			uploader_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(uploader_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			subject TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'open',
			last_reply_at DATETIME,
			last_reply_by INTEGER,
			last_reply_role TEXT NOT NULL DEFAULT 'user',
			closed_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS ticket_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL,
			sender_id INTEGER NOT NULL,
			sender_role TEXT NOT NULL,
			sender_name TEXT,
			sender_qq TEXT,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(ticket_id) REFERENCES tickets(id)
		);`,
		`CREATE TABLE IF NOT EXISTS ticket_resources (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id INTEGER NOT NULL DEFAULT 0,
			resource_name TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(ticket_id) REFERENCES tickets(id)
		);`,
		`CREATE TABLE IF NOT EXISTS user_wallets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			balance INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS wallet_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			amount INTEGER NOT NULL,
			type TEXT NOT NULL,
			ref_type TEXT NOT NULL,
			ref_id INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS wallet_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			type TEXT NOT NULL,
			amount INTEGER NOT NULL,
			currency TEXT NOT NULL DEFAULT 'CNY',
			status TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			meta_json TEXT NOT NULL DEFAULT '',
			reviewed_by INTEGER,
			review_reason TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS scheduled_task_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_key TEXT NOT NULL,
			status TEXT NOT NULL,
			started_at DATETIME NOT NULL,
			finished_at DATETIME,
			duration_sec INTEGER NOT NULL DEFAULT 0,
			message TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			type TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			read_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS realname_verifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			real_name TEXT NOT NULL,
			id_number TEXT NOT NULL,
			status TEXT NOT NULL,
			provider TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			verified_at DATETIME,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_plan_groups_region ON plan_groups(region_id);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_password_reset_tokens_token ON password_reset_tokens(token);`,
		`CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user ON password_reset_tokens(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_plan_groups_line ON plan_groups(line_id);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_plan_groups_line_unique ON plan_groups(line_id) WHERE line_id > 0;`,
		`CREATE INDEX IF NOT EXISTS idx_packages_plan_group ON packages(plan_group_id);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_packages_product_unique ON packages(plan_group_id, product_id) WHERE product_id > 0;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_system_images_image_unique ON system_images(image_id) WHERE image_id > 0;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_line_system_images_unique ON line_system_images(line_id, system_image_id);`,
		`CREATE INDEX IF NOT EXISTS idx_line_system_images_line ON line_system_images(line_id);`,
		`CREATE INDEX IF NOT EXISTS idx_cart_items_user ON cart_items(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);`,
		`CREATE INDEX IF NOT EXISTS idx_vps_instances_user ON vps_instances(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_vps_instances_order_item ON vps_instances(order_item_id);`,
		`CREATE INDEX IF NOT EXISTS idx_order_payments_order ON order_payments(order_id);`,
		`CREATE INDEX IF NOT EXISTS idx_cms_posts_category ON cms_posts(category_id);`,
		`CREATE INDEX IF NOT EXISTS idx_cms_posts_lang ON cms_posts(lang);`,
		`CREATE INDEX IF NOT EXISTS idx_cms_blocks_page ON cms_blocks(page);`,
		`CREATE INDEX IF NOT EXISTS idx_uploads_uploader ON uploads(uploader_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_user ON tickets(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);`,
		`CREATE INDEX IF NOT EXISTS idx_ticket_messages_ticket ON ticket_messages(ticket_id);`,
		`CREATE INDEX IF NOT EXISTS idx_ticket_resources_ticket ON ticket_resources(ticket_id);`,
		`CREATE INDEX IF NOT EXISTS idx_wallet_transactions_user ON wallet_transactions(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_wallet_orders_user ON wallet_orders(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(user_id, read_at);`,
		`CREATE INDEX IF NOT EXISTS idx_realname_user ON realname_verifications(user_id);`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	if err := addColumnIfMissing(db, "plan_groups", "line_id", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	_ = addColumnIfMissing(db, "users", "phone", "TEXT")
	_ = addColumnIfMissing(db, "users", "bio", "TEXT")
	_ = addColumnIfMissing(db, "users", "intro", "TEXT")
	_ = addColumnIfMissing(db, "plan_groups", "add_core_min", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "plan_groups", "add_core_max", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "plan_groups", "add_core_step", "INTEGER NOT NULL DEFAULT 1")
	_ = addColumnIfMissing(db, "plan_groups", "add_mem_min", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "plan_groups", "add_mem_max", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "plan_groups", "add_mem_step", "INTEGER NOT NULL DEFAULT 1")
	_ = addColumnIfMissing(db, "plan_groups", "add_disk_min", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "plan_groups", "add_disk_max", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "plan_groups", "add_disk_step", "INTEGER NOT NULL DEFAULT 1")
	_ = addColumnIfMissing(db, "plan_groups", "add_bw_min", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "plan_groups", "add_bw_max", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "plan_groups", "add_bw_step", "INTEGER NOT NULL DEFAULT 1")
	_ = addColumnIfMissing(db, "plan_groups", "visible", "INTEGER NOT NULL DEFAULT 1")
	_ = addColumnIfMissing(db, "plan_groups", "capacity_remaining", "INTEGER NOT NULL DEFAULT -1")
	_ = addColumnIfMissing(db, "packages", "product_id", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "packages", "port_num", "INTEGER NOT NULL DEFAULT 30")
	_ = addColumnIfMissing(db, "packages", "visible", "INTEGER NOT NULL DEFAULT 1")
	_ = addColumnIfMissing(db, "packages", "capacity_remaining", "INTEGER NOT NULL DEFAULT -1")
	_ = addColumnIfMissing(db, "orders", "user_id", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "orders", "order_no", "TEXT NOT NULL DEFAULT ''")
	_ = addColumnIfMissing(db, "orders", "status", "TEXT NOT NULL DEFAULT 'pending_payment'")
	_ = addColumnIfMissing(db, "orders", "total_amount", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "orders", "currency", "TEXT NOT NULL DEFAULT 'CNY'")
	_ = addColumnIfMissing(db, "orders", "idempotency_key", "TEXT")
	_ = addColumnIfMissing(db, "orders", "pending_reason", "TEXT")
	_ = addColumnIfMissing(db, "orders", "approved_by", "INTEGER")
	_ = addColumnIfMissing(db, "orders", "approved_at", "DATETIME")
	_ = addColumnIfMissing(db, "orders", "rejected_reason", "TEXT")
	_ = addColumnIfMissing(db, "order_items", "duration_months", "INTEGER NOT NULL DEFAULT 1")
	_ = addColumnIfMissing(db, "api_keys", "name", "TEXT NOT NULL DEFAULT ''")
	_ = addColumnIfMissing(db, "api_keys", "status", "TEXT NOT NULL DEFAULT 'active'")
	_ = addColumnIfMissing(db, "api_keys", "scopes_json", "TEXT NOT NULL DEFAULT '[]'")
	_ = addColumnIfMissing(db, "api_keys", "permission_group_id", "INTEGER")
	_ = addColumnIfMissing(db, "vps_instances", "admin_status", "TEXT NOT NULL DEFAULT 'normal'")
	_ = addColumnIfMissing(db, "vps_instances", "last_emergency_renew_at", "DATETIME")
	_ = addColumnIfMissing(db, "vps_instances", "access_info_json", "TEXT")
	_ = addColumnIfMissing(db, "vps_instances", "region_id", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "vps_instances", "line_id", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "vps_instances", "package_id", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "vps_instances", "package_name", "TEXT NOT NULL DEFAULT ''")
	_ = addColumnIfMissing(db, "vps_instances", "cpu", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "vps_instances", "memory_gb", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "vps_instances", "disk_gb", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "vps_instances", "bandwidth_mbps", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "vps_instances", "port_num", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "vps_instances", "monthly_price", "INTEGER NOT NULL DEFAULT 0")
	_ = addColumnIfMissing(db, "users", "avatar", "TEXT")
	_ = addColumnIfMissing(db, "users", "permission_group_id", "INTEGER")
	_ = addColumnIfMissing(db, "ticket_messages", "sender_name", "TEXT")
	_ = addColumnIfMissing(db, "ticket_messages", "sender_qq", "TEXT")
	_ = addColumnIfMissing(db, "permissions", "friendly_name", "TEXT")
	_, _ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_idem ON orders(user_id, idempotency_key)`)
	if err := migrateMoneyToCents(db); err != nil {
		return err
	}
	_ = backfillVPSInstanceSnapshot(db)
	_ = migrateDefaultPermissionGroups(db)

	return nil
}

func migrateDefaultPermissionGroups(db *sql.DB) error {
	oldOps := `["user.list","user.view","order.list","order.view","order.approve","order.reject","vps.*","audit_log.view","scheduled_tasks.*"]`
	oldOpsLite := `["user.list","user.view","order.list","order.view","order.approve","order.reject","vps.*","audit_log.view"]`
	oldCS := `["user.list","user.view","order.list","order.view","vps.list","vps.view"]`
	oldFinance := `["order.list","order.view","order.approve","order.reject","audit_log.view"]`

	newOps := `["dashboard.overview","dashboard.vps_status","server.status","user.list","user.view","user.update","user.reset_password","order.list","order.view","order.approve","order.reject","order.retry","vps.*","regions.*","plan_group.*","line.*","packages.*","billing_cycle.*","system_image.*","scheduled_tasks.*","automation.*","smtp.*","robot.*","realname.*","settings.view","payment.list","plugin.upload","upload.*","tickets.*","api_key.*","email_template.*","audit_log.view"]`
	newCS := `["dashboard.overview","dashboard.vps_status","user.list","user.view","user.update","user.reset_password","order.list","order.view","vps.list","vps.view","vps.refresh","tickets.*","wallet.view","wallet.transactions","upload.*","realname.list"]`
	newFinance := `["dashboard.overview","dashboard.revenue","user.list","user.view","order.list","order.view","order.approve","order.reject","order.mark_paid","payment.*","wallet.*","wallet_order.*","audit_log.view","settings.view"]`

	_, err := db.Exec(`UPDATE permission_groups SET permissions_json = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ? AND (permissions_json = ? OR permissions_json = ?)`, newOps, "运维管理员", oldOps, oldOpsLite)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE permission_groups SET permissions_json = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ? AND permissions_json = ?`, newCS, "客服管理员", oldCS)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE permission_groups SET permissions_json = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ? AND permissions_json = ?`, newFinance, "财务管理员", oldFinance)
	return err
}

func addColumnIfMissing(db *sql.DB, table string, column string, ddl string) error {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, ddl))
	return err
}

type vpsSpecSnapshot struct {
	AddCores  int `json:"add_cores"`
	AddMemGB  int `json:"add_mem_gb"`
	AddDiskGB int `json:"add_disk_gb"`
	AddBWMbps int `json:"add_bw_mbps"`
}

func backfillVPSInstanceSnapshot(db *sql.DB) error {
	type vpsRow struct {
		id           int64
		orderItemID  int64
		specJSON     string
		packageID    int64
		packageName  string
		cpu          int
		memoryGB     int
		diskGB       int
		bandwidthMB  int
		portNum      int
		monthlyPrice int64
		region       string
		regionID     int64
		lineID       int64
	}
	rows, err := db.Query(`SELECT id, order_item_id, spec_json, package_id, package_name, cpu, memory_gb, disk_gb, bandwidth_mbps, port_num, monthly_price, region, region_id, line_id FROM vps_instances`)
	if err != nil {
		return err
	}
	var items []vpsRow
	for rows.Next() {
		var item vpsRow
		if err := rows.Scan(&item.id, &item.orderItemID, &item.specJSON, &item.packageID, &item.packageName, &item.cpu, &item.memoryGB, &item.diskGB, &item.bandwidthMB, &item.portNum, &item.monthlyPrice, &item.region, &item.regionID, &item.lineID); err != nil {
			return err
		}
		items = append(items, item)
	}
	rows.Close()

	for _, item := range items {
		if item.orderItemID == 0 {
			continue
		}
		if item.packageID != 0 && item.cpu != 0 && item.memoryGB != 0 && item.diskGB != 0 && item.bandwidthMB != 0 && item.regionID != 0 && item.lineID != 0 {
			continue
		}

		var oiPackageID int64
		if err := db.QueryRow(`SELECT package_id FROM order_items WHERE id = ?`, item.orderItemID).Scan(&oiPackageID); err != nil {
			continue
		}
		if oiPackageID == 0 {
			continue
		}
		var planGroupID int64
		var pkgName string
		var pkgCPU, pkgMem, pkgDisk, pkgBW, pkgPort int
		var pkgPrice int64
		if err := db.QueryRow(`SELECT plan_group_id, name, cores, memory_gb, disk_gb, bandwidth_mbps, port_num, monthly_price FROM packages WHERE id = ?`, oiPackageID).
			Scan(&planGroupID, &pkgName, &pkgCPU, &pkgMem, &pkgDisk, &pkgBW, &pkgPort, &pkgPrice); err != nil {
			continue
		}
		var pgLineID, pgRegionID int64
		var unitCore, unitMem, unitDisk, unitBW int64
		_ = db.QueryRow(`SELECT line_id, region_id, unit_core, unit_mem, unit_disk, unit_bw FROM plan_groups WHERE id = ?`, planGroupID).
			Scan(&pgLineID, &pgRegionID, &unitCore, &unitMem, &unitDisk, &unitBW)
		regionName := ""
		if pgRegionID > 0 {
			_ = db.QueryRow(`SELECT name FROM regions WHERE id = ?`, pgRegionID).Scan(&regionName)
		}

		addon := vpsSpecSnapshot{}
		_ = json.Unmarshal([]byte(item.specJSON), &addon)
		fCPU := pkgCPU + addon.AddCores
		fMem := pkgMem + addon.AddMemGB
		fDisk := pkgDisk + addon.AddDiskGB
		fBW := pkgBW + addon.AddBWMbps
		fPrice := pkgPrice + int64(addon.AddCores)*unitCore + int64(addon.AddMemGB)*unitMem + int64(addon.AddDiskGB)*unitDisk + int64(addon.AddBWMbps)*unitBW

		if item.packageID == 0 {
			item.packageID = oiPackageID
		}
		if item.packageName == "" {
			item.packageName = pkgName
		}
		if item.cpu == 0 {
			item.cpu = fCPU
		}
		if item.memoryGB == 0 {
			item.memoryGB = fMem
		}
		if item.diskGB == 0 {
			item.diskGB = fDisk
		}
		if item.bandwidthMB == 0 {
			item.bandwidthMB = fBW
		}
		if item.portNum == 0 {
			item.portNum = pkgPort
		}
		if item.monthlyPrice == 0 {
			item.monthlyPrice = fPrice
		}
		if item.regionID == 0 {
			item.regionID = pgRegionID
		}
		if item.lineID == 0 {
			item.lineID = pgLineID
		}
		if item.region == "" {
			item.region = regionName
		}

		_, _ = db.Exec(`UPDATE vps_instances SET package_id = ?, package_name = ?, cpu = ?, memory_gb = ?, disk_gb = ?, bandwidth_mbps = ?, port_num = ?, monthly_price = ?, region = ?, region_id = ?, line_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			item.packageID, item.packageName, item.cpu, item.memoryGB, item.diskGB, item.bandwidthMB, item.portNum, item.monthlyPrice, item.region, item.regionID, item.lineID, item.id)
	}
	return rows.Err()
}

func migrateMoneyToCents(db *sql.DB) error {
	const flagKey = "money_cents_migrated"
	var raw string
	if err := db.QueryRow(`SELECT value_json FROM settings WHERE key = ?`, flagKey).Scan(&raw); err == nil {
		if raw == "1" || raw == "true" || raw == "\"true\"" {
			return nil
		}
	} else if err != sql.ErrNoRows {
		return err
	}

	stmts := []string{
		`UPDATE plan_groups SET unit_core = CAST(ROUND(unit_core * 100) AS INTEGER), unit_mem = CAST(ROUND(unit_mem * 100) AS INTEGER), unit_disk = CAST(ROUND(unit_disk * 100) AS INTEGER), unit_bw = CAST(ROUND(unit_bw * 100) AS INTEGER)`,
		`UPDATE packages SET monthly_price = CAST(ROUND(monthly_price * 100) AS INTEGER)`,
		`UPDATE cart_items SET amount = CAST(ROUND(amount * 100) AS INTEGER)`,
		`UPDATE orders SET total_amount = CAST(ROUND(total_amount * 100) AS INTEGER)`,
		`UPDATE order_items SET amount = CAST(ROUND(amount * 100) AS INTEGER)`,
		`UPDATE vps_instances SET monthly_price = CAST(ROUND(monthly_price * 100) AS INTEGER)`,
		`UPDATE order_payments SET amount = CAST(ROUND(amount * 100) AS INTEGER)`,
		`UPDATE user_wallets SET balance = CAST(ROUND(balance * 100) AS INTEGER)`,
		`UPDATE wallet_transactions SET amount = CAST(ROUND(amount * 100) AS INTEGER)`,
		`UPDATE wallet_orders SET amount = CAST(ROUND(amount * 100) AS INTEGER)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	_, err := db.Exec(`INSERT INTO settings(key,value_json,updated_at) VALUES (?,?,CURRENT_TIMESTAMP) ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json, updated_at = CURRENT_TIMESTAMP`, flagKey, "1")
	return err
}
