package repo

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"xiaoheiplay/internal/domain"
)

func TestNormalizeLegacyCapabilitySettings_NewerRawDocumentWins(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:capability-migrate?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&settingModel{}, &packageCapabilityRow{}, &goodsTypeCapabilityRow{}); err != nil {
		t.Fatal(err)
	}
	old := false
	if err := db.Create(&packageCapabilityRow{PackageID: 7, ResizeEnabled: optionalBoolToInt(&old), UpdatedAt: time.Now().Add(-time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&settingModel{Key: settingKeyPackageCapabilities, ValueJSON: `{"7":{"resize_enabled":true}}`, UpdatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := normalizeLegacyCapabilitySettings(db); err != nil {
		t.Fatal(err)
	}
	policy, err := NewGormRepo(db).GetCapabilityPolicy(context.Background(), domain.CapabilityPolicyScopePackage, 7)
	if err != nil || policy.ResizeEnabled == nil || !*policy.ResizeEnabled {
		t.Fatalf("newer raw policy was lost: %+v err=%v", policy, err)
	}
}
