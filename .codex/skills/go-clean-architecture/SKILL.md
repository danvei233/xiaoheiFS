---
name: go-clean-architecture
description: Expert knowledge in Go clean architecture patterns and best practices
---

# Go Clean Architecture Skill

## Overview

Clean Architecture in Go emphasizes separation of concerns through distinct layers, with dependencies pointing inward toward the domain.

## Layer Structure

### Domain Layer (innermost)
**Location:** `internal/domain/`

**Contains:**
- Business entities (structs)
- Repository interfaces
- Domain logic and validation
- Business rules

**Rules:**
- NO external dependencies
- NO framework dependencies
- Pure business logic
- Defines contracts for outer layers

**Example:**
```go
// internal/domain/account.go
package domain

type Account struct {
    ID      string
    Name    string
    Type    AccountType
    Balance int // cents
}

type AccountRepository interface {
    Create(account *Account) error
    GetByID(id string) (*Account, error)
    Update(account *Account) error
    Delete(id string) error
}

// Domain validation
func (a *Account) Validate() error {
    if a.Name == "" {
        return ErrInvalidName
    }
    if !a.Type.IsValid() {
        return ErrInvalidType
    }
    return nil
}
```

### Application Layer (middle)
**Location:** `internal/application/`

**Contains:**
- Business logic services
- Use case orchestration
- Service interfaces
- Cross-cutting concerns

**Rules:**
- Depends ONLY on domain interfaces
- NO HTTP dependencies
- NO database dependencies
- Orchestrates domain entities

**Example:**
```go
// internal/application/account_service.go
package application

import "internal/domain"

type AccountService struct {
    repo domain.AccountRepository // Interface, not concrete type
}

func NewAccountService(repo domain.AccountRepository) *AccountService {
    return &AccountService{repo: repo}
}

func (s *AccountService) CreateAccount(account *domain.Account) error {
    if err := account.Validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    if err := s.repo.Create(account); err != nil {
        return fmt.Errorf("failed to create account: %w", err)
    }

    return nil
}
```

### Infrastructure Layer (outermost)
**Location:** `internal/infrastructure/`

**Contains:**
- Repository implementations
- HTTP handlers
- Database logic
- External service integrations

**Rules:**
- Implements domain interfaces
- Can have external dependencies
- Handlers should be thin (parse → service → respond)
- Repositories only handle persistence

**Example:**
```go
// internal/infrastructure/repository/account_repository.go
package repository

import (
    "database/sql"
    "internal/domain"
)

type AccountRepository struct {
    db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
    return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(account *domain.Account) error {
    query := `INSERT INTO accounts (id, name, type, balance) VALUES (?, ?, ?, ?)`
    _, err := r.db.Exec(query, account.ID, account.Name, account.Type, account.Balance)
    return err
}

// internal/infrastructure/http/handlers/account_handler.go
package handlers

type AccountHandler struct {
    service *application.AccountService
}

func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request
    var req CreateAccountRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    // 2. Call service
    account := req.ToDomain()
    if err := h.service.CreateAccount(account); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 3. Return response
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(account)
}
```

## Dependency Injection

Wire dependencies in main.go:

```go
// cmd/server/main.go
func main() {
    // Infrastructure
    db := setupDatabase()

    // Repositories (concrete implementations)
    accountRepo := repository.NewAccountRepository(db)

    // Services (injected with interfaces)
    accountService := application.NewAccountService(accountRepo)

    // Handlers (injected with services)
    accountHandler := handlers.NewAccountHandler(accountService)

    // Router
    router := setupRouter(accountHandler)

    http.ListenAndServe(":8080", router)
}
```

## Common Patterns

### Repository Pattern
```go
// Domain defines interface
type Repository interface {
    Create(entity *Entity) error
    GetByID(id string) (*Entity, error)
}

// Infrastructure implements
type SQLRepository struct {
    db *sql.DB
}

func (r *SQLRepository) Create(entity *Entity) error {
    // SQL implementation
}
```

### Service Pattern
```go
type Service struct {
    repo domain.Repository  // Depend on interface
}

func (s *Service) DoBusinessLogic(entity *domain.Entity) error {
    // Validate
    // Transform
    // Call repository
    return s.repo.Create(entity)
}
```

### Handler Pattern
```go
func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
    // Parse → Service → Respond
    req := parseRequest(r)
    result, err := h.service.Do(req)
    respond(w, result, err)
}
```

## Anti-Patterns to Avoid

### ❌ Domain with External Dependencies
```go
// BAD: Domain importing database
import "database/sql"

type Account struct {
    db *sql.DB  // ❌ Domain shouldn't know about database
}
```

### ❌ Service with HTTP/Database
```go
// BAD: Service with HTTP dependency
func (s *Service) Create(w http.ResponseWriter, r *http.Request) {
    // ❌ Service shouldn't handle HTTP
}

// BAD: Service with database dependency
func (s *Service) Create(db *sql.DB, entity *Entity) error {
    // ❌ Service should use repository interface
}
```

### ❌ Handler with Business Logic
```go
// BAD: Complex logic in handler
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    // Parse
    // ❌ Complex validation
    // ❌ Calculations
    // ❌ Business rules
    // Direct database access
}

// GOOD: Thin handler
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    req := parse(r)
    result := h.service.Create(req)  // Service has the logic
    respond(w, result)
}
```

### ❌ Repository with Business Logic
```go
// BAD: Business rules in repository
func (r *Repository) Create(account *Account) error {
    // ❌ Business validation in repository
    if account.Balance < 0 && account.Type != "credit" {
        return errors.New("invalid")
    }
    // Should only handle persistence
}
```

## Testing Strategy

### Domain Tests
```go
func TestAccount_Validate(t *testing.T) {
    // Test entity validation
    // No mocks needed
}
```

### Service Tests (Unit)
```go
func TestService_Create(t *testing.T) {
    mockRepo := &MockRepository{}  // Mock interface
    service := NewService(mockRepo)
    // Test business logic
}
```

### Repository Tests (Integration)
```go
func TestRepository_Create(t *testing.T) {
    db := setupTestDB()  // Real database
    repo := NewRepository(db)
    // Test persistence
}
```

### Handler Tests (E2E)
```go
func TestHandler_Create(t *testing.T) {
    mockService := &MockService{}
    handler := NewHandler(mockService)
    req := httptest.NewRequest("POST", "/", body)
    w := httptest.NewRecorder()
    handler.Create(w, req)
    // Test HTTP layer
}
```

## Benefits

✅ **Testability**: Easy to mock dependencies
✅ **Maintainability**: Clear separation of concerns
✅ **Flexibility**: Easy to swap implementations
✅ **Independence**: Domain logic independent of frameworks
✅ **Scalability**: Easy to add features

## When to Apply

- Multi-layer applications
- Complex business logic
- Long-lived projects
- Team projects requiring clear boundaries
- Applications that may change databases/frameworks

## Quick Checklist

- [ ] Domain has no external dependencies
- [ ] Application uses interfaces, not concrete types
- [ ] Handlers are thin (parse → service → respond)
- [ ] Repositories only handle persistence
- [ ] Dependencies point inward
- [ ] Business logic in services, not handlers
- [ ] Each layer has clear responsibility
