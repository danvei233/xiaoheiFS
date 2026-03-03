<!--
  SYNC IMPACT REPORT
  ==================
  Version change: N/A → 1.0.0 (Initial creation)
  Added principles:
    - I. Validator-First Data Validation
    - II. Centralized Error Management
    - III. Strict Layer Separation
    - IV. Dependency Injection
    - V. Test-Driven Development
    - VI. Observability
    - VII. Simplicity & YAGNI
  Added sections:
    - Technology Standards
    - Development Workflow
    - Governance
  Templates requiring updates:
    - .specify/templates/plan-template.md: ✅ Compatible (Constitution Check section exists)
    - .specify/templates/spec-template.md: ✅ Compatible (requirements structure aligned)
    - .specify/templates/tasks-template.md: ✅ Compatible (task categorization aligned)
  Follow-up TODOs: None
-->

# xiaoheiFS Constitution

## Core Principles

### I. Validator-First Data Validation (NON-NEGOTIABLE)

All incoming data validation in the HTTP handler layer MUST use the `go-playground/validator` library via the established `bindJSON` or `bindJSONOptional` helpers. Manual validation logic in handlers is prohibited.

**Rules:**
- Define validation rules declaratively using struct tags (e.g., `validate:"required,email"`)
- Use `bindJSON(c, payload)` for required body parsing with automatic validation
- Use `bindJSONOptional(c, payload)` for optional body parsing
- Complex business validation belongs in the service layer, not handlers

**Rationale:** Declarative validation is self-documenting, consistent, and eliminates error-prone manual checks scattered throughout handlers.

### II. Centralized Error Management (NON-NEGOTIABLE)

All sentinel errors (domain-level error identifiers) MUST be defined in `internal/domain/errors.go`. Direct use of `errors.New()` outside of this file is prohibited in handler, service, and repository layers.

**Rules:**
- Define all domain errors as package-level `var` declarations in `internal/domain/errors.go`
- Import errors from `internal/app/shared/errors.go` for application-layer aliases when needed
- Never create ad-hoc sentinel errors in handlers or services using `errors.New()`
- **When propagating errors**: Wrap errors with context using `fmt.Errorf("context: %w", err)` — this is REQUIRED for error propagation, not prohibited
- HTTP status mapping happens only in the handler layer via the existing error-to-status conventions

**Distinction:**
- `errors.New("message")` — **ONLY in `internal/domain/errors.go`** for defining reusable sentinel errors
- `fmt.Errorf("context: %w", err)` — **REQUIRED anywhere** when wrapping and propagating errors up the call stack

**Rationale:** Centralized errors ensure consistent error handling, enable compile-time verification, and prevent error string drift across the codebase.

### III. Strict Layer Separation (NON-NEGOTIABLE)

The codebase follows a strict layered architecture. Violating layer boundaries is prohibited.

**Layer Responsibilities:**

| Layer | Path | Allowed Operations |
|-------|------|-------------------|
| HTTP Handlers | `internal/adapter/http/` | Request binding, response serialization, HTTP status mapping, service calls |
| Application Services | `internal/app/` | Business logic, orchestration, domain model manipulation |
| Repositories | `internal/adapter/repo/` | Database operations, GORM queries, data persistence |
| Domain | `internal/domain/` | Entity definitions, value objects, domain errors |

**Prohibited Actions:**
- Handlers MUST NOT execute SQL queries or GORM operations directly
- Handlers MUST NOT access the database layer directly
- Services MUST NOT handle HTTP concerns (request binding, response codes)
- Repositories MUST NOT contain business logic

**Rationale:** Layer separation ensures testability, maintainability, and the ability to swap implementations without cascading changes.

### IV. Dependency Injection

All cross-layer dependencies MUST be injected via interfaces (ports), not concrete implementations.

**Rules:**
- Define repository interfaces in `internal/app/ports/`
- Inject dependencies through struct constructors (e.g., `NewService(repo Repository)`)
- HTTP handlers receive services via `HandlerDeps` struct
- Services receive repositories via port interfaces
- Never instantiate dependencies inside functions using `&SomeType{}`

**Rationale:** Dependency injection enables mocking in tests, loose coupling, and explicit dependency graphs.

### V. Test-Driven Development

Critical business logic MUST have tests written before or alongside implementation.

**Rules:**
- Unit tests belong in `*_test.go` files adjacent to the code under test
- Integration tests use the `testutil` and `testutilhttp` packages
- Use table-driven tests for multiple scenarios
- Mock external dependencies (databases, APIs) in unit tests
- Test files follow the pattern `<source>_test.go`

**Rationale:** Tests document expected behavior and prevent regressions during refactoring.

### VI. Observability

Services MUST emit structured logs for significant operations using `zerolog`.

**Rules:**
- Use structured logging with contextual fields (e.g., `user_id`, `order_id`)
- Log at appropriate levels: Debug (development), Info (operations), Warn (degraded), Error (failures)
- Include request context in logs when available
- Never log sensitive data (passwords, tokens, personal information)

**Rationale:** Structured logs enable debugging, monitoring, and audit trails in production.

### VII. Simplicity & YAGNI

Code MUST be as simple as the requirements allow. Avoid speculative generalization.

**Rules:**
- No premature abstractions: wait for the third occurrence before extracting
- No unused code paths "for future use"
- Prefer standard library solutions over third-party packages when practical
- Keep functions focused on a single responsibility
- Avoid deep nesting; extract helper functions or use guard clauses

**Rationale:** Simplicity reduces cognitive load, maintenance burden, and bug surface area.

## Technology Standards

### Backend

| Category | Technology |
|----------|------------|
| Language | Go 1.25+ |
| HTTP Framework | Gin |
| ORM | GORM |
| Validation | go-playground/validator v10 |
| Logging | zerolog |
| Database | MySQL / PostgreSQL / SQLite |
| Plugin System | go-plugin (gRPC + protobuf) |

### Frontend

| Category | Technology |
|----------|------------|
| Language | TypeScript |
| Framework | Vue 3 |
| State Management | Pinia |
| UI Library | Ant Design Vue |
| Charts | ECharts |

### Code Conventions

- Follow standard Go formatting (`gofmt`, `goimports`)
- Use meaningful variable names; avoid cryptic abbreviations
- Exported functions and types require documentation comments
- Keep test files in the same package as the code under test

## Development Workflow

### Code Review Requirements

1. All changes require review before merge
2. Reviewers MUST verify constitution compliance
3. Tests MUST pass in CI before approval
4. Breaking changes require documentation updates

### Quality Gates

| Gate | Tool | Threshold |
|------|------|-----------|
| Linting | golangci-lint | Zero errors |
| Tests | go test | All pass |
| Coverage | go test -cover | New code covered |

### Branch Naming

- Feature: `feat/###-feature-name`
- Bugfix: `fix/###-bug-name`
- Refactor: `refactor/###-description`

## Governance

### Amendment Procedure

1. Propose amendment with rationale in writing
2. Document impact on existing code
3. Require team approval for principle changes
4. Update constitution version and last amended date
5. Propagate changes to affected templates and documentation

### Versioning Policy

- **MAJOR**: Principle removal or incompatible redefinition
- **MINOR**: New principle added or guidance materially expanded
- **PATCH**: Clarifications, typo fixes, non-semantic refinements

### Compliance Review

- All PRs MUST verify compliance with constitution principles
- Violations require explicit justification in the Sync Impact Report
- Unjustified complexity MUST be refactored before merge

### Runtime Guidance

For development-time guidance and workflow instructions, refer to `AGENTS.md` in the repository root.

**Version**: 1.0.0 | **Ratified**: 2026-03-02 | **Last Amended**: 2026-03-02 | **Author**: 星云猫 nebulamao
