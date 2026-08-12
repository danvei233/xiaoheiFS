# xiaohei Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-02-21

---

## ⚠️ MANDATORY: Read Constitution First

**Before writing or modifying any code, you MUST read the project constitution:**

```
constitution.md
```

The constitution defines **non-negotiable principles** that govern all development:

1. **Validator-First Data Validation** - Use `go-playground/validator`, no manual validation in handlers
2. **Centralized Error Management** - All errors in `internal/domain/errors.go`, no inline `errors.New()`
3. **Strict Layer Separation** - Handlers → Services → Repositories, no SQL in handlers
4. **Dependency Injection** - All dependencies via interfaces/ports
5. **Test-Driven Development** - Tests alongside implementation
6. **Observability** - Structured logging with zerolog
7. **Simplicity & YAGNI** - No premature abstractions

**Non-compliance will be rejected in code review.**

---

## Active Technologies
- Go 1.25.0 (backend), TypeScript (frontend) + Gin, GORM, go-playground/validator, zerolog, Vue 3 + Pinia + Ant Design Vue + ECharts (001-revenue-analytics)
- MySQL/PostgreSQL/SQLite via GORM (orders, order_payments, catalog hierarchy tables) (001-revenue-analytics)

- Go 1.25.0 (backend), TypeScript + Vue 3 (frontend) + Gin, GORM, go-playground/validator, Vue + Pinia + Ant Design Vue, ECharts wrapper (main)

## Project Structure

```text
backend/
frontend/
tests/
```

## Commands

npm test; npm run lint

## Code Style

Go 1.25.0 (backend), TypeScript + Vue 3 (frontend): Follow standard conventions

## Recent Changes
- 001-revenue-analytics: Added Go 1.25.0 (backend), TypeScript (frontend) + Gin, GORM, go-playground/validator, zerolog, Vue 3 + Pinia + Ant Design Vue + ECharts

- main: Added Go 1.25.0 (backend), TypeScript + Vue 3 (frontend) + Gin, GORM, go-playground/validator, Vue + Pinia + Ant Design Vue, ECharts wrapper

<!-- MANUAL ADDITIONS START -->

## Frontend (supersedes the auto-generated stack lines above)

The auto-generated sections above are stale: they predate the web-tier
consolidation and still say "Ant Design Vue", and they list a `backend/ frontend/
tests/` layout that never mentioned `adminweb/`.

Current state:

- **`frontend/`** is the single web application — a history-mode SPA carrying the
  public site, auth pages, user console (`/console`), installer (`/install`) and
  the admin console (at the operator-configured `admin_path`). Built with npm; the
  backend serves it from `./static`.
- **UI standard is Element Plus via the Art Design Pro template**, not Ant Design
  Vue. Constitution v1.1.0 records this change.
  - Conventions (required reading before writing a page):
    `docs/frontend/adp-conventions.md`
  - Migration plan and rules: `docs/frontend/adp-migration-plan.md`
- The legacy `adminweb/` SPA has been fully folded into `frontend/` and
  deleted; do not reference or restore it.

`frontend/src/pages`, `frontend/src/layouts` and the Ant Design components at the
top level of `frontend/src/components` are dead code kept only as a porting
reference. They are excluded from `tsconfig.json` and no route points at them.
New pages go in `frontend/src/views`.

Frontend commands (`frontend/package.json`) — note CONTRIBUTING.md's `npm test`
does not exist here; there is no frontend test runner yet:

```bash
npm run dev         # :5173, proxies /api, /admin/api, /sdk, /uploads to :8080
npm run typecheck   # vue-tsc --noEmit — must stay at zero errors
npm run lint        # eslint
npm run build       # vite build -> frontend/dist
```

Go note: the system `go` on this machine is 1.19; this project needs the 1.25
toolchain at `/usr/local/go/bin/go`.

<!-- MANUAL ADDITIONS END -->

**Author**: 星云猫 nebulamao
