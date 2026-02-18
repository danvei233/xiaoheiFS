# Repo Core Layout

This package keeps one repository implementation (`repo`) but splits files by bounded domain responsibility.

## Naming Rules

- Use `snake_case` file names.
- Prefer `gorm_repo_<domain>.go` for GORM repository behavior.
- Prefer `migrate_*.go` for schema/migration models and migration flow.
- Keep test files as `*_test.go`.

## Current Domain Split

- Catalog and pricing:
  - `gorm_repo_catalog_goods_plan_package.go`
  - `gorm_repo_catalog_system_image.go`
  - `gorm_repo_billing_cycle.go`
- CMS and content:
  - `gorm_repo_cms.go`
  - `gorm_repo_upload.go`
- Orders and commerce:
  - `gorm_repo_order.go`
  - `gorm_repo_order_event.go`
  - `gorm_repo_payment.go`
  - `gorm_repo_cart.go`
- Tickets and notifications:
  - `gorm_repo_ticket.go`
  - `gorm_repo_notification_push.go`
- Wallet:
  - `gorm_repo_wallet.go`
- Auth and permissions:
  - `gorm_repo_auth.go`
  - `gorm_repo_apikey.go`
  - `gorm_repo_password_reset.go`
  - `gorm_repo_admin_permissions.go`
- Settings and plugins:
  - `gorm_repo_settings_plugins.go`
  - `gorm_repo_realname.go`
  - `gorm_repo_automation_tasks.go`
  - `gorm_repo_probe.go`
  - `gorm_repo_vps.go`

## Deprecated Mixed Files (must not reappear)

- `gorm_repo_catalog.go`
- `gorm_repo_content_cms_upload.go`
- `gorm_repo_ticket_notification.go`
- `gorm_repo_scan_models.go`
- `migrate_models_ops.go`

