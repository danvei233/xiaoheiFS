# MockPay Disk Plugin

Test-only payment plugin for local/debug use. Declared `test_only: true` in
`manifest.json` and excluded from release bundles.

Config (dynamic schema from the plugin process):
- `confirm_text` (required): `我已确认我在进行测试而非生产环境`
- `ttl_sec` (optional): checkout link validity in seconds, default `86400`

Note: `public_base_url` is not a plugin config field. The host passes the
callback base URL via the payment `extra` params at CreatePayment time
(`base_url` / `site_url` / `origin` / `public_base_url` / `callback_base`).

Method exposed:
- `mock` (provider key in system: `mockpay.mock`)
