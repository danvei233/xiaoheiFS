package repo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"xiaoheiplay/internal/domain"
)

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
