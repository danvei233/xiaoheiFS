package repo

import (
	"context"
	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreatePasswordResetToken(ctx context.Context, token *domain.PasswordResetToken) error {

	row := passwordResetTokenRow{
		UserID:    token.UserID,
		Token:     token.Token,
		ExpiresAt: token.ExpiresAt,
		Used:      boolToInt(token.Used),
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	token.ID = row.ID
	token.CreatedAt = row.CreatedAt
	return nil

}

func (r *GormRepo) GetPasswordResetToken(ctx context.Context, token string) (domain.PasswordResetToken, error) {

	var row passwordResetTokenRow
	if err := r.gdb.WithContext(ctx).Where("token = ?", token).First(&row).Error; err != nil {
		return domain.PasswordResetToken{}, r.ensure(err)
	}
	return domain.PasswordResetToken{
		ID:        row.ID,
		UserID:    row.UserID,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		Used:      row.Used == 1,
		CreatedAt: row.CreatedAt,
	}, nil

}

func (r *GormRepo) MarkPasswordResetTokenUsed(ctx context.Context, tokenID int64) error {

	return r.gdb.WithContext(ctx).Model(&passwordResetTokenRow{}).Where("id = ?", tokenID).Update("used", 1).Error

}

func (r *GormRepo) DeleteExpiredTokens(ctx context.Context) error {

	return r.gdb.WithContext(ctx).Where("expires_at < CURRENT_TIMESTAMP").Delete(&passwordResetTokenRow{}).Error

}

func (r *GormRepo) CreatePasswordResetTicket(ctx context.Context, ticket *domain.PasswordResetTicket) error {
	row := passwordResetTicketRow{
		UserID:    ticket.UserID,
		Channel:   ticket.Channel,
		Receiver:  ticket.Receiver,
		Token:     ticket.Token,
		ExpiresAt: ticket.ExpiresAt,
		Used:      boolToInt(ticket.Used),
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	ticket.ID = row.ID
	ticket.CreatedAt = row.CreatedAt
	return nil
}

func (r *GormRepo) GetPasswordResetTicket(ctx context.Context, token string) (domain.PasswordResetTicket, error) {
	var row passwordResetTicketRow
	if err := r.gdb.WithContext(ctx).Where("token = ?", token).First(&row).Error; err != nil {
		return domain.PasswordResetTicket{}, r.ensure(err)
	}
	return domain.PasswordResetTicket{
		ID:        row.ID,
		UserID:    row.UserID,
		Channel:   row.Channel,
		Receiver:  row.Receiver,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		Used:      row.Used == 1,
		CreatedAt: row.CreatedAt,
	}, nil
}

func (r *GormRepo) MarkPasswordResetTicketUsed(ctx context.Context, ticketID int64) error {
	return r.gdb.WithContext(ctx).Model(&passwordResetTicketRow{}).Where("id = ?", ticketID).Update("used", 1).Error
}

func (r *GormRepo) DeleteExpiredPasswordResetTickets(ctx context.Context) error {
	return r.gdb.WithContext(ctx).Where("expires_at < CURRENT_TIMESTAMP").Delete(&passwordResetTicketRow{}).Error
}
