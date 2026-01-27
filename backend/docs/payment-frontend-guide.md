# 前端支付适配指南

## 支付方式列表
接口：`GET /api/v1/payments/providers`

返回字段说明：
- `key`：支付方式标识（approval/balance/custom/yipay/插件key）
- `name`：展示名
- `schema_json`：配置结构（可用于渲染动态表单）
- `config_json`：当前配置（仅前端展示用）
- `balance`：余额方式会返回余额

## 选择支付方式
接口：`POST /api/v1/orders/{id}/pay`
```json
{
  "method": "approval|balance|custom|yipay|插件key",
  "return_url": "https://example.com/pay/return",
  "notify_url": "https://api.example.com/api/v1/payments/notify/yipay"
}
```
返回：
- `status`：`pending_payment`/`approved`/`manual`
- `pay_url`：跳转链接（如 yipay/custom）
- `trade_no`：平台订单号
- `extra`：补充字段
- `paid`：是否已完成支付（余额支付为 true）

## 人工审核支付（approval）
仍需调用：
`POST /api/v1/orders/{id}/payments`
提交 `method/amount/trade_no/note/screenshot_url`，然后管理员审核。

## 钱包接口
- `GET /api/v1/wallet`：钱包余额
- `GET /api/v1/wallet/transactions`：交易记录

## 头像与资料
用户资料返回 `avatar_url`，前端禁止上传头像。
用户可编辑字段：`username/email/qq/phone/bio/intro/password`（除 id 外都可改）。

## 典型流程
1) 获取支付方式列表
2) 用户选择方式并调用 `/orders/{id}/pay`
3) 若返回 `pay_url`，前端跳转；若 `manual`，进入人工提交信息页面
4) 支付成功后轮询订单状态或订阅 SSE
