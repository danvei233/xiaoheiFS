package repo

import (
	"context"
	"time"

	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreateRealNameVerification(ctx context.Context, record *domain.RealNameVerification) error {
	row := realnameVerificationRow{
		UserID:     record.UserID,
		RealName:   record.RealName,
		IDNumber:   record.IDNumber,
		Status:     record.Status,
		Provider:   record.Provider,
		Reason:     record.Reason,
		VerifiedAt: record.VerifiedAt,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	record.ID = row.ID
	record.CreatedAt = row.CreatedAt
	return nil
}

func (r *GormRepo) GetLatestRealNameVerification(ctx context.Context, userID int64) (domain.RealNameVerification, error) {
	var row realnameVerificationRow
	if err := r.gdb.WithContext(ctx).Where("user_id = ?", userID).Order("id DESC").Limit(1).First(&row).Error; err != nil {
		return domain.RealNameVerification{}, r.ensure(err)
	}
	return domain.RealNameVerification{
		ID:         row.ID,
		UserID:     row.UserID,
		RealName:   row.RealName,
		IDNumber:   row.IDNumber,
		Status:     row.Status,
		Provider:   row.Provider,
		Reason:     row.Reason,
		CreatedAt:  row.CreatedAt,
		VerifiedAt: row.VerifiedAt,
	}, nil
}

func (r *GormRepo) ListRealNameVerifications(ctx context.Context, userID *int64, limit, offset int) ([]domain.RealNameVerification, int, error) {
	q := r.gdb.WithContext(ctx).Model(&realnameVerificationRow{})
	if userID != nil {
		q = q.Where("user_id = ?", *userID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	var rows []realnameVerificationRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.RealNameVerification, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.RealNameVerification{
			ID:         row.ID,
			UserID:     row.UserID,
			RealName:   row.RealName,
			IDNumber:   row.IDNumber,
			Status:     row.Status,
			Provider:   row.Provider,
			Reason:     row.Reason,
			CreatedAt:  row.CreatedAt,
			VerifiedAt: row.VerifiedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) UpdateRealNameStatus(ctx context.Context, id int64, status string, reason string, verifiedAt *time.Time) error {
	return r.gdb.WithContext(ctx).Model(&realnameVerificationRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":      status,
		"reason":      reason,
		"verified_at": verifiedAt,
	}).Error
}
