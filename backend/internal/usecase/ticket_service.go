package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
)

type TicketService struct {
	repo     TicketRepository
	users    UserRepository
	settings SettingsRepository
	messages *MessageCenterService
}

func NewTicketService(repo TicketRepository, users UserRepository, settings SettingsRepository, messages *MessageCenterService) *TicketService {
	return &TicketService{repo: repo, users: users, settings: settings, messages: messages}
}

func (s *TicketService) Create(ctx context.Context, userID int64, subject, content string, resources []domain.TicketResource) (domain.Ticket, []domain.TicketMessage, []domain.TicketResource, error) {
	subject = strings.TrimSpace(subject)
	content = strings.TrimSpace(content)
	if subject == "" || content == "" {
		return domain.Ticket{}, nil, nil, ErrInvalidInput
	}

	// 获取用户信息
	senderName := ""
	senderQQ := ""
	if s.users != nil {
		if user, err := s.users.GetUserByID(ctx, userID); err == nil {
			senderName = user.Username
			senderQQ = user.QQ
		}
	}

	now := time.Now()
	ticket := domain.Ticket{
		UserID:        userID,
		Subject:       subject,
		Status:        "open",
		LastReplyAt:   &now,
		LastReplyBy:   &userID,
		LastReplyRole: "user",
	}
	msg := domain.TicketMessage{
		SenderID:   userID,
		SenderRole: "user",
		SenderName: senderName,
		SenderQQ:   senderQQ,
		Content:    content,
	}
	for i := range resources {
		resources[i].ResourceType = strings.TrimSpace(resources[i].ResourceType)
		resources[i].ResourceName = strings.TrimSpace(resources[i].ResourceName)
	}
	if err := s.repo.CreateTicketWithDetails(ctx, &ticket, &msg, resources); err != nil {
		return domain.Ticket{}, nil, nil, err
	}
	return ticket, []domain.TicketMessage{msg}, resources, nil
}

func (s *TicketService) List(ctx context.Context, filter TicketFilter) ([]domain.Ticket, int, error) {
	return s.repo.ListTickets(ctx, filter)
}

func (s *TicketService) Get(ctx context.Context, id int64) (domain.Ticket, error) {
	return s.repo.GetTicket(ctx, id)
}

func (s *TicketService) GetDetail(ctx context.Context, id int64) (domain.Ticket, []domain.TicketMessage, []domain.TicketResource, error) {
	ticket, err := s.repo.GetTicket(ctx, id)
	if err != nil {
		return domain.Ticket{}, nil, nil, err
	}
	messages, err := s.repo.ListTicketMessages(ctx, id)
	if err != nil {
		return domain.Ticket{}, nil, nil, err
	}
	resources, err := s.repo.ListTicketResources(ctx, id)
	if err != nil {
		return domain.Ticket{}, nil, nil, err
	}
	return ticket, messages, resources, nil
}

func (s *TicketService) AddMessage(ctx context.Context, ticket domain.Ticket, senderID int64, senderRole, content string) (domain.TicketMessage, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return domain.TicketMessage{}, ErrInvalidInput
	}
	if ticket.Status == "closed" && senderRole == "user" {
		return domain.TicketMessage{}, ErrForbidden
	}

	// 获取发送者信息
	// Resolve sender display info.
	senderName := ""
	senderQQ := ""
	if senderRole == "user" && s.users != nil {
		if user, err := s.users.GetUserByID(ctx, senderID); err == nil {
			senderName = user.Username
			senderQQ = user.QQ
		}
	} else if senderRole == "admin" && s.settings != nil {
		if qqSetting, err := s.settings.GetSetting(ctx, "admin_qq"); err == nil && qqSetting.ValueJSON != "" {
			senderQQ = qqSetting.ValueJSON
			senderName = "Technical Support"
		}
	}

	msg := domain.TicketMessage{
		TicketID:   ticket.ID,
		SenderID:   senderID,
		SenderRole: senderRole,
		SenderName: senderName,
		SenderQQ:   senderQQ,
		Content:    content,
	}
	if err := s.repo.AddTicketMessage(ctx, &msg); err != nil {
		return domain.TicketMessage{}, err
	}
	if senderRole == "admin" && s.messages != nil {
		_ = s.messages.NotifyUser(ctx, ticket.UserID, "ticket_reply", "Ticket Reply", "Your ticket #"+fmt.Sprintf("%d", ticket.ID)+" has a new reply.")
	}
	return msg, nil
}

func (s *TicketService) Close(ctx context.Context, ticket domain.Ticket, userID int64) error {
	if ticket.UserID != userID {
		return ErrForbidden
	}
	if ticket.Status == "closed" {
		return nil
	}
	now := time.Now()
	ticket.Status = "closed"
	ticket.ClosedAt = &now
	return s.repo.UpdateTicket(ctx, ticket)
}

func (s *TicketService) AdminUpdate(ctx context.Context, ticket domain.Ticket) error {
	if ticket.Status == "closed" && ticket.ClosedAt == nil {
		now := time.Now()
		ticket.ClosedAt = &now
	}
	if ticket.Status != "closed" {
		ticket.ClosedAt = nil
	}
	return s.repo.UpdateTicket(ctx, ticket)
}

func (s *TicketService) Delete(ctx context.Context, id int64) error {
	return s.repo.DeleteTicket(ctx, id)
}
