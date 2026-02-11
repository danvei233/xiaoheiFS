package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/usecase"
)

type GormRepo struct {
	db      *sql.DB
	gdb     *gorm.DB
	dialect string
}

func NewGormRepo(gdb *gorm.DB) *GormRepo {
	sqlDB, _ := gdb.DB()
	return &GormRepo{db: sqlDB, gdb: gdb, dialect: gdb.Dialector.Name()}
}

func (r *GormRepo) CreateUser(ctx context.Context, user *domain.User) error {
	if r.gdb != nil {
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
	res, err := r.db.ExecContext(ctx, `INSERT INTO users(username,email,qq,avatar,phone,bio,intro,permission_group_id,password_hash,role,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, user.Username, user.Email, user.QQ, user.Avatar, user.Phone, user.Bio, user.Intro, user.PermissionGroupID, user.PasswordHash, user.Role, user.Status)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	user.ID = id
	return nil
}

func (r *GormRepo) GetUserByID(ctx context.Context, id int64) (domain.User, error) {
	if r.gdb != nil {
		var row userRow
		if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return domain.User{}, r.ensure(err)
		}
		return fromUserRow(row), nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, username, email, qq, avatar, phone, bio, intro, permission_group_id, password_hash, role, status, created_at, updated_at FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (r *GormRepo) GetUserByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (domain.User, error) {
	if r.gdb != nil {
		var row userRow
		if err := r.gdb.WithContext(ctx).Where("username = ? OR email = ?", usernameOrEmail, usernameOrEmail).First(&row).Error; err != nil {
			return domain.User{}, r.ensure(err)
		}
		return fromUserRow(row), nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, username, email, qq, avatar, phone, bio, intro, permission_group_id, password_hash, role, status, created_at, updated_at FROM users WHERE username = ? OR email = ?`, usernameOrEmail, usernameOrEmail)
	return scanUser(row)
}

func (r *GormRepo) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, int, error) {
	if r.gdb != nil {
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
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, username, email, qq, avatar, phone, bio, intro, permission_group_id, password_hash, role, status, created_at, updated_at FROM users ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, user)
	}
	return out, total, nil
}

func (r *GormRepo) ListUsersByRoleStatus(ctx context.Context, role string, status string, limit, offset int) ([]domain.User, int, error) {
	if r.gdb != nil {
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
	var total int
	if status == "" {
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users WHERE role = ?`, role).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err := r.db.QueryContext(ctx, `SELECT id, username, email, qq, avatar, phone, bio, intro, permission_group_id, password_hash, role, status, created_at, updated_at FROM users WHERE role = ? ORDER BY id DESC LIMIT ? OFFSET ?`, role, limit, offset)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		var out []domain.User
		for rows.Next() {
			user, err := scanUser(rows)
			if err != nil {
				return nil, 0, err
			}
			out = append(out, user)
		}
		return out, total, nil
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users WHERE role = ? AND status = ?`, role, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, username, email, qq, avatar, phone, bio, intro, permission_group_id, password_hash, role, status, created_at, updated_at FROM users WHERE role = ? AND status = ? ORDER BY id DESC LIMIT ? OFFSET ?`, role, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, user)
	}
	return out, total, nil
}

func (r *GormRepo) GetMinUserIDByRole(ctx context.Context, role string) (int64, error) {
	if r.gdb != nil {
		var row struct {
			ID int64 `gorm:"column:id"`
		}
		if err := r.gdb.WithContext(ctx).Model(&userRow{}).Select("id").Where("role = ?", role).Order("id ASC").Limit(1).Take(&row).Error; err != nil {
			return 0, r.ensure(err)
		}
		return row.ID, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id FROM users WHERE role = ? ORDER BY id ASC LIMIT 1`, role)
	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *GormRepo) UpdateUserStatus(ctx context.Context, id int64, status domain.UserStatus) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&userRow{}).Where("id = ?", id).Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE users SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	return err
}

func (r *GormRepo) UpdateUser(ctx context.Context, user domain.User) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&userRow{}).Where("id = ?", user.ID).Updates(map[string]any{
			"username":            user.Username,
			"email":               user.Email,
			"qq":                  user.QQ,
			"avatar":              user.Avatar,
			"phone":               user.Phone,
			"bio":                 user.Bio,
			"intro":               user.Intro,
			"permission_group_id": user.PermissionGroupID,
			"role":                user.Role,
			"status":              user.Status,
			"updated_at":          time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE users SET username = ?, email = ?, qq = ?, avatar = ?, phone = ?, bio = ?, intro = ?, permission_group_id = ?, role = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, user.Username, user.Email, user.QQ, user.Avatar, user.Phone, user.Bio, user.Intro, user.PermissionGroupID, user.Role, user.Status, user.ID)
	return err
}

func (r *GormRepo) UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&userRow{}).Where("id = ?", id).Updates(map[string]any{
			"password_hash": passwordHash,
			"updated_at":    time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, passwordHash, id)
	return err
}

func (r *GormRepo) CreateCaptcha(ctx context.Context, captcha domain.Captcha) error {
	if r.gdb != nil {
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
	_, err := r.db.ExecContext(ctx, `INSERT INTO captchas(id, code_hash, expires_at) VALUES (?,?,?)`, captcha.ID, captcha.CodeHash, captcha.ExpiresAt)
	return err
}

func (r *GormRepo) GetCaptcha(ctx context.Context, id string) (domain.Captcha, error) {
	if r.gdb != nil {
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
	row := r.db.QueryRowContext(ctx, `SELECT id, code_hash, expires_at, created_at FROM captchas WHERE id = ?`, id)
	var cap domain.Captcha
	if err := row.Scan(&cap.ID, &cap.CodeHash, &cap.ExpiresAt, &cap.CreatedAt); err != nil {
		return domain.Captcha{}, r.ensure(err)
	}
	return cap, nil
}

func (r *GormRepo) DeleteCaptcha(ctx context.Context, id string) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Delete(&captchaRow{}, "id = ?", id).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM captchas WHERE id = ?`, id)
	return err
}

func (r *GormRepo) CreateVerificationCode(ctx context.Context, code domain.VerificationCode) error {
	if r.gdb != nil {
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
	res, err := r.db.ExecContext(ctx, `INSERT INTO verification_codes(channel,receiver,purpose,code_hash,expires_at,created_at)
		VALUES (?,?,?,?,?,CURRENT_TIMESTAMP)`, code.Channel, code.Receiver, code.Purpose, code.CodeHash, code.ExpiresAt)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	code.ID = id
	return nil
}

func (r *GormRepo) GetLatestVerificationCode(ctx context.Context, channel, receiver, purpose string) (domain.VerificationCode, error) {
	if r.gdb != nil {
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
	row := r.db.QueryRowContext(ctx, `SELECT id, channel, receiver, purpose, code_hash, expires_at, created_at
		FROM verification_codes
		WHERE channel = ? AND receiver = ? AND purpose = ?
		ORDER BY id DESC LIMIT 1`, channel, receiver, purpose)
	var out domain.VerificationCode
	if err := row.Scan(&out.ID, &out.Channel, &out.Receiver, &out.Purpose, &out.CodeHash, &out.ExpiresAt, &out.CreatedAt); err != nil {
		return domain.VerificationCode{}, rEnsure(err)
	}
	return out, nil
}

func (r *GormRepo) DeleteVerificationCodes(ctx context.Context, channel, receiver, purpose string) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Where("channel = ? AND receiver = ? AND purpose = ?", channel, receiver, purpose).Delete(&verificationCodeRow{}).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM verification_codes WHERE channel = ? AND receiver = ? AND purpose = ?`, channel, receiver, purpose)
	return err
}

func (r *GormRepo) ListGoodsTypes(ctx context.Context) ([]domain.GoodsType, error) {
	if r.gdb != nil {
		var rows []goodsTypeRow
		if err := r.gdb.WithContext(ctx).Order("sort_order, id").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.GoodsType, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.GoodsType{
				ID:                   row.ID,
				Code:                 row.Code,
				Name:                 row.Name,
				Active:               row.Active == 1,
				SortOrder:            row.SortOrder,
				AutomationCategory:   row.AutomationCategory,
				AutomationPluginID:   row.AutomationPluginID,
				AutomationInstanceID: row.AutomationInstanceID,
				CreatedAt:            row.CreatedAt,
				UpdatedAt:            row.UpdatedAt,
			})
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, code, name, active, sort_order, automation_category, automation_plugin_id, automation_instance_id, created_at, updated_at FROM goods_types ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.GoodsType
	for rows.Next() {
		var gt domain.GoodsType
		var active int
		if err := rows.Scan(&gt.ID, &gt.Code, &gt.Name, &active, &gt.SortOrder, &gt.AutomationCategory, &gt.AutomationPluginID, &gt.AutomationInstanceID, &gt.CreatedAt, &gt.UpdatedAt); err != nil {
			return nil, err
		}
		gt.Active = active == 1
		out = append(out, gt)
	}
	return out, nil
}

func (r *GormRepo) GetGoodsType(ctx context.Context, id int64) (domain.GoodsType, error) {
	if r.gdb != nil {
		var row goodsTypeRow
		if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return domain.GoodsType{}, r.ensure(err)
		}
		return domain.GoodsType{
			ID:                   row.ID,
			Code:                 row.Code,
			Name:                 row.Name,
			Active:               row.Active == 1,
			SortOrder:            row.SortOrder,
			AutomationCategory:   row.AutomationCategory,
			AutomationPluginID:   row.AutomationPluginID,
			AutomationInstanceID: row.AutomationInstanceID,
			CreatedAt:            row.CreatedAt,
			UpdatedAt:            row.UpdatedAt,
		}, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, code, name, active, sort_order, automation_category, automation_plugin_id, automation_instance_id, created_at, updated_at FROM goods_types WHERE id = ?`, id)
	var gt domain.GoodsType
	var active int
	if err := row.Scan(&gt.ID, &gt.Code, &gt.Name, &active, &gt.SortOrder, &gt.AutomationCategory, &gt.AutomationPluginID, &gt.AutomationInstanceID, &gt.CreatedAt, &gt.UpdatedAt); err != nil {
		return domain.GoodsType{}, r.ensure(err)
	}
	gt.Active = active == 1
	return gt, nil
}

func (r *GormRepo) CreateGoodsType(ctx context.Context, gt *domain.GoodsType) error {
	if r.gdb != nil {
		row := goodsTypeRow{
			Code:                 strings.TrimSpace(gt.Code),
			Name:                 gt.Name,
			Active:               boolToInt(gt.Active),
			SortOrder:            gt.SortOrder,
			AutomationCategory:   gt.AutomationCategory,
			AutomationPluginID:   gt.AutomationPluginID,
			AutomationInstanceID: gt.AutomationInstanceID,
		}
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		gt.ID = row.ID
		gt.CreatedAt = row.CreatedAt
		gt.UpdatedAt = row.UpdatedAt
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO goods_types(code,name,active,sort_order,automation_category,automation_plugin_id,automation_instance_id) VALUES (?,?,?,?,?,?,?)`,
		nullIfEmpty(gt.Code), gt.Name, boolToInt(gt.Active), gt.SortOrder, gt.AutomationCategory, gt.AutomationPluginID, gt.AutomationInstanceID)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	gt.ID = id
	return nil
}

func (r *GormRepo) UpdateGoodsType(ctx context.Context, gt domain.GoodsType) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&goodsTypeRow{}).Where("id = ?", gt.ID).Updates(map[string]any{
			"code":                   strings.TrimSpace(gt.Code),
			"name":                   gt.Name,
			"active":                 boolToInt(gt.Active),
			"sort_order":             gt.SortOrder,
			"automation_category":    gt.AutomationCategory,
			"automation_plugin_id":   gt.AutomationPluginID,
			"automation_instance_id": gt.AutomationInstanceID,
			"updated_at":             time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE goods_types SET code = ?, name = ?, active = ?, sort_order = ?, automation_category = ?, automation_plugin_id = ?, automation_instance_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		nullIfEmpty(gt.Code), gt.Name, boolToInt(gt.Active), gt.SortOrder, gt.AutomationCategory, gt.AutomationPluginID, gt.AutomationInstanceID, gt.ID)
	return err
}

func (r *GormRepo) DeleteGoodsType(ctx context.Context, id int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Delete(&goodsTypeRow{}, id).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM goods_types WHERE id = ?`, id)
	return err
}

func (r *GormRepo) ListRegions(ctx context.Context) ([]domain.Region, error) {
	if r.gdb != nil {
		var rows []regionRow
		if err := r.gdb.WithContext(ctx).Order("id").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.Region, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.Region{
				ID:          row.ID,
				GoodsTypeID: row.GoodsTypeID,
				Code:        row.Code,
				Name:        row.Name,
				Active:      row.Active == 1,
			})
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, goods_type_id, code, name, active FROM regions ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Region
	for rows.Next() {
		var region domain.Region
		var active int
		if err := rows.Scan(&region.ID, &region.GoodsTypeID, &region.Code, &region.Name, &active); err != nil {
			return nil, err
		}
		region.Active = active == 1
		out = append(out, region)
	}
	return out, nil
}

func (r *GormRepo) CreateRegion(ctx context.Context, region *domain.Region) error {
	if r.gdb != nil {
		row := regionRow{
			GoodsTypeID: region.GoodsTypeID,
			Code:        region.Code,
			Name:        region.Name,
			Active:      boolToInt(region.Active),
		}
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		region.ID = row.ID
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO regions(goods_type_id,code,name,active) VALUES (?,?,?,?)`, region.GoodsTypeID, region.Code, region.Name, boolToInt(region.Active))
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	region.ID = id
	return nil
}

func (r *GormRepo) UpdateRegion(ctx context.Context, region domain.Region) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&regionRow{}).Where("id = ?", region.ID).Updates(map[string]any{
			"goods_type_id": region.GoodsTypeID,
			"code":          region.Code,
			"name":          region.Name,
			"active":        boolToInt(region.Active),
			"updated_at":    time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE regions SET goods_type_id = ?, code = ?, name = ?, active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, region.GoodsTypeID, region.Code, region.Name, boolToInt(region.Active), region.ID)
	return err
}

func (r *GormRepo) DeleteRegion(ctx context.Context, id int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Delete(&regionRow{}, id).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM regions WHERE id = ?`, id)
	return err
}

func (r *GormRepo) ListPlanGroups(ctx context.Context) ([]domain.PlanGroup, error) {
	if r.gdb != nil {
		var rows []planGroupRow
		if err := r.gdb.WithContext(ctx).Order("sort_order, id").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.PlanGroup, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.PlanGroup{
				ID:                row.ID,
				GoodsTypeID:       row.GoodsTypeID,
				RegionID:          row.RegionID,
				Name:              row.Name,
				LineID:            row.LineID,
				UnitCore:          row.UnitCore,
				UnitMem:           row.UnitMem,
				UnitDisk:          row.UnitDisk,
				UnitBW:            row.UnitBW,
				AddCoreMin:        row.AddCoreMin,
				AddCoreMax:        row.AddCoreMax,
				AddCoreStep:       row.AddCoreStep,
				AddMemMin:         row.AddMemMin,
				AddMemMax:         row.AddMemMax,
				AddMemStep:        row.AddMemStep,
				AddDiskMin:        row.AddDiskMin,
				AddDiskMax:        row.AddDiskMax,
				AddDiskStep:       row.AddDiskStep,
				AddBWMin:          row.AddBWMin,
				AddBWMax:          row.AddBWMax,
				AddBWStep:         row.AddBWStep,
				Active:            row.Active == 1,
				Visible:           row.Visible == 1,
				CapacityRemaining: row.CapacityRemaining,
				SortOrder:         row.SortOrder,
			})
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, goods_type_id, region_id, name, line_id, unit_core, unit_mem, unit_disk, unit_bw, add_core_min, add_core_max, add_core_step, add_mem_min, add_mem_max, add_mem_step, add_disk_min, add_disk_max, add_disk_step, add_bw_min, add_bw_max, add_bw_step, active, visible, capacity_remaining, sort_order FROM plan_groups ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PlanGroup
	for rows.Next() {
		var pg domain.PlanGroup
		var active int
		var visible int
		if err := rows.Scan(&pg.ID, &pg.GoodsTypeID, &pg.RegionID, &pg.Name, &pg.LineID, &pg.UnitCore, &pg.UnitMem, &pg.UnitDisk, &pg.UnitBW, &pg.AddCoreMin, &pg.AddCoreMax, &pg.AddCoreStep, &pg.AddMemMin, &pg.AddMemMax, &pg.AddMemStep, &pg.AddDiskMin, &pg.AddDiskMax, &pg.AddDiskStep, &pg.AddBWMin, &pg.AddBWMax, &pg.AddBWStep, &active, &visible, &pg.CapacityRemaining, &pg.SortOrder); err != nil {
			return nil, err
		}
		pg.Active = active == 1
		pg.Visible = visible == 1
		out = append(out, pg)
	}
	return out, nil
}

func (r *GormRepo) CreatePlanGroup(ctx context.Context, plan *domain.PlanGroup) error {
	if r.gdb != nil {
		row := planGroupRow{
			GoodsTypeID:       plan.GoodsTypeID,
			RegionID:          plan.RegionID,
			Name:              plan.Name,
			LineID:            plan.LineID,
			UnitCore:          plan.UnitCore,
			UnitMem:           plan.UnitMem,
			UnitDisk:          plan.UnitDisk,
			UnitBW:            plan.UnitBW,
			AddCoreMin:        plan.AddCoreMin,
			AddCoreMax:        plan.AddCoreMax,
			AddCoreStep:       plan.AddCoreStep,
			AddMemMin:         plan.AddMemMin,
			AddMemMax:         plan.AddMemMax,
			AddMemStep:        plan.AddMemStep,
			AddDiskMin:        plan.AddDiskMin,
			AddDiskMax:        plan.AddDiskMax,
			AddDiskStep:       plan.AddDiskStep,
			AddBWMin:          plan.AddBWMin,
			AddBWMax:          plan.AddBWMax,
			AddBWStep:         plan.AddBWStep,
			Active:            boolToInt(plan.Active),
			Visible:           boolToInt(plan.Visible),
			CapacityRemaining: plan.CapacityRemaining,
			SortOrder:         plan.SortOrder,
		}
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		plan.ID = row.ID
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO plan_groups(goods_type_id,region_id,name,line_id,unit_core,unit_mem,unit_disk,unit_bw,add_core_min,add_core_max,add_core_step,add_mem_min,add_mem_max,add_mem_step,add_disk_min,add_disk_max,add_disk_step,add_bw_min,add_bw_max,add_bw_step,active,visible,capacity_remaining,sort_order) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, plan.GoodsTypeID, plan.RegionID, plan.Name, plan.LineID, plan.UnitCore, plan.UnitMem, plan.UnitDisk, plan.UnitBW, plan.AddCoreMin, plan.AddCoreMax, plan.AddCoreStep, plan.AddMemMin, plan.AddMemMax, plan.AddMemStep, plan.AddDiskMin, plan.AddDiskMax, plan.AddDiskStep, plan.AddBWMin, plan.AddBWMax, plan.AddBWStep, boolToInt(plan.Active), boolToInt(plan.Visible), plan.CapacityRemaining, plan.SortOrder)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	plan.ID = id
	return nil
}

func (r *GormRepo) UpdatePlanGroup(ctx context.Context, plan domain.PlanGroup) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&planGroupRow{}).Where("id = ?", plan.ID).Updates(map[string]any{
			"goods_type_id":      plan.GoodsTypeID,
			"region_id":          plan.RegionID,
			"name":               plan.Name,
			"line_id":            plan.LineID,
			"unit_core":          plan.UnitCore,
			"unit_mem":           plan.UnitMem,
			"unit_disk":          plan.UnitDisk,
			"unit_bw":            plan.UnitBW,
			"add_core_min":       plan.AddCoreMin,
			"add_core_max":       plan.AddCoreMax,
			"add_core_step":      plan.AddCoreStep,
			"add_mem_min":        plan.AddMemMin,
			"add_mem_max":        plan.AddMemMax,
			"add_mem_step":       plan.AddMemStep,
			"add_disk_min":       plan.AddDiskMin,
			"add_disk_max":       plan.AddDiskMax,
			"add_disk_step":      plan.AddDiskStep,
			"add_bw_min":         plan.AddBWMin,
			"add_bw_max":         plan.AddBWMax,
			"add_bw_step":        plan.AddBWStep,
			"active":             boolToInt(plan.Active),
			"visible":            boolToInt(plan.Visible),
			"capacity_remaining": plan.CapacityRemaining,
			"sort_order":         plan.SortOrder,
			"updated_at":         time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE plan_groups SET goods_type_id = ?, region_id = ?, name = ?, line_id = ?, unit_core = ?, unit_mem = ?, unit_disk = ?, unit_bw = ?, add_core_min = ?, add_core_max = ?, add_core_step = ?, add_mem_min = ?, add_mem_max = ?, add_mem_step = ?, add_disk_min = ?, add_disk_max = ?, add_disk_step = ?, add_bw_min = ?, add_bw_max = ?, add_bw_step = ?, active = ?, visible = ?, capacity_remaining = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, plan.GoodsTypeID, plan.RegionID, plan.Name, plan.LineID, plan.UnitCore, plan.UnitMem, plan.UnitDisk, plan.UnitBW, plan.AddCoreMin, plan.AddCoreMax, plan.AddCoreStep, plan.AddMemMin, plan.AddMemMax, plan.AddMemStep, plan.AddDiskMin, plan.AddDiskMax, plan.AddDiskStep, plan.AddBWMin, plan.AddBWMax, plan.AddBWStep, boolToInt(plan.Active), boolToInt(plan.Visible), plan.CapacityRemaining, plan.SortOrder, plan.ID)
	return err
}

func (r *GormRepo) DeletePlanGroup(ctx context.Context, id int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Delete(&planGroupRow{}, id).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM plan_groups WHERE id = ?`, id)
	return err
}

func (r *GormRepo) ListPackages(ctx context.Context) ([]domain.Package, error) {
	if r.gdb != nil {
		var rows []packageRow
		if err := r.gdb.WithContext(ctx).Order("sort_order, id").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.Package, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.Package{
				ID:                row.ID,
				GoodsTypeID:       row.GoodsTypeID,
				PlanGroupID:       row.PlanGroupID,
				ProductID:         row.ProductID,
				Name:              row.Name,
				Cores:             row.Cores,
				MemoryGB:          row.MemoryGB,
				DiskGB:            row.DiskGB,
				BandwidthMB:       row.BandwidthMbps,
				CPUModel:          row.CPUModel,
				Monthly:           row.MonthlyPrice,
				PortNum:           row.PortNum,
				SortOrder:         row.SortOrder,
				Active:            row.Active == 1,
				Visible:           row.Visible == 1,
				CapacityRemaining: row.CapacityRemaining,
			})
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, goods_type_id, plan_group_id, product_id, name, cores, memory_gb, disk_gb, bandwidth_mbps, cpu_model, monthly_price, port_num, sort_order, active, visible, capacity_remaining FROM packages ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Package
	for rows.Next() {
		var pkg domain.Package
		var active int
		var visible int
		if err := rows.Scan(&pkg.ID, &pkg.GoodsTypeID, &pkg.PlanGroupID, &pkg.ProductID, &pkg.Name, &pkg.Cores, &pkg.MemoryGB, &pkg.DiskGB, &pkg.BandwidthMB, &pkg.CPUModel, &pkg.Monthly, &pkg.PortNum, &pkg.SortOrder, &active, &visible, &pkg.CapacityRemaining); err != nil {
			return nil, err
		}
		pkg.Active = active == 1
		pkg.Visible = visible == 1
		out = append(out, pkg)
	}
	return out, nil
}

func (r *GormRepo) CreatePackage(ctx context.Context, pkg *domain.Package) error {
	if r.gdb != nil {
		row := packageRow{
			GoodsTypeID:       pkg.GoodsTypeID,
			PlanGroupID:       pkg.PlanGroupID,
			ProductID:         pkg.ProductID,
			Name:              pkg.Name,
			Cores:             pkg.Cores,
			MemoryGB:          pkg.MemoryGB,
			DiskGB:            pkg.DiskGB,
			BandwidthMbps:     pkg.BandwidthMB,
			CPUModel:          pkg.CPUModel,
			MonthlyPrice:      pkg.Monthly,
			PortNum:           pkg.PortNum,
			SortOrder:         pkg.SortOrder,
			Active:            boolToInt(pkg.Active),
			Visible:           boolToInt(pkg.Visible),
			CapacityRemaining: pkg.CapacityRemaining,
		}
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		pkg.ID = row.ID
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO packages(goods_type_id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, pkg.GoodsTypeID, pkg.PlanGroupID, pkg.ProductID, pkg.Name, pkg.Cores, pkg.MemoryGB, pkg.DiskGB, pkg.BandwidthMB, pkg.CPUModel, pkg.Monthly, pkg.PortNum, pkg.SortOrder, boolToInt(pkg.Active), boolToInt(pkg.Visible), pkg.CapacityRemaining)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	pkg.ID = id
	return nil
}

func (r *GormRepo) UpdatePackage(ctx context.Context, pkg domain.Package) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&packageRow{}).Where("id = ?", pkg.ID).Updates(map[string]any{
			"goods_type_id":      pkg.GoodsTypeID,
			"plan_group_id":      pkg.PlanGroupID,
			"product_id":         pkg.ProductID,
			"name":               pkg.Name,
			"cores":              pkg.Cores,
			"memory_gb":          pkg.MemoryGB,
			"disk_gb":            pkg.DiskGB,
			"bandwidth_mbps":     pkg.BandwidthMB,
			"cpu_model":          pkg.CPUModel,
			"monthly_price":      pkg.Monthly,
			"port_num":           pkg.PortNum,
			"sort_order":         pkg.SortOrder,
			"active":             boolToInt(pkg.Active),
			"visible":            boolToInt(pkg.Visible),
			"capacity_remaining": pkg.CapacityRemaining,
			"updated_at":         time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE packages SET goods_type_id = ?, plan_group_id = ?, product_id = ?, name = ?, cores = ?, memory_gb = ?, disk_gb = ?, bandwidth_mbps = ?, cpu_model = ?, monthly_price = ?, port_num = ?, sort_order = ?, active = ?, visible = ?, capacity_remaining = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, pkg.GoodsTypeID, pkg.PlanGroupID, pkg.ProductID, pkg.Name, pkg.Cores, pkg.MemoryGB, pkg.DiskGB, pkg.BandwidthMB, pkg.CPUModel, pkg.Monthly, pkg.PortNum, pkg.SortOrder, boolToInt(pkg.Active), boolToInt(pkg.Visible), pkg.CapacityRemaining, pkg.ID)
	return err
}

func (r *GormRepo) DeletePackage(ctx context.Context, id int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Delete(&packageRow{}, id).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM packages WHERE id = ?`, id)
	return err
}

func (r *GormRepo) GetPackage(ctx context.Context, id int64) (domain.Package, error) {
	if r.gdb != nil {
		var row packageRow
		if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return domain.Package{}, r.ensure(err)
		}
		return domain.Package{
			ID:                row.ID,
			GoodsTypeID:       row.GoodsTypeID,
			PlanGroupID:       row.PlanGroupID,
			ProductID:         row.ProductID,
			Name:              row.Name,
			Cores:             row.Cores,
			MemoryGB:          row.MemoryGB,
			DiskGB:            row.DiskGB,
			BandwidthMB:       row.BandwidthMbps,
			CPUModel:          row.CPUModel,
			Monthly:           row.MonthlyPrice,
			PortNum:           row.PortNum,
			SortOrder:         row.SortOrder,
			Active:            row.Active == 1,
			Visible:           row.Visible == 1,
			CapacityRemaining: row.CapacityRemaining,
		}, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, goods_type_id, plan_group_id, product_id, name, cores, memory_gb, disk_gb, bandwidth_mbps, cpu_model, monthly_price, port_num, sort_order, active, visible, capacity_remaining FROM packages WHERE id = ?`, id)
	var pkg domain.Package
	var active int
	var visible int
	if err := row.Scan(&pkg.ID, &pkg.GoodsTypeID, &pkg.PlanGroupID, &pkg.ProductID, &pkg.Name, &pkg.Cores, &pkg.MemoryGB, &pkg.DiskGB, &pkg.BandwidthMB, &pkg.CPUModel, &pkg.Monthly, &pkg.PortNum, &pkg.SortOrder, &active, &visible, &pkg.CapacityRemaining); err != nil {
		return domain.Package{}, r.ensure(err)
	}
	pkg.Active = active == 1
	pkg.Visible = visible == 1
	return pkg, nil
}

func (r *GormRepo) GetPlanGroup(ctx context.Context, id int64) (domain.PlanGroup, error) {
	if r.gdb != nil {
		var row planGroupRow
		if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return domain.PlanGroup{}, r.ensure(err)
		}
		return domain.PlanGroup{
			ID:                row.ID,
			GoodsTypeID:       row.GoodsTypeID,
			RegionID:          row.RegionID,
			Name:              row.Name,
			LineID:            row.LineID,
			UnitCore:          row.UnitCore,
			UnitMem:           row.UnitMem,
			UnitDisk:          row.UnitDisk,
			UnitBW:            row.UnitBW,
			AddCoreMin:        row.AddCoreMin,
			AddCoreMax:        row.AddCoreMax,
			AddCoreStep:       row.AddCoreStep,
			AddMemMin:         row.AddMemMin,
			AddMemMax:         row.AddMemMax,
			AddMemStep:        row.AddMemStep,
			AddDiskMin:        row.AddDiskMin,
			AddDiskMax:        row.AddDiskMax,
			AddDiskStep:       row.AddDiskStep,
			AddBWMin:          row.AddBWMin,
			AddBWMax:          row.AddBWMax,
			AddBWStep:         row.AddBWStep,
			Active:            row.Active == 1,
			Visible:           row.Visible == 1,
			CapacityRemaining: row.CapacityRemaining,
			SortOrder:         row.SortOrder,
		}, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, goods_type_id, region_id, name, line_id, unit_core, unit_mem, unit_disk, unit_bw, add_core_min, add_core_max, add_core_step, add_mem_min, add_mem_max, add_mem_step, add_disk_min, add_disk_max, add_disk_step, add_bw_min, add_bw_max, add_bw_step, active, visible, capacity_remaining, sort_order FROM plan_groups WHERE id = ?`, id)
	var pg domain.PlanGroup
	var active int
	var visible int
	if err := row.Scan(&pg.ID, &pg.GoodsTypeID, &pg.RegionID, &pg.Name, &pg.LineID, &pg.UnitCore, &pg.UnitMem, &pg.UnitDisk, &pg.UnitBW, &pg.AddCoreMin, &pg.AddCoreMax, &pg.AddCoreStep, &pg.AddMemMin, &pg.AddMemMax, &pg.AddMemStep, &pg.AddDiskMin, &pg.AddDiskMax, &pg.AddDiskStep, &pg.AddBWMin, &pg.AddBWMax, &pg.AddBWStep, &active, &visible, &pg.CapacityRemaining, &pg.SortOrder); err != nil {
		return domain.PlanGroup{}, r.ensure(err)
	}
	pg.Active = active == 1
	pg.Visible = visible == 1
	return pg, nil
}

func (r *GormRepo) GetRegion(ctx context.Context, id int64) (domain.Region, error) {
	if r.gdb != nil {
		var row regionRow
		if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return domain.Region{}, r.ensure(err)
		}
		return domain.Region{
			ID:          row.ID,
			GoodsTypeID: row.GoodsTypeID,
			Code:        row.Code,
			Name:        row.Name,
			Active:      row.Active == 1,
		}, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, goods_type_id, code, name, active FROM regions WHERE id = ?`, id)
	var region domain.Region
	var active int
	if err := row.Scan(&region.ID, &region.GoodsTypeID, &region.Code, &region.Name, &active); err != nil {
		return domain.Region{}, r.ensure(err)
	}
	region.Active = active == 1
	return region, nil
}

func (r *GormRepo) ListSystemImages(ctx context.Context, lineID int64) ([]domain.SystemImage, error) {
	if r.gdb != nil {
		var rows []systemImageRow
		if err := r.gdb.WithContext(ctx).
			Table("system_images si").
			Select("si.id, si.image_id, si.name, si.type, si.enabled, si.created_at, si.updated_at").
			Joins("JOIN line_system_images lsi ON lsi.system_image_id = si.id").
			Where("lsi.line_id = ? AND si.enabled = 1", lineID).
			Order("si.id DESC").
			Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.SystemImage, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.SystemImage{
				ID:        row.ID,
				ImageID:   row.ImageID,
				Name:      row.Name,
				Type:      row.Type,
				Enabled:   row.Enabled == 1,
				CreatedAt: row.CreatedAt,
				UpdatedAt: row.UpdatedAt,
			})
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT si.id, si.image_id, si.name, si.type, si.enabled, si.created_at, si.updated_at FROM system_images si JOIN line_system_images lsi ON lsi.system_image_id = si.id WHERE lsi.line_id = ? AND si.enabled = 1 ORDER BY si.id DESC`, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SystemImage
	for rows.Next() {
		img, err := scanSystemImage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	return out, nil
}

func (r *GormRepo) ListAllSystemImages(ctx context.Context) ([]domain.SystemImage, error) {
	if r.gdb != nil {
		var rows []systemImageRow
		if err := r.gdb.WithContext(ctx).Order("id DESC").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.SystemImage, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.SystemImage{
				ID:        row.ID,
				ImageID:   row.ImageID,
				Name:      row.Name,
				Type:      row.Type,
				Enabled:   row.Enabled == 1,
				CreatedAt: row.CreatedAt,
				UpdatedAt: row.UpdatedAt,
			})
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, image_id, name, type, enabled, created_at, updated_at FROM system_images ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SystemImage
	for rows.Next() {
		img, err := scanSystemImage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	return out, nil
}

func (r *GormRepo) GetSystemImage(ctx context.Context, id int64) (domain.SystemImage, error) {
	if r.gdb != nil {
		var row systemImageRow
		if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return domain.SystemImage{}, r.ensure(err)
		}
		return domain.SystemImage{
			ID:        row.ID,
			ImageID:   row.ImageID,
			Name:      row.Name,
			Type:      row.Type,
			Enabled:   row.Enabled == 1,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, image_id, name, type, enabled, created_at, updated_at FROM system_images WHERE id = ?`, id)
	return scanSystemImage(row)
}

func (r *GormRepo) CreateSystemImage(ctx context.Context, img *domain.SystemImage) error {
	if r.gdb != nil {
		row := systemImageRow{
			ImageID: img.ImageID,
			Name:    img.Name,
			Type:    img.Type,
			Enabled: boolToInt(img.Enabled),
		}
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		img.ID = row.ID
		img.CreatedAt = row.CreatedAt
		img.UpdatedAt = row.UpdatedAt
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO system_images(image_id,name,type,enabled) VALUES (?,?,?,?)`, img.ImageID, img.Name, img.Type, boolToInt(img.Enabled))
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	img.ID = id
	return nil
}

func (r *GormRepo) UpdateSystemImage(ctx context.Context, img domain.SystemImage) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&systemImageRow{}).Where("id = ?", img.ID).Updates(map[string]any{
			"image_id":   img.ImageID,
			"name":       img.Name,
			"type":       img.Type,
			"enabled":    boolToInt(img.Enabled),
			"updated_at": time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE system_images SET image_id = ?, name = ?, type = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, img.ImageID, img.Name, img.Type, boolToInt(img.Enabled), img.ID)
	return err
}

func (r *GormRepo) DeleteSystemImage(ctx context.Context, id int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("system_image_id = ?", id).Delete(&lineSystemImageRow{}).Error; err != nil {
				return err
			}
			return tx.Delete(&systemImageRow{}, id).Error
		})
	}
	_, _ = r.db.ExecContext(ctx, `DELETE FROM line_system_images WHERE system_image_id = ?`, id)
	_, err := r.db.ExecContext(ctx, `DELETE FROM system_images WHERE id = ?`, id)
	return err
}

func (r *GormRepo) SetLineSystemImages(ctx context.Context, lineID int64, systemImageIDs []int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("line_id = ?", lineID).Delete(&lineSystemImageRow{}).Error; err != nil {
				return err
			}
			seen := map[int64]struct{}{}
			rows := make([]lineSystemImageRow, 0, len(systemImageIDs))
			for _, id := range systemImageIDs {
				if id <= 0 {
					continue
				}
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}
				rows = append(rows, lineSystemImageRow{LineID: lineID, SystemImageID: id})
			}
			if len(rows) > 0 {
				if err := tx.Create(&rows).Error; err != nil {
					return err
				}
			}
			return nil
		})
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM line_system_images WHERE line_id = ?`, lineID); err != nil {
		return err
	}
	if len(systemImageIDs) > 0 {
		stmt, err := tx.PrepareContext(ctx, `INSERT INTO line_system_images(line_id, system_image_id) VALUES (?,?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		seen := map[int64]struct{}{}
		for _, id := range systemImageIDs {
			if id <= 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			if _, err := stmt.ExecContext(ctx, lineID, id); err != nil {
				return err
			}
			seen[id] = struct{}{}
		}
	}
	return tx.Commit()
}

func (r *GormRepo) ListCartItems(ctx context.Context, userID int64) ([]domain.CartItem, error) {
	if r.gdb != nil {
		var rows []cartItemRow
		if err := r.gdb.WithContext(ctx).Where("user_id = ?", userID).Order("id DESC").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.CartItem, 0, len(rows))
		for _, row := range rows {
			out = append(out, fromCartItemRow(row))
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, package_id, system_id, spec_json, qty, amount, created_at, updated_at FROM cart_items WHERE user_id = ? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CartItem
	for rows.Next() {
		item, err := scanCartItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *GormRepo) AddCartItem(ctx context.Context, item *domain.CartItem) error {
	if r.gdb != nil {
		row := toCartItemRow(*item)
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		*item = fromCartItemRow(row)
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO cart_items(user_id,package_id,system_id,spec_json,qty,amount) VALUES (?,?,?,?,?,?)`, item.UserID, item.PackageID, item.SystemID, item.SpecJSON, item.Qty, item.Amount)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	item.ID = id
	return nil
}

func (r *GormRepo) UpdateCartItem(ctx context.Context, item domain.CartItem) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&cartItemRow{}).
			Where("id = ? AND user_id = ?", item.ID, item.UserID).
			Updates(map[string]any{"spec_json": item.SpecJSON, "qty": item.Qty, "amount": item.Amount, "updated_at": time.Now()}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE cart_items SET spec_json = ?, qty = ?, amount = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`, item.SpecJSON, item.Qty, item.Amount, item.ID, item.UserID)
	return err
}

func (r *GormRepo) DeleteCartItem(ctx context.Context, id int64, userID int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&cartItemRow{}).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM cart_items WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

func (r *GormRepo) ClearCart(ctx context.Context, userID int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Where("user_id = ?", userID).Delete(&cartItemRow{}).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM cart_items WHERE user_id = ?`, userID)
	return err
}

func (r *GormRepo) CreateOrder(ctx context.Context, order *domain.Order) error {
	if r.gdb != nil {
		row := toOrderRow(*order)
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		*order = fromOrderRow(row)
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO orders(user_id,order_no,status,total_amount,currency,idempotency_key,pending_reason,approved_by,approved_at,rejected_reason) VALUES (?,?,?,?,?,?,?,?,?,?)`, order.UserID, order.OrderNo, order.Status, order.TotalAmount, order.Currency, nullIfEmpty(order.IdempotencyKey), order.PendingReason, order.ApprovedBy, order.ApprovedAt, order.RejectedReason)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	order.ID = id
	return nil
}

func (r *GormRepo) CreateOrderFromCartAtomic(ctx context.Context, order domain.Order, items []domain.OrderItem) (created domain.Order, createdItems []domain.OrderItem, err error) {
	if r.gdb != nil {
		tx := r.gdb.WithContext(ctx).Begin()
		if tx.Error != nil {
			return domain.Order{}, nil, tx.Error
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback().Error
			}
		}()

		or := toOrderRow(order)
		if err = tx.Create(&or).Error; err != nil {
			return domain.Order{}, nil, err
		}
		order = fromOrderRow(or)

		itemRows := make([]orderItemRow, 0, len(items))
		for i := range items {
			items[i].OrderID = order.ID
			itemRows = append(itemRows, toOrderItemRow(items[i]))
		}
		if len(itemRows) > 0 {
			if err = tx.Create(&itemRows).Error; err != nil {
				return domain.Order{}, nil, err
			}
			createdItems = make([]domain.OrderItem, 0, len(itemRows))
			for _, row := range itemRows {
				createdItems = append(createdItems, fromOrderItemRow(row))
			}
		}

		if err = tx.Where("user_id = ?", order.UserID).Delete(&cartItemRow{}).Error; err != nil {
			return domain.Order{}, nil, err
		}
		if err = tx.Commit().Error; err != nil {
			return domain.Order{}, nil, err
		}
		return order, createdItems, nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Order{}, nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.ExecContext(ctx, `INSERT INTO orders(user_id,order_no,status,total_amount,currency,idempotency_key,pending_reason,approved_by,approved_at,rejected_reason) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		order.UserID, order.OrderNo, order.Status, order.TotalAmount, order.Currency, nullIfEmpty(order.IdempotencyKey), order.PendingReason, order.ApprovedBy, order.ApprovedAt, order.RejectedReason)
	if err != nil {
		return domain.Order{}, nil, err
	}
	id, _ := res.LastInsertId()
	order.ID = id

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO order_items(order_id,package_id,system_id,spec_json,qty,amount,status,goods_type_id,automation_instance_id,action,duration_months) VALUES (?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return domain.Order{}, nil, err
	}
	defer stmt.Close()
	for i := range items {
		items[i].OrderID = order.ID
		res, err := stmt.ExecContext(ctx, items[i].OrderID, items[i].PackageID, items[i].SystemID, items[i].SpecJSON, items[i].Qty, items[i].Amount, items[i].Status, items[i].GoodsTypeID, items[i].AutomationInstanceID, items[i].Action, items[i].DurationMonths)
		if err != nil {
			return domain.Order{}, nil, err
		}
		itemID, _ := res.LastInsertId()
		items[i].ID = itemID
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM cart_items WHERE user_id = ?`, order.UserID); err != nil {
		return domain.Order{}, nil, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Order{}, nil, err
	}
	return order, items, nil
}

func (r *GormRepo) GetOrder(ctx context.Context, id int64) (domain.Order, error) {
	if r.gdb != nil {
		var row orderRow
		if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return domain.Order{}, r.ensure(err)
		}
		return fromOrderRow(row), nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, order_no, status, total_amount, currency, idempotency_key, pending_reason, approved_by, approved_at, rejected_reason, created_at, updated_at FROM orders WHERE id = ?`, id)
	return scanOrder(row)
}

func (r *GormRepo) GetOrderByNo(ctx context.Context, orderNo string) (domain.Order, error) {
	if r.gdb != nil {
		var row orderRow
		if err := r.gdb.WithContext(ctx).Where("order_no = ?", orderNo).First(&row).Error; err != nil {
			return domain.Order{}, r.ensure(err)
		}
		return fromOrderRow(row), nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, order_no, status, total_amount, currency, idempotency_key, pending_reason, approved_by, approved_at, rejected_reason, created_at, updated_at FROM orders WHERE order_no = ?`, orderNo)
	return scanOrder(row)
}

func (r *GormRepo) GetOrderByIdempotencyKey(ctx context.Context, userID int64, key string) (domain.Order, error) {
	if r.gdb != nil {
		var row orderRow
		if err := r.gdb.WithContext(ctx).Where("user_id = ? AND idempotency_key = ?", userID, key).First(&row).Error; err != nil {
			return domain.Order{}, r.ensure(err)
		}
		return fromOrderRow(row), nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, order_no, status, total_amount, currency, idempotency_key, pending_reason, approved_by, approved_at, rejected_reason, created_at, updated_at FROM orders WHERE user_id = ? AND idempotency_key = ?`, userID, key)
	return scanOrder(row)
}

func (r *GormRepo) UpdateOrderStatus(ctx context.Context, id int64, status domain.OrderStatus) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&orderRow{}).Where("id = ?", id).Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE orders SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	return err
}

func (r *GormRepo) UpdateOrderMeta(ctx context.Context, order domain.Order) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&orderRow{}).Where("id = ?", order.ID).Updates(map[string]any{
			"status":          order.Status,
			"pending_reason":  order.PendingReason,
			"approved_by":     order.ApprovedBy,
			"approved_at":     order.ApprovedAt,
			"rejected_reason": order.RejectedReason,
			"updated_at":      time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE orders SET status = ?, pending_reason = ?, approved_by = ?, approved_at = ?, rejected_reason = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, order.Status, order.PendingReason, order.ApprovedBy, order.ApprovedAt, order.RejectedReason, order.ID)
	return err
}

func (r *GormRepo) ApproveResizeOrderWithTasks(ctx context.Context, order domain.Order, items []domain.OrderItem, tasks []*domain.ResizeTask) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, task := range tasks {
				if task == nil {
					continue
				}
				var total int64
				if err := tx.Model(&resizeTaskRow{}).
					Where("vps_id = ? AND status IN ?", task.VPSID, []string{string(domain.ResizeTaskStatusPending), string(domain.ResizeTaskStatusRunning)}).
					Count(&total).Error; err != nil {
					return err
				}
				if total > 0 {
					return usecase.ErrResizeInProgress
				}
			}

			if err := tx.Model(&orderRow{}).Where("id = ?", order.ID).Updates(map[string]any{
				"status":          order.Status,
				"pending_reason":  order.PendingReason,
				"approved_by":     order.ApprovedBy,
				"approved_at":     order.ApprovedAt,
				"rejected_reason": order.RejectedReason,
				"updated_at":      time.Now(),
			}).Error; err != nil {
				return err
			}

			itemIDs := make([]int64, 0, len(items))
			for _, item := range items {
				itemIDs = append(itemIDs, item.ID)
			}
			if len(itemIDs) > 0 {
				if err := tx.Model(&orderItemRow{}).Where("id IN ?", itemIDs).Updates(map[string]any{
					"status":     domain.OrderItemStatusApproved,
					"updated_at": time.Now(),
				}).Error; err != nil {
					return err
				}
			}

			for _, task := range tasks {
				if task == nil {
					continue
				}
				row := resizeTaskRow{
					VPSID:       task.VPSID,
					OrderID:     task.OrderID,
					OrderItemID: task.OrderItemID,
					Status:      string(task.Status),
					ScheduledAt: task.ScheduledAt,
					StartedAt:   task.StartedAt,
					FinishedAt:  task.FinishedAt,
				}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
				task.ID = row.ID
			}
			return nil
		})
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, task := range tasks {
		if task == nil {
			continue
		}
		var total int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM resize_tasks WHERE vps_id = ? AND status IN (?, ?)`, task.VPSID, domain.ResizeTaskStatusPending, domain.ResizeTaskStatusRunning).Scan(&total); err != nil {
			return err
		}
		if total > 0 {
			return usecase.ErrResizeInProgress
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE orders SET status = ?, pending_reason = ?, approved_by = ?, approved_at = ?, rejected_reason = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		order.Status, order.PendingReason, order.ApprovedBy, order.ApprovedAt, order.RejectedReason, order.ID); err != nil {
		return err
	}
	for _, item := range items {
		if _, err := tx.ExecContext(ctx, `UPDATE order_items SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, domain.OrderItemStatusApproved, item.ID); err != nil {
			return err
		}
	}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO resize_tasks(vps_id,order_id,order_item_id,status,scheduled_at,started_at,finished_at) VALUES (?,?,?,?,?,?,?)`,
			task.VPSID, task.OrderID, task.OrderItemID, task.Status, task.ScheduledAt, task.StartedAt, task.FinishedAt)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		task.ID = id
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *GormRepo) ListOrders(ctx context.Context, filter usecase.OrderFilter, limit, offset int) ([]domain.Order, int, error) {
	if r.gdb != nil {
		q := r.gdb.WithContext(ctx).Model(&orderRow{})
		if filter.Status != "" {
			q = q.Where("status = ?", filter.Status)
		}
		if filter.UserID > 0 {
			q = q.Where("user_id = ?", filter.UserID)
		}
		if filter.From != nil {
			q = q.Where("created_at >= ?", filter.From)
		}
		if filter.To != nil {
			q = q.Where("created_at <= ?", filter.To)
		}
		var total int64
		if err := q.Count(&total).Error; err != nil {
			return nil, 0, err
		}
		var rows []orderRow
		if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, 0, err
		}
		out := make([]domain.Order, 0, len(rows))
		for _, row := range rows {
			out = append(out, fromOrderRow(row))
		}
		return out, int(total), nil
	}
	query := `SELECT id, user_id, order_no, status, total_amount, currency, idempotency_key, pending_reason, approved_by, approved_at, rejected_reason, created_at, updated_at FROM orders WHERE 1=1`
	args := []any{}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.UserID > 0 {
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}
	if filter.From != nil {
		query += " AND created_at >= ?"
		args = append(args, filter.From)
	}
	if filter.To != nil {
		query += " AND created_at <= ?"
		args = append(args, filter.To)
	}
	countQuery := "SELECT COUNT(1) FROM (" + query + ")"
	row := r.db.QueryRowContext(ctx, countQuery, args...)
	var total int
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}
	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, order)
	}
	return out, total, nil
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func (r *GormRepo) DeleteOrder(ctx context.Context, id int64) (err error) {
	if r.gdb != nil {
		tx := r.gdb.WithContext(ctx).Begin()
		if tx.Error != nil {
			return tx.Error
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback().Error
			}
		}()
		var exists int64
		if err = tx.Model(&orderRow{}).Where("id = ?", id).Count(&exists).Error; err != nil {
			return err
		}
		if exists == 0 {
			return usecase.ErrNotFound
		}
		sub := tx.Model(&orderItemRow{}).Select("id").Where("order_id = ?", id)
		if err = tx.Where("order_item_id IN (?)", sub).Delete(&vpsInstanceRow{}).Error; err != nil {
			return err
		}
		if err = tx.Where("order_id = ?", id).Delete(&provisionJobRow{}).Error; err != nil {
			return err
		}
		if err = tx.Where("order_id = ?", id).Delete(&resizeTaskRow{}).Error; err != nil {
			return err
		}
		if err = tx.Where("order_id = ?", id).Delete(&automationLogRow{}).Error; err != nil {
			return err
		}
		if err = tx.Where("order_id = ?", id).Delete(&orderEventRow{}).Error; err != nil {
			return err
		}
		if err = tx.Where("order_id = ?", id).Delete(&orderPaymentRow{}).Error; err != nil {
			return err
		}
		if err = tx.Where("order_id = ?", id).Delete(&orderItemRow{}).Error; err != nil {
			return err
		}
		if err = tx.Delete(&orderRow{}, id).Error; err != nil {
			return err
		}
		return tx.Commit().Error
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var exists int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM orders WHERE id = ?`, id).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return usecase.ErrNotFound
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM vps_instances WHERE order_item_id IN (SELECT id FROM order_items WHERE order_id = ?)`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM provision_jobs WHERE order_id = ?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM resize_tasks WHERE order_id = ?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM automation_logs WHERE order_id = ?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM order_events WHERE order_id = ?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM order_payments WHERE order_id = ?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM order_items WHERE order_id = ?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM orders WHERE id = ?`, id); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *GormRepo) CreateOrderItems(ctx context.Context, items []domain.OrderItem) error {
	if r.gdb != nil {
		if len(items) == 0 {
			return nil
		}
		rows := make([]orderItemRow, 0, len(items))
		for _, item := range items {
			rows = append(rows, toOrderItemRow(item))
		}
		if err := r.gdb.WithContext(ctx).Create(&rows).Error; err != nil {
			return err
		}
		for i := range rows {
			items[i].ID = rows[i].ID
			items[i].CreatedAt = rows[i].CreatedAt
			items[i].UpdatedAt = rows[i].UpdatedAt
		}
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO order_items(order_id,package_id,system_id,spec_json,qty,amount,status,goods_type_id,automation_instance_id,action,duration_months) VALUES (?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for i := range items {
		item := &items[i]
		res, err := stmt.ExecContext(ctx, item.OrderID, item.PackageID, item.SystemID, item.SpecJSON, item.Qty, item.Amount, item.Status, item.GoodsTypeID, item.AutomationInstanceID, item.Action, item.DurationMonths)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		id, _ := res.LastInsertId()
		item.ID = id
	}
	return tx.Commit()
}

func (r *GormRepo) ListOrderItems(ctx context.Context, orderID int64) ([]domain.OrderItem, error) {
	if r.gdb != nil {
		var rows []orderItemRow
		if err := r.gdb.WithContext(ctx).Where("order_id = ?", orderID).Order("id ASC").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.OrderItem, 0, len(rows))
		for _, row := range rows {
			out = append(out, fromOrderItemRow(row))
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, order_id, package_id, system_id, spec_json, qty, amount, status, goods_type_id, automation_instance_id, action, duration_months, created_at, updated_at FROM order_items WHERE order_id = ? ORDER BY id ASC`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.OrderItem
	for rows.Next() {
		item, err := scanOrderItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *GormRepo) GetOrderItem(ctx context.Context, id int64) (domain.OrderItem, error) {
	if r.gdb != nil {
		var row orderItemRow
		if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return domain.OrderItem{}, r.ensure(err)
		}
		return fromOrderItemRow(row), nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, order_id, package_id, system_id, spec_json, qty, amount, status, goods_type_id, automation_instance_id, action, duration_months, created_at, updated_at FROM order_items WHERE id = ?`, id)
	return scanOrderItem(row)
}

func (r *GormRepo) HasPendingRenewOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	if vpsID <= 0 {
		return false, nil
	}
	if r.gdb != nil {
		var rows []orderItemRow
		if err := r.gdb.WithContext(ctx).
			Joins("JOIN orders o ON o.id = order_items.order_id").
			Where("o.user_id = ? AND order_items.action = ? AND o.status IN ?",
				userID, "renew", []string{string(domain.OrderStatusPendingPayment), string(domain.OrderStatusPendingReview)}).
			Order("order_items.id DESC").
			Limit(20).
			Select("order_items.spec_json").
			Find(&rows).Error; err != nil {
			return false, err
		}
		for _, row := range rows {
			var payload struct {
				VPSID int64 `json:"vps_id"`
			}
			if err := json.Unmarshal([]byte(row.SpecJSON), &payload); err == nil && payload.VPSID == vpsID {
				return true, nil
			}
		}
		return false, nil
	}
	pattern1 := fmt.Sprintf("%%\"vps_id\":%d%%", vpsID)
	pattern2 := fmt.Sprintf("%%\"vps_id\": %d%%", vpsID)
	rows, err := r.db.QueryContext(ctx, `SELECT oi.spec_json FROM order_items oi JOIN orders o ON o.id = oi.order_id WHERE o.user_id = ? AND oi.action = 'renew' AND o.status IN ('pending_payment','pending_review') AND (oi.spec_json LIKE ? OR oi.spec_json LIKE ?) ORDER BY oi.id DESC LIMIT 20`, userID, pattern1, pattern2)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var spec string
		if err := rows.Scan(&spec); err != nil {
			return false, err
		}
		var payload struct {
			VPSID int64 `json:"vps_id"`
		}
		if err := json.Unmarshal([]byte(spec), &payload); err == nil && payload.VPSID == vpsID {
			return true, nil
		}
	}
	return false, nil
}

func (r *GormRepo) HasPendingResizeOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	if vpsID <= 0 {
		return false, nil
	}
	if r.gdb != nil {
		var rows []orderItemRow
		if err := r.gdb.WithContext(ctx).
			Joins("JOIN orders o ON o.id = order_items.order_id").
			Where("o.user_id = ? AND order_items.action = ? AND o.status IN ?",
				userID, "resize", []string{string(domain.OrderStatusPendingPayment), string(domain.OrderStatusPendingReview)}).
			Order("order_items.id DESC").
			Limit(20).
			Select("order_items.spec_json").
			Find(&rows).Error; err != nil {
			return false, err
		}
		for _, row := range rows {
			var payload struct {
				VPSID int64 `json:"vps_id"`
			}
			if err := json.Unmarshal([]byte(row.SpecJSON), &payload); err == nil && payload.VPSID == vpsID {
				return true, nil
			}
		}
		return false, nil
	}
	pattern1 := fmt.Sprintf("%%\"vps_id\":%d%%", vpsID)
	pattern2 := fmt.Sprintf("%%\"vps_id\": %d%%", vpsID)
	rows, err := r.db.QueryContext(ctx, `SELECT oi.spec_json FROM order_items oi JOIN orders o ON o.id = oi.order_id WHERE o.user_id = ? AND oi.action = 'resize' AND o.status IN ('pending_payment','pending_review') AND (oi.spec_json LIKE ? OR oi.spec_json LIKE ?) ORDER BY oi.id DESC LIMIT 20`, userID, pattern1, pattern2)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var spec string
		if err := rows.Scan(&spec); err != nil {
			return false, err
		}
		var payload struct {
			VPSID int64 `json:"vps_id"`
		}
		if err := json.Unmarshal([]byte(spec), &payload); err == nil && payload.VPSID == vpsID {
			return true, nil
		}
	}
	return false, nil
}

func (r *GormRepo) HasPendingRefundOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	if vpsID <= 0 {
		return false, nil
	}
	if r.gdb != nil {
		var rows []orderItemRow
		if err := r.gdb.WithContext(ctx).
			Joins("JOIN orders o ON o.id = order_items.order_id").
			Where("o.user_id = ? AND order_items.action = ? AND o.status IN ?",
				userID, "refund", []string{string(domain.OrderStatusPendingPayment), string(domain.OrderStatusPendingReview)}).
			Order("order_items.id DESC").
			Limit(20).
			Select("order_items.spec_json").
			Find(&rows).Error; err != nil {
			return false, err
		}
		for _, row := range rows {
			var payload struct {
				VPSID int64 `json:"vps_id"`
			}
			if err := json.Unmarshal([]byte(row.SpecJSON), &payload); err == nil && payload.VPSID == vpsID {
				return true, nil
			}
		}
		return false, nil
	}
	pattern1 := fmt.Sprintf("%%\"vps_id\":%d%%", vpsID)
	pattern2 := fmt.Sprintf("%%\"vps_id\": %d%%", vpsID)
	rows, err := r.db.QueryContext(ctx, `SELECT oi.spec_json FROM order_items oi JOIN orders o ON o.id = oi.order_id WHERE o.user_id = ? AND oi.action = 'refund' AND o.status IN ('pending_payment','pending_review') AND (oi.spec_json LIKE ? OR oi.spec_json LIKE ?) ORDER BY oi.id DESC LIMIT 20`, userID, pattern1, pattern2)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var spec string
		if err := rows.Scan(&spec); err != nil {
			return false, err
		}
		var payload struct {
			VPSID int64 `json:"vps_id"`
		}
		if err := json.Unmarshal([]byte(spec), &payload); err == nil && payload.VPSID == vpsID {
			return true, nil
		}
	}
	return false, nil
}

func (r *GormRepo) UpdateOrderItemStatus(ctx context.Context, id int64, status domain.OrderItemStatus) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&orderItemRow{}).Where("id = ?", id).Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE order_items SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	return err
}

func (r *GormRepo) UpdateOrderItemAutomation(ctx context.Context, id int64, automationID string) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&orderItemRow{}).Where("id = ?", id).Updates(map[string]any{
			"automation_instance_id": automationID,
			"updated_at":             time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE order_items SET automation_instance_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, automationID, id)
	return err
}

func (r *GormRepo) CreateInstance(ctx context.Context, inst *domain.VPSInstance) error {
	if r.gdb != nil {
		row := toVPSInstanceRow(*inst)
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		*inst = fromVPSInstanceRow(row)
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO vps_instances(user_id,order_item_id,automation_instance_id,goods_type_id,name,region,region_id,line_id,package_id,package_name,cpu,memory_gb,disk_gb,bandwidth_mbps,port_num,monthly_price,spec_json,system_id,status,automation_state,admin_status,expire_at,panel_url_cache,access_info_json,last_emergency_renew_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		inst.UserID, inst.OrderItemID, inst.AutomationInstanceID, inst.GoodsTypeID, inst.Name, inst.Region, inst.RegionID, inst.LineID, inst.PackageID, inst.PackageName, inst.CPU, inst.MemoryGB, inst.DiskGB, inst.BandwidthMB, inst.PortNum, inst.MonthlyPrice, inst.SpecJSON, inst.SystemID, inst.Status, inst.AutomationState, inst.AdminStatus, inst.ExpireAt, inst.PanelURLCache, inst.AccessInfoJSON, inst.LastEmergencyRenewAt)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	inst.ID = id
	return nil
}

func (r *GormRepo) GetInstance(ctx context.Context, id int64) (domain.VPSInstance, error) {
	if r.gdb != nil {
		var row vpsInstanceRow
		if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return domain.VPSInstance{}, r.ensure(err)
		}
		return fromVPSInstanceRow(row), nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, order_item_id, automation_instance_id, goods_type_id, name, region, region_id, line_id, package_id, package_name, cpu, memory_gb, disk_gb, bandwidth_mbps, port_num, monthly_price, spec_json, system_id, status, automation_state, admin_status, expire_at, panel_url_cache, access_info_json, last_emergency_renew_at, created_at, updated_at FROM vps_instances WHERE id = ?`, id)
	return scanVPSInstance(row)
}

func (r *GormRepo) GetInstanceByOrderItem(ctx context.Context, orderItemID int64) (domain.VPSInstance, error) {
	if r.gdb != nil {
		var row vpsInstanceRow
		if err := r.gdb.WithContext(ctx).Where("order_item_id = ?", orderItemID).First(&row).Error; err != nil {
			return domain.VPSInstance{}, r.ensure(err)
		}
		return fromVPSInstanceRow(row), nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, order_item_id, automation_instance_id, goods_type_id, name, region, region_id, line_id, package_id, package_name, cpu, memory_gb, disk_gb, bandwidth_mbps, port_num, monthly_price, spec_json, system_id, status, automation_state, admin_status, expire_at, panel_url_cache, access_info_json, last_emergency_renew_at, created_at, updated_at FROM vps_instances WHERE order_item_id = ?`, orderItemID)
	return scanVPSInstance(row)
}

func (r *GormRepo) ListInstancesByUser(ctx context.Context, userID int64) ([]domain.VPSInstance, error) {
	if r.gdb != nil {
		var rows []vpsInstanceRow
		if err := r.gdb.WithContext(ctx).Where("user_id = ?", userID).Order("id DESC").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.VPSInstance, 0, len(rows))
		for _, row := range rows {
			out = append(out, fromVPSInstanceRow(row))
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, order_item_id, automation_instance_id, goods_type_id, name, region, region_id, line_id, package_id, package_name, cpu, memory_gb, disk_gb, bandwidth_mbps, port_num, monthly_price, spec_json, system_id, status, automation_state, admin_status, expire_at, panel_url_cache, access_info_json, last_emergency_renew_at, created_at, updated_at FROM vps_instances WHERE user_id = ? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.VPSInstance
	for rows.Next() {
		inst, err := scanVPSInstance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inst)
	}
	return out, nil
}

func (r *GormRepo) ListInstances(ctx context.Context, limit, offset int) ([]domain.VPSInstance, int, error) {
	if r.gdb != nil {
		var total int64
		if err := r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Count(&total).Error; err != nil {
			return nil, 0, err
		}
		var rows []vpsInstanceRow
		if err := r.gdb.WithContext(ctx).Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, 0, err
		}
		out := make([]domain.VPSInstance, 0, len(rows))
		for _, row := range rows {
			out = append(out, fromVPSInstanceRow(row))
		}
		return out, int(total), nil
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM vps_instances`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, order_item_id, automation_instance_id, goods_type_id, name, region, region_id, line_id, package_id, package_name, cpu, memory_gb, disk_gb, bandwidth_mbps, port_num, monthly_price, spec_json, system_id, status, automation_state, admin_status, expire_at, panel_url_cache, access_info_json, last_emergency_renew_at, created_at, updated_at FROM vps_instances ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.VPSInstance
	for rows.Next() {
		inst, err := scanVPSInstance(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, inst)
	}
	return out, total, nil
}

func (r *GormRepo) ListInstancesExpiring(ctx context.Context, before time.Time) ([]domain.VPSInstance, error) {
	if r.gdb != nil {
		var rows []vpsInstanceRow
		if err := r.gdb.WithContext(ctx).Where("expire_at IS NOT NULL AND expire_at <= ?", before).Order("expire_at ASC").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.VPSInstance, 0, len(rows))
		for _, row := range rows {
			out = append(out, fromVPSInstanceRow(row))
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, order_item_id, automation_instance_id, goods_type_id, name, region, region_id, line_id, package_id, package_name, cpu, memory_gb, disk_gb, bandwidth_mbps, port_num, monthly_price, spec_json, system_id, status, automation_state, admin_status, expire_at, panel_url_cache, access_info_json, last_emergency_renew_at, created_at, updated_at FROM vps_instances WHERE expire_at IS NOT NULL AND expire_at <= ? ORDER BY expire_at ASC`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.VPSInstance
	for rows.Next() {
		inst, err := scanVPSInstance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inst)
	}
	return out, nil
}

func (r *GormRepo) DeleteInstance(ctx context.Context, id int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Delete(&vpsInstanceRow{}, id).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM vps_instances WHERE id = ?`, id)
	return err
}

func (r *GormRepo) UpdateInstanceStatus(ctx context.Context, id int64, status domain.VPSStatus, automationState int) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
				"status":           status,
				"automation_state": automationState,
				"updated_at":       time.Now(),
			}).Error; err != nil {
				return err
			}

			var inst vpsInstanceRow
			if err := tx.Where("id = ?", id).First(&inst).Error; err != nil {
				return err
			}
			orderItemID := inst.OrderItemID
			if orderItemID > 0 {
				switch {
				case isReadyVPSStatus(status):
					_ = tx.Model(&orderItemRow{}).
						Where("id = ? AND action = 'create' AND status IN ?", orderItemID, []string{string(domain.OrderItemStatusApproved), string(domain.OrderItemStatusProvisioning)}).
						Updates(map[string]any{"status": domain.OrderItemStatusActive, "updated_at": time.Now()}).Error
				case isFailedVPSStatus(status):
					_ = tx.Model(&orderItemRow{}).
						Where("id = ? AND action = 'create' AND status IN ?", orderItemID, []string{string(domain.OrderItemStatusApproved), string(domain.OrderItemStatusProvisioning)}).
						Updates(map[string]any{"status": domain.OrderItemStatusFailed, "updated_at": time.Now()}).Error
				}

				var item orderItemRow
				if err := tx.Where("id = ?", orderItemID).First(&item).Error; err == nil && item.OrderID > 0 {
					if err := recomputeOrderStatusByItemsGorm(ctx, tx, item.OrderID); err != nil {
						return err
					}
				}
			}
			return nil
		})
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, `UPDATE vps_instances SET status = ?, automation_state = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, automationState, id); err != nil {
		return err
	}

	var orderItemID int64
	if err := tx.QueryRowContext(ctx, `SELECT order_item_id FROM vps_instances WHERE id = ?`, id).Scan(&orderItemID); err != nil {
		return err
	}
	if orderItemID > 0 {
		switch {
		case isReadyVPSStatus(status):
			_, _ = tx.ExecContext(ctx, `UPDATE order_items SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND action = 'create' AND status IN (?,?)`,
				domain.OrderItemStatusActive, orderItemID, domain.OrderItemStatusApproved, domain.OrderItemStatusProvisioning)
		case isFailedVPSStatus(status):
			_, _ = tx.ExecContext(ctx, `UPDATE order_items SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND action = 'create' AND status IN (?,?)`,
				domain.OrderItemStatusFailed, orderItemID, domain.OrderItemStatusApproved, domain.OrderItemStatusProvisioning)
		}

		var orderID int64
		if err := tx.QueryRowContext(ctx, `SELECT order_id FROM order_items WHERE id = ?`, orderItemID).Scan(&orderID); err == nil && orderID > 0 {
			if err := recomputeOrderStatusByItems(ctx, tx, orderID); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func isReadyVPSStatus(status domain.VPSStatus) bool {
	switch status {
	case domain.VPSStatusRunning, domain.VPSStatusStopped, domain.VPSStatusRescue, domain.VPSStatusLocked, domain.VPSStatusExpiredLocked:
		return true
	default:
		return false
	}
}

func isFailedVPSStatus(status domain.VPSStatus) bool {
	return status == domain.VPSStatusReinstallFailed
}

func recomputeOrderStatusByItems(ctx context.Context, tx *sql.Tx, orderID int64) error {
	var currentStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM orders WHERE id = ?`, orderID).Scan(&currentStatus); err != nil {
		return err
	}
	switch currentStatus {
	case string(domain.OrderStatusApproved), string(domain.OrderStatusProvisioning), string(domain.OrderStatusActive), string(domain.OrderStatusFailed):
	default:
		return nil
	}

	var activeCount, failedCount, pendingCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM order_items WHERE order_id = ? AND status = ?`, orderID, domain.OrderItemStatusActive).Scan(&activeCount); err != nil {
		return err
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM order_items WHERE order_id = ? AND status = ?`, orderID, domain.OrderItemStatusFailed).Scan(&failedCount); err != nil {
		return err
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM order_items WHERE order_id = ? AND status NOT IN (?,?,?,?)`,
		orderID, domain.OrderItemStatusActive, domain.OrderItemStatusFailed, domain.OrderItemStatusCanceled, domain.OrderItemStatusRejected).Scan(&pendingCount); err != nil {
		return err
	}

	next := currentStatus
	switch {
	case failedCount > 0:
		next = string(domain.OrderStatusFailed)
	case pendingCount > 0:
		next = string(domain.OrderStatusProvisioning)
	case activeCount > 0:
		next = string(domain.OrderStatusActive)
	}
	if next == currentStatus {
		return nil
	}
	_, err := tx.ExecContext(ctx, `UPDATE orders SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, next, orderID)
	return err
}

func recomputeOrderStatusByItemsGorm(ctx context.Context, tx *gorm.DB, orderID int64) error {
	var order orderRow
	if err := tx.WithContext(ctx).Where("id = ?", orderID).First(&order).Error; err != nil {
		return err
	}
	switch order.Status {
	case string(domain.OrderStatusApproved), string(domain.OrderStatusProvisioning), string(domain.OrderStatusActive), string(domain.OrderStatusFailed):
	default:
		return nil
	}

	var activeCount, failedCount, pendingCount int64
	if err := tx.WithContext(ctx).Model(&orderItemRow{}).Where("order_id = ? AND status = ?", orderID, domain.OrderItemStatusActive).Count(&activeCount).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Model(&orderItemRow{}).Where("order_id = ? AND status = ?", orderID, domain.OrderItemStatusFailed).Count(&failedCount).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Model(&orderItemRow{}).
		Where("order_id = ? AND status NOT IN ?", orderID, []string{
			string(domain.OrderItemStatusActive),
			string(domain.OrderItemStatusFailed),
			string(domain.OrderItemStatusCanceled),
			string(domain.OrderItemStatusRejected),
		}).Count(&pendingCount).Error; err != nil {
		return err
	}

	next := order.Status
	switch {
	case failedCount > 0:
		next = string(domain.OrderStatusFailed)
	case pendingCount > 0:
		next = string(domain.OrderStatusProvisioning)
	case activeCount > 0:
		next = string(domain.OrderStatusActive)
	}
	if next == order.Status {
		return nil
	}
	return tx.WithContext(ctx).Model(&orderRow{}).Where("id = ?", orderID).Updates(map[string]any{
		"status":     next,
		"updated_at": time.Now(),
	}).Error
}

func (r *GormRepo) UpdateInstanceAdminStatus(ctx context.Context, id int64, status domain.VPSAdminStatus) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
			"admin_status": status,
			"updated_at":   time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE vps_instances SET admin_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	return err
}

func (r *GormRepo) UpdateInstanceExpireAt(ctx context.Context, id int64, expireAt time.Time) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
			"expire_at":  expireAt,
			"updated_at": time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE vps_instances SET expire_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, expireAt, id)
	return err
}

func (r *GormRepo) UpdateInstancePanelCache(ctx context.Context, id int64, panelURL string) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
			"panel_url_cache": panelURL,
			"updated_at":      time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE vps_instances SET panel_url_cache = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, panelURL, id)
	return err
}

func (r *GormRepo) UpdateInstanceSpec(ctx context.Context, id int64, specJSON string) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
			"spec_json":  specJSON,
			"updated_at": time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE vps_instances SET spec_json = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, specJSON, id)
	return err
}

func (r *GormRepo) UpdateInstanceAccessInfo(ctx context.Context, id int64, accessJSON string) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
			"access_info_json": accessJSON,
			"updated_at":       time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE vps_instances SET access_info_json = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, accessJSON, id)
	return err
}

func (r *GormRepo) UpdateInstanceEmergencyRenewAt(ctx context.Context, id int64, at time.Time) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
			"last_emergency_renew_at": at,
			"updated_at":              time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE vps_instances SET last_emergency_renew_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, at, id)
	return err
}

func (r *GormRepo) UpdateInstanceLocal(ctx context.Context, inst domain.VPSInstance) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", inst.ID).Updates(map[string]any{
			"package_id":       inst.PackageID,
			"package_name":     inst.PackageName,
			"cpu":              inst.CPU,
			"memory_gb":        inst.MemoryGB,
			"disk_gb":          inst.DiskGB,
			"bandwidth_mbps":   inst.BandwidthMB,
			"port_num":         inst.PortNum,
			"monthly_price":    inst.MonthlyPrice,
			"spec_json":        inst.SpecJSON,
			"system_id":        inst.SystemID,
			"status":           inst.Status,
			"admin_status":     inst.AdminStatus,
			"panel_url_cache":  inst.PanelURLCache,
			"access_info_json": inst.AccessInfoJSON,
			"updated_at":       time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE vps_instances SET package_id = ?, package_name = ?, cpu = ?, memory_gb = ?, disk_gb = ?, bandwidth_mbps = ?, port_num = ?, monthly_price = ?, spec_json = ?, system_id = ?, status = ?, admin_status = ?, panel_url_cache = ?, access_info_json = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		inst.PackageID, inst.PackageName, inst.CPU, inst.MemoryGB, inst.DiskGB, inst.BandwidthMB, inst.PortNum, inst.MonthlyPrice, inst.SpecJSON, inst.SystemID, inst.Status, inst.AdminStatus, inst.PanelURLCache, inst.AccessInfoJSON, inst.ID)
	return err
}

func (r *GormRepo) AppendEvent(ctx context.Context, orderID int64, eventType string, dataJSON string) (domain.OrderEvent, error) {
	if r.gdb != nil {
		tx := r.gdb.WithContext(ctx).Begin()
		if tx.Error != nil {
			return domain.OrderEvent{}, tx.Error
		}
		var seq int64
		if err := tx.Model(&orderEventRow{}).Where("order_id = ?", orderID).Select("COALESCE(MAX(seq),0)").Take(&seq).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			_ = tx.Rollback().Error
			return domain.OrderEvent{}, err
		}
		seq++
		row := orderEventRow{OrderID: orderID, Seq: seq, Type: eventType, DataJSON: dataJSON}
		if err := tx.Create(&row).Error; err != nil {
			_ = tx.Rollback().Error
			return domain.OrderEvent{}, err
		}
		if err := tx.Commit().Error; err != nil {
			return domain.OrderEvent{}, err
		}
		return domain.OrderEvent{ID: row.ID, OrderID: row.OrderID, Seq: row.Seq, Type: row.Type, DataJSON: row.DataJSON, CreatedAt: row.CreatedAt}, nil
	}
	var seq int64
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq),0) FROM order_events WHERE order_id = ?`, orderID).Scan(&seq)
	seq++
	res, err := r.db.ExecContext(ctx, `INSERT INTO order_events(order_id, seq, type, data_json) VALUES (?,?,?,?)`, orderID, seq, eventType, dataJSON)
	if err != nil {
		return domain.OrderEvent{}, err
	}
	id, _ := res.LastInsertId()
	return domain.OrderEvent{ID: id, OrderID: orderID, Seq: seq, Type: eventType, DataJSON: dataJSON, CreatedAt: time.Now()}, nil
}

func (r *GormRepo) ListEventsAfter(ctx context.Context, orderID int64, afterSeq int64, limit int) ([]domain.OrderEvent, error) {
	if r.gdb != nil {
		var rows []orderEventRow
		if err := r.gdb.WithContext(ctx).
			Where("order_id = ? AND seq > ?", orderID, afterSeq).
			Order("seq ASC").
			Limit(limit).
			Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.OrderEvent, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.OrderEvent{ID: row.ID, OrderID: row.OrderID, Seq: row.Seq, Type: row.Type, DataJSON: row.DataJSON, CreatedAt: row.CreatedAt})
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, order_id, seq, type, data_json, created_at FROM order_events WHERE order_id = ? AND seq > ? ORDER BY seq ASC LIMIT ?`, orderID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.OrderEvent
	for rows.Next() {
		var ev domain.OrderEvent
		if err := rows.Scan(&ev.ID, &ev.OrderID, &ev.Seq, &ev.Type, &ev.DataJSON, &ev.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, nil
}

func (r *GormRepo) CreatePayment(ctx context.Context, payment *domain.OrderPayment) error {
	tradeNo := strings.TrimSpace(payment.TradeNo)
	if tradeNo == "" {
		// Keep external semantics for empty trade_no while avoiding unique-key collisions.
		tradeNo = fmt.Sprintf("pending-%d-%d", payment.OrderID, time.Now().UnixNano())
	}
	if r.gdb != nil {
		row := toOrderPaymentRow(*payment)
		row.TradeNo = tradeNo
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		*payment = fromOrderPaymentRow(row)
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO order_payments(order_id,user_id,method,amount,currency,trade_no,note,screenshot_url,status,idempotency_key,reviewed_by,review_reason) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, payment.OrderID, payment.UserID, payment.Method, payment.Amount, payment.Currency, tradeNo, payment.Note, payment.ScreenshotURL, payment.Status, payment.IdempotencyKey, payment.ReviewedBy, payment.ReviewReason)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	payment.ID = id
	return nil
}

func (r *GormRepo) ListPaymentsByOrder(ctx context.Context, orderID int64) ([]domain.OrderPayment, error) {
	if r.gdb != nil {
		var rows []orderPaymentRow
		if err := r.gdb.WithContext(ctx).Where("order_id = ?", orderID).Order("id DESC").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.OrderPayment, 0, len(rows))
		for _, row := range rows {
			out = append(out, fromOrderPaymentRow(row))
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, order_id, user_id, method, amount, currency, trade_no, note, screenshot_url, status, idempotency_key, reviewed_by, review_reason, created_at, updated_at FROM order_payments WHERE order_id = ? ORDER BY id DESC`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.OrderPayment
	for rows.Next() {
		pay, err := scanOrderPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, pay)
	}
	return out, nil
}

func (r *GormRepo) GetPaymentByTradeNo(ctx context.Context, tradeNo string) (domain.OrderPayment, error) {
	if strings.TrimSpace(tradeNo) == "" {
		return domain.OrderPayment{}, sql.ErrNoRows
	}
	if r.gdb != nil {
		var row orderPaymentRow
		if err := r.gdb.WithContext(ctx).Where("trade_no = ?", tradeNo).First(&row).Error; err != nil {
			return domain.OrderPayment{}, r.ensure(err)
		}
		return fromOrderPaymentRow(row), nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, order_id, user_id, method, amount, currency, trade_no, note, screenshot_url, status, idempotency_key, reviewed_by, review_reason, created_at, updated_at FROM order_payments WHERE trade_no = ?`, tradeNo)
	return scanOrderPayment(row)
}

func (r *GormRepo) GetPaymentByIdempotencyKey(ctx context.Context, orderID int64, key string) (domain.OrderPayment, error) {
	if r.gdb != nil {
		var row orderPaymentRow
		if err := r.gdb.WithContext(ctx).Where("order_id = ? AND idempotency_key = ?", orderID, key).First(&row).Error; err != nil {
			return domain.OrderPayment{}, r.ensure(err)
		}
		return fromOrderPaymentRow(row), nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, order_id, user_id, method, amount, currency, trade_no, note, screenshot_url, status, idempotency_key, reviewed_by, review_reason, created_at, updated_at FROM order_payments WHERE order_id = ? AND idempotency_key = ?`, orderID, key)
	return scanOrderPayment(row)
}

func (r *GormRepo) UpdatePaymentStatus(ctx context.Context, id int64, status domain.PaymentStatus, reviewedBy *int64, reason string) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&orderPaymentRow{}).Where("id = ?", id).Updates(map[string]any{
			"status":        status,
			"reviewed_by":   reviewedBy,
			"review_reason": reason,
			"updated_at":    time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE order_payments SET status = ?, reviewed_by = ?, review_reason = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, reviewedBy, reason, id)
	return err
}

func (r *GormRepo) UpdatePaymentTradeNo(ctx context.Context, id int64, tradeNo string) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&orderPaymentRow{}).Where("id = ?", id).Updates(map[string]any{
			"trade_no":   tradeNo,
			"updated_at": time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE order_payments SET trade_no = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, tradeNo, id)
	return err
}

func (r *GormRepo) ListPayments(ctx context.Context, filter usecase.PaymentFilter, limit, offset int) ([]domain.OrderPayment, int, error) {
	if r.gdb != nil {
		q := r.gdb.WithContext(ctx).Model(&orderPaymentRow{})
		if filter.Status != "" {
			q = q.Where("status = ?", filter.Status)
		}
		if filter.From != nil {
			q = q.Where("created_at >= ?", filter.From)
		}
		if filter.To != nil {
			q = q.Where("created_at <= ?", filter.To)
		}
		var total int64
		if err := q.Count(&total).Error; err != nil {
			return nil, 0, err
		}
		if limit <= 0 {
			limit = 20
		}
		var rows []orderPaymentRow
		if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, 0, err
		}
		out := make([]domain.OrderPayment, 0, len(rows))
		for _, row := range rows {
			out = append(out, fromOrderPaymentRow(row))
		}
		return out, int(total), nil
	}
	query := `SELECT id, order_id, user_id, method, amount, currency, trade_no, note, screenshot_url, status, idempotency_key, reviewed_by, review_reason, created_at, updated_at FROM order_payments WHERE 1=1`
	args := []any{}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.From != nil {
		query += " AND created_at >= ?"
		args = append(args, filter.From)
	}
	if filter.To != nil {
		query += " AND created_at <= ?"
		args = append(args, filter.To)
	}
	countQuery := "SELECT COUNT(1) FROM (" + query + ")"
	row := r.db.QueryRowContext(ctx, countQuery, args...)
	var total int
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.OrderPayment
	for rows.Next() {
		pay, err := scanOrderPayment(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, pay)
	}
	return out, total, nil
}

func (r *GormRepo) CreateAPIKey(ctx context.Context, key *domain.APIKey) error {
	if r.gdb != nil {
		row := apiKeyRow{
			Name:              key.Name,
			KeyHash:           key.KeyHash,
			Status:            string(key.Status),
			ScopesJSON:        key.ScopesJSON,
			PermissionGroupID: key.PermissionGroupID,
		}
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		key.ID = row.ID
		key.CreatedAt = row.CreatedAt
		key.UpdatedAt = row.UpdatedAt
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO api_keys(name,key_hash,status,scopes_json,permission_group_id) VALUES (?,?,?,?,?)`, key.Name, key.KeyHash, key.Status, key.ScopesJSON, key.PermissionGroupID)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	key.ID = id
	return nil
}

func (r *GormRepo) GetAPIKeyByHash(ctx context.Context, keyHash string) (domain.APIKey, error) {
	if r.gdb != nil {
		var row apiKeyRow
		if err := r.gdb.WithContext(ctx).Where("key_hash = ?", keyHash).First(&row).Error; err != nil {
			return domain.APIKey{}, r.ensure(err)
		}
		var out domain.APIKey
		out.ID = row.ID
		out.Name = row.Name
		out.KeyHash = row.KeyHash
		out.Status = domain.APIKeyStatus(row.Status)
		out.ScopesJSON = row.ScopesJSON
		out.PermissionGroupID = row.PermissionGroupID
		out.CreatedAt = row.CreatedAt
		out.UpdatedAt = row.UpdatedAt
		return out, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, name, key_hash, status, scopes_json, permission_group_id, created_at, updated_at, last_used_at FROM api_keys WHERE key_hash = ?`, keyHash)
	return scanAPIKey(row)
}

func (r *GormRepo) ListAPIKeys(ctx context.Context, limit, offset int) ([]domain.APIKey, int, error) {
	if r.gdb != nil {
		var total int64
		if err := r.gdb.WithContext(ctx).Model(&apiKeyRow{}).Count(&total).Error; err != nil {
			return nil, 0, err
		}
		var rows []apiKeyRow
		if err := r.gdb.WithContext(ctx).Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, 0, err
		}
		out := make([]domain.APIKey, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.APIKey{
				ID:                row.ID,
				Name:              row.Name,
				KeyHash:           row.KeyHash,
				Status:            domain.APIKeyStatus(row.Status),
				ScopesJSON:        row.ScopesJSON,
				PermissionGroupID: row.PermissionGroupID,
				CreatedAt:         row.CreatedAt,
				UpdatedAt:         row.UpdatedAt,
			})
		}
		return out, int(total), nil
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM api_keys`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, key_hash, status, scopes_json, permission_group_id, created_at, updated_at, last_used_at FROM api_keys ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.APIKey
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, key)
	}
	return out, total, nil
}

func (r *GormRepo) UpdateAPIKeyStatus(ctx context.Context, id int64, status domain.APIKeyStatus) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&apiKeyRow{}).Where("id = ?", id).Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE api_keys SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	return err
}

func (r *GormRepo) TouchAPIKey(ctx context.Context, id int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&apiKeyRow{}).Where("id = ?", id).Update("last_used_at", time.Now()).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

func (r *GormRepo) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	if r.gdb != nil {
		var m settingModel
		if err := r.gdb.WithContext(ctx).Where("`key` = ?", key).First(&m).Error; err != nil {
			return domain.Setting{}, r.ensure(err)
		}
		return domain.Setting{Key: m.Key, ValueJSON: m.ValueJSON, UpdatedAt: m.UpdatedAt}, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT key, value_json, updated_at FROM settings WHERE key = ?`, key)
	var s domain.Setting
	if err := row.Scan(&s.Key, &s.ValueJSON, &s.UpdatedAt); err != nil {
		return domain.Setting{}, r.ensure(err)
	}
	return s, nil
}

func (r *GormRepo) UpsertSetting(ctx context.Context, setting domain.Setting) error {
	if r.gdb == nil {
		_, err := r.db.ExecContext(ctx, `INSERT INTO settings(key,value_json,updated_at) VALUES (?,?,CURRENT_TIMESTAMP) ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json, updated_at = CURRENT_TIMESTAMP`, setting.Key, setting.ValueJSON)
		return err
	}
	m := settingModel{Key: setting.Key, ValueJSON: setting.ValueJSON, UpdatedAt: time.Now()}
	return r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value_json", "updated_at"}),
		}).
		Create(&m).Error
}

func (r *GormRepo) ListSettings(ctx context.Context) ([]domain.Setting, error) {
	if r.gdb != nil {
		var models []settingModel
		if err := r.gdb.WithContext(ctx).Order("`key` ASC").Find(&models).Error; err != nil {
			return nil, err
		}
		out := make([]domain.Setting, 0, len(models))
		for _, m := range models {
			out = append(out, domain.Setting{
				Key:       m.Key,
				ValueJSON: m.ValueJSON,
				UpdatedAt: m.UpdatedAt,
			})
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT key, value_json, updated_at FROM settings ORDER BY key ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Setting
	for rows.Next() {
		var s domain.Setting
		if err := rows.Scan(&s.Key, &s.ValueJSON, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (r *GormRepo) UpsertPluginInstallation(ctx context.Context, inst *domain.PluginInstallation) error {
	if inst == nil || strings.TrimSpace(inst.Category) == "" || strings.TrimSpace(inst.PluginID) == "" || strings.TrimSpace(inst.InstanceID) == "" {
		return usecase.ErrInvalidInput
	}
	if r.gdb == nil {
		_, err := r.db.ExecContext(ctx, `INSERT INTO plugin_installations(category,plugin_id,instance_id,enabled,signature_status,config_cipher,created_at,updated_at)
			VALUES (?,?,?,?,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
			ON CONFLICT(category,plugin_id,instance_id) DO UPDATE SET enabled = excluded.enabled, signature_status = excluded.signature_status, config_cipher = excluded.config_cipher, updated_at = CURRENT_TIMESTAMP`,
			inst.Category, inst.PluginID, inst.InstanceID, boolToInt(inst.Enabled), inst.SignatureStatus, inst.ConfigCipher,
		)
		return err
	}
	m := pluginInstallationModel{
		Category:        inst.Category,
		PluginID:        inst.PluginID,
		InstanceID:      inst.InstanceID,
		Enabled:         boolToInt(inst.Enabled),
		SignatureStatus: string(inst.SignatureStatus),
		ConfigCipher:    inst.ConfigCipher,
	}
	return r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "category"}, {Name: "plugin_id"}, {Name: "instance_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"enabled",
				"signature_status",
				"config_cipher",
				"updated_at",
			}),
		}).
		Create(&m).Error
}

func (r *GormRepo) GetPluginInstallation(ctx context.Context, category, pluginID, instanceID string) (domain.PluginInstallation, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, category, plugin_id, instance_id, enabled, signature_status, config_cipher, created_at, updated_at FROM plugin_installations WHERE category = ? AND plugin_id = ? AND instance_id = ?`, category, pluginID, instanceID)
	var inst domain.PluginInstallation
	var enabled int
	var sig string
	if err := row.Scan(&inst.ID, &inst.Category, &inst.PluginID, &inst.InstanceID, &enabled, &sig, &inst.ConfigCipher, &inst.CreatedAt, &inst.UpdatedAt); err != nil {
		return domain.PluginInstallation{}, r.ensure(err)
	}
	inst.Enabled = enabled != 0
	inst.SignatureStatus = domain.PluginSignatureStatus(sig)
	return inst, nil
}

func (r *GormRepo) ListPluginInstallations(ctx context.Context) ([]domain.PluginInstallation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, category, plugin_id, instance_id, enabled, signature_status, config_cipher, created_at, updated_at FROM plugin_installations ORDER BY category ASC, plugin_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PluginInstallation
	for rows.Next() {
		var inst domain.PluginInstallation
		var enabled int
		var sig string
		if err := rows.Scan(&inst.ID, &inst.Category, &inst.PluginID, &inst.InstanceID, &enabled, &sig, &inst.ConfigCipher, &inst.CreatedAt, &inst.UpdatedAt); err != nil {
			return nil, err
		}
		inst.Enabled = enabled != 0
		inst.SignatureStatus = domain.PluginSignatureStatus(sig)
		out = append(out, inst)
	}
	return out, nil
}

func (r *GormRepo) DeletePluginInstallation(ctx context.Context, category, pluginID, instanceID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM plugin_installations WHERE category = ? AND plugin_id = ? AND instance_id = ?`, category, pluginID, instanceID)
	return err
}

func (r *GormRepo) ListPluginPaymentMethods(ctx context.Context, category, pluginID, instanceID string) ([]domain.PluginPaymentMethod, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, category, plugin_id, instance_id, method, enabled, created_at, updated_at FROM plugin_payment_methods WHERE category = ? AND plugin_id = ? AND instance_id = ? ORDER BY method ASC`, category, pluginID, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PluginPaymentMethod
	for rows.Next() {
		var m domain.PluginPaymentMethod
		var enabled int
		if err := rows.Scan(&m.ID, &m.Category, &m.PluginID, &m.InstanceID, &m.Method, &enabled, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		m.Enabled = enabled != 0
		out = append(out, m)
	}
	return out, nil
}

func (r *GormRepo) UpsertPluginPaymentMethod(ctx context.Context, m *domain.PluginPaymentMethod) error {
	if m == nil || strings.TrimSpace(m.Category) == "" || strings.TrimSpace(m.PluginID) == "" || strings.TrimSpace(m.InstanceID) == "" || strings.TrimSpace(m.Method) == "" {
		return usecase.ErrInvalidInput
	}
	if r.gdb != nil {
		row := pluginPaymentMethodModel{
			Category:   m.Category,
			PluginID:   m.PluginID,
			InstanceID: m.InstanceID,
			Method:     m.Method,
			Enabled:    boolToInt(m.Enabled),
			UpdatedAt:  time.Now(),
		}
		return r.gdb.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "category"},
					{Name: "plugin_id"},
					{Name: "instance_id"},
					{Name: "method"},
				},
				DoUpdates: clause.AssignmentColumns([]string{"enabled", "updated_at"}),
			}).
			Create(&row).Error
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO plugin_payment_methods(category,plugin_id,instance_id,method,enabled,created_at,updated_at)
		VALUES (?,?,?,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		ON CONFLICT(category,plugin_id,instance_id,method) DO UPDATE SET enabled = excluded.enabled, updated_at = CURRENT_TIMESTAMP`,
		m.Category, m.PluginID, m.InstanceID, m.Method, boolToInt(m.Enabled),
	)
	return err
}

func (r *GormRepo) DeletePluginPaymentMethod(ctx context.Context, category, pluginID, instanceID, method string) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).
			Where("category = ? AND plugin_id = ? AND instance_id = ? AND method = ?", category, pluginID, instanceID, method).
			Delete(&pluginPaymentMethodModel{}).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM plugin_payment_methods WHERE category = ? AND plugin_id = ? AND instance_id = ? AND method = ?`, category, pluginID, instanceID, method)
	return err
}

func (r *GormRepo) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	if r.gdb != nil {
		var rows []emailTemplateRow
		if err := r.gdb.WithContext(ctx).Order("id DESC").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.EmailTemplate, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.EmailTemplate{
				ID:        row.ID,
				Name:      row.Name,
				Subject:   row.Subject,
				Body:      row.Body,
				Enabled:   row.Enabled == 1,
				CreatedAt: row.CreatedAt,
				UpdatedAt: row.UpdatedAt,
			})
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, subject, body, enabled, created_at, updated_at FROM email_templates ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.EmailTemplate
	for rows.Next() {
		tmpl, err := scanEmailTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, tmpl)
	}
	return out, nil
}

func (r *GormRepo) GetEmailTemplate(ctx context.Context, id int64) (domain.EmailTemplate, error) {
	if r.gdb != nil {
		var row emailTemplateRow
		if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return domain.EmailTemplate{}, r.ensure(err)
		}
		return domain.EmailTemplate{
			ID:        row.ID,
			Name:      row.Name,
			Subject:   row.Subject,
			Body:      row.Body,
			Enabled:   row.Enabled == 1,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, name, subject, body, enabled, created_at, updated_at FROM email_templates WHERE id = ?`, id)
	return scanEmailTemplate(row)
}

func (r *GormRepo) UpsertEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error {
	if r.gdb != nil {
		if tmpl.ID == 0 {
			row := emailTemplateRow{
				Name:    tmpl.Name,
				Subject: tmpl.Subject,
				Body:    tmpl.Body,
				Enabled: boolToInt(tmpl.Enabled),
			}
			if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
				return err
			}
			tmpl.ID = row.ID
			return nil
		}
		var count int64
		if err := r.gdb.WithContext(ctx).Model(&emailTemplateRow{}).Where("name = ? AND id != ?", tmpl.Name, tmpl.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("email template name already exists")
		}
		return r.gdb.WithContext(ctx).Model(&emailTemplateRow{}).Where("id = ?", tmpl.ID).Updates(map[string]any{
			"name":       tmpl.Name,
			"subject":    tmpl.Subject,
			"body":       tmpl.Body,
			"enabled":    boolToInt(tmpl.Enabled),
			"updated_at": time.Now(),
		}).Error
	}
	if tmpl.ID == 0 {
		res, err := r.db.ExecContext(ctx, `INSERT INTO email_templates(name,subject,body,enabled) VALUES (?,?,?,?)`, tmpl.Name, tmpl.Subject, tmpl.Body, boolToInt(tmpl.Enabled))
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		tmpl.ID = id
		return nil
	}
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM email_templates WHERE name = ? AND id != ?`, tmpl.Name, tmpl.ID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return errors.New("email template name already exists")
	}
	_, err := r.db.ExecContext(ctx, `UPDATE email_templates SET name = ?, subject = ?, body = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, tmpl.Name, tmpl.Subject, tmpl.Body, boolToInt(tmpl.Enabled), tmpl.ID)
	return err
}

func (r *GormRepo) DeleteEmailTemplate(ctx context.Context, id int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Delete(&emailTemplateRow{}, id).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM email_templates WHERE id = ?`, id)
	return err
}

func (r *GormRepo) ListBillingCycles(ctx context.Context) ([]domain.BillingCycle, error) {
	if r.gdb != nil {
		var rows []billingCycleRow
		if err := r.gdb.WithContext(ctx).Order("sort_order, id").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.BillingCycle, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.BillingCycle{
				ID:         row.ID,
				Name:       row.Name,
				Months:     row.Months,
				Multiplier: row.Multiplier,
				MinQty:     row.MinQty,
				MaxQty:     row.MaxQty,
				Active:     row.Active == 1,
				SortOrder:  row.SortOrder,
				CreatedAt:  row.CreatedAt,
				UpdatedAt:  row.UpdatedAt,
			})
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, months, multiplier, min_qty, max_qty, active, sort_order, created_at, updated_at FROM billing_cycles ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.BillingCycle
	for rows.Next() {
		cycle, err := scanBillingCycle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cycle)
	}
	return out, nil
}

func (r *GormRepo) GetBillingCycle(ctx context.Context, id int64) (domain.BillingCycle, error) {
	if r.gdb != nil {
		var row billingCycleRow
		if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return domain.BillingCycle{}, r.ensure(err)
		}
		return domain.BillingCycle{
			ID:         row.ID,
			Name:       row.Name,
			Months:     row.Months,
			Multiplier: row.Multiplier,
			MinQty:     row.MinQty,
			MaxQty:     row.MaxQty,
			Active:     row.Active == 1,
			SortOrder:  row.SortOrder,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		}, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, name, months, multiplier, min_qty, max_qty, active, sort_order, created_at, updated_at FROM billing_cycles WHERE id = ?`, id)
	return scanBillingCycle(row)
}

func (r *GormRepo) CreateBillingCycle(ctx context.Context, cycle *domain.BillingCycle) error {
	if r.gdb != nil {
		row := billingCycleRow{
			Name:       cycle.Name,
			Months:     cycle.Months,
			Multiplier: cycle.Multiplier,
			MinQty:     cycle.MinQty,
			MaxQty:     cycle.MaxQty,
			Active:     boolToInt(cycle.Active),
			SortOrder:  cycle.SortOrder,
		}
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		cycle.ID = row.ID
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO billing_cycles(name,months,multiplier,min_qty,max_qty,active,sort_order) VALUES (?,?,?,?,?,?,?)`, cycle.Name, cycle.Months, cycle.Multiplier, cycle.MinQty, cycle.MaxQty, boolToInt(cycle.Active), cycle.SortOrder)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	cycle.ID = id
	return nil
}

func (r *GormRepo) UpdateBillingCycle(ctx context.Context, cycle domain.BillingCycle) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&billingCycleRow{}).Where("id = ?", cycle.ID).Updates(map[string]any{
			"name":       cycle.Name,
			"months":     cycle.Months,
			"multiplier": cycle.Multiplier,
			"min_qty":    cycle.MinQty,
			"max_qty":    cycle.MaxQty,
			"active":     boolToInt(cycle.Active),
			"sort_order": cycle.SortOrder,
			"updated_at": time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE billing_cycles SET name = ?, months = ?, multiplier = ?, min_qty = ?, max_qty = ?, active = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, cycle.Name, cycle.Months, cycle.Multiplier, cycle.MinQty, cycle.MaxQty, boolToInt(cycle.Active), cycle.SortOrder, cycle.ID)
	return err
}

func (r *GormRepo) DeleteBillingCycle(ctx context.Context, id int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Delete(&billingCycleRow{}, id).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM billing_cycles WHERE id = ?`, id)
	return err
}

func (r *GormRepo) CreateAutomationLog(ctx context.Context, log *domain.AutomationLog) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO automation_logs(order_id,order_item_id,action,request_json,response_json,success,message) VALUES (?,?,?,?,?,?,?)`, log.OrderID, log.OrderItemID, log.Action, log.RequestJSON, log.ResponseJSON, boolToInt(log.Success), log.Message)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	log.ID = id
	return nil
}

func (r *GormRepo) ListAutomationLogs(ctx context.Context, orderID int64, limit, offset int) ([]domain.AutomationLog, int, error) {
	var total int
	where := ""
	args := []any{}
	if orderID > 0 {
		where = " WHERE order_id = ?"
		args = append(args, orderID)
	}
	countQuery := "SELECT COUNT(1) FROM automation_logs" + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	listQuery := "SELECT id, order_id, order_item_id, action, request_json, response_json, success, message, created_at FROM automation_logs" + where + " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.AutomationLog
	for rows.Next() {
		logEntry, err := scanAutomationLog(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, logEntry)
	}
	return out, total, nil
}

func (r *GormRepo) PurgeAutomationLogs(ctx context.Context, before time.Time) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM automation_logs WHERE created_at < ?`, before)
	return err
}

func (r *GormRepo) CreateOrUpdateProvisionJob(ctx context.Context, job *domain.ProvisionJob) error {
	if r.gdb == nil {
		_, err := r.db.ExecContext(ctx, `INSERT INTO provision_jobs(order_id,order_item_id,host_id,host_name,status,attempts,next_run_at,last_error,created_at,updated_at)
			VALUES (?,?,?,?,?,?,?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			ON CONFLICT(order_item_id) DO UPDATE SET
				host_id = excluded.host_id,
				host_name = excluded.host_name,
				status = excluded.status,
				attempts = excluded.attempts,
				next_run_at = excluded.next_run_at,
				last_error = excluded.last_error,
				updated_at = CURRENT_TIMESTAMP`,
			job.OrderID, job.OrderItemID, job.HostID, job.HostName, job.Status, job.Attempts, job.NextRunAt, job.LastError)
		return err
	}
	m := provisionJobModel{
		ID:          job.ID,
		OrderID:     job.OrderID,
		OrderItemID: job.OrderItemID,
		HostID:      job.HostID,
		HostName:    job.HostName,
		Status:      job.Status,
		Attempts:    job.Attempts,
		NextRunAt:   job.NextRunAt,
		LastError:   job.LastError,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "order_item_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"host_id", "host_name", "status", "attempts", "next_run_at", "last_error", "updated_at",
			}),
		}).
		Create(&m).Error; err != nil {
		return err
	}
	var got provisionJobModel
	if err := r.gdb.WithContext(ctx).Select("id").Where("order_item_id = ?", job.OrderItemID).First(&got).Error; err == nil {
		job.ID = got.ID
	}
	return nil
}

func (r *GormRepo) ListDueProvisionJobs(ctx context.Context, limit int) ([]domain.ProvisionJob, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, order_id, order_item_id, host_id, host_name, status, attempts, next_run_at, last_error, created_at, updated_at
		FROM provision_jobs
		WHERE status IN ('pending','retry','running') AND datetime(next_run_at) <= datetime('now')
		ORDER BY id ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ProvisionJob
	for rows.Next() {
		job, err := scanProvisionJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, nil
}

func (r *GormRepo) UpdateProvisionJob(ctx context.Context, job domain.ProvisionJob) error {
	_, err := r.db.ExecContext(ctx, `UPDATE provision_jobs SET status = ?, attempts = ?, next_run_at = ?, last_error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		job.Status, job.Attempts, job.NextRunAt, job.LastError, job.ID)
	return err
}

func (r *GormRepo) CreateTaskRun(ctx context.Context, run *domain.ScheduledTaskRun) error {
	if r.gdb != nil {
		row := scheduledTaskRunRow{
			TaskKey:     run.TaskKey,
			Status:      run.Status,
			StartedAt:   run.StartedAt,
			FinishedAt:  run.FinishedAt,
			DurationSec: run.DurationSec,
			Message:     run.Message,
		}
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		run.ID = row.ID
		run.CreatedAt = row.CreatedAt
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO scheduled_task_runs(task_key,status,started_at,finished_at,duration_sec,message,created_at)
		VALUES (?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		run.TaskKey, run.Status, run.StartedAt, run.FinishedAt, run.DurationSec, run.Message)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	run.ID = id
	return nil
}

func (r *GormRepo) UpdateTaskRun(ctx context.Context, run domain.ScheduledTaskRun) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&scheduledTaskRunRow{}).Where("id = ?", run.ID).Updates(map[string]any{
			"status":       run.Status,
			"finished_at":  run.FinishedAt,
			"duration_sec": run.DurationSec,
			"message":      run.Message,
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE scheduled_task_runs SET status = ?, finished_at = ?, duration_sec = ?, message = ? WHERE id = ?`,
		run.Status, run.FinishedAt, run.DurationSec, run.Message, run.ID)
	return err
}

func (r *GormRepo) ListTaskRuns(ctx context.Context, key string, limit int) ([]domain.ScheduledTaskRun, error) {
	if r.gdb != nil {
		if limit <= 0 {
			limit = 20
		}
		var rows []scheduledTaskRunRow
		if err := r.gdb.WithContext(ctx).Where("task_key = ?", key).Order("id DESC").Limit(limit).Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.ScheduledTaskRun, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.ScheduledTaskRun{
				ID:          row.ID,
				TaskKey:     row.TaskKey,
				Status:      row.Status,
				StartedAt:   row.StartedAt,
				FinishedAt:  row.FinishedAt,
				DurationSec: row.DurationSec,
				Message:     row.Message,
				CreatedAt:   row.CreatedAt,
			})
		}
		return out, nil
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, task_key, status, started_at, finished_at, duration_sec, message, created_at
		FROM scheduled_task_runs WHERE task_key = ? ORDER BY id DESC LIMIT ?`, key, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ScheduledTaskRun
	for rows.Next() {
		var run domain.ScheduledTaskRun
		var finishedAt sql.NullTime
		if err := rows.Scan(&run.ID, &run.TaskKey, &run.Status, &run.StartedAt, &finishedAt, &run.DurationSec, &run.Message, &run.CreatedAt); err != nil {
			return nil, err
		}
		if finishedAt.Valid {
			run.FinishedAt = &finishedAt.Time
		}
		out = append(out, run)
	}
	return out, nil
}

func (r *GormRepo) CreateResizeTask(ctx context.Context, task *domain.ResizeTask) error {
	if r.gdb != nil {
		row := resizeTaskRow{
			VPSID:       task.VPSID,
			OrderID:     task.OrderID,
			OrderItemID: task.OrderItemID,
			Status:      string(task.Status),
			ScheduledAt: task.ScheduledAt,
			StartedAt:   task.StartedAt,
			FinishedAt:  task.FinishedAt,
		}
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		task.ID = row.ID
		task.CreatedAt = row.CreatedAt
		task.UpdatedAt = row.UpdatedAt
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO resize_tasks(vps_id,order_id,order_item_id,status,scheduled_at,started_at,finished_at) VALUES (?,?,?,?,?,?,?)`,
		task.VPSID, task.OrderID, task.OrderItemID, task.Status, task.ScheduledAt, task.StartedAt, task.FinishedAt)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	task.ID = id
	return nil
}

func (r *GormRepo) GetResizeTask(ctx context.Context, id int64) (domain.ResizeTask, error) {
	if r.gdb != nil {
		var row resizeTaskRow
		if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return domain.ResizeTask{}, r.ensure(err)
		}
		return domain.ResizeTask{
			ID:          row.ID,
			VPSID:       row.VPSID,
			OrderID:     row.OrderID,
			OrderItemID: row.OrderItemID,
			Status:      domain.ResizeTaskStatus(row.Status),
			ScheduledAt: row.ScheduledAt,
			StartedAt:   row.StartedAt,
			FinishedAt:  row.FinishedAt,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, vps_id, order_id, order_item_id, status, scheduled_at, started_at, finished_at, created_at, updated_at FROM resize_tasks WHERE id = ?`, id)
	return scanResizeTask(row)
}

func (r *GormRepo) UpdateResizeTask(ctx context.Context, task domain.ResizeTask) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&resizeTaskRow{}).Where("id = ?", task.ID).Updates(map[string]any{
			"status":       task.Status,
			"scheduled_at": task.ScheduledAt,
			"started_at":   task.StartedAt,
			"finished_at":  task.FinishedAt,
			"updated_at":   time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE resize_tasks SET status = ?, scheduled_at = ?, started_at = ?, finished_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		task.Status, task.ScheduledAt, task.StartedAt, task.FinishedAt, task.ID)
	return err
}

func (r *GormRepo) ListDueResizeTasks(ctx context.Context, limit int) ([]domain.ResizeTask, error) {
	if r.gdb != nil {
		if limit <= 0 {
			limit = 20
		}
		var rows []resizeTaskRow
		if err := r.gdb.WithContext(ctx).
			Where("status = ? AND (scheduled_at IS NULL OR scheduled_at <= CURRENT_TIMESTAMP)", domain.ResizeTaskStatusPending).
			Order("scheduled_at ASC, id ASC").
			Limit(limit).
			Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.ResizeTask, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.ResizeTask{
				ID:          row.ID,
				VPSID:       row.VPSID,
				OrderID:     row.OrderID,
				OrderItemID: row.OrderItemID,
				Status:      domain.ResizeTaskStatus(row.Status),
				ScheduledAt: row.ScheduledAt,
				StartedAt:   row.StartedAt,
				FinishedAt:  row.FinishedAt,
				CreatedAt:   row.CreatedAt,
				UpdatedAt:   row.UpdatedAt,
			})
		}
		return out, nil
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, vps_id, order_id, order_item_id, status, scheduled_at, started_at, finished_at, created_at, updated_at FROM resize_tasks WHERE status = ? AND (scheduled_at IS NULL OR scheduled_at <= CURRENT_TIMESTAMP) ORDER BY scheduled_at ASC, id ASC LIMIT ?`, domain.ResizeTaskStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ResizeTask
	for rows.Next() {
		task, err := scanResizeTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, nil
}

func (r *GormRepo) HasPendingResizeTask(ctx context.Context, vpsID int64) (bool, error) {
	if vpsID <= 0 {
		return false, nil
	}
	if r.gdb != nil {
		var total int64
		if err := r.gdb.WithContext(ctx).Model(&resizeTaskRow{}).Where("vps_id = ? AND status IN ?", vpsID, []string{string(domain.ResizeTaskStatusPending), string(domain.ResizeTaskStatusRunning)}).Count(&total).Error; err != nil {
			return false, err
		}
		return total > 0, nil
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM resize_tasks WHERE vps_id = ? AND status IN (?, ?)`, vpsID, domain.ResizeTaskStatusPending, domain.ResizeTaskStatusRunning).Scan(&total); err != nil {
		return false, err
	}
	return total > 0, nil
}

func (r *GormRepo) CreateSyncLog(ctx context.Context, log *domain.IntegrationSyncLog) error {
	if r.gdb != nil {
		row := integrationSyncLogRow{Target: log.Target, Mode: log.Mode, Status: log.Status, Message: log.Message}
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		log.ID = row.ID
		log.CreatedAt = row.CreatedAt
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO integration_sync_logs(target,mode,status,message) VALUES (?,?,?,?)`, log.Target, log.Mode, log.Status, log.Message)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	log.ID = id
	return nil
}

func (r *GormRepo) ListSyncLogs(ctx context.Context, target string, limit, offset int) ([]domain.IntegrationSyncLog, int, error) {
	if r.gdb != nil {
		q := r.gdb.WithContext(ctx).Model(&integrationSyncLogRow{})
		if target != "" {
			q = q.Where("target = ?", target)
		}
		var total int64
		if err := q.Count(&total).Error; err != nil {
			return nil, 0, err
		}
		var rows []integrationSyncLogRow
		if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, 0, err
		}
		out := make([]domain.IntegrationSyncLog, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.IntegrationSyncLog{
				ID:        row.ID,
				Target:    row.Target,
				Mode:      row.Mode,
				Status:    row.Status,
				Message:   row.Message,
				CreatedAt: row.CreatedAt,
			})
		}
		return out, int(total), nil
	}
	query := `SELECT id, target, mode, status, message, created_at FROM integration_sync_logs`
	args := []any{}
	if target != "" {
		query += " WHERE target = ?"
		args = append(args, target)
	}
	countQuery := "SELECT COUNT(1) FROM (" + query + ")"
	row := r.db.QueryRowContext(ctx, countQuery, args...)
	var total int
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}
	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.IntegrationSyncLog
	for rows.Next() {
		logEntry, err := scanIntegrationLog(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, logEntry)
	}
	return out, total, nil
}

func (r *GormRepo) AddAuditLog(ctx context.Context, log domain.AdminAuditLog) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Create(&adminAuditLogRow{
			AdminID:    log.AdminID,
			Action:     log.Action,
			TargetType: log.TargetType,
			TargetID:   log.TargetID,
			DetailJSON: log.DetailJSON,
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO admin_audit_logs(admin_id,action,target_type,target_id,detail_json) VALUES (?,?,?,?,?)`, log.AdminID, log.Action, log.TargetType, log.TargetID, log.DetailJSON)
	return err
}

func (r *GormRepo) ListAuditLogs(ctx context.Context, limit, offset int) ([]domain.AdminAuditLog, int, error) {
	if r.gdb != nil {
		var total int64
		if err := r.gdb.WithContext(ctx).Model(&adminAuditLogRow{}).Count(&total).Error; err != nil {
			return nil, 0, err
		}
		var rows []adminAuditLogRow
		if err := r.gdb.WithContext(ctx).Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, 0, err
		}
		out := make([]domain.AdminAuditLog, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.AdminAuditLog{
				ID:         row.ID,
				AdminID:    row.AdminID,
				Action:     row.Action,
				TargetType: row.TargetType,
				TargetID:   row.TargetID,
				DetailJSON: row.DetailJSON,
				CreatedAt:  row.CreatedAt,
			})
		}
		return out, int(total), nil
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM admin_audit_logs`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, admin_id, action, target_type, target_id, detail_json, created_at FROM admin_audit_logs ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.AdminAuditLog
	for rows.Next() {
		var log domain.AdminAuditLog
		if err := rows.Scan(&log.ID, &log.AdminID, &log.Action, &log.TargetType, &log.TargetID, &log.DetailJSON, &log.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, log)
	}
	return out, total, nil
}

func (r *GormRepo) ListPermissionGroups(ctx context.Context) ([]domain.PermissionGroup, error) {
	if r.gdb != nil {
		var rows []permissionGroupRow
		if err := r.gdb.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.PermissionGroup, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.PermissionGroup{
				ID:              row.ID,
				Name:            row.Name,
				Description:     row.Description,
				PermissionsJSON: row.PermissionsJSON,
				CreatedAt:       row.CreatedAt,
				UpdatedAt:       row.UpdatedAt,
			})
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, description, permissions_json, created_at, updated_at FROM permission_groups ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PermissionGroup
	for rows.Next() {
		var pg domain.PermissionGroup
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&pg.ID, &pg.Name, &pg.Description, &pg.PermissionsJSON, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			pg.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			pg.UpdatedAt = updatedAt.Time
		}
		out = append(out, pg)
	}
	return out, nil
}

func (r *GormRepo) GetPermissionGroup(ctx context.Context, id int64) (domain.PermissionGroup, error) {
	if r.gdb != nil {
		var row permissionGroupRow
		if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return domain.PermissionGroup{}, r.ensure(err)
		}
		return domain.PermissionGroup{
			ID:              row.ID,
			Name:            row.Name,
			Description:     row.Description,
			PermissionsJSON: row.PermissionsJSON,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		}, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, name, description, permissions_json, created_at, updated_at FROM permission_groups WHERE id = ?`, id)
	var pg domain.PermissionGroup
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	if err := row.Scan(&pg.ID, &pg.Name, &pg.Description, &pg.PermissionsJSON, &createdAt, &updatedAt); err != nil {
		return domain.PermissionGroup{}, r.ensure(err)
	}
	if createdAt.Valid {
		pg.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		pg.UpdatedAt = updatedAt.Time
	}
	return pg, nil
}

func (r *GormRepo) CreatePermissionGroup(ctx context.Context, group *domain.PermissionGroup) error {
	if r.gdb != nil {
		row := permissionGroupRow{
			Name:            group.Name,
			Description:     group.Description,
			PermissionsJSON: group.PermissionsJSON,
		}
		if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		group.ID = row.ID
		group.CreatedAt = row.CreatedAt
		group.UpdatedAt = row.UpdatedAt
		return nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO permission_groups(name,description,permissions_json,created_at,updated_at) VALUES (?,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, group.Name, group.Description, group.PermissionsJSON)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	group.ID = id
	return nil
}

func (r *GormRepo) UpdatePermissionGroup(ctx context.Context, group domain.PermissionGroup) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&permissionGroupRow{}).Where("id = ?", group.ID).Updates(map[string]any{
			"name":             group.Name,
			"description":      group.Description,
			"permissions_json": group.PermissionsJSON,
			"updated_at":       time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE permission_groups SET name = ?, description = ?, permissions_json = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, group.Name, group.Description, group.PermissionsJSON, group.ID)
	return err
}

func (r *GormRepo) DeletePermissionGroup(ctx context.Context, id int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Delete(&permissionGroupRow{}, id).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM permission_groups WHERE id = ?`, id)
	return err
}

func (r *GormRepo) CreatePasswordResetToken(ctx context.Context, token *domain.PasswordResetToken) error {
	if r.gdb != nil {
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
	res, err := r.db.ExecContext(ctx, `INSERT INTO password_reset_tokens(user_id,token,expires_at,used) VALUES (?,?,?,?)`, token.UserID, token.Token, token.ExpiresAt, boolToInt(token.Used))
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	token.ID = id
	return nil
}

func (r *GormRepo) GetPasswordResetToken(ctx context.Context, token string) (domain.PasswordResetToken, error) {
	if r.gdb != nil {
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
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, token, expires_at, used, created_at FROM password_reset_tokens WHERE token = ?`, token)
	var t domain.PasswordResetToken
	var used int
	if err := row.Scan(&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &used, &t.CreatedAt); err != nil {
		return domain.PasswordResetToken{}, r.ensure(err)
	}
	t.Used = used == 1
	return t, nil
}

func (r *GormRepo) MarkPasswordResetTokenUsed(ctx context.Context, tokenID int64) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&passwordResetTokenRow{}).Where("id = ?", tokenID).Update("used", 1).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE password_reset_tokens SET used = 1 WHERE id = ?`, tokenID)
	return err
}

func (r *GormRepo) DeleteExpiredTokens(ctx context.Context) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Where("expires_at < CURRENT_TIMESTAMP").Delete(&passwordResetTokenRow{}).Error
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM password_reset_tokens WHERE expires_at < CURRENT_TIMESTAMP`)
	return err
}

func (r *GormRepo) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	if r.gdb != nil {
		var rows []permissionModel
		if err := r.gdb.WithContext(ctx).Order("category, sort_order, code").Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]domain.Permission, 0, len(rows))
		for _, row := range rows {
			out = append(out, domain.Permission{
				ID:           row.ID,
				Code:         row.Code,
				Name:         row.Name,
				FriendlyName: row.FriendlyName,
				Category:     row.Category,
				ParentCode:   row.ParentCode,
				SortOrder:    row.SortOrder,
				CreatedAt:    row.CreatedAt,
				UpdatedAt:    row.UpdatedAt,
			})
		}
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, code, name, friendly_name, category, parent_code, sort_order, created_at, updated_at FROM permissions ORDER BY category, sort_order, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Permission
	for rows.Next() {
		var p domain.Permission
		var friendlyName sql.NullString
		var parentCode sql.NullString
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &friendlyName, &p.Category, &parentCode, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if friendlyName.Valid {
			p.FriendlyName = friendlyName.String
		}
		if parentCode.Valid {
			p.ParentCode = parentCode.String
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *GormRepo) GetPermissionByCode(ctx context.Context, code string) (domain.Permission, error) {
	if r.gdb != nil {
		var row permissionModel
		if err := r.gdb.WithContext(ctx).Where("code = ?", code).First(&row).Error; err != nil {
			return domain.Permission{}, r.ensure(err)
		}
		return domain.Permission{
			ID:           row.ID,
			Code:         row.Code,
			Name:         row.Name,
			FriendlyName: row.FriendlyName,
			Category:     row.Category,
			ParentCode:   row.ParentCode,
			SortOrder:    row.SortOrder,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		}, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, code, name, friendly_name, category, parent_code, sort_order, created_at, updated_at FROM permissions WHERE code = ?`, code)
	var p domain.Permission
	var friendlyName sql.NullString
	var parentCode sql.NullString
	if err := row.Scan(&p.ID, &p.Code, &p.Name, &friendlyName, &p.Category, &parentCode, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return domain.Permission{}, r.ensure(err)
	}
	if friendlyName.Valid {
		p.FriendlyName = friendlyName.String
	}
	if parentCode.Valid {
		p.ParentCode = parentCode.String
	}
	return p, nil
}

func (r *GormRepo) UpsertPermission(ctx context.Context, perm *domain.Permission) error {
	if r.gdb == nil {
		res, err := r.db.ExecContext(ctx, `
			INSERT INTO permissions(code, name, friendly_name, category, parent_code, sort_order) VALUES (?,?,?,?,?,?)
			ON CONFLICT(code) DO UPDATE SET name = excluded.name, friendly_name = excluded.friendly_name, category = excluded.category, parent_code = excluded.parent_code, sort_order = excluded.sort_order, updated_at = CURRENT_TIMESTAMP
		`, perm.Code, perm.Name, perm.FriendlyName, perm.Category, perm.ParentCode, perm.SortOrder)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		perm.ID = id
		return nil
	}
	m := permissionModel{
		Code:         perm.Code,
		Name:         perm.Name,
		FriendlyName: perm.FriendlyName,
		Category:     perm.Category,
		ParentCode:   perm.ParentCode,
		SortOrder:    perm.SortOrder,
		UpdatedAt:    time.Now(),
	}
	if err := r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "code"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "friendly_name", "category", "parent_code", "sort_order", "updated_at",
			}),
		}).
		Create(&m).Error; err != nil {
		return err
	}
	var got permissionModel
	if err := r.gdb.WithContext(ctx).Where("code = ?", perm.Code).First(&got).Error; err == nil {
		perm.ID = got.ID
	}
	return nil
}

func (r *GormRepo) UpdatePermissionName(ctx context.Context, code string, name string) error {
	if r.gdb != nil {
		return r.gdb.WithContext(ctx).Model(&permissionModel{}).Where("code = ?", code).Updates(map[string]any{
			"name":       name,
			"updated_at": time.Now(),
		}).Error
	}
	_, err := r.db.ExecContext(ctx, `UPDATE permissions SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE code = ?`, name, code)
	return err
}

func (r *GormRepo) RegisterPermissions(ctx context.Context, perms []domain.PermissionDefinition) error {
	for _, perm := range perms {
		existing, err := r.GetPermissionByCode(ctx, perm.Code)
		if err != nil && !errors.Is(err, usecase.ErrNotFound) {
			return err
		}
		if err == nil {
			if existing.Name != "" {
				perm.Name = existing.Name
			}
			if existing.FriendlyName != "" {
				perm.FriendlyName = existing.FriendlyName
			}
			if existing.Category != "" {
				perm.Category = existing.Category
			}
			if existing.ParentCode != "" {
				perm.ParentCode = existing.ParentCode
			}
			if existing.SortOrder != 0 {
				perm.SortOrder = existing.SortOrder
			}
		}
		upsert := domain.Permission{
			Code:         perm.Code,
			Name:         perm.Name,
			FriendlyName: perm.FriendlyName,
			Category:     perm.Category,
			ParentCode:   perm.ParentCode,
			SortOrder:    perm.SortOrder,
		}
		if err := r.UpsertPermission(ctx, &upsert); err != nil {
			return err
		}
	}
	return nil
}

func (r *GormRepo) ListCMSCategories(ctx context.Context, lang string, includeHidden bool) ([]domain.CMSCategory, error) {
	query := `SELECT id, key, name, lang, sort_order, visible, created_at, updated_at FROM cms_categories`
	args := []any{}
	conds := []string{}
	if lang != "" {
		conds = append(conds, "lang = ?")
		args = append(args, lang)
	}
	if !includeHidden {
		conds = append(conds, "visible = 1")
	}
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " ORDER BY sort_order, id"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CMSCategory
	for rows.Next() {
		var item domain.CMSCategory
		var visible int
		if err := rows.Scan(&item.ID, &item.Key, &item.Name, &item.Lang, &item.SortOrder, &visible, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Visible = visible == 1
		out = append(out, item)
	}
	return out, nil
}

func (r *GormRepo) GetCMSCategory(ctx context.Context, id int64) (domain.CMSCategory, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, key, name, lang, sort_order, visible, created_at, updated_at FROM cms_categories WHERE id = ?`, id)
	var item domain.CMSCategory
	var visible int
	if err := row.Scan(&item.ID, &item.Key, &item.Name, &item.Lang, &item.SortOrder, &visible, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.CMSCategory{}, r.ensure(err)
	}
	item.Visible = visible == 1
	return item, nil
}

func (r *GormRepo) GetCMSCategoryByKey(ctx context.Context, key, lang string) (domain.CMSCategory, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, key, name, lang, sort_order, visible, created_at, updated_at FROM cms_categories WHERE key = ? AND lang = ?`, key, lang)
	var item domain.CMSCategory
	var visible int
	if err := row.Scan(&item.ID, &item.Key, &item.Name, &item.Lang, &item.SortOrder, &visible, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.CMSCategory{}, r.ensure(err)
	}
	item.Visible = visible == 1
	return item, nil
}

func (r *GormRepo) CreateCMSCategory(ctx context.Context, category *domain.CMSCategory) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO cms_categories(key,name,lang,sort_order,visible) VALUES (?,?,?,?,?)`, category.Key, category.Name, category.Lang, category.SortOrder, boolToInt(category.Visible))
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	category.ID = id
	return nil
}

func (r *GormRepo) UpdateCMSCategory(ctx context.Context, category domain.CMSCategory) error {
	_, err := r.db.ExecContext(ctx, `UPDATE cms_categories SET key = ?, name = ?, lang = ?, sort_order = ?, visible = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, category.Key, category.Name, category.Lang, category.SortOrder, boolToInt(category.Visible), category.ID)
	return err
}

func (r *GormRepo) DeleteCMSCategory(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cms_categories WHERE id = ?`, id)
	return err
}

func (r *GormRepo) ListCMSPosts(ctx context.Context, filter usecase.CMSPostFilter) ([]domain.CMSPost, int, error) {
	conds := []string{}
	args := []any{}
	joinCategory := false
	if filter.CategoryID != nil {
		conds = append(conds, "p.category_id = ?")
		args = append(args, *filter.CategoryID)
	}
	if filter.CategoryKey != "" {
		joinCategory = true
		conds = append(conds, "c.key = ?")
		args = append(args, filter.CategoryKey)
	}
	if filter.Status != "" {
		conds = append(conds, "p.status = ?")
		args = append(args, filter.Status)
	}
	if filter.PublishedOnly {
		conds = append(conds, "p.status = 'published'")
	}
	if filter.Lang != "" {
		conds = append(conds, "p.lang = ?")
		args = append(args, filter.Lang)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	join := ""
	if joinCategory {
		join = " JOIN cms_categories c ON c.id = p.category_id"
	}
	countQuery := "SELECT COUNT(1) FROM cms_posts p" + join + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := "SELECT p.id, p.category_id, p.title, p.slug, p.summary, p.content_html, p.cover_url, p.lang, p.status, p.pinned, p.sort_order, p.published_at, p.created_at, p.updated_at FROM cms_posts p" + join + where + " ORDER BY p.pinned DESC, p.sort_order ASC, p.id DESC LIMIT ? OFFSET ?"
	limit := filter.Limit
	offset := filter.Offset
	if limit <= 0 {
		limit = 20
	}
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.CMSPost
	for rows.Next() {
		var item domain.CMSPost
		var pinned int
		var publishedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.CategoryID, &item.Title, &item.Slug, &item.Summary, &item.ContentHTML, &item.CoverURL, &item.Lang, &item.Status, &pinned, &item.SortOrder, &publishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		item.Pinned = pinned == 1
		if publishedAt.Valid {
			item.PublishedAt = &publishedAt.Time
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (r *GormRepo) GetCMSPost(ctx context.Context, id int64) (domain.CMSPost, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, category_id, title, slug, summary, content_html, cover_url, lang, status, pinned, sort_order, published_at, created_at, updated_at FROM cms_posts WHERE id = ?`, id)
	var item domain.CMSPost
	var pinned int
	var publishedAt sql.NullTime
	if err := row.Scan(&item.ID, &item.CategoryID, &item.Title, &item.Slug, &item.Summary, &item.ContentHTML, &item.CoverURL, &item.Lang, &item.Status, &pinned, &item.SortOrder, &publishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.CMSPost{}, r.ensure(err)
	}
	item.Pinned = pinned == 1
	if publishedAt.Valid {
		item.PublishedAt = &publishedAt.Time
	}
	return item, nil
}

func (r *GormRepo) GetCMSPostBySlug(ctx context.Context, slug string) (domain.CMSPost, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, category_id, title, slug, summary, content_html, cover_url, lang, status, pinned, sort_order, published_at, created_at, updated_at FROM cms_posts WHERE slug = ?`, slug)
	var item domain.CMSPost
	var pinned int
	var publishedAt sql.NullTime
	if err := row.Scan(&item.ID, &item.CategoryID, &item.Title, &item.Slug, &item.Summary, &item.ContentHTML, &item.CoverURL, &item.Lang, &item.Status, &pinned, &item.SortOrder, &publishedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.CMSPost{}, r.ensure(err)
	}
	item.Pinned = pinned == 1
	if publishedAt.Valid {
		item.PublishedAt = &publishedAt.Time
	}
	return item, nil
}

func (r *GormRepo) CreateCMSPost(ctx context.Context, post *domain.CMSPost) error {
	var publishedAt any
	if post.PublishedAt != nil {
		publishedAt = post.PublishedAt.UTC()
	} else {
		publishedAt = nil
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO cms_posts(category_id,title,slug,summary,content_html,cover_url,lang,status,pinned,sort_order,published_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, post.CategoryID, post.Title, post.Slug, post.Summary, post.ContentHTML, post.CoverURL, post.Lang, post.Status, boolToInt(post.Pinned), post.SortOrder, publishedAt)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	post.ID = id
	return nil
}

func (r *GormRepo) UpdateCMSPost(ctx context.Context, post domain.CMSPost) error {
	var publishedAt any
	if post.PublishedAt != nil {
		publishedAt = post.PublishedAt.UTC()
	} else {
		publishedAt = nil
	}
	_, err := r.db.ExecContext(ctx, `UPDATE cms_posts SET category_id = ?, title = ?, slug = ?, summary = ?, content_html = ?, cover_url = ?, lang = ?, status = ?, pinned = ?, sort_order = ?, published_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, post.CategoryID, post.Title, post.Slug, post.Summary, post.ContentHTML, post.CoverURL, post.Lang, post.Status, boolToInt(post.Pinned), post.SortOrder, publishedAt, post.ID)
	return err
}

func (r *GormRepo) DeleteCMSPost(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cms_posts WHERE id = ?`, id)
	return err
}

func (r *GormRepo) ListCMSBlocks(ctx context.Context, page, lang string, includeHidden bool) ([]domain.CMSBlock, error) {
	query := `SELECT id, page, type, title, subtitle, content_json, custom_html, lang, visible, sort_order, created_at, updated_at FROM cms_blocks`
	args := []any{}
	conds := []string{}
	if page != "" {
		conds = append(conds, "page = ?")
		args = append(args, page)
	}
	if lang != "" {
		conds = append(conds, "lang = ?")
		args = append(args, lang)
	}
	if !includeHidden {
		conds = append(conds, "visible = 1")
	}
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " ORDER BY sort_order, id"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CMSBlock
	for rows.Next() {
		var item domain.CMSBlock
		var visible int
		if err := rows.Scan(&item.ID, &item.Page, &item.Type, &item.Title, &item.Subtitle, &item.ContentJSON, &item.CustomHTML, &item.Lang, &visible, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Visible = visible == 1
		out = append(out, item)
	}
	return out, nil
}

func (r *GormRepo) GetCMSBlock(ctx context.Context, id int64) (domain.CMSBlock, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, page, type, title, subtitle, content_json, custom_html, lang, visible, sort_order, created_at, updated_at FROM cms_blocks WHERE id = ?`, id)
	var item domain.CMSBlock
	var visible int
	if err := row.Scan(&item.ID, &item.Page, &item.Type, &item.Title, &item.Subtitle, &item.ContentJSON, &item.CustomHTML, &item.Lang, &visible, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.CMSBlock{}, r.ensure(err)
	}
	item.Visible = visible == 1
	return item, nil
}

func (r *GormRepo) CreateCMSBlock(ctx context.Context, block *domain.CMSBlock) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO cms_blocks(page,type,title,subtitle,content_json,custom_html,lang,visible,sort_order) VALUES (?,?,?,?,?,?,?,?,?)`, block.Page, block.Type, block.Title, block.Subtitle, block.ContentJSON, block.CustomHTML, block.Lang, boolToInt(block.Visible), block.SortOrder)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	block.ID = id
	return nil
}

func (r *GormRepo) UpdateCMSBlock(ctx context.Context, block domain.CMSBlock) error {
	_, err := r.db.ExecContext(ctx, `UPDATE cms_blocks SET page = ?, type = ?, title = ?, subtitle = ?, content_json = ?, custom_html = ?, lang = ?, visible = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, block.Page, block.Type, block.Title, block.Subtitle, block.ContentJSON, block.CustomHTML, block.Lang, boolToInt(block.Visible), block.SortOrder, block.ID)
	return err
}

func (r *GormRepo) DeleteCMSBlock(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cms_blocks WHERE id = ?`, id)
	return err
}

func (r *GormRepo) CreateUpload(ctx context.Context, upload *domain.Upload) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO uploads(name,path,url,mime,size,uploader_id) VALUES (?,?,?,?,?,?)`, upload.Name, upload.Path, upload.URL, upload.Mime, upload.Size, upload.UploaderID)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	upload.ID = id
	return nil
}

func (r *GormRepo) ListUploads(ctx context.Context, limit, offset int) ([]domain.Upload, int, error) {
	if limit <= 0 {
		limit = 20
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM uploads`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, path, url, mime, size, uploader_id, created_at FROM uploads ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.Upload
	for rows.Next() {
		var item domain.Upload
		if err := rows.Scan(&item.ID, &item.Name, &item.Path, &item.URL, &item.Mime, &item.Size, &item.UploaderID, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (r *GormRepo) ListTickets(ctx context.Context, filter usecase.TicketFilter) ([]domain.Ticket, int, error) {
	conds := []string{}
	args := []any{}
	if filter.UserID != nil {
		conds = append(conds, "user_id = ?")
		args = append(args, *filter.UserID)
	}
	if filter.Status != "" {
		conds = append(conds, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Keyword != "" {
		conds = append(conds, "subject LIKE ?")
		args = append(args, "%"+filter.Keyword+"%")
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	countQuery := "SELECT COUNT(1) FROM tickets" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limit := filter.Limit
	offset := filter.Offset
	if limit <= 0 {
		limit = 20
	}
	query := `SELECT id, user_id, subject, status, last_reply_at, last_reply_by, last_reply_role, closed_at, created_at, updated_at,
		(SELECT COUNT(1) FROM ticket_resources tr WHERE tr.ticket_id = tickets.id) AS resource_count
		FROM tickets` + where + ` ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.Ticket
	for rows.Next() {
		var item domain.Ticket
		var lastReplyAt sql.NullTime
		var lastReplyBy sql.NullInt64
		var closedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.UserID, &item.Subject, &item.Status, &lastReplyAt, &lastReplyBy, &item.LastReplyRole, &closedAt, &item.CreatedAt, &item.UpdatedAt, &item.ResourceCount); err != nil {
			return nil, 0, err
		}
		if lastReplyAt.Valid {
			item.LastReplyAt = &lastReplyAt.Time
		}
		if lastReplyBy.Valid {
			val := lastReplyBy.Int64
			item.LastReplyBy = &val
		}
		if closedAt.Valid {
			item.ClosedAt = &closedAt.Time
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (r *GormRepo) GetTicket(ctx context.Context, id int64) (domain.Ticket, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, subject, status, last_reply_at, last_reply_by, last_reply_role, closed_at, created_at, updated_at FROM tickets WHERE id = ?`, id)
	var item domain.Ticket
	var lastReplyAt sql.NullTime
	var lastReplyBy sql.NullInt64
	var closedAt sql.NullTime
	if err := row.Scan(&item.ID, &item.UserID, &item.Subject, &item.Status, &lastReplyAt, &lastReplyBy, &item.LastReplyRole, &closedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.Ticket{}, r.ensure(err)
	}
	if lastReplyAt.Valid {
		item.LastReplyAt = &lastReplyAt.Time
	}
	if lastReplyBy.Valid {
		val := lastReplyBy.Int64
		item.LastReplyBy = &val
	}
	if closedAt.Valid {
		item.ClosedAt = &closedAt.Time
	}
	return item, nil
}

func (r *GormRepo) CreateTicketWithDetails(ctx context.Context, ticket *domain.Ticket, message *domain.TicketMessage, resources []domain.TicketResource) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	res, err := tx.ExecContext(ctx, `INSERT INTO tickets(user_id, subject, status, last_reply_at, last_reply_by, last_reply_role, closed_at) VALUES (?,?,?,?,?,?,?)`,
		ticket.UserID, ticket.Subject, ticket.Status, ticket.LastReplyAt, ticket.LastReplyBy, ticket.LastReplyRole, ticket.ClosedAt)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	ticket.ID = id
	message.TicketID = id
	_, err = tx.ExecContext(ctx, `INSERT INTO ticket_messages(ticket_id, sender_id, sender_role, sender_name, sender_qq, content) VALUES (?,?,?,?,?,?)`,
		message.TicketID, message.SenderID, message.SenderRole, message.SenderName, message.SenderQQ, message.Content)
	if err != nil {
		return err
	}
	for _, resItem := range resources {
		_, err = tx.ExecContext(ctx, `INSERT INTO ticket_resources(ticket_id, resource_type, resource_id, resource_name) VALUES (?,?,?,?)`, id, resItem.ResourceType, resItem.ResourceID, resItem.ResourceName)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *GormRepo) AddTicketMessage(ctx context.Context, message *domain.TicketMessage) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO ticket_messages(ticket_id, sender_id, sender_role, sender_name, sender_qq, content) VALUES (?,?,?,?,?,?)`,
		message.TicketID, message.SenderID, message.SenderRole, message.SenderName, message.SenderQQ, message.Content)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	message.ID = id
	_, err = r.db.ExecContext(ctx, `UPDATE tickets SET last_reply_at = CURRENT_TIMESTAMP, last_reply_by = ?, last_reply_role = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, message.SenderID, message.SenderRole, message.TicketID)
	return err
}

func (r *GormRepo) ListTicketMessages(ctx context.Context, ticketID int64) ([]domain.TicketMessage, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, ticket_id, sender_id, sender_role, sender_name, sender_qq, content, created_at FROM ticket_messages WHERE ticket_id = ? ORDER BY id ASC`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TicketMessage
	for rows.Next() {
		var item domain.TicketMessage
		if err := rows.Scan(&item.ID, &item.TicketID, &item.SenderID, &item.SenderRole, &item.SenderName, &item.SenderQQ, &item.Content, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *GormRepo) ListTicketResources(ctx context.Context, ticketID int64) ([]domain.TicketResource, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, ticket_id, resource_type, resource_id, resource_name, created_at FROM ticket_resources WHERE ticket_id = ? ORDER BY id ASC`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TicketResource
	for rows.Next() {
		var item domain.TicketResource
		if err := rows.Scan(&item.ID, &item.TicketID, &item.ResourceType, &item.ResourceID, &item.ResourceName, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *GormRepo) UpdateTicket(ctx context.Context, ticket domain.Ticket) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tickets SET subject = ?, status = ?, closed_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, ticket.Subject, ticket.Status, ticket.ClosedAt, ticket.ID)
	return err
}

func (r *GormRepo) DeleteTicket(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, `DELETE FROM ticket_messages WHERE ticket_id = ?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM ticket_resources WHERE ticket_id = ?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM tickets WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *GormRepo) CreateNotification(ctx context.Context, notification *domain.Notification) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO notifications(user_id,type,title,content,read_at) VALUES (?,?,?,?,?)`, notification.UserID, notification.Type, notification.Title, notification.Content, notification.ReadAt)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	notification.ID = id
	return nil
}

func (r *GormRepo) ListNotifications(ctx context.Context, filter usecase.NotificationFilter) ([]domain.Notification, int, error) {
	conds := []string{}
	args := []any{}
	if filter.UserID != nil {
		conds = append(conds, "user_id = ?")
		args = append(args, *filter.UserID)
	}
	switch filter.Status {
	case "unread":
		conds = append(conds, "read_at IS NULL")
	case "read":
		conds = append(conds, "read_at IS NOT NULL")
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	countQuery := "SELECT COUNT(1) FROM notifications" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limit := filter.Limit
	offset := filter.Offset
	if limit <= 0 {
		limit = 20
	}
	query := `SELECT id, user_id, type, title, content, read_at, created_at FROM notifications` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.Notification
	for rows.Next() {
		var item domain.Notification
		var readAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.UserID, &item.Type, &item.Title, &item.Content, &readAt, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		if readAt.Valid {
			item.ReadAt = &readAt.Time
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (r *GormRepo) CountUnread(ctx context.Context, userID int64) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM notifications WHERE user_id = ? AND read_at IS NULL`, userID).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *GormRepo) MarkNotificationRead(ctx context.Context, userID, notificationID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notifications SET read_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`, notificationID, userID)
	return err
}

func (r *GormRepo) MarkAllRead(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notifications SET read_at = CURRENT_TIMESTAMP WHERE user_id = ? AND read_at IS NULL`, userID)
	return err
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
	if r.gdb != nil {
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
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO push_tokens(user_id, platform, token, device_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?)
		ON CONFLICT(user_id, token) DO UPDATE SET
			platform = excluded.platform,
			device_id = excluded.device_id,
			updated_at = excluded.updated_at
	`, token.UserID, token.Platform, token.Token, token.DeviceID, token.CreatedAt, token.UpdatedAt)
	return err
}

func (r *GormRepo) DeletePushToken(ctx context.Context, userID int64, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM push_tokens WHERE user_id = ? AND token = ?`, userID, token)
	return err
}

func (r *GormRepo) ListPushTokensByUserIDs(ctx context.Context, userIDs []int64) ([]domain.PushToken, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, 0, len(userIDs))
	args := make([]any, 0, len(userIDs))
	for _, id := range userIDs {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	query := `SELECT id, user_id, platform, token, device_id, created_at, updated_at FROM push_tokens WHERE user_id IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PushToken
	for rows.Next() {
		var item domain.PushToken
		if err := rows.Scan(&item.ID, &item.UserID, &item.Platform, &item.Token, &item.DeviceID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *GormRepo) CreateRealNameVerification(ctx context.Context, record *domain.RealNameVerification) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO realname_verifications(user_id, real_name, id_number, status, provider, reason, verified_at) VALUES (?,?,?,?,?,?,?)`,
		record.UserID, record.RealName, record.IDNumber, record.Status, record.Provider, record.Reason, record.VerifiedAt)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	record.ID = id
	return nil
}

func (r *GormRepo) GetLatestRealNameVerification(ctx context.Context, userID int64) (domain.RealNameVerification, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, real_name, id_number, status, provider, reason, created_at, verified_at FROM realname_verifications WHERE user_id = ? ORDER BY id DESC LIMIT 1`, userID)
	return scanRealNameVerification(row)
}

func (r *GormRepo) ListRealNameVerifications(ctx context.Context, userID *int64, limit, offset int) ([]domain.RealNameVerification, int, error) {
	query := `SELECT id, user_id, real_name, id_number, status, provider, reason, created_at, verified_at FROM realname_verifications`
	args := []any{}
	if userID != nil {
		query += " WHERE user_id = ?"
		args = append(args, *userID)
	}
	countQuery := "SELECT COUNT(1) FROM (" + query + ")"
	row := r.db.QueryRowContext(ctx, countQuery, args...)
	var total int
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.RealNameVerification
	for rows.Next() {
		item, err := scanRealNameVerification(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (r *GormRepo) UpdateRealNameStatus(ctx context.Context, id int64, status string, reason string, verifiedAt *time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE realname_verifications SET status = ?, reason = ?, verified_at = ? WHERE id = ?`, status, reason, verifiedAt, id)
	return err
}

func (r *GormRepo) GetWallet(ctx context.Context, userID int64) (domain.Wallet, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, balance, updated_at FROM user_wallets WHERE user_id = ?`, userID)
	var w domain.Wallet
	if err := row.Scan(&w.ID, &w.UserID, &w.Balance, &w.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w = domain.Wallet{UserID: userID, Balance: 0}
			if err := r.UpsertWallet(ctx, &w); err != nil {
				return domain.Wallet{}, err
			}
			return r.GetWallet(ctx, userID)
		}
		return domain.Wallet{}, err
	}
	return w, nil
}

func (r *GormRepo) UpsertWallet(ctx context.Context, wallet *domain.Wallet) error {
	if r.gdb == nil {
		res, err := r.db.ExecContext(ctx, `INSERT INTO user_wallets(user_id,balance) VALUES (?,?) ON CONFLICT(user_id) DO UPDATE SET balance = excluded.balance, updated_at = CURRENT_TIMESTAMP`, wallet.UserID, wallet.Balance)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		if wallet.ID == 0 && id > 0 {
			wallet.ID = id
		}
		return nil
	}
	m := walletModel{
		ID:        wallet.ID,
		UserID:    wallet.UserID,
		Balance:   wallet.Balance,
		UpdatedAt: time.Now(),
	}
	if err := r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"balance", "updated_at"}),
		}).
		Create(&m).Error; err != nil {
		return err
	}
	var got walletModel
	if err := r.gdb.WithContext(ctx).Select("id").Where("user_id = ?", wallet.UserID).First(&got).Error; err == nil {
		wallet.ID = got.ID
	}
	return nil
}

func (r *GormRepo) AddWalletTransaction(ctx context.Context, txItem *domain.WalletTransaction) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO wallet_transactions(user_id,amount,type,ref_type,ref_id,note) VALUES (?,?,?,?,?,?)`, txItem.UserID, txItem.Amount, txItem.Type, txItem.RefType, txItem.RefID, txItem.Note)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	txItem.ID = id
	return nil
}

func (r *GormRepo) ListWalletTransactions(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletTransaction, int, error) {
	if limit <= 0 {
		limit = 20
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM wallet_transactions WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, amount, type, ref_type, ref_id, note, created_at FROM wallet_transactions WHERE user_id = ? ORDER BY id DESC LIMIT ? OFFSET ?`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.WalletTransaction
	for rows.Next() {
		var item domain.WalletTransaction
		if err := rows.Scan(&item.ID, &item.UserID, &item.Amount, &item.Type, &item.RefType, &item.RefID, &item.Note, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, nil
}

func (r *GormRepo) AdjustWalletBalance(ctx context.Context, userID int64, amount int64, txType, refType string, refID int64, note string) (wallet domain.Wallet, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Wallet{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	row := tx.QueryRowContext(ctx, `SELECT id, user_id, balance, updated_at FROM user_wallets WHERE user_id = ?`, userID)
	if err = row.Scan(&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_, err = tx.ExecContext(ctx, `INSERT INTO user_wallets(user_id,balance) VALUES (?,0)`, userID)
			if err != nil {
				return domain.Wallet{}, err
			}
			wallet = domain.Wallet{UserID: userID, Balance: 0}
		} else {
			return domain.Wallet{}, err
		}
	}
	newBalance := wallet.Balance + amount
	if newBalance < 0 {
		err = usecase.ErrInsufficientBalance
		return domain.Wallet{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE user_wallets SET balance = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`, newBalance, userID); err != nil {
		return domain.Wallet{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO wallet_transactions(user_id,amount,type,ref_type,ref_id,note) VALUES (?,?,?,?,?,?)`, userID, amount, txType, refType, refID, note); err != nil {
		return domain.Wallet{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.Wallet{}, err
	}
	wallet.Balance = newBalance
	return wallet, nil
}

func (r *GormRepo) HasWalletTransaction(ctx context.Context, userID int64, refType string, refID int64) (bool, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM wallet_transactions WHERE user_id = ? AND ref_type = ? AND ref_id = ?`, userID, refType, refID).Scan(&total); err != nil {
		return false, err
	}
	return total > 0, nil
}

func (r *GormRepo) CreateWalletOrder(ctx context.Context, order *domain.WalletOrder) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO wallet_orders(user_id,type,amount,currency,status,note,meta_json,created_at,updated_at) VALUES (?,?,?,?,?,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, order.UserID, order.Type, order.Amount, order.Currency, order.Status, order.Note, order.MetaJSON)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	order.ID = id
	return nil
}

func (r *GormRepo) GetWalletOrder(ctx context.Context, id int64) (domain.WalletOrder, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, type, amount, currency, status, note, meta_json, reviewed_by, review_reason, created_at, updated_at FROM wallet_orders WHERE id = ?`, id)
	return scanWalletOrder(row)
}

func (r *GormRepo) ListWalletOrders(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletOrder, int, error) {
	if limit <= 0 {
		limit = 20
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM wallet_orders WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, type, amount, currency, status, note, meta_json, reviewed_by, review_reason, created_at, updated_at FROM wallet_orders WHERE user_id = ? ORDER BY id DESC LIMIT ? OFFSET ?`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.WalletOrder
	for rows.Next() {
		order, err := scanWalletOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, order)
	}
	return out, total, nil
}

func (r *GormRepo) ListAllWalletOrders(ctx context.Context, status string, limit, offset int) ([]domain.WalletOrder, int, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `SELECT id, user_id, type, amount, currency, status, note, meta_json, reviewed_by, review_reason, created_at, updated_at FROM wallet_orders`
	args := []any{}
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	countQuery := "SELECT COUNT(1) FROM (" + query + ")"
	row := r.db.QueryRowContext(ctx, countQuery, args...)
	var total int
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}
	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.WalletOrder
	for rows.Next() {
		order, err := scanWalletOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, order)
	}
	return out, total, nil
}

func (r *GormRepo) UpdateWalletOrderStatus(ctx context.Context, id int64, status domain.WalletOrderStatus, reviewedBy *int64, reason string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE wallet_orders SET status = ?, reviewed_by = ?, review_reason = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, reviewedBy, reason, id)
	return err
}

func scanUser(row scanner) (domain.User, error) {
	var u domain.User
	var qq sql.NullString
	var avatar sql.NullString
	var phone sql.NullString
	var bio sql.NullString
	var intro sql.NullString
	var permissionGroupID sql.NullInt64
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &qq, &avatar, &phone, &bio, &intro, &permissionGroupID, &u.PasswordHash, &u.Role, &u.Status, &createdAt, &updatedAt); err != nil {
		return domain.User{}, rEnsure(err)
	}
	if qq.Valid {
		u.QQ = qq.String
	}
	if avatar.Valid {
		u.Avatar = avatar.String
	}
	if phone.Valid {
		u.Phone = phone.String
	}
	if bio.Valid {
		u.Bio = bio.String
	}
	if intro.Valid {
		u.Intro = intro.String
	}
	if permissionGroupID.Valid {
		u.PermissionGroupID = &permissionGroupID.Int64
	}
	if createdAt.Valid {
		u.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		u.UpdatedAt = updatedAt.Time
	}
	return u, nil
}

func scanWalletOrder(row scanner) (domain.WalletOrder, error) {
	var order domain.WalletOrder
	var reviewed sql.NullInt64
	var reason sql.NullString
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	if err := row.Scan(&order.ID, &order.UserID, &order.Type, &order.Amount, &order.Currency, &order.Status, &order.Note, &order.MetaJSON, &reviewed, &reason, &createdAt, &updatedAt); err != nil {
		return domain.WalletOrder{}, rEnsure(err)
	}
	if reviewed.Valid {
		order.ReviewedBy = &reviewed.Int64
	}
	if reason.Valid {
		order.ReviewReason = reason.String
	}
	if createdAt.Valid {
		order.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		order.UpdatedAt = updatedAt.Time
	}
	return order, nil
}

func scanOrder(row scanner) (domain.Order, error) {
	var o domain.Order
	var idem sql.NullString
	var approvedBy sql.NullInt64
	var approvedAt sql.NullTime
	var rejectedReason sql.NullString
	var pendingReason sql.NullString
	if err := row.Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Status, &o.TotalAmount, &o.Currency, &idem, &pendingReason, &approvedBy, &approvedAt, &rejectedReason, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return domain.Order{}, rEnsure(err)
	}
	if idem.Valid {
		o.IdempotencyKey = idem.String
	}
	if approvedBy.Valid {
		o.ApprovedBy = &approvedBy.Int64
	}
	if approvedAt.Valid {
		o.ApprovedAt = &approvedAt.Time
	}
	if pendingReason.Valid {
		o.PendingReason = pendingReason.String
	}
	if rejectedReason.Valid {
		o.RejectedReason = rejectedReason.String
	}
	return o, nil
}

func scanOrderItem(row scanner) (domain.OrderItem, error) {
	var item domain.OrderItem
	if err := row.Scan(&item.ID, &item.OrderID, &item.PackageID, &item.SystemID, &item.SpecJSON, &item.Qty, &item.Amount, &item.Status, &item.GoodsTypeID, &item.AutomationInstanceID, &item.Action, &item.DurationMonths, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.OrderItem{}, rEnsure(err)
	}
	return item, nil
}

func scanCartItem(row scanner) (domain.CartItem, error) {
	var item domain.CartItem
	if err := row.Scan(&item.ID, &item.UserID, &item.PackageID, &item.SystemID, &item.SpecJSON, &item.Qty, &item.Amount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.CartItem{}, rEnsure(err)
	}
	return item, nil
}

func scanSystemImage(row scanner) (domain.SystemImage, error) {
	var img domain.SystemImage
	var enabled int
	if err := row.Scan(&img.ID, &img.ImageID, &img.Name, &img.Type, &enabled, &img.CreatedAt, &img.UpdatedAt); err != nil {
		return domain.SystemImage{}, rEnsure(err)
	}
	img.Enabled = enabled == 1
	return img, nil
}

func scanVPSInstance(row scanner) (domain.VPSInstance, error) {
	var inst domain.VPSInstance
	var expire sql.NullTime
	var adminStatus sql.NullString
	var lastEmergency sql.NullTime
	var panelURL sql.NullString
	var accessInfo sql.NullString
	if err := row.Scan(&inst.ID, &inst.UserID, &inst.OrderItemID, &inst.AutomationInstanceID, &inst.GoodsTypeID, &inst.Name, &inst.Region, &inst.RegionID, &inst.LineID, &inst.PackageID, &inst.PackageName, &inst.CPU, &inst.MemoryGB, &inst.DiskGB, &inst.BandwidthMB, &inst.PortNum, &inst.MonthlyPrice, &inst.SpecJSON, &inst.SystemID, &inst.Status, &inst.AutomationState, &adminStatus, &expire, &panelURL, &accessInfo, &lastEmergency, &inst.CreatedAt, &inst.UpdatedAt); err != nil {
		return domain.VPSInstance{}, rEnsure(err)
	}
	if expire.Valid {
		inst.ExpireAt = &expire.Time
	}
	if adminStatus.Valid {
		inst.AdminStatus = domain.VPSAdminStatus(adminStatus.String)
	} else {
		inst.AdminStatus = domain.VPSAdminStatusNormal
	}
	if panelURL.Valid {
		inst.PanelURLCache = panelURL.String
	}
	if accessInfo.Valid {
		inst.AccessInfoJSON = accessInfo.String
	}
	if lastEmergency.Valid {
		inst.LastEmergencyRenewAt = &lastEmergency.Time
	}
	return inst, nil
}

func scanAPIKey(row scanner) (domain.APIKey, error) {
	var key domain.APIKey
	var lastUsed sql.NullTime
	var groupID sql.NullInt64
	if err := row.Scan(&key.ID, &key.Name, &key.KeyHash, &key.Status, &key.ScopesJSON, &groupID, &key.CreatedAt, &key.UpdatedAt, &lastUsed); err != nil {
		return domain.APIKey{}, rEnsure(err)
	}
	if groupID.Valid {
		v := groupID.Int64
		key.PermissionGroupID = &v
	}
	if lastUsed.Valid {
		key.LastUsedAt = &lastUsed.Time
	}
	return key, nil
}

func scanEmailTemplate(row scanner) (domain.EmailTemplate, error) {
	var tmpl domain.EmailTemplate
	var enabled int
	if err := row.Scan(&tmpl.ID, &tmpl.Name, &tmpl.Subject, &tmpl.Body, &enabled, &tmpl.CreatedAt, &tmpl.UpdatedAt); err != nil {
		return domain.EmailTemplate{}, rEnsure(err)
	}
	tmpl.Enabled = enabled == 1
	return tmpl, nil
}

func scanOrderPayment(row scanner) (domain.OrderPayment, error) {
	var pay domain.OrderPayment
	var reviewedBy sql.NullInt64
	var reviewReason sql.NullString
	var idem sql.NullString
	if err := row.Scan(&pay.ID, &pay.OrderID, &pay.UserID, &pay.Method, &pay.Amount, &pay.Currency, &pay.TradeNo, &pay.Note, &pay.ScreenshotURL, &pay.Status, &idem, &reviewedBy, &reviewReason, &pay.CreatedAt, &pay.UpdatedAt); err != nil {
		return domain.OrderPayment{}, rEnsure(err)
	}
	if idem.Valid {
		pay.IdempotencyKey = idem.String
	}
	if reviewedBy.Valid {
		pay.ReviewedBy = &reviewedBy.Int64
	}
	if reviewReason.Valid {
		pay.ReviewReason = reviewReason.String
	}
	return pay, nil
}

func scanRealNameVerification(row scanner) (domain.RealNameVerification, error) {
	var record domain.RealNameVerification
	var verifiedAt sql.NullTime
	if err := row.Scan(&record.ID, &record.UserID, &record.RealName, &record.IDNumber, &record.Status, &record.Provider, &record.Reason, &record.CreatedAt, &verifiedAt); err != nil {
		return domain.RealNameVerification{}, rEnsure(err)
	}
	if verifiedAt.Valid {
		record.VerifiedAt = &verifiedAt.Time
	}
	return record, nil
}

func scanBillingCycle(row scanner) (domain.BillingCycle, error) {
	var cycle domain.BillingCycle
	var active int
	if err := row.Scan(&cycle.ID, &cycle.Name, &cycle.Months, &cycle.Multiplier, &cycle.MinQty, &cycle.MaxQty, &active, &cycle.SortOrder, &cycle.CreatedAt, &cycle.UpdatedAt); err != nil {
		return domain.BillingCycle{}, rEnsure(err)
	}
	cycle.Active = active == 1
	return cycle, nil
}

func scanAutomationLog(row scanner) (domain.AutomationLog, error) {
	var logEntry domain.AutomationLog
	var success int
	if err := row.Scan(&logEntry.ID, &logEntry.OrderID, &logEntry.OrderItemID, &logEntry.Action, &logEntry.RequestJSON, &logEntry.ResponseJSON, &success, &logEntry.Message, &logEntry.CreatedAt); err != nil {
		return domain.AutomationLog{}, rEnsure(err)
	}
	logEntry.Success = success == 1
	return logEntry, nil
}

func scanProvisionJob(row scanner) (domain.ProvisionJob, error) {
	var job domain.ProvisionJob
	if err := row.Scan(&job.ID, &job.OrderID, &job.OrderItemID, &job.HostID, &job.HostName, &job.Status, &job.Attempts, &job.NextRunAt, &job.LastError, &job.CreatedAt, &job.UpdatedAt); err != nil {
		return domain.ProvisionJob{}, rEnsure(err)
	}
	return job, nil
}

func scanResizeTask(row scanner) (domain.ResizeTask, error) {
	var task domain.ResizeTask
	var scheduledAt sql.NullTime
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	if err := row.Scan(&task.ID, &task.VPSID, &task.OrderID, &task.OrderItemID, &task.Status, &scheduledAt, &startedAt, &finishedAt, &task.CreatedAt, &task.UpdatedAt); err != nil {
		return domain.ResizeTask{}, rEnsure(err)
	}
	if scheduledAt.Valid {
		task.ScheduledAt = &scheduledAt.Time
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		task.FinishedAt = &finishedAt.Time
	}
	return task, nil
}

func scanIntegrationLog(row scanner) (domain.IntegrationSyncLog, error) {
	var logEntry domain.IntegrationSyncLog
	if err := row.Scan(&logEntry.ID, &logEntry.Target, &logEntry.Mode, &logEntry.Status, &logEntry.Message, &logEntry.CreatedAt); err != nil {
		return domain.IntegrationSyncLog{}, rEnsure(err)
	}
	return logEntry, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func toUserRow(u domain.User) userRow {
	return userRow{
		ID:                u.ID,
		Username:          u.Username,
		Email:             u.Email,
		QQ:                u.QQ,
		Avatar:            u.Avatar,
		Phone:             u.Phone,
		Bio:               u.Bio,
		Intro:             u.Intro,
		PermissionGroupID: u.PermissionGroupID,
		PasswordHash:      u.PasswordHash,
		Role:              string(u.Role),
		Status:            string(u.Status),
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
	}
}

func fromUserRow(r userRow) domain.User {
	return domain.User{
		ID:                r.ID,
		Username:          r.Username,
		Email:             r.Email,
		QQ:                r.QQ,
		Avatar:            r.Avatar,
		Phone:             r.Phone,
		Bio:               r.Bio,
		Intro:             r.Intro,
		PermissionGroupID: r.PermissionGroupID,
		PasswordHash:      r.PasswordHash,
		Role:              domain.UserRole(r.Role),
		Status:            domain.UserStatus(r.Status),
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

func toCartItemRow(item domain.CartItem) cartItemRow {
	return cartItemRow{
		ID:        item.ID,
		UserID:    item.UserID,
		PackageID: item.PackageID,
		SystemID:  item.SystemID,
		SpecJSON:  item.SpecJSON,
		Qty:       item.Qty,
		Amount:    item.Amount,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func fromCartItemRow(r cartItemRow) domain.CartItem {
	return domain.CartItem{
		ID:        r.ID,
		UserID:    r.UserID,
		PackageID: r.PackageID,
		SystemID:  r.SystemID,
		SpecJSON:  r.SpecJSON,
		Qty:       r.Qty,
		Amount:    r.Amount,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func toOrderRow(order domain.Order) orderRow {
	var idem *string
	if strings.TrimSpace(order.IdempotencyKey) != "" {
		v := strings.TrimSpace(order.IdempotencyKey)
		idem = &v
	}
	return orderRow{
		ID:             order.ID,
		UserID:         order.UserID,
		OrderNo:        order.OrderNo,
		Status:         string(order.Status),
		TotalAmount:    order.TotalAmount,
		Currency:       order.Currency,
		IdempotencyKey: idem,
		PendingReason:  order.PendingReason,
		ApprovedBy:     order.ApprovedBy,
		ApprovedAt:     order.ApprovedAt,
		RejectedReason: order.RejectedReason,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
	}
}

func fromOrderRow(r orderRow) domain.Order {
	out := domain.Order{
		ID:             r.ID,
		UserID:         r.UserID,
		OrderNo:        r.OrderNo,
		Status:         domain.OrderStatus(r.Status),
		TotalAmount:    r.TotalAmount,
		Currency:       r.Currency,
		PendingReason:  r.PendingReason,
		ApprovedBy:     r.ApprovedBy,
		ApprovedAt:     r.ApprovedAt,
		RejectedReason: r.RejectedReason,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
	if r.IdempotencyKey != nil {
		out.IdempotencyKey = *r.IdempotencyKey
	}
	return out
}

func toOrderItemRow(item domain.OrderItem) orderItemRow {
	return orderItemRow{
		ID:                   item.ID,
		OrderID:              item.OrderID,
		PackageID:            item.PackageID,
		SystemID:             item.SystemID,
		SpecJSON:             item.SpecJSON,
		Qty:                  item.Qty,
		Amount:               item.Amount,
		Status:               string(item.Status),
		GoodsTypeID:          item.GoodsTypeID,
		AutomationInstanceID: item.AutomationInstanceID,
		Action:               item.Action,
		DurationMonths:       item.DurationMonths,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
	}
}

func fromOrderItemRow(r orderItemRow) domain.OrderItem {
	return domain.OrderItem{
		ID:                   r.ID,
		OrderID:              r.OrderID,
		PackageID:            r.PackageID,
		SystemID:             r.SystemID,
		SpecJSON:             r.SpecJSON,
		Qty:                  r.Qty,
		Amount:               r.Amount,
		Status:               domain.OrderItemStatus(r.Status),
		GoodsTypeID:          r.GoodsTypeID,
		AutomationInstanceID: r.AutomationInstanceID,
		Action:               r.Action,
		DurationMonths:       r.DurationMonths,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func toVPSInstanceRow(inst domain.VPSInstance) vpsInstanceRow {
	return vpsInstanceRow{
		ID:                   inst.ID,
		UserID:               inst.UserID,
		OrderItemID:          inst.OrderItemID,
		AutomationInstanceID: inst.AutomationInstanceID,
		GoodsTypeID:          inst.GoodsTypeID,
		Name:                 inst.Name,
		Region:               inst.Region,
		RegionID:             inst.RegionID,
		LineID:               inst.LineID,
		PackageID:            inst.PackageID,
		PackageName:          inst.PackageName,
		CPU:                  inst.CPU,
		MemoryGB:             inst.MemoryGB,
		DiskGB:               inst.DiskGB,
		BandwidthMbps:        inst.BandwidthMB,
		PortNum:              inst.PortNum,
		MonthlyPrice:         inst.MonthlyPrice,
		SpecJSON:             inst.SpecJSON,
		SystemID:             inst.SystemID,
		Status:               string(inst.Status),
		AutomationState:      inst.AutomationState,
		AdminStatus:          string(inst.AdminStatus),
		ExpireAt:             inst.ExpireAt,
		PanelURLCache:        inst.PanelURLCache,
		AccessInfoJSON:       inst.AccessInfoJSON,
		LastEmergencyRenewAt: inst.LastEmergencyRenewAt,
		CreatedAt:            inst.CreatedAt,
		UpdatedAt:            inst.UpdatedAt,
	}
}

func fromVPSInstanceRow(r vpsInstanceRow) domain.VPSInstance {
	return domain.VPSInstance{
		ID:                   r.ID,
		UserID:               r.UserID,
		OrderItemID:          r.OrderItemID,
		AutomationInstanceID: r.AutomationInstanceID,
		GoodsTypeID:          r.GoodsTypeID,
		Name:                 r.Name,
		Region:               r.Region,
		RegionID:             r.RegionID,
		LineID:               r.LineID,
		PackageID:            r.PackageID,
		PackageName:          r.PackageName,
		CPU:                  r.CPU,
		MemoryGB:             r.MemoryGB,
		DiskGB:               r.DiskGB,
		BandwidthMB:          r.BandwidthMbps,
		PortNum:              r.PortNum,
		MonthlyPrice:         r.MonthlyPrice,
		SpecJSON:             r.SpecJSON,
		SystemID:             r.SystemID,
		Status:               domain.VPSStatus(r.Status),
		AutomationState:      r.AutomationState,
		AdminStatus:          domain.VPSAdminStatus(r.AdminStatus),
		ExpireAt:             r.ExpireAt,
		PanelURLCache:        r.PanelURLCache,
		AccessInfoJSON:       r.AccessInfoJSON,
		LastEmergencyRenewAt: r.LastEmergencyRenewAt,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func toOrderPaymentRow(pay domain.OrderPayment) orderPaymentRow {
	var note *string
	if strings.TrimSpace(pay.Note) != "" {
		v := pay.Note
		note = &v
	}
	var screenshot *string
	if strings.TrimSpace(pay.ScreenshotURL) != "" {
		v := pay.ScreenshotURL
		screenshot = &v
	}
	var idem *string
	if strings.TrimSpace(pay.IdempotencyKey) != "" {
		v := pay.IdempotencyKey
		idem = &v
	}
	return orderPaymentRow{
		ID:             pay.ID,
		OrderID:        pay.OrderID,
		UserID:         pay.UserID,
		Method:         pay.Method,
		Amount:         pay.Amount,
		Currency:       pay.Currency,
		TradeNo:        pay.TradeNo,
		Note:           note,
		ScreenshotURL:  screenshot,
		Status:         string(pay.Status),
		IdempotencyKey: idem,
		ReviewedBy:     pay.ReviewedBy,
		ReviewReason:   pay.ReviewReason,
		CreatedAt:      pay.CreatedAt,
		UpdatedAt:      pay.UpdatedAt,
	}
}

func fromOrderPaymentRow(r orderPaymentRow) domain.OrderPayment {
	out := domain.OrderPayment{
		ID:           r.ID,
		OrderID:      r.OrderID,
		UserID:       r.UserID,
		Method:       r.Method,
		Amount:       r.Amount,
		Currency:     r.Currency,
		TradeNo:      r.TradeNo,
		Status:       domain.PaymentStatus(r.Status),
		ReviewedBy:   r.ReviewedBy,
		ReviewReason: r.ReviewReason,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
	if r.Note != nil {
		out.Note = *r.Note
	}
	if r.ScreenshotURL != nil {
		out.ScreenshotURL = *r.ScreenshotURL
	}
	if r.IdempotencyKey != nil {
		out.IdempotencyKey = *r.IdempotencyKey
	}
	return out
}

type settingModel struct {
	Key       string    `gorm:"primaryKey;column:key"`
	ValueJSON string    `gorm:"column:value_json"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (settingModel) TableName() string { return "settings" }

type pluginInstallationModel struct {
	ID              int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Category        string    `gorm:"column:category;uniqueIndex:idx_plugin_installations_cat_id_instance"`
	PluginID        string    `gorm:"column:plugin_id;uniqueIndex:idx_plugin_installations_cat_id_instance"`
	InstanceID      string    `gorm:"column:instance_id;uniqueIndex:idx_plugin_installations_cat_id_instance"`
	Enabled         int       `gorm:"column:enabled"`
	SignatureStatus string    `gorm:"column:signature_status"`
	ConfigCipher    string    `gorm:"column:config_cipher"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (pluginInstallationModel) TableName() string { return "plugin_installations" }

type pluginPaymentMethodModel struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Category   string    `gorm:"column:category"`
	PluginID   string    `gorm:"column:plugin_id"`
	InstanceID string    `gorm:"column:instance_id"`
	Method     string    `gorm:"column:method"`
	Enabled    int       `gorm:"column:enabled"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (pluginPaymentMethodModel) TableName() string { return "plugin_payment_methods" }

type provisionJobModel struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;column:id"`
	OrderID     int64     `gorm:"column:order_id"`
	OrderItemID int64     `gorm:"column:order_item_id;uniqueIndex:idx_provision_jobs_item"`
	HostID      int64     `gorm:"column:host_id"`
	HostName    string    `gorm:"column:host_name"`
	Status      string    `gorm:"column:status"`
	Attempts    int       `gorm:"column:attempts"`
	NextRunAt   time.Time `gorm:"column:next_run_at"`
	LastError   string    `gorm:"column:last_error"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (provisionJobModel) TableName() string { return "provision_jobs" }

type permissionModel struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Code         string    `gorm:"column:code;uniqueIndex"`
	Name         string    `gorm:"column:name"`
	FriendlyName string    `gorm:"column:friendly_name"`
	Category     string    `gorm:"column:category"`
	ParentCode   string    `gorm:"column:parent_code"`
	SortOrder    int       `gorm:"column:sort_order"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (permissionModel) TableName() string { return "permissions" }

type walletModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64     `gorm:"column:user_id;uniqueIndex"`
	Balance   int64     `gorm:"column:balance"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (walletModel) TableName() string { return "user_wallets" }

type pushTokenModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64     `gorm:"column:user_id"`
	Platform  string    `gorm:"column:platform"`
	Token     string    `gorm:"column:token"`
	DeviceID  string    `gorm:"column:device_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (pushTokenModel) TableName() string { return "push_tokens" }

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (r *GormRepo) ensure(err error) error {
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, gorm.ErrRecordNotFound) {
		return usecase.ErrNotFound
	}
	return err
}

func rEnsure(err error) error {
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, gorm.ErrRecordNotFound) {
		return usecase.ErrNotFound
	}
	return err
}

var (
	_ usecase.UserRepository               = (*GormRepo)(nil)
	_ usecase.CaptchaRepository            = (*GormRepo)(nil)
	_ usecase.CatalogRepository            = (*GormRepo)(nil)
	_ usecase.SystemImageRepository        = (*GormRepo)(nil)
	_ usecase.CartRepository               = (*GormRepo)(nil)
	_ usecase.OrderRepository              = (*GormRepo)(nil)
	_ usecase.OrderItemRepository          = (*GormRepo)(nil)
	_ usecase.PaymentRepository            = (*GormRepo)(nil)
	_ usecase.VPSRepository                = (*GormRepo)(nil)
	_ usecase.EventRepository              = (*GormRepo)(nil)
	_ usecase.APIKeyRepository             = (*GormRepo)(nil)
	_ usecase.SettingsRepository           = (*GormRepo)(nil)
	_ usecase.AuditRepository              = (*GormRepo)(nil)
	_ usecase.BillingCycleRepository       = (*GormRepo)(nil)
	_ usecase.AutomationLogRepository      = (*GormRepo)(nil)
	_ usecase.ProvisionJobRepository       = (*GormRepo)(nil)
	_ usecase.ResizeTaskRepository         = (*GormRepo)(nil)
	_ usecase.IntegrationLogRepository     = (*GormRepo)(nil)
	_ usecase.PermissionGroupRepository    = (*GormRepo)(nil)
	_ usecase.PasswordResetTokenRepository = (*GormRepo)(nil)
	_ usecase.PermissionRepository         = (*GormRepo)(nil)
	_ usecase.CMSCategoryRepository        = (*GormRepo)(nil)
	_ usecase.CMSPostRepository            = (*GormRepo)(nil)
	_ usecase.CMSBlockRepository           = (*GormRepo)(nil)
	_ usecase.UploadRepository             = (*GormRepo)(nil)
	_ usecase.TicketRepository             = (*GormRepo)(nil)
	_ usecase.NotificationRepository       = (*GormRepo)(nil)
	_ usecase.PushTokenRepository          = (*GormRepo)(nil)
	_ usecase.WalletRepository             = (*GormRepo)(nil)
	_ usecase.WalletOrderRepository        = (*GormRepo)(nil)
)
