package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type OrderService struct {
	orders      OrderRepository
	items       OrderItemRepository
	cart        CartRepository
	catalog     CatalogRepository
	images      SystemImageRepository
	billing     BillingCycleRepository
	vps         VPSRepository
	wallets     WalletRepository
	events      EventPublisher
	automation  AutomationClientResolver
	robot       RobotNotifier
	audit       AuditRepository
	users       UserRepository
	email       EmailSender
	settings    SettingsRepository
	payments    PaymentRepository
	autoLogs    AutomationLogRepository
	provision   ProvisionJobRepository
	resizeTasks ResizeTaskRepository
	messages    *MessageCenterService
	realname    *RealNameService
}

type OrderItemInput struct {
	PackageID int64    `json:"package_id"`
	SystemID  int64    `json:"system_id"`
	Spec      CartSpec `json:"spec"`
	Qty       int      `json:"qty"`
}

type PaymentInput struct {
	Method        string `json:"method"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	TradeNo       string `json:"trade_no"`
	Note          string `json:"note"`
	ScreenshotURL string `json:"screenshot_url"`
}

func NewOrderService(orders OrderRepository, items OrderItemRepository, cart CartRepository, catalog CatalogRepository, images SystemImageRepository, billing BillingCycleRepository, vps VPSRepository, wallets WalletRepository, payments PaymentRepository, events EventPublisher, automation AutomationClientResolver, robot RobotNotifier, audit AuditRepository, users UserRepository, email EmailSender, settings SettingsRepository, autoLogs AutomationLogRepository, provision ProvisionJobRepository, resizeTasks ResizeTaskRepository, messages *MessageCenterService, realname *RealNameService) *OrderService {
	return &OrderService{orders: orders, items: items, cart: cart, catalog: catalog, images: images, billing: billing, vps: vps, wallets: wallets, payments: payments, events: events, automation: automation, robot: robot, audit: audit, users: users, email: email, settings: settings, autoLogs: autoLogs, provision: provision, resizeTasks: resizeTasks, messages: messages, realname: realname}
}

func (s *OrderService) client(ctx context.Context, goodsTypeID int64) (AutomationClient, error) {
	if s.automation == nil {
		return nil, ErrInvalidInput
	}
	return s.automation.ClientForGoodsType(ctx, goodsTypeID)
}

func (s *OrderService) CreateOrderFromCart(ctx context.Context, userID int64, currency string, idemKey string) (domain.Order, []domain.OrderItem, error) {
	if s.realname != nil {
		if err := s.realname.RequireAction(ctx, userID, "purchase_vps"); err != nil {
			return domain.Order{}, nil, err
		}
	}
	if idemKey != "" {
		if existing, err := s.orders.GetOrderByIdempotencyKey(ctx, userID, idemKey); err == nil {
			items, _ := s.items.ListOrderItems(ctx, existing.ID)
			return existing, items, nil
		}
	}
	items, err := s.cart.ListCartItems(ctx, userID)
	if err != nil {
		return domain.Order{}, nil, err
	}
	if len(items) == 0 {
		return domain.Order{}, nil, ErrInvalidInput
	}
	if currency == "" {
		currency = "CNY"
	}
	var total int64
	orderNo := fmt.Sprintf("ORD-%d-%d", userID, time.Now().Unix())
	order := domain.Order{
		UserID:         userID,
		OrderNo:        orderNo,
		Status:         domain.OrderStatusPendingPayment,
		TotalAmount:    total,
		Currency:       currency,
		IdempotencyKey: idemKey,
	}

	var orderItems []domain.OrderItem
	for _, item := range items {
		pkg, err := s.catalog.GetPackage(ctx, item.PackageID)
		if err != nil {
			return domain.Order{}, nil, err
		}
		spec := parseCartSpecJSON(item.SpecJSON)
		if err := normalizeCartSpec(&spec); err != nil {
			return domain.Order{}, nil, err
		}
		unitAmount, months, err := s.priceForPackage(ctx, item.PackageID, spec)
		if err != nil {
			return domain.Order{}, nil, err
		}
		spec.DurationMonths = months
		durationMonths := months
		specJSON := mustJSON(spec)
		qty := item.Qty
		if qty <= 0 {
			qty = 1
		}
		total += unitAmount * int64(qty)
		for i := 0; i < qty; i++ {
			orderItems = append(orderItems, domain.OrderItem{
				OrderID:        order.ID,
				PackageID:      item.PackageID,
				SystemID:       item.SystemID,
				SpecJSON:       specJSON,
				Qty:            1,
				Amount:         unitAmount,
				Status:         domain.OrderItemStatusPendingPayment,
				GoodsTypeID:    pkg.GoodsTypeID,
				Action:         "create",
				DurationMonths: durationMonths,
			})
		}
	}
	order.TotalAmount = total

	type orderFromCartAtomicCreator interface {
		CreateOrderFromCartAtomic(ctx context.Context, order domain.Order, items []domain.OrderItem) (domain.Order, []domain.OrderItem, error)
	}
	if atomic, ok := s.orders.(orderFromCartAtomicCreator); ok {
		createdOrder, createdItems, err := atomic.CreateOrderFromCartAtomic(ctx, order, orderItems)
		if err != nil {
			return domain.Order{}, nil, err
		}
		order = createdOrder
		orderItems = createdItems
	} else {
		if err := s.orders.CreateOrder(ctx, &order); err != nil {
			return domain.Order{}, nil, err
		}
		for i := range orderItems {
			orderItems[i].OrderID = order.ID
		}
		if err := s.items.CreateOrderItems(ctx, orderItems); err != nil {
			return domain.Order{}, nil, err
		}
		if err := s.cart.ClearCart(ctx, userID); err != nil {
			return domain.Order{}, nil, err
		}
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.pending_payment", map[string]any{
			"status": order.Status,
			"total":  total,
		})
	}
	return order, orderItems, nil
}

func (s *OrderService) CreateOrderFromItems(ctx context.Context, userID int64, currency string, inputs []OrderItemInput, idemKey string) (domain.Order, []domain.OrderItem, error) {
	if len(inputs) == 0 {
		return domain.Order{}, nil, ErrInvalidInput
	}
	if s.realname != nil {
		if err := s.realname.RequireAction(ctx, userID, "purchase_vps"); err != nil {
			return domain.Order{}, nil, err
		}
	}
	if idemKey != "" {
		if existing, err := s.orders.GetOrderByIdempotencyKey(ctx, userID, idemKey); err == nil {
			items, _ := s.items.ListOrderItems(ctx, existing.ID)
			return existing, items, nil
		}
	}
	if currency == "" {
		currency = "CNY"
	}
	var total int64
	var orderItems []domain.OrderItem
	for _, in := range inputs {
		if in.PackageID == 0 || in.SystemID == 0 {
			return domain.Order{}, nil, ErrInvalidInput
		}
		if err := normalizeCartSpec(&in.Spec); err != nil {
			return domain.Order{}, nil, err
		}
		unitAmount, months, err := s.priceForPackage(ctx, in.PackageID, in.Spec)
		if err != nil {
			return domain.Order{}, nil, err
		}
		pkg, err := s.catalog.GetPackage(ctx, in.PackageID)
		if err != nil {
			return domain.Order{}, nil, err
		}
		qty := in.Qty
		if qty <= 0 {
			qty = 1
		}
		total += unitAmount * int64(qty)
		in.Spec.DurationMonths = months
		specJSON := mustJSON(in.Spec)
		for i := 0; i < qty; i++ {
			orderItems = append(orderItems, domain.OrderItem{
				OrderID:        0,
				PackageID:      in.PackageID,
				SystemID:       in.SystemID,
				SpecJSON:       specJSON,
				Qty:            1,
				Amount:         unitAmount,
				Status:         domain.OrderItemStatusPendingPayment,
				GoodsTypeID:    pkg.GoodsTypeID,
				Action:         "create",
				DurationMonths: months,
			})
		}
	}
	orderNo := fmt.Sprintf("ORD-%d-%d", userID, time.Now().Unix())
	order := domain.Order{
		UserID:         userID,
		OrderNo:        orderNo,
		Status:         domain.OrderStatusPendingPayment,
		TotalAmount:    total,
		Currency:       currency,
		IdempotencyKey: idemKey,
	}
	if err := s.orders.CreateOrder(ctx, &order); err != nil {
		return domain.Order{}, nil, err
	}
	for i := range orderItems {
		orderItems[i].OrderID = order.ID
	}
	if err := s.items.CreateOrderItems(ctx, orderItems); err != nil {
		return domain.Order{}, nil, err
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.pending_payment", map[string]any{
			"status": order.Status,
			"total":  total,
		})
	}
	return order, orderItems, nil
}

func (s *OrderService) GetOrder(ctx context.Context, orderID int64, userID int64) (domain.Order, []domain.OrderItem, error) {
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return domain.Order{}, nil, err
	}
	if order.UserID != userID {
		return domain.Order{}, nil, ErrForbidden
	}
	items, err := s.items.ListOrderItems(ctx, orderID)
	if err != nil {
		return domain.Order{}, nil, err
	}
	return order, items, nil
}

func (s *OrderService) SubmitPayment(ctx context.Context, userID int64, orderID int64, input PaymentInput, idemKey string) (domain.OrderPayment, error) {
	if s.payments == nil {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	var err error
	input.Method, err = trimAndValidateRequired(input.Method, maxLenPaymentMethod)
	if err != nil {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	input.TradeNo, err = trimAndValidateOptional(input.TradeNo, maxLenPaymentTradeNo)
	if err != nil {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	input.Note, err = trimAndValidateOptional(input.Note, maxLenPaymentNote)
	if err != nil {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	input.ScreenshotURL, err = trimAndValidateOptional(input.ScreenshotURL, maxLenPaymentImageURL)
	if err != nil {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return domain.OrderPayment{}, err
	}
	if order.UserID != userID {
		return domain.OrderPayment{}, ErrForbidden
	}
	if order.Status != domain.OrderStatusPendingPayment && order.Status != domain.OrderStatusPendingReview {
		return domain.OrderPayment{}, ErrConflict
	}
	if order.TotalAmount <= 0 {
		return domain.OrderPayment{}, ErrNoPaymentRequired
	}
	if idemKey != "" {
		if existing, err := s.payments.GetPaymentByIdempotencyKey(ctx, orderID, idemKey); err == nil {
			return existing, nil
		}
	}
	if input.TradeNo != "" {
		if existing, err := s.payments.GetPaymentByTradeNo(ctx, input.TradeNo); err == nil {
			if existing.OrderID != order.ID {
				return domain.OrderPayment{}, ErrConflict
			}
			return existing, nil
		}
	}
	if input.Amount <= 0 {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	if input.Currency == "" {
		input.Currency = order.Currency
	}
	if order.Currency != "" && input.Currency != order.Currency {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	payment := domain.OrderPayment{
		OrderID:        order.ID,
		UserID:         userID,
		Method:         input.Method,
		Amount:         input.Amount,
		Currency:       input.Currency,
		TradeNo:        input.TradeNo,
		Note:           input.Note,
		ScreenshotURL:  input.ScreenshotURL,
		Status:         domain.PaymentStatusPendingReview,
		IdempotencyKey: idemKey,
	}
	if err := s.payments.CreatePayment(ctx, &payment); err != nil {
		return domain.OrderPayment{}, err
	}
	order.Status = domain.OrderStatusPendingReview
	order.PendingReason = ""
	if err := s.orders.UpdateOrderMeta(ctx, order); err != nil {
		return domain.OrderPayment{}, err
	}
	items, _ := s.items.ListOrderItems(ctx, order.ID)
	for _, item := range items {
		_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusPendingReview)
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.pending_review", map[string]any{
			"status": order.Status,
			"total":  order.TotalAmount,
			"payment": map[string]any{
				"method":   payment.Method,
				"amount":   payment.Amount,
				"trade_no": payment.TradeNo,
			},
		})
	}
	if s.robot != nil {
		user, _ := s.users.GetUserByID(ctx, userID)
		payload := RobotOrderPayload{
			OrderNo:    order.OrderNo,
			UserID:     userID,
			Username:   user.Username,
			Email:      user.Email,
			QQ:         user.QQ,
			Amount:     order.TotalAmount,
			Currency:   order.Currency,
			Items:      s.buildRobotItems(ctx, items),
			ApproveURL: fmt.Sprintf("/admin/orders/%d", order.ID),
		}
		_ = s.robot.NotifyOrderPending(ctx, payload)
	}
	return payment, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, userID int64, orderID int64) error {
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.UserID != userID {
		return ErrForbidden
	}
	if order.Status != domain.OrderStatusPendingPayment && order.Status != domain.OrderStatusPendingReview {
		return ErrConflict
	}
	order.Status = domain.OrderStatusCanceled
	if err := s.orders.UpdateOrderMeta(ctx, order); err != nil {
		return err
	}
	items, _ := s.items.ListOrderItems(ctx, order.ID)
	for _, item := range items {
		_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusCanceled)
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.canceled", map[string]any{"status": order.Status})
	}
	return nil
}

func (s *OrderService) MarkPaid(ctx context.Context, adminID int64, orderID int64, input PaymentInput) (domain.OrderPayment, error) {
	if s.payments == nil {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	var err error
	input.Method, err = trimAndValidateRequired(input.Method, maxLenPaymentMethod)
	if err != nil {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	input.TradeNo, err = trimAndValidateOptional(input.TradeNo, maxLenPaymentTradeNo)
	if err != nil {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	input.Note, err = trimAndValidateOptional(input.Note, maxLenPaymentNote)
	if err != nil {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	input.ScreenshotURL, err = trimAndValidateOptional(input.ScreenshotURL, maxLenPaymentImageURL)
	if err != nil {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return domain.OrderPayment{}, err
	}
	if order.Status != domain.OrderStatusPendingPayment {
		return domain.OrderPayment{}, ErrConflict
	}
	if order.TotalAmount <= 0 {
		return domain.OrderPayment{}, ErrNoPaymentRequired
	}
	if input.Amount <= 0 {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	if input.Currency == "" {
		input.Currency = order.Currency
	}
	if order.Currency != "" && input.Currency != order.Currency {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	payment := domain.OrderPayment{
		OrderID:       order.ID,
		UserID:        order.UserID,
		Method:        input.Method,
		Amount:        input.Amount,
		Currency:      input.Currency,
		TradeNo:       input.TradeNo,
		Note:          input.Note,
		ScreenshotURL: input.ScreenshotURL,
		Status:        domain.PaymentStatusPendingReview,
	}
	if err := s.payments.CreatePayment(ctx, &payment); err != nil {
		return domain.OrderPayment{}, err
	}
	order.Status = domain.OrderStatusPendingReview
	if err := s.orders.UpdateOrderMeta(ctx, order); err != nil {
		return domain.OrderPayment{}, err
	}
	items, _ := s.items.ListOrderItems(ctx, order.ID)
	for _, item := range items {
		_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusPendingReview)
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "order.mark_paid", TargetType: "order", TargetID: fmt.Sprintf("%d", order.ID), DetailJSON: mustJSON(map[string]any{"amount": input.Amount})})
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.pending_review", map[string]any{"status": order.Status})
	}
	return payment, nil
}

func (s *OrderService) ListOrders(ctx context.Context, filter OrderFilter, limit, offset int) ([]domain.Order, int, error) {
	return s.orders.ListOrders(ctx, filter, limit, offset)
}

func (s *OrderService) RefreshOrder(ctx context.Context, userID int64, orderID int64) ([]domain.VPSInstance, error) {
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, ErrForbidden
	}
	items, err := s.items.ListOrderItems(ctx, orderID)
	if err != nil {
		return nil, err
	}
	var updated []domain.VPSInstance
	for _, item := range items {
		if item.Action != "create" {
			continue
		}
		itemCtx := WithAutomationLogContext(ctx, order.ID, item.ID)
		inst, err := s.vps.GetInstanceByOrderItem(ctx, item.ID)
		if err != nil {
			continue
		}
		hostID := parseHostID(inst.AutomationInstanceID)
		if hostID == 0 {
			continue
		}
		cli, err := s.client(itemCtx, inst.GoodsTypeID)
		if err != nil {
			continue
		}
		info, err := cli.GetHostInfo(itemCtx, hostID)
		if err != nil {
			continue
		}
		effectiveHostID := hostID
		if info.HostID > 0 {
			effectiveHostID = info.HostID
		}
		if effectiveHostID > 0 && inst.AutomationInstanceID != fmt.Sprintf("%d", effectiveHostID) {
			inst.AutomationInstanceID = fmt.Sprintf("%d", effectiveHostID)
			if info.HostName != "" {
				inst.Name = info.HostName
			}
			_ = s.vps.UpdateInstanceLocal(ctx, inst)
		}
		status := MapAutomationState(info.State)
		_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, status, info.State)
		if info.RemoteIP != "" || info.PanelPassword != "" || info.VNCPassword != "" {
			_ = s.vps.UpdateInstanceAccessInfo(ctx, inst.ID, mustJSON(map[string]any{
				"remote_ip":      info.RemoteIP,
				"panel_password": info.PanelPassword,
				"vnc_password":   info.VNCPassword,
			}))
		}
		if isReadyState(info.State) {
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusActive)
		}
		if inst.AutomationInstanceID != "" && item.AutomationInstanceID != inst.AutomationInstanceID {
			_ = s.items.UpdateOrderItemAutomation(ctx, item.ID, inst.AutomationInstanceID)
		}
		refreshed, _ := s.vps.GetInstance(ctx, inst.ID)
		updated = append(updated, refreshed)
	}
	s.refreshOrderStatus(ctx, order.ID)
	return updated, nil
}

func (s *OrderService) ApproveOrder(ctx context.Context, adminID int64, orderID int64) error {
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != domain.OrderStatusPendingReview &&
		order.Status != domain.OrderStatusPendingPayment &&
		order.Status != domain.OrderStatusRejected {
		return ErrConflict
	}
	now := time.Now()
	order.Status = domain.OrderStatusApproved
	order.ApprovedAt = &now
	if adminID > 0 {
		order.ApprovedBy = &adminID
	}
	order.RejectedReason = ""
	items, _ := s.items.ListOrderItems(ctx, order.ID)
	hasResize := false
	hasNonResize := false
	var resizeTasks []*domain.ResizeTask
	if s.resizeTasks != nil {
		for _, item := range items {
			if item.Action == "resize" {
				hasResize = true
				vpsID, scheduledAt, err := parseResizeTaskSpec(item.SpecJSON)
				if err != nil {
					return err
				}
				if vpsID <= 0 {
					return ErrInvalidInput
				}
				if pending, err := s.resizeTasks.HasPendingResizeTask(ctx, vpsID); err != nil {
					return err
				} else if pending {
					return ErrResizeInProgress
				}
				resizeTasks = append(resizeTasks, &domain.ResizeTask{
					VPSID:       vpsID,
					OrderID:     order.ID,
					OrderItemID: item.ID,
					Status:      domain.ResizeTaskStatusPending,
					ScheduledAt: scheduledAt,
				})
			} else {
				hasNonResize = true
			}
		}
	} else {
		for _, item := range items {
			if item.Action == "resize" {
				hasResize = true
			} else {
				hasNonResize = true
			}
		}
	}

	if hasResize && s.resizeTasks != nil {
		if approver, ok := s.orders.(resizeOrderAtomicApprover); ok {
			if err := approver.ApproveResizeOrderWithTasks(ctx, order, items, resizeTasks); err != nil {
				return err
			}
		} else {
			if err := s.orders.UpdateOrderMeta(ctx, order); err != nil {
				return err
			}
			for _, item := range items {
				_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusApproved)
			}
			for _, task := range resizeTasks {
				if err := s.resizeTasks.CreateResizeTask(ctx, task); err != nil {
					return err
				}
			}
		}
	} else {
		if err := s.orders.UpdateOrderMeta(ctx, order); err != nil {
			return err
		}
		for _, item := range items {
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusApproved)
		}
	}
	for _, item := range items {
		if item.Action != "resize" {
			continue
		}
		if err := s.creditResizeRefundOnApprove(ctx, item); err != nil {
			return err
		}
	}
	if s.payments != nil {
		pays, _ := s.payments.ListPaymentsByOrder(ctx, order.ID)
		for _, pay := range pays {
			_ = s.payments.UpdatePaymentStatus(ctx, pay.ID, domain.PaymentStatusApproved, &adminID, "")
		}
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "order.approve", TargetType: "order", TargetID: fmt.Sprintf("%d", order.ID), DetailJSON: "{}"})
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.approved", map[string]any{"status": domain.OrderStatusApproved})
	}
	s.notifyOrderDecision(ctx, order.UserID, order.OrderNo, "order_approved", "Order Approved: {{.order.no}}", "Your order has been approved.")
	if hasResize && s.resizeTasks != nil {
		now := time.Now()
		for _, task := range resizeTasks {
			if task.ScheduledAt == nil || !task.ScheduledAt.After(now) {
				go s.executeResizeTask(context.Background(), *task)
			}
		}
		if hasNonResize {
			go s.provisionOrder(order.ID)
		}
		return nil
	}
	go s.provisionOrder(order.ID)
	return nil
}

type resizeOrderAtomicApprover interface {
	ApproveResizeOrderWithTasks(ctx context.Context, order domain.Order, items []domain.OrderItem, tasks []*domain.ResizeTask) error
}

func (s *OrderService) RetryProvision(orderID int64) error {
	ctx := context.Background()
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status == domain.OrderStatusFailed && s.items != nil {
		items, err := s.items.ListOrderItems(ctx, order.ID)
		if err != nil {
			return err
		}
		for _, item := range items {
			switch item.Action {
			case "renew", "emergency_renew", "resize", "refund":
				vpsID := parseOrderItemVPSID(item.SpecJSON)
				if vpsID <= 0 {
					continue
				}
				conflict, err := s.items.HasPendingRenewOrder(ctx, order.UserID, vpsID)
				if err != nil {
					return err
				}
				if conflict {
					return ErrConflict
				}
			}
		}
	}
	switch order.Status {
	case domain.OrderStatusApproved, domain.OrderStatusProvisioning, domain.OrderStatusFailed:
		go s.provisionOrder(orderID)
		return nil
	default:
		return ErrConflict
	}
}

func (s *OrderService) RejectOrder(ctx context.Context, adminID int64, orderID int64, reason string) error {
	normalizedReason, err := trimAndValidateOptional(reason, maxLenReviewReason)
	if err != nil {
		return ErrInvalidInput
	}
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != domain.OrderStatusPendingReview &&
		order.Status != domain.OrderStatusPendingPayment &&
		order.Status != domain.OrderStatusRejected {
		return ErrConflict
	}
	order.Status = domain.OrderStatusRejected
	order.RejectedReason = normalizedReason
	order.PendingReason = ""
	if err := s.orders.UpdateOrderMeta(ctx, order); err != nil {
		return err
	}
	items, _ := s.items.ListOrderItems(ctx, order.ID)
	for _, item := range items {
		_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusRejected)
	}
	if s.payments != nil {
		pays, _ := s.payments.ListPaymentsByOrder(ctx, order.ID)
		for _, pay := range pays {
			_ = s.payments.UpdatePaymentStatus(ctx, pay.ID, domain.PaymentStatusRejected, &adminID, normalizedReason)
		}
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "order.reject", TargetType: "order", TargetID: fmt.Sprintf("%d", order.ID), DetailJSON: mustJSON(map[string]any{"reason": normalizedReason})})
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.rejected", map[string]any{"status": domain.OrderStatusRejected, "reason": normalizedReason})
	}
	s.notifyOrderDecision(ctx, order.UserID, order.OrderNo, "order_rejected", "Order Rejected: {{.order.no}}", normalizedReason)
	return nil
}

func (s *OrderService) provisionOrder(orderID int64) {
	ctx := context.Background()
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return
	}
	switch order.Status {
	case domain.OrderStatusApproved, domain.OrderStatusProvisioning, domain.OrderStatusFailed:
	default:
		return
	}
	items, err := s.items.ListOrderItems(ctx, orderID)
	if err != nil {
		return
	}
	order.Status = domain.OrderStatusProvisioning
	_ = s.orders.UpdateOrderMeta(ctx, order)
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.provisioning", map[string]any{"status": domain.OrderStatusProvisioning})
	}

	allActive := true
	anyProvisioning := false
	anyFailed := false
	shouldProcess := func(status domain.OrderItemStatus) bool {
		switch status {
		case domain.OrderItemStatusActive, domain.OrderItemStatusRejected, domain.OrderItemStatusCanceled:
			return false
		default:
			return true
		}
	}
	for _, item := range items {
		if !shouldProcess(item.Status) {
			if item.Status != domain.OrderItemStatusActive {
				allActive = false
			}
			continue
		}
		switch item.Action {
		case "create":
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusProvisioning)
			inst, err := s.provisionItem(ctx, order, item)
			if err != nil {
				allActive = false
				if errors.Is(err, ErrProvisioning) {
					anyProvisioning = true
					continue
				}
				anyFailed = true
				_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusFailed)
				if s.events != nil {
					_, _ = s.events.Publish(ctx, order.ID, "order.item.failed", map[string]any{"item_id": item.ID, "reason": err.Error()})
				}
				continue
			}
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusActive)
			_ = s.items.UpdateOrderItemAutomation(ctx, item.ID, inst.AutomationInstanceID)
			if s.events != nil {
				_, _ = s.events.Publish(ctx, order.ID, "order.item.active", map[string]any{"item_id": item.ID, "instance_id": inst.ID})
			}
		case "renew":
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusProvisioning)
			if err := s.handleRenew(ctx, item); err != nil {
				allActive = false
				anyFailed = true
				_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusFailed)
				if s.events != nil {
					_, _ = s.events.Publish(ctx, order.ID, "order.item.failed", map[string]any{"item_id": item.ID, "reason": err.Error()})
				}
				continue
			}
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusActive)
			if s.events != nil {
				_, _ = s.events.Publish(ctx, order.ID, "order.item.active", map[string]any{"item_id": item.ID})
			}
		case "emergency_renew":
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusProvisioning)
			if err := s.handleEmergencyRenew(ctx, item); err != nil {
				allActive = false
				anyFailed = true
				_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusFailed)
				if s.events != nil {
					_, _ = s.events.Publish(ctx, order.ID, "order.item.failed", map[string]any{"item_id": item.ID, "reason": err.Error()})
				}
				continue
			}
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusActive)
			if s.events != nil {
				_, _ = s.events.Publish(ctx, order.ID, "order.item.active", map[string]any{"item_id": item.ID})
			}
		case "resize":
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusProvisioning)
			if err := s.handleResize(ctx, item); err != nil {
				allActive = false
				anyFailed = true
				_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusFailed)
				if s.events != nil {
					_, _ = s.events.Publish(ctx, order.ID, "order.item.failed", map[string]any{"item_id": item.ID, "reason": err.Error()})
				}
				continue
			}
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusActive)
			if s.events != nil {
				_, _ = s.events.Publish(ctx, order.ID, "order.item.active", map[string]any{"item_id": item.ID})
			}
		case "refund":
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusProvisioning)
			if err := s.handleRefund(ctx, item); err != nil {
				allActive = false
				anyFailed = true
				_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusFailed)
				if s.events != nil {
					_, _ = s.events.Publish(ctx, order.ID, "order.item.failed", map[string]any{"item_id": item.ID, "reason": err.Error()})
				}
				continue
			}
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusActive)
			if s.events != nil {
				_, _ = s.events.Publish(ctx, order.ID, "order.item.active", map[string]any{"item_id": item.ID})
			}
		default:
			continue
		}
	}

	finalStatus := domain.OrderStatusActive
	if anyFailed {
		finalStatus = domain.OrderStatusFailed
	} else if !allActive || anyProvisioning {
		finalStatus = domain.OrderStatusProvisioning
	}
	order.Status = finalStatus
	_ = s.orders.UpdateOrderMeta(ctx, order)
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.completed", map[string]any{"status": finalStatus})
	}
	if finalStatus == domain.OrderStatusActive {
		s.notifyOrderActive(ctx, order.UserID, order.OrderNo)
		if s.messages != nil {
			_ = s.messages.NotifyUser(ctx, order.UserID, "provisioned", "VPS Provisioned", "Order "+order.OrderNo+" has been provisioned.")
		}
	} else if finalStatus == domain.OrderStatusFailed && s.messages != nil {
		_ = s.messages.NotifyUser(ctx, order.UserID, "provision_failed", "Provision Failed", "Order "+order.OrderNo+" failed to provision.")
	}
}

func (s *OrderService) provisionItem(ctx context.Context, order domain.Order, item domain.OrderItem) (domain.VPSInstance, error) {
	ctx = WithAutomationLogContext(ctx, order.ID, item.ID)
	pkg, err := s.catalog.GetPackage(ctx, item.PackageID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	cli, err := s.client(ctx, pkg.GoodsTypeID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	plan, err := s.catalog.GetPlanGroup(ctx, pkg.PlanGroupID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	img, err := s.images.GetSystemImage(ctx, item.SystemID)
	if err != nil {
		return domain.VPSInstance{}, err
	}
	var spec CartSpec
	_ = json.Unmarshal([]byte(item.SpecJSON), &spec)
	cpu := pkg.Cores + spec.AddCores
	mem := pkg.MemoryGB + spec.AddMemGB
	disk := pkg.DiskGB + spec.AddDiskGB
	bw := pkg.BandwidthMB + spec.AddBWMbps

	portNum := pkg.PortNum
	if portNum <= 0 {
		portNum = 30
	}

	hostName := fmt.Sprintf("ecs-%d-%d", order.UserID, time.Now().UnixNano())
	sysPwd := randomPass(10)
	vncPwd := randomPass(8)
	months := item.DurationMonths
	if months <= 0 {
		months = 1
	}
	expireAt := time.Now().AddDate(0, months, 0)
	req := AutomationCreateHostRequest{
		LineID:     plan.LineID,
		OS:         img.Name,
		CPU:        cpu,
		MemoryGB:   mem,
		DiskGB:     disk,
		Bandwidth:  bw,
		PortNum:    portNum,
		ExpireTime: expireAt,
		HostName:   hostName,
		SysPwd:     sysPwd,
		VNCPwd:     vncPwd,
	}
	res, err := cli.CreateHost(ctx, req)
	if err != nil {
		s.logAutomation(ctx, order.ID, item.ID, "create_host", req, map[string]any{"error": err.Error()}, false, err.Error())
		return domain.VPSInstance{}, err
	}
	var hostID int64
	if res.HostID > 0 {
		hostID = res.HostID
	} else {
		hosts, _ := cli.ListHostSimple(ctx, hostName)
		for _, h := range hosts {
			if h.HostName == hostName {
				hostID = h.ID
				break
			}
		}
	}
	if hostID == 0 {
		s.logAutomation(ctx, order.ID, item.ID, "create_host", req, res.Raw, false, "host id not found")
		return domain.VPSInstance{}, fmt.Errorf("host id not found")
	}
	info, err := s.waitHostActive(ctx, cli, hostID, 5, 6*time.Second)
	if err != nil {
		if errors.Is(err, ErrProvisioning) {
			if err := s.ensureProvisioningInstance(ctx, order, item, hostID, hostName, sysPwd, vncPwd, expireAt); err != nil {
				return domain.VPSInstance{}, err
			}
			_ = s.items.UpdateOrderItemAutomation(ctx, item.ID, fmt.Sprintf("%d", hostID))
			s.enqueueProvisionJob(ctx, order.ID, item.ID, hostID, hostName)
			return domain.VPSInstance{}, ErrProvisioning
		}
		s.logAutomation(ctx, order.ID, item.ID, "hostinfo", map[string]any{"host_id": hostID}, map[string]any{"error": err.Error()}, false, err.Error())
		return domain.VPSInstance{}, err
	}
	status := MapAutomationState(info.State)
	accessInfo := mustJSON(map[string]any{
		"remote_ip":      info.RemoteIP,
		"panel_password": info.PanelPassword,
		"vnc_password":   info.VNCPassword,
		"os_password":    sysPwd,
	})
	exp := info.ExpireAt
	if exp == nil {
		exp = &expireAt
	}
	snap := s.buildVPSLocalSnapshot(ctx, item)
	specJSON := setCurrentPeriod(item.SpecJSON, time.Now(), *exp)
	inst := domain.VPSInstance{
		UserID:               order.UserID,
		OrderItemID:          item.ID,
		AutomationInstanceID: fmt.Sprintf("%d", hostID),
		GoodsTypeID:          pkg.GoodsTypeID,
		Name:                 info.HostName,
		Region:               snap.Region,
		RegionID:             snap.RegionID,
		LineID:               snap.LineID,
		PackageID:            snap.PackageID,
		PackageName:          snap.PackageName,
		CPU:                  snap.CPU,
		MemoryGB:             snap.MemoryGB,
		DiskGB:               snap.DiskGB,
		BandwidthMB:          snap.BandwidthMB,
		PortNum:              snap.PortNum,
		MonthlyPrice:         snap.MonthlyPrice,
		SpecJSON:             specJSON,
		SystemID:             item.SystemID,
		Status:               status,
		AutomationState:      info.State,
		AdminStatus:          domain.VPSAdminStatusNormal,
		ExpireAt:             exp,
		AccessInfoJSON:       accessInfo,
	}
	if err := s.vps.CreateInstance(ctx, &inst); err != nil {
		return domain.VPSInstance{}, err
	}
	s.logAutomation(ctx, order.ID, item.ID, "create_host", req, res.Raw, true, "ok")
	return inst, nil
}

func (s *OrderService) handleRenew(ctx context.Context, item domain.OrderItem) error {
	ctx = WithAutomationLogContext(ctx, item.OrderID, item.ID)
	var payload struct {
		VPSID          int64 `json:"vps_id"`
		RenewDays      int   `json:"renew_days"`
		DurationMonths int   `json:"duration_months"`
	}
	if err := json.Unmarshal([]byte(item.SpecJSON), &payload); err != nil {
		return err
	}
	if payload.DurationMonths > 0 {
		payload.RenewDays = payload.DurationMonths * 30
	}
	if payload.RenewDays <= 0 {
		payload.RenewDays = 30
	}
	inst, err := s.vps.GetInstance(ctx, payload.VPSID)
	if err != nil {
		return err
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	next := time.Now().Add(time.Duration(payload.RenewDays) * 24 * time.Hour)
	if inst.ExpireAt != nil && inst.ExpireAt.After(time.Now()) {
		next = inst.ExpireAt.Add(time.Duration(payload.RenewDays) * 24 * time.Hour)
	}
	if err := cli.RenewHost(ctx, hostID, next); err != nil {
		s.logAutomation(ctx, item.OrderID, item.ID, "renew", map[string]any{"host_id": hostID, "next_due": next}, map[string]any{"error": err.Error()}, false, err.Error())
		return err
	}
	if inst.AdminStatus != domain.VPSAdminStatusNormal || inst.Status == domain.VPSStatusExpiredLocked {
		_ = cli.UnlockHost(ctx, hostID)
	}
	_ = s.vps.UpdateInstanceExpireAt(ctx, inst.ID, next)
	_ = s.vps.UpdateInstanceSpec(ctx, inst.ID, setCurrentPeriod(inst.SpecJSON, time.Now(), next))
	s.logAutomation(ctx, item.OrderID, item.ID, "renew", map[string]any{"host_id": hostID, "next_due": next}, map[string]any{"status": "ok"}, true, "ok")
	return nil
}

func (s *OrderService) handleEmergencyRenew(ctx context.Context, item domain.OrderItem) error {
	ctx = WithAutomationLogContext(ctx, item.OrderID, item.ID)
	var payload struct {
		VPSID     int64 `json:"vps_id"`
		RenewDays int   `json:"renew_days"`
	}
	if err := json.Unmarshal([]byte(item.SpecJSON), &payload); err != nil {
		return err
	}
	if payload.VPSID == 0 {
		return ErrInvalidInput
	}
	inst, err := s.vps.GetInstance(ctx, payload.VPSID)
	if err != nil {
		return err
	}
	policy := loadEmergencyRenewPolicy(ctx, s.settings)
	if !policy.Enabled {
		return ErrForbidden
	}
	if !emergencyRenewInWindow(time.Now(), inst.ExpireAt, policy.WindowDays) {
		return ErrForbidden
	}
	if inst.LastEmergencyRenewAt != nil {
		if time.Since(*inst.LastEmergencyRenewAt) < time.Duration(policy.IntervalHours)*time.Hour {
			return ErrConflict
		}
	}
	renewDays := payload.RenewDays
	if renewDays <= 0 {
		renewDays = policy.RenewDays
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	next := time.Now().Add(time.Duration(renewDays) * 24 * time.Hour)
	if inst.ExpireAt != nil && inst.ExpireAt.After(time.Now()) {
		next = inst.ExpireAt.Add(time.Duration(renewDays) * 24 * time.Hour)
	}
	if err := cli.RenewHost(ctx, hostID, next); err != nil {
		s.logAutomation(ctx, item.OrderID, item.ID, "emergency_renew", map[string]any{"host_id": hostID, "next_due": next}, map[string]any{"error": err.Error()}, false, err.Error())
		return err
	}
	if inst.AdminStatus != domain.VPSAdminStatusNormal || inst.Status == domain.VPSStatusExpiredLocked {
		_ = cli.UnlockHost(ctx, hostID)
	}
	_ = s.vps.UpdateInstanceExpireAt(ctx, inst.ID, next)
	_ = s.vps.UpdateInstanceSpec(ctx, inst.ID, setCurrentPeriod(inst.SpecJSON, time.Now(), next))
	_ = s.vps.UpdateInstanceEmergencyRenewAt(ctx, inst.ID, time.Now())
	s.logAutomation(ctx, item.OrderID, item.ID, "emergency_renew", map[string]any{"host_id": hostID, "next_due": next}, map[string]any{"status": "ok"}, true, "ok")
	return nil
}

func (s *OrderService) handleResize(ctx context.Context, item domain.OrderItem) error {
	ctx = WithAutomationLogContext(ctx, item.OrderID, item.ID)
	var payload struct {
		VPSID           int64    `json:"vps_id"`
		Spec            CartSpec `json:"spec"`
		TargetPackageID int64    `json:"target_package_id"`
		TargetCPU       int      `json:"target_cpu"`
		TargetMemGB     int      `json:"target_mem_gb"`
		TargetDiskGB    int      `json:"target_disk_gb"`
		TargetBWMbps    int      `json:"target_bw_mbps"`
		ChargeAmount    int64    `json:"charge_amount"`
		RefundAmount    int64    `json:"refund_amount"`
		RefundToWallet  bool     `json:"refund_to_wallet"`
	}
	if err := json.Unmarshal([]byte(item.SpecJSON), &payload); err != nil {
		return err
	}
	inst, err := s.vps.GetInstance(ctx, payload.VPSID)
	if err != nil {
		return err
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	cpu := payload.TargetCPU
	mem := payload.TargetMemGB
	disk := payload.TargetDiskGB
	bw := payload.TargetBWMbps
	if disk > 0 && inst.DiskGB > 0 && disk < inst.DiskGB {
		return ErrInvalidInput
	}
	req := AutomationElasticUpdateRequest{HostID: hostID}
	if cpu > 0 {
		req.CPU = &cpu
	}
	if mem > 0 {
		req.MemoryGB = &mem
	}
	if disk > 0 {
		req.DiskGB = &disk
	}
	if bw > 0 {
		req.Bandwidth = &bw
	}
	if err := cli.ElasticUpdate(ctx, req); err != nil {
		s.logAutomation(ctx, item.OrderID, item.ID, "elastic_update", req, map[string]any{"error": err.Error()}, false, err.Error())
		return err
	}
	targetPkgID := inst.PackageID
	if payload.TargetPackageID > 0 {
		targetPkgID = payload.TargetPackageID
	}
	if pkg, err := s.catalog.GetPackage(ctx, targetPkgID); err == nil {
		if pkg.GoodsTypeID > 0 && inst.GoodsTypeID > 0 && pkg.GoodsTypeID != inst.GoodsTypeID {
			return ErrInvalidInput
		}
		plan, _ := s.catalog.GetPlanGroup(ctx, pkg.PlanGroupID)
		inst.PackageID = pkg.ID
		inst.PackageName = pkg.Name
		inst.CPU = cpu
		inst.MemoryGB = mem
		inst.DiskGB = disk
		inst.BandwidthMB = bw
		inst.PortNum = pkg.PortNum
		inst.MonthlyPrice = pkg.Monthly
		if plan.ID > 0 {
			inst.MonthlyPrice += int64(payload.Spec.AddCores)*plan.UnitCore +
				int64(payload.Spec.AddMemGB)*plan.UnitMem +
				int64(payload.Spec.AddDiskGB)*plan.UnitDisk +
				int64(payload.Spec.AddBWMbps)*plan.UnitBW
		}
		inst.SpecJSON = mergeSpecJSON(inst.SpecJSON, payload.Spec)
		_ = s.vps.UpdateInstanceLocal(ctx, inst)
	} else {
		_ = s.vps.UpdateInstanceSpec(ctx, inst.ID, mergeSpecJSON(inst.SpecJSON, payload.Spec))
	}
	if payload.RefundAmount > 0 && payload.RefundToWallet && s.wallets != nil {
		refType := "resize_refund"
		exists, err := s.wallets.HasWalletTransaction(ctx, inst.UserID, refType, item.OrderID)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := s.wallets.AdjustWalletBalance(ctx, inst.UserID, payload.RefundAmount, "credit", refType, item.OrderID, fmt.Sprintf("resize refund order %d", item.OrderID)); err != nil {
				return err
			}
		}
	}
	if info, err := cli.GetHostInfo(ctx, hostID); err == nil {
		status := MapAutomationState(info.State)
		_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, status, info.State)
		if info.RemoteIP != "" || info.PanelPassword != "" || info.VNCPassword != "" || info.OSPassword != "" {
			_ = s.vps.UpdateInstanceAccessInfo(ctx, inst.ID, mergeAccessInfo(inst.AccessInfoJSON, info))
		}
		if info.CPU > 0 || info.MemoryGB > 0 || info.DiskGB > 0 || info.Bandwidth > 0 {
			if merged := mergeSpecInfo(inst.SpecJSON, info); merged != "" {
				_ = s.vps.UpdateInstanceSpec(ctx, inst.ID, merged)
			}
		}
	}
	s.logAutomation(ctx, item.OrderID, item.ID, "elastic_update", req, map[string]any{"status": "ok"}, true, "ok")
	return nil
}

func parseResizeTaskSpec(specJSON string) (int64, *time.Time, error) {
	if strings.TrimSpace(specJSON) == "" {
		return 0, nil, ErrInvalidInput
	}
	var payload struct {
		VPSID            int64  `json:"vps_id"`
		ScheduledAt      string `json:"scheduled_at"`
		ScheduledAtUnix  int64  `json:"scheduled_at_unix"`
		ScheduledAtEpoch int64  `json:"scheduled_at_epoch"`
	}
	if err := json.Unmarshal([]byte(specJSON), &payload); err != nil {
		return 0, nil, err
	}
	var scheduledAt *time.Time
	if payload.ScheduledAtUnix > 0 {
		t := time.Unix(payload.ScheduledAtUnix, 0)
		scheduledAt = &t
	} else if payload.ScheduledAtEpoch > 0 {
		t := time.Unix(payload.ScheduledAtEpoch, 0)
		scheduledAt = &t
	} else if strings.TrimSpace(payload.ScheduledAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(payload.ScheduledAt))
		if err != nil {
			return 0, nil, ErrInvalidInput
		}
		scheduledAt = &t
	}
	return payload.VPSID, scheduledAt, nil
}

func (s *OrderService) handleRefund(ctx context.Context, item domain.OrderItem) error {
	var payload struct {
		VPSID           int64  `json:"vps_id"`
		RefundAmount    int64  `json:"refund_amount"`
		RefundToWallet  bool   `json:"refund_to_wallet"`
		DeleteOnApprove bool   `json:"delete_on_approve"`
		Reason          string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(item.SpecJSON), &payload); err != nil {
		return err
	}
	if payload.VPSID == 0 || payload.RefundAmount <= 0 {
		return ErrInvalidInput
	}
	if s.vps == nil || s.wallets == nil {
		return ErrInvalidInput
	}
	inst, err := s.vps.GetInstance(ctx, payload.VPSID)
	if err != nil {
		return err
	}
	meta := map[string]any{
		"vps_id":            inst.ID,
		"order_id":          item.OrderID,
		"order_item_id":     item.ID,
		"refund_amount":     payload.RefundAmount,
		"refund_to_wallet":  true,
		"reason":            strings.TrimSpace(payload.Reason),
		"delete_on_approve": payload.DeleteOnApprove,
	}
	if err := s.createAndApproveWalletRefund(ctx, inst.UserID, payload.RefundAmount, strings.TrimSpace(payload.Reason), meta, "vps_refund", item.OrderID); err != nil {
		return err
	}
	if payload.DeleteOnApprove {
		return s.deleteVPSForRefund(ctx, inst)
	}
	return nil
}

func (s *OrderService) deleteVPSForRefund(ctx context.Context, inst domain.VPSInstance) error {
	if s.vps == nil || s.automation == nil {
		return ErrInvalidInput
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	if err := cli.DeleteHost(ctx, hostID); err != nil {
		return err
	}
	return s.vps.DeleteInstance(ctx, inst.ID)
}

func (s *OrderService) createAndApproveWalletRefund(ctx context.Context, userID int64, amount int64, note string, meta map[string]any, txRefType string, txRefID int64) error {
	if amount <= 0 {
		return ErrInvalidInput
	}
	if s.wallets == nil {
		return ErrInvalidInput
	}
	walletOrders, ok := s.wallets.(WalletOrderRepository)
	if !ok {
		return ErrInvalidInput
	}
	if txRefType != "" && txRefID > 0 {
		exists, err := s.wallets.HasWalletTransaction(ctx, userID, txRefType, txRefID)
		if err != nil {
			return err
		}
		if exists {
			s.promoteRefundWalletOrderIfNeeded(ctx, walletOrders, userID, txRefID)
			return nil
		}
	}
	order := domain.WalletOrder{
		UserID:   userID,
		Type:     domain.WalletOrderRefund,
		Amount:   amount,
		Currency: "CNY",
		Status:   domain.WalletOrderPendingReview,
		Note:     strings.TrimSpace(note),
		MetaJSON: mustJSON(meta),
	}
	if err := walletOrders.CreateWalletOrder(ctx, &order); err != nil {
		return err
	}
	if _, err := s.wallets.AdjustWalletBalance(ctx, userID, amount, "credit", txRefType, txRefID, fmt.Sprintf("refund wallet order %d", order.ID)); err != nil {
		alreadyDone := false
		if txRefType != "" && txRefID > 0 {
			if exists, checkErr := s.wallets.HasWalletTransaction(ctx, userID, txRefType, txRefID); checkErr == nil && exists {
				alreadyDone = true
			}
		}
		if !alreadyDone {
			return err
		}
	}
	if err := walletOrders.UpdateWalletOrderStatus(ctx, order.ID, domain.WalletOrderApproved, nil, ""); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{
			AdminID:    0,
			Action:     "wallet_order.approve",
			TargetType: "wallet_order",
			TargetID:   fmt.Sprintf("%d", order.ID),
			DetailJSON: mustJSON(map[string]any{"type": order.Type, "amount": order.Amount}),
		})
	}
	return nil
}

func (s *OrderService) promoteRefundWalletOrderIfNeeded(ctx context.Context, walletOrders WalletOrderRepository, userID int64, txRefID int64) {
	if walletOrders == nil || txRefID <= 0 {
		return
	}
	orders, _, err := walletOrders.ListWalletOrders(ctx, userID, 50, 0)
	if err != nil {
		return
	}
	for _, candidate := range orders {
		if candidate.Type != domain.WalletOrderRefund || candidate.Status != domain.WalletOrderPendingReview {
			continue
		}
		meta := parseJSON(candidate.MetaJSON)
		if getInt64(meta["order_id"]) != txRefID {
			continue
		}
		_ = walletOrders.UpdateWalletOrderStatus(ctx, candidate.ID, domain.WalletOrderApproved, nil, "")
		return
	}
}

func (s *OrderService) creditResizeRefundOnApprove(ctx context.Context, item domain.OrderItem) error {
	var payload struct {
		VPSID          int64 `json:"vps_id"`
		ChargeAmount   int64 `json:"charge_amount"`
		RefundAmount   int64 `json:"refund_amount"`
		RefundToWallet bool  `json:"refund_to_wallet"`
	}
	if strings.TrimSpace(item.SpecJSON) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(item.SpecJSON), &payload); err != nil {
		return err
	}
	refundAmount := payload.RefundAmount
	if refundAmount <= 0 && payload.ChargeAmount < 0 {
		refundAmount = -payload.ChargeAmount
	}
	if payload.VPSID == 0 || refundAmount <= 0 {
		return nil
	}
	if s.vps == nil || s.wallets == nil {
		return ErrInvalidInput
	}
	inst, err := s.vps.GetInstance(ctx, payload.VPSID)
	if err != nil {
		return err
	}
	meta := map[string]any{
		"vps_id":           inst.ID,
		"order_id":         item.OrderID,
		"order_item_id":    item.ID,
		"refund_amount":    refundAmount,
		"refund_to_wallet": true,
	}
	return s.createAndApproveWalletRefund(ctx, inst.UserID, refundAmount, "resize refund", meta, "resize_refund", item.OrderID)
}

func (s *OrderService) executeResizeTask(ctx context.Context, task domain.ResizeTask) error {
	if s.resizeTasks == nil || s.items == nil || s.vps == nil || s.automation == nil || s.catalog == nil {
		return ErrInvalidInput
	}
	now := time.Now()
	task.Status = domain.ResizeTaskStatusRunning
	task.StartedAt = &now
	if err := s.resizeTasks.UpdateResizeTask(ctx, task); err != nil {
		return err
	}
	item, err := s.items.GetOrderItem(ctx, task.OrderItemID)
	if err != nil {
		task.Status = domain.ResizeTaskStatusFailed
		finished := time.Now()
		task.FinishedAt = &finished
		_ = s.resizeTasks.UpdateResizeTask(ctx, task)
		return err
	}
	_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusProvisioning)
	if err := s.handleResize(ctx, item); err != nil {
		_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusFailed)
		if s.events != nil {
			_, _ = s.events.Publish(ctx, task.OrderID, "order.item.failed", map[string]any{"item_id": item.ID, "reason": err.Error()})
		}
		task.Status = domain.ResizeTaskStatusFailed
		finished := time.Now()
		task.FinishedAt = &finished
		_ = s.resizeTasks.UpdateResizeTask(ctx, task)
		s.refreshOrderStatus(ctx, task.OrderID)
		return err
	}
	_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusActive)
	if s.events != nil {
		_, _ = s.events.Publish(ctx, task.OrderID, "order.item.active", map[string]any{"item_id": item.ID})
	}
	task.Status = domain.ResizeTaskStatusDone
	finished := time.Now()
	task.FinishedAt = &finished
	_ = s.resizeTasks.UpdateResizeTask(ctx, task)
	s.refreshOrderStatus(ctx, task.OrderID)
	return nil
}

func (s *OrderService) buildRobotItems(ctx context.Context, items []domain.OrderItem) []RobotOrderItem {
	var out []RobotOrderItem
	for _, item := range items {
		pkg, _ := s.catalog.GetPackage(ctx, item.PackageID)
		img, _ := s.images.GetSystemImage(ctx, item.SystemID)
		out = append(out, RobotOrderItem{
			PackageName: pkg.Name,
			SystemName:  img.Name,
			SpecJSON:    item.SpecJSON,
			Amount:      item.Amount,
		})
	}
	return out
}

func (s *OrderService) notifyOrderActive(ctx context.Context, userID int64, orderNo string) {
	if s.email == nil || s.settings == nil {
		return
	}
	setting, err := s.settings.GetSetting(ctx, "email_enabled")
	if err != nil || strings.ToLower(setting.ValueJSON) != "true" {
		return
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil || user.Email == "" {
		return
	}
	templates, _ := s.settings.ListEmailTemplates(ctx)
	subject := "Order {{.order.no}} activated"
	body := "Your order {{.order.no}} is active."
	for _, tmpl := range templates {
		if tmpl.Name == "provision_success" && tmpl.Enabled {
			subject = tmpl.Subject
			body = tmpl.Body
			break
		}
	}
	data := map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"qq":       user.QQ,
		},
		"order": map[string]any{
			"no": orderNo,
		},
	}
	subject = RenderTemplate(subject, data, false)
	body = RenderTemplate(body, data, IsHTMLContent(body))
	_ = s.email.Send(ctx, user.Email, subject, body)
}

func (s *OrderService) notifyOrderDecision(ctx context.Context, userID int64, orderNo string, tmplName string, defaultSubject string, message string) {
	if s.email == nil || s.settings == nil {
		return
	}
	setting, err := s.settings.GetSetting(ctx, "email_enabled")
	if err != nil || strings.ToLower(setting.ValueJSON) != "true" {
		return
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil || user.Email == "" {
		return
	}
	templates, _ := s.settings.ListEmailTemplates(ctx)
	subject := defaultSubject
	body := message
	for _, tmpl := range templates {
		if tmpl.Name == tmplName && tmpl.Enabled {
			subject = tmpl.Subject
			body = tmpl.Body
			break
		}
	}
	data := map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"qq":       user.QQ,
		},
		"order": map[string]any{
			"no": orderNo,
		},
		"message": message,
	}
	subject = RenderTemplate(subject, data, false)
	body = RenderTemplate(body, data, IsHTMLContent(body))
	_ = s.email.Send(ctx, user.Email, subject, body)
}

func (s *OrderService) priceForPackage(ctx context.Context, packageID int64, spec CartSpec) (int64, int, error) {
	pkg, err := s.catalog.GetPackage(ctx, packageID)
	if err != nil {
		return 0, 0, err
	}
	plan, err := s.catalog.GetPlanGroup(ctx, pkg.PlanGroupID)
	if err != nil {
		return 0, 0, err
	}
	if err := validateAddonSpec(spec, plan); err != nil {
		return 0, 0, err
	}
	months, multiplier, err := resolveBillingCycle(ctx, s.billing, spec)
	if err != nil {
		return 0, 0, err
	}
	baseMonthly := pkg.Monthly
	addonMonthly := int64(spec.AddCores)*plan.UnitCore + int64(spec.AddMemGB)*plan.UnitMem + int64(spec.AddDiskGB)*plan.UnitDisk + int64(spec.AddBWMbps)*plan.UnitBW
	total := int64(math.Round(float64(baseMonthly+addonMonthly) * multiplier))
	return total, months, nil
}

func resolveBillingCycle(ctx context.Context, billing BillingCycleRepository, spec CartSpec) (int, float64, error) {
	if billing == nil || spec.BillingCycleID == 0 {
		return 1, 1, nil
	}
	cycle, err := billing.GetBillingCycle(ctx, spec.BillingCycleID)
	if err != nil {
		return 0, 0, err
	}
	if !cycle.Active {
		return 0, 0, ErrInvalidInput
	}
	qty := spec.CycleQty
	if qty <= 0 {
		qty = 1
	}
	if cycle.MinQty > 0 && qty < cycle.MinQty {
		return 0, 0, ErrInvalidInput
	}
	if cycle.MaxQty > 0 && qty > cycle.MaxQty {
		return 0, 0, ErrInvalidInput
	}
	months := cycle.Months * qty
	multiplier := cycle.Multiplier * float64(qty)
	return months, multiplier, nil
}

func (s *OrderService) waitHostActive(ctx context.Context, cli AutomationClient, hostID int64, attempts int, interval time.Duration) (AutomationHostInfo, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		info, err := cli.GetHostInfo(ctx, hostID)
		if err != nil {
			lastErr = err
		} else {
			if info.State == 2 || info.State == 3 || info.State == 10 {
				return info, nil
			}
			lastErr = ErrProvisioning
		}
		time.Sleep(interval)
	}
	if lastErr != nil {
		return AutomationHostInfo{}, lastErr
	}
	return AutomationHostInfo{}, ErrProvisioning
}

type vpsLocalSnapshot struct {
	Region       string
	RegionID     int64
	LineID       int64
	PackageID    int64
	PackageName  string
	CPU          int
	MemoryGB     int
	DiskGB       int
	BandwidthMB  int
	PortNum      int
	MonthlyPrice int64
}

func (s *OrderService) buildVPSLocalSnapshot(ctx context.Context, item domain.OrderItem) vpsLocalSnapshot {
	snap := vpsLocalSnapshot{PackageID: item.PackageID}
	pkg, err := s.catalog.GetPackage(ctx, item.PackageID)
	if err != nil {
		return snap
	}
	snap.PackageName = pkg.Name
	snap.CPU = pkg.Cores
	snap.MemoryGB = pkg.MemoryGB
	snap.DiskGB = pkg.DiskGB
	snap.BandwidthMB = pkg.BandwidthMB
	snap.PortNum = pkg.PortNum
	snap.MonthlyPrice = pkg.Monthly
	plan, err := s.catalog.GetPlanGroup(ctx, pkg.PlanGroupID)
	if err == nil {
		snap.LineID = plan.LineID
		snap.RegionID = plan.RegionID
		addon := int64(0)
		var spec CartSpec
		if err := json.Unmarshal([]byte(item.SpecJSON), &spec); err == nil {
			snap.CPU += spec.AddCores
			snap.MemoryGB += spec.AddMemGB
			snap.DiskGB += spec.AddDiskGB
			snap.BandwidthMB += spec.AddBWMbps
			addon = int64(spec.AddCores)*plan.UnitCore +
				int64(spec.AddMemGB)*plan.UnitMem +
				int64(spec.AddDiskGB)*plan.UnitDisk +
				int64(spec.AddBWMbps)*plan.UnitBW
		}
		snap.MonthlyPrice += addon
		if region, err := s.catalog.GetRegion(ctx, plan.RegionID); err == nil {
			snap.Region = region.Name
		}
	}
	return snap
}

func (s *OrderService) ensureProvisioningInstance(ctx context.Context, order domain.Order, item domain.OrderItem, hostID int64, hostName, sysPwd, vncPwd string, expireAt time.Time) error {
	if s.vps == nil {
		return nil
	}
	access := map[string]any{
		"os_password":    sysPwd,
		"vnc_password":   vncPwd,
		"panel_password": vncPwd,
	}
	inst, err := s.vps.GetInstanceByOrderItem(ctx, item.ID)
	if err == nil {
		if hostID > 0 && inst.AutomationInstanceID != fmt.Sprintf("%d", hostID) {
			inst.AutomationInstanceID = fmt.Sprintf("%d", hostID)
			if hostName != "" {
				inst.Name = hostName
			}
			_ = s.vps.UpdateInstanceLocal(ctx, inst)
		}
		_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, domain.VPSStatusProvisioning, 0)
		if inst.ExpireAt == nil {
			_ = s.vps.UpdateInstanceExpireAt(ctx, inst.ID, expireAt)
		}
		if inst.AccessInfoJSON != "" {
			var existing map[string]any
			if err := json.Unmarshal([]byte(inst.AccessInfoJSON), &existing); err == nil {
				if v, ok := existing["remote_ip"]; ok && v != "" {
					access["remote_ip"] = v
				}
				if v, ok := existing["panel_password"]; ok && v != "" {
					access["panel_password"] = v
				}
				if v, ok := existing["vnc_password"]; ok && v != "" {
					access["vnc_password"] = v
				}
				if v, ok := existing["os_password"]; ok && v != "" {
					access["os_password"] = v
				}
			}
		}
		return s.vps.UpdateInstanceAccessInfo(ctx, inst.ID, mustJSON(access))
	}
	if !errors.Is(err, ErrNotFound) {
		return err
	}
	snap := s.buildVPSLocalSnapshot(ctx, item)
	inst = domain.VPSInstance{
		UserID:               order.UserID,
		OrderItemID:          item.ID,
		AutomationInstanceID: fmt.Sprintf("%d", hostID),
		GoodsTypeID:          item.GoodsTypeID,
		Name:                 hostName,
		Region:               snap.Region,
		RegionID:             snap.RegionID,
		LineID:               snap.LineID,
		PackageID:            snap.PackageID,
		PackageName:          snap.PackageName,
		CPU:                  snap.CPU,
		MemoryGB:             snap.MemoryGB,
		DiskGB:               snap.DiskGB,
		BandwidthMB:          snap.BandwidthMB,
		PortNum:              snap.PortNum,
		MonthlyPrice:         snap.MonthlyPrice,
		SpecJSON:             item.SpecJSON,
		SystemID:             item.SystemID,
		Status:               domain.VPSStatusProvisioning,
		AutomationState:      0,
		AdminStatus:          domain.VPSAdminStatusNormal,
		ExpireAt:             &expireAt,
		AccessInfoJSON:       mustJSON(access),
	}
	return s.vps.CreateInstance(ctx, &inst)
}

func (s *OrderService) enqueueProvisionJob(ctx context.Context, orderID, orderItemID, hostID int64, hostName string) {
	if s.provision == nil {
		return
	}
	job := domain.ProvisionJob{
		OrderID:     orderID,
		OrderItemID: orderItemID,
		HostID:      hostID,
		HostName:    hostName,
		Status:      provisionJobStatusPending,
		Attempts:    0,
		NextRunAt:   time.Now(),
	}
	_ = s.provision.CreateOrUpdateProvisionJob(ctx, &job)
}

func (s *OrderService) logAutomation(ctx context.Context, orderID, orderItemID int64, action string, req any, resp any, success bool, message string) {
	if s.autoLogs == nil {
		return
	}
	if !success {
		if traceReq, traceResp, traceAction, traceMsg, ok := extractHTTPTraceFromMessage(message); ok {
			if traceReq != nil {
				req = traceReq
			}
			if traceResp != nil {
				resp = traceResp
			}
			if strings.TrimSpace(traceAction) != "" {
				action = traceAction
			}
			if strings.TrimSpace(traceMsg) != "" {
				message = traceMsg
			}
		}
	}
	requestPayload := map[string]any{
		"method":  "RPC",
		"url":     strings.TrimSpace(action),
		"headers": map[string]string{},
		"body":    req,
	}
	status := 500
	if success {
		status = 200
	}
	responsePayload := map[string]any{
		"status":      status,
		"headers":     map[string]string{},
		"duration_ms": 0,
	}
	if resp != nil {
		responsePayload["body"] = resp
		responsePayload["format"] = "json"
		responsePayload["body_json"] = resp
	} else if strings.TrimSpace(message) != "" {
		responsePayload["body"] = message
		responsePayload["format"] = "text"
	} else {
		responsePayload["body"] = map[string]any{}
		responsePayload["format"] = "json"
	}
	logEntry := domain.AutomationLog{
		OrderID:      orderID,
		OrderItemID:  orderItemID,
		Action:       action,
		RequestJSON:  mustJSON(requestPayload),
		ResponseJSON: mustJSON(responsePayload),
		Success:      success,
		Message:      message,
	}
	_ = s.autoLogs.CreateAutomationLog(ctx, &logEntry)
}

func extractHTTPTraceFromMessage(message string) (map[string]any, map[string]any, string, string, bool) {
	const marker = "http_trace="
	index := strings.LastIndex(message, marker)
	if index < 0 {
		return nil, nil, "", "", false
	}
	raw := strings.TrimSpace(message[index+len(marker):])
	if raw == "" {
		return nil, nil, "", "", false
	}
	var trace struct {
		Action   string         `json:"action"`
		Request  map[string]any `json:"request"`
		Response map[string]any `json:"response"`
		Message  string         `json:"message"`
	}
	if json.Unmarshal([]byte(raw), &trace) != nil {
		return nil, nil, "", "", false
	}
	return trace.Request, trace.Response, trace.Action, trace.Message, true
}

func parseDurationMonths(specJSON string) int {
	if specJSON == "" {
		return 1
	}
	var spec CartSpec
	if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
		return 1
	}
	if spec.DurationMonths > 0 {
		return spec.DurationMonths
	}
	return 1
}

func parseOrderItemVPSID(specJSON string) int64 {
	if strings.TrimSpace(specJSON) == "" {
		return 0
	}
	var payload struct {
		VPSID int64 `json:"vps_id"`
	}
	if err := json.Unmarshal([]byte(specJSON), &payload); err != nil {
		return 0
	}
	return payload.VPSID
}

func randomPass(n int) string {
	letters := []rune("ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%")
	out := make([]rune, n)
	for i := range out {
		out[i] = letters[rand.Intn(len(letters))]
	}
	return string(out)
}

func (s *OrderService) CreateRenewOrder(ctx context.Context, userID int64, vpsID int64, renewDays int, durationMonths int) (domain.Order, error) {
	if s.realname != nil {
		if err := s.realname.RequireAction(ctx, userID, "renew_vps"); err != nil {
			return domain.Order{}, err
		}
	}
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return domain.Order{}, err
	}
	if inst.UserID != userID {
		return domain.Order{}, ErrForbidden
	}
	if s.items != nil {
		if pending, err := s.items.HasPendingRenewOrder(ctx, userID, vpsID); err != nil {
			return domain.Order{}, err
		} else if pending {
			return domain.Order{}, fmt.Errorf("该实例已有待审批或执行中的互斥订单: %w", ErrConflict)
		}
	}
	months := durationMonths
	if months <= 0 {
		if renewDays <= 0 {
			renewDays = 30
		}
		months = int(math.Ceil(float64(renewDays) / 30.0))
	}
	if months <= 0 {
		months = 1
	}
	renewDays = months * 30
	monthlyPrice := inst.MonthlyPrice
	if monthlyPrice <= 0 && inst.PackageID > 0 {
		if pkg, err := s.catalog.GetPackage(ctx, inst.PackageID); err == nil {
			monthlyPrice = pkg.Monthly
		}
	}
	amount := monthlyPrice * int64(months)
	if amount <= 0 {
		return domain.Order{}, ErrInvalidInput
	}
	if renewDays <= 0 {
		renewDays = 30
	}
	orderNo := fmt.Sprintf("REN-%d-%d", userID, time.Now().Unix())
	order := domain.Order{
		UserID:      userID,
		OrderNo:     orderNo,
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: amount,
		Currency:    "CNY",
	}
	if err := s.orders.CreateOrder(ctx, &order); err != nil {
		return domain.Order{}, err
	}
	item := domain.OrderItem{
		OrderID:  order.ID,
		Qty:      1,
		Amount:   amount,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "renew",
		SpecJSON: mustJSON(map[string]any{"vps_id": vpsID, "renew_days": renewDays, "duration_months": months}),
	}
	if err := s.items.CreateOrderItems(ctx, []domain.OrderItem{item}); err != nil {
		return domain.Order{}, err
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.pending_payment", map[string]any{"status": order.Status, "total": amount})
	}
	return order, nil
}

func (s *OrderService) CreateEmergencyRenewOrder(ctx context.Context, userID int64, vpsID int64) (domain.Order, error) {
	if s.realname != nil {
		if err := s.realname.RequireAction(ctx, userID, "renew_vps"); err != nil {
			return domain.Order{}, err
		}
	}
	if s.settings == nil {
		return domain.Order{}, ErrInvalidInput
	}
	policy := loadEmergencyRenewPolicy(ctx, s.settings)
	if !policy.Enabled {
		return domain.Order{}, ErrForbidden
	}
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return domain.Order{}, err
	}
	if inst.UserID != userID {
		return domain.Order{}, ErrForbidden
	}
	if !emergencyRenewInWindow(time.Now(), inst.ExpireAt, policy.WindowDays) {
		return domain.Order{}, ErrForbidden
	}
	if inst.LastEmergencyRenewAt != nil {
		if time.Since(*inst.LastEmergencyRenewAt) < time.Duration(policy.IntervalHours)*time.Hour {
			return domain.Order{}, ErrConflict
		}
	}
	if s.items != nil {
		if pending, err := s.items.HasPendingRenewOrder(ctx, userID, vpsID); err != nil {
			return domain.Order{}, err
		} else if pending {
			return domain.Order{}, ErrConflict
		}
	}
	orderNo := fmt.Sprintf("EMR-%d-%d", userID, time.Now().Unix())
	order := domain.Order{
		UserID:      userID,
		OrderNo:     orderNo,
		Status:      domain.OrderStatusPendingReview,
		TotalAmount: 0,
		Currency:    "CNY",
	}
	if err := s.orders.CreateOrder(ctx, &order); err != nil {
		return domain.Order{}, err
	}
	item := domain.OrderItem{
		OrderID:  order.ID,
		Qty:      1,
		Amount:   0,
		Status:   domain.OrderItemStatusPendingReview,
		Action:   "emergency_renew",
		SpecJSON: mustJSON(map[string]any{"vps_id": vpsID, "renew_days": policy.RenewDays}),
	}
	if err := s.items.CreateOrderItems(ctx, []domain.OrderItem{item}); err != nil {
		return domain.Order{}, err
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.pending_review", map[string]any{"status": order.Status, "total": 0})
	}
	if err := s.ApproveOrder(ctx, 0, order.ID); err != nil {
		return domain.Order{}, err
	}
	updated, err := s.orders.GetOrder(ctx, order.ID)
	if err != nil {
		return order, nil
	}
	return updated, nil
}

func (s *OrderService) CreateResizeOrder(ctx context.Context, userID int64, vpsID int64, spec *CartSpec, targetPackageID int64, resetAddons bool, scheduledAt *time.Time) (domain.Order, ResizeQuote, error) {
	if s.realname != nil {
		if err := s.realname.RequireAction(ctx, userID, "resize_vps"); err != nil {
			return domain.Order{}, ResizeQuote{}, err
		}
	}
	if v, ok := getSettingBool(ctx, s.settings, "resize_enabled"); ok && !v {
		return domain.Order{}, ResizeQuote{}, ErrResizeDisabled
	}
	if scheduledAt != nil {
		if v, ok := getSettingBool(ctx, s.settings, "resize_scheduled_enabled"); ok && !v {
			return domain.Order{}, ResizeQuote{}, ErrForbidden
		}
	}
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return domain.Order{}, ResizeQuote{}, err
	}
	if inst.UserID != userID {
		return domain.Order{}, ResizeQuote{}, ErrForbidden
	}
	if s.items != nil {
		if pending, err := s.items.HasPendingResizeOrder(ctx, userID, vpsID); err != nil {
			return domain.Order{}, ResizeQuote{}, err
		} else if pending {
			return domain.Order{}, ResizeQuote{}, ErrResizeInProgress
		}
	}
	if s.resizeTasks != nil {
		if pending, err := s.resizeTasks.HasPendingResizeTask(ctx, vpsID); err != nil {
			return domain.Order{}, ResizeQuote{}, err
		} else if pending {
			return domain.Order{}, ResizeQuote{}, ErrResizeInProgress
		}
	}
	if inst.ExpireAt != nil && !inst.ExpireAt.After(time.Now()) {
		return domain.Order{}, ResizeQuote{}, ErrForbidden
	}
	quote, targetSpec, err := s.quoteResize(ctx, inst, spec, targetPackageID, resetAddons)
	if err != nil {
		return domain.Order{}, ResizeQuote{}, err
	}
	amount := quote.ChargeAmount
	orderNo := fmt.Sprintf("UPG-%d-%d", userID, time.Now().Unix())
	status := domain.OrderStatusPendingPayment
	itemStatus := domain.OrderItemStatusPendingPayment
	if amount <= 0 {
		status = domain.OrderStatusPendingReview
		itemStatus = domain.OrderItemStatusPendingReview
	}
	order := domain.Order{
		UserID:      userID,
		OrderNo:     orderNo,
		Status:      status,
		TotalAmount: amount,
		Currency:    "CNY",
	}
	if err := s.orders.CreateOrder(ctx, &order); err != nil {
		return domain.Order{}, ResizeQuote{}, err
	}
	specPayload := quote.ToPayload(vpsID, targetSpec)
	if scheduledAt != nil && !scheduledAt.IsZero() {
		specPayload["scheduled_at"] = scheduledAt.UTC().Format(time.RFC3339)
	}
	item := domain.OrderItem{
		OrderID:  order.ID,
		Qty:      1,
		Amount:   amount,
		Status:   itemStatus,
		Action:   "resize",
		SpecJSON: mustJSON(specPayload),
	}
	if err := s.items.CreateOrderItems(ctx, []domain.OrderItem{item}); err != nil {
		return domain.Order{}, ResizeQuote{}, err
	}
	if s.events != nil {
		eventName := "order.pending_payment"
		if status == domain.OrderStatusPendingReview {
			eventName = "order.pending_review"
		}
		_, _ = s.events.Publish(ctx, order.ID, eventName, map[string]any{"status": order.Status, "total": amount})
	}
	if amount <= 0 {
		if err := s.ApproveOrder(ctx, 0, order.ID); err != nil {
			return domain.Order{}, ResizeQuote{}, err
		}
		if updated, err := s.orders.GetOrder(ctx, order.ID); err == nil {
			order = updated
		}
	}
	return order, quote, nil
}

func (s *OrderService) CreateRefundOrder(ctx context.Context, userID int64, vpsID int64, reason string) (domain.Order, int64, error) {
	if userID == 0 || vpsID == 0 {
		return domain.Order{}, 0, ErrInvalidInput
	}
	reason, err := trimAndValidateRequired(reason, maxLenRefundReason)
	if err != nil {
		return domain.Order{}, 0, ErrInvalidInput
	}
	if s.vps == nil || s.items == nil || s.orders == nil || s.wallets == nil {
		return domain.Order{}, 0, ErrInvalidInput
	}
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return domain.Order{}, 0, err
	}
	if inst.UserID != userID {
		return domain.Order{}, 0, ErrForbidden
	}
	if pending, err := s.items.HasPendingResizeOrder(ctx, userID, vpsID); err != nil {
		return domain.Order{}, 0, err
	} else if pending {
		return domain.Order{}, 0, ErrConflict
	}
	if s.resizeTasks != nil {
		if pending, err := s.resizeTasks.HasPendingResizeTask(ctx, vpsID); err != nil {
			return domain.Order{}, 0, err
		} else if pending {
			return domain.Order{}, 0, ErrConflict
		}
	}
	item, err := s.items.GetOrderItem(ctx, inst.OrderItemID)
	if err != nil {
		return domain.Order{}, 0, err
	}
	policy := loadRefundPolicy(ctx, s.settings)
	baseAmount := inst.MonthlyPrice
	if baseAmount <= 0 {
		baseAmount = item.Amount
	}
	amount := calculateRefundAmountForAmount(inst, baseAmount, policy)
	if amount <= 0 {
		return domain.Order{}, 0, ErrForbidden
	}
	orderNo := fmt.Sprintf("REF-%d-%d", userID, time.Now().Unix())
	order := domain.Order{
		UserID:         userID,
		OrderNo:        orderNo,
		Status:         domain.OrderStatusPendingReview,
		TotalAmount:    -amount,
		Currency:       "CNY",
		PendingReason:  strings.TrimSpace(reason),
		RejectedReason: "",
	}
	if err := s.orders.CreateOrder(ctx, &order); err != nil {
		return domain.Order{}, 0, err
	}
	specPayload := map[string]any{
		"vps_id":            inst.ID,
		"refund_amount":     amount,
		"refund_to_wallet":  true,
		"reason":            strings.TrimSpace(reason),
		"delete_on_approve": true,
	}
	refundItem := domain.OrderItem{
		OrderID:  order.ID,
		Qty:      1,
		Amount:   -amount,
		Status:   domain.OrderItemStatusPendingReview,
		Action:   "refund",
		SpecJSON: mustJSON(specPayload),
	}
	if err := s.items.CreateOrderItems(ctx, []domain.OrderItem{refundItem}); err != nil {
		// Keep refund creation atomic at service level: if item creation fails,
		// remove the just-created order to avoid orphan orders.
		if delErr := s.orders.DeleteOrder(ctx, order.ID); delErr != nil {
			return domain.Order{}, 0, fmt.Errorf("create refund item failed: %w; rollback order failed: %v", err, delErr)
		}
		return domain.Order{}, 0, err
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.pending_review", map[string]any{"status": order.Status, "total": order.TotalAmount})
	}
	return order, amount, nil
}

func setCurrentPeriod(specJSON string, start, end time.Time) string {
	payload := map[string]any{}
	if strings.TrimSpace(specJSON) != "" {
		_ = json.Unmarshal([]byte(specJSON), &payload)
	}
	payload["current_period_start"] = start.UTC().Format(time.RFC3339)
	payload["current_period_end"] = end.UTC().Format(time.RFC3339)
	return mustJSON(payload)
}

func mergeSpecJSON(existing string, spec CartSpec) string {
	payload := map[string]any{}
	if strings.TrimSpace(existing) != "" {
		_ = json.Unmarshal([]byte(existing), &payload)
	}
	payload["add_cores"] = spec.AddCores
	payload["add_mem_gb"] = spec.AddMemGB
	payload["add_disk_gb"] = spec.AddDiskGB
	payload["add_bw_mbps"] = spec.AddBWMbps
	if spec.DurationMonths > 0 {
		payload["duration_months"] = spec.DurationMonths
	}
	return mustJSON(payload)
}

func (s *OrderService) QuoteResizeOrder(ctx context.Context, userID int64, vpsID int64, spec *CartSpec, targetPackageID int64, resetAddons bool) (ResizeQuote, CartSpec, error) {
	if s.realname != nil {
		if err := s.realname.RequireAction(ctx, userID, "resize_vps"); err != nil {
			return ResizeQuote{}, CartSpec{}, err
		}
	}
	if v, ok := getSettingBool(ctx, s.settings, "resize_enabled"); ok && !v {
		return ResizeQuote{}, CartSpec{}, ErrResizeDisabled
	}
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return ResizeQuote{}, CartSpec{}, err
	}
	if inst.UserID != userID {
		return ResizeQuote{}, CartSpec{}, ErrForbidden
	}
	if s.items != nil {
		if pending, err := s.items.HasPendingResizeOrder(ctx, userID, vpsID); err != nil {
			return ResizeQuote{}, CartSpec{}, err
		} else if pending {
			return ResizeQuote{}, CartSpec{}, ErrResizeInProgress
		}
	}
	if s.resizeTasks != nil {
		if pending, err := s.resizeTasks.HasPendingResizeTask(ctx, vpsID); err != nil {
			return ResizeQuote{}, CartSpec{}, err
		} else if pending {
			return ResizeQuote{}, CartSpec{}, ErrResizeInProgress
		}
	}
	if inst.ExpireAt != nil && !inst.ExpireAt.After(time.Now()) {
		return ResizeQuote{}, CartSpec{}, ErrForbidden
	}
	return s.quoteResize(ctx, inst, spec, targetPackageID, resetAddons)
}
