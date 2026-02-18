package http_test

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_RobotApproveReject(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)

	user := domain.User{Username: "robotu", Email: "robotu@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := env.Repo.CreateUser(context.Background(), &user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	orderApprove := domain.Order{UserID: user.ID, OrderNo: "O-RA", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "USD"}
	if err := env.Repo.CreateOrder(context.Background(), &orderApprove); err != nil {
		t.Fatalf("create order approve: %v", err)
	}
	items := []domain.OrderItem{{OrderID: orderApprove.ID, SpecJSON: "{}", Qty: 1, Amount: 1000, Status: domain.OrderItemStatusPendingReview, Action: "create", DurationMonths: 1}}
	if err := env.Repo.CreateOrderItems(context.Background(), items); err != nil {
		t.Fatalf("create order items: %v", err)
	}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/integrations/robot/approve", nil)
	ctx.Params = gin.Params{gin.Param{Key: "id", Value: strconv.FormatInt(orderApprove.ID, 10)}}
	env.Handler.RobotApprove(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("robot approve: %d", rec.Code)
	}

	orderReject := domain.Order{UserID: user.ID, OrderNo: "O-RR", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "USD"}
	if err := env.Repo.CreateOrder(context.Background(), &orderReject); err != nil {
		t.Fatalf("create order reject: %v", err)
	}

	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/integrations/robot/reject", bytes.NewBufferString(`{"reason":"no"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{gin.Param{Key: "id", Value: strconv.FormatInt(orderReject.ID, 10)}}
	env.Handler.RobotReject(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("robot reject: %d", rec.Code)
	}
}
