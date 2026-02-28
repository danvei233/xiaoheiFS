# MockPay Disk Plugin

Test-only payment plugin for local/debug use.

Required config:
- `confirm_text`: `我已确认我在进行测试而非生产环境`
- `public_base_url`: public URL of your backend (ngrok/cloudflared)
- `sign_key`: any non-empty secret

Method exposed:
- `mock` (provider key in system: `mockpay.mock`)
