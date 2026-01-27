# 支付插件设计说明

本系统支付模块分为内置方式与插件方式�?- 内置：`approval`（人工审核）、`balance`（余额支付）、`custom`（自定义支付页）、`yipay`（易支付�?- 插件：通过 Hashicorp go-plugin 动态加�?
## 核心设置�?支付模块的配置都保存�?settings 表：
- `payment_providers_enabled`：JSON map，key 为支付方式，value 为是否启�?- `payment_providers_config`：JSON map，key 为支付方式，value 为该方式配置 JSON
- `payment_plugins`：JSON array，定义插件路径与 key
- `payment_plugin_dir`：自动扫描目录（默认 `plugins/payment`�?- `payment_plugin_upload_password`：上传校验密码（默认 `qweasd123456`，仅配置 + 重启生效�?
示例�?```json
{
  "payment_providers_enabled": {
    "approval": true,
    "balance": true,
    "custom": true,
    "yipay": false
  },
  "payment_providers_config": {
    "custom": { "pay_url": "", "instructions": "" },
    "yipay": {
      "base_url": "https://pays.org.cn/submit.php",
      "pid": "",
      "key": "",
      "pay_type": "",
      "notify_url": "",
      "return_url": "",
      "sign_type": "MD5"
    }
  },
  "payment_plugins": [
    { "key": "stripe", "path": "D:/plugins/stripe-pay.exe" }
  ]
}
```

## 自动扫描与启用策�?- 后端会扫�?`payment_plugin_dir` 下的可执行文件�?- 新插件默认只出现在列表里，只有控制台启用后才会真正加载�?- 如需变更上传密码，请修改配置并重启服务�?
## 插件上传 API（最高管理员�?- `POST /admin/api/v1/plugins/payment/upload`
- `multipart/form-data`�?  - `file`：插件二进制
  - `password`：上传安全密码（�?`X-Plugin-Password` Header�?
## 插件接口
插件通过 go-plugin NetRPC 协议实现，接口定义在�?`backend/internal/adapter/payment/plugin/plugin.go`

插件需实现 Provider 接口�?- `Key() string`
- `Name() string`
- `SchemaJSON() string`
- `SetConfig(configJSON string) error`
- `CreatePayment(req PaymentCreateRequest) (PaymentCreateResult, error)`
- `VerifyNotify(params map[string]string) (PaymentNotifyResult, error)`

### CreatePayment
输入�?- 订单ID、用户ID、金额、币种、标题、回调URL

输出�?- `trade_no`（可自定义）
- `pay_url`（跳转地址�?- `extra`（补充字段）

### VerifyNotify
用于支付平台回调校验与解析，返回�?- `trade_no`
- `paid` 是否已支�?- `amount` 实际金额

## 内置支付方式行为
- `approval`：不生成第三方支付链接，只提示走人工审核（提�?`/orders/{id}/payments`）�?- `balance`：直接扣余额并标记支付通过，进入后续审批与开通流程�?- `custom`：返�?`pay_url` �?`instructions`�?- `yipay`：生成支付跳转链接，并走 `/payments/notify/yipay` 回调�?
## 后端加载逻辑
- 注册内置方式
- 读取 settings 并合并配�?- 读取 `payment_plugins` 启动插件进程并挂载到注册�?
如需新增支付插件，只需提供可执行文件并写入 `payment_plugins`�?
## ��ע
- �������������� `payment_plugin_dir` Ŀ¼�仯���Զ�ˢ�²���б��������Զ����ã���


## Demo plugin
The repository includes a demo payment plugin source at:
- ackend/pkg/payment_demo`r

Build example (Windows):
` 
go build -o plugins/payment/demo_pay.exe ./pkg/payment_demo
` 

Then upload the binary or add it to payment_plugins in settings to load it.
