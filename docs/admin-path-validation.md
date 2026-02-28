# 管理端登录路径验证功能

## 概述

管理端登录API (`POST /admin/api/v1/auth/login`) 支持 `admin_path` 参数，用于验证前端访问路径的合法性。

## 使用方法

### 请求示例

```json
{
  "username": "admin",
  "password": "password123",
  "admin_path": "admin"
}
```

### 验证逻辑

1. **格式验证**：路径只能包含字母和数字，不能包含特殊字符
2. **保留字检查**：不能使用系统保留路径（如 login, api, console 等）
3. **路径匹配**：必须与系统配置的管理路径一致

### 错误响应

- `400 Bad Request` - 路径格式无效或使用了保留字
- `403 Forbidden` - 路径与系统配置不匹配

## 相关API

### 检查管理路径
```
POST /api/v1/check-admin-path
```

请求：
```json
{
  "path": "admin"
}
```

响应：
```json
{
  "is_admin": true,
  "admin_path": "admin"
}
```

## 配置管理路径

通过设置 `admin_path` 配置项来自定义管理路径：

```bash
# 通过API设置
curl -X PATCH http://localhost:8080/admin/api/v1/settings \
  -H "Authorization: Bearer TOKEN" \
  -d '{"key": "admin_path", "value": "mySecureAdmin"}'
```

## 前端集成

前端已在 `Login.vue` 中完成适配，会自动从路由中获取当前路径并传递给登录API。

### 自动获取路径
```javascript
const getCurrentAdminPath = () => {
  const pathSegments = route.path.split("/").filter(Boolean);
  return pathSegments[0] || "admin";
};
```

### 登录请求
```javascript
const adminPath = getCurrentAdminPath();
await admin.login({
  username,
  password,
  admin_path: adminPath
});
```

### 错误处理
前端会自动识别路径验证错误并显示友好提示：
```javascript
// 处理路径验证相关错误
if (errorMsg?.includes("admin path")) {
  message.error("管理路径验证失败，请检查访问地址");
} else if (error?.response?.status === 403) {
  message.error("访问被拒绝，请确认管理路径正确");
}
```

## 相关文件

### 后端
- **登录实现**：`backend/internal/adapter/http/handlers_admin_users_robot.go` (第157-177行)
- **路径验证**：`backend/internal/adapter/http/admin_path_validator.go`
- **检查API**：`backend/internal/adapter/http/install.go` (CheckAdminPath函数)

### 前端
- **登录页面**：`frontend/src/pages/admin/Login.vue` (第58-75行)
- **认证Store**：`frontend/src/stores/adminAuth.ts`
- **API服务**：`frontend/src/services/admin.ts`

## 测试

### 测试正常登录
```bash
curl -X POST http://localhost:8080/admin/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "password123",
    "admin_path": "admin"
  }'
```

### 测试路径不匹配
```bash
curl -X POST http://localhost:8080/admin/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "password123",
    "admin_path": "wrongpath"
  }'
```

预期响应：
```json
{
  "error": "admin path mismatch"
}
```

### 测试无效路径格式
```bash
curl -X POST http://localhost:8080/admin/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "password123",
    "admin_path": "admin-path!"
  }'
```

预期响应：
```json
{
  "error": "admin path invalid"
}
```

## 安全建议

1. **自定义路径**：不要使用默认的 `admin` 路径，使用随机生成的路径提高安全性
2. **定期更换**：定期更换管理路径，降低被攻击风险
3. **监控日志**：记录所有路径验证失败的尝试
4. **IP白名单**：结合IP白名单进一步限制访问
