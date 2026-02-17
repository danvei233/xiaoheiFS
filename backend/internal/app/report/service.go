package report

import (
	"context"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	orders   appports.OrderRepository
	vps      appports.VPSRepository
	payments appports.PaymentRepository
}

type OverviewReport struct {
	TotalOrders   int            `json:"total_orders"`
	PendingReview int            `json:"pending_review"`
	Revenue       int64          `json:"revenue"`
	VPSCount      int            `json:"vps_count"`
	ExpiringSoon  int            `json:"expiring_soon"`
	Series        []RevenuePoint `json:"series"`
}

type RevenuePoint struct {
	Date   string `json:"date"`
	Amount int64  `json:"amount"`
}

type StatusPoint struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

func NewService(orders appports.OrderRepository, vps appports.VPSRepository, payments appports.PaymentRepository) *Service {
	return &Service{orders: orders, vps: vps, payments: payments}
}

func (s *Service) Overview(ctx context.Context) (OverviewReport, error) {
	orders, err := s.listAllOrders(ctx, appshared.OrderFilter{})
	if err != nil {
		return OverviewReport{}, err
	}
	pending := 0
	revenue := int64(0)
	for _, o := range orders {
		if o.Status == domain.OrderStatusPendingReview {
			pending++
		}
	}
	payments, _ := s.listAllPayments(ctx, appshared.PaymentFilter{Status: string(domain.PaymentStatusApproved)})
	for _, pay := range payments {
		revenue += pay.Amount
	}
	vpsCount := 0
	if s.vps != nil {
		_, total, _ := s.vps.ListInstances(ctx, 1, 0)
		vpsCount = total
	}
	expiring := 0
	if s.vps != nil {
		instances, _ := s.vps.ListInstancesExpiring(ctx, time.Now().Add(7*24*time.Hour))
		expiring = len(instances)
	}
	series, _ := s.RevenueByDay(ctx, 30)
	return OverviewReport{
		TotalOrders:   len(orders),
		PendingReview: pending,
		Revenue:       revenue,
		VPSCount:      vpsCount,
		ExpiringSoon:  expiring,
		Series:        series,
	}, nil
}

func (s *Service) RevenueByDay(ctx context.Context, days int) ([]RevenuePoint, error) {
	if days <= 0 {
		days = 30
	}
	from := time.Now().AddDate(0, 0, -days)
	to := time.Now()
	payments, err := s.listAllPayments(ctx, appshared.PaymentFilter{Status: string(domain.PaymentStatusApproved), From: &from, To: &to})
	if err != nil {
		return nil, err
	}
	points := map[string]int64{}
	for _, pay := range payments {
		key := pay.CreatedAt.Format("2006-01-02")
		points[key] += pay.Amount
	}
	var out []RevenuePoint
	for i := days; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		out = append(out, RevenuePoint{Date: d, Amount: points[d]})
	}
	return out, nil
}

func (s *Service) RevenueByMonth(ctx context.Context, months int) ([]RevenuePoint, error) {
	if months <= 0 {
		months = 6
	}
	from := time.Now().AddDate(0, -months, 0)
	to := time.Now()
	payments, err := s.listAllPayments(ctx, appshared.PaymentFilter{Status: string(domain.PaymentStatusApproved), From: &from, To: &to})
	if err != nil {
		return nil, err
	}
	points := map[string]int64{}
	for _, pay := range payments {
		key := pay.CreatedAt.Format("2006-01")
		points[key] += pay.Amount
	}
	var out []RevenuePoint
	for i := months; i >= 0; i-- {
		d := time.Now().AddDate(0, -i, 0).Format("2006-01")
		out = append(out, RevenuePoint{Date: d, Amount: points[d]})
	}
	return out, nil
}

func (s *Service) VPSStatus(ctx context.Context) ([]StatusPoint, error) {
	if s.vps == nil {
		return nil, nil
	}
	instances, _, err := s.vps.ListInstances(ctx, 10000, 0)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, inst := range instances {
		counts[string(inst.Status)]++
	}
	var out []StatusPoint
	for status, count := range counts {
		out = append(out, StatusPoint{Status: status, Count: count})
	}
	return out, nil
}

func (s *Service) listAllOrders(ctx context.Context, filter appshared.OrderFilter) ([]domain.Order, error) {
	limit := 200
	offset := 0
	var out []domain.Order
	for {
		items, total, err := s.orders.ListOrders(ctx, filter, limit, offset)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		offset += len(items)
		if offset >= total || len(items) == 0 {
			break
		}
	}
	return out, nil
}

func (s *Service) listAllPayments(ctx context.Context, filter appshared.PaymentFilter) ([]domain.OrderPayment, error) {
	if s.payments == nil {
		return nil, nil
	}
	limit := 200
	offset := 0
	var out []domain.OrderPayment
	for {
		items, total, err := s.payments.ListPayments(ctx, filter, limit, offset)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		offset += len(items)
		if offset >= total || len(items) == 0 {
			break
		}
	}
	return out, nil
}
