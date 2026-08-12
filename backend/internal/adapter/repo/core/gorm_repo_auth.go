package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreateUser(ctx context.Context, user *domain.User) error {

	row := toUserRow(*user)
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now()
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = row.CreatedAt
	}
	create := func(db *gorm.DB) error {
		if err := db.Create(&row).Error; err != nil {
			return err
		}
		*user = fromUserRow(row)
		return nil
	}
	if user.PermissionGroupID != nil {
		// Serialize with group deletion: lock the referenced group so a
		// concurrent DeletePermissionGroupSafe cannot remove it under us and
		// strand a freshly-created user with a dangling reference.
		return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := lockPermissionGroupForWrite(tx, user.PermissionGroupID); err != nil {
				return err
			}
			return create(tx)
		})
	}
	return create(r.gdb.WithContext(ctx))

}

func (r *GormRepo) GetUserByID(ctx context.Context, id int64) (domain.User, error) {

	var row userRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.User{}, r.ensure(err)
	}
	return fromUserRow(row), nil

}

func (r *GormRepo) GetUserByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (domain.User, error) {
	// Deterministic lookup: username wins over email so a single input cannot
	// silently match an arbitrary row when both a user with that username and
	// a different user with that email exist (P2-27).
	var row userRow
	if err := r.gdb.WithContext(ctx).Where("username = ?", usernameOrEmail).First(&row).Error; err == nil {
		return fromUserRow(row), nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, r.ensure(err)
	}
	if err := r.gdb.WithContext(ctx).Where("email = ?", usernameOrEmail).First(&row).Error; err != nil {
		return domain.User{}, r.ensure(err)
	}
	return fromUserRow(row), nil
}

func (r *GormRepo) GetUserByPhone(ctx context.Context, phone string) (domain.User, error) {
	var row userRow
	if err := r.gdb.WithContext(ctx).Where("phone = ?", strings.TrimSpace(phone)).First(&row).Error; err != nil {
		return domain.User{}, r.ensure(err)
	}
	return fromUserRow(row), nil
}

func (r *GormRepo) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, int, error) {
	return r.ListUsersFiltered(ctx, "", "", limit, offset)
}

func (r *GormRepo) ListUsersFiltered(ctx context.Context, keyword, status string, limit, offset int) ([]domain.User, int, error) {
	q := r.gdb.WithContext(ctx).Model(&userRow{})
	// 与前端语义一致：管理端用户列表只展示普通用户，排除管理员。
	q = q.Where("role <> ?", domain.UserRoleAdmin)
	if st := strings.TrimSpace(status); st != "" && st != "all" {
		q = q.Where("status = ?", st)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where(
			"username LIKE ? OR email LIKE ? OR qq LIKE ? OR CAST(id AS TEXT) LIKE ?",
			like, like, like, like,
		)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []userRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.User, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromUserRow(row))
	}
	return out, int(total), nil

}

func (r *GormRepo) ListUsersByRoleStatus(ctx context.Context, role string, status string, limit, offset int) ([]domain.User, int, error) {

	q := r.gdb.WithContext(ctx).Model(&userRow{}).Where("role = ?", role)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []userRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.User, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromUserRow(row))
	}
	return out, int(total), nil

}

func (r *GormRepo) GetMinUserIDByRole(ctx context.Context, role string) (int64, error) {

	var row struct {
		ID int64 `gorm:"column:id"`
	}
	if err := r.gdb.WithContext(ctx).Model(&userRow{}).Select("id").Where("role = ?", role).Order("id ASC").Limit(1).Take(&row).Error; err != nil {
		return 0, r.ensure(err)
	}
	return row.ID, nil

}

func (r *GormRepo) CountUsersByPermissionGroup(ctx context.Context, groupID int64) (int64, error) {
	var count int64
	if err := r.gdb.WithContext(ctx).Model(&userRow{}).Where("permission_group_id = ?", groupID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *GormRepo) UpdateUserStatus(ctx context.Context, id int64, status domain.UserStatus) error {

	return r.gdb.WithContext(ctx).Model(&userRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}).Error

}

// UpdateAdminStatusSafe atomically updates an admin's status while guaranteeing
// the system never ends up with zero active admins. It locks every active admin
// row (FOR UPDATE) so two concurrent disables serialize: the second sees the
// post-first-commit active set and is rejected if it would disable the last one.
func (r *GormRepo) UpdateAdminStatusSafe(ctx context.Context, id int64, status domain.UserStatus) error {
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if status != domain.UserStatusActive {
			var active []userRow
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Model(&userRow{}).
				Where("role = ? AND status = ?", string(domain.UserRoleAdmin), string(domain.UserStatusActive)).
				Find(&active).Error; err != nil {
				return err
			}
			activeCount := len(active)
			targetActive := false
			for _, a := range active {
				if a.ID == id {
					targetActive = true
					break
				}
			}
			if activeCount == 0 || (targetActive && activeCount == 1) {
				return domain.ErrCannotDisableLastActiveAdmin
			}
		}
		return tx.Model(&userRow{}).Where("id = ?", id).Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
	})
}

func (r *GormRepo) UpdateUser(ctx context.Context, user domain.User) error {
	var email any
	if strings.TrimSpace(user.Email) != "" {
		email = strings.TrimSpace(user.Email)
	}
	updates := map[string]any{
		"username":                user.Username,
		"email":                   email,
		"qq":                      user.QQ,
		"avatar":                  user.Avatar,
		"phone":                   user.Phone,
		"last_login_ip":           user.LastLoginIP,
		"last_login_at":           user.LastLoginAt,
		"last_login_city":         user.LastLoginCity,
		"last_login_tz":           user.LastLoginTZ,
		"totp_enabled":            boolToInt(user.TOTPEnabled),
		"totp_secret_enc":         user.TOTPSecretEnc,
		"totp_pending_secret_enc": user.TOTPPendingSecretEnc,
		"bio":                     user.Bio,
		"intro":                   user.Intro,
		"permission_group_id":     user.PermissionGroupID,
		"user_tier_group_id":      user.UserTierGroupID,
		"user_tier_expire_at":     user.UserTierExpireAt,
		"role":                    user.Role,
		"status":                  user.Status,
		"updated_at":              time.Now(),
	}
	if user.PermissionGroupID != nil {
		// Serialize with group deletion (see CreateUser).
		return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := lockPermissionGroupForWrite(tx, user.PermissionGroupID); err != nil {
				return err
			}
			return tx.Model(&userRow{}).Where("id = ?", user.ID).Updates(updates).Error
		})
	}
	return r.gdb.WithContext(ctx).Model(&userRow{}).Where("id = ?", user.ID).Updates(updates).Error

}

func (r *GormRepo) UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error {
	now := time.Now()

	// password_version is a monotonic counter so JWT validation can revoke
	// every previously issued token on a password change without depending on
	// timestamp precision. The increment is expressed in SQL so concurrent
	// changes cannot collide.
	return r.gdb.WithContext(ctx).Model(&userRow{}).Where("id = ?", id).Updates(map[string]any{
		"password_hash":       passwordHash,
		"password_changed_at": now,
		"password_version":    gorm.Expr("password_version + 1"),
		"updated_at":          now,
	}).Error

}

func (r *GormRepo) CreateCaptcha(ctx context.Context, captcha domain.Captcha) error {

	row := captchaRow{
		ID:        captcha.ID,
		CodeHash:  captcha.CodeHash,
		ExpiresAt: captcha.ExpiresAt,
		CreatedAt: captcha.CreatedAt,
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now()
	}
	return r.gdb.WithContext(ctx).Create(&row).Error

}

func (r *GormRepo) GetCaptcha(ctx context.Context, id string) (domain.Captcha, error) {

	var row captchaRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.Captcha{}, r.ensure(err)
	}
	return domain.Captcha{
		ID:        row.ID,
		CodeHash:  row.CodeHash,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil

}

func (r *GormRepo) DeleteCaptcha(ctx context.Context, id string) error {

	return r.gdb.WithContext(ctx).Delete(&captchaRow{}, "id = ?", id).Error

}

func (r *GormRepo) CreateVerificationCode(ctx context.Context, code domain.VerificationCode) error {

	row := verificationCodeRow{
		Channel:   code.Channel,
		Receiver:  code.Receiver,
		Purpose:   code.Purpose,
		CodeHash:  code.CodeHash,
		ExpiresAt: code.ExpiresAt,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	code.ID = row.ID
	return nil

}

func (r *GormRepo) GetLatestVerificationCode(ctx context.Context, channel, receiver, purpose string) (domain.VerificationCode, error) {

	var row verificationCodeRow
	if err := r.gdb.WithContext(ctx).
		Where("channel = ? AND receiver = ? AND purpose = ?", channel, receiver, purpose).
		Order("id DESC").
		Limit(1).
		First(&row).Error; err != nil {
		return domain.VerificationCode{}, rEnsure(err)
	}
	return domain.VerificationCode{
		ID:        row.ID,
		Channel:   row.Channel,
		Receiver:  row.Receiver,
		Purpose:   row.Purpose,
		CodeHash:  row.CodeHash,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil

}

func (r *GormRepo) DeleteVerificationCodes(ctx context.Context, channel, receiver, purpose string) error {

	return r.gdb.WithContext(ctx).Where("channel = ? AND receiver = ? AND purpose = ?", channel, receiver, purpose).Delete(&verificationCodeRow{}).Error

}
