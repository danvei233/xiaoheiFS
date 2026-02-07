# 核心业务逻辑文档

> 项目：Cloud VPS Console Backend
> 文档日期：2026-02-07
> 说明：本文档描述系统核心业务逻辑和分支流程

---

## 目录

1. [订单状态流转](#订单状态流转)
2. [购买流程](#购买流程)
3. [支付流程](#支付流程)
4. [订单审批流程](#订单审批流程)
5. [VPS 配置 (Provisioning)](#vps-配置-provisioning)
6. [续费流程](#续费流程)
7. [紧急续费流程](#紧急续费流程)
8. [升降配流程](#升降配流程)
9. [退款流程](#退款流程)
10. [钱包充值流程](#钱包充值流程)
11. [钱包提现流程](#钱包提现流程)
12. [钱包余额调整](#钱包余额调整)

---

## 订单状态流转

### 订单状态 (OrderStatus)

| 状态 | 说明 |
|------|------|
| `draft` | 草稿 |
| `pending_payment` | 待支付 |
| `pending_review` | 待审核 |
| `rejected` | 已拒绝 |
| `approved` | 已批准 |
| `provisioning` | 配置中 |
| `active` | 已激活 |
| `failed` | 失败 |
| `canceled` | 已取消 |

### 订单项状态 (OrderItemStatus)

| 状态 | 说明 |
|------|------|
| `pending_payment` | 待支付 |
| `pending_review` | 待审核 |
| `approved` | 已批准 |
| `provisioning` | 配置中 |
| `active` | 已激活 |
| `failed` | 失败 |
| `rejected` | 已拒绝 |
| `canceled` | 已取消 |

### VPS 状态 (VPSStatus)

| 状态 | 说明 |
|------|------|
| `provisioning` | 配置中 |
| `running` | 运行中 |
| `stopped` | 已停止 |
| `reinstalling` | 重装中 |
| `reinstall_failed` | 重装失败 |
| `expired_locked` | 到期锁定 |
| `rescue` | 救援模式 |
| `cracking_password` | 破解密码中 |
| `locked` | 已锁定 |
| `unknown` | 未知 |

---

## 购买流程

### 入口：`CreateOrderFromCart` 或 `CreateOrderFromItems`

### 主流程

```
用户请求创建订单
    ↓
检查实名认证要求 (如果启用)
    ↓
幂等性检查 (如果提供 idemKey)
    ↓
获取购物车商品 / 直接使用商品列表
    ↓
计算订单总金额
    ↓
生成订单号 (格式: ORD-{userID}-{timestamp})
    ↓
创建订单记录 (状态: pending_payment)
    ↓
拆分商品为订单项 (按 Qty 分拆)
    ↓
【原子操作分支】如果支持原子创建
    ├── 原子创建订单和订单项
    └── 清空购物车
【非原子分支】如果不支持原子创建
    ├── 创建订单
    ├── 创建订单项
    └── 清空购物车
    ↓
发布事件: order.pending_payment
    ↓
返回订单和订单项
```

### 价格计算逻辑

**位置**: `priceForPackage()` 函数 (`order_service.go:1510-1530`)

1. 获取套餐基础月价格 (`pkg.Monthly`)
2. 获取计划组的单价配置:
   - `UnitCore`: 每增加1核CPU的价格
   - `UnitMem`: 每增加1GB内存的价格
   - `UnitDisk`: 每增加1GB磁盘的价格
   - `UnitBW`: 每增加1Mbps带宽的价格
3. 计算附加项月价格:
   ```
   addonMonthly = AddCores * UnitCore
                + AddMemGB * UnitMem
                + AddDiskGB * UnitDisk
                + AddBWMbps * UnitBW
   ```
4. 解析计费周期 (BillingCycle):
   - 获取周期月数和乘数
   - 检查最小/最大数量限制
5. 计算总价:
   ```
   total = Round((baseMonthly + addonMonthly) * cycleMultiplier)
   ```

### 分支逻辑

#### 1. 幂等性处理
- 如果提供了 `idemKey` 且已存在相同订单，直接返回现有订单
- 防止重复提交

#### 2. 实名认证检查
- 如果启用实名认证 (`realname.RequireAction`)，检查用户是否已通过认证
- 认证类型: `purchase_vps`

#### 3. 购物车为空
- 返回 `ErrInvalidInput`

---

## 支付流程

### 入口：`SelectPayment` (支付方式选择) 或 `SubmitPayment` (手动提交支付凭证)

### 主流程 (选择支付方式)

```
用户选择支付方式
    ↓
验证订单归属和状态
    ↓
【分支 1】订单金额 <= 0
    └── 返回 no_payment_required
【分支 2】支付方式 = "approval"
    └── 返回 manual 状态，提示用户提交支付凭证
【分支 3】支付方式 = "custom"
    └── 返回自定义支付方式配置 (手动支付)
【分支 4】支付方式 = "balance"
    └── 钱包余额支付 (见下方详细流程)
【分支 5】其他支付方式
    └── 第三方支付 (见下方详细流程)
```

### 钱包余额支付流程

**位置**: `payWithBalance()` 函数 (`payment_service.go:264-303`)

```
检查钱包服务可用
    ↓
调用 AdjustWalletBalance 扣除余额
    ↓
生成交易流水号 (格式: BAL-{orderID}-{timestamp})
    ↓
创建支付记录 (状态: approved)
    ↓
【分支】确保订单进入待审核状态
    └── ensurePendingReview() 更新订单状态
    ↓
【分支】自动审批订单
    └── approver.ApproveOrder()
    ↓
发布事件: payment.approved
    ↓
返回支付结果 (包含钱包余额)
```

### 第三方支付流程

**位置**: `payWithProvider()` 函数 (`payment_service.go:306-357`)

```
获取支付提供者 (Provider)
    ↓
调用 Provider.CreatePayment() 创建支付
    ↓
生成交易流水号 (如果 Provider 未返回)
    ↓
创建支付记录 (状态: pending_payment)
    ↓
发布事件: payment.created
    ↓
返回支付 URL 和相关信息
```

### 手动提交支付凭证流程

**位置**: `SubmitPayment()` 函数 (`order_service.go:272-362`)

```
用户提交支付凭证
    ↓
验证订单归属和状态
    ↓
【幂等性检查 1】按 idemKey 去重
【幂等性检查 2】按 tradeNo 去重
    ↓
验证金额和币种
    ↓
创建支付记录 (状态: pending_review)
    ↓
更新订单状态为 pending_review
    ↓
更新所有订单项状态为 pending_review
    ↓
发布事件: order.pending_review
    ↓
【分支】发送机器人通知 (如果配置)
    └── robot.NotifyOrderPending()
    ↓
返回支付记录
```

### 支付回调处理流程

**位置**: `HandleNotify()` 函数 (`payment_service.go:162-244`)

```
接收支付平台回调
    ↓
调用 Provider.VerifyNotify() 验证签名和参数
    ↓
【分支 1】未支付
    └── 返回 ErrInvalidInput
    ↓
【订单关联逻辑】
    ├── 优先通过 OrderNo + Method 关联
    └── 回退到 TradeNo 查询
    ↓
【分支 2】未找到支付记录
    └── 返回 ErrInvalidInput
    ↓
更新 TradeNo (如果回调返回新的)
    ↓
【分支 3】支付状态已审批
    └── 跳过后续处理
    ↓
更新支付状态为 approved
    ↓
确保订单进入待审核状态
    ↓
【分支 4】自动审批订单
    └── approver.ApproveOrder() (错误被忽略!)
    ↓
发布事件: payment.confirmed
    ↓
返回确认响应给支付平台
```

---

## 订单审批流程

### 入口：`ApproveOrder`

### 主流程

```
管理员/系统请求审批订单
    ↓
验证订单状态 (必须是 pending_review/pending_payment/rejected)
    ↓
更新订单状态为 approved
    └── 设置审批人和审批时间
    ↓
遍历订单项分类处理
    ├── resize 类型 → 创建 ResizeTask
    └── 其他类型 → 直接更新状态为 approved
    ↓
【分支 1】处理升降配退款
    └── creditResizeRefundOnApprove()
    ↓
更新所有支付记录状态为 approved
    ↓
记录审计日志
    ↓
发布事件: order.approved
    ↓
发送审批通知邮件
    ↓
【分支 2】启动异步配置
    ├── 有升降配任务 → 立即执行非定时任务
    └── 无升降配任务 → 启动配置流程
    ↓
返回成功
```

### 升降配任务处理

**位置**: `ApproveOrder()` 函数中 (`order_service.go:523-597`)

```
检查是否有升降配订单项
    ↓
【分支】检查是否已存在待处理的升降配任务
    └── 返回 ErrResizeInProgress
    ↓
解析升降配规格
    ├── vps_id: 目标 VPS ID
    └── scheduled_at: 定时执行时间 (可选)
    ↓
创建 ResizeTask (状态: pending)
    ↓
【原子操作分支】如果支持原子审批
    └── ApproveResizeOrderWithTasks()
    ↓
【非原子分支】
    ├── 更新订单状态
    ├── 更新订单项状态
    └── 创建升降配任务
    ↓
【分支】立即执行非定时任务
    └── go executeResizeTask(context.Background(), task)
    ↓
【分支】同时配置其他订单项
    └── go provisionOrder(orderID)
```

### 升降配退款处理

**位置**: `creditResizeRefundOnApprove()` 函数 (`order_service.go:1338-1373`)

```
解析升降配规格
    ├── charge_amount: 需补差价
    └── refund_amount: 需退差价
    ↓
计算退款金额 (如果 charge_amount < 0)
    ↓
检查是否需要退款
    └── refund_amount > 0
    ↓
创建钱包退款订单并自动审批
    └── createAndApproveWalletRefund()
    ├── ref_type = "resize_refund"
    └── ref_id = order.ID
```

### 订单拒绝流程

**位置**: `RejectOrder()` 函数 (`order_service.go:646-680`)

```
管理员拒绝订单
    ↓
验证订单状态
    ↓
更新订单状态为 rejected
    └── 设置拒绝原因
    ↓
更新所有订单项状态为 rejected
    ↓
更新所有支付记录状态为 rejected
    ↓
记录审计日志
    ↓
发布事件: order.rejected
    ↓
发送拒绝通知邮件
```

---

## VPS 配置 (Provisioning)

### 入口：`provisionOrder()` 异步执行

### 主流程

```
启动配置流程 (异步)
    ↓
更新订单状态为 provisioning
    ↓
发布事件: order.provisioning
    ↓
遍历所有订单项
    ↓
【跳过已完成的项】active/rejected/canceled
    ↓
按订单类型处理
    ├── create → 新建 VPS
    ├── renew → 续费 VPS
    ├── emergency_renew → 紧急续费
    ├── resize → 升降配
    └── refund → 退款
    ↓
【状态聚合】确定最终订单状态
    ├── 有失败 → failed
    ├── 全部激活 → active
    └── 其他 → provisioning
    ↓
更新订单状态
    ↓
发布事件: order.completed
    ↓
【分支】发送激活/失败通知
```

### 新建 VPS 流程

**位置**: `provisionItem()` 函数 (`order_service.go:829-957`)

```
更新订单项状态为 provisioning
    ↓
获取套餐、计划组、镜像信息
    ↓
计算最终规格
    ├── CPU = pkg.Cores + spec.AddCores
    ├── 内存 = pkg.MemoryGB + spec.AddMemGB
    ├── 磁盘 = pkg.DiskGB + spec.AddDiskGB
    └── 带宽 = pkg.BandwidthMB + spec.AddBWMbps
    ↓
生成主机名 (格式: ecs-{userID}-{nanoTime})
    ↓
生成随机密码 (系统密码、VNC密码)
    ↓
计算到期时间 (当前时间 + 购买月数)
    ↓
调用 AutomationClient.CreateHost()
    ↓
【分支】获取 HostID
    ├── 返回值包含 HostID
    └── 回退到 ListHostSimple() 查询
    ↓
【分支】等待主机就绪
    └── waitHostActive() 最多30秒
    ↓
【分支 1】主机就绪
    ├── 获取主机信息 (IP、密码等)
    ├── 创建 VPSInstance 记录 (状态: running)
    ├── 更新订单项状态为 active
    └── 返回实例
    ↓
【分支 2】主机未就绪 (配置中)
    ├── 创建/更新 VPSInstance 记录 (状态: provisioning)
    ├── 保存已知信息 (密码、到期时间)
    ├── 入队重试任务 (ProvisionJob)
    └── 返回 ErrProvisioning
    ↓
【分支 3】配置失败
    └── 返回错误
```

### 续费 VPS 流程

**位置**: `handleRenew()` 函数 (`order_service.go:959-1002`)

```
更新订单项状态为 provisioning
    ↓
解析续费参数
    ├── vps_id: 目标 VPS
    ├── renew_days: 续费天数
    └── duration_months: 续费月数
    ↓
获取 VPS 实例
    ↓
计算新到期时间
    ├── 当前未到期 → 到期时间 + 续费天数
    └── 当前已到期 → 当前时间 + 续费天数
    ↓
调用 AutomationClient.RenewHost()
    ↓
【分支】解锁主机 (如果被锁定)
    └── UnlockHost()
    ↓
更新本地实例到期时间
    ↓
更新订单项状态为 active
```

### 紧急续费流程

**位置**: `handleEmergencyRenew()` 函数 (`order_service.go:1004-1060`)

```
更新订单项状态为 provisioning
    ↓
检查紧急续费策略
    ├── 是否启用
    ├── 时间窗口 (到期前 N 天内)
    └── 间隔限制 (两次紧急续费间隔 N 小时)
    ↓
【分支】不符合策略
    └── 返回 ErrForbidden
    ↓
【分支】上次紧急续费时间太近
    └── 返回 ErrConflict
    ↓
计算续费天数 (使用策略配置)
    ↓
计算新到期时间
    ↓
调用 AutomationClient.RenewHost()
    ↓
解锁主机 (如果被锁定)
    ↓
更新本地实例到期时间
    ↓
更新紧急续费时间戳
    ↓
更新订单项状态为 active
```

### 升降配流程

**位置**: `handleResize()` 函数 (`order_service.go:1062-1169`)

```
更新订单项状态为 provisioning
    ↓
解析升降配参数
    ├── vps_id: 目标 VPS
    ├── spec: 目标规格
    ├── target_package_id: 目标套餐
    ├── charge_amount: 补差价
    ├── refund_amount: 退差价
    └── refund_to_wallet: 退款到钱包
    ↓
获取 VPS 实例
    ↓
调用 AutomationClient.ElasticUpdate()
    ├── CPU
    ├── MemoryGB
    ├── DiskGB
    └── Bandwidth
    ↓
更新本地实例规格
    ├── 套餐信息
    ├── 月价格
    └── 附加规格
    ↓
【分支】处理退款到钱包
    └── AdjustWalletBalance() credit
    ↓
刷新主机信息
    ├── 状态
    ├── 到期时间
    ├── 访问信息
    └── 规格信息
    ↓
更新订单项状态为 active
```

### 退款流程 (配置阶段)

**位置**: `handleRefund()` 函数 (`order_service.go:1201-1238`)

```
更新订单项状态为 provisioning
    ↓
解析退款参数
    ├── vps_id: 目标 VPS
    ├── refund_amount: 退款金额
    └── delete_on_approve: 是否删除 VPS
    ↓
创建钱包退款订单并自动审批
    └── createAndApproveWalletRefund()
    ├── ref_type = "vps_refund"
    └── ref_id = order.ID
    ↓
【分支】删除 VPS
    └── deleteVPSForRefund()
    ├── 调用 AutomationClient.DeleteHost()
    └── 删除本地 VPSInstance 记录
```

---

## 续费流程

### 入口：`CreateRenewOrder`

### 主流程

```
用户请求续费 VPS
    ↓
检查实名认证要求 (如果启用)
    ↓
验证 VPS 归属
    ↓
【分支】检查是否有待处理的续费订单
    └── 返回 ErrConflict
    ↓
计算续费月数
    ├── 优先使用 duration_months
    └── 回退使用 renewDays / 30
    ↓
计算续费金额
    ├── 使用 VPS 实例的 MonthlyPrice
    └── 回退使用套餐的 Monthly 价格
    ↓
生成续费订单号 (格式: REN-{userID}-{timestamp})
    ↓
创建订单 (状态: pending_payment)
    ↓
创建续费订单项 (action: renew)
    └── spec_json 包含 vps_id, renew_days, duration_months
    ↓
发布事件: order.pending_payment
    ↓
返回订单
```

---

## 紧急续费流程

### 入口：`CreateEmergencyRenewOrder`

### 主流程

```
用户请求紧急续费
    ↓
检查紧急续费策略
    ├── 是否启用 (resize_enabled)
    └── 返回 ErrForbidden
    ↓
验证 VPS 归属
    ↓
【时间窗口检查】是否在到期前 N 天内
    └── 返回 ErrForbidden
    ↓
【间隔检查】上次紧急续费是否超过 N 小时
    └── 返回 ErrConflict
    ↓
【分支】检查是否有待处理的续费订单
    └── 返回 ErrConflict
    ↓
创建紧急续费订单 (状态: pending_review)
    ├── TotalAmount = 0 (免费)
    └── action: emergency_renew
    ↓
发布事件: order.pending_review
    ↓
【自动审批】立即调用 ApproveOrder
    ↓
返回已审批的订单
```

### 紧急续费策略配置

**位置**: `loadEmergencyRenewPolicy()`

- `enabled`: 是否启用
- `window_days`: 时间窗口 (到期前几天)
- `renew_days`: 续费天数
- `interval_hours`: 两次紧急续费最小间隔

---

## 升降配流程

### 入口：`CreateResizeOrder`

### 主流程

```
用户请求升降配
    ↓
检查实名认证要求 (如果启用)
    ↓
检查升降配功能是否启用
    └── 返回 ErrResizeDisabled
    ↓
【定时升降配】检查是否启用定时升降配
    └── 返回 ErrForbidden
    ↓
验证 VPS 归属
    ↓
【分支 1】检查是否有待处理的升降配订单
    └── 返回 ErrResizeInProgress
    ↓
【分支 2】检查是否有待执行的升降配任务
    └── 返回 ErrResizeInProgress
    ↓
计算升降配报价
    ├── 补差价金额
    └── 退款金额
    ↓
生成升降配订单号 (格式: UPG-{userID}-{timestamp})
    ↓
确定订单状态
    ├── 需补差价 → pending_payment
    └── 需退款 → pending_review
    ↓
创建升降配订单
    └── action: resize
    ↓
【分支】如果是退款类型，自动审批
    └── 调用 ApproveOrder
    ↓
返回订单和报价
```

### 升降配报价计算

**位置**: `quoteResize()` 函数

```
获取当前 VPS 规格
    ↓
确定目标规格
    ├── 使用 targetPackageId 获取套餐
    └── 或使用 spec 参数计算
    ↓
计算价格差
    ├── 当前价格 vs 目标价格
    └── 按比例计算剩余周期差价
    ↓
确定补差价或退款
    ├── 目标价格高 → charge_amount
    └── 目标价格低 → refund_amount
    ↓
返回报价
```

### 定时升降配执行

**位置**: `executeResizeTask()` 函数 (`order_service.go:1375-1416`)

```
任务调度器触发执行
    ↓
更新任务状态为 running
    ↓
获取订单项
    ↓
更新订单项状态为 provisioning
    ↓
调用 handleResize() 执行升降配
    ↓
【分支 1】执行成功
    ├── 更新订单项状态为 active
    ├── 更新任务状态为 done
    └── 刷新订单状态
    ↓
【分支 2】执行失败
    ├── 更新订单项状态为 failed
    ├── 更新任务状态为 failed
    └── 刷新订单状态
```

---

## 退款流程

### 入口：`CreateRefundOrder` 或 `RequestRefund`

### 主流程

```
用户请求退款
    ↓
验证 VPS 归属
    ↓
获取原始订单项金额
    ↓
加载退款策略
    ├── 全额退款时间
    ├── 按比例退款时间
    ├── 不退款时间
    └── 是否需要审批
    ↓
计算退款金额
    ├── 全额退款 period < full_hours/days
    ├── 按比例退款 period < prorate_hours/days
    └── 不退款 period >= no_refund_hours/days
    ↓
【分支】退款金额 <= 0
    └── 返回 ErrForbidden
    ↓
生成退款订单号 (格式: REF-{userID}-{timestamp})
    ↓
创建退款订单 (状态: pending_review)
    ├── TotalAmount = -refundAmount
    └── action: refund
    ↓
创建退款订单项
    └── spec_json 包含退款详情
    ↓
发布事件: order.pending_review
    ↓
返回订单和退款金额
```

### 退款金额计算

**位置**: `calculateRefundAmountForAmount()`

```
计算已使用时间比例
    ├── elapsed = now - created
    └── total = expire - created
    ↓
确定退款策略
    ├── period < full_hours/days → 100% 退款
    ├── period < prorate_hours/days → 按比例退款
    └── period >= no_refund_hours/days → 0% 退款
    ↓
计算退款金额
    └── amount * (1 - elapsed_ratio) * curve_factor
    ↓
返回退款金额
```

### 退款订单审批

**位置**: `provisionOrder` 中的 `handleRefund()`

```
订单审批通过后执行配置
    ↓
检测到 action = refund 的订单项
    ↓
创建钱包退款订单并自动审批
    ├── ref_type = "vps_refund"
    └── ref_id = 原订单 ID
    ↓
【分支】delete_on_approve = true
    └── 删除 VPS 实例
    ├── 调用 AutomationClient.DeleteHost()
    └── 删除本地 VPSInstance 记录
    ↓
更新订单项状态为 active
```

---

## 钱包充值流程

### 入口：`CreateRecharge`

### 主流程

```
用户请求充值
    ↓
验证输入参数
    ├── userID > 0
    └── amount > 0
    ↓
创建充值订单 (状态: pending_review)
    ├── type = recharge
    ├── amount = 充值金额
    └── currency = CNY
    ↓
返回充值订单
```

### 充值订单审批

**位置**: `WalletOrderService.Approve()`

```
管理员审批充值订单
    ↓
验证订单状态 (必须是 pending_review)
    ↓
调用 approveOrder() 执行审批
    ↓
【幂等性检查】检查是否已有交易记录
    ├── 存在 → 获取当前钱包余额
    └── 不存在 → 调整余额
    ↓
【调整余额】AdjustWalletBalance()
    ├── amount = +充值金额
    ├── type = credit
    ├── ref_type = wallet_order
    └── ref_id = 充值订单 ID
    ↓
更新订单状态为 approved
    ↓
记录审计日志
    ↓
返回钱包信息
```

### 拒绝充值

**位置**: `WalletOrderService.Reject()`

```
管理员拒绝充值订单
    ↓
验证订单状态
    ↓
更新订单状态为 rejected
    ├── 设置拒绝原因
    └── 记录审批人
    ↓
记录审计日志
```

---

## 钱包提现流程

### 入口：`CreateWithdraw`

### 主流程

```
用户请求提现
    ↓
验证输入参数
    ├── userID > 0
    └── amount > 0
    ↓
获取用户钱包余额
    ↓
【分支】余额不足检查
    └── 返回 ErrInsufficientBalance
    ↓
创建提现订单 (状态: pending_review)
    ├── type = withdraw
    ├── amount = 提现金额
    └── currency = CNY
    ↓
返回提现订单
```

### 提现订单审批

**位置**: `WalletOrderService.Approve()`

```
管理员审批提现订单
    ↓
验证订单状态
    ↓
调用 approveOrder() 执行审批
    ↓
【幂等性检查】检查是否已有交易记录
    ├── 存在 → 获取当前钱包余额
    └── 不存在 → 调整余额
    ↓
【调整余额】AdjustWalletBalance()
    ├── amount = -提现金额 (负数)
    ├── type = debit
    ├── ref_type = wallet_order
    └── ref_id = 提现订单 ID
    ↓
更新订单状态为 approved
    ↓
记录审计日志
    ↓
返回钱包信息
```

---

## 钱包余额调整

### 入口：`AdjustBalance` (管理员操作)

### 主流程

```
管理员调整用户余额
    ↓
调用 AdjustWalletBalance()
    ├── amount = 调整金额 (正数增加，负数减少)
    ├── type = credit/debit
    ├── ref_type = admin_adjust
    └── ref_id = adminID
    ↓
记录审计日志
    └── action = wallet.adjust
    ↓
返回钱包信息
```

### 底层余额调整逻辑

**位置**: `AdjustWalletBalance()` 函数 (`sqlite_repo.go:2724-2762`)

```
开始数据库事务
    ↓
查询当前钱包余额
    ↓
【分支】钱包不存在
    ├── 创建钱包 (余额 = 0)
    └── 使用余额 = 0 继续
    ↓
计算新余额 = 当前余额 + amount
    ↓
【分支】余额不足检查
    └── newBalance < 0 → 返回 ErrInsufficientBalance
    ↓
UPDATE 钱包余额
    ↓
INSERT 交易记录
    ├── user_id
    ├── amount (正负数)
    ├── type (credit/debit)
    ├── ref_type (关联类型)
    ├── ref_id (关联ID)
    └── note (备注)
    ↓
提交事务
    ↓
返回钱包信息
```

---

## 钱包订单退款 (自动审批)

### 入口：`createAndApproveWalletRefund`

**位置**: `order_service.go:1258-1315`

### 主流程

```
创建退款订单
    ↓
【幂等性检查】检查是否已有交易记录
    ├── ref_type = vps_refund / resize_refund
    └── ref_id = 原订单 ID
    ↓
【分支】已存在交易记录
    └── promoteRefundWalletOrderIfNeeded()
    └── 返回成功 (幂等)
    ↓
创建钱包订单 (状态: pending_review)
    ├── type = refund
    └── amount = 退款金额
    ↓
调整钱包余额
    ├── amount = +退款金额
    ├── type = credit
    ├── ref_type = 订单类型
    └── ref_id = 订单 ID
    ↓
【分支】余额调整失败但已存在交易
    └── 再次检查并确认幂等
    ↓
更新钱包订单状态为 approved
    ↓
记录审计日志
    ↓
返回成功
```

---

## 附录：订单项类型 (Action)

| Action | 说明 | 主要处理函数 |
|--------|------|-------------|
| `create` | 新建 VPS | `provisionItem()` |
| `renew` | 续费 VPS | `handleRenew()` |
| `emergency_renew` | 紧急续费 | `handleEmergencyRenew()` |
| `resize` | 升降配 | `handleResize()` / `executeResizeTask()` |
| `refund` | 退款 | `handleRefund()` |

---

## 附录：钱包订单类型

| 类型 | 说明 | 余额变化 |
|------|------|---------|
| `recharge` | 充值 | 增加 (credit) |
| `withdraw` | 提现 | 减少 (debit) |
| `refund` | 退款 | 增加 (credit) |

---

## 附录：支付状态

| 状态 | 说明 |
|------|------|
| `pending_payment` | 待支付 (第三方支付初始状态) |
| `pending_review` | 待审核 (手动支付初始状态) |
| `approved` | 已支付 |
| `rejected` | 已拒绝 |

---

*文档结束*

> 代码位置参考:
> - `internal/usecase/order_service.go` - 订单服务
> - `internal/usecase/payment_service.go` - 支付服务
> - `internal/usecase/wallet_service.go` - 钱包服务
> - `internal/usecase/wallet_order_service.go` - 钱包订单服务
> - `internal/adapter/repo/sqlite_repo.go` - 数据库操作
