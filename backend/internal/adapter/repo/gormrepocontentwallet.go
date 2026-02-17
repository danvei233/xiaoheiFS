package repo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

func (r *GormRepo) ListTickets(ctx context.Context, filter appshared.TicketFilter) ([]domain.Ticket, int, error) {
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

func (r *GormRepo) ListNotifications(ctx context.Context, filter appshared.NotificationFilter) ([]domain.Notification, int, error) {
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
			return appshared.ErrInsufficientBalance
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
