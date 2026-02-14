# mangzhu_realname（芒竹云实名认证）

## 能力

- `Start`：发起实名认证
- `QueryResult`：查询实名认证结果

## 配置项

- `base_url`：默认 `https://e.mangzhuyun.cn`
- `api_key`：芒竹云分配密钥（必填）
- `auth_mode`：`two_factor` / `three_factor` / `face`
- `face_provider`：`baidu` / `wechat`（`auth_mode=face` 时使用）
- `callback_url`：面容流程回调地址（可由 Start 参数覆盖）
- `timeout_sec`：HTTP 超时秒数

## Start 参数约定

- 通用必填：`params.name`、`params.id_number`
- 三要素：额外传 `params.mobile`（兼容 `params.phone`）
- 面容流程：可传 `params.callback_url`

## 状态说明

- 同步模式（二要素/三要素）：
  - `Start` 会调用上游并缓存结果，返回 `next_step=query_result`
  - `QueryResult` 返回 `VERIFIED` 或 `FAILED`
- 面容模式（百度/微信）：
  - `Start` 返回 `url` 与 `token`，`next_step=redirect`
  - `QueryResult` 返回 `PENDING`/`VERIFIED`/`FAILED`

