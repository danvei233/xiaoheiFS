package order

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"xiaoheiplay/internal/domain"
)

const (
	provisionJobStatusPending = "pending"
	provisionJobStatusRunning = "running"
	provisionJobStatusRetry   = "retry"
	provisionJobStatusDone    = "done"
)

func (s *OrderService) StartProvisionWorker(ctx context.Context) {
	if s.provision == nil || s.automation == nil || s.vps == nil || s.items == nil || s.orders == nil {
		return
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		s.processProvisionJobs(ctx, 20)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *OrderService) ProcessProvisionJobs(ctx context.Context, limit int) error {
	if s.provision == nil || s.automation == nil || s.vps == nil || s.items == nil || s.orders == nil {
		return ErrInvalidInput
	}
	if limit <= 0 {
		limit = 20
	}
	if v, ok := getSettingInt(ctx, s.settings, "provision_watchdog_max_jobs"); ok && v > 0 && limit > v {
		limit = v
	} else if limit > 8 {
		limit = 8
	}
	s.processProvisionJobs(ctx, limit)
	return nil
}

func (s *OrderService) processProvisionJobs(ctx context.Context, limit int) {
	jobs, err := s.provision.ListDueProvisionJobs(ctx, limit)
	if err != nil {
		return
	}
	for _, job := range jobs {
		s.handleProvisionJob(ctx, job)
	}
}

func (s *OrderService) handleProvisionJob(ctx context.Context, job domain.ProvisionJob) {
	now := time.Now()
	job.Attempts++
	job.Status = provisionJobStatusRunning
	job.LastError = ""
	job.NextRunAt = now.Add(5 * time.Second)
	_ = s.provision.UpdateProvisionJob(ctx, job)

	jobCtx := WithAutomationLogContext(ctx, job.OrderID, job.OrderItemID)
	order, err := s.orders.GetOrder(jobCtx, job.OrderID)
	if err != nil {
		s.finishProvisionJob(ctx, job, "order not found")
		return
	}
	item, err := s.items.GetOrderItem(jobCtx, job.OrderItemID)
	if err != nil {
		s.finishProvisionJob(ctx, job, "order item not found")
		return
	}
	if limitMinutes := provisionMaxMinutes(ctx, s.settings); limitMinutes > 0 && !job.CreatedAt.IsZero() {
		if time.Since(job.CreatedAt) > time.Duration(limitMinutes)*time.Minute {
			_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusFailed)
			s.refreshOrderStatus(ctx, order.ID)
			s.finishProvisionJob(ctx, job, "timeout")
			return
		}
	}
	if order.Status == domain.OrderStatusCanceled || order.Status == domain.OrderStatusRejected {
		s.finishProvisionJob(ctx, job, "order stopped")
		return
	}
	cli, err := s.client(jobCtx, item.GoodsTypeID)
	if err != nil {
		s.retryProvisionJob(ctx, job, err.Error())
		return
	}
	info, err := cli.GetHostInfo(jobCtx, job.HostID)
	if err != nil {
		s.retryProvisionJob(ctx, job, err.Error())
		return
	}
	if info.State == 0 {
		_ = s.touchProvisioningInstance(ctx, job)
		s.retryProvisionJob(ctx, job, "provisioning")
		return
	}
	if isProvisionFailedState(info.State) {
		_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusFailed)
		if inst, instErr := s.vps.GetInstanceByOrderItem(ctx, item.ID); instErr == nil {
			_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, MapAutomationState(info.State), info.State)
		}
		s.refreshOrderStatus(ctx, order.ID)
		s.finishProvisionJob(ctx, job, fmt.Sprintf("state=%d", info.State))
		return
	}
	if isReadyState(info.State) {
		if err := s.completeProvision(ctx, job, info); err != nil {
			s.retryProvisionJob(ctx, job, err.Error())
			return
		}
		s.finishProvisionJob(ctx, job, "")
		return
	}
	s.retryProvisionJob(ctx, job, fmt.Sprintf("state=%d", info.State))
}

func (s *OrderService) retryProvisionJob(ctx context.Context, job domain.ProvisionJob, reason string) {
	job.Status = provisionJobStatusRetry
	job.LastError = reason
	job.NextRunAt = time.Now().Add(provisionRetryDelay(job.Attempts))
	_ = s.provision.UpdateProvisionJob(ctx, job)
}

func provisionRetryDelay(attempts int) time.Duration {
	if attempts < 0 {
		attempts = 0
	}
	delay := 5 * time.Second
	if attempts > 6 {
		delay = 10 * time.Second
	}
	if attempts > 12 {
		delay = 15 * time.Second
	}
	return delay
}

func isReadyState(state int) bool {
	return state == 2 || state == 3 || state == 10
}

func isProvisionFailedState(state int) bool {
	return state == 11 || state == 5
}

func (s *OrderService) finishProvisionJob(ctx context.Context, job domain.ProvisionJob, reason string) {
	job.Status = provisionJobStatusDone
	job.LastError = reason
	job.NextRunAt = time.Now().Add(365 * 24 * time.Hour)
	_ = s.provision.UpdateProvisionJob(ctx, job)
}

func provisionMaxMinutes(ctx context.Context, settings SettingsRepository) int {
	if v, ok := getSettingInt(ctx, settings, "provision_watchdog_max_minutes"); ok && v > 0 {
		return v
	}
	return 20
}

func (s *OrderService) touchProvisioningInstance(ctx context.Context, job domain.ProvisionJob) error {
	inst, err := s.vps.GetInstanceByOrderItem(ctx, job.OrderItemID)
	if err == nil {
		_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, domain.VPSStatusProvisioning, 0)
		return nil
	}
	return err
}

func (s *OrderService) completeProvision(ctx context.Context, job domain.ProvisionJob, info AutomationHostInfo) error {
	order, err := s.orders.GetOrder(ctx, job.OrderID)
	if err != nil {
		return err
	}
	item, err := s.items.GetOrderItem(ctx, job.OrderItemID)
	if err != nil {
		return err
	}
	effectiveHostID := job.HostID
	if info.HostID > 0 {
		effectiveHostID = info.HostID
	}
	status := MapAutomationState(info.State)
	expireAt := info.ExpireAt
	if expireAt == nil {
		months := item.DurationMonths
		if months <= 0 {
			months = 1
		}
		t := time.Now().AddDate(0, months, 0)
		expireAt = &t
	}
	name := info.HostName
	if name == "" {
		name = job.HostName
	}
	accessInfo := mergeAccessInfo("", info)
	inst, err := s.vps.GetInstanceByOrderItem(ctx, job.OrderItemID)
	if err == nil {
		if effectiveHostID > 0 && inst.AutomationInstanceID != fmt.Sprintf("%d", effectiveHostID) {
			inst.AutomationInstanceID = fmt.Sprintf("%d", effectiveHostID)
			if name != "" {
				inst.Name = name
			}
			_ = s.vps.UpdateInstanceLocal(ctx, inst)
		}
		accessInfo = mergeAccessInfo(inst.AccessInfoJSON, info)
		_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, status, info.State)
		if expireAt != nil {
			_ = s.vps.UpdateInstanceExpireAt(ctx, inst.ID, *expireAt)
		}
		_ = s.vps.UpdateInstanceAccessInfo(ctx, inst.ID, accessInfo)
	} else if err == ErrNotFound {
		snap := s.buildVPSLocalSnapshot(ctx, order.UserID, item)
		newInst := domain.VPSInstance{
			UserID:               order.UserID,
			OrderItemID:          item.ID,
			AutomationInstanceID: fmt.Sprintf("%d", effectiveHostID),
			GoodsTypeID:          item.GoodsTypeID,
			Name:                 name,
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
			Status:               status,
			AutomationState:      info.State,
			AdminStatus:          domain.VPSAdminStatusNormal,
			ExpireAt:             expireAt,
			AccessInfoJSON:       accessInfo,
		}
		if err := s.vps.CreateInstance(ctx, &newInst); err != nil {
			return err
		}
	}
	_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusActive)
	_ = s.items.UpdateOrderItemAutomation(ctx, item.ID, fmt.Sprintf("%d", effectiveHostID))
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.item.active", map[string]any{"item_id": item.ID})
	}
	s.refreshOrderStatus(ctx, order.ID)
	return nil
}

func mergeAccessInfo(existing string, info AutomationHostInfo) string {
	osPwd := ""
	remoteIP := ""
	panelPwd := ""
	vncPwd := ""
	if existing != "" {
		var current map[string]any
		if err := json.Unmarshal([]byte(existing), &current); err == nil {
			if v, ok := current["os_password"]; ok {
				osPwd = fmt.Sprintf("%v", v)
			}
			if v, ok := current["remote_ip"]; ok {
				remoteIP = fmt.Sprintf("%v", v)
			}
			if v, ok := current["panel_password"]; ok {
				panelPwd = fmt.Sprintf("%v", v)
			}
			if v, ok := current["vnc_password"]; ok {
				vncPwd = fmt.Sprintf("%v", v)
			}
		}
	}
	if info.RemoteIP != "" {
		remoteIP = info.RemoteIP
	}
	if info.PanelPassword != "" {
		panelPwd = info.PanelPassword
	}
	if info.VNCPassword != "" {
		vncPwd = info.VNCPassword
	}
	if info.OSPassword != "" {
		osPwd = info.OSPassword
	}
	payload := map[string]any{
		"remote_ip":      remoteIP,
		"panel_password": panelPwd,
		"vnc_password":   vncPwd,
	}
	if osPwd != "" {
		payload["os_password"] = osPwd
	}
	return mustJSON(payload)
}

func (s *OrderService) refreshOrderStatus(ctx context.Context, orderID int64) {
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return
	}
	// Only reconcile runtime provisioning states here.
	// Pending-review/payment orders are controlled by payment-review flow.
	switch order.Status {
	case domain.OrderStatusApproved, domain.OrderStatusProvisioning, domain.OrderStatusActive, domain.OrderStatusFailed:
	default:
		return
	}
	items, err := s.items.ListOrderItems(ctx, orderID)
	if err != nil {
		return
	}
	allActive := true
	anyFailed := false
	anyPending := false
	for _, item := range items {
		switch item.Status {
		case domain.OrderItemStatusActive:
		case domain.OrderItemStatusFailed:
			allActive = false
			anyFailed = true
		default:
			allActive = false
			anyPending = true
		}
	}
	// Broken-state guard: provisioning order with only review items means it was
	// transitioned too early; move it back to pending_review for admin decision.
	if order.Status == domain.OrderStatusProvisioning {
		allPendingReview := len(items) > 0
		hasRuntime := false
		for _, item := range items {
			switch item.Status {
			case domain.OrderItemStatusApproved, domain.OrderItemStatusProvisioning, domain.OrderItemStatusActive, domain.OrderItemStatusFailed:
				hasRuntime = true
			}
			if item.Status != domain.OrderItemStatusPendingReview {
				allPendingReview = false
			}
		}
		if allPendingReview && !hasRuntime {
			order.Status = domain.OrderStatusPendingReview
			_ = s.orders.UpdateOrderMeta(ctx, order)
			if s.events != nil {
				_, _ = s.events.Publish(ctx, order.ID, "order.pending_review", map[string]any{"status": domain.OrderStatusPendingReview})
			}
			return
		}
	}
	status := order.Status
	if anyFailed {
		status = domain.OrderStatusFailed
	} else if allActive {
		status = domain.OrderStatusActive
	} else if anyPending {
		status = domain.OrderStatusProvisioning
	}
	if status == order.Status {
		return
	}
	order.Status = status
	_ = s.orders.UpdateOrderMeta(ctx, order)
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.completed", map[string]any{"status": status})
	}
	if status == domain.OrderStatusActive {
		s.notifyOrderActive(ctx, order.UserID, order.OrderNo)
		if s.messages != nil {
			_ = s.messages.NotifyUser(ctx, order.UserID, "provisioned", "VPS Provisioned", "Order "+order.OrderNo+" has been provisioned.")
		}
	} else if status == domain.OrderStatusFailed && s.messages != nil {
		_ = s.messages.NotifyUser(ctx, order.UserID, "provision_failed", "Provision Failed", "Order "+order.OrderNo+" failed to provision.")
	}
}

// ReconcileProvisioningOrders fixes orders stuck in provisioning when VPS is already active.
func (s *OrderService) ReconcileProvisioningOrders(ctx context.Context, limit int) (int, error) {
	if s.orders == nil || s.items == nil || s.vps == nil {
		return 0, ErrInvalidInput
	}
	if limit <= 0 {
		limit = 50
	}
	offset := 0
	processed := 0
	for {
		orders, total, err := s.orders.ListOrders(ctx, OrderFilter{Status: string(domain.OrderStatusProvisioning)}, limit, offset)
		if err != nil {
			return processed, err
		}
		for _, order := range orders {
			items, err := s.items.ListOrderItems(ctx, order.ID)
			if err != nil {
				continue
			}
			for _, item := range items {
				switch item.Status {
				case domain.OrderItemStatusActive, domain.OrderItemStatusFailed, domain.OrderItemStatusCanceled, domain.OrderItemStatusRejected:
					continue
				}
				inst, err := s.vps.GetInstanceByOrderItem(ctx, item.ID)
				if err != nil {
					continue
				}
				if isVPSReadyStatus(inst.Status) {
					_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusActive)
					if inst.AutomationInstanceID != "" && item.AutomationInstanceID != inst.AutomationInstanceID {
						_ = s.items.UpdateOrderItemAutomation(ctx, item.ID, inst.AutomationInstanceID)
					}
				}
			}
			s.refreshOrderStatus(ctx, order.ID)
			processed++
		}
		offset += len(orders)
		if offset >= total || len(orders) == 0 {
			break
		}
	}
	return processed, nil
}

func isVPSReadyStatus(status domain.VPSStatus) bool {
	switch status {
	case domain.VPSStatusRunning, domain.VPSStatusStopped, domain.VPSStatusRescue, domain.VPSStatusLocked, domain.VPSStatusExpiredLocked:
		return true
	default:
		return false
	}
}
