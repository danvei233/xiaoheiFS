package walletorder

import (
	"context"
	"log"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/safego"
	"xiaoheiplay/internal/pkg/scrub"
)

// StartCleanupWorker polls the refund-cleanup outbox and drives any pending
// host deletions until they are done or failed. It is safe to run on multiple
// instances: each job is claimed with a lease+fence so only one worker acts.
func (s *Service) StartCleanupWorker(ctx context.Context) {
	if s.cleanup == nil {
		return
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		safego.Run("wallet refund cleanup tick", func() { _ = s.ProcessCleanupJobs(ctx, 20) })
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// ProcessCleanupJobs processes up to limit due refund-cleanup jobs.
func (s *Service) ProcessCleanupJobs(ctx context.Context, limit int) error {
	if s.cleanup == nil {
		return appshared.ErrInvalidInput
	}
	if limit <= 0 {
		limit = 20
	}
	jobs, err := s.cleanup.ListDueRefundCleanupJobs(ctx, limit)
	if err != nil {
		return err
	}
	var firstErr error
	for _, job := range jobs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					safego.LogPanic("wallet refund cleanup job", r)
					if current, err := s.cleanup.GetRefundCleanupJobByWalletOrder(ctx, job.WalletOrderID); err == nil && current.LeaseOwner == s.workerID {
						current.Status = domain.RefundCleanupRetry
						current.LastErrorCode = "panic"
						current.LastError = scrub.SanitizeSummary("panic")
						current.NextRunAt = time.Now().Add(cleanupRetryDelay(current.Attempts))
						current.LeaseUntil = nil
						_, _ = s.cleanup.UpdateRefundCleanupJob(ctx, current)
					}
				}
			}()
			if err := s.handleCleanupJob(ctx, job); err != nil && firstErr == nil {
				firstErr = err
			}
		}()
	}
	return firstErr
}

// handleCleanupJob claims a refund-cleanup job and deletes the refunded host
// idempotently, then updates the local instance state. A host that is already
// gone is treated as success so retries converge. A failure to update local
// state after a successful external delete leaves the job pending so the next
// attempt reconciles the local state (DeleteHost is re-run idempotently).
func (s *Service) handleCleanupJob(ctx context.Context, job domain.RefundCleanupJob) error {
	if s.cleanup == nil || s.automation == nil {
		return appshared.ErrInvalidInput
	}
	claimed, ok, err := s.cleanup.ClaimRefundCleanupJob(ctx, job.ID, s.workerID, time.Now().Add(domain.ProvisionLeaseDuration))
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	job = claimed

	cli, err := s.automation.ClientForGoodsType(ctx, job.GoodsTypeID)
	if err != nil {
		s.retryCleanup(ctx, job, err)
		return err
	}
	err = cli.DeleteHost(ctx, job.HostID)
	if err != nil && !cleanupHostNotFound(err) {
		s.retryCleanup(ctx, job, err)
		return err
	}
	// External delete succeeded (or the host was already gone). Update the local
	// instance so the refunded VPS is no longer usable.
	if s.vps != nil {
		if uerr := s.vps.UpdateInstanceStatus(ctx, job.VPSID, domain.VPSStatusUnknown, 0); uerr != nil {
			s.retryCleanup(ctx, job, uerr)
			return uerr
		}
	}
	job.Status = domain.RefundCleanupDone
	job.LastErrorCode = ""
	job.LastError = ""
	job.LeaseUntil = nil
	_, _ = s.cleanup.UpdateRefundCleanupJob(ctx, job)
	return nil
}

// retryCleanup releases the lease and schedules the job for another attempt
// with bounded backoff, persisting only the stable error code and a sanitized
// summary (the detailed error goes to the log through the secret scrubber).
func (s *Service) retryCleanup(ctx context.Context, job domain.RefundCleanupJob, err error) {
	job.Status = domain.RefundCleanupRetry
	// The claim already counted this attempt; do not double-count.
	job.LastErrorCode = cleanupErrorCode(err)
	job.LastError = scrub.SanitizeSummary(job.LastErrorCode)
	job.NextRunAt = time.Now().Add(cleanupRetryDelay(job.Attempts))
	job.LeaseUntil = nil
	_, _ = s.cleanup.UpdateRefundCleanupJob(ctx, job)
	log.Printf("wallet_order.refund_cleanup_retry wallet_order_id=%d code=%s detail=%s", job.WalletOrderID, job.LastErrorCode, scrub.Redact(err.Error()))
}

// cleanupHostNotFound reports whether a DeleteHost error means the host is
// already gone — which is an idempotent success for a cleanup job.
func cleanupHostNotFound(err error) bool {
	return appshared.IsHostNotFoundError(err)
}

func cleanupRetryDelay(attempts int) time.Duration {
	if attempts < 0 {
		attempts = 0
	}
	switch {
	case attempts > 20:
		return 30 * time.Minute
	case attempts > 10:
		return 5 * time.Minute
	case attempts > 5:
		return 30 * time.Second
	default:
		return 5 * time.Second
	}
}

// enqueueRefundCleanup writes the durable cleanup outbox entry for a refund
// approval so a crash after the balance credit still deletes the host.
func (s *Service) enqueueRefundCleanup(ctx context.Context, order domain.WalletOrder) {
	if s.cleanup == nil || s.vps == nil {
		return
	}
	meta := parseJSON(order.MetaJSON)
	vpsID := getInt64(meta["vps_id"])
	if vpsID == 0 {
		return
	}
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return
	}
	job := domain.RefundCleanupJob{
		WalletOrderID: order.ID,
		UserID:        order.UserID,
		VPSID:         vpsID,
		HostID:        hostID,
		GoodsTypeID:   inst.GoodsTypeID,
		Status:        domain.RefundCleanupPending,
		NextRunAt:     time.Now(),
	}
	_ = s.cleanup.CreateRefundCleanupJob(ctx, &job)
}

// ListCleanupJobs exposes the cleanup outbox for admin reconciliation.
func (s *Service) ListCleanupJobs(ctx context.Context, limit, offset int) ([]domain.RefundCleanupJob, int, error) {
	if s.cleanup == nil {
		return nil, 0, appshared.ErrInvalidInput
	}
	return s.cleanup.ListRefundCleanupJobs(ctx, limit, offset)
}
