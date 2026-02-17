package app_test

import (
	"context"
	"testing"
	appcms "xiaoheiplay/internal/app/cms"
	appmessage "xiaoheiplay/internal/app/message"
	apprealname "xiaoheiplay/internal/app/realname"
	appreport "xiaoheiplay/internal/app/report"
	appscheduledtask "xiaoheiplay/internal/app/scheduledtask"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestCMSService_BasicFlow(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	messageSvc := appmessage.NewService(repo, repo)
	svc := appcms.NewService(repo, repo, repo, messageSvc)

	cat := domain.CMSCategory{Key: "announcements", Name: "Ann", Lang: "zh-CN", Visible: true}
	if err := svc.CreateCategory(context.Background(), &cat); err != nil {
		t.Fatalf("create category: %v", err)
	}
	if list, err := svc.ListCategories(context.Background(), "zh-CN", true); err != nil || len(list) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	if _, err := svc.GetCategory(context.Background(), cat.ID); err != nil {
		t.Fatalf("get category: %v", err)
	}
	cat.Name = "Ann2"
	if err := svc.UpdateCategory(context.Background(), cat); err != nil {
		t.Fatalf("update category: %v", err)
	}

	post := domain.CMSPost{CategoryID: cat.ID, Title: "Post", Slug: "post-1", Summary: "sum", ContentHTML: "body", Lang: "zh-CN", Status: "published"}
	if err := svc.CreatePost(context.Background(), &post); err != nil {
		t.Fatalf("create post: %v", err)
	}
	if _, err := svc.GetPost(context.Background(), post.ID); err != nil {
		t.Fatalf("get post: %v", err)
	}
	if _, err := svc.GetPostBySlug(context.Background(), "post-1"); err != nil {
		t.Fatalf("get post by slug: %v", err)
	}
	post.Title = "Post2"
	if err := svc.UpdatePost(context.Background(), post); err != nil {
		t.Fatalf("update post: %v", err)
	}
	if _, _, err := svc.ListPosts(context.Background(), appshared.CMSPostFilter{Lang: "zh-CN", Limit: 10}); err != nil {
		t.Fatalf("list posts: %v", err)
	}

	block := domain.CMSBlock{Page: "home", Type: "html", Title: "Hero", Lang: "zh-CN", Visible: true}
	if err := svc.CreateBlock(context.Background(), &block); err != nil {
		t.Fatalf("create block: %v", err)
	}
	if _, err := svc.GetBlock(context.Background(), block.ID); err != nil {
		t.Fatalf("get block: %v", err)
	}
	block.Title = "Hero2"
	if err := svc.UpdateBlock(context.Background(), block); err != nil {
		t.Fatalf("update block: %v", err)
	}
	if _, err := svc.ListBlocks(context.Background(), "home", "zh-CN", true); err != nil {
		t.Fatalf("list blocks: %v", err)
	}

	if err := svc.DeletePost(context.Background(), post.ID); err != nil {
		t.Fatalf("delete post: %v", err)
	}
	if err := svc.DeleteBlock(context.Background(), block.ID); err != nil {
		t.Fatalf("delete block: %v", err)
	}
	if err := svc.DeleteCategory(context.Background(), cat.ID); err != nil {
		t.Fatalf("delete category: %v", err)
	}
}

func TestRealNameService_Flow(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "rn", "rn@example.com", "pass")
	reg := testutil.NewFakeRealNameRegistry()
	svc := apprealname.NewService(repo, reg, repo)

	if err := svc.UpdateConfig(context.Background(), true, "fake", []string{"purchase_vps"}); err != nil {
		t.Fatalf("update config: %v", err)
	}
	if enabled, provider, _ := svc.GetConfig(context.Background()); !enabled || provider != "fake" {
		t.Fatalf("expected enabled config")
	}
	record, err := svc.Verify(context.Background(), user.ID, "Test User", "11010519491231002X")
	if err != nil || record.Status != "verified" {
		t.Fatalf("verify: %v", err)
	}
	if _, err := svc.Latest(context.Background(), user.ID); err != nil {
		t.Fatalf("latest: %v", err)
	}
	if list, _, err := svc.List(context.Background(), &user.ID, 10, 0); err != nil || len(list) == 0 {
		t.Fatalf("list: %v", err)
	}
	if list := svc.Providers(); len(list) == 0 {
		t.Fatalf("providers empty")
	}
	if err := svc.RequireAction(context.Background(), user.ID, "purchase_vps"); err != nil {
		t.Fatalf("require action: %v", err)
	}
}

func TestRealNameService_EmptyBlockActionsDoesNotIntercept(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "rn-empty", "rn-empty@example.com", "pass")
	reg := testutil.NewFakeRealNameRegistry()
	svc := apprealname.NewService(repo, reg, repo)

	if err := svc.UpdateConfig(context.Background(), true, "fake", []string{}); err != nil {
		t.Fatalf("update config: %v", err)
	}
	_, _, actions := svc.GetConfig(context.Background())
	if len(actions) != 0 {
		t.Fatalf("expected empty block actions, got %v", actions)
	}
	if err := svc.RequireAction(context.Background(), user.ID, "purchase_vps"); err != nil {
		t.Fatalf("unexpected intercept for empty block_actions: %v", err)
	}
}

func TestReportService_OverviewAndSeries(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "report", "report@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-RPT", Status: domain.OrderStatusPendingReview, TotalAmount: 2000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, Amount: 2000, Status: domain.OrderItemStatusPendingReview, Action: "create", SpecJSON: "{}"}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)
	pay := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "manual", Amount: 2000, Currency: "CNY", TradeNo: "TN-RPT", Status: domain.PaymentStatusApproved}
	if err := repo.CreatePayment(context.Background(), &pay); err != nil {
		t.Fatalf("create payment: %v", err)
	}
	inst := domain.VPSInstance{UserID: user.ID, OrderItemID: items[0].ID, Name: "vm", Status: domain.VPSStatusRunning, SpecJSON: "{}"}
	_ = repo.CreateInstance(context.Background(), &inst)

	svc := appreport.NewService(repo, repo, repo)
	if _, err := svc.Overview(context.Background()); err != nil {
		t.Fatalf("overview: %v", err)
	}
	if _, err := svc.RevenueByDay(context.Background(), 7); err != nil {
		t.Fatalf("revenue by day: %v", err)
	}
	if _, err := svc.RevenueByMonth(context.Background(), 2); err != nil {
		t.Fatalf("revenue by month: %v", err)
	}
	if _, err := svc.VPSStatus(context.Background()); err != nil {
		t.Fatalf("vps status: %v", err)
	}
}

func TestScheduledTaskService_ListAndUpdate(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appscheduledtask.NewService(repo, nil, nil, nil, repo)

	if list, err := svc.ListTasks(context.Background()); err != nil || len(list) == 0 {
		t.Fatalf("list tasks: %v", err)
	}
	enabled := false
	interval := 600
	_, err := svc.UpdateTask(context.Background(), "vps_refresh", appshared.ScheduledTaskUpdate{
		Enabled:     &enabled,
		IntervalSec: &interval,
	})
	if err != nil {
		t.Fatalf("update task: %v", err)
	}
}
