package repo

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreateProbeNode(ctx context.Context, node *domain.ProbeNode) error {
	row := probeNodeRow{
		Name:             node.Name,
		AgentID:          node.AgentID,
		SecretHash:       node.SecretHash,
		Status:           string(node.Status),
		OSType:           node.OSType,
		TagsJSON:         node.TagsJSON,
		LastHeartbeatAt:  node.LastHeartbeatAt,
		LastSnapshotAt:   node.LastSnapshotAt,
		LastSnapshotJSON: node.LastSnapshotJSON,
	}
	if row.Status == "" {
		row.Status = string(domain.ProbeStatusOffline)
	}
	if row.TagsJSON == "" {
		row.TagsJSON = "[]"
	}
	if row.LastSnapshotJSON == "" {
		row.LastSnapshotJSON = "{}"
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	node.ID = row.ID
	node.CreatedAt = row.CreatedAt
	node.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormRepo) GetProbeNode(ctx context.Context, id int64) (domain.ProbeNode, error) {
	var row probeNodeRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.ProbeNode{}, r.ensure(err)
	}
	return fromProbeNodeRow(row), nil
}

func (r *GormRepo) GetProbeNodeByAgentID(ctx context.Context, agentID string) (domain.ProbeNode, error) {
	var row probeNodeRow
	if err := r.gdb.WithContext(ctx).Where("agent_id = ?", agentID).First(&row).Error; err != nil {
		return domain.ProbeNode{}, r.ensure(err)
	}
	return fromProbeNodeRow(row), nil
}

func (r *GormRepo) ListProbeNodes(ctx context.Context, filter appshared.ProbeNodeFilter, limit, offset int) ([]domain.ProbeNode, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 500 {
		limit = 500
	}
	q := r.gdb.WithContext(ctx).Model(&probeNodeRow{})
	if strings.TrimSpace(filter.Status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(filter.Status))
	}
	if kw := strings.TrimSpace(filter.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR agent_id LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []probeNodeRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.ProbeNode, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromProbeNodeRow(row))
	}
	return out, int(total), nil
}

func (r *GormRepo) UpdateProbeNode(ctx context.Context, node domain.ProbeNode) error {
	return r.gdb.WithContext(ctx).Model(&probeNodeRow{}).Where("id = ?", node.ID).Updates(map[string]any{
		"name":        strings.TrimSpace(node.Name),
		"agent_id":    strings.TrimSpace(node.AgentID),
		"secret_hash": strings.TrimSpace(node.SecretHash),
		"os_type":     strings.TrimSpace(node.OSType),
		"tags_json":   node.TagsJSON,
		"updated_at":  time.Now(),
	}).Error
}

func (r *GormRepo) DeleteProbeNode(ctx context.Context, id int64) error {
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("probe_id = ?", id).Delete(&probeLogSessionRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("probe_id = ?", id).Delete(&probeStatusEventRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("probe_id = ?", id).Delete(&probeEnrollTokenRow{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&probeNodeRow{}).Error
	})
}

func (r *GormRepo) UpdateProbeNodeStatus(ctx context.Context, id int64, status domain.ProbeStatus, reason string, at time.Time) error {
	_ = reason
	return r.gdb.WithContext(ctx).Model(&probeNodeRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":            string(status),
		"updated_at":        time.Now(),
		"last_heartbeat_at": clause.Expr{SQL: "COALESCE(last_heartbeat_at, ?)", Vars: []any{at}},
	}).Error
}

func (r *GormRepo) UpdateProbeNodeHeartbeat(ctx context.Context, id int64, at time.Time) error {
	return r.gdb.WithContext(ctx).Model(&probeNodeRow{}).Where("id = ?", id).Updates(map[string]any{
		"last_heartbeat_at": at,
		"updated_at":        time.Now(),
	}).Error
}

func (r *GormRepo) UpdateProbeNodeSnapshot(ctx context.Context, id int64, at time.Time, snapshotJSON string, osType string) error {
	updates := map[string]any{
		"last_snapshot_at":   at,
		"last_snapshot_json": snapshotJSON,
		"updated_at":         time.Now(),
	}
	if strings.TrimSpace(osType) != "" {
		updates["os_type"] = strings.TrimSpace(osType)
	}
	return r.gdb.WithContext(ctx).Model(&probeNodeRow{}).Where("id = ?", id).Updates(updates).Error
}

func (r *GormRepo) CreateProbeEnrollToken(ctx context.Context, token *domain.ProbeEnrollToken) error {
	row := probeEnrollTokenRow{
		ProbeID:   token.ProbeID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
		UsedAt:    token.UsedAt,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	token.ID = row.ID
	token.CreatedAt = row.CreatedAt
	return nil
}

func (r *GormRepo) GetValidProbeEnrollTokenByHash(ctx context.Context, tokenHash string, now time.Time) (domain.ProbeEnrollToken, error) {
	var row probeEnrollTokenRow
	if err := r.gdb.WithContext(ctx).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, now).
		First(&row).Error; err != nil {
		return domain.ProbeEnrollToken{}, r.ensure(err)
	}
	return fromProbeEnrollTokenRow(row), nil
}

func (r *GormRepo) MarkProbeEnrollTokenUsed(ctx context.Context, id int64, usedAt time.Time) error {
	return r.gdb.WithContext(ctx).Model(&probeEnrollTokenRow{}).Where("id = ?", id).Update("used_at", usedAt).Error
}

func (r *GormRepo) DeleteProbeEnrollTokensByProbe(ctx context.Context, probeID int64) error {
	return r.gdb.WithContext(ctx).Where("probe_id = ? AND used_at IS NULL", probeID).Delete(&probeEnrollTokenRow{}).Error
}

func (r *GormRepo) CreateProbeStatusEvent(ctx context.Context, ev *domain.ProbeStatusEvent) error {
	row := probeStatusEventRow{
		ProbeID: ev.ProbeID,
		Status:  string(ev.Status),
		At:      ev.At,
		Reason:  ev.Reason,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	ev.ID = row.ID
	ev.CreatedAt = row.CreatedAt
	return nil
}

func (r *GormRepo) ListProbeStatusEvents(ctx context.Context, probeID int64, from, to time.Time) ([]domain.ProbeStatusEvent, error) {
	var rows []probeStatusEventRow
	if err := r.gdb.WithContext(ctx).
		Where("probe_id = ? AND at >= ? AND at <= ?", probeID, from, to).
		Order("at ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ProbeStatusEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromProbeStatusEventRow(row))
	}
	return out, nil
}

func (r *GormRepo) GetLatestProbeStatusEventBefore(ctx context.Context, probeID int64, before time.Time) (domain.ProbeStatusEvent, error) {
	var row probeStatusEventRow
	if err := r.gdb.WithContext(ctx).
		Where("probe_id = ? AND at < ?", probeID, before).
		Order("at DESC, id DESC").
		First(&row).Error; err != nil {
		return domain.ProbeStatusEvent{}, r.ensure(err)
	}
	return fromProbeStatusEventRow(row), nil
}

func (r *GormRepo) DeleteProbeStatusEventsBefore(ctx context.Context, before time.Time) error {
	return r.gdb.WithContext(ctx).Where("at < ?", before).Delete(&probeStatusEventRow{}).Error
}

func (r *GormRepo) CreateProbeLogSession(ctx context.Context, session *domain.ProbeLogSession) error {
	row := probeLogSessionRow{
		ProbeID:    session.ProbeID,
		OperatorID: session.OperatorID,
		Source:     session.Source,
		Status:     session.Status,
		StartedAt:  session.StartedAt,
		EndedAt:    session.EndedAt,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	session.ID = row.ID
	session.CreatedAt = row.CreatedAt
	return nil
}

func (r *GormRepo) GetProbeLogSession(ctx context.Context, id int64) (domain.ProbeLogSession, error) {
	var row probeLogSessionRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.ProbeLogSession{}, r.ensure(err)
	}
	return fromProbeLogSessionRow(row), nil
}

func (r *GormRepo) UpdateProbeLogSession(ctx context.Context, session domain.ProbeLogSession) error {
	return r.gdb.WithContext(ctx).Model(&probeLogSessionRow{}).Where("id = ?", session.ID).Updates(map[string]any{
		"status":   session.Status,
		"ended_at": session.EndedAt,
	}).Error
}

func (r *GormRepo) PurgeProbeLogSessions(ctx context.Context, before time.Time) error {
	return r.gdb.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&probeLogSessionRow{}).Error
}

func fromProbeNodeRow(row probeNodeRow) domain.ProbeNode {
	return domain.ProbeNode{
		ID:               row.ID,
		Name:             row.Name,
		AgentID:          row.AgentID,
		SecretHash:       row.SecretHash,
		Status:           domain.ProbeStatus(row.Status),
		OSType:           row.OSType,
		TagsJSON:         row.TagsJSON,
		LastHeartbeatAt:  row.LastHeartbeatAt,
		LastSnapshotAt:   row.LastSnapshotAt,
		LastSnapshotJSON: row.LastSnapshotJSON,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func fromProbeEnrollTokenRow(row probeEnrollTokenRow) domain.ProbeEnrollToken {
	return domain.ProbeEnrollToken{
		ID:        row.ID,
		ProbeID:   row.ProbeID,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt,
		UsedAt:    row.UsedAt,
		CreatedAt: row.CreatedAt,
	}
}

func fromProbeStatusEventRow(row probeStatusEventRow) domain.ProbeStatusEvent {
	return domain.ProbeStatusEvent{
		ID:        row.ID,
		ProbeID:   row.ProbeID,
		Status:    domain.ProbeStatus(row.Status),
		At:        row.At,
		Reason:    row.Reason,
		CreatedAt: row.CreatedAt,
	}
}

func fromProbeLogSessionRow(row probeLogSessionRow) domain.ProbeLogSession {
	return domain.ProbeLogSession{
		ID:         row.ID,
		ProbeID:    row.ProbeID,
		OperatorID: row.OperatorID,
		Source:     row.Source,
		Status:     row.Status,
		StartedAt:  row.StartedAt,
		EndedAt:    row.EndedAt,
		CreatedAt:  row.CreatedAt,
	}
}
