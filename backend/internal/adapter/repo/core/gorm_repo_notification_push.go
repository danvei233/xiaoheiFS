package repo

import (
	"context"
	"time"

	"gorm.io/gorm/clause"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreateNotification(ctx context.Context, notification *domain.Notification) error {
	row := notificationRow{
		UserID:  notification.UserID,
		Type:    notification.Type,
		Title:   notification.Title,
		Content: notification.Content,
		ReadAt:  notification.ReadAt,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	notification.ID = row.ID
	notification.CreatedAt = row.CreatedAt
	return nil
}

func (r *GormRepo) ListNotifications(ctx context.Context, filter appshared.NotificationFilter) ([]domain.Notification, int, error) {
	q := r.gdb.WithContext(ctx).Model(&notificationRow{})
	if filter.UserID != nil {
		q = q.Where("user_id = ?", *filter.UserID)
	}
	switch filter.Status {
	case "unread":
		q = q.Where("read_at IS NULL")
	case "read":
		q = q.Where("read_at IS NOT NULL")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit := filter.Limit
	offset := filter.Offset
	if limit <= 0 {
		limit = 20
	}
	var rows []notificationRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Notification, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Notification{
			ID:        row.ID,
			UserID:    row.UserID,
			Type:      row.Type,
			Title:     row.Title,
			Content:   row.Content,
			ReadAt:    row.ReadAt,
			CreatedAt: row.CreatedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) CountUnread(ctx context.Context, userID int64) (int, error) {
	var total int64
	if err := r.gdb.WithContext(ctx).Model(&notificationRow{}).Where("user_id = ? AND read_at IS NULL", userID).Count(&total).Error; err != nil {
		return 0, err
	}
	return int(total), nil
}

func (r *GormRepo) MarkNotificationRead(ctx context.Context, userID, notificationID int64) error {
	return r.gdb.WithContext(ctx).Model(&notificationRow{}).Where("id = ? AND user_id = ?", notificationID, userID).Update("read_at", time.Now()).Error
}

func (r *GormRepo) MarkAllRead(ctx context.Context, userID int64) error {
	return r.gdb.WithContext(ctx).Model(&notificationRow{}).Where("user_id = ? AND read_at IS NULL", userID).Update("read_at", time.Now()).Error
}

func (r *GormRepo) UpsertPushToken(ctx context.Context, token *domain.PushToken) error {
	if token == nil {
		return nil
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}
	if token.UpdatedAt.IsZero() {
		token.UpdatedAt = time.Now()
	}
	row := pushTokenModel{
		UserID:    token.UserID,
		Platform:  token.Platform,
		Token:     token.Token,
		DeviceID:  token.DeviceID,
		CreatedAt: token.CreatedAt,
		UpdatedAt: token.UpdatedAt,
	}
	return r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "token"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"platform", "device_id", "updated_at",
			}),
		}).
		Create(&row).Error
}

func (r *GormRepo) DeletePushToken(ctx context.Context, userID int64, token string) error {
	return r.gdb.WithContext(ctx).Where("user_id = ? AND token = ?", userID, token).Delete(&pushTokenRow{}).Error
}

func (r *GormRepo) ListPushTokensByUserIDs(ctx context.Context, userIDs []int64) ([]domain.PushToken, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	var rows []pushTokenRow
	if err := r.gdb.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PushToken, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.PushToken{
			ID:        row.ID,
			UserID:    row.UserID,
			Platform:  row.Platform,
			Token:     row.Token,
			DeviceID:  row.DeviceID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}
