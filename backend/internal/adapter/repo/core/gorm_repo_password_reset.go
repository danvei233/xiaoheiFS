package repo

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	appshared "xiaoheiplay/internal/app/shared"
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

// MarkPasswordResetTokenUsedIfUnused atomically claims a token: the update only
// succeeds for a row that is still unused, so concurrent consumers cannot both
// consume the same token (single-use TOCTOU fix).
func (r *GormRepo) MarkPasswordResetTokenUsedIfUnused(ctx context.Context, tokenID int64) (bool, error) {
	res := r.gdb.WithContext(ctx).Model(&passwordResetTokenRow{}).
		Where("id = ? AND used = 0", tokenID).
		Update("used", 1)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *GormRepo) DeleteExpiredTokens(ctx context.Context) error {

	return r.gdb.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&passwordResetTokenRow{}).Error

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

// MarkPasswordResetTicketUsedIfUnused atomically claims a ticket for
// single-use consumption (see MarkPasswordResetTokenUsedIfUnused).
func (r *GormRepo) MarkPasswordResetTicketUsedIfUnused(ctx context.Context, ticketID int64) (bool, error) {
	res := r.gdb.WithContext(ctx).Model(&passwordResetTicketRow{}).
		Where("id = ? AND used = 0", ticketID).
		Update("used", 1)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *GormRepo) DeleteExpiredPasswordResetTickets(ctx context.Context) error {
	return r.gdb.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&passwordResetTicketRow{}).Error
}

// consumePasswordResetTokenInTx claims a single-use reset token and applies the
// password hash to its owner inside the given transaction. It reports whether
// this caller won the claim. A failed password update rolls the claim back, so
// a usable token is never burned by a bcrypt/database error.
func consumePasswordResetTokenInTx(tx *gorm.DB, tokenID, userID int64, passwordHash string, now time.Time) (bool, error) {
	var row passwordResetTokenRow
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", tokenID).First(&row).Error; err != nil {
		return false, rEnsure(err)
	}
	if row.Used == 1 || row.ExpiresAt.Before(now) {
		return false, nil
	}
	if err := tx.Model(&passwordResetTokenRow{}).Where("id = ? AND used = 0", tokenID).Update("used", 1).Error; err != nil {
		return false, err
	}
	res := tx.Model(&userRow{}).Where("id = ?", userID).Updates(map[string]any{
		"password_hash":       passwordHash,
		"password_changed_at": now,
		"password_version":    gorm.Expr("password_version + 1"),
		"updated_at":          now,
	})
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected != 1 {
		// The owner row does not exist: the token must not be consumed.
		return false, appshared.ErrNotFound
	}
	return true, nil
}

// ConsumePasswordResetTokenAndUpdatePassword atomically verifies a reset token
// is unused and unexpired, marks it used and updates the owner's password — all
// in one transaction. A failed password update rolls back the used flag so the
// token stays usable (Issue #51 P2-8).
func (r *GormRepo) ConsumePasswordResetTokenAndUpdatePassword(ctx context.Context, tokenID, userID int64, passwordHash string, now time.Time) (bool, error) {
	var claimed bool
	err := r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		claimed, err = consumePasswordResetTokenInTx(tx, tokenID, userID, passwordHash, now)
		return err
	})
	if err != nil {
		return false, err
	}
	return claimed, nil
}

// ConsumePasswordResetTicketAndUpdatePassword is the ticket-based variant of
// ConsumePasswordResetTokenAndUpdatePassword.
func (r *GormRepo) ConsumePasswordResetTicketAndUpdatePassword(ctx context.Context, ticketID, userID int64, passwordHash string, now time.Time) (bool, error) {
	var claimed bool
	err := r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row passwordResetTicketRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", ticketID).First(&row).Error; err != nil {
			return rEnsure(err)
		}
		if row.Used == 1 || row.ExpiresAt.Before(now) {
			return nil
		}
		if err := tx.Model(&passwordResetTicketRow{}).Where("id = ? AND used = 0", ticketID).Update("used", 1).Error; err != nil {
			return err
		}
		claimed = true
		res := tx.Model(&userRow{}).Where("id = ?", userID).Updates(map[string]any{
			"password_hash":       passwordHash,
			"password_changed_at": now,
			"password_version":    gorm.Expr("password_version + 1"),
			"updated_at":          now,
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return appshared.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return claimed, nil
}

// ConsumeContactChangeTicketAndUpdateContact atomically claims a security
// ticket, re-checks the contact uniqueness constraint and applies the new
// email/phone in a single transaction. The caller reports whether it won the
// claim; a uniqueness violation or update failure rolls back the used flag so a
// concurrent loser cannot complete a sensitive contact change (Issue #51 P2-8).
func (r *GormRepo) ConsumeContactChangeTicketAndUpdateContact(ctx context.Context, ticketID, userID int64, kind, value string, now time.Time) (bool, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	value = strings.TrimSpace(value)
	if value == "" || (kind != "email" && kind != "phone") {
		return false, appshared.ErrInvalidInput
	}
	claimed := false
	err := r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row passwordResetTicketRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", ticketID).First(&row).Error; err != nil {
			return rEnsure(err)
		}
		if row.Used == 1 || row.ExpiresAt.Before(now) {
			return nil
		}
		// Uniqueness check inside the same transaction as the claim and update,
		// so two concurrent requests cannot both pass the check.
		var count int64
		q := tx.Model(&userRow{}).Where("id != ?", userID)
		if kind == "email" {
			q = q.Where("email = ?", value)
		} else {
			q = q.Where("phone = ?", value)
		}
		if err := q.Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			if kind == "email" {
				return domain.ErrEmailAlreadyExists
			}
			return domain.ErrPhoneAlreadyExists
		}
		if err := tx.Model(&passwordResetTicketRow{}).Where("id = ? AND used = 0", ticketID).Update("used", 1).Error; err != nil {
			return err
		}
		claimed = true
		updates := map[string]any{"updated_at": now}
		if kind == "email" {
			updates["email"] = value
		} else {
			updates["phone"] = value
		}
		res := tx.Model(&userRow{}).Where("id = ?", userID).Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return appshared.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return claimed, nil
}
