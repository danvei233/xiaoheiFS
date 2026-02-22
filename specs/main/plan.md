# Implementation Plan: 收入统计初版

**Branch**: `main` | **Date**: 2026-02-20 | **Spec**: `D:/项目/golang/xiaohei/specs/main/spec.md`
**Input**: Feature specification from `D:/项目/golang/xiaohei/specs/main/spec.md`

## Summary

为后台管理员提供“收入统计初版”能力，支持自由时间跨度与四级商品维度（类型/地区/线路/套餐）筛选，输出占比图、趋势图、同比环比、Top5 与明细表。实现路径基于现有 `report` 领域扩展：在 `app/report` 增加聚合与对比计算用例，在 `adapter/repo/core` 增加聚合查询，在 `adapter/http` 增加 admin API 与 DTO，前端管理端对接新接口渲染图表与表格，严格使用真实支付/订单数据。

## Technical Context

**Language/Version**: Go 1.25.0 (backend), TypeScript + Vue 3 (frontend)  
**Primary Dependencies**: Gin, GORM, go-playground/validator, Vue + Pinia + Ant Design Vue, ECharts wrapper  
**Storage**: MySQL/PostgreSQL/SQLite via GORM repositories (source data from orders + order_payments + catalog tables)  
**Testing**: `go test` (usecase + repository + HTTP integration), frontend store/service tests where modified  
**Target Platform**: Linux server backend + Web admin console
**Project Type**: Web application (backend + frontend)  
**Performance Goals**: p95 <= 2s for 100k payment rows query; details list <= 1s p95 for single page  
**Constraints**: 仅管理员可访问；金额按 cents(int64) 计算；不得使用 mock/fallback 数据；错误响应结构化且稳定  
**Scale/Scope**: 首版覆盖 1 个后台页面 + 4 个统计接口（总览、趋势、排行、明细）

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

Post-Design Re-check (after Phase 1): PASS. 设计产物已明确分层、金额口径、权限、契约、测试与性能验证路径，无需额外宪法豁免。

## Project Structure

### Documentation (this feature)

```text
specs/main/
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
│   │   ├── http/
│   │   └── repo/core/
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

**Structure Decision**: 采用现有 Web application 双端结构，在 `backend/internal/app/report` 扩展用例，在 `backend/internal/adapter/http` 与 `backend/internal/adapter/repo/core` 扩展接口和实现，前端在 `frontend/src/pages/admin` 新增统计页并通过 `services/stores` 调用后端。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
