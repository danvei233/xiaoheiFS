# Contributing to xiaoheiFS

Thank you for contributing to xiaoheiFS! This document outlines the conventions and principles that all contributors must follow.

## Mandatory Reading

Before writing any code, you **MUST** read and understand the project constitution:

- **Location**: `constitution.md`
- **Why**: The constitution defines non-negotiable principles that govern all code in this project

## Quick Reference

### 1. Validator-First Data Validation

**Always** use `go-playground/validator` with struct tags. Never write manual validation in handlers.

```go
// ✅ CORRECT: Declarative validation
type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Phone    string `json:"phone" validate:"omitempty,e164"`
}

func (h *Handler) CreateUser(c *gin.Context) {
    var req CreateUserRequest
    if err := bindJSON(c, &req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // ... proceed with valid data
}

// ❌ WRONG: Manual validation in handler
func (h *Handler) CreateUser(c *gin.Context) {
    var req CreateUserRequest
    c.ShouldBindJSON(&req)
    if req.Email == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
        return
    }
    if !strings.Contains(req.Email, "@") {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
        return
    }
}
```

### 2. Centralized Error Management

**Always** use predefined errors from `internal/domain/errors.go`. Never create errors inline.

```go
// ✅ CORRECT: Use predefined errors
import "xiaoheiplay/internal/domain"

func (s *Service) GetUser(id int64) (*User, error) {
    if id <= 0 {
        return nil, domain.ErrInvalidId
    }
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("find user: %w", err)
    }
    if user == nil {
        return nil, domain.ErrUserNotFound
    }
    return user, nil
}

// ❌ WRONG: Inline error creation
func (s *Service) GetUser(id int64) (*User, error) {
    if id <= 0 {
        return nil, errors.New("invalid id")  // Forbidden!
    }
    if user == nil {
        return nil, errors.New("user not found")  // Forbidden!
    }
}
```

**Adding new errors**: Add to `internal/domain/errors.go`:
```go
var (
    ErrNewFeatureNotEnabled = errors.New("new feature not enabled")
    ErrInvalidConfiguration = errors.New("invalid configuration")
)
```

### 3. Strict Layer Separation

**Never** bypass layer boundaries. Follow the dependency direction:

```
HTTP Handlers → Application Services → Repositories → Database
```

```go
// ✅ CORRECT: Handler calls service
func (h *Handler) GetOrder(c *gin.Context) {
    order, err := h.orderSvc.GetOrder(ctx, orderID)
    // Handler only handles HTTP concerns
}

// Service calls repository
func (s *Service) GetOrder(ctx context.Context, id int64) (*Order, error) {
    return s.orderRepo.FindByID(ctx, id)
}

// ❌ WRONG: Handler directly accesses database
func (h *Handler) GetOrder(c *gin.Context) {
    var order domain.Order
    h.db.First(&order, orderID)  // Forbidden! SQL in handler!
}
```

### 4. Dependency Injection

**Always** inject dependencies via constructors. Never instantiate directly.

```go
// ✅ CORRECT: Dependency injection
type Service struct {
    repo   OrderRepository
    broker EventBroker
}

func NewService(repo OrderRepository, broker EventBroker) *Service {
    return &Service{repo: repo, broker: broker}
}

// ❌ WRONG: Direct instantiation
func (h *Handler) GetOrder(c *gin.Context) {
    svc := &Service{repo: &GormRepo{}}  // Forbidden!
}
```

## Development Setup

### Prerequisites

- Go 1.25+
- Node.js 18+ (for frontend)
- MySQL / PostgreSQL / SQLite

### Backend

```bash
cd backend
go mod download
go run ./cmd/server
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

### Running Tests

```bash
# Backend
cd backend
go test ./...

# With coverage
go test -cover ./...

# Frontend
cd frontend
npm test
```

### Linting

```bash
# Backend
golangci-lint run

# Frontend
npm run lint
```

## Pull Request Process

1. **Create a branch** following naming convention:
   - Feature: `feat/###-feature-name`
   - Bugfix: `fix/###-bug-name`
   - Refactor: `refactor/###-description`

2. **Write tests** for new functionality

3. **Ensure all tests pass**:
   ```bash
   go test ./...
   npm test
   ```

4. **Run linter**:
   ```bash
   golangci-lint run
   npm run lint
   ```

5. **Verify constitution compliance** (see checklist below)

6. **Submit PR** with clear description

## Constitution Compliance Checklist

Before submitting a PR, verify:

- [ ] All validation uses `go-playground/validator` via `bindJSON` helpers
- [ ] No inline `errors.New()` in handlers, services, or repositories
- [ ] All new errors defined in `internal/domain/errors.go`
- [ ] No SQL/GORM operations in HTTP handlers
- [ ] All dependencies injected via constructors
- [ ] Tests written for new functionality
- [ ] No sensitive data in logs

## Architecture Overview

```
backend/
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── adapter/
│   │   ├── http/        # HTTP handlers, routing, middleware
│   │   └── repo/        # Repository implementations (GORM)
│   ├── app/             # Application services (business logic)
│   │   └── ports/       # Interface definitions
│   ├── domain/          # Entities, value objects, errors
│   └── pkg/             # Shared utilities
├── plugin-sdk/          # Plugin SDK for extensions
└── plugins/             # Plugin implementations
```

## Getting Help

- Review the constitution: `constitution.md`
- Check existing code patterns in the codebase
- Open an issue for questions

---

By contributing to this project, you agree to follow the principles defined in the constitution.

**Author**: 星云猫 nebulamao
