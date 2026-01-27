# 更新记录

## 2026-01-08
### 支付与钱包
- 新增支付方式选择接口：`POST /api/v1/orders/{id}/pay`，支持 `approval`（人工）、`balance`（余额）、`custom`（自定义）、`yipay`（易支付）以及插件扩展。
- 新增支付方式列表：`GET /api/v1/payments/providers`，返回可用方式、`schema_json`、`config_json`；余额方式会返回 `balance`。
- 新增支付回调：`POST /api/v1/payments/notify/{provider}`，用于支付平台异步通知。
- 新增钱包接口：`GET /api/v1/wallet`、`GET /api/v1/wallet/transactions`。
- 新增管理端钱包调整：`POST /admin/api/v1/wallets/{user_id}/adjust`、`GET /admin/api/v1/wallets/{user_id}/transactions`。
- 新增管理端支付方式配置：`GET /admin/api/v1/payments/providers`、`PATCH /admin/api/v1/payments/providers/{key}`。

### 账号资料
- 用户资料字段新增：`phone`、`bio`、`intro`。
- 头像不允许上传，接口返回 `avatar_url`（优先QQ头像，其次默认生成）。

### 插件化支付
- 引入 Hashicorp go-plugin 支付插件框架，可通过配置加载第三方支付插件。
- 新增支付相关设置项：
  - `payment_providers_enabled`
  - `payment_providers_config`
  - `payment_plugins`

### 文档与规范
- OpenAPI 与 API 文档已同步更新：`backend/docs/openapi.yaml`、`backend/docs/api.md`。
- 插件设计与前端适配说明见：
  - `backend/docs/payment-plugins.md`
  - `backend/docs/payment-frontend-guide.md`

### �������ά
- ����֧������ϴ��ӿڣ�`POST /admin/api/v1/plugins/payment/upload`��֧�ֶ������ϴ��밲ȫ����У�顣
- ֧�����֧��Ŀ¼�Զ�ɨ���������δ���õĲ��ֻչʾ�����ء�
- ����������״̬�ӿڣ�`GET /admin/api/v1/server/status`��

### ��Ϣ����
- ������Ϣ���Ľӿڣ�`GET /api/v1/notifications`��`GET /api/v1/notifications/unread-count`��`POST /api/v1/notifications/{id}/read`��`POST /api/v1/notifications/read-all`��
- ����֪ͨ��������ͨ�ɹ����������ѡ����١������ظ����¹��档


## 2026-01-09
### Plugins
- Added demo payment plugin source: ackend/pkg/payment_demo`r
- Added demo realname provider source: ackend/pkg/realname_demo`r
- Added realname provider doc: ackend/docs/realname-plugins.md`r
