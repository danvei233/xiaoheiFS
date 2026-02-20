# Implementation Plan: 收入统计初版

**Branch**: `001-revenue-analytics` | **Date**: 2026-02-20 | **Spec**: `D:/项目/golang/xiaohei/specs/001-revenue-analytics/spec.md`
**Input**: Feature specification from `D:/项目/golang/xiaohei/specs/001-revenue-analytics/spec.md`

## Summary

面向后台管理员新增收入统计初版：支持自由时间跨度、递进层级筛选（类型→地区→线路→套餐）、占比图、趋势图、同比环比、Top5 与明细表，并强制使用真实入账数据。技术方案是在现有 `report` 业务域中扩展统计用例和仓储聚合查询，通过 admin 接口提供统一口径数据给后台页面。

## Technical Context

**Language/Version**: Go 1.25.0 (backend), TypeScript (frontend)  
**Primary Dependencies**: Gin, GORM, go-playground/validator, zerolog, Vue 3 + Pinia + Ant Design Vue + ECharts  
**Storage**: MySQL/PostgreSQL/SQLite via GORM (orders, order_payments, catalog hierarchy tables)  
**Testing**: `go test ./...` for usecase/repo/http integration + frontend service/store tests for changed modules  
**Target Platform**: Linux server + web admin console
**Project Type**: web (backend + frontend)  
**Performance Goals**: 95% analytics queries <= 3s; details query <= 1s in normal pagination path  
**Constraints**: strict admin-only access; hierarchical filters must be validated; real settled data only; deterministic error model  
**Scale/Scope**: one admin analytics page, four analytics endpoints (overview/trend/top/details), initial scope for current product hierarchy

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] Architecture boundary preserved: `domain -> usecase -> adapter -> delivery`
      with inward-only dependencies and no framework imports in domain/usecase.
- [x] Financial correctness controls defined: decimal-safe money model,
      idempotency strategy, and immutable audit trail for balance-impacting flows.
- [x] API and validation discipline defined: versioning approach, DTO validation,
      business invariant checks, and deterministic error contract.
- [x] Test gates defined: usecase unit tests + repository/API integration tests
      for all business-critical paths, including bug-first regression tests.
- [x] Observability and performance evidence planned: structured logs with
      trace IDs and measurable latency/throughput/query budget checks.
- [x] Security and tenancy controls verified as server-enforced, with migration
      safety (forward migration + explicit rollback script) documented.

Post-Design Re-check: PASS. Phase 1 artifacts define layered boundaries, fixed money semantics, contract validation, test gates, and observability/performance checks with no unresolved constitutional violation.

## Project Structure

### Documentation (this feature)

```text
specs/001-revenue-analytics/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── revenue-analytics.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/
├── internal/
│   ├── domain/
│   ├── app/
│   │   └── report/
│   ├── adapter/
│   │   ├── repo/core/
│   │   └── http/
│   └── pkg/
├── migrations/
└── tests/

frontend/
├── src/
│   ├── pages/admin/
│   ├── stores/
│   └── services/
└── tests/
```

**Structure Decision**: 采用现有前后端分层结构，后端在 `report` 用例 + `repo/core` 聚合查询 + `http` delivery 落地，前端在 admin 页和 store/service 增量接入，避免跨层耦合。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |

## Performance Validation (2026-02-21)

- Validation command: `go test ./internal/adapter/http -run RevenueAnalytics -v`
- Sampling: 20 requests per endpoint in test env, p95 computed in `TestHandlers_AdminRevenueAnalyticsLatencyBudget`
- p95 results:
  - `POST /admin/api/v1/dashboard/revenue-analytics/overview`: 9.1413ms
  - `POST /admin/api/v1/dashboard/revenue-analytics/trend`: 9.0124ms
  - `POST /admin/api/v1/dashboard/revenue-analytics/top`: 7.987ms
  - `POST /admin/api/v1/dashboard/revenue-analytics/details`: 9.3082ms
- Budget check:
  - overview/trend/top target <= 3s: PASS
  - details target <= 1s: PASS
