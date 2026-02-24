# automation_mofang_openapi

Automation plugin for Mofang (ZJMF) OpenAPI.

## Config

```json
{
  "base_url": "https://mofang.example.com",
  "account": "user@example.com",
  "api_password": "******",
  "product_id": 1001,
  "billing_cycle": "monthly",
  "cancel_type": "Immediate",
  "cancel_reason": "automation cancel request",
  "configoption_template": {
    "1": "{{cpu}}",
    "2": "{{memory_gb}}",
    "3": "{{disk_gb}}"
  },
  "customfield_template": {
    "10": "{{host_name}}"
  },
  "os_template": {
    "101": "{{os}}"
  },
  "timeout_sec": 12,
  "retry": 1,
  "dry_run": false
}
```

## API mapping

- Auth: `POST /v1/login_api` (header `Authorization: JWT <token>`)
- Catalog: `GET /v1/products`
- Instance lifecycle:
  - `GET /v1/hosts`, `GET /v1/hosts/:id`
  - `PUT /v1/hosts/:id/module/on|off|reboot|repassword|reinstall|vnc`
  - `GET /v1/hosts/:id/module/status`
  - `POST /v1/hosts/:id/renew`
  - `POST /v1/hosts/:id/cancel`
- Create flow:
  - `POST /v1/cart/products`
  - `POST /v1/cart/checkout`
