package repo

import (
	"context"
	"strings"
	"time"
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
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*user = fromUserRow(row)
	return nil

}

func (r *GormRepo) GetUserByID(ctx context.Context, id int64) (domain.User, error) {

	var row userRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.User{}, r.ensure(err)
	}
	return fromUserRow(row), nil

}

func (r *GormRepo) GetUserByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (domain.User, error) {

	var row userRow
	if err := r.gdb.WithContext(ctx).Where("username = ? OR email = ?", usernameOrEmail, usernameOrEmail).First(&row).Error; err != nil {
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

	var total int64
	if err := r.gdb.WithContext(ctx).Model(&userRow{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []userRow
	if err := r.gdb.WithContext(ctx).Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
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

func (r *GormRepo) UpdateUserStatus(ctx context.Context, id int64, status domain.UserStatus) error {

	return r.gdb.WithContext(ctx).Model(&userRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) UpdateUser(ctx context.Context, user domain.User) error {
	var email any
	if strings.TrimSpace(user.Email) != "" {
		email = strings.TrimSpace(user.Email)
	}
	return r.gdb.WithContext(ctx).Model(&userRow{}).Where("id = ?", user.ID).Updates(map[string]any{
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
	}).Error

}

func (r *GormRepo) UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error {
	now := time.Now()

	return r.gdb.WithContext(ctx).Model(&userRow{}).Where("id = ?", id).Updates(map[string]any{
		"password_hash":       passwordHash,
		"password_changed_at": now,
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
