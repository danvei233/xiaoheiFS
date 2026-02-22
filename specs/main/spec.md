# Feature Specification: 收入统计初版

**Feature Branch**: `main`  
**Created**: 2026-02-20  
**Status**: Draft  
**Input**: User description: "收入统计初版 时间跨度可自由选择 介于目前商品等级为:类型-地区-线路-套餐 ... 四个层级可选择 占比图 趋势图 同比环比 top5 真实数据 明细表. 后台要能查看 给管理员看的"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 管理员查看收入总览分析 (Priority: P1)

管理员在后台选择时间跨度和商品层级（类型/地区/线路/套餐），查看收入占比图、趋势图、同比环比与 Top5 排名，用于快速判断经营表现。

**Why this priority**: 这是收入统计模块的核心价值，没有总览分析就无法支撑管理决策。

**Independent Test**: 使用真实交易数据，选择任意合法时间范围与层级后，页面可一次性返回占比、趋势、同比环比、Top5 且指标计算正确。

**Acceptance Scenarios**:

1. **Given** 管理员已登录后台且存在交易数据, **When** 选择最近30天+按地区聚合, **Then** 返回地区收入占比、每日趋势、同比环比、Top5。
2. **Given** 管理员切换层级为套餐, **When** 保持同一时间范围查询, **Then** 所有图表和指标按套餐维度重算并保持口径一致。

---

### User Story 2 - 管理员查看收入明细表 (Priority: P2)

管理员在完成总览后，可查看与筛选收入明细表，定位异常数据并进行核对。

**Why this priority**: 明细是总览结果可追溯的依据，是运营核账的重要支撑。

**Independent Test**: 选择同一时间范围和层级后，明细表总额与总览汇总口径一致，可按维度筛选并分页浏览。

**Acceptance Scenarios**:

1. **Given** 管理员已查询总览, **When** 打开明细表并筛选某线路, **Then** 仅展示该线路收入明细且分页信息正确。

---

### User Story 3 - 管理员安全访问统计能力 (Priority: P3)

只有具备管理员权限的账号可以访问收入统计接口与页面。

**Why this priority**: 财务数据敏感，必须保证最小权限访问。

**Independent Test**: 非管理员访问同接口时返回明确授权错误；管理员访问成功。

**Acceptance Scenarios**:

1. **Given** 普通用户已登录, **When** 调用收入统计接口, **Then** 返回 403 且错误结构化。

---

### Edge Cases

- 当时间范围超过系统允许跨度时，返回可读错误并给出允许范围。
- 当某时间段无交易数据时，图表返回空集与0值，不报错。
- 当跨年查询同比时，若去年同期无数据，明确标记“不可比”而非伪造0增长。
- 当层级筛选值不存在或已下线时，返回参数校验错误。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow admin users to choose arbitrary time range within configured limits for revenue analytics.
- **FR-002**: System MUST support aggregation by exactly one product hierarchy level per query: 类型, 地区, 线路, 套餐.
- **FR-003**: System MUST return revenue share dataset for pie/donut chart based on selected level and time range.
- **FR-004**: System MUST return revenue trend time-series dataset for the selected time granularity.
- **FR-005**: System MUST compute and return period-over-period metrics: 同比 and 环比 with explicit comparability flags.
- **FR-006**: System MUST return Top5 ranking by revenue under the same filter scope.
- **FR-007**: System MUST provide revenue detail table query with pagination and consistent filter semantics.
- **FR-008**: System MUST use real persisted transaction/order data as analytics source; mock/fallback synthetic data is forbidden.
- **FR-009**: System MUST enforce admin-only authorization server-side for all analytics endpoints.
- **FR-010**: System MUST provide deterministic, structured error responses for validation/auth/business errors.
- **FR-011**: System MUST log analytics queries with trace id, operator id, and filter summary without leaking secrets.
- **FR-012**: System MUST keep monetary computations decimal-safe and auditable to source transactions.

### Constitution Alignment Requirements *(mandatory)*

- **CA-001 (Architecture)**: Feature MUST preserve clean-architecture boundaries (`domain -> usecase -> adapter -> delivery`) and list any required exceptions.
- **CA-002 (Financial Safety)**: Feature MUST declare money precision strategy, idempotency behavior, and audit trail requirements for balance-impacting flows.
- **CA-003 (Contracts & Validation)**: Feature MUST define API versioning impact, boundary validation rules, and deterministic error response expectations.
- **CA-004 (Performance)**: Feature MUST include measurable performance targets (latency/throughput/query budget) and validation approach.
- **CA-005 (Observability & Security)**: Feature MUST define structured logging, trace identifiers, and server-side auth/tenant boundary checks.

### Key Entities *(include if feature involves data)*

- **RevenueAnalyticsQuery**: 管理员发起统计查询的参数集合，包含时间范围、层级、维度筛选、分页参数。
- **RevenueSummaryMetric**: 总览指标，包含总收入、同比环比、可比性标记、计算口径元数据。
- **RevenueShareItem**: 占比图单项，包含维度值、收入金额、占比。
- **RevenueTrendPoint**: 趋势图数据点，包含时间桶、收入金额、环比变化。
- **RevenueTopItem**: Top5 项目，包含排名、维度值、收入金额、占比。
- **RevenueDetailRecord**: 明细表记录，包含订单/交易标识、商品层级字段、收入金额、时间戳、数据来源。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 管理员在 30 秒内可完成一次查询并看到总览+Top5+明细首屏结果。
- **SC-002**: 在 10 万条交易数据范围内，95% 分析查询响应时间小于 2 秒。
- **SC-003**: 统计结果与财务对账样本比对时，金额误差为 0（按最小货币单位一致）。
- **SC-004**: 非管理员访问统计接口拦截率 100%，且所有拒绝请求均产生结构化审计日志。
