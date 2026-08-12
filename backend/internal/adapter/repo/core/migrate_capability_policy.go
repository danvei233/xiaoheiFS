package repo

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	appshared "xiaoheiplay/internal/app/shared"
)

// normalizeLegacyCapabilitySettings migrates the old shared JSON documents to
// independently updateable policy rows. A mixed-version deployment may have
// both forms; the newer representation wins.
func normalizeLegacyCapabilitySettings(db *gorm.DB) error {
	type capabilityTable struct {
		key   string
		model any
	}
	for _, item := range []capabilityTable{
		{key: settingKeyPackageCapabilities, model: &packageCapabilityRow{}},
		{key: settingKeyGoodsCapabilities, model: &goodsTypeCapabilityRow{}},
	} {
		var count int64
		if err := db.Model(item.model).Count(&count).Error; err != nil {
			return err
		}
		var raw settingModel
		err := db.Where("`key` = ?", item.key).First(&raw).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if count == 0 {
				continue
			}
			if err := upsertCapabilityLockRow(db, item.key); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if count > 0 {
			latest := struct{ UpdatedAt time.Time }{}
			if err := db.Model(item.model).Select("updated_at").Order("updated_at DESC").Take(&latest).Error; err != nil {
				return err
			}
			if raw.UpdatedAt.After(latest.UpdatedAt) && strings.TrimSpace(raw.ValueJSON) != "" && strings.TrimSpace(raw.ValueJSON) != "{}" {
				if !json.Valid([]byte(raw.ValueJSON)) {
					return appshared.ErrInvalidInput
				}
				repository := &GormRepo{gdb: db}
				if item.key == settingKeyPackageCapabilities {
					if err := repository.upsertPackageCapabilitiesSetting(db, raw.ValueJSON); err != nil {
						return err
					}
					continue
				}
				if err := repository.upsertGoodsTypeCapabilitiesSetting(db, raw.ValueJSON); err != nil {
					return err
				}
				continue
			}
			if err := upsertCapabilityLockRow(db, item.key); err != nil {
				return err
			}
			continue
		}
		repository := &GormRepo{gdb: db}
		if item.key == settingKeyPackageCapabilities {
			if err := repository.upsertPackageCapabilitiesSetting(db, raw.ValueJSON); err != nil {
				return err
			}
			continue
		}
		if err := repository.upsertGoodsTypeCapabilitiesSetting(db, raw.ValueJSON); err != nil {
			return err
		}
	}
	return nil
}
