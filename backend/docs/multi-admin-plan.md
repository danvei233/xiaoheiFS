# 多管理员系统实现计划

## 一、需求概述

1. 支持多管理员
2. 支持权限和权限组配置
3. 支持管理员信息修改（用户名、密码、邮箱、QQ、权限组）
4. 头像从QQ头像获取（使用网络API）
5. 支持管理员修改自己的密码、邮箱和QQ
6. 支持email找回密码
7. 添加密码重置邮件模板
8. 完善文档和更新记录

## 二、现状分析

### 已有功能
- ✅ users表（id, username, email, qq, password_hash, role, status, created_at, updated_at）
- ✅ 用户角色：user, admin
- ✅ JWT认证和RequireAdmin中间件
- ✅ 邮件发送功能（SMTP）
- ✅ 邮件模板系统
- ✅ 审计日志系统

### 缺失功能
- ❌ 权限组和细粒度权限控制
- ❌ 管理员头像
- ❌ 密码找回功能
- ❌ 权限验证中间件
- ❌ 管理员权限组分配

## 三、数据库设计

### 3.1 修改 users 表
```sql
ALTER TABLE users ADD COLUMN avatar TEXT;
ALTER TABLE users ADD COLUMN permission_group_id INTEGER;
```

### 3.2 新建 permission_groups 表
```sql
CREATE TABLE permission_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    permissions_json TEXT NOT NULL, -- JSON数组，存储权限列表
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### 3.3 新建 password_reset_tokens 表
```sql
CREATE TABLE password_reset_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    used INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id)
);
CREATE INDEX idx_password_reset_tokens_token ON password_reset_tokens(token);
CREATE INDEX idx_password_reset_tokens_user ON password_reset_tokens(user_id);
```

### 3.4 新建 audit_logs 表（如果不存在）
```sql
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    action TEXT NOT NULL,
    detail_json TEXT NOT NULL,
    ip_address TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id)
);
```

## 四、权限设计

### 4.1 权限列表
```go
const (
    // 用户管理
    PermissionUserList      = "user.list"
    PermissionUserCreate    = "user.create"
    PermissionUserUpdate    = "user.update"
    PermissionUserDelete    = "user.delete"
    PermissionUserResetPass = "user.reset_password"
    PermissionUserView      = "user.view"

    // 订单管理
    PermissionOrderList   = "order.list"
    PermissionOrderView   = "order.view"
    PermissionOrderApprove = "order.approve"
    PermissionOrderReject  = "order.reject"
    PermissionOrderDelete  = "order.delete"

    // VPS管理
    PermissionVPSList        = "vps.list"
    PermissionVPSView        = "vps.view"
    PermissionVPSCreate      = "vps.create"
    PermissionVPSUpdate      = "vps.update"
    PermissionVPSDelete      = "vps.delete"
    PermissionVPSResize      = "vps.resize"
    PermissionVPSRenew       = "vps.renew"
    PermissionVPSAdminStatus = "vps.admin_status"

    // 系统设置
    PermissionSettingsView   = "settings.view"
    PermissionSettingsUpdate = "settings.update"

    // 管理员管理
    PermissionAdminList   = "admin.list"
    PermissionAdminCreate = "admin.create"
    PermissionAdminUpdate = "admin.update"
    PermissionAdminDelete = "admin.delete"

    // 审计日志
    PermissionAuditLogView = "audit_log.view"

    // API密钥管理
    PermissionAPIKeyList   = "api_key.list"
    PermissionAPIKeyCreate = "api_key.create"
    PermissionAPIKeyUpdate = "api_key.update"
    PermissionAPIKeyDelete = "api_key.delete"

    // 邮件模板管理
    PermissionEmailTemplateList   = "email_template.list"
    PermissionEmailTemplateUpdate = "email_template.update"
    PermissionEmailTemplateDelete = "email_template.delete"

    // 产品管理
    PermissionProductList   = "product.list"
    PermissionProductCreate = "product.create"
    PermissionProductUpdate = "product.update"
    PermissionProductDelete = "product.delete"

    // 区域管理
    PermissionRegionList   = "region.list"
    PermissionRegionCreate = "region.create"
    PermissionRegionUpdate = "region.update"
    PermissionRegionDelete = "region.delete"

    // 计费周期管理
    PermissionBillingCycleList   = "billing_cycle.list"
    PermissionBillingCycleCreate = "billing_cycle.create"
    PermissionBillingCycleUpdate = "billing_cycle.update"
    PermissionBillingCycleDelete = "billing_cycle.delete"

    // 系统镜像管理
    PermissionSystemImageList   = "system_image.list"
    PermissionSystemImageCreate = "system_image.create"
    PermissionSystemImageUpdate = "system_image.update"
    PermissionSystemImageDelete = "system_image.delete"
)
```

### 4.2 默认权限组

#### 超级管理员
```json
["*"] // 所有权限
```

#### 运维管理员
```json
[
    "user.list", "user.view",
    "order.list", "order.view", "order.approve", "order.reject",
    "vps.*",
    "audit_log.view"
]
```

#### 客服管理员
```json
[
    "user.list", "user.view",
    "order.list", "order.view",
    "vps.list", "vps.view"
]
```

#### 财务管理员
```json
[
    "order.list", "order.view", "order.approve",
    "order.*",
    "audit_log.view"
]
```

## 五、API设计

### 5.1 管理员管理API

#### 获取管理员列表
```
GET /admin/api/v1/admins
Authorization: Bearer <admin_jwt>
```

#### 创建管理员
```
POST /admin/api/v1/admins
Authorization: Bearer <admin_jwt>
Content-Type: application/json

{
    "username": "admin2",
    "email": "admin2@example.com",
    "qq": "123456",
    "password": "password123",
    "permission_group_id": 1
}
```

#### 获取管理员详情
```
GET /admin/api/v1/admins/:id
Authorization: Bearer <admin_jwt>
```

#### 更新管理员信息
```
PATCH /admin/api/v1/admins/:id
Authorization: Bearer <admin_jwt>
Content-Type: application/json

{
    "username": "admin2",
    "email": "newemail@example.com",
    "qq": "654321",
    "permission_group_id": 2
}
```

#### 删除管理员
```
DELETE /admin/api/v1/admins/:id
Authorization: Bearer <admin_jwt>
```

### 5.2 权限组管理API

#### 获取权限组列表
```
GET /admin/api/v1/permission-groups
Authorization: Bearer <admin_jwt>
```

#### 创建权限组
```
POST /admin/api/v1/permission-groups
Authorization: Bearer <admin_jwt>
Content-Type: application/json

{
    "name": "运维管理员",
    "description": "负责VPS运维和订单审核",
    "permissions": ["user.list", "user.view", "order.list", "order.view", "order.approve", "order.reject", "vps.*"]
}
```

#### 更新权限组
```
PATCH /admin/api/v1/permission-groups/:id
Authorization: Bearer <admin_jwt>
Content-Type: application/json

{
    "name": "运维管理员",
    "description": "负责VPS运维",
    "permissions": ["user.list", "user.view", "vps.*"]
}
```

#### 删除权限组
```
DELETE /admin/api/v1/permission-groups/:id
Authorization: Bearer <admin_jwt>
```

#### 获取所有可用权限
```
GET /admin/api/v1/permissions
Authorization: Bearer <admin_jwt>
```

### 5.3 个人资料管理API

#### 获取当前管理员信息
```
GET /admin/api/v1/profile
Authorization: Bearer <admin_jwt>
```

#### 更新当前管理员信息
```
PATCH /admin/api/v1/profile
Authorization: Bearer <admin_jwt>
Content-Type: application/json

{
    "email": "newemail@example.com",
    "qq": "654321"
}
```

#### 修改当前管理员密码
```
POST /admin/api/v1/profile/change-password
Authorization: Bearer <admin_jwt>
Content-Type: application/json

{
    "old_password": "oldpass123",
    "new_password": "newpass123"
}
```

### 5.4 密码找回API

#### 请求密码重置（管理员）
```
POST /admin/api/v1/auth/forgot-password
Content-Type: application/json

{
    "email": "admin@example.com"
}
```

#### 重置密码
```
POST /admin/api/v1/auth/reset-password
Content-Type: application/json

{
    "token": "reset_token_here",
    "new_password": "newpass123"
}
```

## 六、实现步骤

### 第1步：数据库迁移
1. 修改 `internal/adapter/repo/migrate.go`
   - 添加 `avatar` 和 `permission_group_id` 字段到 users 表
   - 创建 `permission_groups` 表
   - 创建 `password_reset_tokens` 表
   - 创建索引

2. 修改 `internal/domain/models.go`
   - 添加 `Avatar` 和 `PermissionGroupID` 字段到 User 结构体
   - 添加 `PermissionGroup` 结构体
   - 添加 `PasswordResetToken` 结构体
   - 添加权限常量

### 第2步：Repository层
1. 修改 `internal/adapter/repo/sqlite_repo.go`
   - 更新 `CreateUser`、`UpdateUser` 方法
   - 添加权限组相关方法：`ListPermissionGroups`、`GetPermissionGroup`、`CreatePermissionGroup`、`UpdatePermissionGroup`、`DeletePermissionGroup`
   - 添加密码重置令牌方法：`CreatePasswordResetToken`、`GetPasswordResetToken`、`MarkPasswordResetTokenUsed`
   - 更新 `ListUsers` 方法以支持过滤管理员

2. 更新 `internal/usecase/ports.go`
   - 添加对应的接口定义

### 第3步：Service层
1. 修改 `internal/usecase/admin_service.go`
   - 添加权限组管理方法
   - 添加管理员列表、创建、更新、删除方法
   - 更新权限验证逻辑

2. 创建 `internal/usecase/permission_service.go`
   - 权限验证服务
   - 检查用户是否具有特定权限

3. 创建 `internal/usecase/password_reset_service.go`
   - 生成重置令牌
   - 验证令牌
   - 发送重置邮件
   - 重置密码

4. 修改 `internal/usecase/admin_service.go`
   - 添加个人资料管理方法
   - 添加修改密码方法

### 第4步：HTTP Handler层
1. 修改 `internal/adapter/http/handlers.go`
   - 添加管理员管理handlers
   - 添加权限组管理handlers
   - 添加个人资料管理handlers
   - 添加密码找回handlers

2. 修改 `internal/adapter/http/dto.go`
   - 添加相关的DTO结构体

3. 修改 `internal/adapter/http/router.go`
   - 添加新的路由

### 第5步：中间件
1. 修改 `internal/adapter/http/middleware.go`
   - 创建 `RequirePermission` 中间件
   - 修改 `RequireAdmin` 中间件以支持权限验证

2. 应用权限中间件到各个管理API

### 第6步：邮件模板
1. 在 `internal/adapter/seed/seed.go` 中添加密码重置模板
   - `password_reset` 邮件模板

### 第7步：头像获取
1. 创建 `internal/pkg/avatar/qq_avatar.go`
   - 使用QQ头像API：`https://q1.qlogo.cn/g?b=qq&nk={qq}&s=100`
   - 实现头像URL生成函数

### 第8步：初始化数据
1. 在 `internal/adapter/seed/seed.go` 中添加默认权限组
   - 超级管理员
   - 运维管理员
   - 客服管理员
   - 财务管理员

2. 更新 `cmd/server/main.go`
   - 确保超级管理员拥有正确的权限组

### 第9步：测试和文档
1. 更新 `docs/api.md`
2. 更新 `docs/openapi.yaml`
3. 创建 `docs/multi-admin-guide.md`（多管理员管理指南）
4. 创建 `CHANGELOG.md`（更新记录）

## 七、关键文件清单

### 需要修改的文件
- `internal/adapter/repo/migrate.go`
- `internal/domain/models.go`
- `internal/adapter/repo/sqlite_repo.go`
- `internal/usecase/ports.go`
- `internal/usecase/admin_service.go`
- `internal/adapter/http/handlers.go`
- `internal/adapter/http/dto.go`
- `internal/adapter/http/router.go`
- `internal/adapter/http/middleware.go`
- `internal/adapter/seed/seed.go`
- `cmd/server/main.go`
- `docs/api.md`
- `docs/openapi.yaml`

### 需要新建的文件
- `internal/usecase/permission_service.go`
- `internal/usecase/password_reset_service.go`
- `internal/pkg/avatar/qq_avatar.go`
- `docs/multi-admin-guide.md`
- `CHANGELOG.md`

## 八、技术细节

### 8.1 JWT Token结构
```go
type Claims struct {
    UserID   int64    `json:"user_id"`
    Username string   `json:"username"`
    Role     string   `json:"role"`
    Permissions []string `json:"permissions,omitempty"` // 新增：权限列表
    jwt.RegisteredClaims
}
```

### 8.2 权限验证逻辑
1. 超级管理员（role="admin"且permission_group_id=1）：拥有所有权限
2. 普通管理员：根据权限组的permissions_json验证
3. 权限检查：使用通配符支持（如 "vps.*" 代表所有VPS相关权限）

### 8.3 密码重置流程
1. 用户请求密码重置（提供邮箱）
2. 生成随机token（24小时有效）
3. 发送包含token的邮件到用户邮箱
4. 用户点击邮件中的链接或输入token和新密码
5. 验证token有效性
6. 重置密码并标记token已使用

### 8.4 头像URL格式
```
https://q1.qlogo.cn/g?b=qq&nk={qq}&s=100
```
- `b=qq`：QQ头像
- `nk={qq}`：QQ号
- `s=100`：尺寸（100px）

## 九、安全考虑

1. 密码重置令牌：
   - 使用加密安全的随机数生成器
   - 设置合理的过期时间（24小时）
   - 使用后立即标记为已使用
   - 限制重试次数

2. 权限验证：
   - 每个敏感操作都需要权限验证
   - 超级管理员权限不可被删除
   - 至少保留一个超级管理员账户

3. 审计日志：
   - 记录所有管理员的敏感操作
   - 包括操作者、操作时间、操作详情、IP地址

## 十、前端适配要点

1. 新增管理员管理页面
2. 新增权限组管理页面
3. 管理员列表显示头像（使用返回的avatar URL）
4. 个人资料页面支持修改邮箱、QQ和密码
5. 密码找回页面
6. 权限不足时显示友好的错误提示

## 十一、时间估算

- 数据库设计和迁移：1-2小时
- Repository层实现：2-3小时
- Service层实现：3-4小时
- HTTP层实现：2-3小时
- 中间件和权限验证：2-3小时
- 邮件模板和初始化数据：1-2小时
- 测试：2-3小时
- 文档编写：2-3小时

**总计：约15-23小时**