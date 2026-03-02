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
<!-- MANUAL ADDITIONS END -->

**Author**: 星云猫 nebulamao
