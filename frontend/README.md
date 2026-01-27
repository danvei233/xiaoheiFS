# Cloud VPS Console (Vue 3 + Vite + Pinia + Ant Design Vue)

## 启动

```bash
cd frontend
npm i
npm run dev
```

## 代理配置

默认走 Vite 代理（`vite.config.ts`）：

- `/api` -> `http://localhost:8080`
- `/admin/api` -> `http://localhost:8080`
- `/sdk` -> `http://localhost:8080`

如需直连环境，可设置：

```bash
VITE_API_BASE=http://localhost:8080
```

## 与后端对接说明（严格以 openapi.yaml + frontend-readme.md 为准）

用户端：
- `GET /api/v1/captcha`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/me`
- `GET /api/v1/dashboard`
- `GET /api/v1/catalog`
- `GET/POST/PATCH/DELETE /api/v1/cart`
- `GET/POST /api/v1/orders`
- `GET /api/v1/orders/{id}`
- `POST /api/v1/orders/{id}/refresh`
- `GET /api/v1/orders/{id}/events`（SSE）
- `GET /api/v1/vps`
- `GET /api/v1/vps/{id}`
- `POST /api/v1/vps/{id}/refresh`
- `GET /api/v1/vps/{id}/panel`
- `POST /api/v1/vps/{id}/renew`
- `POST /api/v1/vps/{id}/resize`

管理端：
- `POST /admin/api/v1/auth/login`
- `GET/PATCH /admin/api/v1/users`
- `GET /admin/api/v1/orders` + approve/reject/retry
- `GET /admin/api/v1/vps` + lock/unlock/delete/resize/refresh/status/emergency-renew
- `GET/POST/PATCH/DELETE /admin/api/v1/regions`
- `GET/POST/PATCH/DELETE /admin/api/v1/plan-groups`
- `GET/POST/PATCH/DELETE /admin/api/v1/packages`
- `GET/POST/PATCH/DELETE /admin/api/v1/system-images` + sync
- `GET/POST/PATCH /admin/api/v1/api-keys`
- `GET/PATCH /admin/api/v1/settings`
- `GET/POST/PATCH /admin/api/v1/email-templates`
- `GET /admin/api/v1/audit-logs`

## Settings Keys

- robot_webhook_url
- robot_webhook_key
- smtp_host, smtp_port, smtp_user, smtp_pass, smtp_from
- email_enabled (true/false)
- email_expire_enabled (true/false)
- expire_reminder_days
- emergency_renew_days
- emergency_renew_interval_hours

## 关键对接文件

- 用户端 API：`src/api/user.ts`
- 管理端 API：`src/api/admin.ts`
- 用户端鉴权：`src/stores/auth.ts`
- 用户端 Dashboard：`src/stores/dashboard.ts`
- 用户端订单：`src/stores/orders.ts`
- 用户端 VPS：`src/stores/vps.ts`

## SSE

订单详情页使用 `fetch + ReadableStream` 解析：

```
GET /api/v1/orders/{id}/events
```

会读取 `data:` 字段并追加到事件列表。
