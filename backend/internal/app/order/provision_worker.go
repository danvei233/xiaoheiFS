package order

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/cryptox"
	"xiaoheiplay/internal/pkg/safego"
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
	if v, ok := getSettingInt(ctx, s.settings, "provision_watchdog_max_jobs"); ok && v > 0 {
		if limit > v {
			limit = v
		}
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
		// A panic in a buggy automation client must not take down the whole
		// watchdog. If this worker still owns the lease, leave the job in retry
		// (with the lease released) so it is picked up again; if the lease was
		// lost the takeover worker owns the outcome.
		func() {
			defer func() {
				if r := recover(); r != nil {
					safego.LogPanic("order provision job", r)
					if current, err := s.provision.GetProvisionJob(ctx, job.ID); err == nil && current.LeaseOwner == s.workerID {
						current.Status = provisionJobStatusRetry
						current.LastError = "panic"
						current.NextRunAt = time.Now().Add(provisionRetryDelay(current.Attempts))
						current.LeaseUntil = nil
						_, _ = s.provision.UpdateProvisionJob(ctx, current)
					}
				}
			}()
			s.handleProvisionJob(ctx, job)
			// The stage machine owns item/instance state; the order status is
			// reconciled here by the watchdog so a terminal item outcome is
			// reflected atomically without a mid-flight stale overwrite.
			s.refreshOrderStatus(ctx, job.OrderID)
		}()
	}
}

// handleProvisionJob atomically claims a due job for this worker and drives the
// durable provision intent to its next stage. The claim carries the lease
// owner and bumps the fence; every subsequent write is CAS'd on owner+fence so
// a worker that lost the lease cannot overwrite a takeover worker's results.
func (s *OrderService) handleProvisionJob(ctx context.Context, job domain.ProvisionJob) {
	claimedJob, claimed, err := s.provision.ClaimProvisionJob(ctx, job.ID, s.workerID, time.Now().Add(domain.ProvisionLeaseDuration))
	if err != nil || !claimed {
		return
	}
	job = claimedJob

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
	if order.Status == domain.OrderStatusCanceled || order.Status == domain.OrderStatusRejected {
		s.finishProvisionJob(ctx, job, "order stopped")
		return
	}
	// The job is now actively being processed by this worker: reflect that on
	// the item so the order is seen as provisioning while reconciliation / host
	// waiting is in flight (a retry must not leave the item in a terminal failed
	// state while the watchdog is still driving it).
	_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusProvisioning)
	cli, err := s.client(jobCtx, item.GoodsTypeID)
	if err != nil {
		s.retryProvisionJob(ctx, job, err.Error())
		return
	}

	stage := job.Stage
	if stage == "" {
		// Legacy jobs created by the pre-intent enqueueProvisionJob always have a
		// host id; treat them as already-created hosts waiting for ready.
		if job.HostID > 0 {
			stage = domain.ProvisionStageHostCreated
		} else {
			stage = domain.ProvisionStageIntent
		}
	}
	switch stage {
	case domain.ProvisionStageIntent:
		s.provisionStageCreate(ctx, job, order, item, cli)
	case domain.ProvisionStageCreateHostSent:
		s.provisionStageReconcile(ctx, job, order, item, cli)
	case domain.ProvisionStageHostCreated:
		s.provisionStageWait(ctx, job, order, item, cli)
	default:
		s.finishProvisionJob(ctx, job, "unexpected stage: "+stage)
	}
}

// provisionStageCreate executes the first CreateHost for a durable intent. The
// intent (creds + hostname) is persisted before the external call, the item is
// marked `creating:<hostname>`, and the stage is advanced to create_host_sent
// BEFORE CreateHost so a crash mid-call is reconciled — never blindly
// re-created — by the next worker.
func (s *OrderService) provisionStageCreate(ctx context.Context, job domain.ProvisionJob, order domain.Order, item domain.OrderItem, cli AutomationClient) {
	pkg, err := s.catalog.GetPackage(ctx, item.PackageID)
	if err != nil {
		s.retryProvisionJob(ctx, job, err.Error())
		return
	}
	plan, err := s.catalog.GetPlanGroup(ctx, pkg.PlanGroupID)
	if err != nil {
		s.retryProvisionJob(ctx, job, err.Error())
		return
	}
	img, err := s.images.GetSystemImage(ctx, item.SystemID)
	if err != nil {
		s.retryProvisionJob(ctx, job, err.Error())
		return
	}
	var spec CartSpec
	_ = json.Unmarshal([]byte(item.SpecJSON), &spec)
	cpu := pkg.Cores + spec.AddCores
	mem := pkg.MemoryGB + spec.AddMemGB
	disk := pkg.DiskGB + spec.AddDiskGB
	bw := pkg.BandwidthMB + spec.AddBWMbps

	portNum := pkg.PortNum
	if portNum <= 0 {
		portNum = defaultPortNum
	}
	hostName := strings.TrimSpace(job.HostName)
	if hostName == "" {
		hostName = fmt.Sprintf("ecs-%d-%d-%d", order.UserID, order.ID, item.ID)
	}

	sysPwd, vncPwd := s.decryptProvisionCreds(job.CredsEncrypted)
	if sysPwd == "" || vncPwd == "" {
		var genErr error
		sysPwd, genErr = randomPass(systemPasswordLength)
		if genErr != nil {
			s.retryProvisionJob(ctx, job, fmt.Sprintf("generate system password: %v", genErr))
			return
		}
		vncPwd, genErr = randomPass(vncPasswordLength)
		if genErr != nil {
			s.retryProvisionJob(ctx, job, fmt.Sprintf("generate VNC password: %v", genErr))
			return
		}
		job.CredsEncrypted = s.encryptProvisionCreds(sysPwd, vncPwd)
		job.HostName = hostName
		if !s.updateProvisionJob(ctx, job) {
			return
		}
	}
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

	// Durable marker + stage before the external call so a crash here leaves a
	// recoverable intent, not a stranded `creating:` item.
	_ = s.items.UpdateOrderItemAutomation(ctx, item.ID, "creating:"+hostName)
	job.Stage = domain.ProvisionStageCreateHostSent
	if !s.updateProvisionJob(ctx, job) {
		return
	}

	res, createErr := cli.CreateHost(ctx, req)
	if createErr != nil {
		if createHostFailureIsDefinitive(createErr) {
			if clearErr := s.items.UpdateOrderItemAutomation(ctx, item.ID, ""); clearErr != nil {
				s.retryProvisionJob(ctx, job, clearErr.Error())
				return
			}
			s.failProvisionItem(ctx, order.ID, item.ID, createErr.Error())
			job.Stage = domain.ProvisionStageFailed
			s.finishProvisionJob(ctx, job, createErr.Error())
			return
		}
		// Result unknown: the request may or may not have reached the provider.
		// Never issue a second CreateHost. Surface the failure to the order and
		// keep the job's create_host_sent stage so a manual retry reconciles by
		// hostname instead of re-creating a duplicate resource.
		s.logAutomation(ctx, order.ID, item.ID, "create_host", req, map[string]any{"error": createErr.Error()}, false, createErr.Error())
		s.failProvisionItem(ctx, order.ID, item.ID, createErr.Error())
		s.finishProvisionJob(ctx, job, createErr.Error())
		return
	}

	hostID := res.HostID
	if hostID == 0 {
		hosts, listErr := cli.ListHostSimple(ctx, hostName)
		if listErr != nil {
			s.retryProvisionJob(ctx, job, listErr.Error())
			return
		}
		for _, host := range hosts {
			if strings.EqualFold(strings.TrimSpace(host.HostName), hostName) {
				hostID = host.ID
				break
			}
		}
	}
	if hostID == 0 {
		s.logAutomation(ctx, order.ID, item.ID, "create_host", req, res.Raw, false, "host id not found")
		s.retryProvisionJob(ctx, job, "host id not found")
		return
	}

	job.HostID = hostID
	job.Stage = domain.ProvisionStageHostCreated
	if !s.updateProvisionJob(ctx, job) {
		return
	}
	_ = s.items.UpdateOrderItemAutomation(ctx, item.ID, fmt.Sprintf("%d", hostID))
	s.provisionStageWait(ctx, job, order, item, cli)
}

// provisionStageReconcile handles a CreateHost whose result was unknown: it
// looks the host up by the deterministic hostname and never issues a second
// CreateHost. When reconciliation cannot find a host after the watchdog budget
// elapses, the job enters an explicit failed state instead of manufacturing a
// duplicate resource.
func (s *OrderService) provisionStageReconcile(ctx context.Context, job domain.ProvisionJob, order domain.Order, item domain.OrderItem, cli AutomationClient) {
	hostName := strings.TrimSpace(job.HostName)
	if hostName == "" {
		s.finishProvisionJob(ctx, job, "missing hostname for reconcile")
		return
	}
	if provisionJobTimedOut(job, ctx, s.settings) {
		s.failProvisionItem(ctx, order.ID, item.ID, "create-host result unknown and watchdog budget elapsed")
		job.Stage = domain.ProvisionStageFailed
		s.finishProvisionJob(ctx, job, "create-host result unknown and watchdog budget elapsed")
		return
	}
	hosts, err := cli.ListHostSimple(ctx, hostName)
	if err != nil {
		s.retryProvisionJob(ctx, job, err.Error())
		return
	}
	var hostID int64
	for _, host := range hosts {
		if strings.EqualFold(strings.TrimSpace(host.HostName), hostName) {
			hostID = host.ID
			break
		}
	}
	if hostID == 0 {
		s.retryProvisionJob(ctx, job, "create-host result unknown, still reconciling")
		return
	}
	job.HostID = hostID
	job.Stage = domain.ProvisionStageHostCreated
	if !s.updateProvisionJob(ctx, job) {
		return
	}
	_ = s.items.UpdateOrderItemAutomation(ctx, item.ID, fmt.Sprintf("%d", hostID))
	s.provisionStageWait(ctx, job, order, item, cli)
}

// provisionStageWait monitors an already-created host until it is ready, failed,
// or the watchdog budget elapses. Long-running calls renew the lease so a
// slow-but-alive worker is not reclaimed by a second instance.
func (s *OrderService) provisionStageWait(ctx context.Context, job domain.ProvisionJob, order domain.Order, item domain.OrderItem, cli AutomationClient) {
	hostID := job.HostID
	if hostID <= 0 {
		s.retryProvisionJob(ctx, job, "missing host id")
		return
	}
	info, err := cli.GetHostInfo(ctx, hostID)
	if err != nil {
		if provisionJobTimedOut(job, ctx, s.settings) {
			s.failProvisionItem(ctx, order.ID, item.ID, "timeout: "+err.Error())
			s.finishProvisionJob(ctx, job, "timeout: "+err.Error())
			return
		}
		s.retryProvisionJob(ctx, job, err.Error())
		return
	}
	if isReadyState(info.State) {
		if err := s.completeProvision(ctx, job, info); err != nil {
			// A local persistence failure surfaces as a failed order; the
			// created host is retained (job host_id) so a manual retry reuses it
			// instead of manufacturing a duplicate.
			s.failProvisionItem(ctx, order.ID, item.ID, err.Error())
			s.finishProvisionJob(ctx, job, err.Error())
			return
		}
		s.finishProvisionJob(ctx, job, "")
		return
	}
	if isProvisionFailedState(info.State) {
		s.failProvisionItem(ctx, order.ID, item.ID, fmt.Sprintf("state=%d", info.State))
		if inst, instErr := s.vps.GetInstanceByOrderItem(ctx, item.ID); instErr == nil {
			_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, MapAutomationState(info.State), info.State)
		}
		s.finishProvisionJob(ctx, job, fmt.Sprintf("state=%d", info.State))
		return
	}
	if provisionJobTimedOut(job, ctx, s.settings) {
		s.failProvisionItem(ctx, order.ID, item.ID, "timeout")
		s.finishProvisionJob(ctx, job, "timeout")
		return
	}
	if info.State == 0 {
		_ = s.touchProvisioningInstance(ctx, job)
	}
	s.retryProvisionJob(ctx, job, fmt.Sprintf("state=%d", info.State))
}

// retryProvisionJob marks a job for another attempt, releasing the lease so a
// later watchdog cycle (or another instance) can reclaim it. The write is CAS'd
// on owner+fence; a lost lease silently abandons the write.
func (s *OrderService) retryProvisionJob(ctx context.Context, job domain.ProvisionJob, reason string) {
	job.Status = provisionJobStatusRetry
	job.LastError = reason
	job.NextRunAt = time.Now().Add(provisionRetryDelay(job.Attempts))
	job.LeaseUntil = nil
	_ = s.updateProvisionJob(ctx, job)
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

// finishProvisionJob marks a job terminal, releasing the lease.
func (s *OrderService) finishProvisionJob(ctx context.Context, job domain.ProvisionJob, reason string) {
	job.Status = provisionJobStatusDone
	job.LastError = reason
	job.NextRunAt = time.Now().Add(365 * 24 * time.Hour)
	job.LeaseUntil = nil
	_ = s.updateProvisionJob(ctx, job)
}

func provisionMaxMinutes(ctx context.Context, settings SettingsRepository) int {
	if v, ok := getSettingInt(ctx, settings, "provision_watchdog_max_minutes"); ok && v > 0 {
		return v
	}
	return 20
}

// provisionJobTimedOut reports whether the job has exceeded the provisioning
// watchdog budget. A zero CreatedAt (never written) never times out.
func provisionJobTimedOut(job domain.ProvisionJob, ctx context.Context, settings SettingsRepository) bool {
	if limitMinutes := provisionMaxMinutes(ctx, settings); limitMinutes > 0 && !job.CreatedAt.IsZero() {
		return time.Since(job.CreatedAt) > time.Duration(limitMinutes)*time.Minute
	}
	return false
}

func (s *OrderService) touchProvisioningInstance(ctx context.Context, job domain.ProvisionJob) error {
	inst, err := s.vps.GetInstanceByOrderItem(ctx, job.OrderItemID)
	if err == nil {
		_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, domain.VPSStatusProvisioning, 0)
		return nil
	}
	return err
}

// updateProvisionJob persists a job write CAS'd on the lease owner and fence,
// returning whether the lease is still held. It is the single write path used
// by the provision worker.
func (s *OrderService) updateProvisionJob(ctx context.Context, job domain.ProvisionJob) bool {
	if s.provision == nil {
		return false
	}
	ok, err := s.provision.UpdateProvisionJob(ctx, job)
	return err == nil && ok
}

func (s *OrderService) completeProvision(ctx context.Context, job domain.ProvisionJob, info AutomationHostInfo) error {
	order, err := s.orders.GetOrder(ctx, job.OrderID)
	if err != nil {
		return err
	}
	// The order may have been canceled/rejected while the slow GetHostInfo call
	// was in flight. Never activate items or VPS instances for a dead order.
	if order.Status == domain.OrderStatusCanceled || order.Status == domain.OrderStatusRejected {
		return fmt.Errorf("order %s during provisioning: %w", order.Status, ErrConflict)
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
	// Credentials that survived a crash (persisted encrypted in the intent) are
	// merged back so a provider that never returns them again does not lose them.
	sysPwd, vncPwd := s.decryptProvisionCreds(job.CredsEncrypted)
	accessInfo := mergeAccessInfoWithCreds("", info, sysPwd, vncPwd)
	inst, err := s.vps.GetInstanceByOrderItem(ctx, job.OrderItemID)
	if err == nil {
		if effectiveHostID > 0 && inst.AutomationInstanceID != fmt.Sprintf("%d", effectiveHostID) {
			inst.AutomationInstanceID = fmt.Sprintf("%d", effectiveHostID)
			if name != "" {
				inst.Name = name
			}
			_ = s.vps.UpdateInstanceLocal(ctx, inst)
		}
		accessInfo = mergeAccessInfoWithCreds(inst.AccessInfoJSON, info, sysPwd, vncPwd)
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
	return nil
}

func mergeAccessInfo(existing string, info AutomationHostInfo) string {
	return mergeAccessInfoWithCreds(existing, info, "", "")
}

// mergeAccessInfoWithCreds merges provider-returned access fields with the
// durable (decrypted) credentials so a crash between CreateHost and the local
// host-id write cannot lose the system/VNC passwords.
func mergeAccessInfoWithCreds(existing string, info AutomationHostInfo, osPwd, vncPwd string) string {
	remoteIP := ""
	panelPwd := ""
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
			updated, err := s.orders.UpdateOrderStatusIfCurrent(ctx, order.ID, domain.OrderStatusPendingReview, []domain.OrderStatus{domain.OrderStatusProvisioning})
			if err != nil || !updated {
				return
			}
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
	// Transition atomically so a stale worker cannot overwrite a concurrent
	// terminal state (and so concurrent consumers do not both publish
	// order.completed / send notifications).
	current := order.Status
	updated, err := s.orders.UpdateOrderStatusIfCurrent(ctx, order.ID, status, []domain.OrderStatus{current})
	if err != nil || !updated {
		return
	}
	order.Status = status
	// A terminal lifecycle state releases the per-VPS reservation so the VPS is
	// available for the next exclusive operation.
	if status == domain.OrderStatusActive || status == domain.OrderStatusFailed {
		s.releaseVPSLifecycleLocks(ctx, order.ID)
	}
	if s.events != nil {
		switch status {
		case domain.OrderStatusActive:
			_, _ = s.events.Publish(ctx, order.ID, "order.completed", map[string]any{"status": status})
		case domain.OrderStatusFailed:
			_, _ = s.events.Publish(ctx, order.ID, "order.failed", map[string]any{"status": status})
		}
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
					// A `creating:<host>` marker with no local instance and no
					// durable provision job is a stranded pre-intent attempt
					// (legacy data, or a crash in a version before the intent
					// was persisted). Recover it by enqueuing a
					// create_host_sent-stage job so the watchdog reconciles by
					// hostname — never blindly re-issues CreateHost — and
					// fails the item after the provision budget elapses.
					if strings.HasPrefix(item.AutomationInstanceID, "creating:") {
						s.recoverStrandedCreatingItem(ctx, order, item)
					}
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

// recoverStrandedCreatingItem rescues a provisioning order item that carries a
// `creating:<host>` marker but no durable provision job (a state only reachable
// from data written before the durable provision intent existed). A fresh
// create_host_sent-stage job is enqueued so the watchdog reconciles by the
// deterministic hostname — never issuing a second CreateHost — and moves the
// item/order to a terminal state once the provision budget elapses.
func (s *OrderService) recoverStrandedCreatingItem(ctx context.Context, order domain.Order, item domain.OrderItem) {
	if s.provision == nil {
		return
	}
	if _, err := s.provision.GetProvisionJobByOrderItem(ctx, item.ID); err == nil {
		return // a job already exists; the watchdog will drive it
	}
	hostName := strings.TrimPrefix(item.AutomationInstanceID, "creating:")
	if strings.TrimSpace(hostName) == "" {
		hostName = fmt.Sprintf("ecs-%d-%d-%d", order.UserID, order.ID, item.ID)
	}
	job := domain.ProvisionJob{
		OrderID:     order.ID,
		OrderItemID: item.ID,
		HostName:    hostName,
		Status:      provisionJobStatusPending,
		Attempts:    1,
		NextRunAt:   time.Now(),
		Stage:       domain.ProvisionStageCreateHostSent,
		AttemptID:   fmt.Sprintf("recover-%d-%d", order.ID, item.ID),
	}
	_ = s.provision.CreateOrUpdateProvisionJob(ctx, &job)
}

func isVPSReadyStatus(status domain.VPSStatus) bool {
	switch status {
	case domain.VPSStatusRunning, domain.VPSStatusStopped, domain.VPSStatusRescue, domain.VPSStatusLocked, domain.VPSStatusExpiredLocked:
		return true
	default:
		return false
	}
}

// failProvisionItem marks a create item failed and publishes the failure event
// so consumers are notified exactly once per terminal failure.
func (s *OrderService) failProvisionItem(ctx context.Context, orderID, itemID int64, reason string) {
	_ = s.items.UpdateOrderItemStatus(ctx, itemID, domain.OrderItemStatusFailed)
	if s.events != nil {
		_, _ = s.events.Publish(ctx, orderID, "order.item.failed", map[string]any{"item_id": itemID, "reason": reason})
	}
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

// buildVPSLocalSnapshot derives the local VPS catalog snapshot for an order item
// so a completed provision persists the same package/spec the user purchased.
func (s *OrderService) buildVPSLocalSnapshot(ctx context.Context, userID int64, item domain.OrderItem) vpsLocalSnapshot {
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
		unitCore := plan.UnitCore
		unitMem := plan.UnitMem
		unitDisk := plan.UnitDisk
		unitBW := plan.UnitBW
		if s.pricer != nil && userID > 0 {
			if pricing, _, err := s.pricer.ResolvePackagePricing(ctx, userID, item.PackageID); err == nil {
				snap.MonthlyPrice = pricing.MonthlyPrice
				unitCore = pricing.UnitCore
				unitMem = pricing.UnitMem
				unitDisk = pricing.UnitDisk
				unitBW = pricing.UnitBW
			}
		}
		addon := int64(0)
		var spec CartSpec
		if err := json.Unmarshal([]byte(item.SpecJSON), &spec); err == nil {
			snap.CPU += spec.AddCores
			snap.MemoryGB += spec.AddMemGB
			snap.DiskGB += spec.AddDiskGB
			snap.BandwidthMB += spec.AddBWMbps
			addon = int64(spec.AddCores)*unitCore +
				int64(spec.AddMemGB)*unitMem +
				int64(spec.AddDiskGB)*unitDisk +
				int64(spec.AddBWMbps)*unitBW
		}
		snap.MonthlyPrice += addon
		if region, err := s.catalog.GetRegion(ctx, plan.RegionID); err == nil {
			snap.Region = region.Name
		}
	}
	return snap
}

// SetProvisionCipher wires the AES-GCM cipher used to encrypt the provision
// intent credentials at rest. Without a cipher the creds are kept in memory
// only (the durable intent still tracks stage/host for crash recovery, but the
// passwords are not persisted).
func (s *OrderService) SetProvisionCipher(cipher *cryptox.AESGCM) {
	s.provisionCipher = cipher
}

// SetRefundCleanupRepo wires the durable wallet-refund VPS cleanup outbox so
// the order-refund path (handleRefund → deleteVPSForRefund) enqueues a cleanup
// job before deleting the host. Without it, an inline delete failure fails the
// refund item and relies on a manual RetryProvision.
func (s *OrderService) SetRefundCleanupRepo(repo appports.RefundCleanupRepository) {
	s.cleanup = repo
}

// encryptProvisionCreds returns an encrypted payload of the system/VNC
// passwords, or "" when no cipher is configured (nothing is persisted then).
func (s *OrderService) encryptProvisionCreds(sysPwd, vncPwd string) string {
	if s.provisionCipher == nil || (sysPwd == "" && vncPwd == "") {
		return ""
	}
	b, err := json.Marshal(map[string]string{"sys_pwd": sysPwd, "vnc_pwd": vncPwd})
	if err != nil {
		return ""
	}
	out, err := s.provisionCipher.EncryptToString(b)
	if err != nil {
		return ""
	}
	return out
}

// decryptProvisionCreds returns the decrypted system/VNC passwords, or empty
// strings when no cipher is configured or the ciphertext cannot be decrypted.
func (s *OrderService) decryptProvisionCreds(enc string) (sysPwd, vncPwd string) {
	if s.provisionCipher == nil || enc == "" {
		return "", ""
	}
	b, err := s.provisionCipher.DecryptString(enc)
	if err != nil {
		return "", ""
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return "", ""
	}
	return m["sys_pwd"], m["vnc_pwd"]
}
