# Frontend Integration Guide

This guide describes the public and admin API usage from a frontend perspective.

## Base URL
- Default: http://localhost:8080

## Recent changes
- System images are global (no per-line duplicates); line availability is managed via line image mapping.
- Admin line image mapping endpoint: POST `/admin/api/v1/lines/{id}/system-images`.
- Admin can create VPS records with optional real provisioning via `provision` flag on `POST /admin/api/v1/vps`.
- Permissions are auto-registered from admin routes on startup; use `/admin/api/v1/permissions/sync` to resync.

## Development notes
- When adding new features, add default data in `backend/internal/adapter/seed/seed.go`.
- Always include a backfill step for existing databases (e.g., `EnsurePermissionDefaults`).

## Site & CMS (Homepage + Products page)
- Public site settings: GET `/api/v1/site/settings` (keys: `site_name`, `site_logo`, `site_icp`, `site_maintenance_mode`, `site_maintenance_message`, `site_nav_items`).
- Public CMS blocks: GET `/api/v1/cms/blocks?page=home&lang=zh-CN` (render by `sort_order`).
- Public CMS posts: GET `/api/v1/cms/posts?category_key=docs&lang=zh-CN`.
- Public CMS post detail: GET `/api/v1/cms/posts/{slug}`.
- Admin CMS categories: GET/POST/PATCH/DELETE `/admin/api/v1/cms/categories`.
- Admin CMS posts: GET/POST/PATCH/DELETE `/admin/api/v1/cms/posts`.
- Admin CMS blocks: GET/POST/PATCH/DELETE `/admin/api/v1/cms/blocks`.
- Admin upload: POST `/admin/api/v1/uploads` (multipart `file`), assets served under `/uploads/...`.

Homepage/Product page rendering notes:
- Page layout is driven by `cms_blocks` per `page` (`home` or `products`), `visible`, and `sort_order`.
- `type=custom_html` only allows safe HTML; `<script>`/`<iframe>` are blocked.
- `content_json` is a JSON string for structured blocks (hero, product_list, feature_cards, etc.).

## Authentication
- User JWT: `Authorization: Bearer <jwt>`
- Admin JWT: `Authorization: Bearer <jwt>`
- Robot inbound API key: `Authorization: Bearer <api_key>` or `X-API-Key`

User login flow:
1) GET `/api/v1/captcha`
2) POST `/api/v1/auth/register`
3) POST `/api/v1/auth/login` -> `access_token`

Admin login:
- POST `/admin/api/v1/auth/login` -> `access_token`

## Key Statuses
Order status:
- pending_review, rejected, approved, provisioning, active, failed, canceled

Order item status:
- pending_review, approved, provisioning, active, failed, rejected, canceled

VPS admin status (set by admin):
- normal, abuse, fraud, locked

Automation state mapping (from automation platform):
- 1 running
- 2 stopped
- 3 reinstalling
- 4 reinstall failed
- 5 expired locked
- 6 rescue
- 7 cracking password

## User Console APIs
- GET `/api/v1/me`
- PATCH `/api/v1/me`
- GET `/api/v1/dashboard`
- GET `/api/v1/catalog`
- GET `/api/v1/plan-groups?region_id=...`
- GET `/api/v1/packages?plan_group_id=...`
- Cart:
  - GET `/api/v1/cart`
  - POST `/api/v1/cart`
  - DELETE `/api/v1/cart`
  - PATCH `/api/v1/cart/{id}`
  - DELETE `/api/v1/cart/{id}`
- System images:
  - GET `/api/v1/system-images?plan_group_id=...`
- Billing cycles:
  - GET `/api/v1/billing-cycles`
- Orders:
  - POST `/api/v1/orders` (from cart)
  - POST `/api/v1/orders/items` (from items)
  - GET `/api/v1/orders`
  - GET `/api/v1/orders/{id}`
  - POST `/api/v1/orders/{id}/refresh`
  - GET `/api/v1/orders/{id}/events` (SSE)
- Wallet:
  - GET `/api/v1/wallet`
  - GET `/api/v1/wallet/transactions`
  - POST `/api/v1/wallet/recharge`
  - POST `/api/v1/wallet/withdraw`
  - GET `/api/v1/wallet/orders`
- Tickets:
  - POST `/api/v1/tickets`
  - GET `/api/v1/tickets`
  - GET `/api/v1/tickets/{id}`
  - POST `/api/v1/tickets/{id}/messages`
  - POST `/api/v1/tickets/{id}/close`
- VPS:
  - GET `/api/v1/vps`
  - GET `/api/v1/vps/{id}`
  - POST `/api/v1/vps/{id}/refresh`
  - GET `/api/v1/vps/{id}/panel`
  - POST `/api/v1/vps/{id}/renew`
  - POST `/api/v1/vps/{id}/resize`
  - POST `/api/v1/vps/{id}/refund`

SSE notes:
- Content-Type: text/event-stream
- Heartbeat every 15s
- Reconnect with `Last-Event-ID`

## Admin Console APIs
- Users:
  - GET `/admin/api/v1/users`
  - PATCH `/admin/api/v1/users/{id}`
  - PATCH `/admin/api/v1/users/{id}/status`
- Orders:
  - GET `/admin/api/v1/orders`
  - GET `/admin/api/v1/orders/{id}`
  - DELETE `/admin/api/v1/orders/{id}`
  - POST `/admin/api/v1/orders/{id}/approve`
  - POST `/admin/api/v1/orders/{id}/reject`
  - POST `/admin/api/v1/orders/{id}/retry`
  - GET `/admin/api/v1/scheduled-tasks`
  - PATCH `/admin/api/v1/scheduled-tasks/{key}`
- Tickets:
  - GET `/admin/api/v1/tickets`
  - GET `/admin/api/v1/tickets/{id}`
  - PATCH `/admin/api/v1/tickets/{id}`
  - POST `/admin/api/v1/tickets/{id}/messages`
  - DELETE `/admin/api/v1/tickets/{id}`
- VPS:
  - GET `/admin/api/v1/vps`
  - POST `/admin/api/v1/vps`
  - POST `/admin/api/v1/vps/{id}/lock`
  - POST `/admin/api/v1/vps/{id}/unlock`
  - POST `/admin/api/v1/vps/{id}/status` (set admin status + reason)
  - POST `/admin/api/v1/vps/{id}/emergency-renew`
  - POST `/admin/api/v1/vps/{id}/resize`
  - POST `/admin/api/v1/vps/{id}/refresh`
  - POST `/admin/api/v1/vps/{id}/delete`
- Catalog admin:
  - GET/POST/PATCH/DELETE `/admin/api/v1/regions`
  - GET/POST/PATCH/DELETE `/admin/api/v1/plan-groups`
  - GET/POST/PATCH/DELETE `/admin/api/v1/packages`
  - POST `/admin/api/v1/{regions|plan-groups|lines|packages|billing-cycles|system-images}/bulk-delete`
- System images:
  - GET `/admin/api/v1/system-images`
  - POST `/admin/api/v1/system-images`
  - PATCH `/admin/api/v1/system-images/{id}`
  - DELETE `/admin/api/v1/system-images/{id}`
  - POST `/admin/api/v1/system-images/sync?line_id=...`
  - POST `/admin/api/v1/lines/{id}/system-images` (set enabled images for a line)
- API keys:
  - GET `/admin/api/v1/api-keys`
  - POST `/admin/api/v1/api-keys`
  - PATCH `/admin/api/v1/api-keys/{id}`
- Wallet:
  - POST `/admin/api/v1/wallets/{user_id}/adjust`
  - GET `/admin/api/v1/wallets/{user_id}/transactions`
  - GET `/admin/api/v1/wallet/orders`
  - POST `/admin/api/v1/wallet/orders/{id}/approve`
  - POST `/admin/api/v1/wallet/orders/{id}/reject`
- Settings:
  - GET `/admin/api/v1/settings`
  - PATCH `/admin/api/v1/settings`
- SMTP test:
  - POST `/admin/api/v1/integrations/smtp/test`
- Email templates:
  - GET `/admin/api/v1/email-templates`
  - POST `/admin/api/v1/email-templates`
  - PATCH `/admin/api/v1/email-templates/{id}`
- Audit logs:
  - GET `/admin/api/v1/audit-logs`

## Settings Keys
- default_line_id
- default_port_num
- robot_webhook_url
- robot_webhook_secret
- robot_webhook_enabled
- smtp_host, smtp_port, smtp_user, smtp_pass, smtp_from
- smtp_enabled (true/false)
- email_enabled (true/false)
- email_expire_enabled (true/false)
- expire_reminder_days
- emergency_renew_enabled (true/false)
- emergency_renew_window_days
- emergency_renew_days
- emergency_renew_interval_hours
- auto_delete_enabled (true/false)
- auto_delete_days
- refund_full_days
- refund_prorate_days
- refund_no_refund_days
- refund_curve_json
- refund_requires_approval
- refund_on_admin_delete
- resize_charge_curve_json
- automation_base_url
- automation_api_key
- automation_enabled (true/false)
- automation_timeout_sec
- automation_retry
- automation_dry_run (true/false)

## Notes
- `plan_groups.line_id` is required for provisioning (automation create_host).
- `system_images` are global and enabled per line via `/admin/api/v1/lines/{id}/system-images`; `os` uses image name.
- `system_images.type` supports `linux` or `windows`; configure availability per `line_id` mapping.
- Automation sync updates line image availability from `/mirror_image?line_id=...`.
- Panel URL uses automation `/panel` with host_name + panel_password (fetched live).
- plan_groups/packages include `visible` and `capacity_remaining` and are editable via admin update; user APIs only return `active && visible`.
- packages include `port_num` (default 30) for NAT/port mapping count.
- admin package list supports `plan_group_id` filter for line-scoped editing.
- VPS/Order data is not auto-backfilled; use admin endpoints or provisioning flow explicitly.
- Admin VPS create supports `provision=true` for real provisioning; otherwise it only creates a record.
- Email templates support variables and HTML. See `backend/docs/email_template.md`.
