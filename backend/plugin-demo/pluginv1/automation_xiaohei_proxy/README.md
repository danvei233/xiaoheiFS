# automation_xiaohei_proxy

Xiaohei-to-Xiaohei proxy automation plugin.

## Config

```json
{
  "base_url": "https://upstream.example.com",
  "open_akid": "uak_xxx",
  "open_secret": "<secret returned when creating user api key>",
  "admin_api_key": "<optional but recommended>",
  "price_rate": 1.0,
  "goods_type_id": 0,
  "timeout_sec": 12,
  "retry": 1,
  "dry_run": false
}
```

## Upstream requirements

- `open_akid/open_secret` must belong to a user with enough wallet balance for instant orders.
- `admin_api_key` should have permissions for:
  - catalog read (`regions/plan-groups/packages/system-images`)
  - optional admin VPS actions (`lock/unlock/delete`)

## API mapping

- Catalog sync:
  - `GET /admin/api/v1/regions`
  - `GET /admin/api/v1/plan-groups`
  - `GET /admin/api/v1/packages`
  - `GET /admin/api/v1/system-images`
- Lifecycle / orders (signed open API):
  - `POST /api/v1/open/orders/instant/create`
  - `POST /api/v1/open/orders/instant/renew`
  - `POST /api/v1/open/orders/instant/resize`
  - `GET/POST /api/v1/open/vps/*`

## Notes

- Stock sync uses `capacity_remaining` from upstream package list.
- `price_rate` applies an upstream price multiplier when returning package prices.
- `goods_type_id > 0` limits catalog sync to a specific upstream goods type.
- If upstream plugin does not expose inventory, stock may remain `-1` (unlimited semantic).
