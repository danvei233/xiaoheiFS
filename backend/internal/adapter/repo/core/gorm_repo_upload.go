package repo

import (
	"context"

	"xiaoheiplay/internal/domain"
)

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
