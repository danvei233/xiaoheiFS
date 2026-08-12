package order

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"strings"
	"time"

	appcoupon "xiaoheiplay/internal/app/coupon"
	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/cryptox"
	"xiaoheiplay/internal/pkg/safego"
)

const (
	defaultPortNum             = 30
	systemPasswordLength       = 10
	vncPasswordLength          = 8
	defaultOrderRenewDays      = 30
	defaultWalletOrderPageSize = 50
)

type OrderService struct {
	orders           OrderRepository
	items            OrderItemRepository
	cart             CartRepository
	catalog          CatalogRepository
	images           SystemImageRepository
	billing          BillingCycleRepository
	vps              VPSRepository
	wallets          WalletRepository
	events           EventPublisher
	automation       AutomationClientResolver
	robot            RobotNotifier
	audit            AuditRepository
	users            UserRepository
	email            EmailSender
	settings         SettingsRepository
	capabilityPolicy appports.CapabilityPolicyRepository
	payments         PaymentRepository
	autoLogs         AutomationLogRepository
	provision        ProvisionJobRepository
	resizeTasks      ResizeTaskRepository
	cleanup          appports.RefundCleanupRepository
	messages         messageNotifier
	realname         realNameActionChecker
	pricer           userTierPricingResolver
	userTiers        userTierAutoApprover
	coupon           couponEngine

	// workerID uniquely identifies this service instance so job leases are owned
	// and fenced per process (multi-instance deployments never double-consume).
	workerID string

	// provisionCipher encrypts the provision intent credentials at rest. Nil
	// means credentials are kept in memory only (stage/host recovery still
	// works, but the passwords are not durable across a crash).
	provisionCipher *cryptox.AESGCM
}

type messageNotifier interface {
	NotifyUser(ctx context.Context, userID int64, typ, title, content string) error
}

// vpsLifecycleLockPort is the explicit operation reservation repository that
// guarantees at most one exclusive lifecycle operation per VPS via a unique
// index (CAS). It is optional: the real repository implements it, and test
// fakes that do not simply skip reservation.
type vpsLifecycleLockPort interface {
	AcquireVPSLifecycleLock(ctx context.Context, vpsID, orderID int64, action string) error
	BindVPSLifecycleLock(ctx context.Context, vpsID, orderID int64) error
	ReleaseVPSLifecycleLockByVPS(ctx context.Context, vpsID int64) error
	ReleaseVPSLifecycleLocksForOrder(ctx context.Context, orderID int64) error
}

func (s *OrderService) vpsLocks() vpsLifecycleLockPort {
	if l, ok := s.orders.(vpsLifecycleLockPort); ok {
		return l
	}
	return nil
}

// reserveVPSLifecycleLock atomically reserves a VPS for one exclusive lifecycle
// operation. It is the TOCTOU backstop: two concurrent refund/renew/resize
// requests for the same VPS cannot both pass a check-then-insert because the
// unique index makes the second reservation fail.
func (s *OrderService) reserveVPSLifecycleLock(ctx context.Context, vpsID, orderID int64, action string) error {
	l := s.vpsLocks()
	if l == nil {
		return nil
	}
	return l.AcquireVPSLifecycleLock(ctx, vpsID, orderID, action)
}

// releaseVPSLifecycleLocks releases reservations held by an order reaching a
// terminal state, atomically freeing the VPS for the next lifecycle operation.
func (s *OrderService) releaseVPSLifecycleLocks(ctx context.Context, orderID int64) {
	if l := s.vpsLocks(); l != nil {
		_ = l.ReleaseVPSLifecycleLocksForOrder(ctx, orderID)
	}
}

type realNameActionChecker interface {
	RequireAction(ctx context.Context, userID int64, action string) error
}

type OrderItemInput = appshared.OrderItemInput

type PaymentInput = appshared.PaymentInput

type CouponPreview struct {
	CouponCode string
	Original   int64
	Discount   int64
	Final      int64
}

func NewOrderService(orders OrderRepository, items OrderItemRepository, cart CartRepository, catalog CatalogRepository, images SystemImageRepository, billing BillingCycleRepository, vps VPSRepository, wallets WalletRepository, payments PaymentRepository, events EventPublisher, automation AutomationClientResolver, robot RobotNotifier, audit AuditRepository, users UserRepository, email EmailSender, settings SettingsRepository, autoLogs AutomationLogRepository, provision ProvisionJobRepository, resizeTasks ResizeTaskRepository, messages messageNotifier, realname realNameActionChecker) *OrderService {
	service := &OrderService{
		orders: orders, items: items, cart: cart, catalog: catalog, images: images, billing: billing,
		vps: vps, wallets: wallets, payments: payments, events: events, automation: automation, robot: robot,
		audit: audit, users: users, email: email, settings: settings, autoLogs: autoLogs, provision: provision,
		resizeTasks: resizeTasks, messages: messages, realname: realname,
		workerID: fmt.Sprintf("order-svc-%d", time.Now().UnixNano()),
	}
	service.capabilityPolicy, _ = settings.(appports.CapabilityPolicyRepository)
	return service
}

type userTierPricingResolver interface {
	ResolvePackagePricing(ctx context.Context, userID, packageID int64) (domain.UserTierPriceCache, int64, error)
}

type userTierAutoApprover interface {
	TryAutoApproveForUser(ctx context.Context, userID int64, reason string) error
}

type couponEngine interface {
	PreviewDiscount(ctx context.Context, userID int64, code string, items []appcoupon.QuoteItem) (appcoupon.ApplyResult, error)
	CreateRedemption(ctx context.Context, redemption *domain.CouponRedemption) error
	MarkOrderCanceled(ctx context.Context, orderID int64) error
	MarkOrderConfirmed(ctx context.Context, orderID int64) error
}

func (s *OrderService) SetUserTierPricingResolver(resolver userTierPricingResolver) {
	s.pricer = resolver
}

func (s *OrderService) SetUserTierAutoApprover(approver userTierAutoApprover) {
	s.userTiers = approver
}

func (s *OrderService) SetCouponService(coupon couponEngine) {
	s.coupon = coupon
}

func (s *OrderService) client(ctx context.Context, goodsTypeID int64) (AutomationClient, error) {
	if s.automation == nil {
		return nil, ErrInvalidInput
	}
	return s.automation.ClientForGoodsType(ctx, goodsTypeID)
}

// replayExistingOrder resolves an already-created order for an idempotency key.
// It is used after an insert fails on the unique index because a concurrent
// request with the same key won the race, so the loser replays the winner's
// result instead of surfacing a raw DB error.
func (s *OrderService) replayExistingOrder(ctx context.Context, userID int64, idemKey string) (domain.Order, []domain.OrderItem, bool) {
	if idemKey == "" {
		return domain.Order{}, nil, false
	}
	existing, err := s.orders.GetOrderByIdempotencyKey(ctx, userID, idemKey)
	if err != nil {
		return domain.Order{}, nil, false
	}
	items, _ := s.items.ListOrderItems(ctx, existing.ID)
	return existing, items, true
}

// restoreCart re-adds the user's cart items after a failed coupon redemption.
// The atomic cart->order path clears the cart before the redemption record is
// written, so a redemption failure would otherwise lose the user's cart.
func restoreCart(ctx context.Context, cart CartRepository, items []domain.CartItem) {
	if cart == nil {
		return
	}
	for i := range items {
		item := items[i]
		item.ID = 0
		item.CreatedAt = time.Time{}
		item.UpdatedAt = time.Time{}
		_ = cart.AddCartItem(ctx, &item)
	}
}

// normalizeOrderQty validates an order-line quantity, defaulting non-positive
// values to 1 and rejecting quantities above maxOrderItemQty so a single line
// cannot trigger an unbounded item expansion.
func normalizeOrderQty(qty int) (int, error) {
	if qty <= 0 {
		qty = 1
	}
	if qty > maxOrderItemQty {
		return 0, ErrInvalidInput
	}
	return qty, nil
}

// addLineTotal adds unitTotal*qty to the running order total, rejecting qty
// beyond maxOrderItemQty and any int64 overflow so an attacker cannot wrap the
// total to a negative or tiny amount.
func addLineTotal(total, unitTotal int64, qty int) (int64, error) {
	normalized, err := normalizeOrderQty(qty)
	if err != nil {
		return 0, err
	}
	if unitTotal < 0 {
		return 0, ErrInvalidInput
	}
	if unitTotal == 0 {
		return total, nil
	}
	if unitTotal > math.MaxInt64/int64(normalized) {
		return 0, ErrInvalidInput
	}
	product := unitTotal * int64(normalized)
	if total > math.MaxInt64-product {
		return 0, ErrInvalidInput
	}
	return total + product, nil
}

func (s *OrderService) CreateOrderFromCart(ctx context.Context, userID int64, currency string, idemKey string, couponCode string) (domain.Order, []domain.OrderItem, error) {
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
	orderNo := fmt.Sprintf("ORD-%d-%d", userID, time.Now().UnixNano())
	order := domain.Order{
		UserID:         userID,
		OrderNo:        orderNo,
		Source:         resolveOrderSource(ctx),
		Status:         domain.OrderStatusPendingPayment,
		TotalAmount:    0,
		Currency:       currency,
		IdempotencyKey: idemKey,
	}
	couponCode = strings.ToUpper(strings.TrimSpace(couponCode))
	quotes := make([]appcoupon.QuoteItem, 0, len(items))
	metas := make([]struct {
		PackageID int64
		SystemID  int64
		SpecJSON  string
		Months    int
		Qty       int
		UnitTotal int64
	}, 0, len(items))
	var total int64
	for _, item := range items {
		pkg, err := s.catalog.GetPackage(ctx, item.PackageID)
		if err != nil {
			return domain.Order{}, nil, err
		}
		plan, err := s.catalog.GetPlanGroup(ctx, pkg.PlanGroupID)
		if err != nil {
			return domain.Order{}, nil, err
		}
		spec := parseCartSpecJSON(item.SpecJSON)
		if err := normalizeCartSpec(&spec); err != nil {
			return domain.Order{}, nil, err
		}
		unitTotal, unitBase, addonCore, addonMem, addonDisk, addonBW, months, err := s.priceBreakdownForPackage(ctx, userID, pkg, plan, spec)
		if err != nil {
			return domain.Order{}, nil, err
		}
		spec.DurationMonths = months
		specJSON := mustJSON(spec)
		qty, err := normalizeOrderQty(item.Qty)
		if err != nil {
			return domain.Order{}, nil, err
		}
		metas = append(metas, struct {
			PackageID int64
			SystemID  int64
			SpecJSON  string
			Months    int
			Qty       int
			UnitTotal int64
		}{
			PackageID: item.PackageID,
			SystemID:  item.SystemID,
			SpecJSON:  specJSON,
			Months:    months,
			Qty:       qty,
			UnitTotal: unitTotal,
		})
		quotes = append(quotes, appcoupon.QuoteItem{
			PackageID:       pkg.ID,
			GoodsTypeID:     pkg.GoodsTypeID,
			RegionID:        plan.RegionID,
			PlanGroupID:     plan.ID,
			AddonCore:       spec.AddCores,
			AddonMemGB:      spec.AddMemGB,
			AddonDiskGB:     spec.AddDiskGB,
			AddonBWMbps:     spec.AddBWMbps,
			UnitBaseAmount:  unitBase,
			UnitAddonAmount: addonCore + addonMem + addonDisk + addonBW,
			UnitAddonCore:   addonCore,
			UnitAddonMem:    addonMem,
			UnitAddonDisk:   addonDisk,
			UnitAddonBW:     addonBW,
			UnitTotalAmount: unitTotal,
			Qty:             qty,
		})
		total, err = addLineTotal(total, unitTotal, qty)
		if err != nil {
			return domain.Order{}, nil, err
		}
	}
	order.TotalAmount = total
	var couponResult *appcoupon.ApplyResult
	if couponCode != "" {
		if s.coupon == nil {
			return domain.Order{}, nil, ErrInvalidInput
		}
		res, err := s.coupon.PreviewDiscount(ctx, userID, couponCode, quotes)
		if err != nil {
			return domain.Order{}, nil, err
		}
		couponResult = &res
		order.CouponCode = res.Coupon.Code
		order.CouponID = &res.Coupon.ID
		order.CouponDiscount = res.TotalDiscount
		if order.CouponDiscount > order.TotalAmount {
			order.CouponDiscount = order.TotalAmount
		}
		order.TotalAmount -= order.CouponDiscount
	}
	var orderItems []domain.OrderItem
	totalItems := 0
	for idx, meta := range metas {
		unitAmount := meta.UnitTotal
		if couponResult != nil && idx < len(couponResult.UnitDiscount) {
			unitAmount -= couponResult.UnitDiscount[idx]
			if unitAmount < 0 {
				unitAmount = 0
			}
		}
		totalItems += meta.Qty
		if totalItems > maxOrderTotalItems {
			return domain.Order{}, nil, ErrInvalidInput
		}
		for i := 0; i < meta.Qty; i++ {
			orderItems = append(orderItems, domain.OrderItem{
				OrderID:        order.ID,
				PackageID:      meta.PackageID,
				SystemID:       meta.SystemID,
				SpecJSON:       meta.SpecJSON,
				Qty:            1,
				Amount:         unitAmount,
				Status:         domain.OrderItemStatusPendingPayment,
				GoodsTypeID:    quotes[idx].GoodsTypeID,
				Action:         "create",
				DurationMonths: meta.Months,
			})
		}
	}

	type orderFromCartAtomicCreator interface {
		CreateOrderFromCartAtomic(ctx context.Context, order domain.Order, items []domain.OrderItem) (domain.Order, []domain.OrderItem, error)
	}
	if atomic, ok := s.orders.(orderFromCartAtomicCreator); ok {
		createdOrder, createdItems, err := atomic.CreateOrderFromCartAtomic(ctx, order, orderItems)
		if err != nil {
			if existing, replayItems, replayOK := s.replayExistingOrder(ctx, userID, idemKey); replayOK {
				return existing, replayItems, nil
			}
			return domain.Order{}, nil, err
		}
		order = createdOrder
		orderItems = createdItems
	} else {
		if err := s.orders.CreateOrder(ctx, &order); err != nil {
			if existing, replayItems, replayOK := s.replayExistingOrder(ctx, userID, idemKey); replayOK {
				return existing, replayItems, nil
			}
			return domain.Order{}, nil, err
		}
		for i := range orderItems {
			orderItems[i].OrderID = order.ID
		}
		if err := s.items.CreateOrderItems(ctx, orderItems); err != nil {
			_ = s.orders.DeleteOrder(ctx, order.ID)
			return domain.Order{}, nil, err
		}
		if err := s.cart.ClearCart(ctx, userID); err != nil {
			// Keep the non-atomic fallback transactional at the service level:
			// order + items + cart-clearing must live or die together.
			_ = s.orders.DeleteOrder(ctx, order.ID)
			return domain.Order{}, nil, err
		}
	}
	if couponResult != nil && order.CouponID != nil && s.coupon != nil {
		if err := s.coupon.CreateRedemption(ctx, &domain.CouponRedemption{
			CouponID:       *order.CouponID,
			OrderID:        order.ID,
			UserID:         order.UserID,
			Status:         domain.CouponRedemptionStatusApplied,
			DiscountAmount: order.CouponDiscount,
		}); err != nil {
			_ = s.orders.DeleteOrder(ctx, order.ID)
			// The cart was already cleared (atomically in the common path), so
			// restore it to avoid destroying the user's pending order contents
			// when the coupon redemption fails after order creation.
			restoreCart(ctx, s.cart, items)
			return domain.Order{}, nil, err
		}
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.pending_payment", map[string]any{
			"status": order.Status,
			"total":  order.TotalAmount,
		})
	}
	return order, orderItems, nil
}

func (s *OrderService) CreateOrderFromItems(ctx context.Context, userID int64, currency string, inputs []OrderItemInput, idemKey string, couponCode string) (domain.Order, []domain.OrderItem, error) {
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
	quotes := make([]appcoupon.QuoteItem, 0, len(inputs))
	metas := make([]struct {
		PackageID int64
		SystemID  int64
		SpecJSON  string
		Months    int
		Qty       int
		UnitTotal int64
	}, 0, len(inputs))
	for _, in := range inputs {
		if in.PackageID == 0 || in.SystemID == 0 {
			return domain.Order{}, nil, ErrInvalidInput
		}
		if err := normalizeCartSpec(&in.Spec); err != nil {
			return domain.Order{}, nil, err
		}
		pkg, err := s.catalog.GetPackage(ctx, in.PackageID)
		if err != nil {
			return domain.Order{}, nil, err
		}
		plan, err := s.catalog.GetPlanGroup(ctx, pkg.PlanGroupID)
		if err != nil {
			return domain.Order{}, nil, err
		}
		unitTotal, unitBase, addonCore, addonMem, addonDisk, addonBW, months, err := s.priceBreakdownForPackage(ctx, userID, pkg, plan, in.Spec)
		if err != nil {
			return domain.Order{}, nil, err
		}
		qty, err := normalizeOrderQty(in.Qty)
		if err != nil {
			return domain.Order{}, nil, err
		}
		in.Spec.DurationMonths = months
		specJSON := mustJSON(in.Spec)
		metas = append(metas, struct {
			PackageID int64
			SystemID  int64
			SpecJSON  string
			Months    int
			Qty       int
			UnitTotal int64
		}{
			PackageID: in.PackageID,
			SystemID:  in.SystemID,
			SpecJSON:  specJSON,
			Months:    months,
			Qty:       qty,
			UnitTotal: unitTotal,
		})
		quotes = append(quotes, appcoupon.QuoteItem{
			PackageID:       pkg.ID,
			GoodsTypeID:     pkg.GoodsTypeID,
			RegionID:        plan.RegionID,
			PlanGroupID:     plan.ID,
			AddonCore:       in.Spec.AddCores,
			AddonMemGB:      in.Spec.AddMemGB,
			AddonDiskGB:     in.Spec.AddDiskGB,
			AddonBWMbps:     in.Spec.AddBWMbps,
			UnitBaseAmount:  unitBase,
			UnitAddonAmount: addonCore + addonMem + addonDisk + addonBW,
			UnitAddonCore:   addonCore,
			UnitAddonMem:    addonMem,
			UnitAddonDisk:   addonDisk,
			UnitAddonBW:     addonBW,
			UnitTotalAmount: unitTotal,
			Qty:             qty,
		})
		total, err = addLineTotal(total, unitTotal, qty)
		if err != nil {
			return domain.Order{}, nil, err
		}
	}
	couponCode = strings.ToUpper(strings.TrimSpace(couponCode))
	var couponResult *appcoupon.ApplyResult
	if couponCode != "" {
		if s.coupon == nil {
			return domain.Order{}, nil, ErrInvalidInput
		}
		res, err := s.coupon.PreviewDiscount(ctx, userID, couponCode, quotes)
		if err != nil {
			return domain.Order{}, nil, err
		}
		couponResult = &res
	}
	var orderItems []domain.OrderItem
	totalItems := 0
	for idx, meta := range metas {
		unitAmount := meta.UnitTotal
		if couponResult != nil && idx < len(couponResult.UnitDiscount) {
			unitAmount -= couponResult.UnitDiscount[idx]
			if unitAmount < 0 {
				unitAmount = 0
			}
		}
		totalItems += meta.Qty
		if totalItems > maxOrderTotalItems {
			return domain.Order{}, nil, ErrInvalidInput
		}
		for i := 0; i < meta.Qty; i++ {
			orderItems = append(orderItems, domain.OrderItem{
				OrderID:        0,
				PackageID:      meta.PackageID,
				SystemID:       meta.SystemID,
				SpecJSON:       meta.SpecJSON,
				Qty:            1,
				Amount:         unitAmount,
				Status:         domain.OrderItemStatusPendingPayment,
				GoodsTypeID:    quotes[idx].GoodsTypeID,
				Action:         "create",
				DurationMonths: meta.Months,
			})
		}
	}
	orderNo := fmt.Sprintf("ORD-%d-%d", userID, time.Now().UnixNano())
	order := domain.Order{
		UserID:         userID,
		OrderNo:        orderNo,
		Source:         resolveOrderSource(ctx),
		Status:         domain.OrderStatusPendingPayment,
		TotalAmount:    total,
		Currency:       currency,
		IdempotencyKey: idemKey,
	}
	if couponResult != nil {
		order.CouponCode = couponResult.Coupon.Code
		order.CouponID = &couponResult.Coupon.ID
		order.CouponDiscount = couponResult.TotalDiscount
		if order.CouponDiscount > order.TotalAmount {
			order.CouponDiscount = order.TotalAmount
		}
		order.TotalAmount -= order.CouponDiscount
	}
	if err := s.orders.CreateOrder(ctx, &order); err != nil {
		// A concurrent request with the same idempotency key may have won the
		// race; replay its order instead of surfacing the unique-index error.
		if existing, replayItems, ok := s.replayExistingOrder(ctx, userID, idemKey); ok {
			return existing, replayItems, nil
		}
		return domain.Order{}, nil, err
	}
	for i := range orderItems {
		orderItems[i].OrderID = order.ID
	}
	if err := s.items.CreateOrderItems(ctx, orderItems); err != nil {
		// Roll back the just-created order so a partial failure does not leave
		// an itemless orphan burning the idempotency key.
		_ = s.orders.DeleteOrder(ctx, order.ID)
		return domain.Order{}, nil, err
	}
	if couponResult != nil && order.CouponID != nil && s.coupon != nil {
		if err := s.coupon.CreateRedemption(ctx, &domain.CouponRedemption{
			CouponID:       *order.CouponID,
			OrderID:        order.ID,
			UserID:         order.UserID,
			Status:         domain.CouponRedemptionStatusApplied,
			DiscountAmount: order.CouponDiscount,
		}); err != nil {
			_ = s.orders.DeleteOrder(ctx, order.ID)
			return domain.Order{}, nil, err
		}
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.pending_payment", map[string]any{
			"status": order.Status,
			"total":  order.TotalAmount,
		})
	}
	return order, orderItems, nil
}

func (s *OrderService) PreviewCouponFromItems(ctx context.Context, userID int64, inputs []OrderItemInput, couponCode string) (CouponPreview, error) {
	if len(inputs) == 0 || strings.TrimSpace(couponCode) == "" || s.coupon == nil {
		return CouponPreview{}, ErrInvalidInput
	}
	quotes, total, err := s.buildCouponQuotesFromInputs(ctx, userID, inputs)
	if err != nil {
		return CouponPreview{}, err
	}
	res, err := s.coupon.PreviewDiscount(ctx, userID, couponCode, quotes)
	if err != nil {
		return CouponPreview{}, err
	}
	discount := res.TotalDiscount
	if discount > total {
		discount = total
	}
	return CouponPreview{
		CouponCode: res.Coupon.Code,
		Original:   total,
		Discount:   discount,
		Final:      total - discount,
	}, nil
}

func (s *OrderService) PreviewCouponFromCart(ctx context.Context, userID int64, couponCode string) (CouponPreview, error) {
	if strings.TrimSpace(couponCode) == "" || s.coupon == nil {
		return CouponPreview{}, ErrInvalidInput
	}
	items, err := s.cart.ListCartItems(ctx, userID)
	if err != nil {
		return CouponPreview{}, err
	}
	if len(items) == 0 {
		return CouponPreview{}, ErrInvalidInput
	}
	inputs := make([]OrderItemInput, 0, len(items))
	for _, item := range items {
		spec := parseCartSpecJSON(item.SpecJSON)
		inputs = append(inputs, OrderItemInput{
			PackageID: item.PackageID,
			SystemID:  item.SystemID,
			Spec:      spec,
			Qty:       item.Qty,
		})
	}
	return s.PreviewCouponFromItems(ctx, userID, inputs, couponCode)
}

func (s *OrderService) buildCouponQuotesFromInputs(ctx context.Context, userID int64, inputs []OrderItemInput) ([]appcoupon.QuoteItem, int64, error) {
	quotes := make([]appcoupon.QuoteItem, 0, len(inputs))
	var total int64
	for _, in := range inputs {
		if in.PackageID == 0 || in.SystemID == 0 {
			return nil, 0, ErrInvalidInput
		}
		if err := normalizeCartSpec(&in.Spec); err != nil {
			return nil, 0, err
		}
		pkg, err := s.catalog.GetPackage(ctx, in.PackageID)
		if err != nil {
			return nil, 0, err
		}
		plan, err := s.catalog.GetPlanGroup(ctx, pkg.PlanGroupID)
		if err != nil {
			return nil, 0, err
		}
		unitTotal, unitBase, addonCore, addonMem, addonDisk, addonBW, _, err := s.priceBreakdownForPackage(ctx, userID, pkg, plan, in.Spec)
		if err != nil {
			return nil, 0, err
		}
		qty, err := normalizeOrderQty(in.Qty)
		if err != nil {
			return nil, 0, err
		}
		quotes = append(quotes, appcoupon.QuoteItem{
			PackageID:       pkg.ID,
			GoodsTypeID:     pkg.GoodsTypeID,
			RegionID:        plan.RegionID,
			PlanGroupID:     plan.ID,
			AddonCore:       in.Spec.AddCores,
			AddonMemGB:      in.Spec.AddMemGB,
			AddonDiskGB:     in.Spec.AddDiskGB,
			AddonBWMbps:     in.Spec.AddBWMbps,
			UnitBaseAmount:  unitBase,
			UnitAddonAmount: addonCore + addonMem + addonDisk + addonBW,
			UnitAddonCore:   addonCore,
			UnitAddonMem:    addonMem,
			UnitAddonDisk:   addonDisk,
			UnitAddonBW:     addonBW,
			UnitTotalAmount: unitTotal,
			Qty:             qty,
		})
		total, err = addLineTotal(total, unitTotal, qty)
		if err != nil {
			return nil, 0, err
		}
	}
	return quotes, total, nil
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

func (s *OrderService) GetOrderForAdmin(ctx context.Context, orderID int64) (domain.Order, []domain.OrderItem, error) {
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return domain.Order{}, nil, err
	}
	items, err := s.items.ListOrderItems(ctx, orderID)
	if err != nil {
		return domain.Order{}, nil, err
	}
	return order, items, nil
}

func (s *OrderService) ListPaymentsForOrder(ctx context.Context, userID int64, orderID int64) ([]domain.OrderPayment, error) {
	if s.payments == nil {
		return nil, nil
	}
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, ErrForbidden
	}
	return s.payments.ListPaymentsByOrder(ctx, orderID)
}

func (s *OrderService) ListPaymentsForOrderAdmin(ctx context.Context, orderID int64) ([]domain.OrderPayment, error) {
	if s.payments == nil {
		return nil, nil
	}
	if _, err := s.orders.GetOrder(ctx, orderID); err != nil {
		return nil, err
	}
	return s.payments.ListPaymentsByOrder(ctx, orderID)
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
	if input.Amount != order.TotalAmount {
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
		// A concurrent submit with the same idempotency key/trade_no may have
		// won the race; replay its payment instead of surfacing the raw
		// unique-index error.
		if idemKey != "" {
			if existing, replayErr := s.payments.GetPaymentByIdempotencyKey(ctx, order.ID, idemKey); replayErr == nil {
				return existing, nil
			}
		}
		if input.TradeNo != "" {
			if existing, replayErr := s.payments.GetPaymentByTradeNo(ctx, input.TradeNo); replayErr == nil {
				if existing.OrderID != order.ID {
					return domain.OrderPayment{}, ErrConflict
				}
				return existing, nil
			}
		}
		return domain.OrderPayment{}, err
	}
	// Transition to pending_review conditionally: if the order was canceled or
	// rejected between the read above and this write (concurrent CancelOrder /
	// RejectOrder), do not resurrect it. The payment record is left in place;
	// the order stays canceled/rejected.
	transitioned, err := s.orders.UpdateOrderStatusIfCurrent(ctx, order.ID, domain.OrderStatusPendingReview, []domain.OrderStatus{
		domain.OrderStatusPendingPayment,
		domain.OrderStatusPendingReview,
	})
	if err != nil {
		return domain.OrderPayment{}, err
	}
	if !transitioned {
		return domain.OrderPayment{}, ErrConflict
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
	// The order CAS, the "no approved external payment" guard and the rejection
	// of still-pending payments must all commit in one transaction, or a
	// concurrent provider callback can approve a payment in the window between
	// the status read and the payment write — the order ends canceled with real
	// external money confirmed and no automatic refund (finance F1).
	claimed, err := s.orders.CancelOrderAtomic(ctx, order, []domain.OrderStatus{
		domain.OrderStatusPendingPayment,
		domain.OrderStatusPendingReview,
	})
	if err != nil {
		return err
	}
	if !claimed {
		// The order is no longer cancellable (already approved elsewhere, or it
		// now carries an approved external payment — the user must go through
		// the refund flow).
		return ErrConflict
	}
	order.Status = domain.OrderStatusCanceled
	s.releaseVPSLifecycleLocks(ctx, order.ID)
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.canceled", map[string]any{"status": order.Status})
	}
	if s.coupon != nil {
		_ = s.coupon.MarkOrderCanceled(ctx, order.ID)
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
	if input.Amount != order.TotalAmount {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	if input.Currency == "" {
		input.Currency = order.Currency
	}
	if order.Currency != "" && input.Currency != order.Currency {
		return domain.OrderPayment{}, ErrInvalidInput
	}
	// Idempotency: re-submitting the same trade_no must return the existing
	// payment instead of hitting the unique index and surfacing a raw DB error.
	if input.TradeNo != "" {
		if existing, err := s.payments.GetPaymentByTradeNo(ctx, input.TradeNo); err == nil {
			if existing.OrderID != order.ID {
				return domain.OrderPayment{}, ErrConflict
			}
			return existing, nil
		}
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
	transitioned, err := s.orders.UpdateOrderStatusIfCurrent(ctx, order.ID, domain.OrderStatusPendingReview, []domain.OrderStatus{
		domain.OrderStatusPendingPayment,
	})
	if err != nil {
		return domain.OrderPayment{}, err
	}
	if !transitioned {
		return domain.OrderPayment{}, ErrConflict
	}
	order.Status = domain.OrderStatusPendingReview
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
	if order.Status == domain.OrderStatusApproved {
		items, listErr := s.items.ListOrderItems(ctx, order.ID)
		if listErr != nil {
			return listErr
		}
		if err := s.finishOrderApprovalPersistence(ctx, adminID, order, items); err != nil {
			return err
		}
		for _, item := range items {
			if item.Action != "resize" {
				s.provisionOrderAsync(order.ID)
				break
			}
		}
		return nil
	}
	// An automatic approval (adminID==0 — a provider callback or a
	// configuration-driven auto-refund) must never resurrect a rejected order:
	// only a human admin (adminID>0) may re-approve an order the admin just
	// rejected. Otherwise a late payment callback racing RejectOrderAtomic
	// would undo the rejection (F5 doublepay).
	if adminID == 0 && order.Status == domain.OrderStatusRejected {
		return ErrConflict
	}
	if order.Status != domain.OrderStatusPendingReview &&
		order.Status != domain.OrderStatusPendingPayment &&
		order.Status != domain.OrderStatusRejected {
		return ErrConflict
	}
	expectedStatus := order.Status
	now := time.Now()
	order.Status = domain.OrderStatusApproved
	order.ApprovedAt = &now
	if adminID > 0 {
		order.ApprovedBy = &adminID
	}
	order.RejectedReason = ""
	items, err := s.items.ListOrderItems(ctx, order.ID)
	if err != nil {
		return err
	}
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

	itemIDs := make([]int64, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	claimed, err := s.orders.ApproveOrderAtomic(ctx, order, itemIDs, resizeTasks, []domain.OrderStatus{expectedStatus})
	if err != nil {
		return err
	}
	if !claimed {
		return ErrConflict
	}
	if err := s.finishOrderApprovalPersistence(ctx, adminID, order, items); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "order.approve", TargetType: "order", TargetID: fmt.Sprintf("%d", order.ID), DetailJSON: "{}"})
	}
	if s.coupon != nil {
		_ = s.coupon.MarkOrderConfirmed(ctx, order.ID)
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.approved", map[string]any{"status": domain.OrderStatusApproved})
	}
	if s.userTiers != nil {
		_ = s.userTiers.TryAutoApproveForUser(ctx, order.UserID, "order_success")
	}
	s.notifyOrderDecision(ctx, order.UserID, order.OrderNo, "order_approved", "Order Approved: {{.order.no}}", "Your order has been approved.")
	if hasResize && s.resizeTasks != nil {
		now := time.Now()
		for _, task := range resizeTasks {
			if task.ScheduledAt == nil || !task.ScheduledAt.After(now) {
				s.executeResizeTaskAsync(*task)
			}
		}
		if hasNonResize {
			s.provisionOrderAsync(order.ID)
		}
		return nil
	}
	s.provisionOrderAsync(order.ID)
	return nil
}

func (s *OrderService) finishOrderApprovalPersistence(ctx context.Context, adminID int64, order domain.Order, items []domain.OrderItem) error {
	for _, item := range items {
		if item.Action != "resize" {
			continue
		}
		if err := s.creditResizeRefundOnApprove(ctx, item); err != nil {
			return err
		}
	}
	if s.payments == nil {
		return nil
	}
	pays, err := s.payments.ListPaymentsByOrder(ctx, order.ID)
	if err != nil {
		return err
	}
	for _, pay := range pays {
		// Only upgrade the admin-reviewable payment. A provider payment that
		// never entered manual review (pending_payment) must not be
		// force-approved: the money has not necessarily arrived, and approving
		// it inflates revenue with payments that may never settle.
		if pay.Status != domain.PaymentStatusPendingReview {
			continue
		}
		if err := s.payments.UpdatePaymentStatus(ctx, pay.ID, domain.PaymentStatusApproved, &adminID, ""); err != nil {
			return err
		}
	}
	return nil
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
		s.provisionOrderAsync(orderID)
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
	expectedStatus := order.Status
	order.Status = domain.OrderStatusRejected
	order.RejectedReason = normalizedReason
	order.PendingReason = ""
	claimed, err := s.orders.RejectOrderAtomic(ctx, order, adminID, []domain.OrderStatus{expectedStatus})
	if err != nil {
		return err
	}
	if !claimed {
		return ErrConflict
	}
	// A balance-paid order (approved balance payment) is now rejected: return
	// the deducted wallet funds. The refund is idempotent via the
	// balance_refund:<orderID> key, so a concurrent compensation in payWithBalance
	// cannot double-credit.
	s.refundBalanceOnReject(ctx, order)
	s.releaseVPSLifecycleLocks(ctx, order.ID)
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "order.reject", TargetType: "order", TargetID: fmt.Sprintf("%d", order.ID), DetailJSON: mustJSON(map[string]any{"reason": normalizedReason})})
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.rejected", map[string]any{"status": domain.OrderStatusRejected, "reason": normalizedReason})
	}
	if s.coupon != nil {
		_ = s.coupon.MarkOrderCanceled(ctx, order.ID)
	}
	s.notifyOrderDecision(ctx, order.UserID, order.OrderNo, "order_rejected", "Order Rejected: {{.order.no}}", normalizedReason)
	return nil
}

// refundBalanceOnReject returns the wallet debit of a balance-paid order that
// is being rejected. It only acts when a balance settlement actually exists for
// the order (ref_type=order), and the credit itself is idempotent via the
// balance_refund:<orderID> key so concurrent compensation (payWithBalance) or a
// retried reject cannot double-credit the wallet.
func (s *OrderService) refundBalanceOnReject(ctx context.Context, order domain.Order) {
	if s.wallets == nil || order.TotalAmount <= 0 {
		return
	}
	hasDebit, err := s.wallets.HasWalletTransaction(ctx, order.UserID, "order", order.ID)
	if err != nil || !hasDebit {
		return
	}
	const refType = "balance_refund"
	idemKey := fmt.Sprintf("balance_refund:%d", order.ID)
	if idem, ok := s.wallets.(interface {
		AdjustWalletBalanceIdempotent(context.Context, int64, int64, string, string, int64, string, string) (domain.Wallet, error)
	}); ok {
		_, _ = idem.AdjustWalletBalanceIdempotent(ctx, order.UserID, order.TotalAmount, "credit", refType, order.ID, "refund: order rejected after balance payment", idemKey)
		return
	}
	if has, err := s.wallets.HasWalletTransaction(ctx, order.UserID, refType, order.ID); err == nil && has {
		return
	}
	_, _ = s.wallets.AdjustWalletBalance(ctx, order.UserID, order.TotalAmount, "credit", refType, order.ID, "refund: order rejected after balance payment")
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
	transitioned, err := s.orders.UpdateOrderStatusIfCurrent(ctx, order.ID, domain.OrderStatusProvisioning, []domain.OrderStatus{
		domain.OrderStatusApproved,
		domain.OrderStatusProvisioning,
		domain.OrderStatusFailed,
	})
	if err != nil || !transitioned {
		return
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.provisioning", map[string]any{"status": domain.OrderStatusProvisioning})
	}

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
			continue
		}
		if item.Action == "resize" && s.resizeTasks != nil {
			continue
		}
		switch item.Action {
		case "create", "renew", "emergency_renew", "resize", "refund":
		default:
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusFailed)
			continue
		}
		if item.Action == "create" {
			// Create items are driven by the durable provision job (lease +
			// fence), not the order-item status claim. The job is the ownership
			// token: a fresh lease is never stolen, and a crashed attempt is
			// resumed from its persisted stage instead of being blindly re-created.
			s.provisionCreateItem(ctx, order, item)
			continue
		}
		claimed, claimErr := s.items.ClaimOrderItemForProvision(ctx, item.ID)
		if claimErr != nil {
			continue
		}
		if !claimed {
			continue
		}
		switch item.Action {
		case "renew":
			if err := s.handleRenew(ctx, item); err != nil {
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
			if err := s.handleEmergencyRenew(ctx, item); err != nil {
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
			if err := s.handleResize(ctx, item); err != nil {
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
			if err := s.handleRefund(ctx, item); err != nil {
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

	refreshedItems, err := s.items.ListOrderItems(ctx, orderID)
	if err != nil {
		return
	}
	finalStatus := orderStatusFromProvisionedItems(refreshedItems)
	updated, err := s.orders.UpdateOrderStatusIfCurrent(ctx, order.ID, finalStatus, []domain.OrderStatus{
		domain.OrderStatusApproved,
		domain.OrderStatusProvisioning,
		domain.OrderStatusFailed,
	})
	if err != nil || !updated {
		return
	}
	// Only a genuinely active/completed order publishes the completion event; a
	// failed order publishes an explicit failure event and an order still
	// provisioning publishes neither (the watchdog will keep driving it). This
	// prevents a premature `order.completed` — and the "VPS provisioned"
	// notification it triggers — while provisioning is still running or after it
	// has definitively failed.
	if s.events != nil {
		switch finalStatus {
		case domain.OrderStatusActive:
			_, _ = s.events.Publish(ctx, order.ID, "order.completed", map[string]any{"status": finalStatus})
		case domain.OrderStatusFailed:
			_, _ = s.events.Publish(ctx, order.ID, "order.failed", map[string]any{"status": finalStatus})
		}
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

func (s *OrderService) provisionOrderAsync(orderID int64) {
	safego.Go("order provision", func() { s.provisionOrder(orderID) })
}

func (s *OrderService) executeResizeTaskAsync(task domain.ResizeTask) {
	safego.Go("order resize", func() { s.executeResizeTask(context.Background(), task) })
}

// provisionCreateItem drives a create order item through the durable provision
// job. If a job already exists for the item (a crashed earlier attempt), it is
// resumed from its persisted stage; otherwise a fresh intent is written first so
// a crash before CreateHost leaves a recoverable intent, not a stranded
// `creating:` item with no owner.
func (s *OrderService) provisionCreateItem(ctx context.Context, order domain.Order, item domain.OrderItem) {
	if s.provision == nil {
		return
	}
	job, err := s.provision.GetProvisionJobByOrderItem(ctx, item.ID)
	if err == ErrNotFound {
		job = s.buildCreateProvisionIntent(ctx, order, item)
		if job.ID == 0 {
			return
		}
	} else if err != nil {
		return
	}
	if job.Status == provisionJobStatusDone {
		// A terminal job from an earlier attempt is requeued for a manual retry.
		// RequeueProvisionJob preserves create_host_sent (reconcile-only) and
		// clears any other terminal stage so the next run resolves from host id.
		job, err = s.provision.RequeueProvisionJob(ctx, job.ID)
		if err != nil {
			return
		}
	}
	s.handleProvisionJob(ctx, job)
}

func (s *OrderService) buildCreateProvisionIntent(ctx context.Context, order domain.Order, item domain.OrderItem) domain.ProvisionJob {
	hostName := fmt.Sprintf("ecs-%d-%d-%d", order.UserID, order.ID, item.ID)
	job := domain.ProvisionJob{
		OrderID:     order.ID,
		OrderItemID: item.ID,
		HostName:    hostName,
		Status:      provisionJobStatusPending,
		Attempts:    0,
		NextRunAt:   time.Now(),
		Stage:       domain.ProvisionStageIntent,
		AttemptID:   fmt.Sprintf("%d-%d-%d", order.ID, item.ID, time.Now().UnixNano()),
	}
	if err := s.provision.CreateOrUpdateProvisionJob(ctx, &job); err != nil {
		return domain.ProvisionJob{}
	}
	return job
}

func orderStatusFromProvisionedItems(items []domain.OrderItem) domain.OrderStatus {
	if len(items) == 0 {
		return domain.OrderStatusFailed
	}
	allActive := true
	for _, item := range items {
		switch item.Status {
		case domain.OrderItemStatusFailed, domain.OrderItemStatusRejected, domain.OrderItemStatusCanceled:
			return domain.OrderStatusFailed
		case domain.OrderItemStatusActive:
		default:
			allActive = false
		}
	}
	if allActive {
		return domain.OrderStatusActive
	}
	return domain.OrderStatusProvisioning
}

func createHostFailureIsDefinitive(err error) bool {
	if errors.Is(err, ErrInvalidInput) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr) && strings.EqualFold(strings.TrimSpace(opErr.Op), "dial")
}

// specCurrentPeriodEnd returns the locally recorded current_period_end from an
// instance's spec JSON, or nil when absent/unparseable.
func specCurrentPeriodEnd(specJSON string) *time.Time {
	payload := parseJSON(specJSON)
	raw, ok := payload["current_period_end"]
	if !ok {
		return nil
	}
	s, ok := raw.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

// renewalNextDue computes the new due date for a renewal. It prefers the
// instance's locally recorded current_period_end over its ExpireAt so a retry
// after a partially applied renewal (remote renewed and ExpireAt advanced, but
// the period/spec update failed) re-asserts the same due date instead of
// extending the already-advanced expiry a second time.
func renewalNextDue(inst domain.VPSInstance, renewDays int, now time.Time) time.Time {
	if renewDays <= 0 {
		renewDays = defaultOrderRenewDays
	}
	if periodEnd := specCurrentPeriodEnd(inst.SpecJSON); periodEnd != nil {
		if periodEnd.After(now) {
			return periodEnd.Add(time.Duration(renewDays) * 24 * time.Hour)
		}
		if inst.ExpireAt != nil && inst.ExpireAt.After(now) {
			diffDays := int(inst.ExpireAt.Sub(*periodEnd).Hours() / 24)
			if diffDays == renewDays {
				// The previous attempt already advanced the local expiry by this
				// renewal's duration; re-assert it idempotently.
				return *inst.ExpireAt
			}
		}
	}
	if inst.ExpireAt != nil && inst.ExpireAt.After(now) {
		return inst.ExpireAt.Add(time.Duration(renewDays) * 24 * time.Hour)
	}
	return now.Add(time.Duration(renewDays) * 24 * time.Hour)
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
		payload.RenewDays = defaultOrderRenewDays
	}
	// Guard against time.Duration overflow: cap renewDays to the same safe
	// upper bound enforced at order creation (600 months = 18000 days).
	const maxRenewDays = 600 * 30
	if payload.RenewDays > maxRenewDays {
		return ErrInvalidInput
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
	next := renewalNextDue(inst, payload.RenewDays, time.Now())
	if err := cli.RenewHost(ctx, hostID, next); err != nil {
		s.logAutomation(ctx, item.OrderID, item.ID, "renew", map[string]any{"host_id": hostID, "next_due": next}, map[string]any{"error": err.Error()}, false, err.Error())
		return err
	}
	if inst.AdminStatus != domain.VPSAdminStatusNormal || inst.Status == domain.VPSStatusExpiredLocked {
		_ = cli.UnlockHost(ctx, hostID)
		// The renewal succeeded and the host is unlocked: persist the reset of
		// the stale local auto-lock markers. Otherwise the next expiry is never
		// re-locked (AutoLockExpired treats status=locked as a permanent
		// exemption) and the instance stays unlocked forever.
		_ = s.vps.UpdateInstanceAdminStatus(ctx, inst.ID, domain.VPSAdminStatusNormal)
		if inst.Status == domain.VPSStatusExpiredLocked {
			_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, domain.VPSStatusUnknown, 0)
		}
	}
	if err := s.vps.UpdateInstanceExpireAt(ctx, inst.ID, next); err != nil {
		s.logAutomation(ctx, item.OrderID, item.ID, "renew", map[string]any{"host_id": hostID, "next_due": next}, map[string]any{"error": err.Error(), "remote_renewed": true}, false, err.Error())
		return err
	}
	if err := s.vps.UpdateInstanceSpec(ctx, inst.ID, setCurrentPeriod(inst.SpecJSON, time.Now(), next)); err != nil {
		s.logAutomation(ctx, item.OrderID, item.ID, "renew", map[string]any{"host_id": hostID, "next_due": next}, map[string]any{"error": err.Error(), "remote_renewed": true}, false, err.Error())
		return err
	}
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
	// Guard against time.Duration overflow: emergency renew days should be
	// a small value (configured via policy), but cap defensively.
	const maxEmergencyRenewDays = 365
	if renewDays > maxEmergencyRenewDays {
		renewDays = maxEmergencyRenewDays
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	next := renewalNextDue(inst, renewDays, time.Now())
	if err := cli.RenewHost(ctx, hostID, next); err != nil {
		s.logAutomation(ctx, item.OrderID, item.ID, "emergency_renew", map[string]any{"host_id": hostID, "next_due": next}, map[string]any{"error": err.Error()}, false, err.Error())
		return err
	}
	if inst.AdminStatus != domain.VPSAdminStatusNormal || inst.Status == domain.VPSStatusExpiredLocked {
		_ = cli.UnlockHost(ctx, hostID)
		_ = s.vps.UpdateInstanceAdminStatus(ctx, inst.ID, domain.VPSAdminStatusNormal)
		if inst.Status == domain.VPSStatusExpiredLocked {
			_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, domain.VPSStatusUnknown, 0)
		}
	}
	if err := s.vps.UpdateInstanceExpireAt(ctx, inst.ID, next); err != nil {
		s.logAutomation(ctx, item.OrderID, item.ID, "emergency_renew", map[string]any{"host_id": hostID, "next_due": next}, map[string]any{"error": err.Error(), "remote_renewed": true}, false, err.Error())
		return err
	}
	if err := s.vps.UpdateInstanceSpec(ctx, inst.ID, setCurrentPeriod(inst.SpecJSON, time.Now(), next)); err != nil {
		s.logAutomation(ctx, item.OrderID, item.ID, "emergency_renew", map[string]any{"host_id": hostID, "next_due": next}, map[string]any{"error": err.Error(), "remote_renewed": true}, false, err.Error())
		return err
	}
	if err := s.vps.UpdateInstanceEmergencyRenewAt(ctx, inst.ID, time.Now()); err != nil {
		s.logAutomation(ctx, item.OrderID, item.ID, "emergency_renew", map[string]any{"host_id": hostID, "next_due": next}, map[string]any{"error": err.Error(), "remote_renewed": true}, false, err.Error())
		return err
	}
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
		// Recompute the monthly price from the same pricing source used at
		// creation, otherwise a member-tier instance loses its discounted price
		// after a resize and is overcharged on later renewals.
		baseMonthly := pkg.Monthly
		unitCore, unitMem, unitDisk, unitBW := plan.UnitCore, plan.UnitMem, plan.UnitDisk, plan.UnitBW
		if s.pricer != nil && inst.UserID > 0 {
			if pricing, _, err := s.pricer.ResolvePackagePricing(ctx, inst.UserID, targetPkgID); err == nil {
				baseMonthly = pricing.MonthlyPrice
				unitCore = pricing.UnitCore
				unitMem = pricing.UnitMem
				unitDisk = pricing.UnitDisk
				unitBW = pricing.UnitBW
			}
		}
		inst.MonthlyPrice = baseMonthly
		if plan.ID > 0 {
			inst.MonthlyPrice += int64(payload.Spec.AddCores)*unitCore +
				int64(payload.Spec.AddMemGB)*unitMem +
				int64(payload.Spec.AddDiskGB)*unitDisk +
				int64(payload.Spec.AddBWMbps)*unitBW
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
	walletOrderID, err := s.createAndApproveWalletRefund(ctx, inst.UserID, payload.RefundAmount, strings.TrimSpace(payload.Reason), meta, "vps_refund", item.OrderID)
	if err != nil {
		return err
	}
	if !payload.DeleteOnApprove {
		return nil
	}
	if s.cleanup == nil {
		// No durable outbox wired (e.g. tests): keep the previous behavior —
		// an inline delete failure fails the item so a manual RetryProvision
		// re-runs handleRefund (the wallet credit is idempotent).
		return s.deleteVPSForRefund(ctx, inst)
	}
	// Persist the durable cleanup outbox entry BEFORE the inline delete so a
	// crash after the wallet credit still deletes the refunded host. If the
	// inline delete fails the job stays pending and the cleanup worker retries
	// it with bounded backoff — the refund item is NOT failed, mirroring the
	// walletorder path.
	enqueued := s.enqueueRefundCleanupForOrder(ctx, inst, walletOrderID)
	if err := s.deleteVPSForRefund(ctx, inst); err != nil {
		if !enqueued {
			// No outbox entry to converge the deletion (host id or wallet order
			// id unavailable): fail the item so a manual retry can act rather
			// than silently completing the refund with a live VPS.
			return err
		}
		// The durable outbox will converge the deletion; the money was already
		// credited so the refund order completes.
		return nil
	}
	if enqueued && walletOrderID > 0 {
		if job, jerr := s.cleanup.GetRefundCleanupJobByWalletOrder(ctx, walletOrderID); jerr == nil {
			_ = s.cleanup.MarkRefundCleanupDone(ctx, job.ID)
		}
	}
	return nil
}

// enqueueRefundCleanupForOrder writes the durable cleanup outbox entry for a
// refund order approval so a crash after the wallet credit still deletes the
// host. Mirrors walletorder.enqueueRefundCleanup. A zero walletOrderID (an
// idempotent retry whose refund already settled) cannot be deduplicated, so no
// duplicate job is created — the first attempt already enqueued it. It returns
// whether an outbox entry is now in place (created or already present).
func (s *OrderService) enqueueRefundCleanupForOrder(ctx context.Context, inst domain.VPSInstance, walletOrderID int64) bool {
	if s.cleanup == nil || walletOrderID <= 0 {
		return false
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return false
	}
	if _, err := s.cleanup.GetRefundCleanupJobByWalletOrder(ctx, walletOrderID); err == nil {
		return true
	}
	job := domain.RefundCleanupJob{
		WalletOrderID: walletOrderID,
		UserID:        inst.UserID,
		VPSID:         inst.ID,
		HostID:        hostID,
		GoodsTypeID:   inst.GoodsTypeID,
		Status:        domain.RefundCleanupPending,
		NextRunAt:     time.Now(),
	}
	if err := s.cleanup.CreateRefundCleanupJob(ctx, &job); err != nil {
		return false
	}
	return true
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
	if err := cli.DeleteHost(ctx, hostID); err != nil && !deleteHostNotFound(err) {
		return err
	}
	// A DeleteHost that reports the host is already gone is an idempotent
	// success: the refund already credited the wallet, and the retry after a
	// crash between DeleteHost and DeleteInstance must converge instead of
	// leaving a phantom local instance.
	return s.vps.DeleteInstance(ctx, inst.ID)
}

// deleteHostNotFound reports whether a DeleteHost error means the host is
// already gone — an idempotent success for a refund deletion. Delegates to the
// shared classifier used by the admin VPS delete and the walletorder cleanup
// worker so every delete path converges identically on retry.
func deleteHostNotFound(err error) bool {
	return appshared.IsHostNotFoundError(err)
}

func (s *OrderService) createAndApproveWalletRefund(ctx context.Context, userID int64, amount int64, note string, meta map[string]any, txRefType string, txRefID int64) (int64, error) {
	if amount <= 0 {
		return 0, ErrInvalidInput
	}
	if s.wallets == nil {
		return 0, ErrInvalidInput
	}
	walletOrders, ok := s.wallets.(WalletOrderRepository)
	if !ok {
		return 0, ErrInvalidInput
	}
	if txRefType != "" && txRefID > 0 {
		exists, err := s.wallets.HasWalletTransaction(ctx, userID, txRefType, txRefID)
		if err != nil {
			return 0, err
		}
		if exists {
			// Idempotent retry: the refund already settled. Promote any still
			// pending wallet order and return its id so the cleanup outbox can
			// dedupe (a 0 means "already handled", and enqueue skips).
			s.promoteRefundWalletOrderIfNeeded(ctx, walletOrders, userID, txRefID)
			if order, found := s.findRefundWalletOrderByOrder(ctx, walletOrders, userID, txRefID); found {
				return order.ID, nil
			}
			return 0, nil
		}
	}
	order, found := s.findPendingRefundWalletOrder(ctx, walletOrders, userID, txRefID)
	if !found {
		order = domain.WalletOrder{
			UserID:   userID,
			Type:     domain.WalletOrderRefund,
			Amount:   amount,
			Currency: "CNY",
			Status:   domain.WalletOrderPendingReview,
			Note:     strings.TrimSpace(note),
			MetaJSON: mustJSON(meta),
		}
		if err := walletOrders.CreateWalletOrder(ctx, &order); err != nil {
			return 0, err
		}
	}
	adjust := s.wallets.AdjustWalletBalance
	if idempotent, ok := s.wallets.(interface {
		AdjustWalletBalanceIdempotent(context.Context, int64, int64, string, string, int64, string, string) (domain.Wallet, error)
	}); ok {
		adjust = func(ctx context.Context, userID, amount int64, txType, refType string, refID int64, note string) (domain.Wallet, error) {
			return idempotent.AdjustWalletBalanceIdempotent(ctx, userID, amount, txType, refType, refID, note, fmt.Sprintf("%s:%d", txRefType, txRefID))
		}
	}
	if _, err := adjust(ctx, userID, amount, "credit", txRefType, txRefID, fmt.Sprintf("refund wallet order %d", order.ID)); err != nil {
		alreadyDone := false
		if txRefType != "" && txRefID > 0 {
			if exists, checkErr := s.wallets.HasWalletTransaction(ctx, userID, txRefType, txRefID); checkErr == nil && exists {
				alreadyDone = true
			}
		}
		if !alreadyDone {
			return 0, err
		}
	}
	if err := walletOrders.UpdateWalletOrderStatus(ctx, order.ID, domain.WalletOrderApproved, nil, ""); err != nil {
		return 0, err
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
	return order.ID, nil
}

// findRefundWalletOrderByOrder returns any refund wallet order whose meta
// references the given order, regardless of status. Used to resolve the wallet
// order id on an idempotent refund retry so the cleanup outbox can dedupe.
func (s *OrderService) findRefundWalletOrderByOrder(ctx context.Context, walletOrders WalletOrderRepository, userID, txRefID int64) (domain.WalletOrder, bool) {
	if txRefID <= 0 {
		return domain.WalletOrder{}, false
	}
	orders, _, err := walletOrders.ListWalletOrders(ctx, userID, defaultWalletOrderPageSize, 0)
	if err != nil {
		return domain.WalletOrder{}, false
	}
	for _, candidate := range orders {
		if candidate.Type != domain.WalletOrderRefund {
			continue
		}
		if getInt64(parseJSON(candidate.MetaJSON)["order_id"]) == txRefID {
			return candidate, true
		}
	}
	return domain.WalletOrder{}, false
}

func (s *OrderService) findPendingRefundWalletOrder(ctx context.Context, walletOrders WalletOrderRepository, userID, txRefID int64) (domain.WalletOrder, bool) {
	if txRefID <= 0 {
		return domain.WalletOrder{}, false
	}
	orders, _, err := walletOrders.ListWalletOrders(ctx, userID, defaultWalletOrderPageSize, 0)
	if err != nil {
		return domain.WalletOrder{}, false
	}
	for _, candidate := range orders {
		if candidate.Type != domain.WalletOrderRefund || candidate.Status != domain.WalletOrderPendingReview {
			continue
		}
		if getInt64(parseJSON(candidate.MetaJSON)["order_id"]) == txRefID {
			return candidate, true
		}
	}
	return domain.WalletOrder{}, false
}

func (s *OrderService) promoteRefundWalletOrderIfNeeded(ctx context.Context, walletOrders WalletOrderRepository, userID int64, txRefID int64) {
	if walletOrders == nil || txRefID <= 0 {
		return
	}
	orders, _, err := walletOrders.ListWalletOrders(ctx, userID, defaultWalletOrderPageSize, 0)
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
	_, err = s.createAndApproveWalletRefund(ctx, inst.UserID, refundAmount, "resize refund", meta, "resize_refund", item.OrderID)
	return err
}

func (s *OrderService) executeResizeTask(ctx context.Context, task domain.ResizeTask) error {
	if s.resizeTasks == nil || s.items == nil || s.vps == nil || s.automation == nil || s.catalog == nil {
		return ErrInvalidInput
	}
	// Atomically claim the task with a lease so a second worker cannot
	// double-execute it, and a running task whose lease elapsed (a crashed
	// worker) can be reclaimed. If another worker won the claim, skip. The
	// returned task carries the fresh owner/fence; all final writes are CAS'd on
	// them so a worker that lost the lease cannot overwrite a takeover.
	claimedTask, claimed, err := s.resizeTasks.ClaimResizeTask(ctx, task.ID, s.workerID, time.Now().Add(domain.ResizeTaskStaleDuration))
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	task = claimedTask

	item, err := s.items.GetOrderItem(ctx, task.OrderItemID)
	if err != nil {
		s.failResizeTask(ctx, task, err.Error())
		return err
	}
	_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusProvisioning)
	if err := s.handleResize(ctx, item); err != nil {
		_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusFailed)
		if s.events != nil {
			_, _ = s.events.Publish(ctx, task.OrderID, "order.item.failed", map[string]any{"item_id": item.ID, "reason": err.Error()})
		}
		s.failResizeTask(ctx, task, err.Error())
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
	task.LeaseUntil = nil
	_, _ = s.resizeTasks.UpdateResizeTask(ctx, task)
	s.refreshOrderStatus(ctx, task.OrderID)
	return nil
}

// failResizeTask marks a resize task failed and releases the lease. The write is
// CAS'd on owner+fence; a lost lease abandons the write.
func (s *OrderService) failResizeTask(ctx context.Context, task domain.ResizeTask, reason string) {
	task.Status = domain.ResizeTaskStatusFailed
	finished := time.Now()
	task.FinishedAt = &finished
	task.LeaseUntil = nil
	_, _ = s.resizeTasks.UpdateResizeTask(ctx, task)
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

func (s *OrderService) priceForPackage(ctx context.Context, userID int64, packageID int64, spec CartSpec) (int64, int, error) {
	pkg, err := s.catalog.GetPackage(ctx, packageID)
	if err != nil {
		return 0, 0, err
	}
	plan, err := s.catalog.GetPlanGroup(ctx, pkg.PlanGroupID)
	if err != nil {
		return 0, 0, err
	}
	total, _, _, _, _, _, months, err := s.priceBreakdownForPackage(ctx, userID, pkg, plan, spec)
	if err != nil {
		return 0, 0, err
	}
	return total, months, nil
}

func (s *OrderService) priceBreakdownForPackage(ctx context.Context, userID int64, pkg domain.Package, plan domain.PlanGroup, spec CartSpec) (int64, int64, int64, int64, int64, int64, int, error) {
	if err := validateAddonSpec(spec, plan); err != nil {
		return 0, 0, 0, 0, 0, 0, 0, err
	}
	months, multiplier, err := resolveBillingCycle(ctx, s.billing, spec)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, 0, err
	}
	baseMonthly := pkg.Monthly
	unitCore := plan.UnitCore
	unitMem := plan.UnitMem
	unitDisk := plan.UnitDisk
	unitBW := plan.UnitBW
	if s.pricer != nil && userID > 0 {
		if pricing, _, err := s.pricer.ResolvePackagePricing(ctx, userID, pkg.ID); err == nil {
			baseMonthly = pricing.MonthlyPrice
			unitCore = pricing.UnitCore
			unitMem = pricing.UnitMem
			unitDisk = pricing.UnitDisk
			unitBW = pricing.UnitBW
		}
	}
	coreMonthly := int64(spec.AddCores) * unitCore
	memMonthly := int64(spec.AddMemGB) * unitMem
	diskMonthly := int64(spec.AddDiskGB) * unitDisk
	bwMonthly := int64(spec.AddBWMbps) * unitBW
	baseAmount := int64(math.Round(float64(baseMonthly) * multiplier))
	coreAmount := int64(math.Round(float64(coreMonthly) * multiplier))
	memAmount := int64(math.Round(float64(memMonthly) * multiplier))
	diskAmount := int64(math.Round(float64(diskMonthly) * multiplier))
	bwAmount := int64(math.Round(float64(bwMonthly) * multiplier))
	addonAmount := coreAmount + memAmount + diskAmount + bwAmount
	total := baseAmount + addonAmount
	return total, baseAmount, coreAmount, memAmount, diskAmount, bwAmount, months, nil
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
			} else {
				message = messageWithoutHTTPTrace(message)
			}
		}
	}
	requestPayload := map[string]any{
		"method":  "RPC",
		"url":     strings.TrimSpace(action),
		"headers": map[string]string{},
		"body":    redactAutomationPayload(req),
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
		redactedResponse := redactAutomationPayload(resp)
		responsePayload["body"] = redactedResponse
		responsePayload["format"] = "json"
		responsePayload["body_json"] = redactedResponse
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

func messageWithoutHTTPTrace(message string) string {
	const marker = "http_trace="
	if index := strings.LastIndex(message, marker); index >= 0 {
		message = strings.TrimSpace(message[:index])
	}
	if message == "" {
		return "upstream request failed"
	}
	return message
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

func randomPass(n int) (string, error) {
	letters := []byte("ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%")
	out := make([]byte, n)
	max := big.NewInt(int64(len(letters)))
	for i := range out {
		index, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = letters[index.Int64()]
	}
	return string(out), nil
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
	// Reserve the VPS atomically BEFORE creating the order so two concurrent
	// requests cannot both pass the check above (the unique index is the CAS).
	if err := s.reserveVPSLifecycleLock(ctx, vpsID, 0, "renew"); err != nil {
		return domain.Order{}, fmt.Errorf("该实例已有待审批或执行中的互斥订单: %w", err)
	}
	lockBound := false
	defer func() {
		if !lockBound {
			if l := s.vpsLocks(); l != nil {
				_ = l.ReleaseVPSLifecycleLockByVPS(ctx, vpsID)
			}
		}
	}()
	months := durationMonths
	if months <= 0 {
		if renewDays <= 0 {
			renewDays = defaultOrderRenewDays
		}
		months = int(math.Ceil(float64(renewDays) / 30.0))
	}
	if months <= 0 {
		months = 1
	}
	// Limit months to a safe upper bound that prevents time.Duration overflow.
	// time.Duration(days)*24*time.Hour overflows when days > ~106751 (292 years).
	// We cap at 600 months (50 years) which is a reasonable business maximum
	// and stays well within time.Duration's safe range.
	const maxRenewMonths = 600
	if months > maxRenewMonths {
		return domain.Order{}, ErrInvalidInput
	}
	renewDays = months * 30
	monthlyPrice := inst.MonthlyPrice
	if monthlyPrice < 0 {
		return domain.Order{}, ErrInvalidInput
	}
	maxInt64 := int64(^uint64(0) >> 1)
	if monthlyPrice > 0 && int64(months) > maxInt64/monthlyPrice {
		return domain.Order{}, ErrInvalidInput
	}
	amount := monthlyPrice * int64(months)
	if amount < 0 {
		return domain.Order{}, ErrInvalidInput
	}
	if renewDays <= 0 {
		renewDays = defaultOrderRenewDays
	}
	orderNo := fmt.Sprintf("REN-%d-%d", userID, time.Now().UnixNano())
	status := domain.OrderStatusPendingPayment
	itemStatus := domain.OrderItemStatusPendingPayment
	if amount == 0 {
		status = domain.OrderStatusPendingReview
		itemStatus = domain.OrderItemStatusPendingReview
	}
	order := domain.Order{
		UserID:      userID,
		OrderNo:     orderNo,
		Source:      resolveOrderSource(ctx),
		Status:      status,
		TotalAmount: amount,
		Currency:    "CNY",
	}
	if err := s.orders.CreateOrder(ctx, &order); err != nil {
		return domain.Order{}, err
	}
	// Bind the reservation to the created order; the deferred cleanup releases
	// it if anything below fails.
	if l := s.vpsLocks(); l != nil {
		if err := l.BindVPSLifecycleLock(ctx, vpsID, order.ID); err != nil {
			return domain.Order{}, err
		}
	}
	item := domain.OrderItem{
		OrderID:  order.ID,
		Qty:      1,
		Amount:   amount,
		Status:   itemStatus,
		Action:   "renew",
		VPSID:    vpsID,
		SpecJSON: mustJSON(map[string]any{"vps_id": vpsID, "renew_days": renewDays, "duration_months": months}),
	}
	if err := s.items.CreateOrderItems(ctx, []domain.OrderItem{item}); err != nil {
		return domain.Order{}, err
	}
	lockBound = true
	if s.events != nil {
		eventName := "order.pending_payment"
		if status == domain.OrderStatusPendingReview {
			eventName = "order.pending_review"
		}
		_, _ = s.events.Publish(ctx, order.ID, eventName, map[string]any{"status": order.Status, "total": amount})
	}
	if amount == 0 {
		if err := s.ApproveOrder(ctx, 0, order.ID); err != nil {
			return domain.Order{}, err
		}
		if updated, err := s.orders.GetOrder(ctx, order.ID); err == nil {
			order = updated
		}
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
	if err := s.reserveVPSLifecycleLock(ctx, vpsID, 0, "emergency_renew"); err != nil {
		return domain.Order{}, err
	}
	lockBound := false
	defer func() {
		if !lockBound {
			if l := s.vpsLocks(); l != nil {
				_ = l.ReleaseVPSLifecycleLockByVPS(ctx, vpsID)
			}
		}
	}()
	orderNo := fmt.Sprintf("EMR-%d-%d", userID, time.Now().UnixNano())
	order := domain.Order{
		UserID:      userID,
		OrderNo:     orderNo,
		Source:      resolveOrderSource(ctx),
		Status:      domain.OrderStatusPendingReview,
		TotalAmount: 0,
		Currency:    "CNY",
	}
	if err := s.orders.CreateOrder(ctx, &order); err != nil {
		return domain.Order{}, err
	}
	if l := s.vpsLocks(); l != nil {
		if err := l.BindVPSLifecycleLock(ctx, vpsID, order.ID); err != nil {
			return domain.Order{}, err
		}
	}
	item := domain.OrderItem{
		OrderID:  order.ID,
		Qty:      1,
		Amount:   0,
		Status:   domain.OrderItemStatusPendingReview,
		Action:   "emergency_renew",
		VPSID:    vpsID,
		SpecJSON: mustJSON(map[string]any{"vps_id": vpsID, "renew_days": policy.RenewDays}),
	}
	if err := s.items.CreateOrderItems(ctx, []domain.OrderItem{item}); err != nil {
		return domain.Order{}, err
	}
	lockBound = true
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

func (s *OrderService) capabilityPolicyGoodsTypeID(ctx context.Context, inst domain.VPSInstance) int64 {
	if inst.GoodsTypeID > 0 {
		return inst.GoodsTypeID
	}
	if s.catalog == nil || inst.PackageID <= 0 {
		return 0
	}
	pkg, err := s.catalog.GetPackage(ctx, inst.PackageID)
	if err != nil {
		return 0
	}
	return pkg.GoodsTypeID
}

func (s *OrderService) automationFeatureAllowed(ctx context.Context, inst domain.VPSInstance, feature string, policyAllowed bool) (bool, error) {
	if !policyAllowed {
		return false, nil
	}
	fallback := policyAllowed
	if resolver, ok := s.automation.(appports.AutomationCapabilityResolver); ok {
		capability, err := resolver.CapabilitiesForGoodsType(ctx, s.capabilityPolicyGoodsTypeID(ctx, inst))
		if err != nil {
			return false, err
		}
		fallback = capability != nil && featureSliceContains(capability.Features, feature)
	}
	return isAutomationFeatureAllowed(inst, feature, fallback), nil
}

func (s *OrderService) CreateResizeOrder(ctx context.Context, userID int64, vpsID int64, spec *CartSpec, targetPackageID int64, resetAddons bool, scheduledAt *time.Time) (domain.Order, ResizeQuote, error) {
	if s.realname != nil {
		if err := s.realname.RequireAction(ctx, userID, "resize_vps"); err != nil {
			return domain.Order{}, ResizeQuote{}, err
		}
	}
	resizeDefault := true
	if v, ok := getSettingBool(ctx, s.settings, "resize_enabled"); ok {
		resizeDefault = v
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
	policy, err := loadCapabilityPolicy(ctx, s.capabilityPolicy, s.settings, inst.PackageID, s.capabilityPolicyGoodsTypeID(ctx, inst))
	if err != nil {
		return domain.Order{}, ResizeQuote{}, err
	}
	if policy.ResizeEnabled != nil {
		resizeDefault = *policy.ResizeEnabled
	}
	resizeAllowed, err := s.automationFeatureAllowed(ctx, inst, "resize", resizeDefault)
	if err != nil {
		return domain.Order{}, ResizeQuote{}, err
	}
	if !resizeAllowed {
		return domain.Order{}, ResizeQuote{}, ErrResizeDisabled
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
	if err := s.reserveVPSLifecycleLock(ctx, vpsID, 0, "resize"); err != nil {
		return domain.Order{}, ResizeQuote{}, ErrResizeInProgress
	}
	lockBound := false
	defer func() {
		if !lockBound {
			if l := s.vpsLocks(); l != nil {
				_ = l.ReleaseVPSLifecycleLockByVPS(ctx, vpsID)
			}
		}
	}()
	amount := quote.ChargeAmount
	orderNo := fmt.Sprintf("UPG-%d-%d", userID, time.Now().UnixNano())
	status := domain.OrderStatusPendingPayment
	itemStatus := domain.OrderItemStatusPendingPayment
	if amount <= 0 {
		status = domain.OrderStatusPendingReview
		itemStatus = domain.OrderItemStatusPendingReview
	}
	order := domain.Order{
		UserID:      userID,
		OrderNo:     orderNo,
		Source:      resolveOrderSource(ctx),
		Status:      status,
		TotalAmount: amount,
		Currency:    "CNY",
	}
	if err := s.orders.CreateOrder(ctx, &order); err != nil {
		return domain.Order{}, ResizeQuote{}, err
	}
	if l := s.vpsLocks(); l != nil {
		if err := l.BindVPSLifecycleLock(ctx, vpsID, order.ID); err != nil {
			return domain.Order{}, ResizeQuote{}, err
		}
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
		VPSID:    vpsID,
		SpecJSON: mustJSON(specPayload),
	}
	if err := s.items.CreateOrderItems(ctx, []domain.OrderItem{item}); err != nil {
		return domain.Order{}, ResizeQuote{}, err
	}
	lockBound = true
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
	refundDefault := true
	if v, ok := getSettingBool(ctx, s.settings, "refund_enabled"); ok {
		refundDefault = v
	}
	pkgPolicy, err := loadCapabilityPolicy(ctx, s.capabilityPolicy, s.settings, inst.PackageID, s.capabilityPolicyGoodsTypeID(ctx, inst))
	if err != nil {
		return domain.Order{}, 0, err
	}
	if pkgPolicy.RefundEnabled != nil {
		refundDefault = *pkgPolicy.RefundEnabled
	}
	refundAllowed, err := s.automationFeatureAllowed(ctx, inst, "refund", refundDefault)
	if err != nil {
		return domain.Order{}, 0, err
	}
	if !refundAllowed {
		return domain.Order{}, 0, ErrForbidden
	}
	if inst.UserID != userID {
		return domain.Order{}, 0, ErrForbidden
	}
	if pending, err := s.items.HasPendingResizeOrder(ctx, userID, vpsID); err != nil {
		return domain.Order{}, 0, err
	} else if pending {
		return domain.Order{}, 0, ErrConflict
	}
	// A refund order is exclusive per VPS: without this guard a user could open
	// two refund orders for the same VPS and, under an auto-approve path, get a
	// double wallet credit when both are provisioned concurrently.
	if pending, err := s.items.HasPendingRefundOrder(ctx, userID, vpsID); err != nil {
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
	if err := s.reserveVPSLifecycleLock(ctx, vpsID, 0, "refund"); err != nil {
		return domain.Order{}, 0, ErrConflict
	}
	lockBound := false
	defer func() {
		if !lockBound {
			if l := s.vpsLocks(); l != nil {
				_ = l.ReleaseVPSLifecycleLockByVPS(ctx, vpsID)
			}
		}
	}()
	item, err := s.items.GetOrderItem(ctx, inst.OrderItemID)
	if err != nil {
		return domain.Order{}, 0, err
	}
	refundPolicy := loadRefundPolicy(ctx, s.settings)
	// The refund base must be the amount actually paid for the item, which
	// already accounts for prepaid DurationMonths and any coupon discount.
	// MonthlyPrice (single month) is only a fallback when the paid amount is
	// unavailable; using it for multi-month prepays systematically under-refunds
	// (the walletorder path fixed this first — see calculateRefundAmount).
	baseAmount := item.Amount
	if baseAmount <= 0 {
		baseAmount = inst.MonthlyPrice
	}
	amount := calculateRefundAmountForAmount(inst, baseAmount, refundPolicy)
	if amount <= 0 {
		return domain.Order{}, 0, ErrForbidden
	}
	orderNo := fmt.Sprintf("REF-%d-%d", userID, time.Now().UnixNano())
	order := domain.Order{
		UserID:         userID,
		OrderNo:        orderNo,
		Source:         resolveOrderSource(ctx),
		Status:         domain.OrderStatusPendingReview,
		TotalAmount:    -amount,
		Currency:       "CNY",
		PendingReason:  strings.TrimSpace(reason),
		RejectedReason: "",
	}
	if err := s.orders.CreateOrder(ctx, &order); err != nil {
		return domain.Order{}, 0, err
	}
	if l := s.vpsLocks(); l != nil {
		if err := l.BindVPSLifecycleLock(ctx, vpsID, order.ID); err != nil {
			return domain.Order{}, 0, err
		}
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
		VPSID:    vpsID,
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
	lockBound = true
	// A user API key does not get to bypass the refund review gate: the
	// `refund_requires_approval` setting is an admin control, and auto-approving
	// an API-key refund would both credit the wallet without review and delete
	// the VPS (delete_on_approve) on a single unauthenticated-key call. Only the
	// explicit `refund_requires_approval=false` configuration enables instant
	// refunds, regardless of the request source.
	if !refundPolicy.RequireApproval {
		if err := s.ApproveOrder(ctx, 0, order.ID); err != nil {
			return domain.Order{}, 0, err
		}
		if updated, err := s.orders.GetOrder(ctx, order.ID); err == nil {
			order = updated
		}
	}
	if s.events != nil && order.Status == domain.OrderStatusPendingReview {
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
	resizeDefault := true
	if v, ok := getSettingBool(ctx, s.settings, "resize_enabled"); ok {
		resizeDefault = v
	}
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return ResizeQuote{}, CartSpec{}, err
	}
	policy, err := loadCapabilityPolicy(ctx, s.capabilityPolicy, s.settings, inst.PackageID, s.capabilityPolicyGoodsTypeID(ctx, inst))
	if err != nil {
		return ResizeQuote{}, CartSpec{}, err
	}
	if policy.ResizeEnabled != nil {
		resizeDefault = *policy.ResizeEnabled
	}
	resizeAllowed, err := s.automationFeatureAllowed(ctx, inst, "resize", resizeDefault)
	if err != nil {
		return ResizeQuote{}, CartSpec{}, err
	}
	if !resizeAllowed {
		return ResizeQuote{}, CartSpec{}, ErrResizeDisabled
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
