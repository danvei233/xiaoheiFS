package usecase_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/usecase"
)

func TestAPIKeyServiceValidate(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	raw := "key-raw"
	sum := sha256.Sum256([]byte(raw))
	hash := hex.EncodeToString(sum[:])
	key := &domain.APIKey{Name: "test", KeyHash: hash, Status: domain.APIKeyStatusActive, ScopesJSON: `["*"]`}
	if err := repo.CreateAPIKey(ctx, key); err != nil {
		t.Fatalf("create api key: %v", err)
	}
	svc := usecase.NewAPIKeyService(repo)
	if _, err := svc.Validate(ctx, raw); err != nil {
		t.Fatalf("validate api key: %v", err)
	}
}

func TestCartServiceRemoveClearAndIntegrationLogs(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user := domain.User{Username: "cartu", Email: "cartu@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	item := &domain.CartItem{UserID: user.ID, PackageID: 1, SystemID: 1, SpecJSON: "{}", Qty: 1, Amount: 99000}
	if err := repo.AddCartItem(ctx, item); err != nil {
		t.Fatalf("add cart item: %v", err)
	}
	cartSvc := usecase.NewCartService(repo, repo, repo)
	if err := cartSvc.Remove(ctx, user.ID, item.ID); err != nil {
		t.Fatalf("remove cart item: %v", err)
	}
	item2 := &domain.CartItem{UserID: user.ID, PackageID: 2, SystemID: 2, SpecJSON: "{}", Qty: 1, Amount: 99000}
	if err := repo.AddCartItem(ctx, item2); err != nil {
		t.Fatalf("add cart item2: %v", err)
	}
	if err := cartSvc.Clear(ctx, user.ID); err != nil {
		t.Fatalf("clear cart: %v", err)
	}

	svc := usecase.NewIntegrationService(repo, repo, repo, nil, repo)
	if _, _, err := svc.ListSyncLogs(ctx, "", 10, 0); err != nil {
		t.Fatalf("list sync logs: %v", err)
	}
}
