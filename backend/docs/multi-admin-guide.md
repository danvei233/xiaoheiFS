# 多管理员系统指南

## 概述

本系统支持多管理员管理，每个管理员可以分配不同的权限组，实现精细化的权限控制。

## 权限组设计

### 预设权限组

#### 超级管理员
- **权限**：`["*"]`（所有权限）
- **说明**：拥有系统的所有操作权限

#### 运维管理员
- **权限**：
  ```json
  [
    "user.list", "user.view",
    "order.list", "order.view", "order.approve", "order.reject",
    "vps.*",
    "audit_log.view"
  ]
  ```
- **说明**：负责VPS运维和订单审核

#### 客服管理员
- **权限**：
  ```json
  [
    "user.list", "user.view",
    "order.list", "order.view",
    "vps.list", "vps.view"
  ]
  ```
- **说明**：负责用户和订单查询

#### 财务管理员
- **权限**：
  ```json
  [
    "order.list", "order.view", "order.approve", "order.reject",
    "audit_log.view"
  ]
  ```
- **说明**：负责订单审核和财务管理

## API 使用说明

### 1. 获取权限列表

```
GET /admin/api/v1/permissions
Authorization: Bearer <admin_jwt>
```

响应示例：
```json
{
  "items": [
    "user.list",
    "user.create",
    "user.update",
    "user.delete",
    "user.reset_password",
    "user.view",
    ...
  ]
}
```

### 2. 创建管理员

```
POST /admin/api/v1/admins
Authorization: Bearer <admin_jwt>
Content-Type: application/json

{
  "username": "admin2",
  "email": "admin2@example.com",
  "qq": "123456789",
  "password": "password123",
  "permission_group_id": 2
}
```

### 3. 更新管理员

```
PATCH /admin/api/v1/admins/2
Authorization: Bearer <admin_jwt>
Content-Type: application/json

{
  "username": "admin2",
  "email": "newemail@example.com",
  "qq": "987654321",
  "permission_group_id": 3
}
```

### 4. 创建权限组

```
POST /admin/api/v1/permission-groups
Authorization: Bearer <admin_jwt>
Content-Type: application/json

{
  "name": "销售管理员",
  "description": "负责产品销售相关",
  "permissions": [
    "product.list",
    "product.create",
    "product.update",
    "order.list",
    "order.view"
  ]
}
```

### 5. 修改个人密码

```
POST /admin/api/v1/profile/change-password
Authorization: Bearer <admin_jwt>
Content-Type: application/json

{
  "old_password": "oldpass123",
  "new_password": "newpass123"
}
```

### 6. 密码找回

#### 请求重置
```
POST /api/v1/auth/forgot-password
Content-Type: application/json

{
  "email": "admin@example.com"
}
```

#### 重置密码
```
POST /api/v1/auth/reset-password
Content-Type: application/json

{
  "token": "reset_token_from_email",
  "new_password": "newpass123"
}
```

## 权限常量参考

### 用户管理
- `user.list` - 查看用户列表
- `user.create` - 创建用户
- `user.update` - 更新用户
- `user.delete` - 删除用户
- `user.reset_password` - 重置用户密码
- `user.view` - 查看用户详情

### 订单管理
- `order.list` - 查看订单列表
- `order.view` - 查看订单详情
- `order.approve` - 批准订单
- `order.reject` - 驳回订单
- `order.delete` - 删除订单

### VPS管理
- `vps.list` - 查看VPS列表
- `vps.view` - 查看VPS详情
- `vps.create` - 创建VPS
- `vps.update` - 更新VPS
- `vps.delete` - 删除VPS
- `vps.resize` - 调整VPS配置
- `vps.renew` - 续费VPS
- `vps.admin_status` - 设置VPS管理员状态

### 系统设置
- `settings.view` - 查看系统设置
- `settings.update` - 更新系统设置

### 管理员管理
- `admin.list` - 查看管理员列表
- `admin.create` - 创建管理员
- `admin.update` - 更新管理员
- `admin.delete` - 删除管理员

### 审计日志
- `audit_log.view` - 查看审计日志

### API密钥
- `api_key.list` - 查看API密钥列表
- `api_key.create` - 创建API密钥
- `api_key.update` - 更新API密钥
- `api_key.delete` - 删除API密钥

### 邮件模板
- `email_template.list` - 查看邮件模板列表
- `email_template.update` - 更新邮件模板
- `email_template.delete` - 删除邮件模板

### 产品管理
- `product.list` - 查看产品列表
- `product.create` - 创建产品
- `product.update` - 更新产品
- `product.delete` - 删除产品

### 区域管理
- `region.list` - 查看区域列表
- `region.create` - 创建区域
- `region.update` - 更新区域
- `region.delete` - 删除区域

### 计费周期
- `billing_cycle.list` - 查看计费周期列表
- `billing_cycle.create` - 创建计费周期
- `billing_cycle.update` - 更新计费周期
- `billing_cycle.delete` - 删除计费周期

### 系统镜像
- `system_image.list` - 查看系统镜像列表
- `system_image.create` - 创建系统镜像
- `system_image.update` - 更新系统镜像
- `system_image.delete` - 删除系统镜像

### 权限组
- `permission_group.list` - 查看权限组列表
- `permission_group.create` - 创建权限组
- `permission_group.update` - 更新权限组
- `permission_group.delete` - 删除权限组

## 头像功能

管理员头像使用QQ头像API自动生成：

```
https://q1.qlogo.cn/g?b=qq&nk={qq}&s=100
```

- `b=qq`：QQ头像
- `nk={qq}`：QQ号
- `s=100`：尺寸（100px）

当管理员的QQ号变更时，头像URL会自动更新。

## 安全注意事项

1. **权限分配**：谨慎分配权限，遵循最小权限原则
2. **密码重置**：密码重置令牌有效期24小时，使用后立即失效
3. **自我保护**：管理员不能删除自己
4. **审计日志**：所有管理员操作都会被记录到审计日志

## 前端集成建议

### 1. 管理员列表页面
- 显示管理员头像、用户名、邮箱、QQ、权限组
- 提供创建、编辑、删除按钮
- 删除按钮对当前用户隐藏

### 2. 权限组管理页面
- 显示所有权限组及其权限
- 创建/编辑时提供权限选择器
- 支持搜索和过滤权限

### 3. 个人资料页面
- 显示当前管理员信息
- 提供邮箱和QQ修改功能
- 提供密码修改功能（需要验证旧密码）

### 4. 权限控制
- 根据用户权限显示/隐藏菜单项
- 根据用户权限显示/隐藏操作按钮
- 权限不足时显示友好提示

### 5. 密码找回流程
1. 用户点击"忘记密码"
2. 输入邮箱
3. 系统发送重置邮件
4. 用户点击邮件中的链接或输入令牌
5. 输入新密码完成重置