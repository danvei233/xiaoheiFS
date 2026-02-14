# 芒竹智能云视觉智能开放平台 API 文档

- 版本：V1.0  
- 平台地址：https://e.mangzhuyun.cn  
- 更新日期：2025年5月  

> ⚠️ 全链路 HTTPS 协议：本平台所有 API 接口均强制使用 HTTPS 协议，禁止 HTTP 请求。

---

## 系统级错误码

| 错误码 | 说明 |
|---:|---|
| 400 | 请求格式错误 |
| 403 | 无权限访问接口 |
| 405 | 请求方法不支持 |
| 408 | 请求超时 |
| 500 | 服务器内部错误 |
| 503 | 服务暂不可用（维护中） |

---

## 二要素验证 API

- 接口地址：https://e.mangzhuyun.cn/index/sm_api  
- 请求方式：GET/POST  
- 返回格式：JSON  

### 请求参数

| 参数名 | 必填 | 说明 |
|---|---|---|
| key | 是 | 密钥（由平台分配） |
| name | 是 | 姓名 |
| idcard | 是 | 身份证号 |

### 请求示例

GET/POST  
`https://e.mangzhuyun.cn/index/sm_api?key=YOUR_KEY&name=张三&idcard=11010119900307001X`

### 返回示例

```json
{
  "code": 200,
  "result": "验证通过",
  "request_id": "1234567890"
}
```

---

## 三要素验证 API

- 接口地址：https://e.mangzhuyun.cn/index/sm3_api  
- 请求方式：GET/POST  
- 返回格式：JSON  

### 请求参数

| 参数名 | 必填 | 说明 |
|---|---|---|
| key | 是 | 密钥（由平台分配） |
| name | 是 | 姓名 |
| idcard | 是 | 身份证号 |
| mobile | 是 | 手机号码 |

### 请求示例

GET/POST  
`https://e.mangzhuyun.cn/index/sm3_api?key=YOUR_KEY&name=李四&idcard=310105198005060032&mobile=138001380001`

### 返回示例

```json
{
  "code": 200,
  "result": "三要素验证成功",
  "request_id": "0987654321"
}
```

---

## 微信面容ID验证 API【已下线】

- 实名接口地址：https://e.mangzhuyun.cn/index/wx_sm  
- 结果查询接口地址：https://e.mangzhuyun.cn/index/wx_cx  
- 请求方式：GET/POST  

### 实名请求参数

| 参数名 | 必填 | 说明 |
|---|---|---|
| key | 是 | 密钥（由平台分配） |
| name | 是 | 姓名 |
| idcard | 是 | 身份证号 |
| url | 是 | 回调网址 |

### 实名请求示例

GET/POST  
`https://e.mangzhuyun.cn/index/wx_sm?key=YOUR_KEY&name=王五&idcard=330103199012120025&url=abc.com`

### 实名返回示例

```json
{
  "code": 200,
  "url": "https://...",
  "token": "efgh5678"
}
```

### 结果查询参数

| 参数名 | 必填 | 说明 |
|---|---|---|
| key | 是 | 密钥（由平台分配） |
| token | 是 | 实名接口返回的令牌 |

### 结果查询示例

GET/POST  
`https://e.mangzhuyun.cn/index/wx_cx?key=YOUR_KEY&token=330103199012120025`

### 结果查询返回示例

```json
{
  "code": 200,
  "url": "https://...",
  "token": "abcd1234"
}
```

---

## 百度面容ID验证 API

- 实名接口地址：https://e.mangzhuyun.cn/index/bd_sm  
- 结果查询接口地址：https://e.mangzhuyun.cn/index/bd_cx  
- 请求方式：GET/POST  

### 实名请求参数

| 参数名 | 必填 | 说明 |
|---|---|---|
| key | 是 | 密钥（由平台分配） |
| name | 是 | 姓名 |
| idcard | 是 | 身份证号 |
| url | 是 | 回调网址 |

### 实名请求示例

GET/POST  
`https://e.mangzhuyun.cn/index/bd_sm?key=YOUR_KEY&name=王五&idcard=330103199012120025&url=abc.com`

### 实名返回示例

```json
{
  "code": 200,
  "url": "https://...",
  "token": "efgh5678"
}
```

### 结果查询参数

| 参数名 | 必填 | 说明 |
|---|---|---|
| key | 是 | 密钥（由平台分配） |
| token | 是 | 实名接口返回的令牌 |

### 结果查询示例

GET/POST  
`https://e.mangzhuyun.cn/index/bd_cx?key=YOUR_KEY&token=330103199012120025`

### 结果查询返回示例

```json
{
  "code": 200,
  "url": "https://...",
  "token": "efgh5678"
}
```

---

## 注意事项

- 密钥安全：`key` 为敏感信息，请勿泄露或明文传输。  
- 错误处理：业务错误请根据返回的 `code` 和 `msg` 字段处理。  
- 结果查询需通过 `token` 调用专用接口。  

---

## 技术支持

- 邮箱：hr@mangzhuyun.cn  
- 作者：admin  
- 创建时间：2025-05-31 01:33  
- 最后编辑：admin  
- 更新时间：2025-05-31 02:03  
