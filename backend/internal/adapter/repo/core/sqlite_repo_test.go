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

// TestSQLiteRepo_PhoneUniqueIndex verifies Finding 4 at the database level: the
// phone_uniq generated column + unique index must reject a second account with
// the same non-empty phone, while accounts without a phone (empty string) are
// unaffected because the generated value is NULL for them.
func TestSQLiteRepo_PhoneUniqueIndex(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	mk := func(username, email, phone string) domain.User {
		return domain.User{
			Username:     username,
			Email:        email,
			Phone:        phone,
			PasswordHash: "hash",
			Role:         domain.UserRoleUser,
			Status:       domain.UserStatusActive,
		}
	}
	a := mk("phone-a", "phone-a@example.com", "13800000123")
	if err := repo.CreateUser(ctx, &a); err != nil {
		t.Fatalf("create phone-a: %v", err)
	}
	// Two accounts with no phone must coexist.
	b := mk("phone-b", "phone-b@example.com", "")
	if err := repo.CreateUser(ctx, &b); err != nil {
		t.Fatalf("create no-phone user: %v", err)
	}
	// Duplicate non-empty phone must be rejected by the DB unique index.
	dup := mk("phone-dup", "phone-dup@example.com", "13800000123")
	if err := repo.CreateUser(ctx, &dup); err == nil {
		t.Fatalf("duplicate phone must be rejected by the unique index")
	}
	// Updating an existing user onto an occupied phone must also fail.
	b.Phone = "13800000123"
	if err := repo.UpdateUser(ctx, b); err == nil {
		t.Fatalf("updating onto an occupied phone must be rejected by the unique index")
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
