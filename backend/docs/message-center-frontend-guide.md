# 消息中心前端适配说明

## 目标
- 顶部或侧栏显示未读红点
- 支持未读/已读/全部筛选

## 接口
- `GET /api/v1/notifications?status=unread|read`
  - 响应：`{ items: Notification[], total: number }`
- `GET /api/v1/notifications/unread-count`
  - 响应：`{ unread: number }`
- `POST /api/v1/notifications/{id}/read`
- `POST /api/v1/notifications/read-all`

## 通知类型
- `provisioned`：开通成功
- `provision_failed`：开通失败
- `expire`：到期提醒
- `vps_destroyed`：销毁通知
- `ticket_reply`：工单回复
- `announcement`：新公告

## 建议交互
- 页面加载时拉取 `unread-count` 作为红点依据
- 列表页按需分页查询 `notifications`
- 点击消息或“全部已读”时调用对应接口
