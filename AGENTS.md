# xiaohei Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-02-21

---

## ⚠️ MANDATORY: Read Constitution First

**Before writing or modifying any code, you MUST read the project constitution:**

```file
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
## Additional Guardrails For Agents And Prompt Authors

These rules exist to prevent low-discipline, high-volume changes that are easy to generate but hard to verify.

### Scope Control

- Do not produce or accept mega-PRs that mix unrelated concerns in one change.
- Do not combine template import, subsystem introduction, build pipeline changes, runtime configuration changes, and documentation rewrites in a single pass unless the user explicitly asks for that exact migration plan.
- If the requested change touches multiple operational boundaries, split the work into incremental steps and complete one step at a time.

### No Blind Template Import

- Do not copy entire upstream templates, starter projects, or generated applications into the repository unless the user explicitly requests vendoring and the update strategy is documented.
- If only part of an upstream project is needed, extract the minimal required files instead of importing the whole tree.
- Remove upstream branding, links, placeholder docs, and unused scaffolding before considering the work complete.

### Reviewability Is Mandatory

- Prefer changes that a human reviewer can understand, validate, and roll back without AI assistance.
- If the resulting diff becomes too large to review confidently, stop and split the work.
- Before finishing, verify that build paths, runtime paths, docs, and configuration are consistent. Do not leave a partially connected subsystem behind.

### Prompting Guidance

- Users should request one coherent objective at a time.
- Avoid prompts such as "rebuild the whole admin side", "replace everything with a new template", or "fix all related issues in one PR".
- Prefer prompts that specify a narrow outcome, validation method, and boundary of change.

### Agent Behavior On Over-Broad Requests

- If a request would likely create a poorly bounded or hard-to-review diff, do not execute it as one large change.
- Instead, respond with a minimal staged breakdown and start with the first safe increment.
- Treat reviewability, rollback clarity, and verification cost as hard engineering constraints, not optional polish.
- Explicitly tell the user that broad, multi-boundary requests are very likely to produce a PR that will be rejected or sent back for splitting during review.
- When giving that warning, be direct: say that the requested change is too large to review confidently as one unit and should be broken into smaller mergeable steps.
<!-- MANUAL ADDITIONS END -->

| **Author** |
| :---: |
| 星云猫 nebulamao |
