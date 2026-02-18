package repo_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSQLiteRepo_UniqueConstraints(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := domain.User{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hash",
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	userDup := domain.User{
		Username:     "alice",
		Email:        "alice2@example.com",
		PasswordHash: "hash",
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &userDup); err == nil {
		t.Fatalf("expected duplicate username error")
	}

	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-1",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 1000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	orderDup := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-1",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 2000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(ctx, &orderDup); err == nil {
		t.Fatalf("expected duplicate order_no error")
	}

	payment := domain.OrderPayment{
		OrderID:  order.ID,
		UserID:   user.ID,
		Method:   "manual",
		Amount:   1000,
		Currency: "CNY",
		TradeNo:  "TRADE-1",
		Status:   domain.PaymentStatusPendingReview,
	}
	if err := repo.CreatePayment(ctx, &payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}
	paymentDup := domain.OrderPayment{
		OrderID:  order.ID,
		UserID:   user.ID,
		Method:   "manual",
		Amount:   1000,
		Currency: "CNY",
		TradeNo:  "TRADE-1",
		Status:   domain.PaymentStatusPendingReview,
	}
	if err := repo.CreatePayment(ctx, &paymentDup); err == nil {
		t.Fatalf("expected duplicate trade_no error")
	}

	apiKey := domain.APIKey{Name: "k1", KeyHash: "hash1", Status: domain.APIKeyStatusActive, ScopesJSON: "[]"}
	if err := repo.CreateAPIKey(ctx, &apiKey); err != nil {
		t.Fatalf("create api key: %v", err)
	}
	apiKeyDup := domain.APIKey{Name: "k2", KeyHash: "hash1", Status: domain.APIKeyStatusActive, ScopesJSON: "[]"}
	if err := repo.CreateAPIKey(ctx, &apiKeyDup); err == nil {
		t.Fatalf("expected duplicate key_hash error")
	}
}

func TestSQLiteRepo_AdjustWalletBalance(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := domain.User{
		Username:     "bob",
		Email:        "bob@example.com",
		PasswordHash: "hash",
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, 50, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("credit balance: %v", err)
	}
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, -100, "debit", "order", 2, "charge"); err != shared.ErrInsufficientBalance {
		t.Fatalf("expected insufficient balance, got %v", err)
	}
}
