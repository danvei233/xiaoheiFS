package repo

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNewGormRepoNilDoesNotPanic(t *testing.T) {
	if r := NewGormRepo(nil); r == nil {
		t.Fatal("NewGormRepo(nil) returned nil")
	}
}

func TestNewGormRepoKeepsHandles(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "repo.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	defer sqlDB.Close()

	r := NewGormRepo(db)
	if r == nil || r.gdb == nil {
		t.Fatal("NewGormRepo did not keep the gorm handle")
	}
	if r.db == nil {
		t.Fatal("NewGormRepo did not keep the sql handle")
	}
}
