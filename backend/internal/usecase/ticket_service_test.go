package usecase_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestTicketService_CreateAndReply(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "t1", "t1@example.com", "pass")
	msgSvc := usecase.NewMessageCenterService(repo, repo)
	svc := usecase.NewTicketService(repo, repo, repo, msgSvc)

	ticket, messages, _, err := svc.Create(context.Background(), user.ID, "subject", "content", nil)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if ticket.ID == 0 || len(messages) != 1 {
		t.Fatalf("expected ticket and message")
	}

	if _, err := svc.AddMessage(context.Background(), ticket, user.ID, "user", "reply"); err != nil {
		t.Fatalf("add message: %v", err)
	}
}

func TestTicketService_AddMessageClosedForbidden(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "t2", "t2@example.com", "pass")
	svc := usecase.NewTicketService(repo, repo, repo, nil)

	ticket := domain.Ticket{ID: 1, UserID: user.ID, Status: "closed"}
	if _, err := svc.AddMessage(context.Background(), ticket, user.ID, "user", "reply"); err != usecase.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}
