package order

import (
	"context"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
)

type Service = OrderService

type messageCenter interface {
	NotifyUser(ctx context.Context, userID int64, typ, title, content string) error
}

type realnameChecker interface {
	RequireAction(ctx context.Context, userID int64, action string) error
}

func NewService(orders appports.OrderRepository, items appports.OrderItemRepository, cart appports.CartRepository, catalog appports.CatalogRepository, images appports.SystemImageRepository, billing appports.BillingCycleRepository, vps appports.VPSRepository, wallets appports.WalletRepository, payments appports.PaymentRepository, events appports.EventPublisher, automation appports.AutomationClientResolver, robot appshared.RobotNotifier, audit appports.AuditRepository, users appports.UserRepository, email appports.EmailSender, settings appports.SettingsRepository, autoLogs appports.AutomationLogRepository, provision appports.ProvisionJobRepository, resizeTasks appports.ResizeTaskRepository, messages messageCenter, realname realnameChecker) *Service {
	return NewOrderService(orders, items, cart, catalog, images, billing, vps, wallets, payments, events, automation, robot, audit, users, email, settings, autoLogs, provision, resizeTasks, messages, realname)
}
