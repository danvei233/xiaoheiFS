# Data Model - 收入统计初版

## RevenueAnalyticsQuery
- Fields:
  - `from_at` (datetime, required)
  - `to_at` (datetime, required)
  - `level` (enum, required): `goods_type|region|line|package`
  - `goods_type_id` (int64, conditional)
  - `region_id` (int64, conditional)
  - `line_id` (int64, conditional)
  - `package_id` (int64, conditional)
  - `page` (int, optional, default 1)
  - `page_size` (int, optional, default 20, max 200)
  - `sort_field` (enum, optional): `paid_at|amount`
  - `sort_order` (enum, optional): `asc|desc`
- Validation:
  - `from_at < to_at`
  - time range must be within configured upper bound
  - hierarchy must be contiguous by selected `level`:
    - level=`goods_type`: only `goods_type_id` required
    - level=`region`: `goods_type_id` + `region_id` required
    - level=`line`: `goods_type_id` + `region_id` + `line_id` required
    - level=`package`: all four ids required

## RevenueSummary
- Fields:
  - `total_revenue_cents` (int64)
  - `order_count` (int)
  - `yoy_ratio` (decimal nullable)
  - `mom_ratio` (decimal nullable)
  - `yoy_comparable` (bool)
  - `mom_comparable` (bool)

## RevenueShareItem
- Fields:
  - `dimension_id` (int64)
  - `dimension_name` (string)
  - `revenue_cents` (int64)
  - `ratio` (decimal 0..1)

## RevenueTrendPoint
- Fields:
  - `bucket` (date/datetime)
  - `revenue_cents` (int64)
  - `order_count` (int)

## RevenueTopItem
- Fields:
  - `rank` (int 1..5)
  - `dimension_id` (int64)
  - `dimension_name` (string)
  - `revenue_cents` (int64)
  - `ratio` (decimal 0..1)

## RevenueDetailRecord
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
  - `paid_at` (datetime)
  - `status` (enum: approved)

## Relationships
- `RevenueAnalyticsQuery` drives all output models.
- `RevenueSummary` / `RevenueShareItem[]` / `RevenueTrendPoint[]` / `RevenueTopItem[]` / `RevenueDetailRecord[]` are generated from the same filtered data range.
- `RevenueDetailRecord` is traceable to original payment and order entities.

## State Transitions
- Read-only query flow, no write-side state transitions.
- Comparable state for yoy/mom:
  - `true` when baseline period has valid sample set
  - `false` when baseline sample set absent
