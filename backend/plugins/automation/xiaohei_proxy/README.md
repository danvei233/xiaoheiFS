# xiaohei_proxy（代理对接自动化）

该内置插件用于把另一个 xiaohei 财务系统作为上游自动化节点进行对接。

## 配置项

- `base_url`: 上游地址，例如 `https://upstream.example.com`
- `open_akid`: 上游开放接口 `AKID`
- `open_secret`: 上游开放接口 `Secret`
- `admin_api_key`: 上游管理 API Key（用于目录/管理动作）
- `price_rate`: 价格倍率（同步套餐价格时使用）
- `goods_type_id`: 上游商品类型 ID（`0` 表示不过滤）
- `timeout_sec`: 请求超时秒数
- `retry`: 重试次数
- `dry_run`: 调试模式

