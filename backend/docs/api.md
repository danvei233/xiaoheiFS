# Cloud VPS API Documentation

See openapi.yaml for machine-readable spec.

# Cloud VPS Console API

## Authentication
- User JWT: Authorization: Bearer <jwt>
- Admin JWT: Authorization: Bearer <jwt>
- Robot API Key: Authorization: Bearer <api_key> or X-API-Key

## Admin profile
- Get: GET /admin/api/v1/profile
- Update: PATCH /admin/api/v1/profile
- Change password: POST /admin/api/v1/profile/change-password

## Scheduled tasks
- List: GET /admin/api/v1/scheduled-tasks
- Update: PATCH /admin/api/v1/scheduled-tasks/{key}

## Settings keys
- robot_webhook_url, robot_webhook_secret, robot_webhook_enabled, robot_webhooks
- smtp_host, smtp_port, smtp_user, smtp_pass, smtp_from, smtp_enabled
- email_enabled, email_expire_enabled, expire_reminder_days
- emergency_renew_enabled, emergency_renew_window_days, emergency_renew_days, emergency_renew_interval_hours
- auto_delete_enabled, auto_delete_days
- refund_full_days, refund_prorate_days, refund_no_refund_days
- refund_full_hours, refund_prorate_hours, refund_no_refund_hours
- refund_curve_json
- refund_requires_approval, refund_on_admin_delete
- resize_price_mode, resize_refund_ratio, resize_rounding, resize_min_charge, resize_min_refund, resize_refund_to_wallet, resize_charge_curve_json
- debug_enabled
- automation_base_url, automation_api_key, automation_enabled, automation_timeout_sec, automation_retry, automation_dry_run
- payment_providers_enabled, payment_providers_config, payment_plugins
- payment_plugin_dir, payment_plugin_upload_password
- realname_enabled, realname_provider, realname_block_actions

## Debug
- Status: GET /admin/api/v1/debug/status
- Update: PATCH /admin/api/v1/debug/status
- Logs: GET /admin/api/v1/debug/logs?types=audit,automation,sync

## Order status flow
- draft -> pending_payment -> pending_review -> approved -> provisioning -> active
- rejected / canceled / failed track the final state
- Order field: `can_review` indicates whether admin can change status (pending_payment or rejected)

## Admin orders
- Delete order (super admin): DELETE /admin/api/v1/orders/{id}

## SSE
Endpoint: GET /api/v1/orders/{id}/events
- Content-Type: text/event-stream
- Heartbeat: every 15s
- Reconnect: pass Last-Event-ID to receive events with higher seq

## Message center
- List: GET /api/v1/notifications?status=unread
- Unread count: GET /api/v1/notifications/unread-count
- Mark read: POST /api/v1/notifications/{id}/read
- Mark all read: POST /api/v1/notifications/read-all

## Wallet orders
- Recharge: POST /api/v1/wallet/recharge
- Withdraw: POST /api/v1/wallet/withdraw
- List: GET /api/v1/wallet/orders
- Admin list: GET /admin/api/v1/wallet/orders
- Admin approve/reject: POST /admin/api/v1/wallet/orders/{id}/approve | /reject

## Refunds
- Request refund: POST /api/v1/vps/{id}/refund

## Renew VPS
- Create renew order: POST /api/v1/vps/{id}/renew
  - Body: {"duration_months": 1} or {"renew_days": 30}
  - 409: 已有待处理续费订单，请先处理或撤销

## Resize VPS
- Create resize order: POST /api/v1/vps/{id}/resize
  - Body: {"target_package_id":123,"spec":{"add_cores":2,"add_mem_gb":2,"add_disk_gb":20,"add_bw_mbps":10}}
  - Body: {"target_package_id":123,"reset_addons":true,"spec":{"add_cores":0,"add_mem_gb":0,"add_disk_gb":0,"add_bw_mbps":0}}
  - Order amount is the price delta (may be negative for downgrade)

## Real name verification
- Status: GET /api/v1/realname/status
- Verify: POST /api/v1/realname/verify

## End-to-end curl flow
1) Captcha + register + login

    curl http://localhost:8080/api/v1/captcha
    curl -X POST http://localhost:8080/api/v1/auth/register \
      -H "Content-Type: application/json" \
      -d '{"username":"u1","email":"u1@example.com","qq":"123","password":"pass","captcha_id":"...","captcha_code":"..."}'
    curl -X POST http://localhost:8080/api/v1/auth/login \
      -H "Content-Type: application/json" \
      -d '{"username":"u1","password":"pass"}'

2) Catalog

    curl -H "Authorization: Bearer <jwt>" http://localhost:8080/api/v1/catalog

3) Add to cart or create order with items

    curl -X POST http://localhost:8080/api/v1/cart \
      -H "Authorization: Bearer <jwt>" \
      -H "Content-Type: application/json" \
      -d '{"package_id":1,"system_id":1,"spec":{"add_cores":0,"add_mem_gb":0,"add_disk_gb":0,"add_bw_mbps":0,"billing_cycle_id":1,"cycle_qty":1},"qty":1}'

    curl -X POST http://localhost:8080/api/v1/orders \
      -H "Authorization: Bearer <jwt>" \
      -H "Idempotency-Key: order-001"

4) Select payment method and submit payment info (manual review)

    curl -H "Authorization: Bearer <jwt>" http://localhost:8080/api/v1/payments/providers

    curl -X POST http://localhost:8080/api/v1/orders/1/pay \
      -H "Authorization: Bearer <jwt>" \
      -H "Content-Type: application/json" \
      -d '{"method":"approval"}'

    curl -X POST http://localhost:8080/api/v1/orders/1/payments \
      -H "Authorization: Bearer <jwt>" \
      -H "Content-Type: application/json" \
      -H "Idempotency-Key: pay-001" \
      -d '{"method":"bank","amount":99,"trade_no":"T20250101001","note":"manual transfer"}'

5) Robot webhook approve (API key)

    curl -X POST http://localhost:8080/api/v1/integrations/robot/webhook \
      -H "Authorization: Bearer <api_key>" \
      -H "Content-Type: application/json" \
      -d '{"text":"通过订单 1","sender":"bot","timestamp":1735680000}'

6) Admin approve and provisioning

    curl -X POST http://localhost:8080/admin/api/v1/orders/1/approve \
      -H "Authorization: Bearer <admin_jwt>"

7) Query VPS list and panel redirect

    curl -H "Authorization: Bearer <jwt>" http://localhost:8080/api/v1/vps
    curl -I -H "Authorization: Bearer <jwt>" http://localhost:8080/api/v1/vps/1/panel
