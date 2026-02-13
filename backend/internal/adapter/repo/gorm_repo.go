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

func (r *GormRepo) UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error {

	return r.gdb.WithContext(ctx).Model(&userRow{}).Where("id = ?", id).Updates(map[string]any{
		"password_hash": passwordHash,
		"updated_at":    time.Now(),
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

func (r *GormRepo) ListGoodsTypes(ctx context.Context) ([]domain.GoodsType, error) {

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

func (r *GormRepo) GetGoodsType(ctx context.Context, id int64) (domain.GoodsType, error) {

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

func (r *GormRepo) CreateGoodsType(ctx context.Context, gt *domain.GoodsType) error {

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

func (r *GormRepo) UpdateGoodsType(ctx context.Context, gt domain.GoodsType) error {

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

func (r *GormRepo) DeleteGoodsType(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&goodsTypeRow{}, id).Error

}

func (r *GormRepo) ListRegions(ctx context.Context) ([]domain.Region, error) {

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

func (r *GormRepo) CreateRegion(ctx context.Context, region *domain.Region) error {

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

func (r *GormRepo) UpdateRegion(ctx context.Context, region domain.Region) error {

	return r.gdb.WithContext(ctx).Model(&regionRow{}).Where("id = ?", region.ID).Updates(map[string]any{
		"goods_type_id": region.GoodsTypeID,
		"code":          region.Code,
		"name":          region.Name,
		"active":        boolToInt(region.Active),
		"updated_at":    time.Now(),
	}).Error

}

func (r *GormRepo) DeleteRegion(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&regionRow{}, id).Error

}

func (r *GormRepo) ListPlanGroups(ctx context.Context) ([]domain.PlanGroup, error) {

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

func (r *GormRepo) CreatePlanGroup(ctx context.Context, plan *domain.PlanGroup) error {

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

func (r *GormRepo) UpdatePlanGroup(ctx context.Context, plan domain.PlanGroup) error {

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

func (r *GormRepo) DeletePlanGroup(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&planGroupRow{}, id).Error

}

func (r *GormRepo) ListPackages(ctx context.Context) ([]domain.Package, error) {

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

func (r *GormRepo) CreatePackage(ctx context.Context, pkg *domain.Package) error {

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

func (r *GormRepo) UpdatePackage(ctx context.Context, pkg domain.Package) error {

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

func (r *GormRepo) DeletePackage(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&packageRow{}, id).Error

}

func (r *GormRepo) GetPackage(ctx context.Context, id int64) (domain.Package, error) {

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

func (r *GormRepo) GetPlanGroup(ctx context.Context, id int64) (domain.PlanGroup, error) {

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

func (r *GormRepo) GetRegion(ctx context.Context, id int64) (domain.Region, error) {

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

func (r *GormRepo) ListSystemImages(ctx context.Context, lineID int64) ([]domain.SystemImage, error) {

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

func (r *GormRepo) ListAllSystemImages(ctx context.Context) ([]domain.SystemImage, error) {

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

func (r *GormRepo) GetSystemImage(ctx context.Context, id int64) (domain.SystemImage, error) {

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

func (r *GormRepo) CreateSystemImage(ctx context.Context, img *domain.SystemImage) error {

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

func (r *GormRepo) UpdateSystemImage(ctx context.Context, img domain.SystemImage) error {

	return r.gdb.WithContext(ctx).Model(&systemImageRow{}).Where("id = ?", img.ID).Updates(map[string]any{
		"image_id":   img.ImageID,
		"name":       img.Name,
		"type":       img.Type,
		"enabled":    boolToInt(img.Enabled),
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) DeleteSystemImage(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("system_image_id = ?", id).Delete(&lineSystemImageRow{}).Error; err != nil {
			return err
		}
		return tx.Delete(&systemImageRow{}, id).Error
	})

}

func (r *GormRepo) SetLineSystemImages(ctx context.Context, lineID int64, systemImageIDs []int64) error {

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

func (r *GormRepo) ListCartItems(ctx context.Context, userID int64) ([]domain.CartItem, error) {

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

func (r *GormRepo) AddCartItem(ctx context.Context, item *domain.CartItem) error {

	row := toCartItemRow(*item)
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*item = fromCartItemRow(row)
	return nil

}

func (r *GormRepo) UpdateCartItem(ctx context.Context, item domain.CartItem) error {

	return r.gdb.WithContext(ctx).Model(&cartItemRow{}).
		Where("id = ? AND user_id = ?", item.ID, item.UserID).
		Updates(map[string]any{"spec_json": item.SpecJSON, "qty": item.Qty, "amount": item.Amount, "updated_at": time.Now()}).Error

}

func (r *GormRepo) DeleteCartItem(ctx context.Context, id int64, userID int64) error {

	return r.gdb.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&cartItemRow{}).Error

}

func (r *GormRepo) ClearCart(ctx context.Context, userID int64) error {

	return r.gdb.WithContext(ctx).Where("user_id = ?", userID).Delete(&cartItemRow{}).Error

}

func (r *GormRepo) CreateOrder(ctx context.Context, order *domain.Order) error {

	row := toOrderRow(*order)
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*order = fromOrderRow(row)
	return nil

}

func (r *GormRepo) CreateOrderFromCartAtomic(ctx context.Context, order domain.Order, items []domain.OrderItem) (created domain.Order, createdItems []domain.OrderItem, err error) {

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

func (r *GormRepo) GetOrder(ctx context.Context, id int64) (domain.Order, error) {

	var row orderRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.Order{}, r.ensure(err)
	}
	return fromOrderRow(row), nil

}

func (r *GormRepo) GetOrderByNo(ctx context.Context, orderNo string) (domain.Order, error) {

	var row orderRow
	if err := r.gdb.WithContext(ctx).Where("order_no = ?", orderNo).First(&row).Error; err != nil {
		return domain.Order{}, r.ensure(err)
	}
	return fromOrderRow(row), nil

}

func (r *GormRepo) GetOrderByIdempotencyKey(ctx context.Context, userID int64, key string) (domain.Order, error) {

	var row orderRow
	if err := r.gdb.WithContext(ctx).Where("user_id = ? AND idempotency_key = ?", userID, key).First(&row).Error; err != nil {
		return domain.Order{}, r.ensure(err)
	}
	return fromOrderRow(row), nil

}

func (r *GormRepo) UpdateOrderStatus(ctx context.Context, id int64, status domain.OrderStatus) error {

	return r.gdb.WithContext(ctx).Model(&orderRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) UpdateOrderMeta(ctx context.Context, order domain.Order) error {

	return r.gdb.WithContext(ctx).Model(&orderRow{}).Where("id = ?", order.ID).Updates(map[string]any{
		"status":          order.Status,
		"pending_reason":  order.PendingReason,
		"approved_by":     order.ApprovedBy,
		"approved_at":     order.ApprovedAt,
		"rejected_reason": order.RejectedReason,
		"updated_at":      time.Now(),
	}).Error

}

func (r *GormRepo) ApproveResizeOrderWithTasks(ctx context.Context, order domain.Order, items []domain.OrderItem, tasks []*domain.ResizeTask) error {

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

func (r *GormRepo) ListOrders(ctx context.Context, filter usecase.OrderFilter, limit, offset int) ([]domain.Order, int, error) {

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

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func (r *GormRepo) DeleteOrder(ctx context.Context, id int64) (err error) {

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

func (r *GormRepo) CreateOrderItems(ctx context.Context, items []domain.OrderItem) error {

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

func (r *GormRepo) ListOrderItems(ctx context.Context, orderID int64) ([]domain.OrderItem, error) {

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

func (r *GormRepo) GetOrderItem(ctx context.Context, id int64) (domain.OrderItem, error) {

	var row orderItemRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.OrderItem{}, r.ensure(err)
	}
	return fromOrderItemRow(row), nil

}

func (r *GormRepo) HasPendingRenewOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return r.hasExclusiveVPSOrderInProgress(ctx, userID, vpsID)
}

func (r *GormRepo) HasPendingResizeOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return r.hasExclusiveVPSOrderInProgress(ctx, userID, vpsID)
}

func (r *GormRepo) HasPendingRefundOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return r.hasExclusiveVPSOrderInProgress(ctx, userID, vpsID)
}

func (r *GormRepo) hasExclusiveVPSOrderInProgress(ctx context.Context, userID, vpsID int64) (bool, error) {
	if vpsID <= 0 {
		return false, nil
	}
	progressStatuses := []string{
		string(domain.OrderStatusPendingPayment),
		string(domain.OrderStatusPendingReview),
		string(domain.OrderStatusApproved),
		string(domain.OrderStatusProvisioning),
	}
	actions := []string{"renew", "emergency_renew", "resize", "refund"}
	var rows []orderItemRow
	if err := r.gdb.WithContext(ctx).
		Joins("JOIN orders o ON o.id = order_items.order_id").
		Where("o.user_id = ? AND order_items.action IN ? AND o.status IN ?",
			userID, actions, progressStatuses).
		Order("order_items.id DESC").
		Limit(50).
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

func (r *GormRepo) UpdateOrderItemStatus(ctx context.Context, id int64, status domain.OrderItemStatus) error {

	return r.gdb.WithContext(ctx).Model(&orderItemRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) UpdateOrderItemAutomation(ctx context.Context, id int64, automationID string) error {

	return r.gdb.WithContext(ctx).Model(&orderItemRow{}).Where("id = ?", id).Updates(map[string]any{
		"automation_instance_id": automationID,
		"updated_at":             time.Now(),
	}).Error

}

func (r *GormRepo) CreateInstance(ctx context.Context, inst *domain.VPSInstance) error {

	row := toVPSInstanceRow(*inst)
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*inst = fromVPSInstanceRow(row)
	return nil

}

func (r *GormRepo) GetInstance(ctx context.Context, id int64) (domain.VPSInstance, error) {

	var row vpsInstanceRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.VPSInstance{}, r.ensure(err)
	}
	return fromVPSInstanceRow(row), nil

}

func (r *GormRepo) GetInstanceByOrderItem(ctx context.Context, orderItemID int64) (domain.VPSInstance, error) {

	var row vpsInstanceRow
	if err := r.gdb.WithContext(ctx).Where("order_item_id = ?", orderItemID).First(&row).Error; err != nil {
		return domain.VPSInstance{}, r.ensure(err)
	}
	return fromVPSInstanceRow(row), nil

}

func (r *GormRepo) ListInstancesByUser(ctx context.Context, userID int64) ([]domain.VPSInstance, error) {

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

func (r *GormRepo) ListInstances(ctx context.Context, limit, offset int) ([]domain.VPSInstance, int, error) {

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

func (r *GormRepo) ListInstancesExpiring(ctx context.Context, before time.Time) ([]domain.VPSInstance, error) {

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

func (r *GormRepo) DeleteInstance(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&vpsInstanceRow{}, id).Error

}

func (r *GormRepo) UpdateInstanceStatus(ctx context.Context, id int64, status domain.VPSStatus, automationState int) error {

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

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
		"admin_status": status,
		"updated_at":   time.Now(),
	}).Error

}

func (r *GormRepo) UpdateInstanceExpireAt(ctx context.Context, id int64, expireAt time.Time) error {

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
		"expire_at":  expireAt,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) UpdateInstancePanelCache(ctx context.Context, id int64, panelURL string) error {

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
		"panel_url_cache": panelURL,
		"updated_at":      time.Now(),
	}).Error

}

func (r *GormRepo) UpdateInstanceSpec(ctx context.Context, id int64, specJSON string) error {

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
		"spec_json":  specJSON,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) UpdateInstanceAccessInfo(ctx context.Context, id int64, accessJSON string) error {

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
		"access_info_json": accessJSON,
		"updated_at":       time.Now(),
	}).Error

}

func (r *GormRepo) UpdateInstanceEmergencyRenewAt(ctx context.Context, id int64, at time.Time) error {

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
		"last_emergency_renew_at": at,
		"updated_at":              time.Now(),
	}).Error

}

func (r *GormRepo) UpdateInstanceLocal(ctx context.Context, inst domain.VPSInstance) error {

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", inst.ID).Updates(map[string]any{
		"automation_instance_id": inst.AutomationInstanceID,
		"name":                   inst.Name,
		"package_id":             inst.PackageID,
		"package_name":           inst.PackageName,
		"cpu":                    inst.CPU,
		"memory_gb":              inst.MemoryGB,
		"disk_gb":                inst.DiskGB,
		"bandwidth_mbps":         inst.BandwidthMB,
		"port_num":               inst.PortNum,
		"monthly_price":          inst.MonthlyPrice,
		"spec_json":              inst.SpecJSON,
		"system_id":              inst.SystemID,
		"status":                 inst.Status,
		"admin_status":           inst.AdminStatus,
		"panel_url_cache":        inst.PanelURLCache,
		"access_info_json":       inst.AccessInfoJSON,
		"updated_at":             time.Now(),
	}).Error

}

func (r *GormRepo) AppendEvent(ctx context.Context, orderID int64, eventType string, dataJSON string) (domain.OrderEvent, error) {

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

func (r *GormRepo) ListEventsAfter(ctx context.Context, orderID int64, afterSeq int64, limit int) ([]domain.OrderEvent, error) {

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

func (r *GormRepo) CreatePayment(ctx context.Context, payment *domain.OrderPayment) error {
	tradeNo := strings.TrimSpace(payment.TradeNo)
	if tradeNo == "" {
		// Keep external semantics for empty trade_no while avoiding unique-key collisions.
		tradeNo = fmt.Sprintf("pending-%d-%d", payment.OrderID, time.Now().UnixNano())
	}
	row := toOrderPaymentRow(*payment)
	row.TradeNo = tradeNo
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*payment = fromOrderPaymentRow(row)
	return nil
}

func (r *GormRepo) ListPaymentsByOrder(ctx context.Context, orderID int64) ([]domain.OrderPayment, error) {

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

func (r *GormRepo) GetPaymentByTradeNo(ctx context.Context, tradeNo string) (domain.OrderPayment, error) {
	if strings.TrimSpace(tradeNo) == "" {
		return domain.OrderPayment{}, sql.ErrNoRows
	}
	var row orderPaymentRow
	if err := r.gdb.WithContext(ctx).Where("trade_no = ?", tradeNo).First(&row).Error; err != nil {
		return domain.OrderPayment{}, r.ensure(err)
	}
	return fromOrderPaymentRow(row), nil
}

func (r *GormRepo) GetPaymentByIdempotencyKey(ctx context.Context, orderID int64, key string) (domain.OrderPayment, error) {

	var row orderPaymentRow
	if err := r.gdb.WithContext(ctx).Where("order_id = ? AND idempotency_key = ?", orderID, key).First(&row).Error; err != nil {
		return domain.OrderPayment{}, r.ensure(err)
	}
	return fromOrderPaymentRow(row), nil

}

func (r *GormRepo) UpdatePaymentStatus(ctx context.Context, id int64, status domain.PaymentStatus, reviewedBy *int64, reason string) error {

	return r.gdb.WithContext(ctx).Model(&orderPaymentRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":        status,
		"reviewed_by":   reviewedBy,
		"review_reason": reason,
		"updated_at":    time.Now(),
	}).Error

}

func (r *GormRepo) UpdatePaymentTradeNo(ctx context.Context, id int64, tradeNo string) error {

	return r.gdb.WithContext(ctx).Model(&orderPaymentRow{}).Where("id = ?", id).Updates(map[string]any{
		"trade_no":   tradeNo,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) ListPayments(ctx context.Context, filter usecase.PaymentFilter, limit, offset int) ([]domain.OrderPayment, int, error) {

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

func (r *GormRepo) CreateAPIKey(ctx context.Context, key *domain.APIKey) error {

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

func (r *GormRepo) GetAPIKeyByHash(ctx context.Context, keyHash string) (domain.APIKey, error) {

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

func (r *GormRepo) ListAPIKeys(ctx context.Context, limit, offset int) ([]domain.APIKey, int, error) {

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

func (r *GormRepo) UpdateAPIKeyStatus(ctx context.Context, id int64, status domain.APIKeyStatus) error {

	return r.gdb.WithContext(ctx).Model(&apiKeyRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) TouchAPIKey(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Model(&apiKeyRow{}).Where("id = ?", id).Update("last_used_at", time.Now()).Error

}

func (r *GormRepo) GetSetting(ctx context.Context, key string) (domain.Setting, error) {

	var m settingModel
	if err := r.gdb.WithContext(ctx).Where("`key` = ?", key).First(&m).Error; err != nil {
		return domain.Setting{}, r.ensure(err)
	}
	return domain.Setting{Key: m.Key, ValueJSON: m.ValueJSON, UpdatedAt: m.UpdatedAt}, nil

}

func (r *GormRepo) UpsertSetting(ctx context.Context, setting domain.Setting) error {
	m := settingModel{Key: setting.Key, ValueJSON: setting.ValueJSON, UpdatedAt: time.Now()}
	return r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value_json", "updated_at"}),
		}).
		Create(&m).Error
}

func (r *GormRepo) ListSettings(ctx context.Context) ([]domain.Setting, error) {

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

func (r *GormRepo) UpsertPluginInstallation(ctx context.Context, inst *domain.PluginInstallation) error {
	if inst == nil || strings.TrimSpace(inst.Category) == "" || strings.TrimSpace(inst.PluginID) == "" || strings.TrimSpace(inst.InstanceID) == "" {
		return usecase.ErrInvalidInput
	}
	m := pluginInstallationRow{
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
	var row pluginInstallationRow
	if err := r.gdb.WithContext(ctx).
		Where("category = ? AND plugin_id = ? AND instance_id = ?", category, pluginID, instanceID).
		First(&row).Error; err != nil {
		return domain.PluginInstallation{}, r.ensure(err)
	}
	return domain.PluginInstallation{
		ID:              row.ID,
		Category:        row.Category,
		PluginID:        row.PluginID,
		InstanceID:      row.InstanceID,
		Enabled:         row.Enabled == 1,
		SignatureStatus: domain.PluginSignatureStatus(row.SignatureStatus),
		ConfigCipher:    row.ConfigCipher,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func (r *GormRepo) ListPluginInstallations(ctx context.Context) ([]domain.PluginInstallation, error) {
	var rows []pluginInstallationRow
	if err := r.gdb.WithContext(ctx).Order("category ASC, plugin_id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PluginInstallation, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.PluginInstallation{
			ID:              row.ID,
			Category:        row.Category,
			PluginID:        row.PluginID,
			InstanceID:      row.InstanceID,
			Enabled:         row.Enabled == 1,
			SignatureStatus: domain.PluginSignatureStatus(row.SignatureStatus),
			ConfigCipher:    row.ConfigCipher,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) DeletePluginInstallation(ctx context.Context, category, pluginID, instanceID string) error {
	return r.gdb.WithContext(ctx).
		Where("category = ? AND plugin_id = ? AND instance_id = ?", category, pluginID, instanceID).
		Delete(&pluginInstallationRow{}).Error
}

func (r *GormRepo) ListPluginPaymentMethods(ctx context.Context, category, pluginID, instanceID string) ([]domain.PluginPaymentMethod, error) {
	var rows []pluginPaymentMethodRow
	if err := r.gdb.WithContext(ctx).
		Where("category = ? AND plugin_id = ? AND instance_id = ?", category, pluginID, instanceID).
		Order("method ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PluginPaymentMethod, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.PluginPaymentMethod{
			ID:         row.ID,
			Category:   row.Category,
			PluginID:   row.PluginID,
			InstanceID: row.InstanceID,
			Method:     row.Method,
			Enabled:    row.Enabled == 1,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) UpsertPluginPaymentMethod(ctx context.Context, m *domain.PluginPaymentMethod) error {
	if m == nil || strings.TrimSpace(m.Category) == "" || strings.TrimSpace(m.PluginID) == "" || strings.TrimSpace(m.InstanceID) == "" || strings.TrimSpace(m.Method) == "" {
		return usecase.ErrInvalidInput
	}
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

func (r *GormRepo) DeletePluginPaymentMethod(ctx context.Context, category, pluginID, instanceID, method string) error {

	return r.gdb.WithContext(ctx).
		Where("category = ? AND plugin_id = ? AND instance_id = ? AND method = ?", category, pluginID, instanceID, method).
		Delete(&pluginPaymentMethodModel{}).Error

}

func (r *GormRepo) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {

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

func (r *GormRepo) GetEmailTemplate(ctx context.Context, id int64) (domain.EmailTemplate, error) {

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

func (r *GormRepo) UpsertEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error {

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

func (r *GormRepo) DeleteEmailTemplate(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&emailTemplateRow{}, id).Error

}

func (r *GormRepo) ListBillingCycles(ctx context.Context) ([]domain.BillingCycle, error) {

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

func (r *GormRepo) GetBillingCycle(ctx context.Context, id int64) (domain.BillingCycle, error) {

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

func (r *GormRepo) CreateBillingCycle(ctx context.Context, cycle *domain.BillingCycle) error {

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

func (r *GormRepo) UpdateBillingCycle(ctx context.Context, cycle domain.BillingCycle) error {

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

func (r *GormRepo) DeleteBillingCycle(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&billingCycleRow{}, id).Error

}

func (r *GormRepo) CreateAutomationLog(ctx context.Context, log *domain.AutomationLog) error {
	row := automationLogRow{
		OrderID:      log.OrderID,
		OrderItemID:  log.OrderItemID,
		Action:       log.Action,
		RequestJSON:  log.RequestJSON,
		ResponseJSON: log.ResponseJSON,
		Success:      boolToInt(log.Success),
		Message:      log.Message,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	log.ID = row.ID
	log.CreatedAt = row.CreatedAt
	return nil
}

func (r *GormRepo) ListAutomationLogs(ctx context.Context, orderID int64, limit, offset int) ([]domain.AutomationLog, int, error) {
	q := r.gdb.WithContext(ctx).Model(&automationLogRow{})
	if orderID > 0 {
		q = q.Where("order_id = ?", orderID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	var rows []automationLogRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.AutomationLog, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.AutomationLog{
			ID:           row.ID,
			OrderID:      row.OrderID,
			OrderItemID:  row.OrderItemID,
			Action:       row.Action,
			RequestJSON:  row.RequestJSON,
			ResponseJSON: row.ResponseJSON,
			Success:      row.Success == 1,
			Message:      row.Message,
			CreatedAt:    row.CreatedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) PurgeAutomationLogs(ctx context.Context, before time.Time) error {
	return r.gdb.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&automationLogRow{}).Error
}

func (r *GormRepo) CreateOrUpdateProvisionJob(ctx context.Context, job *domain.ProvisionJob) error {
	now := time.Now()
	m := provisionJobRow{
		ID:          job.ID,
		OrderID:     job.OrderID,
		OrderItemID: job.OrderItemID,
		HostID:      job.HostID,
		HostName:    job.HostName,
		Status:      job.Status,
		Attempts:    job.Attempts,
		NextRunAt:   job.NextRunAt,
		LastError:   job.LastError,
		CreatedAt:   now,
		UpdatedAt:   now,
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
	var got provisionJobRow
	if err := r.gdb.WithContext(ctx).Select("id").Where("order_item_id = ?", job.OrderItemID).First(&got).Error; err == nil {
		job.ID = got.ID
	}
	return nil
}

func (r *GormRepo) ListDueProvisionJobs(ctx context.Context, limit int) ([]domain.ProvisionJob, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []provisionJobRow
	if err := r.gdb.WithContext(ctx).
		Where("status IN ? AND next_run_at <= ?", []string{"pending", "retry", "running"}, time.Now()).
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ProvisionJob, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ProvisionJob{
			ID:          row.ID,
			OrderID:     row.OrderID,
			OrderItemID: row.OrderItemID,
			HostID:      row.HostID,
			HostName:    row.HostName,
			Status:      row.Status,
			Attempts:    row.Attempts,
			NextRunAt:   row.NextRunAt,
			LastError:   row.LastError,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) UpdateProvisionJob(ctx context.Context, job domain.ProvisionJob) error {
	return r.gdb.WithContext(ctx).Model(&provisionJobRow{}).Where("id = ?", job.ID).Updates(map[string]any{
		"status":      job.Status,
		"attempts":    job.Attempts,
		"next_run_at": job.NextRunAt,
		"last_error":  job.LastError,
		"updated_at":  time.Now(),
	}).Error
}

func (r *GormRepo) CreateTaskRun(ctx context.Context, run *domain.ScheduledTaskRun) error {

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

func (r *GormRepo) UpdateTaskRun(ctx context.Context, run domain.ScheduledTaskRun) error {

	return r.gdb.WithContext(ctx).Model(&scheduledTaskRunRow{}).Where("id = ?", run.ID).Updates(map[string]any{
		"status":       run.Status,
		"finished_at":  run.FinishedAt,
		"duration_sec": run.DurationSec,
		"message":      run.Message,
	}).Error

}

func (r *GormRepo) ListTaskRuns(ctx context.Context, key string, limit int) ([]domain.ScheduledTaskRun, error) {

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

func (r *GormRepo) CreateResizeTask(ctx context.Context, task *domain.ResizeTask) error {

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

func (r *GormRepo) GetResizeTask(ctx context.Context, id int64) (domain.ResizeTask, error) {

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

func (r *GormRepo) UpdateResizeTask(ctx context.Context, task domain.ResizeTask) error {

	return r.gdb.WithContext(ctx).Model(&resizeTaskRow{}).Where("id = ?", task.ID).Updates(map[string]any{
		"status":       task.Status,
		"scheduled_at": task.ScheduledAt,
		"started_at":   task.StartedAt,
		"finished_at":  task.FinishedAt,
		"updated_at":   time.Now(),
	}).Error

}

func (r *GormRepo) ListDueResizeTasks(ctx context.Context, limit int) ([]domain.ResizeTask, error) {

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

func (r *GormRepo) HasPendingResizeTask(ctx context.Context, vpsID int64) (bool, error) {
	if vpsID <= 0 {
		return false, nil
	}
	var total int64
	if err := r.gdb.WithContext(ctx).Model(&resizeTaskRow{}).Where("vps_id = ? AND status IN ?", vpsID, []string{string(domain.ResizeTaskStatusPending), string(domain.ResizeTaskStatusRunning)}).Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

func (r *GormRepo) CreateSyncLog(ctx context.Context, log *domain.IntegrationSyncLog) error {

	row := integrationSyncLogRow{Target: log.Target, Mode: log.Mode, Status: log.Status, Message: log.Message}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	log.ID = row.ID
	log.CreatedAt = row.CreatedAt
	return nil

}

func (r *GormRepo) ListSyncLogs(ctx context.Context, target string, limit, offset int) ([]domain.IntegrationSyncLog, int, error) {

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

func (r *GormRepo) AddAuditLog(ctx context.Context, log domain.AdminAuditLog) error {

	return r.gdb.WithContext(ctx).Create(&adminAuditLogRow{
		AdminID:    log.AdminID,
		Action:     log.Action,
		TargetType: log.TargetType,
		TargetID:   log.TargetID,
		DetailJSON: log.DetailJSON,
	}).Error

}

func (r *GormRepo) ListAuditLogs(ctx context.Context, limit, offset int) ([]domain.AdminAuditLog, int, error) {

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

func (r *GormRepo) ListPermissionGroups(ctx context.Context) ([]domain.PermissionGroup, error) {

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

func (r *GormRepo) GetPermissionGroup(ctx context.Context, id int64) (domain.PermissionGroup, error) {

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

func (r *GormRepo) CreatePermissionGroup(ctx context.Context, group *domain.PermissionGroup) error {

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

func (r *GormRepo) UpdatePermissionGroup(ctx context.Context, group domain.PermissionGroup) error {

	return r.gdb.WithContext(ctx).Model(&permissionGroupRow{}).Where("id = ?", group.ID).Updates(map[string]any{
		"name":             group.Name,
		"description":      group.Description,
		"permissions_json": group.PermissionsJSON,
		"updated_at":       time.Now(),
	}).Error

}

func (r *GormRepo) DeletePermissionGroup(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&permissionGroupRow{}, id).Error

}

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

func (r *GormRepo) ListPermissions(ctx context.Context) ([]domain.Permission, error) {

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

func (r *GormRepo) GetPermissionByCode(ctx context.Context, code string) (domain.Permission, error) {

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

func (r *GormRepo) UpsertPermission(ctx context.Context, perm *domain.Permission) error {
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

	return r.gdb.WithContext(ctx).Model(&permissionModel{}).Where("code = ?", code).Updates(map[string]any{
		"name":       name,
		"updated_at": time.Now(),
	}).Error

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
	q := r.gdb.WithContext(ctx).Model(&cmsCategoryRow{})
	if lang != "" {
		q = q.Where("lang = ?", lang)
	}
	if !includeHidden {
		q = q.Where("visible = 1")
	}
	var rows []cmsCategoryRow
	if err := q.Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CMSCategory, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.CMSCategory{
			ID:        row.ID,
			Key:       row.Key,
			Name:      row.Name,
			Lang:      row.Lang,
			SortOrder: row.SortOrder,
			Visible:   row.Visible == 1,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) GetCMSCategory(ctx context.Context, id int64) (domain.CMSCategory, error) {
	var row cmsCategoryRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.CMSCategory{}, r.ensure(err)
	}
	return domain.CMSCategory{
		ID:        row.ID,
		Key:       row.Key,
		Name:      row.Name,
		Lang:      row.Lang,
		SortOrder: row.SortOrder,
		Visible:   row.Visible == 1,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *GormRepo) GetCMSCategoryByKey(ctx context.Context, key, lang string) (domain.CMSCategory, error) {
	var row cmsCategoryRow
	if err := r.gdb.WithContext(ctx).Where("`key` = ? AND lang = ?", key, lang).First(&row).Error; err != nil {
		return domain.CMSCategory{}, r.ensure(err)
	}
	return domain.CMSCategory{
		ID:        row.ID,
		Key:       row.Key,
		Name:      row.Name,
		Lang:      row.Lang,
		SortOrder: row.SortOrder,
		Visible:   row.Visible == 1,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *GormRepo) CreateCMSCategory(ctx context.Context, category *domain.CMSCategory) error {
	row := cmsCategoryRow{
		Key:       category.Key,
		Name:      category.Name,
		Lang:      category.Lang,
		SortOrder: category.SortOrder,
		Visible:   boolToInt(category.Visible),
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	category.ID = row.ID
	category.CreatedAt = row.CreatedAt
	category.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormRepo) UpdateCMSCategory(ctx context.Context, category domain.CMSCategory) error {
	return r.gdb.WithContext(ctx).Model(&cmsCategoryRow{}).Where("id = ?", category.ID).Updates(map[string]any{
		"key":        category.Key,
		"name":       category.Name,
		"lang":       category.Lang,
		"sort_order": category.SortOrder,
		"visible":    boolToInt(category.Visible),
		"updated_at": time.Now(),
	}).Error
}

func (r *GormRepo) DeleteCMSCategory(ctx context.Context, id int64) error {
	return r.gdb.WithContext(ctx).Delete(&cmsCategoryRow{}, id).Error
}

func (r *GormRepo) ListCMSPosts(ctx context.Context, filter usecase.CMSPostFilter) ([]domain.CMSPost, int, error) {
	q := r.gdb.WithContext(ctx).Model(&cmsPostRow{})
	if filter.CategoryID != nil {
		q = q.Where("cms_posts.category_id = ?", *filter.CategoryID)
	}
	if filter.CategoryKey != "" {
		q = q.Joins("JOIN cms_categories c ON c.id = cms_posts.category_id").Where("c.key = ?", filter.CategoryKey)
	}
	if filter.Status != "" {
		q = q.Where("cms_posts.status = ?", filter.Status)
	}
	if filter.PublishedOnly {
		q = q.Where("cms_posts.status = ?", "published")
	}
	if filter.Lang != "" {
		q = q.Where("cms_posts.lang = ?", filter.Lang)
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
	var rows []cmsPostRow
	if err := q.Order("cms_posts.pinned DESC, cms_posts.sort_order ASC, cms_posts.id DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.CMSPost, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.CMSPost{
			ID:          row.ID,
			CategoryID:  row.CategoryID,
			Title:       row.Title,
			Slug:        row.Slug,
			Summary:     row.Summary,
			ContentHTML: row.ContentHTML,
			CoverURL:    row.CoverURL,
			Lang:        row.Lang,
			Status:      row.Status,
			Pinned:      row.Pinned == 1,
			SortOrder:   row.SortOrder,
			PublishedAt: row.PublishedAt,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) GetCMSPost(ctx context.Context, id int64) (domain.CMSPost, error) {
	var row cmsPostRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.CMSPost{}, r.ensure(err)
	}
	return domain.CMSPost{
		ID:          row.ID,
		CategoryID:  row.CategoryID,
		Title:       row.Title,
		Slug:        row.Slug,
		Summary:     row.Summary,
		ContentHTML: row.ContentHTML,
		CoverURL:    row.CoverURL,
		Lang:        row.Lang,
		Status:      row.Status,
		Pinned:      row.Pinned == 1,
		SortOrder:   row.SortOrder,
		PublishedAt: row.PublishedAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *GormRepo) GetCMSPostBySlug(ctx context.Context, slug string) (domain.CMSPost, error) {
	var row cmsPostRow
	if err := r.gdb.WithContext(ctx).Where("slug = ?", slug).First(&row).Error; err != nil {
		return domain.CMSPost{}, r.ensure(err)
	}
	return domain.CMSPost{
		ID:          row.ID,
		CategoryID:  row.CategoryID,
		Title:       row.Title,
		Slug:        row.Slug,
		Summary:     row.Summary,
		ContentHTML: row.ContentHTML,
		CoverURL:    row.CoverURL,
		Lang:        row.Lang,
		Status:      row.Status,
		Pinned:      row.Pinned == 1,
		SortOrder:   row.SortOrder,
		PublishedAt: row.PublishedAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *GormRepo) CreateCMSPost(ctx context.Context, post *domain.CMSPost) error {
	var publishedAt *time.Time
	if post.PublishedAt != nil {
		utc := post.PublishedAt.UTC()
		publishedAt = &utc
	}
	row := cmsPostRow{
		CategoryID:  post.CategoryID,
		Title:       post.Title,
		Slug:        post.Slug,
		Summary:     post.Summary,
		ContentHTML: post.ContentHTML,
		CoverURL:    post.CoverURL,
		Lang:        post.Lang,
		Status:      post.Status,
		Pinned:      boolToInt(post.Pinned),
		SortOrder:   post.SortOrder,
		PublishedAt: publishedAt,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	post.ID = row.ID
	post.CreatedAt = row.CreatedAt
	post.UpdatedAt = row.UpdatedAt
	post.PublishedAt = row.PublishedAt
	return nil
}

func (r *GormRepo) UpdateCMSPost(ctx context.Context, post domain.CMSPost) error {
	var publishedAt *time.Time
	if post.PublishedAt != nil {
		utc := post.PublishedAt.UTC()
		publishedAt = &utc
	}
	return r.gdb.WithContext(ctx).Model(&cmsPostRow{}).Where("id = ?", post.ID).Updates(map[string]any{
		"category_id":  post.CategoryID,
		"title":        post.Title,
		"slug":         post.Slug,
		"summary":      post.Summary,
		"content_html": post.ContentHTML,
		"cover_url":    post.CoverURL,
		"lang":         post.Lang,
		"status":       post.Status,
		"pinned":       boolToInt(post.Pinned),
		"sort_order":   post.SortOrder,
		"published_at": publishedAt,
		"updated_at":   time.Now(),
	}).Error
}

func (r *GormRepo) DeleteCMSPost(ctx context.Context, id int64) error {
	return r.gdb.WithContext(ctx).Delete(&cmsPostRow{}, id).Error
}

func (r *GormRepo) ListCMSBlocks(ctx context.Context, page, lang string, includeHidden bool) ([]domain.CMSBlock, error) {
	q := r.gdb.WithContext(ctx).Model(&cmsBlockRow{})
	if page != "" {
		q = q.Where("page = ?", page)
	}
	if lang != "" {
		q = q.Where("lang = ?", lang)
	}
	if !includeHidden {
		q = q.Where("visible = 1")
	}
	var rows []cmsBlockRow
	if err := q.Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CMSBlock, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.CMSBlock{
			ID:          row.ID,
			Page:        row.Page,
			Type:        row.Type,
			Title:       row.Title,
			Subtitle:    row.Subtitle,
			ContentJSON: row.ContentJSON,
			CustomHTML:  row.CustomHTML,
			Lang:        row.Lang,
			Visible:     row.Visible == 1,
			SortOrder:   row.SortOrder,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) GetCMSBlock(ctx context.Context, id int64) (domain.CMSBlock, error) {
	var row cmsBlockRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.CMSBlock{}, r.ensure(err)
	}
	return domain.CMSBlock{
		ID:          row.ID,
		Page:        row.Page,
		Type:        row.Type,
		Title:       row.Title,
		Subtitle:    row.Subtitle,
		ContentJSON: row.ContentJSON,
		CustomHTML:  row.CustomHTML,
		Lang:        row.Lang,
		Visible:     row.Visible == 1,
		SortOrder:   row.SortOrder,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *GormRepo) CreateCMSBlock(ctx context.Context, block *domain.CMSBlock) error {
	row := cmsBlockRow{
		Page:        block.Page,
		Type:        block.Type,
		Title:       block.Title,
		Subtitle:    block.Subtitle,
		ContentJSON: block.ContentJSON,
		CustomHTML:  block.CustomHTML,
		Lang:        block.Lang,
		Visible:     boolToInt(block.Visible),
		SortOrder:   block.SortOrder,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	block.ID = row.ID
	block.CreatedAt = row.CreatedAt
	block.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormRepo) UpdateCMSBlock(ctx context.Context, block domain.CMSBlock) error {
	return r.gdb.WithContext(ctx).Model(&cmsBlockRow{}).Where("id = ?", block.ID).Updates(map[string]any{
		"page":         block.Page,
		"type":         block.Type,
		"title":        block.Title,
		"subtitle":     block.Subtitle,
		"content_json": block.ContentJSON,
		"custom_html":  block.CustomHTML,
		"lang":         block.Lang,
		"visible":      boolToInt(block.Visible),
		"sort_order":   block.SortOrder,
		"updated_at":   time.Now(),
	}).Error
}

func (r *GormRepo) DeleteCMSBlock(ctx context.Context, id int64) error {
	return r.gdb.WithContext(ctx).Delete(&cmsBlockRow{}, id).Error
}

func (r *GormRepo) CreateUpload(ctx context.Context, upload *domain.Upload) error {
	row := uploadRow{
		Name:       upload.Name,
		Path:       upload.Path,
		URL:        upload.URL,
		Mime:       upload.Mime,
		Size:       upload.Size,
		UploaderID: upload.UploaderID,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	upload.ID = row.ID
	upload.CreatedAt = row.CreatedAt
	return nil
}

func (r *GormRepo) ListUploads(ctx context.Context, limit, offset int) ([]domain.Upload, int, error) {
	if limit <= 0 {
		limit = 20
	}
	q := r.gdb.WithContext(ctx).Model(&uploadRow{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []uploadRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Upload, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Upload{
			ID:         row.ID,
			Name:       row.Name,
			Path:       row.Path,
			URL:        row.URL,
			Mime:       row.Mime,
			Size:       row.Size,
			UploaderID: row.UploaderID,
			CreatedAt:  row.CreatedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) ListTickets(ctx context.Context, filter usecase.TicketFilter) ([]domain.Ticket, int, error) {
	q := r.gdb.WithContext(ctx).Model(&ticketRow{})
	if filter.UserID != nil {
		q = q.Where("user_id = ?", *filter.UserID)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		q = q.Where("subject LIKE ?", "%"+filter.Keyword+"%")
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
	var rows []ticketRow
	if err := q.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	resourceCount := map[int64]int{}
	if len(rows) > 0 {
		ids := make([]int64, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
		type resourceAgg struct {
			TicketID int64 `gorm:"column:ticket_id"`
			Total    int   `gorm:"column:total"`
		}
		var aggs []resourceAgg
		if err := r.gdb.WithContext(ctx).
			Model(&ticketResourceRow{}).
			Select("ticket_id, COUNT(1) AS total").
			Where("ticket_id IN ?", ids).
			Group("ticket_id").
			Find(&aggs).Error; err != nil {
			return nil, 0, err
		}
		for _, agg := range aggs {
			resourceCount[agg.TicketID] = agg.Total
		}
	}
	out := make([]domain.Ticket, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Ticket{
			ID:            row.ID,
			UserID:        row.UserID,
			Subject:       row.Subject,
			Status:        row.Status,
			ResourceCount: resourceCount[row.ID],
			LastReplyAt:   row.LastReplyAt,
			LastReplyBy:   row.LastReplyBy,
			LastReplyRole: row.LastReplyRole,
			ClosedAt:      row.ClosedAt,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) GetTicket(ctx context.Context, id int64) (domain.Ticket, error) {
	var row ticketRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.Ticket{}, r.ensure(err)
	}
	var resourceCount int64
	if err := r.gdb.WithContext(ctx).Model(&ticketResourceRow{}).Where("ticket_id = ?", id).Count(&resourceCount).Error; err != nil {
		return domain.Ticket{}, err
	}
	return domain.Ticket{
		ID:            row.ID,
		UserID:        row.UserID,
		Subject:       row.Subject,
		Status:        row.Status,
		ResourceCount: int(resourceCount),
		LastReplyAt:   row.LastReplyAt,
		LastReplyBy:   row.LastReplyBy,
		LastReplyRole: row.LastReplyRole,
		ClosedAt:      row.ClosedAt,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

func (r *GormRepo) CreateTicketWithDetails(ctx context.Context, ticket *domain.Ticket, message *domain.TicketMessage, resources []domain.TicketResource) error {
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tRow := ticketRow{
			UserID:        ticket.UserID,
			Subject:       ticket.Subject,
			Status:        ticket.Status,
			LastReplyAt:   ticket.LastReplyAt,
			LastReplyBy:   ticket.LastReplyBy,
			LastReplyRole: ticket.LastReplyRole,
			ClosedAt:      ticket.ClosedAt,
		}
		if err := tx.Create(&tRow).Error; err != nil {
			return err
		}
		mRow := ticketMessageRow{
			TicketID:   tRow.ID,
			SenderID:   message.SenderID,
			SenderRole: message.SenderRole,
			SenderName: message.SenderName,
			SenderQQ:   message.SenderQQ,
			Content:    message.Content,
		}
		if err := tx.Create(&mRow).Error; err != nil {
			return err
		}
		if len(resources) > 0 {
			rRows := make([]ticketResourceRow, 0, len(resources))
			for _, resource := range resources {
				rRows = append(rRows, ticketResourceRow{
					TicketID:     tRow.ID,
					ResourceType: resource.ResourceType,
					ResourceID:   resource.ResourceID,
					ResourceName: resource.ResourceName,
				})
			}
			if err := tx.Create(&rRows).Error; err != nil {
				return err
			}
		}
		ticket.ID = tRow.ID
		ticket.CreatedAt = tRow.CreatedAt
		ticket.UpdatedAt = tRow.UpdatedAt
		message.ID = mRow.ID
		message.TicketID = tRow.ID
		message.CreatedAt = mRow.CreatedAt
		return nil
	})
}

func (r *GormRepo) AddTicketMessage(ctx context.Context, message *domain.TicketMessage) error {
	row := ticketMessageRow{
		TicketID:   message.TicketID,
		SenderID:   message.SenderID,
		SenderRole: message.SenderRole,
		SenderName: message.SenderName,
		SenderQQ:   message.SenderQQ,
		Content:    message.Content,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	now := time.Now()
	message.ID = row.ID
	message.CreatedAt = row.CreatedAt
	return r.gdb.WithContext(ctx).Model(&ticketRow{}).Where("id = ?", message.TicketID).Updates(map[string]any{
		"last_reply_at":   now,
		"last_reply_by":   message.SenderID,
		"last_reply_role": message.SenderRole,
		"updated_at":      now,
	}).Error
}

func (r *GormRepo) ListTicketMessages(ctx context.Context, ticketID int64) ([]domain.TicketMessage, error) {
	var rows []ticketMessageRow
	if err := r.gdb.WithContext(ctx).Where("ticket_id = ?", ticketID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.TicketMessage, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.TicketMessage{
			ID:         row.ID,
			TicketID:   row.TicketID,
			SenderID:   row.SenderID,
			SenderRole: row.SenderRole,
			SenderName: row.SenderName,
			SenderQQ:   row.SenderQQ,
			Content:    row.Content,
			CreatedAt:  row.CreatedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) ListTicketResources(ctx context.Context, ticketID int64) ([]domain.TicketResource, error) {
	var rows []ticketResourceRow
	if err := r.gdb.WithContext(ctx).Where("ticket_id = ?", ticketID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.TicketResource, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.TicketResource{
			ID:           row.ID,
			TicketID:     row.TicketID,
			ResourceType: row.ResourceType,
			ResourceID:   row.ResourceID,
			ResourceName: row.ResourceName,
			CreatedAt:    row.CreatedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) UpdateTicket(ctx context.Context, ticket domain.Ticket) error {
	return r.gdb.WithContext(ctx).Model(&ticketRow{}).Where("id = ?", ticket.ID).Updates(map[string]any{
		"subject":    ticket.Subject,
		"status":     ticket.Status,
		"closed_at":  ticket.ClosedAt,
		"updated_at": time.Now(),
	}).Error
}

func (r *GormRepo) DeleteTicket(ctx context.Context, id int64) error {
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ticket_id = ?", id).Delete(&ticketMessageRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("ticket_id = ?", id).Delete(&ticketResourceRow{}).Error; err != nil {
			return err
		}
		return tx.Delete(&ticketRow{}, id).Error
	})
}

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

func (r *GormRepo) ListNotifications(ctx context.Context, filter usecase.NotificationFilter) ([]domain.Notification, int, error) {
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

func (r *GormRepo) GetWallet(ctx context.Context, userID int64) (domain.Wallet, error) {
	var row walletRow
	if err := r.gdb.WithContext(ctx).Where("user_id = ?", userID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w := domain.Wallet{UserID: userID, Balance: 0}
			if err := r.UpsertWallet(ctx, &w); err != nil {
				return domain.Wallet{}, err
			}
			return r.GetWallet(ctx, userID)
		}
		return domain.Wallet{}, err
	}
	return domain.Wallet{
		ID:        row.ID,
		UserID:    row.UserID,
		Balance:   row.Balance,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *GormRepo) UpsertWallet(ctx context.Context, wallet *domain.Wallet) error {
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
	row := walletTransactionRow{
		UserID:  txItem.UserID,
		Amount:  txItem.Amount,
		Type:    txItem.Type,
		RefType: txItem.RefType,
		RefID:   txItem.RefID,
		Note:    txItem.Note,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	txItem.ID = row.ID
	txItem.CreatedAt = row.CreatedAt
	return nil
}

func (r *GormRepo) ListWalletTransactions(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletTransaction, int, error) {
	if limit <= 0 {
		limit = 20
	}
	q := r.gdb.WithContext(ctx).Model(&walletTransactionRow{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []walletTransactionRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.WalletTransaction, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.WalletTransaction{
			ID:        row.ID,
			UserID:    row.UserID,
			Amount:    row.Amount,
			Type:      row.Type,
			RefType:   row.RefType,
			RefID:     row.RefID,
			Note:      row.Note,
			CreatedAt: row.CreatedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) AdjustWalletBalance(ctx context.Context, userID int64, amount int64, txType, refType string, refID int64, note string) (wallet domain.Wallet, err error) {
	err = r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var w walletRow
		lock := clause.Locking{Strength: "UPDATE"}
		if e := tx.Clauses(lock).Where("user_id = ?", userID).First(&w).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				w = walletRow{UserID: userID, Balance: 0, UpdatedAt: time.Now()}
				if e = tx.Create(&w).Error; e != nil {
					return e
				}
			} else {
				return e
			}
		}
		newBalance := w.Balance + amount
		if newBalance < 0 {
			return usecase.ErrInsufficientBalance
		}
		now := time.Now()
		if e := tx.Model(&walletRow{}).Where("user_id = ?", userID).Updates(map[string]any{
			"balance":    newBalance,
			"updated_at": now,
		}).Error; e != nil {
			return e
		}
		txRow := walletTransactionRow{
			UserID:  userID,
			Amount:  amount,
			Type:    txType,
			RefType: refType,
			RefID:   refID,
			Note:    note,
		}
		if e := tx.Create(&txRow).Error; e != nil {
			return e
		}
		wallet = domain.Wallet{
			ID:        w.ID,
			UserID:    userID,
			Balance:   newBalance,
			UpdatedAt: now,
		}
		return nil
	})
	if err != nil {
		return domain.Wallet{}, err
	}
	return wallet, nil
}

func (r *GormRepo) HasWalletTransaction(ctx context.Context, userID int64, refType string, refID int64) (bool, error) {
	var total int64
	if err := r.gdb.WithContext(ctx).Model(&walletTransactionRow{}).
		Where("user_id = ? AND ref_type = ? AND ref_id = ?", userID, refType, refID).
		Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

func (r *GormRepo) CreateWalletOrder(ctx context.Context, order *domain.WalletOrder) error {
	row := walletOrderRow{
		UserID:   order.UserID,
		Type:     string(order.Type),
		Amount:   order.Amount,
		Currency: order.Currency,
		Status:   string(order.Status),
		Note:     order.Note,
		MetaJSON: order.MetaJSON,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	order.ID = row.ID
	order.CreatedAt = row.CreatedAt
	order.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormRepo) GetWalletOrder(ctx context.Context, id int64) (domain.WalletOrder, error) {
	var row walletOrderRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.WalletOrder{}, r.ensure(err)
	}
	return domain.WalletOrder{
		ID:           row.ID,
		UserID:       row.UserID,
		Type:         domain.WalletOrderType(row.Type),
		Amount:       row.Amount,
		Currency:     row.Currency,
		Status:       domain.WalletOrderStatus(row.Status),
		Note:         row.Note,
		MetaJSON:     row.MetaJSON,
		ReviewedBy:   row.ReviewedBy,
		ReviewReason: row.ReviewReason,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

func (r *GormRepo) ListWalletOrders(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletOrder, int, error) {
	if limit <= 0 {
		limit = 20
	}
	q := r.gdb.WithContext(ctx).Model(&walletOrderRow{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []walletOrderRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.WalletOrder, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.WalletOrder{
			ID:           row.ID,
			UserID:       row.UserID,
			Type:         domain.WalletOrderType(row.Type),
			Amount:       row.Amount,
			Currency:     row.Currency,
			Status:       domain.WalletOrderStatus(row.Status),
			Note:         row.Note,
			MetaJSON:     row.MetaJSON,
			ReviewedBy:   row.ReviewedBy,
			ReviewReason: row.ReviewReason,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) ListAllWalletOrders(ctx context.Context, status string, limit, offset int) ([]domain.WalletOrder, int, error) {
	if limit <= 0 {
		limit = 20
	}
	q := r.gdb.WithContext(ctx).Model(&walletOrderRow{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []walletOrderRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.WalletOrder, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.WalletOrder{
			ID:           row.ID,
			UserID:       row.UserID,
			Type:         domain.WalletOrderType(row.Type),
			Amount:       row.Amount,
			Currency:     row.Currency,
			Status:       domain.WalletOrderStatus(row.Status),
			Note:         row.Note,
			MetaJSON:     row.MetaJSON,
			ReviewedBy:   row.ReviewedBy,
			ReviewReason: row.ReviewReason,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) UpdateWalletOrderStatus(ctx context.Context, id int64, status domain.WalletOrderStatus, reviewedBy *int64, reason string) error {
	return r.gdb.WithContext(ctx).Model(&walletOrderRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":        string(status),
		"reviewed_by":   reviewedBy,
		"review_reason": reason,
		"updated_at":    time.Now(),
	}).Error
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
	_ usecase.ProbeNodeRepository          = (*GormRepo)(nil)
	_ usecase.ProbeEnrollTokenRepository   = (*GormRepo)(nil)
	_ usecase.ProbeStatusEventRepository   = (*GormRepo)(nil)
	_ usecase.ProbeLogSessionRepository    = (*GormRepo)(nil)
)
