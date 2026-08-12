package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sort"
	"strconv"
	"strings"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

const (
	settingKeySMSTemplates        = "sms_templates_json"
	settingKeyRobotWebhooks       = "robot_webhooks"
	settingKeyPackageCapabilities = "package_capabilities_json"
	settingKeyGoodsCapabilities   = "goods_type_capabilities_json"
)

// normalizedListSettingKeys is derived from the single source of truth
// (app/shared JSONSettingKeys, P3-4) so the repo's list-key set and the HTTP
// handler's isLikelyJSONSettingKey cannot drift apart. getNormalizedSettingValue
// special-cases sms_templates_json / robot_webhooks / package_capabilities_json
// before this set is consulted, so the extra named keys are harmless on read.
var normalizedListSettingKeys = buildNormalizedListSettingKeys()

func buildNormalizedListSettingKeys() map[string]struct{} {
	keys := make(map[string]struct{}, len(appshared.JSONSettingKeys))
	for _, k := range appshared.JSONSettingKeys {
		keys[k] = struct{}{}
	}
	return keys
}

func (r *GormRepo) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	if raw, ok, err := r.getNormalizedSettingValue(ctx, key); err != nil {
		return domain.Setting{}, err
	} else if ok {
		return domain.Setting{Key: key, ValueJSON: raw, UpdatedAt: time.Now()}, nil
	}

	var m settingModel
	if err := r.gdb.WithContext(ctx).Where("`key` = ?", key).First(&m).Error; err != nil {
		return domain.Setting{}, r.ensure(err)
	}
	return domain.Setting{Key: m.Key, ValueJSON: m.ValueJSON, UpdatedAt: m.UpdatedAt}, nil

}

// ClaimSettingIfAbsent atomically inserts a setting row only when the key is not
// already present. It returns true when the caller performed the insert (the key
// was absent), false when the row already exists. The installer uses it to make
// the DB-side install-completion marker an atomic claim: two hosts sharing one
// database cannot both pass a check-then-insert, and a replayed install cannot
// reach the admin side effects once a claim exists.
func (r *GormRepo) ClaimSettingIfAbsent(ctx context.Context, key, valueJSON string) (bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return false, appshared.ErrInvalidInput
	}
	m := settingModel{Key: key, ValueJSON: valueJSON, UpdatedAt: time.Now()}
	err := r.gdb.WithContext(ctx).Create(&m).Error
	if err == nil {
		return true, nil
	}
	if isUniqueConstraintViolation(err) {
		return false, nil
	}
	return false, err
}

// DeleteSetting removes a raw scalar setting. The installer uses this only to
// roll back a completion-marker claim from the same failed attempt.
func (r *GormRepo) DeleteSetting(ctx context.Context, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return appshared.ErrInvalidInput
	}
	return r.gdb.WithContext(ctx).Where("`key` = ?", key).Delete(&settingModel{}).Error
}

// isUniqueConstraintViolation reports whether err is a unique-constraint /
// duplicate-key violation across the supported drivers (SQLite, MySQL, Postgres).
func isUniqueConstraintViolation(err error) bool {
	if err == nil {
		return false
	}
	var me *mysqlDriver.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1062
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "constraint failed")
}

func (r *GormRepo) UpsertSetting(ctx context.Context, setting domain.Setting) error {
	return r.UpsertSettings(ctx, []domain.Setting{setting})
}

// UpsertSettings writes a batch of settings atomically: either every key is
// persisted or none is. The admin settings batch endpoint uses it so a failure
// on the k-th key does not leave the first k-1 keys half-written.
func (r *GormRepo) UpsertSettings(ctx context.Context, settings []domain.Setting) error {
	if len(settings) == 0 {
		return nil
	}
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, setting := range settings {
			if handled, err := r.upsertNormalizedSetting(tx, setting); handled || err != nil {
				if err != nil {
					return err
				}
				continue
			}
			m := settingModel{Key: setting.Key, ValueJSON: setting.ValueJSON, UpdatedAt: time.Now()}
			if err := tx.
				Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "key"}},
					DoUpdates: clause.AssignmentColumns([]string{"value_json", "updated_at"}),
				}).
				Create(&m).Error; err != nil {
				return err
			}
		}
		return nil
	})
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
	out = r.overlayNormalizedSettings(ctx, out)
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil

}

func (r *GormRepo) overlayNormalizedSettings(ctx context.Context, base []domain.Setting) []domain.Setting {
	type keyState struct {
		value string
		ok    bool
	}
	normalized := map[string]keyState{}
	keys := []string{
		settingKeySMSTemplates,
		settingKeyRobotWebhooks,
		settingKeyPackageCapabilities,
		settingKeyGoodsCapabilities,
	}
	for k := range normalizedListSettingKeys {
		keys = append(keys, k)
	}
	for _, key := range keys {
		v, ok, err := r.getNormalizedSettingValue(ctx, key)
		if err != nil {
			continue
		}
		normalized[key] = keyState{value: v, ok: ok}
	}
	taskRows, err := r.listScheduledTaskConfigRows(ctx)
	if err == nil {
		for _, row := range taskRows {
			key := "task." + strings.TrimSpace(row.TaskKey)
			if key == "task." {
				continue
			}
			raw, err := marshalTaskSettingRow(row)
			if err != nil {
				continue
			}
			normalized[key] = keyState{value: raw, ok: true}
		}
	}

	result := make([]domain.Setting, 0, len(base)+len(normalized))
	seen := map[string]bool{}
	for _, item := range base {
		if n, ok := normalized[item.Key]; ok && n.ok {
			item.ValueJSON = n.value
			item.UpdatedAt = time.Now()
			seen[item.Key] = true
		}
		result = append(result, item)
	}
	for key, n := range normalized {
		if !n.ok || seen[key] {
			continue
		}
		result = append(result, domain.Setting{Key: key, ValueJSON: n.value, UpdatedAt: time.Now()})
	}
	return result
}

func (r *GormRepo) getNormalizedSettingValue(ctx context.Context, key string) (string, bool, error) {
	switch {
	case key == settingKeySMSTemplates:
		rows, err := r.listSMSTemplateRows(ctx)
		if err != nil {
			return "", false, err
		}
		if len(rows) == 0 {
			return "", false, nil
		}
		type smsTemplatePayload struct {
			ID        int64     `json:"id"`
			Name      string    `json:"name"`
			Content   string    `json:"content"`
			Enabled   bool      `json:"enabled"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
		}
		payload := make([]smsTemplatePayload, 0, len(rows))
		for _, row := range rows {
			payload = append(payload, smsTemplatePayload{
				ID:        row.ID,
				Name:      row.Name,
				Content:   row.Content,
				Enabled:   row.Enabled == 1,
				CreatedAt: row.CreatedAt,
				UpdatedAt: row.UpdatedAt,
			})
		}
		raw, _ := json.Marshal(payload)
		return string(raw), true, nil
	case key == settingKeyRobotWebhooks:
		rows, err := r.listRobotWebhookRows(ctx)
		if err != nil {
			return "", false, err
		}
		if len(rows) == 0 {
			return "", false, nil
		}
		type webhookPayload struct {
			Name    string   `json:"name"`
			URL     string   `json:"url"`
			Secret  string   `json:"secret"`
			Enabled bool     `json:"enabled"`
			Events  []string `json:"events"`
		}
		payload := make([]webhookPayload, 0, len(rows))
		for _, row := range rows {
			var events []string
			_ = json.Unmarshal([]byte(row.EventsJSON), &events)
			payload = append(payload, webhookPayload{
				Name:    row.Name,
				URL:     row.URL,
				Secret:  row.Secret,
				Enabled: row.Enabled == 1,
				Events:  events,
			})
		}
		raw, _ := json.Marshal(payload)
		return string(raw), true, nil
	case key == settingKeyPackageCapabilities:
		rows, err := r.listPackageCapabilityRows(ctx)
		if err != nil {
			return "", false, err
		}
		if len(rows) == 0 {
			return "", false, nil
		}
		type capabilityPolicy struct {
			ResizeEnabled *bool `json:"resize_enabled,omitempty"`
			RefundEnabled *bool `json:"refund_enabled,omitempty"`
		}
		payload := map[string]capabilityPolicy{}
		for _, row := range rows {
			item := capabilityPolicy{}
			if row.ResizeEnabled != nil {
				v := *row.ResizeEnabled == 1
				item.ResizeEnabled = &v
			}
			if row.RefundEnabled != nil {
				v := *row.RefundEnabled == 1
				item.RefundEnabled = &v
			}
			payload[strconv.FormatInt(row.PackageID, 10)] = item
		}
		raw, _ := json.Marshal(payload)
		return string(raw), true, nil
	case key == settingKeyGoodsCapabilities:
		rows, err := r.listGoodsTypeCapabilityRows(ctx)
		if err != nil {
			return "", false, err
		}
		if len(rows) == 0 {
			return "", false, nil
		}
		payload := map[string]domain.CapabilityPolicy{}
		for _, row := range rows {
			payload[strconv.FormatInt(row.GoodsTypeID, 10)] = capabilityPolicyFromInts(row.ResizeEnabled, row.RefundEnabled)
		}
		raw, _ := json.Marshal(payload)
		return string(raw), true, nil
	case isListSettingKey(key):
		rows, err := r.listSettingListValues(ctx, key)
		if err != nil {
			return "", false, err
		}
		if len(rows) == 0 {
			return "", false, nil
		}
		out := make([]string, 0, len(rows))
		for _, row := range rows {
			out = append(out, row.Value)
		}
		raw, _ := json.Marshal(out)
		return string(raw), true, nil
	case strings.HasPrefix(key, "task."):
		taskKey := strings.TrimSpace(strings.TrimPrefix(key, "task."))
		if taskKey == "" {
			return "", false, nil
		}
		row, found, err := r.getScheduledTaskConfigRow(ctx, taskKey)
		if err != nil {
			return "", false, err
		}
		if !found {
			return "", false, nil
		}
		raw, err := marshalTaskSettingRow(row)
		if err != nil {
			return "", false, err
		}
		return raw, true, nil
	default:
		return "", false, nil
	}
}

func (r *GormRepo) upsertNormalizedSetting(db *gorm.DB, setting domain.Setting) (bool, error) {
	key := strings.TrimSpace(setting.Key)
	switch {
	case key == settingKeySMSTemplates:
		return true, r.upsertSMSTemplatesSetting(db, setting.ValueJSON)
	case key == settingKeyRobotWebhooks:
		return true, r.upsertRobotWebhooksSetting(db, setting.ValueJSON)
	case key == settingKeyPackageCapabilities:
		return true, r.upsertPackageCapabilitiesSetting(db, setting.ValueJSON)
	case key == settingKeyGoodsCapabilities:
		return true, r.upsertGoodsTypeCapabilitiesSetting(db, setting.ValueJSON)
	case key == "site_nav_items":
		// site_nav_items is a JSON document (array of nav objects) stored as a
		// raw settings row, not as scalar list values. It is part of the shared
		// JSON-settings key set (P3-4) but must NOT be routed through the list
		// write path; return (false, nil) so the caller does the raw upsert.
		return false, nil
	case isListSettingKey(key):
		return true, r.upsertListSetting(db, key, setting.ValueJSON)
	case strings.HasPrefix(key, "task."):
		taskKey := strings.TrimSpace(strings.TrimPrefix(key, "task."))
		if taskKey == "" {
			return true, appshared.ErrInvalidInput
		}
		return true, r.upsertTaskSetting(db, taskKey, setting.ValueJSON)
	default:
		return false, nil
	}
}

func isListSettingKey(key string) bool {
	_, ok := normalizedListSettingKeys[strings.TrimSpace(key)]
	return ok
}

func (r *GormRepo) upsertSMSTemplatesSetting(db *gorm.DB, raw string) error {
	type smsTemplatePayload struct {
		ID        int64     `json:"id"`
		Name      string    `json:"name"`
		Content   string    `json:"content"`
		Enabled   bool      `json:"enabled"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	var items []smsTemplatePayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &items); err != nil {
		return appshared.ErrInvalidInput
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&smsTemplateRow{}).Error; err != nil {
			return err
		}
		now := time.Now()
		for i := range items {
			item := items[i]
			name := strings.TrimSpace(item.Name)
			content := strings.TrimSpace(item.Content)
			if name == "" || content == "" {
				continue
			}
			row := smsTemplateRow{
				Name:      name,
				Content:   content,
				Enabled:   boolToInt(item.Enabled),
				CreatedAt: now,
				UpdatedAt: now,
			}
			if !item.CreatedAt.IsZero() {
				row.CreatedAt = item.CreatedAt
			}
			if !item.UpdatedAt.IsZero() {
				row.UpdatedAt = item.UpdatedAt
			}
			if item.ID > 0 {
				row.ID = item.ID
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return tx.Where("`key` = ?", settingKeySMSTemplates).Delete(&settingModel{}).Error
	})
}

func (r *GormRepo) upsertRobotWebhooksSetting(db *gorm.DB, raw string) error {
	type webhookPayload struct {
		Name    string   `json:"name"`
		URL     string   `json:"url"`
		Secret  string   `json:"secret"`
		Enabled bool     `json:"enabled"`
		Events  []string `json:"events"`
	}
	var items []webhookPayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &items); err != nil {
		return appshared.ErrInvalidInput
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&robotWebhookRow{}).Error; err != nil {
			return err
		}
		for i, item := range items {
			eventsRaw, _ := json.Marshal(item.Events)
			row := robotWebhookRow{
				Name:       strings.TrimSpace(item.Name),
				URL:        strings.TrimSpace(item.URL),
				Secret:     strings.TrimSpace(item.Secret),
				Enabled:    boolToInt(item.Enabled),
				EventsJSON: string(eventsRaw),
				SortOrder:  i,
			}
			if row.URL == "" {
				continue
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return tx.Where("`key` = ?", settingKeyRobotWebhooks).Delete(&settingModel{}).Error
	})
}

func (r *GormRepo) upsertPackageCapabilitiesSetting(db *gorm.DB, raw string) error {
	var payload map[string]domain.CapabilityPolicy
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return appshared.ErrInvalidInput
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&packageCapabilityRow{}).Error; err != nil {
			return err
		}
		for key, item := range payload {
			packageID, err := strconv.ParseInt(strings.TrimSpace(key), 10, 64)
			if err != nil || packageID <= 0 {
				continue
			}
			row := packageCapabilityRow{PackageID: packageID}
			if item.ResizeEnabled != nil {
				v := 0
				if *item.ResizeEnabled {
					v = 1
				}
				row.ResizeEnabled = &v
			}
			if item.RefundEnabled != nil {
				v := 0
				if *item.RefundEnabled {
					v = 1
				}
				row.RefundEnabled = &v
			}
			if row.ResizeEnabled == nil && row.RefundEnabled == nil {
				continue
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return upsertCapabilityLockRow(tx, settingKeyPackageCapabilities)
	})
}

func (r *GormRepo) upsertGoodsTypeCapabilitiesSetting(db *gorm.DB, raw string) error {
	var payload map[string]domain.CapabilityPolicy
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return appshared.ErrInvalidInput
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&goodsTypeCapabilityRow{}).Error; err != nil {
			return err
		}
		for key, item := range payload {
			goodsTypeID, err := strconv.ParseInt(strings.TrimSpace(key), 10, 64)
			if err != nil || goodsTypeID <= 0 {
				continue
			}
			row := goodsTypeCapabilityRow{GoodsTypeID: goodsTypeID}
			row.ResizeEnabled = optionalBoolToInt(item.ResizeEnabled)
			row.RefundEnabled = optionalBoolToInt(item.RefundEnabled)
			if row.ResizeEnabled == nil && row.RefundEnabled == nil {
				continue
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return upsertCapabilityLockRow(tx, settingKeyGoodsCapabilities)
	})
}

func upsertCapabilityLockRow(db *gorm.DB, key string) error {
	row := settingModel{Key: key, ValueJSON: "{}", UpdatedAt: time.Now()}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value_json", "updated_at"}),
	}).Create(&row).Error
}

func optionalBoolToInt(value *bool) *int {
	if value == nil {
		return nil
	}
	v := boolToInt(*value)
	return &v
}

func capabilityPolicyFromInts(resize, refund *int) domain.CapabilityPolicy {
	policy := domain.CapabilityPolicy{}
	if resize != nil {
		v := *resize == 1
		policy.ResizeEnabled = &v
	}
	if refund != nil {
		v := *refund == 1
		policy.RefundEnabled = &v
	}
	return policy
}

func (r *GormRepo) upsertListSetting(db *gorm.DB, key string, raw string) error {
	var values []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &values); err != nil {
		return appshared.ErrInvalidInput
	}
	uniq := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, item := range values {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		if _, ok := uniq[v]; ok {
			continue
		}
		uniq[v] = struct{}{}
		normalized = append(normalized, v)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("setting_key = ?", key).Delete(&settingListValueRow{}).Error; err != nil {
			return err
		}
		if len(normalized) > 0 {
			rows := make([]settingListValueRow, 0, len(normalized))
			for i, v := range normalized {
				rows = append(rows, settingListValueRow{
					SettingKey: key,
					Value:      v,
					SortOrder:  i,
				})
			}
			if err := tx.Create(&rows).Error; err != nil {
				return err
			}
			return tx.Where("`key` = ?", key).Delete(&settingModel{}).Error
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value_json", "updated_at"}),
		}).Create(&settingModel{Key: key, ValueJSON: "[]", UpdatedAt: time.Now()}).Error
	})
}

func (r *GormRepo) upsertTaskSetting(db *gorm.DB, taskKey string, raw string) error {
	var payload struct {
		Enabled     *bool   `json:"enabled"`
		Strategy    *string `json:"strategy"`
		IntervalSec *int    `json:"interval_sec"`
		DailyAt     *string `json:"daily_at"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return appshared.ErrInvalidInput
	}
	row := scheduledTaskConfigRow{TaskKey: taskKey}
	if payload.Enabled != nil {
		row.Enabled = boolToInt(*payload.Enabled)
	}
	if payload.Strategy != nil {
		row.Strategy = strings.TrimSpace(*payload.Strategy)
	}
	if payload.IntervalSec != nil {
		row.IntervalSec = *payload.IntervalSec
	}
	if payload.DailyAt != nil {
		row.DailyAt = strings.TrimSpace(*payload.DailyAt)
	}
	if row.Strategy == "" {
		row.Strategy = "interval"
	}
	if row.IntervalSec <= 0 {
		row.IntervalSec = 60
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "task_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"enabled", "strategy", "interval_sec", "daily_at", "updated_at"}),
		}).Create(&row).Error; err != nil {
			return err
		}
		return tx.Where("`key` = ?", "task."+taskKey).Delete(&settingModel{}).Error
	})
}

func (r *GormRepo) upsertLegacySettingOnly(ctx context.Context, key string, value string) error {
	m := settingModel{Key: key, ValueJSON: value, UpdatedAt: time.Now()}
	return r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value_json", "updated_at"}),
		}).
		Create(&m).Error
}

func (r *GormRepo) listSMSTemplateRows(ctx context.Context) ([]smsTemplateRow, error) {
	var rows []smsTemplateRow
	if err := r.gdb.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *GormRepo) listRobotWebhookRows(ctx context.Context) ([]robotWebhookRow, error) {
	var rows []robotWebhookRow
	if err := r.gdb.WithContext(ctx).Order("sort_order ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *GormRepo) listPackageCapabilityRows(ctx context.Context) ([]packageCapabilityRow, error) {
	var rows []packageCapabilityRow
	if err := r.gdb.WithContext(ctx).Order("package_id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *GormRepo) listGoodsTypeCapabilityRows(ctx context.Context) ([]goodsTypeCapabilityRow, error) {
	var rows []goodsTypeCapabilityRow
	if err := r.gdb.WithContext(ctx).Order("goods_type_id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *GormRepo) ListCapabilityPolicies(ctx context.Context, scope domain.CapabilityPolicyScope) (map[int64]domain.CapabilityPolicy, error) {
	out := map[int64]domain.CapabilityPolicy{}
	switch scope {
	case domain.CapabilityPolicyScopePackage:
		rows, err := r.listPackageCapabilityRows(ctx)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			out[row.PackageID] = capabilityPolicyFromInts(row.ResizeEnabled, row.RefundEnabled)
		}
	case domain.CapabilityPolicyScopeGoodsType:
		rows, err := r.listGoodsTypeCapabilityRows(ctx)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			out[row.GoodsTypeID] = capabilityPolicyFromInts(row.ResizeEnabled, row.RefundEnabled)
		}
	default:
		return nil, appshared.ErrInvalidInput
	}
	return out, nil
}

func (r *GormRepo) GetCapabilityPolicy(ctx context.Context, scope domain.CapabilityPolicyScope, itemID int64) (domain.CapabilityPolicy, error) {
	if itemID <= 0 {
		return domain.CapabilityPolicy{}, appshared.ErrInvalidInput
	}
	switch scope {
	case domain.CapabilityPolicyScopePackage:
		var row packageCapabilityRow
		if err := r.gdb.WithContext(ctx).Where("package_id = ?", itemID).First(&row).Error; err != nil {
			return domain.CapabilityPolicy{}, r.ensure(err)
		}
		return capabilityPolicyFromInts(row.ResizeEnabled, row.RefundEnabled), nil
	case domain.CapabilityPolicyScopeGoodsType:
		var row goodsTypeCapabilityRow
		if err := r.gdb.WithContext(ctx).Where("goods_type_id = ?", itemID).First(&row).Error; err != nil {
			return domain.CapabilityPolicy{}, r.ensure(err)
		}
		return capabilityPolicyFromInts(row.ResizeEnabled, row.RefundEnabled), nil
	default:
		return domain.CapabilityPolicy{}, appshared.ErrInvalidInput
	}
}

func (r *GormRepo) UpsertCapabilityPolicy(ctx context.Context, scope domain.CapabilityPolicyScope, itemID int64, policy domain.CapabilityPolicy) error {
	if itemID <= 0 {
		return appshared.ErrInvalidInput
	}
	resize := optionalBoolToInt(policy.ResizeEnabled)
	refund := optionalBoolToInt(policy.RefundEnabled)
	db := r.gdb.WithContext(ctx)
	switch scope {
	case domain.CapabilityPolicyScopePackage:
		if resize == nil && refund == nil {
			return db.Where("package_id = ?", itemID).Delete(&packageCapabilityRow{}).Error
		}
		row := packageCapabilityRow{PackageID: itemID, ResizeEnabled: resize, RefundEnabled: refund}
		return db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "package_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"resize_enabled", "refund_enabled", "updated_at"}),
		}).Create(&row).Error
	case domain.CapabilityPolicyScopeGoodsType:
		if resize == nil && refund == nil {
			return db.Where("goods_type_id = ?", itemID).Delete(&goodsTypeCapabilityRow{}).Error
		}
		row := goodsTypeCapabilityRow{GoodsTypeID: itemID, ResizeEnabled: resize, RefundEnabled: refund}
		return db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "goods_type_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"resize_enabled", "refund_enabled", "updated_at"}),
		}).Create(&row).Error
	default:
		return appshared.ErrInvalidInput
	}
}

func (r *GormRepo) listSettingListValues(ctx context.Context, key string) ([]settingListValueRow, error) {
	var rows []settingListValueRow
	if err := r.gdb.WithContext(ctx).
		Where("setting_key = ?", key).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *GormRepo) getScheduledTaskConfigRow(ctx context.Context, taskKey string) (scheduledTaskConfigRow, bool, error) {
	var row scheduledTaskConfigRow
	if err := r.gdb.WithContext(ctx).Where("task_key = ?", taskKey).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return scheduledTaskConfigRow{}, false, nil
		}
		return scheduledTaskConfigRow{}, false, err
	}
	return row, true, nil
}

func (r *GormRepo) listScheduledTaskConfigRows(ctx context.Context) ([]scheduledTaskConfigRow, error) {
	var rows []scheduledTaskConfigRow
	if err := r.gdb.WithContext(ctx).Order("task_key ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func marshalTaskSettingRow(row scheduledTaskConfigRow) (string, error) {
	payload := map[string]any{
		"enabled":      row.Enabled == 1,
		"strategy":     strings.TrimSpace(row.Strategy),
		"interval_sec": row.IntervalSec,
		"daily_at":     strings.TrimSpace(row.DailyAt),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
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
		ContentDigest:   inst.ContentDigest,
		ConfigCipher:    inst.ConfigCipher,
	}
	return r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "category"}, {Name: "plugin_id"}, {Name: "instance_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"enabled",
				"signature_status",
				"content_digest",
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
		ContentDigest:   row.ContentDigest,
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
			ContentDigest:   row.ContentDigest,
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

func (r *GormRepo) PurgeSyncLogs(ctx context.Context, before time.Time) error {
	return r.gdb.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&integrationSyncLogRow{}).Error
}

// UpdateSettingAtomic applies mutate to the stored value of a settings key
// inside a transaction with a row lock (P2-8). Concurrent read-modify-write of
// a shared settings document across instances is serialized on the row, so a
// later writer re-reads the other writer's committed value instead of
// last-write-wins dropping one key.
func (r *GormRepo) UpdateSettingAtomic(ctx context.Context, key string, mutate func(current string) (string, error)) error {
	key = strings.TrimSpace(key)
	if key == "" || mutate == nil {
		return appshared.ErrInvalidInput
	}
	// Seed a stable lock row before entering the transaction. SELECT FOR UPDATE
	// cannot serialize a row that does not exist; a conflict-safe insert closes
	// the first-writer race on MySQL/Postgres. Capability policy updates use
	// independent row upserts on every dialect and do not depend on this lock.
	seed := settingModel{Key: key, ValueJSON: "", UpdatedAt: time.Now()}
	if key == settingKeyPackageCapabilities || key == settingKeyGoodsCapabilities {
		seed.ValueJSON = "{}"
	}
	if err := r.gdb.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&seed).Error; err != nil {
		return err
	}
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row settingModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("key = ?", key).
			First(&row).Error; err != nil {
			return err
		}
		current := row.ValueJSON
		if key == settingKeyPackageCapabilities || key == settingKeyGoodsCapabilities {
			var err error
			current, err = capabilitySettingValueFromDB(tx, key)
			if err != nil {
				return err
			}
		}
		next, err := mutate(current)
		if err != nil {
			return err
		}
		if key == settingKeyPackageCapabilities {
			return r.upsertPackageCapabilitiesSetting(tx, next)
		}
		if key == settingKeyGoodsCapabilities {
			return r.upsertGoodsTypeCapabilitiesSetting(tx, next)
		}
		return tx.Model(&settingModel{}).Where("key = ?", key).
			Updates(map[string]any{"value_json": next, "updated_at": time.Now()}).Error
	})
}

func capabilitySettingValueFromDB(db *gorm.DB, key string) (string, error) {
	payload := map[string]domain.CapabilityPolicy{}
	switch key {
	case settingKeyPackageCapabilities:
		var rows []packageCapabilityRow
		if err := db.Order("package_id ASC").Find(&rows).Error; err != nil {
			return "", err
		}
		for _, row := range rows {
			payload[strconv.FormatInt(row.PackageID, 10)] = capabilityPolicyFromInts(row.ResizeEnabled, row.RefundEnabled)
		}
	case settingKeyGoodsCapabilities:
		var rows []goodsTypeCapabilityRow
		if err := db.Order("goods_type_id ASC").Find(&rows).Error; err != nil {
			return "", err
		}
		for _, row := range rows {
			payload[strconv.FormatInt(row.GoodsTypeID, 10)] = capabilityPolicyFromInts(row.ResizeEnabled, row.RefundEnabled)
		}
	default:
		return "", appshared.ErrInvalidInput
	}
	raw, err := json.Marshal(payload)
	return string(raw), err
}
