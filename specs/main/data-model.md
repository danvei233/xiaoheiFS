# Data Model - 收入统计初版

## 1. RevenueAnalyticsQuery
- Purpose: 管理员发起统计查询的标准参数。
- Fields:
  - `from` (datetime, required): 开始时间，`from < to`。
  - `to` (datetime, required): 结束时间。
  - `dimension` (enum, required): `goods_type|region|line|package`。
  - `dimension_id` (int64, optional): 维度过滤值；未传表示该层级全量。
  - `granularity` (enum, optional): `day|week|month`，默认 `day`。
  - `page` (int, optional): 明细分页页码，默认 1。
  - `page_size` (int, optional): 明细分页大小，默认 20，最大 200。
  - `sort_by` (enum, optional): `paid_at|amount`。
  - `sort_order` (enum, optional): `asc|desc`。
- Validation:
  - 时间跨度 `to-from <= 366d`。
  - `dimension_id` 必须存在于对应层级主数据。
  - 分页参数为正整数，超限返回 400。

## 2. RevenueSummaryMetric
- Purpose: 总览指标。
- Fields:
  - `total_revenue_cents` (int64)
  - `total_orders` (int)
  - `yoy_rate` (number|null)
  - `mom_rate` (number|null)
  - `yoy_comparable` (bool)
  - `mom_comparable` (bool)
  - `currency` (string, e.g. CNY)
- Derived from:
  - `order_payments` (approved only) + 时间窗口。

## 3. RevenueShareItem
- Purpose: 占比图数据。
- Fields:
  - `dimension` (enum)
  - `dimension_id` (int64)
  - `dimension_name` (string)
  - `revenue_cents` (int64)
  - `ratio` (number, 0..1)

## 4. RevenueTrendPoint
- Purpose: 趋势图序列点。
- Fields:
  - `bucket_start` (datetime)
  - `bucket_end` (datetime)
  - `revenue_cents` (int64)
  - `order_count` (int)

## 5. RevenueTopItem
- Purpose: Top5 排名。
- Fields:
  - `rank` (int, 1..5)
  - `dimension` (enum)
  - `dimension_id` (int64)
  - `dimension_name` (string)
  - `revenue_cents` (int64)
  - `ratio` (number, 0..1)

## 6. RevenueDetailRecord
- Purpose: 明细表记录（可追溯到原始支付/订单）。
- Fields:
  - `payment_id` (int64)
  - `order_id` (int64)
  - `order_no` (string)
  - `user_id` (int64)
  - `goods_type_id` (int64)
  - `region_id` (int64)
  - `line_id` (int64)
  - `package_id` (int64)
  - `amount_cents` (int64)
  - `currency` (string)
  - `method` (string)
  - `status` (string, must be `approved`)
  - `paid_at` (datetime)

## Relationships
- `RevenueAnalyticsQuery` drives all aggregates.
- `RevenueSummaryMetric`、`RevenueShareItem[]`、`RevenueTrendPoint[]`、`RevenueTopItem[]`、`RevenueDetailRecord[]` share the same filter semantics.
- 明细记录通过 `payment_id/order_id` 可回溯到 `domain.OrderPayment` 与 `domain.Order`。

## State Transitions
- 查询对象是无状态读操作。
- 可比性状态:
  - `comparable=true` when baseline period has data.
  - `comparable=false` when baseline period has zero/empty source rows.
