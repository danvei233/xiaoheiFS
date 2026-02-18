package repo

import (
	"context"
	"fmt"
	"gorm.io/gorm/clause"
	"strings"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

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
		return appshared.ErrInvalidInput
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
		return appshared.ErrInvalidInput
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
		return fmt.Errorf("email template name already exists")
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
