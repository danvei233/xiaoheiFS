package repo

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	appshared "xiaoheiplay/internal/app/shared"
)

// RotateRefreshToken serializes refreshes on a family row, consumes the
// presented digest exactly once, and records the successor before it can be
// returned to the caller. A replay revokes the whole family in the same
// transaction, including successors issued by another application replica.
func (r *GormRepo) RotateRefreshToken(ctx context.Context, currentHash, nextHash, familyID string, currentExpiresAt, nextExpiresAt time.Time) (rotated bool, familyRevoked bool, err error) {
	familyID = strings.TrimSpace(familyID)
	now := time.Now()
	if len(currentHash) != 64 || len(nextHash) != 64 || currentHash == nextHash || familyID == "" || len(familyID) > 64 ||
		!currentExpiresAt.After(now) || !nextExpiresAt.After(now) {
		return false, false, appshared.ErrInvalidInput
	}
	if err := r.deleteExpiredRefreshState(ctx, now); err != nil {
		return false, false, err
	}
	err = r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		family, lockErr := lockRefreshTokenFamily(tx, familyID, laterTime(currentExpiresAt, nextExpiresAt))
		if lockErr != nil {
			return lockErr
		}
		if family.RevokedAt != nil {
			familyRevoked = true
			return nil
		}

		var current refreshTokenUseRow
		findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "token_hash = ?", currentHash).Error
		switch {
		case findErr == nil && current.UsedAt != nil:
			return revokeRefreshTokenFamilyRows(tx, familyID, now, nextExpiresAt)
		case findErr == nil:
			if current.FamilyID != familyID {
				return appshared.ErrInvalidInput
			}
			if updateErr := tx.Model(&refreshTokenUseRow{}).Where("token_hash = ? AND used_at IS NULL", currentHash).Update("used_at", now).Error; updateErr != nil {
				return updateErr
			}
		case findErr == gorm.ErrRecordNotFound:
			usedAt := now
			if createErr := tx.Create(&refreshTokenUseRow{
				TokenHash: currentHash,
				FamilyID:  familyID,
				ExpiresAt: currentExpiresAt,
				UsedAt:    &usedAt,
			}).Error; createErr != nil {
				return createErr
			}
		default:
			return findErr
		}

		if createErr := tx.Create(&refreshTokenUseRow{
			TokenHash: nextHash,
			FamilyID:  familyID,
			ExpiresAt: nextExpiresAt,
		}).Error; createErr != nil {
			return createErr
		}
		rotated = true
		return nil
	})
	if err == nil && !rotated && !familyRevoked {
		// revokeRefreshTokenFamilyRows is used as a transaction branch above.
		// Re-read the family result to distinguish replay from storage failure.
		var family refreshTokenFamilyRow
		if readErr := r.gdb.WithContext(ctx).First(&family, "family_id = ?", familyID).Error; readErr != nil {
			return false, false, readErr
		}
		familyRevoked = family.RevokedAt != nil
	}
	return rotated, familyRevoked, err
}

// RevokeRefreshTokenFamily creates a tombstone even when no refresh in the
// family has been consumed yet, which makes logout revoke a freshly issued pair.
func (r *GormRepo) RevokeRefreshTokenFamily(ctx context.Context, familyID, tombstoneHash string, expiresAt time.Time) error {
	familyID = strings.TrimSpace(familyID)
	now := time.Now()
	if familyID == "" || len(familyID) > 64 || len(tombstoneHash) != 64 || !expiresAt.After(now) {
		return appshared.ErrInvalidInput
	}
	if err := r.deleteExpiredRefreshState(ctx, now); err != nil {
		return err
	}
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := lockRefreshTokenFamily(tx, familyID, expiresAt); err != nil {
			return err
		}
		if err := revokeRefreshTokenFamilyRows(tx, familyID, now, expiresAt); err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "token_hash"}}, DoNothing: true}).Create(&refreshTokenUseRow{
			TokenHash: tombstoneHash,
			FamilyID:  familyID,
			ExpiresAt: expiresAt,
			RevokedAt: &now,
			UsedAt:    &now,
		}).Error
	})
}

func (r *GormRepo) deleteExpiredRefreshState(ctx context.Context, now time.Time) error {
	if err := r.gdb.WithContext(ctx).Where("expires_at <= ?", now).Delete(&refreshTokenUseRow{}).Error; err != nil {
		return err
	}
	return r.gdb.WithContext(ctx).Where("expires_at <= ?", now).Delete(&refreshTokenFamilyRow{}).Error
}

func lockRefreshTokenFamily(tx *gorm.DB, familyID string, expiresAt time.Time) (refreshTokenFamilyRow, error) {
	if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "family_id"}}, DoNothing: true}).Create(&refreshTokenFamilyRow{
		FamilyID:  familyID,
		ExpiresAt: expiresAt,
	}).Error; err != nil {
		return refreshTokenFamilyRow{}, err
	}
	var family refreshTokenFamilyRow
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&family, "family_id = ?", familyID).Error; err != nil {
		return refreshTokenFamilyRow{}, err
	}
	if expiresAt.After(family.ExpiresAt) {
		if err := tx.Model(&refreshTokenFamilyRow{}).Where("family_id = ?", familyID).Update("expires_at", expiresAt).Error; err != nil {
			return refreshTokenFamilyRow{}, err
		}
		family.ExpiresAt = expiresAt
	}
	return family, nil
}

func revokeRefreshTokenFamilyRows(tx *gorm.DB, familyID string, now, expiresAt time.Time) error {
	if err := tx.Model(&refreshTokenFamilyRow{}).Where("family_id = ?", familyID).Updates(map[string]any{
		"revoked_at": now,
		"expires_at": expiresAt,
	}).Error; err != nil {
		return err
	}
	return tx.Model(&refreshTokenUseRow{}).Where("family_id = ?", familyID).Update("revoked_at", now).Error
}

func laterTime(left, right time.Time) time.Time {
	if right.After(left) {
		return right
	}
	return left
}
