# 代码问题排查文档

生成时间：2026-01-27
项目：Cloud VPS Console
版本：main 分支

## 文档说明

本文档记录了代码审查中发现的所有潜在问题，按严重程度和功能模块分类。每个问题包含：
- 问题描述
- 代码位置
- 严重程度
- 潜在影响
- 建议修复

---

## 目录

1. [退款功能问题](#1-退款功能问题)
2. [订单管理问题](#2-订单管理问题)
3. [VPS 管理问题](#3-vps-管理问题)
4. [钱包功能问题](#4-钱包功能问题)
5. [认证权限安全问题](#5-认证权限安全问题)
6. [CMS 功能问题](#6-cms-功能问题)
7. [数据库迁移问题](#7-数据库迁移问题)

---

## 1. 退款功能问题

### 1.1 退款曲线最后一点不强制为 0 [中等]
**位置**: `internal/usecase/refund_curve.go:84-85`

**问题描述**:
```go
return clamp01(points[len(points)-1].Ratio), true
```

如果退款曲线最后一点的 `ratio` 不是 0（例如 0.1），超过 100% 生命周期后仍然能获得部分退款。

**示例配置**:
```json
[{"hours": 0, "ratio": 1}, {"hours": 100, "ratio": 0.1}]
```

**建议修复**: 当 `elapsedRatio >= 100` 时强制返回 0。

---

### 1.2 VPS 删除失败但钱已退 [中等]
**位置**: `internal/usecase/wallet_order_service.go:294-299`

**问题描述**:
```go
if deleteOnApprove(order.MetaJSON) {
    if err := s.deleteVPS(ctx, order.MetaJSON); err != nil {
        return domain.Wallet{}, err  // 返回错误，但钱已经加了
    }
}
```

如果 `AdjustWalletBalance` 成功但 `deleteVPS` 失败，会导致：
- 钱已退到钱包
- VPS 仍然存在
- 订单状态保持 `pending_review`

---

### 1.3 退款曲线单位语义混淆 [低]
**位置**: `internal/usecase/refund_curve.go:62`

**问题描述**:
参数名为 `ageHours` 但实际传入的是百分比（0-100），字段名 `hours` 误导配置者。

---

## 2. 订单管理问题

### 2.1 CreateOrderFromCart 缺少事务保护 [严重]
**位置**: `internal/usecase/order_service.go:64-142`

**问题描述**:
订单创建过程涉及多个数据库操作（创建订单、创建订单项、清空购物车），但没有事务保护。

**潜在影响**:
- 订单创建成功但订单项创建失败，导致订单无明细
- 订单项创建成功但购物车未清空，导致重复下单

---

### 2.2 ApproveOrder 允许从 rejected 状态转换 [中等]
**位置**: `internal/usecase/order_service.go:464-467`

**问题描述**:
```go
if order.Status != domain.OrderStatusPendingReview &&
    order.Status != domain.OrderStatusPendingPayment &&
    order.Status != domain.OrderStatusRejected {  // 允许从 rejected 直接 approved
    return ErrConflict
}
```

被拒绝的订单应该创建新订单而不是直接审批。

---

### 2.3 订单项数量分配逻辑错误 [中等]
**位置**: `internal/usecase/order_service.go:107-110`

**问题描述**:
```go
for i := 0; i < item.Qty; i++ {
    amount := unitAmount
    if int64(i) < remainder {  // 余数分配逻辑
        amount++
    }
```

如果 `remainder` 是大数值可能导致错误分配。

---

### 2.4 SubmitPayment 金额验证不完整 [中等]
**位置**: `internal/usecase/order_service.go:265-267`

**问题描述**:
只验证 `input.Amount > 0`，但没有验证支付金额是否与订单金额匹配。

```go
if input.Amount <= 0 {
    return domain.OrderPayment{}, ErrInvalidInput
}
// 缺少: if input.Amount != order.TotalAmount { return error }
```

---

### 2.5 MarkPaid 没有幂等性保护 [中等]
**位置**: `internal/usecase/order_service.go:349-398`

**问题描述**:
管理员标记已支付时没有 idemkey 幂等性保护（`SubmitPayment` 有），可能导致重复标记。

---

### 2.6 provisionOrder 错误被静默忽略 [低]
**位置**: `internal/usecase/order_service.go:676, 691, 697, 707, 712, 722`

**问题描述**:
多处使用 `_ = s.items.UpdateOrderItemStatus()` 忽略错误，可能导致状态不一致。

---

### 2.7 goroutine 可能泄漏 [低]
**位置**: `internal/usecase/order_service.go:568, 572, 576`

**问题描述**:
```go
go s.executeResizeTask(context.Background(), *task)
```

使用 `context.Background()` 创建新的 goroutine，无法取消或超时控制。

---

## 3. VPS 管理问题

### 3.1 CreateVPS 缺少事务保护 [严重]
**位置**: `internal/usecase/admin_vps_service.go:92-167`

**问题描述**:
如果自动化平台创建主机成功但数据库保存失败，会导致资源泄漏（孤立实例）。

---

### 3.2 DeleteVPS 删除不原子 [严重]
**位置**: `internal/usecase/admin_vps_service.go:316-336`

**问题描述**:
```go
if err := s.automation.DeleteHost(ctx, hostID); err != nil {
    return err
}
_ = s.vps.DeleteInstance(ctx, id)  // 错误被忽略
```

数据库删除错误被忽略，可能产生幽灵记录。

---

### 3.3 Resize 任务 worker 错误被忽略 [严重]
**位置**: `internal/usecase/resize_task_worker.go:5-21`

**问题描述**:
```go
func (w *ResizeTaskWorker) Execute(ctx context.Context, taskID int64) error {
    // ...
    _ = w.svc.ExecuteResizeTask(ctx, taskID)  // 错误被忽略
    return nil
}
```

Resize 失败不会被察觉。

---

### 3.4 RenewNow 解锁即使续费失败 [高]
**位置**: `internal/usecase/vps_service.go:111-130`

**问题描述**:
即使续费失败，主机也会被解锁。

---

### 3.5 ResetOS 状态先更新 [高]
**位置**: `internal/usecase/vps_service.go:156-189`

**问题描述**:
状态在自动化调用完成前就更新为 "reinstalling"，如果重装失败状态不会回滚。

---

### 3.6 automation 状态映射不完整 [中等]
**位置**: `internal/usecase/automation_state.go:5-22`

**问题描述**:
状态 6, 7, 8, 9, 11, 12 没有映射，会显示为 "unknown"。

---

### 3.7 RefreshStatus 更新不原子 [中等]
**位置**: `internal/usecase/vps_service.go:62-88`

**问题描述**:
多个更新调用，不是原子操作，部分失败可能发生。

---

## 4. 钱包功能问题

### 4.1 提现流程竞态条件 [严重]
**位置**: `internal/usecase/wallet_order_service.go:96-127`

**问题描述**:
余额检查和订单创建之间存在竞态窗口，可能导致超额提现。

```go
if wallet.Balance < input.Amount {
    return domain.WalletOrder{}, ErrInsufficientBalance
}
// ... 竞态窗口 ...
order := domain.WalletOrder{...}
```

**建议修复**: 使用数据库乐观锁或 "冻结金额" 机制。

---

### 4.2 Approve 缺少幂等性保护 [严重]
**位置**: `internal/usecase/wallet_order_service.go:250-264`

**问题描述**:
```go
order, err := s.orders.GetWalletOrder(ctx, orderID)
if order.Status != domain.OrderStatusPendingReview {
    return domain.WalletOrder{}, nil, ErrConflict
}
// ... 调整余额 ...
// 没有 WHERE status = ? 条件
```

并发审批可能导致同一订单被重复处理。

**建议修复**: 使用 CAS 操作：
```sql
UPDATE wallet_orders
SET status = ?, reviewed_by = ?
WHERE id = ? AND status = 'pending_review'
```

---

### 4.3 自动退款事务一致性问题 [中等]
**位置**: `internal/usecase/wallet_order_service.go:183-189`

**问题描述**:
```go
if err := s.orders.CreateWalletOrder(ctx, &order); err != nil {
    return domain.WalletOrder{}, nil, err
}
if status == domain.WalletOrderApproved {
    wallet, err := s.approveOrder(ctx, 0, order, true)  // 如果失败，订单已创建
```

如果 `approveOrder` 失败，订单已创建但状态不是 `approved`。

---

### 4.4 缺少钱包交易幂等性检查 [中等]
**位置**: `internal/adapter/repo/sqlite_repo.go:2289`

**问题描述**:
`HasWalletTransaction` 方法存在但从未被使用，可能导致重复交易。

---

### 4.5 VPS 删除和余额调整不原子 [中等]
**位置**: `internal/usecase/wallet_order_service.go:294-316`

**问题描述**:
VPS 删除成功但余额调整失败，用户损失资金，且无补偿机制。

---

## 5. 认证权限安全问题

### 5.1 JWT 算法未验证 [高]
**位置**: `internal/adapter/http/middleware.go:214-216`

**问题描述**:
```go
token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
    return m.jwtSecret, nil  // 没有验证 token.Method
})
```

可能受到算法混淆攻击（none 算法）。

**建议修复**:
```go
if token.Method != jwt.SigningMethodHS256 {
    return nil, errors.New("unexpected signing method")
}
```

---

### 5.2 JWT 签名错误被忽略 [高]
**位置**: `internal/adapter/http/handlers.go:194, 211, 1776, 2027`

**问题描述**:
```go
signed, _ := token.SignedString(h.jwtSecret)
```

签名失败会返回无效 token 但不记录错误。

---

### 5.3 Token 通过查询参数传递 [中等]
**位置**: `internal/adapter/http/middleware.go:223-236`

**问题描述**:
允许通过 URL 查询参数传递 token，可能被记录在访问日志中。

---

### 5.4 忘记密码缺少速率限制 [中等]
**位置**: `internal/adapter/http/handlers.go:4326-4343`

**问题描述**:
`AdminForgotPassword` 端点没有速率限制，可被用于枚举邮箱或垃圾邮件轰炸。

---

### 5.5 用户模拟缺少审计 [中等]
**位置**: `internal/adapter/http/handlers.go:2007-2029`

**问题描述**:
`AdminUserImpersonate` 功能没有审计日志记录。

---

### 5.6 ParseInt 错误被忽略 [中等]
**位置**: `internal/adapter/http/handlers.go` 多处（40+ 处）

**问题描述**:
```go
id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
```

如果 ID 参数无效，会导致 `id = 0`，可能访问错误数据。

---

## 6. CMS 功能问题

### 6.1 XSS 防护不完整 [高]
**位置**: `internal/adapter/http/handlers.go:4999-5002`

**问题描述**:
```go
func containsDisallowedHTML(raw string) bool {
    lower := strings.ToLower(raw)
    return strings.Contains(lower, "<script") || strings.Contains(lower, "<iframe")
}
```

仅检查 `<script` 和 `<iframe`，可被绕过：
- `<img src=x onerror=alert(1)>`
- `<svg onload=alert(1)>`
- `<a href="javascript:alert(1)">`

**建议修复**: 使用专门的 HTML 清理库（如 `bluemonday`）。

---

### 6.2 文件上传缺少类型验证 [中等]
**位置**: `internal/adapter/http/handlers.go:4950-4983`

**问题描述**:
仅依赖 `Content-Type` header，没有验证实际文件类型或扩展名白名单。可能上传恶意文件（.php, .jsp 等）。

---

### 6.3 文件访问控制缺失 [中等]
**位置**: `internal/adapter/http/router.go:16`

**问题描述**:
```go
r.Static("/uploads", "./uploads")
```

所有上传文件公开访问，无需认证。敏感文件可能被公开访问。

---

### 6.4 Slug 唯一性检查不完整 [低]
**位置**: `handlers.go:4697-4704`

**问题描述**:
创建文章时只检查 slug 非空，没有检查唯一性。

---

## 7. 数据库迁移问题

### 7.1 外键约束缺少 ON DELETE 子句 [高]
**位置**: `internal/adapter/repo/migrate.go` 所有外键定义

**问题描述**:
所有外键约束都没有定义级联删除或更新行为。

**影响**:
- 删除父表记录时可能导致外键约束错误
- 无法自动清理关联数据，可能产生孤立记录

**建议**: 根据业务需求添加 `ON DELETE RESTRICT` 或 `ON DELETE CASCADE`。

---

### 7.2 迁移错误被忽略 [高]
**位置**: `internal/adapter/repo/migrate.go:517-574`

**问题描述**:
```go
_ = addColumnIfMissing(db, "users", "phone", "TEXT")
_ = addColumnIfMissing(db, "users", "bio", "TEXT")
// ... 50+ 个类似的调用
```

迁移失败时不会报告错误，可能导致数据库状态不一致。

---

### 7.3 EnsureCMSDefaults 删除所有数据 [中等]
**位置**: `internal/adapter/seed/seed.go:378-381`

**问题描述**:
```go
if _, err := db.Exec(`DELETE FROM cms_blocks`); err != nil {
    return err
}
```

每次调用都删除所有 CMS blocks，用户自定义数据会丢失。

---

### 7.4 种子数据错误被忽略 [中等]
**位置**: `internal/adapter/seed/seed.go:219-249`

**问题描述**:
```go
_, _ = tx.Exec(`INSERT INTO email_templates...`)
_, _ = tx.Exec(`INSERT INTO permission_groups...`)
```

15 处数据库操作错误被忽略，种子数据可能不完整。

---

### 7.5 硬编码敏感信息 [中等]
**位置**: `internal/adapter/seed/seed.go:302-303`

**问题描述**:
```go
"automation_api_key": "zPVhku8TueXcQbTcsdcu",
```

API 密钥硬编码在代码中。

---

### 7.6 缺少复合索引 [低]
**位置**: `internal/adapter/repo/migrate.go`

**问题描述**:
查询通常需要按 `user_id + status` 筛选，但缺少复合索引。

**建议**: 添加 `CREATE INDEX idx_orders_user_status ON orders(user_id, status)`

---

### 7.7 NOT NULL 列没有默认值 [低]
**位置**: `migrate.go` 多处

**问题描述**:
```go
line_id INTEGER NOT NULL,  -- 没有默认值
system_image_id INTEGER NOT NULL, -- 没有默认值
```

插入数据时必须显式提供值。

---

## 严重程度汇总

### 严重 (Critical) - 需要立即修复
| # | 问题 | 模块 |
|---|------|------|
| 1 | CreateOrderFromCart 缺少事务保护 | 订单 |
| 2 | CreateVPS 缺少事务保护 | VPS |
| 3 | DeleteVPS 删除不原子 | VPS |
| 4 | Resize 任务 worker 错误被忽略 | VPS |
| 5 | 提现流程竞态条件 | 钱包 |
| 6 | Approve 缺少幂等性保护 | 钱包 |
| 7 | JWT 算法未验证 | 认证 |
| 8 | 外键约束缺少 ON DELETE 子句 | 数据库 |
| 9 | 迁移错误被忽略 | 数据库 |

### 高 (High) - 尽快修复
| # | 问题 | 模块 |
|---|------|------|
| 1 | RenewNow 解锁即使续费失败 | VPS |
| 2 | ResetOS 状态先更新 | VPS |
| 3 | XSS 防护不完整 | CMS |
| 4 | JWT 签名错误被忽略 | 认证 |

### 中等 (Medium) - 计划修复
| # | 问题 | 模块 |
|---|------|------|
| 1 | ApproveOrder 状态转换过于宽松 | 订单 |
| 2 | 订单项数量分配逻辑错误 | 订单 |
| 3 | SubmitPayment 金额验证不完整 | 订单 |
| 4 | MarkPaid 没有幂等性保护 | 订单 |
| 5 | VPS 删除失败但钱已退 | 退款 |
| 6 | automation 状态映射不完整 | VPS |
| 7 | 自动退款事务一致性问题 | 钱包 |
| 8 | 缺少钱包交易幂等性检查 | 钱包 |
| 9 | VPS 删除和余额调整不原子 | 钱包 |
| 10 | Token 通过查询参数传递 | 认证 |
| 11 | 忘记密码缺少速率限制 | 认证 |
| 12 | 用户模拟缺少审计 | 认证 |
| 13 | ParseInt 错误被忽略 | 认证 |
| 14 | 文件上传缺少类型验证 | CMS |
| 15 | 文件访问控制缺失 | CMS |
| 16 | EnsureCMSDefaults 删除所有数据 | 数据库 |
| 17 | 种子数据错误被忽略 | 数据库 |
| 18 | 硬编码敏感信息 | 数据库 |

### 低 (Low) - 逐步改进
| # | 问题 | 模块 |
|---|------|------|
| 1 | 退款曲线单位语义混淆 | 退款 |
| 2 | provisionOrder 错误被静默忽略 | 订单 |
| 3 | goroutine 可能泄漏 | 订单 |
| 4 | RefreshStatus 更新不原子 | VPS |
| 5 | Slug 唯一性检查不完整 | CMS |
| 6 | 缺少复合索引 | 数据库 |
| 7 | NOT NULL 列没有默认值 | 数据库 |

---

## 修复优先级建议

### 第一阶段（立即）
1. 修复所有严重级别的竞态条件和事务一致性问题
2. 添加 JWT 算法验证
3. 修复外键约束和迁移错误处理

### 第二阶段（1-2 周）
1. 增强所有关键操作的幂等性保护
2. 添加 XSS 防护和文件上传验证
3. 实现审计日志记录

### 第三阶段（1 个月）
1. 改进错误处理，不再静默忽略错误
2. 添加必要的复合索引
3. 完善状态映射和边界条件处理

---

## 附录：代码审查检查清单

- [ ] 所有多步操作都在事务中
- [ ] 所有关键操作都有幂等性保护
- [ ] 所有外部 API 调用都有超时控制
- [ ] 所有错误都被正确处理和记录
- [ ] 所有用户新入都有验证和清理
- [ ] 所有关键操作都有审计日志
- [ ] 所有文件上传都有类型验证
- [ ] 所有状态转换都有验证
- [ ] 所有外键都有级联规则
- [ ] 所有敏感配置都不硬编码
