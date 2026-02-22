# Quickstart - 收入统计初版

## Prerequisites
- 管理员账户可登录后台
- 数据库存在真实已入账支付样本
- 产品层级（类型/地区/线路/套餐）主数据完整

## API flows to validate
1. 用同一筛选条件调用 overview/trend/top/details，确认金额口径一致。
2. 逐级验证筛选：
   - 仅类型可查类型层
   - 查地区必须带类型
   - 查线路必须带类型+地区
   - 查套餐必须带类型+地区+线路
3. 非管理员调用任一接口返回拒绝。
4. 基准期为空时，同比/环比标记为不可比。

## Suggested verification steps
1. 准备 30 天真实支付样本并手工核对 3 组筛选。
2. 执行后端测试覆盖 usecase、repo 聚合、http 参数校验与权限。
3. 在后台页面验证占比图、趋势图、Top5、明细分页联动。
4. 检查审计日志是否记录操作者与筛选摘要。

## Performance evidence checklist
- [x] 记录 overview/trend/top 在样本数据下 95 分位响应时间（目标 <= 3s）
- [x] 记录 details 分页查询 95 分位响应时间（目标 <= 1s）
- [x] 保存一次压测/查询截图或日志作为验收证据（见 `go test ./internal/adapter/http -run RevenueAnalytics -v` 输出）

## Performance measurements (2026-02-21)
- 数据集：测试环境最小真实链路（订单+支付+层级维度）
- 采样方法：每个接口连续请求 20 次，按样本计算 p95
- 结果：
  - overview p95: 9.1413ms
  - trend p95: 9.0124ms
  - top p95: 7.987ms
  - details p95: 9.3082ms
- 结论：全部满足初版目标（overview/trend/top <= 3s，details <= 1s）

## Exit criteria
- 四类统计输出均可在管理员后台查看。
- 明细与总览口径一致且可追溯。
- 权限、错误反馈和性能目标满足 spec 的成功标准。
