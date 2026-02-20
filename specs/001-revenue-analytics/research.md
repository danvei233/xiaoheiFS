# Phase 0 Research - 收入统计初版

## Decision: 层级筛选采用严格级联
- Decision: 过滤条件必须按 `类型 -> 地区 -> 线路 -> 套餐` 递进校验，下级查询必须带齐上级范围。
- Rationale: 与业务层级定义一致，防止“套餐跨线路/地区”导致统计口径歧义。
- Alternatives considered: 任意层级独立筛选；被拒绝，因为会导致口径不一致和结果不可解释。

## Decision: 数据范围口径
- Decision: 收入统计仅使用真实已入账数据（已审核通过支付记录），不允许模拟/兜底数据。
- Rationale: 保证财务可追溯性并满足审计要求。
- Alternatives considered: 混入待审核或订单金额；被拒绝，因为会引入未结算偏差。

## Decision: 时间跨度策略
- Decision: 支持自由起止时间，但设置可配置最大跨度并返回明确超限错误。
- Rationale: 满足业务灵活性，同时控制查询成本与响应稳定性。
- Alternatives considered: 仅固定预设周期；被拒绝，因为不满足“自由选择”要求。

## Decision: 同比环比可比性
- Decision: 返回同比/环比时附带可比标识；基准数据缺失时标记不可比而非伪造0值。
- Rationale: 避免误导管理判断。
- Alternatives considered: 缺失视为0；被拒绝，因为语义错误。

## Decision: 图表与明细同口径
- Decision: 占比、趋势、Top5、明细共用同一过滤语义与数据范围。
- Rationale: 保证“总览可下钻，明细可对账”。
- Alternatives considered: 各接口独立参数；被拒绝，因为易产生口径漂移。

## Decision: 技术依赖最佳实践
- Decision: Gin 仅处理 DTO 和鉴权，usecase 承担业务规则，repo 仅做持久层聚合，金额保持 cents(int64) 计算，日志包含 trace/operator/filter 摘要。
- Rationale: 对齐宪法中的 clean architecture、财务正确性与可观测性要求。
- Alternatives considered: 在 handler/repo 夹杂业务规则；被拒绝，因为破坏层次边界与可测试性。

## Decision: 未决项状态
- Decision: Technical Context 中无 `NEEDS CLARIFICATION` 残留。
- Rationale: 关键范围、层级规则、口径和质量目标均已明确。
- Alternatives considered: 延后到实现阶段再定；被拒绝，因为会增加返工风险。
