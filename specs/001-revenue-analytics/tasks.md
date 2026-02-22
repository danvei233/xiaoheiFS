# Tasks: 收入统计初版

**Input**: Design documents from `/specs/001-revenue-analytics/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are risk-based. This feature touches finance analytics and admin authorization, so contract/integration/usecase tests are mandatory.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- All task descriptions include exact file paths

## Path Conventions

- Backend: `backend/internal/...`, `backend/cmd/...`
- Frontend: `frontend/src/...`
- Specs: `specs/001-revenue-analytics/...`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare baseline contracts, docs, and feature scaffolding.

- [X] T001 Validate and align API contract naming in `specs/001-revenue-analytics/contracts/revenue-analytics.openapi.yaml`
- [X] T002 Create backend report DTO file scaffold in `backend/internal/app/report/revenue_analytics.go`
- [X] T003 [P] Create backend HTTP DTO scaffold in `backend/internal/adapter/http/dto_revenue_analytics.go`
- [X] T004 [P] Create frontend page scaffold in `frontend/src/pages/admin/RevenueAnalytics.vue`
- [X] T005 [P] Create frontend store scaffold in `frontend/src/stores/revenueAnalytics.ts`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core prerequisites required before any user story implementation.

**CRITICAL**: No user story work can begin until this phase is complete.

- [X] T006 Define revenue analytics service/repository contracts in `backend/internal/app/ports/ports.go`
- [X] T007 Implement shared hierarchy validation and filter normalization in `backend/internal/app/report/revenue_analytics.go`
- [X] T008 [P] Implement shared response/error mapping helpers for analytics endpoints in `backend/internal/adapter/http/dto_revenue_analytics.go`
- [X] T009 [P] Add repository SQL aggregate/query base methods in `backend/internal/adapter/repo/core/gorm_repo_report_analytics.go`
- [X] T010 Wire report service dependencies in server bootstrap in `backend/cmd/server/main.go`
- [X] T011 Register admin routes placeholders for revenue analytics endpoints in `backend/internal/adapter/http/router_routes_admin.go`
- [X] T012 Add permission mapping and route-to-permission inference updates in `backend/internal/pkg/permissions/auto.go`
- [X] T013 Add performance measurement checklist notes for this feature in `specs/001-revenue-analytics/quickstart.md`

**Checkpoint**: Foundation ready - user story work can start.

---

## Phase 3: User Story 1 - 查看收入总览 (Priority: P1) 🎯 MVP

**Goal**: 管理员可按时间范围与递进层级查看占比、趋势、同比环比、Top5。

**Independent Test**: 在真实数据环境下调用 overview/trend/top 接口并打开页面，三类图表与指标均返回且口径一致。

### Tests for User Story 1 (MANDATORY) ⚠️

- [X] T014 [P] [US1] Add contract tests for overview/trend/top endpoints in `backend/internal/adapter/http/handlers_admin_dashboard_test.go`
- [X] T015 [P] [US1] Add report usecase unit tests for hierarchy validation and comparability flags in `backend/internal/app/report/service_test.go`
- [X] T016 [P] [US1] Add repository integration tests for aggregate queries in `backend/internal/adapter/repo/core/sqlite_repo_report_analytics_test.go`

### Implementation for User Story 1

- [X] T017 [US1] Implement overview/trend/top domain models and query logic in `backend/internal/app/report/revenue_analytics.go`
- [X] T018 [US1] Implement overview/trend/top repository methods in `backend/internal/adapter/repo/core/gorm_repo_report_analytics.go`
- [X] T019 [US1] Implement admin handlers for overview/trend/top in `backend/internal/adapter/http/handlers_admin_dashboard.go`
- [X] T020 [US1] Register overview/trend/top routes in `backend/internal/adapter/http/router_routes_admin.go`
- [X] T021 [US1] Add request DTO validation for from/to/level and hierarchy ids in `backend/internal/adapter/http/dto_revenue_analytics.go`
- [X] T022 [US1] Add structured logs (trace id/operator/filter summary) for overview/trend/top in `backend/internal/adapter/http/handlers_admin_dashboard.go`
- [X] T023 [US1] Implement admin service calls for overview/trend/top in `frontend/src/services/admin.ts`
- [X] T024 [US1] Implement store actions/state for overview/trend/top in `frontend/src/stores/revenueAnalytics.ts`
- [X] T025 [US1] Build overview charts and KPI cards in `frontend/src/pages/admin/RevenueAnalytics.vue`
- [X] T026 [US1] Add route/menu entry for revenue analytics page in `frontend/src/router/index.ts`

**Checkpoint**: User Story 1 is independently functional and demo-ready (MVP).

---

## Phase 4: User Story 2 - 查看收入明细 (Priority: P2)

**Goal**: 管理员在同口径下查看分页明细并与总览核对。

**Independent Test**: 调用 details 接口后明细可分页排序，汇总可对齐 US1 总览口径。

### Tests for User Story 2 (MANDATORY) ⚠️

- [X] T027 [P] [US2] Add contract tests for details endpoint paging/sorting in `backend/internal/adapter/http/handlers_admin_dashboard_test.go`
- [X] T028 [P] [US2] Add repository integration tests for details query and ordering in `backend/internal/adapter/repo/core/sqlite_repo_report_analytics_test.go`
- [X] T029 [P] [US2] Add frontend store tests for details pagination state in `frontend/src/stores/__tests__/revenueAnalytics.spec.ts`

### Implementation for User Story 2

- [X] T030 [US2] Implement details query model and service method in `backend/internal/app/report/revenue_analytics.go`
- [X] T031 [US2] Implement details repository method with same filter semantics in `backend/internal/adapter/repo/core/gorm_repo_report_analytics.go`
- [X] T032 [US2] Implement details admin handler and response mapping in `backend/internal/adapter/http/handlers_admin_dashboard.go`
- [X] T033 [US2] Implement frontend admin service call for details in `frontend/src/services/admin.ts`
- [X] T034 [US2] Implement store action for details pagination/sorting/filter sync in `frontend/src/stores/revenueAnalytics.ts`
- [X] T035 [US2] Implement details table UI and linkage with current filters in `frontend/src/pages/admin/RevenueAnalytics.vue`

**Checkpoint**: User Stories 1 and 2 are independently testable and consistent.

---

## Phase 5: User Story 3 - 安全访问与可审计 (Priority: P3)

**Goal**: 仅管理员可访问统计能力，查询操作可审计追踪。

**Independent Test**: 非管理员请求被拒绝；管理员请求成功并留存可检索日志。

### Tests for User Story 3 (MANDATORY) ⚠️

- [X] T036 [P] [US3] Add authorization integration tests for analytics endpoints in `backend/internal/adapter/http/admin_security_guard_test.go`
- [X] T037 [P] [US3] Add audit log verification tests for analytics query actions in `backend/internal/adapter/http/handlers_admin_dashboard_test.go`

### Implementation for User Story 3

- [X] T038 [US3] Enforce analytics endpoint permissions and deny-path error consistency in `backend/internal/adapter/http/router_routes_admin.go`
- [X] T039 [US3] Add audit action names and persistence fields for analytics queries in `backend/internal/adapter/http/handlers_admin_dashboard.go`
- [X] T040 [US3] Expose permission-aware UI state (hide/disable access) in `frontend/src/pages/admin/RevenueAnalytics.vue`

**Checkpoint**: All user stories meet security and auditability expectations.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final hardening across stories.

- [X] T041 [P] Update feature quickstart verification steps and evidence checklist in `specs/001-revenue-analytics/quickstart.md`
- [X] T042 Run full backend regression focused on analytics modules via `backend/internal/app/report/service_test.go`
- [X] T043 [P] Run frontend lint/type checks and fix issues in `frontend/src/pages/admin/RevenueAnalytics.vue`
- [X] T044 Validate p95 timing targets and document measurements in `specs/001-revenue-analytics/plan.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: no dependencies.
- **Phase 2 (Foundational)**: depends on Phase 1; blocks all user stories.
- **Phase 3 (US1)**: depends on Phase 2; defines MVP.
- **Phase 4 (US2)**: depends on Phase 2 and reuses US1 filters/state contracts.
- **Phase 5 (US3)**: depends on Phase 2 and analytics endpoints from US1/US2.
- **Phase 6 (Polish)**: depends on all selected user stories.

### User Story Completion Order

1. **US1 (P1)** → MVP baseline
2. **US2 (P2)** → detail drill-down
3. **US3 (P3)** → security/audit hardening

### Within Each User Story

- Tests first and expected to fail before implementation.
- Service/query logic before handler wiring.
- Backend contract stabilization before frontend integration.

---

## Parallel Execution Examples

### User Story 1

```bash
# Parallel tests
T014 + T015 + T016

# Parallel frontend scaffolding after backend contracts are stable
T023 + T024
```

### User Story 2

```bash
# Parallel validation
T027 + T028 + T029

# Parallel implementation on separate layers
T033 + T034
```

### User Story 3

```bash
# Parallel security and audit tests
T036 + T037
```

---

## Implementation Strategy

### MVP First (US1 only)

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 (US1).
3. Validate independent test for US1.
4. Demo/deploy MVP.

### Incremental Delivery

1. Add US2 for drill-down details and verify independent test.
2. Add US3 for permission/audit hardening and verify independent test.
3. Execute Phase 6 polish and performance evidence collection.

### Compatibility Note

- Tasks are aligned to existing modules (`report`, admin router/handler, admin service/store) to maximize compatibility with current system structure and avoid cross-layer rewrites.
