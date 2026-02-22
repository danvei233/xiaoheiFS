# Phase 0 Research - 收入统计初版

## Decision 1: 时间范围与粒度策略
- Decision: 接口入参使用 `from` + `to`（RFC3339 date-time）与 `granularity`（day/week/month），并限制最大跨度 366 天。
- Rationale: 兼顾“自由选择时间跨度”与查询性能，避免超长跨度拖垮聚合查询。
- Alternatives considered: 仅支持固定预设范围（7/30/90天）；被拒绝，因为不满足“可自由选择”。

## Decision 2: 商品层级筛选模型
- Decision: 单次查询只允许一个聚合层级 `dimension`，枚举为 `goods_type|region|line|package`。
- Rationale: 与现有目录层级一致，口径清晰，避免多层交叉聚合导致 SQL 和图表语义复杂化。
- Alternatives considered: 允许多层 group by；被拒绝，因为首版会显著增加复杂度且不利于前端展示。

## Decision 3: 收入口径与数据来源
- Decision: 仅统计 `order_payments.status=approved` 的真实支付记录，金额字段沿用 cents(int64)。
- Rationale: 当前系统金额主模型为 `int64` 分单位，且 `report` 已基于 approved payment 统计，能保证与现有财务口径一致。
- Alternatives considered: 统计订单总额或待审支付；被拒绝，因为会引入未到账/未审核数据偏差。

## Decision 4: 同比环比计算与不可比标记
- Decision: 响应中提供 `yoy`/`mom` 指标与 `comparable` 布尔标记；若对比区间无基准数据，返回 `comparable=false` 与空变化率。
- Rationale: 避免把无基准错误解读为 0% 增长，满足财务分析准确性。
- Alternatives considered: 缺数据时返回0；被拒绝，因为会产生误导。

## Decision 5: Top5 与明细一致性
- Decision: Top5 和明细共用同一过滤器（时间、层级、层级值），并在响应附带 `query_hash` 供追踪。
- Rationale: 保证“图表-排行-明细”同口径，便于管理员核对。
- Alternatives considered: 各接口独立过滤参数；被拒绝，因为容易口径漂移。

## Decision 6: 权限与审计
- Decision: 复用 admin 路由与 `RequireAdminPermissionAuto`，新增权限码建议为 `dashboard.revenue_analytics`（或保持 `dashboard.revenue` 并扩展语义），记录 operator id + trace id + filter 摘要日志。
- Rationale: 现有系统已基于路由推导权限，增量成本低，符合服务端强校验要求。
- Alternatives considered: 只做前端鉴权；被拒绝，因为违反宪法安全要求。

## Decision 7: 性能方案
- Decision: 首版先落地聚合 SQL + 必要索引校验（支付时间、状态、订单关联），并为明细接口提供分页与排序白名单。
- Rationale: 在不引入预计算任务的前提下满足首版 p95 指标，复杂度可控。
- Alternatives considered: 预聚合物化表；被拒绝，因为首版迭代成本高且需要额外数据同步机制。

## Decision 8: 设计约束消解结果
- Decision: Technical Context 中的不确定项已全部收敛，无 `NEEDS CLARIFICATION` 残留。
- Rationale: 已明确时间/维度/口径/权限/性能/可比性策略，可直接进入设计与契约阶段。
- Alternatives considered: 保留开放项到开发阶段；被拒绝，因为会增加返工与契约漂移风险。
