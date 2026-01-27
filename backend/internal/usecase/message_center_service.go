package usecase

import (
	"context"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
)

type MessageCenterService struct {
	repo  NotificationRepository
	users UserRepository
}

func NewMessageCenterService(repo NotificationRepository, users UserRepository) *MessageCenterService {
	return &MessageCenterService{repo: repo, users: users}
}

func (s *MessageCenterService) List(ctx context.Context, userID int64, status string, limit, offset int) ([]domain.Notification, int, error) {
	filter := NotificationFilter{UserID: &userID, Status: strings.TrimSpace(status), Limit: limit, Offset: offset}
	return s.repo.ListNotifications(ctx, filter)
}

func (s *MessageCenterService) UnreadCount(ctx context.Context, userID int64) (int, error) {
	return s.repo.CountUnread(ctx, userID)
}

func (s *MessageCenterService) MarkRead(ctx context.Context, userID, notificationID int64) error {
	return s.repo.MarkNotificationRead(ctx, userID, notificationID)
}

func (s *MessageCenterService) MarkAllRead(ctx context.Context, userID int64) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *MessageCenterService) NotifyUser(ctx context.Context, userID int64, typ, title, content string) error {
	if userID == 0 {
		return ErrInvalidInput
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Notification"
	}
	notification := domain.Notification{
		UserID:    userID,
		Type:      strings.TrimSpace(typ),
		Title:     title,
		Content:   strings.TrimSpace(content),
		CreatedAt: time.Now(),
	}
	return s.repo.CreateNotification(ctx, &notification)
}

func (s *MessageCenterService) NotifyUsers(ctx context.Context, userIDs []int64, typ, title, content string) error {
	for _, userID := range userIDs {
		_ = s.NotifyUser(ctx, userID, typ, title, content)
	}
	return nil
}

func (s *MessageCenterService) NotifyAllUsers(ctx context.Context, typ, title, content string) error {
	if s.users == nil {
		return nil
	}
	offset := 0
	for {
		users, total, err := s.users.ListUsers(ctx, 200, offset)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			break
		}
		for _, user := range users {
			_ = s.NotifyUser(ctx, user.ID, typ, title, content)
		}
		offset += len(users)
		if offset >= total {
			break
		}
	}
	return nil
}
