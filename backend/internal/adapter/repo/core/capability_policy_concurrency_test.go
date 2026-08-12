package repo_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"xiaoheiplay/internal/adapter/repo/core"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
)

func TestCapabilityPolicies_IndependentRowsSurviveConcurrentInstances(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "capabilities.db")
	connA, err := db.Open(config.Config{DBType: "sqlite", DBPath: dbPath})
	if err != nil {
		t.Fatalf("open A: %v", err)
	}
	t.Cleanup(func() { _ = connA.SQL.Close() })
	if err := repo.Migrate(connA.Gorm); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	connB, err := db.Open(config.Config{DBType: "sqlite", DBPath: dbPath})
	if err != nil {
		t.Fatalf("open B: %v", err)
	}
	t.Cleanup(func() { _ = connB.SQL.Close() })

	repos := []*repo.GormRepo{repo.NewGormRepo(connA.Gorm), repo.NewGormRepo(connB.Gorm)}
	start := make(chan struct{})
	var wg sync.WaitGroup
	const count = 20
	for i := int64(1); i <= count; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			<-start
			enabled := true
			if err := repos[id%2].UpsertCapabilityPolicy(context.Background(), domain.CapabilityPolicyScopeGoodsType, id, domain.CapabilityPolicy{ResizeEnabled: &enabled}); err != nil {
				t.Errorf("upsert %d: %v", id, err)
			}
		}(i)
	}
	close(start)
	wg.Wait()

	stored, err := repos[0].ListCapabilityPolicies(context.Background(), domain.CapabilityPolicyScopeGoodsType)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(stored) != count {
		t.Fatalf("got %d policies, want %d", len(stored), count)
	}
}

func TestCapabilityPolicies_PackageAndGoodsTypeIDsDoNotCollide(t *testing.T) {
	_, repository := newCapabilityTestDB(t)
	disabled := false
	enabled := true
	ctx := context.Background()
	if err := repository.UpsertCapabilityPolicy(ctx, domain.CapabilityPolicyScopePackage, 7, domain.CapabilityPolicy{ResizeEnabled: &disabled}); err != nil {
		t.Fatal(err)
	}
	if err := repository.UpsertCapabilityPolicy(ctx, domain.CapabilityPolicyScopeGoodsType, 7, domain.CapabilityPolicy{ResizeEnabled: &enabled}); err != nil {
		t.Fatal(err)
	}
	pkg, err := repository.GetCapabilityPolicy(ctx, domain.CapabilityPolicyScopePackage, 7)
	if err != nil || pkg.ResizeEnabled == nil || *pkg.ResizeEnabled {
		t.Fatalf("package policy mismatch: %+v err=%v", pkg, err)
	}
	goodsType, err := repository.GetCapabilityPolicy(ctx, domain.CapabilityPolicyScopeGoodsType, 7)
	if err != nil || goodsType.ResizeEnabled == nil || !*goodsType.ResizeEnabled {
		t.Fatalf("goods type policy mismatch: %+v err=%v", goodsType, err)
	}
}

func newCapabilityTestDB(t *testing.T) (*db.Conn, *repo.GormRepo) {
	t.Helper()
	conn, err := db.Open(config.Config{DBType: "sqlite", DBPath: filepath.Join(t.TempDir(), "capability.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.SQL.Close() })
	if err := repo.Migrate(conn.Gorm); err != nil {
		t.Fatal(err)
	}
	return conn, repo.NewGormRepo(conn.Gorm)
}
