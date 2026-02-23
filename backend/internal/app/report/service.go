package report

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	orders     appports.OrderRepository
	orderItems appports.OrderItemRepository
	payments   appports.PaymentRepository
	vps        appports.VPSRepository
	catalog    appports.CatalogRepository
	goodsTypes appports.GoodsTypeRepository
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

func NewService(
	orders appports.OrderRepository,
	orderItems appports.OrderItemRepository,
	payments appports.PaymentRepository,
	vps appports.VPSRepository,
	catalog appports.CatalogRepository,
	goodsTypes appports.GoodsTypeRepository,
) *Service {
	return &Service{
		orders:     orders,
		orderItems: orderItems,
		payments:   payments,
		vps:        vps,
		catalog:    catalog,
		goodsTypes: goodsTypes,
	}
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

type RevenueAnalyticsLevel string

const (
	RevenueLevelOverall   RevenueAnalyticsLevel = "overall"
	RevenueLevelGoodsType RevenueAnalyticsLevel = "goods_type"
	RevenueLevelRegion    RevenueAnalyticsLevel = "region"
	RevenueLevelLine      RevenueAnalyticsLevel = "line"
	RevenueLevelPackage   RevenueAnalyticsLevel = "package"
)

type RevenueAnalyticsQuery struct {
	FromAt      time.Time
	ToAt        time.Time
	Level       RevenueAnalyticsLevel
	UserID      int64
	GoodsTypeID int64
	RegionID    int64
	LineID      int64
	PackageID   int64
	Page        int
	PageSize    int
	SortField   string
	SortOrder   string
}

type RevenueSummary struct {
	TotalRevenueCents int64    `json:"total_revenue_cents"`
	OrderCount        int      `json:"order_count"`
	YoYRatio          *float64 `json:"yoy_ratio,omitempty"`
	MoMRatio          *float64 `json:"mom_ratio,omitempty"`
	YoYComparable     bool     `json:"yoy_comparable"`
	MoMComparable     bool     `json:"mom_comparable"`
}

type RevenueShareItem struct {
	DimensionID   int64   `json:"dimension_id"`
	DimensionName string  `json:"dimension_name"`
	RevenueCents  int64   `json:"revenue_cents"`
	Ratio         float64 `json:"ratio"`
}

type RevenueTrendPoint struct {
	Bucket       string `json:"bucket"`
	RevenueCents int64  `json:"revenue_cents"`
	OrderCount   int    `json:"order_count"`
}

type RevenueTopItem struct {
	Rank          int     `json:"rank"`
	DimensionID   int64   `json:"dimension_id"`
	DimensionName string  `json:"dimension_name"`
	RevenueCents  int64   `json:"revenue_cents"`
	Ratio         float64 `json:"ratio"`
}

type RevenueDetailRecord struct {
	PaymentID   int64     `json:"payment_id"`
	OrderID     int64     `json:"order_id"`
	OrderNo     string    `json:"order_no"`
	UserID      int64     `json:"user_id"`
	GoodsTypeID int64     `json:"goods_type_id"`
	RegionID    int64     `json:"region_id"`
	LineID      int64     `json:"line_id"`
	PackageID   int64     `json:"package_id"`
	AmountCents int64     `json:"amount_cents"`
	PaidAt      time.Time `json:"paid_at"`
	Status      string    `json:"status"`
}

type RevenueOverview struct {
	Summary    RevenueSummary     `json:"summary"`
	ShareItems []RevenueShareItem `json:"share_items"`
	TopItems   []RevenueTopItem   `json:"top_items"`
}

type paymentSlice struct {
	payment domain.OrderPayment
	amount  int64
	dimID   int64
	dimName string
	goods   int64
	region  int64
	line    int64
	pkg     int64
	item    domain.OrderItem
	order   domain.Order
}

type revenueScope struct {
	goodsTypeID int64
	regionID    int64
	lineID      int64
	packageID   int64
	lineName    string
}

func (s *Service) RevenueAnalyticsOverview(ctx context.Context, q RevenueAnalyticsQuery) (RevenueOverview, error) {
	data, total, err := s.collectRevenueData(ctx, q)
	if err != nil {
		return RevenueOverview{}, err
	}
	displayLevel := nextRevenueDimensionLevel(q.Level)
	displayData, _, err := s.collectRevenueDataByDimension(ctx, q, displayLevel)
	if err != nil {
		return RevenueOverview{}, err
	}
	summary := RevenueSummary{
		TotalRevenueCents: total,
		OrderCount:        uniqueOrderCount(data),
	}
	yoy, yoyCmp := s.calcYoY(ctx, q, total)
	mom, momCmp := s.calcMoM(ctx, q, total)
	summary.YoYRatio, summary.YoYComparable = yoy, yoyCmp
	summary.MoMRatio, summary.MoMComparable = mom, momCmp
	return RevenueOverview{
		Summary:    summary,
		ShareItems: buildShareItems(displayData, total),
		TopItems:   buildTopItems(displayData, total, 5),
	}, nil
}

func (s *Service) RevenueAnalyticsTrend(ctx context.Context, q RevenueAnalyticsQuery) ([]RevenueTrendPoint, error) {
	data, _, err := s.collectRevenueData(ctx, q)
	if err != nil {
		return nil, err
	}
	buckets := map[string]*RevenueTrendPoint{}
	for _, item := range data {
		key := item.payment.CreatedAt.Format("2006-01-02")
		if _, ok := buckets[key]; !ok {
			buckets[key] = &RevenueTrendPoint{Bucket: key}
		}
		buckets[key].RevenueCents += item.amount
		buckets[key].OrderCount++
	}
	var keys []string
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]RevenueTrendPoint, 0, len(keys))
	for _, k := range keys {
		out = append(out, *buckets[k])
	}
	return out, nil
}

func (s *Service) RevenueAnalyticsTop(ctx context.Context, q RevenueAnalyticsQuery) ([]RevenueTopItem, error) {
	displayLevel := nextRevenueDimensionLevel(q.Level)
	data, total, err := s.collectRevenueDataByDimension(ctx, q, displayLevel)
	if err != nil {
		return nil, err
	}
	return buildTopItems(data, total, 5), nil
}

func (s *Service) RevenueAnalyticsDetails(ctx context.Context, q RevenueAnalyticsQuery) ([]RevenueDetailRecord, int, error) {
	data, _, err := s.collectRevenueData(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	sort.Slice(data, func(i, j int) bool {
		if q.SortField == "amount" {
			if q.SortOrder == "asc" {
				return data[i].amount < data[j].amount
			}
			return data[i].amount > data[j].amount
		}
		if q.SortOrder == "asc" {
			return data[i].payment.CreatedAt.Before(data[j].payment.CreatedAt)
		}
		return data[i].payment.CreatedAt.After(data[j].payment.CreatedAt)
	})
	total := len(data)
	page := q.Page
	if page <= 0 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []RevenueDetailRecord{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	out := make([]RevenueDetailRecord, 0, end-start)
	for _, row := range data[start:end] {
		out = append(out, RevenueDetailRecord{
			PaymentID:   row.payment.ID,
			OrderID:     row.order.ID,
			OrderNo:     row.order.OrderNo,
			UserID:      row.order.UserID,
			GoodsTypeID: row.goods,
			RegionID:    row.dimRegionID(),
			LineID:      row.dimLineID(),
			PackageID:   row.pkg,
			AmountCents: row.amount,
			PaidAt:      row.payment.CreatedAt,
			Status:      string(row.payment.Status),
		})
	}
	return out, total, nil
}

func (p paymentSlice) dimRegionID() int64 { return p.region }
func (p paymentSlice) dimLineID() int64   { return p.line }

func (s *Service) normalizeRevenueQuery(q RevenueAnalyticsQuery) (RevenueAnalyticsQuery, error) {
	if q.FromAt.IsZero() || q.ToAt.IsZero() {
		return q, errors.New("from_at and to_at are required")
	}
	if !q.FromAt.Before(q.ToAt) {
		return q, errors.New("from_at must be before to_at")
	}
	if q.ToAt.Sub(q.FromAt) > 366*24*time.Hour {
		return q, errors.New("time range exceeds limit")
	}
	switch q.Level {
	case RevenueLevelOverall:
		// overall allows querying without hierarchy filters
	case RevenueLevelGoodsType:
		if q.GoodsTypeID <= 0 {
			return q, errors.New("goods_type_id is required")
		}
	case RevenueLevelRegion:
		if q.GoodsTypeID <= 0 || q.RegionID <= 0 {
			return q, errors.New("goods_type_id and region_id are required")
		}
	case RevenueLevelLine:
		if q.GoodsTypeID <= 0 || q.RegionID <= 0 || q.LineID <= 0 {
			return q, errors.New("goods_type_id, region_id and line_id are required")
		}
	case RevenueLevelPackage:
		if q.GoodsTypeID <= 0 || q.RegionID <= 0 || q.LineID <= 0 || q.PackageID <= 0 {
			return q, errors.New("all hierarchy ids are required")
		}
	default:
		return q, errors.New("invalid level")
	}
	if q.SortField == "" {
		q.SortField = "paid_at"
	}
	if q.SortOrder == "" {
		q.SortOrder = "desc"
	}
	return q, nil
}

func (s *Service) collectRevenueData(ctx context.Context, q RevenueAnalyticsQuery) ([]paymentSlice, int64, error) {
	return s.collectRevenueDataByDimension(ctx, q, q.Level)
}

func (s *Service) collectRevenueDataByDimension(ctx context.Context, q RevenueAnalyticsQuery, dimLevel RevenueAnalyticsLevel) ([]paymentSlice, int64, error) {
	q, err := s.normalizeRevenueQuery(q)
	if err != nil {
		return nil, 0, err
	}
	orders, err := s.listAllOrders(ctx, appshared.OrderFilter{})
	if err != nil {
		return nil, 0, err
	}
	out := make([]paymentSlice, 0, len(orders))
	var total int64
	for _, order := range orders {
		if !shouldIncludeRevenueOrder(order.Status) {
			continue
		}
		if q.UserID > 0 && order.UserID != q.UserID {
			continue
		}

		pays, err := s.payments.ListPaymentsByOrder(ctx, order.ID)
		if err != nil {
			continue
		}
		recognizedAmount := order.TotalAmount
		effectiveAt := order.CreatedAt
		var paymentID int64 = -order.ID
		for _, p := range pays {
			if p.Status != domain.PaymentStatusApproved {
				continue
			}
			if p.CreatedAt.After(effectiveAt) {
				effectiveAt = p.CreatedAt
			}
			if paymentID < 0 {
				paymentID = p.ID
			}
		}
		if order.ApprovedAt != nil && !order.ApprovedAt.IsZero() {
			effectiveAt = *order.ApprovedAt
		}
		if !effectiveAt.After(q.FromAt) && !effectiveAt.Equal(q.FromAt) {
			continue
		}
		if !effectiveAt.Before(q.ToAt) && !effectiveAt.Equal(q.ToAt) {
			continue
		}

		items, err := s.orderItems.ListOrderItems(ctx, order.ID)
		if err != nil || len(items) == 0 {
			continue
		}
		weights := int64(0)
		for _, it := range items {
			if it.Amount > 0 {
				weights += it.Amount
			}
		}
		for idx, it := range items {
			amount := recognizedAmount / int64(len(items))
			if weights > 0 {
				amount = recognizedAmount * it.Amount / weights
				if idx == len(items)-1 {
					assigned := int64(0)
					for i := 0; i < len(items)-1; i++ {
						if weights > 0 {
							assigned += recognizedAmount * items[i].Amount / weights
						}
					}
					amount = recognizedAmount - assigned
				}
			}
			scope := s.resolveRevenueScope(ctx, it)
			if !s.matchHierarchy(q, scope) {
				continue
			}
			dimID, dimName, matched := s.resolveDimension(ctx, dimLevel, scope)
			if !matched {
				dimID = 0
				dimName = ""
			}
			total += amount
			out = append(out, paymentSlice{
				payment: domain.OrderPayment{
					ID:        paymentID,
					OrderID:   order.ID,
					UserID:    order.UserID,
					Amount:    recognizedAmount,
					Status:    domain.PaymentStatusApproved,
					CreatedAt: effectiveAt,
				},
				amount:  amount,
				dimID:   dimID,
				dimName: dimName,
				goods:   scope.goodsTypeID,
				region:  scope.regionID,
				line:    scope.lineID,
				pkg:     scope.packageID,
				item:    it,
				order:   order,
			})
		}
	}
	return out, total, nil
}

func shouldIncludeRevenueOrder(status domain.OrderStatus) bool {
	switch status {
	case domain.OrderStatusCanceled, domain.OrderStatusFailed, domain.OrderStatusPendingPayment:
		return false
	default:
		return true
	}
}

func nextRevenueDimensionLevel(level RevenueAnalyticsLevel) RevenueAnalyticsLevel {
	switch level {
	case RevenueLevelOverall:
		return RevenueLevelGoodsType
	case RevenueLevelGoodsType:
		return RevenueLevelRegion
	case RevenueLevelRegion:
		return RevenueLevelLine
	case RevenueLevelLine:
		return RevenueLevelPackage
	default:
		return RevenueLevelPackage
	}
}

func (s *Service) resolveDimension(ctx context.Context, level RevenueAnalyticsLevel, scope revenueScope) (int64, string, bool) {
	switch level {
	case RevenueLevelOverall, RevenueLevelGoodsType:
		if scope.goodsTypeID <= 0 {
			return 0, "", false
		}
		gt, err := s.goodsTypes.GetGoodsType(ctx, scope.goodsTypeID)
		if err != nil {
			return scope.goodsTypeID, fmt.Sprintf("类型-%d", scope.goodsTypeID), true
		}
		return gt.ID, gt.Name, true
	case RevenueLevelRegion:
		if scope.regionID <= 0 {
			return 0, "", false
		}
		region, err := s.catalog.GetRegion(ctx, scope.regionID)
		if err != nil {
			return scope.regionID, fmt.Sprintf("地区-%d", scope.regionID), true
		}
		return region.ID, region.Name, true
	case RevenueLevelLine:
		if scope.lineID <= 0 {
			return 0, "", false
		}
		name := scope.lineName
		if name == "" {
			name = fmt.Sprintf("line-%d", scope.lineID)
		}
		return scope.lineID, name, true
	case RevenueLevelPackage:
		if scope.packageID <= 0 {
			return 0, "", false
		}
		pkg, err := s.catalog.GetPackage(ctx, scope.packageID)
		if err != nil {
			return scope.packageID, fmt.Sprintf("套餐-%d", scope.packageID), true
		}
		return pkg.ID, pkg.Name, true
	default:
		return 0, "", false
	}
}

func (s *Service) matchHierarchy(q RevenueAnalyticsQuery, scope revenueScope) bool {
	if q.GoodsTypeID > 0 && scope.goodsTypeID != q.GoodsTypeID {
		return false
	}
	if q.RegionID > 0 && scope.regionID != q.RegionID {
		return false
	}
	if q.LineID > 0 && scope.lineID != q.LineID {
		return false
	}
	if q.PackageID > 0 && scope.packageID != q.PackageID {
		return false
	}
	return true
}

func (s *Service) resolveRevenueScope(ctx context.Context, item domain.OrderItem) revenueScope {
	scope := revenueScope{
		goodsTypeID: item.GoodsTypeID,
		packageID:   item.PackageID,
	}
	targetPackageID, vpsID := parseOrderItemSpecHints(item.SpecJSON)
	if scope.packageID <= 0 && targetPackageID > 0 {
		scope.packageID = targetPackageID
	}
	if scope.packageID > 0 && s.catalog != nil {
		if pkg, err := s.catalog.GetPackage(ctx, scope.packageID); err == nil {
			scope.packageID = pkg.ID
			if scope.goodsTypeID <= 0 {
				scope.goodsTypeID = pkg.GoodsTypeID
			}
			if plan, err := s.catalog.GetPlanGroup(ctx, pkg.PlanGroupID); err == nil {
				scope.regionID = plan.RegionID
				scope.lineID = plan.LineID
				scope.lineName = plan.Name
			}
		}
	}
	if vpsID > 0 && s.vps != nil {
		if inst, err := s.vps.GetInstance(ctx, vpsID); err == nil {
			if scope.goodsTypeID <= 0 {
				scope.goodsTypeID = inst.GoodsTypeID
			}
			if scope.regionID <= 0 {
				scope.regionID = inst.RegionID
			}
			if scope.lineID <= 0 {
				scope.lineID = inst.LineID
			}
			if scope.packageID <= 0 {
				scope.packageID = inst.PackageID
			}
		}
	}
	return scope
}

func parseOrderItemSpecHints(specJSON string) (targetPackageID int64, vpsID int64) {
	if specJSON == "" {
		return 0, 0
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(specJSON), &m); err != nil {
		return 0, 0
	}
	targetPackageID = readInt64Any(m["target_package_id"])
	if targetPackageID <= 0 {
		targetPackageID = readInt64Any(m["package_id"])
	}
	vpsID = readInt64Any(m["vps_id"])
	return targetPackageID, vpsID
}

func readInt64Any(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case int32:
		return int64(t)
	case json.Number:
		n, _ := t.Int64()
		return n
	default:
		return 0
	}
}

func (s *Service) resolveRegionLine(ctx context.Context, item domain.OrderItem) (int64, int64) {
	scope := s.resolveRevenueScope(ctx, item)
	return scope.regionID, scope.lineID
}

func uniqueOrderCount(rows []paymentSlice) int {
	set := map[int64]struct{}{}
	for _, row := range rows {
		set[row.order.ID] = struct{}{}
	}
	return len(set)
}

func buildShareItems(rows []paymentSlice, total int64) []RevenueShareItem {
	agg := map[int64]*RevenueShareItem{}
	for _, row := range rows {
		if row.dimID <= 0 || row.dimName == "" {
			continue
		}
		item := agg[row.dimID]
		if item == nil {
			item = &RevenueShareItem{DimensionID: row.dimID, DimensionName: row.dimName}
			agg[row.dimID] = item
		}
		item.RevenueCents += row.amount
	}
	out := make([]RevenueShareItem, 0, len(agg))
	for _, item := range agg {
		if total > 0 {
			item.Ratio = float64(item.RevenueCents) / float64(total)
		}
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RevenueCents > out[j].RevenueCents })
	return out
}

func buildTopItems(rows []paymentSlice, total int64, limit int) []RevenueTopItem {
	share := buildShareItems(rows, total)
	if limit > len(share) {
		limit = len(share)
	}
	out := make([]RevenueTopItem, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, RevenueTopItem{
			Rank:          i + 1,
			DimensionID:   share[i].DimensionID,
			DimensionName: share[i].DimensionName,
			RevenueCents:  share[i].RevenueCents,
			Ratio:         share[i].Ratio,
		})
	}
	return out
}

func (s *Service) calcYoY(ctx context.Context, q RevenueAnalyticsQuery, current int64) (*float64, bool) {
	span := q.ToAt.Sub(q.FromAt)
	prevFrom := q.FromAt.AddDate(-1, 0, 0)
	prevTo := prevFrom.Add(span)
	prevRows, prevTotal, err := s.collectRevenueData(ctx, RevenueAnalyticsQuery{
		FromAt:      prevFrom,
		ToAt:        prevTo,
		Level:       q.Level,
		UserID:      q.UserID,
		GoodsTypeID: q.GoodsTypeID,
		RegionID:    q.RegionID,
		LineID:      q.LineID,
		PackageID:   q.PackageID,
	})
	if err != nil || len(prevRows) == 0 || prevTotal == 0 {
		return nil, false
	}
	ratio := float64(current-prevTotal) / float64(prevTotal)
	return &ratio, true
}

func (s *Service) calcMoM(ctx context.Context, q RevenueAnalyticsQuery, current int64) (*float64, bool) {
	span := q.ToAt.Sub(q.FromAt)
	prevTo := q.FromAt
	prevFrom := prevTo.Add(-span)
	prevRows, prevTotal, err := s.collectRevenueData(ctx, RevenueAnalyticsQuery{
		FromAt:      prevFrom,
		ToAt:        prevTo,
		Level:       q.Level,
		UserID:      q.UserID,
		GoodsTypeID: q.GoodsTypeID,
		RegionID:    q.RegionID,
		LineID:      q.LineID,
		PackageID:   q.PackageID,
	})
	if err != nil || len(prevRows) == 0 || prevTotal == 0 {
		return nil, false
	}
	ratio := float64(current-prevTotal) / float64(prevTotal)
	return &ratio, true
}
