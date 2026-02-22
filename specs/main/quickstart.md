# Quickstart - 收入统计初版

## 1. Prerequisites
- Go 1.25+
- Node.js 20+
- 已准备可用数据库并包含真实订单/支付数据
- 管理员账号具备 dashboard 相关权限

## 2. Backend implementation steps
1. 在 `backend/internal/app/report` 增加收入统计查询参数、汇总/趋势/Top/明细用例。
2. 在 `backend/internal/app/ports` 增加统计查询所需 repository 接口（若现有接口不足）。
3. 在 `backend/internal/adapter/repo/core` 实现聚合 SQL 与明细分页查询。
4. 在 `backend/internal/adapter/http` 增加 DTO、handler、路由：
   - `POST /admin/api/v1/dashboard/revenue-analytics/overview`
   - `POST /admin/api/v1/dashboard/revenue-analytics/trend`
   - `POST /admin/api/v1/dashboard/revenue-analytics/top`
   - `POST /admin/api/v1/dashboard/revenue-analytics/details`
5. 增加请求校验（时间范围、维度枚举、分页边界）和结构化错误响应。

## 3. Frontend implementation steps
1. 在 `frontend/src/services/admin.ts` 增加 4 个接口调用方法。
2. 在 `frontend/src/stores/admin.ts` 增加统计状态与请求动作。
3. 在 `frontend/src/pages/admin` 新增收入统计页面（占比图、趋势图、同比环比卡片、Top5、明细表）。
4. 将页面加入 admin 路由与菜单。

## 4. Verification checklist
- `go test ./...`（至少覆盖 report usecase、repo 聚合查询、admin handler 权限与参数校验）
- 非管理员调用接口返回 403
- 统计总额与样本对账一致（分单位完全一致）
- 同比/环比在无基准数据时返回 `comparable=false`
- 明细分页、排序、筛选口径与总览一致

## 5. Performance checks
- 准备 >=100k 支付记录测试集
- 采样 200 次查询：
  - overview/trend/top p95 <= 2s
  - details p95 <= 1s
- 记录 SQL explain 与关键索引命中情况
