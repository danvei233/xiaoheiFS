package repo

import (
	"context"
	"time"

	"gorm.io/gorm"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

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
