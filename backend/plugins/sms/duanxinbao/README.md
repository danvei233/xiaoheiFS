# Duanxinbao SMS Plugin

- Plugin ID: `duanxinbao`
- Category: `sms`

## Config

- `username`: 短信宝用户名（必填）
- `passwd`: 密码或 32 位 MD5（必填）
- `passwd_is_md5`: `true` 表示 `passwd` 已是 MD5，`false` 表示插件内自动转 MD5
- `goods_id`: 专用通道产品 ID（可选）
- `endpoint`: 发送接口地址，默认 `https://api.smsbao.com/sms`
- `timeout_sec`: 请求超时秒数

## Send behavior

- 使用 `SendSmsRequest.content` 作为短信内容（必填）
- `SendSmsRequest.phones` 支持单发/逗号拼接群发
- 当 `SendSmsRequest.template_id` 非空时，作为 `g` 参数覆盖 `goods_id`
- 返回 `0` 视为成功
