package usecase_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"xiaoheiplay/internal/adapter/repo"
	"xiaoheiplay/internal/adapter/seed"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
	"xiaoheiplay/internal/usecase"
)

func newTestRepo(t *testing.T) *repo.SQLiteRepo {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "usecase.db")
	conn, err := db.Open(config.Config{DBType: "sqlite", DBPath: dbPath})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.SQL.Close()
	})
	if err := repo.Migrate(conn.Gorm); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seed.EnsureSettings(conn.SQL, conn.Dialect); err != nil {
		t.Fatalf("seed settings: %v", err)
	}
	if err := seed.EnsurePermissionDefaults(conn.SQL, conn.Dialect); err != nil {
		t.Fatalf("seed permissions: %v", err)
	}
	if err := seed.EnsurePermissionGroups(conn.SQL, conn.Dialect); err != nil {
		t.Fatalf("seed permission groups: %v", err)
	}
	return repo.NewSQLiteRepo(conn.Gorm)
}

func TestCatalogService_Getters(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	region := domain.Region{Code: "r1", Name: "Region1", Active: true}
	if err := repo.CreateRegion(ctx, &region); err != nil {
		t.Fatalf("create region: %v", err)
	}
	plan := domain.PlanGroup{RegionID: region.ID, Name: "Plan1", LineID: 1, UnitCore: 1, UnitMem: 1, UnitDisk: 1, UnitBW: 1, Active: true, Visible: true}
	if err := repo.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan group: %v", err)
	}
	pkg := domain.Package{PlanGroupID: plan.ID, Name: "Pkg1", Cores: 1, MemoryGB: 1, DiskGB: 10, BandwidthMB: 10, Monthly: 100, PortNum: 1, Active: true, Visible: true}
	if err := repo.CreatePackage(ctx, &pkg); err != nil {
		t.Fatalf("create package: %v", err)
	}
	img := domain.SystemImage{ImageID: 1, Name: "Ubuntu", Type: "linux", Enabled: true}
	if err := repo.CreateSystemImage(ctx, &img); err != nil {
		t.Fatalf("create image: %v", err)
	}
	cycle := domain.BillingCycle{Name: "monthly", Months: 1, Multiplier: 1, MinQty: 1, MaxQty: 12, Active: true, SortOrder: 1}
	if err := repo.CreateBillingCycle(ctx, &cycle); err != nil {
		t.Fatalf("create billing cycle: %v", err)
	}

	svc := usecase.NewCatalogService(repo, repo, repo)
	if _, _, _, _, _, err := svc.Catalog(ctx); err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if _, err := svc.GetRegion(ctx, region.ID); err != nil {
		t.Fatalf("get region: %v", err)
	}
	if _, err := svc.GetPlanGroup(ctx, plan.ID); err != nil {
		t.Fatalf("get plan group: %v", err)
	}
	if _, err := svc.GetPackage(ctx, pkg.ID); err != nil {
		t.Fatalf("get package: %v", err)
	}
}

func TestTicketService_Flow(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user := domain.User{Username: "ticketu", Email: "ticketu@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	svc := usecase.NewTicketService(repo, repo, repo, nil)
	ticket, _, _, err := svc.Create(ctx, user.ID, "subject", "content", []domain.TicketResource{{ResourceType: "order", ResourceID: 1, ResourceName: "O-1"}})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, _, err := svc.List(ctx, usecase.TicketFilter{UserID: &user.ID}); err != nil {
		t.Fatalf("list tickets: %v", err)
	}
	if _, err := svc.Get(ctx, ticket.ID); err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if _, _, _, err := svc.GetDetail(ctx, ticket.ID); err != nil {
		t.Fatalf("get detail: %v", err)
	}
	if err := svc.Close(ctx, ticket, user.ID); err != nil {
		t.Fatalf("close ticket: %v", err)
	}
	ticket.Status = "closed"
	if err := svc.AdminUpdate(ctx, ticket); err != nil {
		t.Fatalf("admin update: %v", err)
	}
	if err := svc.Delete(ctx, ticket.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestAuthCartVPSAndStatus(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user := domain.User{Username: "profileu", Email: "profileu@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	authSvc := usecase.NewAuthService(repo, repo)
	updated, err := authSvc.UpdateProfile(ctx, user.ID, usecase.UpdateProfileInput{Username: "profileu2", Email: "profileu2@example.com", QQ: "123"})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.Username != "profileu2" {
		t.Fatalf("username not updated")
	}

	cartSvc := usecase.NewCartService(repo, repo, repo)
	item := &domain.CartItem{UserID: user.ID, PackageID: 1, SystemID: 1, SpecJSON: "{}", Qty: 1, Amount: 990}
	if err := repo.AddCartItem(ctx, item); err != nil {
		t.Fatalf("add cart item: %v", err)
	}
	if _, err := cartSvc.List(ctx, user.ID); err != nil {
		t.Fatalf("list cart: %v", err)
	}

	order := domain.Order{UserID: user.ID, OrderNo: "O-VPS2", Status: domain.OrderStatusPendingPayment, TotalAmount: 100, Currency: "USD"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	items := []domain.OrderItem{{OrderID: order.ID, SpecJSON: "{}", Qty: 1, Amount: 100, Status: domain.OrderItemStatusPendingPayment, Action: "create", DurationMonths: 1}}
	if err := repo.CreateOrderItems(ctx, items); err != nil {
		t.Fatalf("create order items: %v", err)
	}
	expireAt := time.Now().Add(24 * time.Hour)
	inst := &domain.VPSInstance{UserID: user.ID, OrderItemID: items[0].ID, AutomationInstanceID: "1", Name: "vm", Status: domain.VPSStatusRunning, ExpireAt: &expireAt}
	if err := repo.CreateInstance(ctx, inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	vpsSvc := usecase.NewVPSService(repo, nil, repo)
	if _, err := vpsSvc.ListByUser(ctx, user.ID); err != nil {
		t.Fatalf("list vps: %v", err)
	}
	if err := vpsSvc.SetStatus(ctx, *inst, domain.VPSStatusRunning, 1); err != nil {
		t.Fatalf("set status: %v", err)
	}

	provider := &fakeSystemProvider{}
	statusSvc := usecase.NewServerStatusService(provider)
	if _, err := statusSvc.Status(ctx); err != nil {
		t.Fatalf("status: %v", err)
	}
}

type fakeSystemProvider struct{}

func (p *fakeSystemProvider) Status(ctx context.Context) (usecase.ServerStatus, error) {
	return usecase.ServerStatus{Hostname: "host"}, nil
}
