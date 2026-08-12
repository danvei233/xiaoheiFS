package push

import (
	"context"
	"strings"
	"testing"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/testutil"
)

func TestRegisterTokenValidation(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "push1", "push1@example.com", "pass")
	svc := NewService(repo, repo, repo, nil)

	if err := svc.RegisterToken(context.Background(), user.ID, "android", strings.Repeat("t", 193), ""); err != appshared.ErrInvalidInput {
		t.Fatalf("expected over-long token rejected, got %v", err)
	}
	if err := svc.RegisterToken(context.Background(), user.ID, "unknown-platform", "tok", ""); err != appshared.ErrInvalidInput {
		t.Fatalf("expected unknown platform rejected, got %v", err)
	}
	if err := svc.RegisterToken(context.Background(), user.ID, "android", "tok", strings.Repeat("d", 193)); err != appshared.ErrInvalidInput {
		t.Fatalf("expected over-long device id rejected, got %v", err)
	}
	if err := svc.RegisterToken(context.Background(), user.ID, "android", "tok", ""); err != nil {
		t.Fatalf("expected valid registration accepted, got %v", err)
	}
}
