# lightboat（轻舟自动化 / 面板对接）

## 能力

manifest 中 `capabilities.automation.features` 声明支持的模块（当前内置 lightboat 默认支持完整实例能力：catalog_sync / lifecycle / panel_login / vnc / reinstall / reset_password / resize / refund / port_mapping / backup / snapshot / firewall）。

## 配置项（插件管理页 -> 配置）

来自 `schemas/config.schema.json`：

- `base_url`：轻舟 API Base URL（示例：`https://panel.example.com/index.php/api/cloud`）
- `api_key`（secret）：API Key
- `timeout_sec`：请求超时（秒）
- `retry`：重试次数（仅对幂等请求生效）
- `dry_run`：仅调试用，开启后不会执行破坏性操作

## 多实例（多套轻舟）

同一个 `plugin_id=lightboat` 可以创建多个 `instance_id`（例如 `qz_a`、`qz_b`），每个实例独立配置/启停。
