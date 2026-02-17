package integration_test

import (
	"context"
	"testing"
	"time"
	appintegration "xiaoheiplay/internal/app/integration"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

type fakeAutomationSync struct {
	areas    []appshared.AutomationArea
	lines    []appshared.AutomationLine
	products map[int64][]appshared.AutomationProduct
	images   map[int64][]appshared.AutomationImage
}

func (f fakeAutomationSync) CreateHost(ctx context.Context, req appshared.AutomationCreateHostRequest) (appshared.AutomationCreateHostResult, error) {
	return appshared.AutomationCreateHostResult{}, nil
}
func (f fakeAutomationSync) GetHostInfo(ctx context.Context, hostID int64) (appshared.AutomationHostInfo, error) {
	return appshared.AutomationHostInfo{}, nil
}
func (f fakeAutomationSync) ListHostSimple(ctx context.Context, searchTag string) ([]appshared.AutomationHostSimple, error) {
	return nil, nil
}
func (f fakeAutomationSync) ElasticUpdate(ctx context.Context, req appshared.AutomationElasticUpdateRequest) error {
	return nil
}
func (f fakeAutomationSync) RenewHost(ctx context.Context, hostID int64, nextDueDate time.Time) error {
	return nil
}
func (f fakeAutomationSync) LockHost(ctx context.Context, hostID int64) error     { return nil }
func (f fakeAutomationSync) UnlockHost(ctx context.Context, hostID int64) error   { return nil }
func (f fakeAutomationSync) DeleteHost(ctx context.Context, hostID int64) error   { return nil }
func (f fakeAutomationSync) StartHost(ctx context.Context, hostID int64) error    { return nil }
func (f fakeAutomationSync) ShutdownHost(ctx context.Context, hostID int64) error { return nil }
func (f fakeAutomationSync) RebootHost(ctx context.Context, hostID int64) error   { return nil }
func (f fakeAutomationSync) ResetOS(ctx context.Context, hostID int64, templateID int64, password string) error {
	return nil
}
func (f fakeAutomationSync) ResetOSPassword(ctx context.Context, hostID int64, password string) error {
	return nil
}
func (f fakeAutomationSync) ListSnapshots(ctx context.Context, hostID int64) ([]appshared.AutomationSnapshot, error) {
	return nil, nil
}
func (f fakeAutomationSync) CreateSnapshot(ctx context.Context, hostID int64) error { return nil }
func (f fakeAutomationSync) DeleteSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	return nil
}
func (f fakeAutomationSync) RestoreSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	return nil
}
func (f fakeAutomationSync) ListBackups(ctx context.Context, hostID int64) ([]appshared.AutomationBackup, error) {
	return nil, nil
}
func (f fakeAutomationSync) CreateBackup(ctx context.Context, hostID int64) error { return nil }
func (f fakeAutomationSync) DeleteBackup(ctx context.Context, hostID int64, backupID int64) error {
	return nil
}
func (f fakeAutomationSync) RestoreBackup(ctx context.Context, hostID int64, backupID int64) error {
	return nil
}
func (f fakeAutomationSync) ListFirewallRules(ctx context.Context, hostID int64) ([]appshared.AutomationFirewallRule, error) {
	return nil, nil
}
func (f fakeAutomationSync) AddFirewallRule(ctx context.Context, req appshared.AutomationFirewallRuleCreate) error {
	return nil
}
func (f fakeAutomationSync) DeleteFirewallRule(ctx context.Context, hostID int64, ruleID int64) error {
	return nil
}
func (f fakeAutomationSync) ListPortMappings(ctx context.Context, hostID int64) ([]appshared.AutomationPortMapping, error) {
	return nil, nil
}
func (f fakeAutomationSync) AddPortMapping(ctx context.Context, req appshared.AutomationPortMappingCreate) error {
	return nil
}
func (f fakeAutomationSync) FindPortCandidates(ctx context.Context, hostID int64, keywords string) ([]int64, error) {
	return []int64{}, nil
}
func (f fakeAutomationSync) DeletePortMapping(ctx context.Context, hostID int64, mappingID int64) error {
	return nil
}
func (f fakeAutomationSync) GetPanelURL(ctx context.Context, hostName string, panelPassword string) (string, error) {
	return "", nil
}
func (f fakeAutomationSync) ListAreas(ctx context.Context) ([]appshared.AutomationArea, error) {
	return f.areas, nil
}
func (f fakeAutomationSync) ListImages(ctx context.Context, lineID int64) ([]appshared.AutomationImage, error) {
	return f.images[lineID], nil
}
func (f fakeAutomationSync) ListLines(ctx context.Context) ([]appshared.AutomationLine, error) {
	return f.lines, nil
}
func (f fakeAutomationSync) ListProducts(ctx context.Context, lineID int64) ([]appshared.AutomationProduct, error) {
	return f.products[lineID], nil
}
func (f fakeAutomationSync) GetMonitor(ctx context.Context, hostID int64) (appshared.AutomationMonitor, error) {
	return appshared.AutomationMonitor{}, nil
}
func (f fakeAutomationSync) GetVNCURL(ctx context.Context, hostID int64) (string, error) {
	return "", nil
}

func TestIntegrationService_SyncAutomation(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	gt := domain.GoodsType{
		Code:      "default",
		Name:      "Default",
		Active:    true,
		SortOrder: 1,
	}
	if err := repo.CreateGoodsType(context.Background(), &gt); err != nil {
		t.Fatalf("create goods type: %v", err)
	}
	auto := fakeAutomationSync{
		areas: []appshared.AutomationArea{{ID: 1, Name: "Area1", State: 1}},
		lines: []appshared.AutomationLine{{ID: 10, Name: "Line1", AreaID: 1, State: 1}},
		products: map[int64][]appshared.AutomationProduct{
			10: {{ID: 100, Name: "P1", CPU: 1, MemoryGB: 1, DiskGB: 10, Bandwidth: 10, Price: 100}},
		},
		images: map[int64][]appshared.AutomationImage{
			10: {{ImageID: 200, Name: "Ubuntu", Type: "linux"}},
		},
	}
	svc := appintegration.NewService(repo, repo, repo, repo, &testutil.FakeAutomationResolver{Client: auto}, repo)
	if _, err := svc.SyncAutomation(context.Background(), "merge"); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if _, err := repo.GetRegion(context.Background(), 1); err != nil {
		t.Fatalf("expected region created: %v", err)
	}
	items, _ := repo.ListPlanGroups(context.Background())
	if len(items) == 0 {
		t.Fatalf("expected plan groups")
	}
	pkgs, _ := repo.ListPackages(context.Background())
	if len(pkgs) == 0 {
		t.Fatalf("expected packages")
	}
	images, _ := repo.ListAllSystemImages(context.Background())
	if len(images) == 0 {
		t.Fatalf("expected images")
	}
}

func TestIntegrationService_SyncAutomationImagesForLine(t *testing.T) {
	ctx := context.Background()
	_, repo := testutil.NewTestDB(t, false)
	gt := domain.GoodsType{
		Code:      "default",
		Name:      "Default",
		Active:    true,
		SortOrder: 1,
	}
	if err := repo.CreateGoodsType(ctx, &gt); err != nil {
		t.Fatalf("create goods type: %v", err)
	}
	region := domain.Region{GoodsTypeID: gt.ID, Code: "r1", Name: "Region1", Active: true}
	if err := repo.CreateRegion(ctx, &region); err != nil {
		t.Fatalf("create region: %v", err)
	}
	plan := domain.PlanGroup{
		GoodsTypeID:       gt.ID,
		RegionID:          region.ID,
		Name:              "Line1",
		LineID:            10,
		UnitCore:          1,
		UnitMem:           1,
		UnitDisk:          1,
		UnitBW:            1,
		Active:            true,
		Visible:           true,
		CapacityRemaining: -1,
	}
	if err := repo.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan group: %v", err)
	}

	auto := fakeAutomationSync{
		images: map[int64][]appshared.AutomationImage{
			10: {
				{ImageID: 201, Name: "Ubuntu 22.04", Type: "linux"},
				{ImageID: 202, Name: "Windows 2022", Type: "windows"},
			},
		},
	}
	svc := appintegration.NewService(repo, repo, repo, repo, &testutil.FakeAutomationResolver{Client: auto}, repo)
	count, err := svc.SyncAutomationImagesForLine(ctx, 10, "merge")
	if err != nil {
		t.Fatalf("sync line images: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected count=2, got %d", count)
	}
	items, err := repo.ListSystemImages(ctx, 10)
	if err != nil {
		t.Fatalf("list line images: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 line images, got %d", len(items))
	}
}

func TestIntegrationService_SyncAutomationImagesForLine_PlanGroupIDFallback(t *testing.T) {
	ctx := context.Background()
	_, repo := testutil.NewTestDB(t, false)
	gt := domain.GoodsType{
		Code:      "default",
		Name:      "Default",
		Active:    true,
		SortOrder: 1,
	}
	if err := repo.CreateGoodsType(ctx, &gt); err != nil {
		t.Fatalf("create goods type: %v", err)
	}
	region := domain.Region{GoodsTypeID: gt.ID, Code: "r1", Name: "Region1", Active: true}
	if err := repo.CreateRegion(ctx, &region); err != nil {
		t.Fatalf("create region: %v", err)
	}
	plan := domain.PlanGroup{
		GoodsTypeID:       gt.ID,
		RegionID:          region.ID,
		Name:              "Line1",
		LineID:            10,
		UnitCore:          1,
		UnitMem:           1,
		UnitDisk:          1,
		UnitBW:            1,
		Active:            true,
		Visible:           true,
		CapacityRemaining: -1,
	}
	if err := repo.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan group: %v", err)
	}

	auto := fakeAutomationSync{
		images: map[int64][]appshared.AutomationImage{
			10: {
				{ImageID: 301, Name: "Ubuntu", Type: "linux"},
			},
		},
	}
	svc := appintegration.NewService(repo, repo, repo, repo, &testutil.FakeAutomationResolver{Client: auto}, repo)
	count, err := svc.SyncAutomationImagesForLine(ctx, plan.ID, "merge")
	if err != nil {
		t.Fatalf("sync line images by plan group id: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count=1, got %d", count)
	}
	items, err := repo.ListSystemImages(ctx, 10)
	if err != nil {
		t.Fatalf("list line images: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 line image, got %d", len(items))
	}
}

var _ appshared.AutomationClient = (*fakeAutomationSync)(nil)
