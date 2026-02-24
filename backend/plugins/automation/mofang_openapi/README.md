# mofang_openapi（魔方 OpenAPI 自动化）

该内置插件用于对接魔方系统（`/v1/login_api` + JWT）的自动化能力。

## 配置项

- `base_url`: 魔方地址，例如 `https://mofang.example.com`
- `account`: 开启 API 对接功能的账号（邮箱或手机号）
- `api_password`: 该账号的 API 密码
- `product_id`: 创建实例时默认产品 ID
- `billing_cycle`: 默认续费/开通周期（默认 `monthly`）
- `cancel_type`: 销毁策略（`Immediate`/`Endofbilling`）
- `cancel_reason`: 销毁原因
- `configoption_template`: 开通参数模板（支持 `{{cpu}}/{{memory_gb}}/{{disk_gb}}/{{bandwidth}}/{{host_name}}/{{sys_pwd}}`）
- `customfield_template`: 自定义字段模板（支持同上占位符）
- `os_template`: 系统选择模板（支持 `{{os}}`）
- `timeout_sec`: 请求超时秒数
- `retry`: 重试次数
- `dry_run`: 调试模式

