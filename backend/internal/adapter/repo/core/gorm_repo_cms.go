package repo

import (
	"context"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

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

func (r *GormRepo) ListCMSPosts(ctx context.Context, filter appshared.CMSPostFilter) ([]domain.CMSPost, int, error) {
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
