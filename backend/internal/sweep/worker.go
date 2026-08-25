package sweep

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
)

// Worker runs the periodic sweep lifecycle (crash recovery + state machine only):
// 1. Reconfirm SWEEPING legs (state machine).
// 2. Mark stuck legs as FAILED/MANUAL_REVIEW.
// 3. Expire stale PENDING_SIGN bundles (24h raw_tx expiry).
// 4. Resume broadcasting any persisted BROADCASTING bundles (crash recovery, Q3).
//
// Per ADR-0026 §10.4: sweep is admin-manual-trigger only. The worker does NOT
// auto-build unsigned bundles — that is done via ExportUnsignedBundle / admin RPC.
// The worker does NOT sign transactions — signing is done on the air-gapped
// coldsign machine. The worker only reconfirms, recovers, and broadcasts.
type Worker struct {
	builder     *Builder
	broadcaster *Broadcaster
	state       *StateMachine
	bundleRepo  *BundleRepository
	sweepRepo   *repository.SweepLogRepository
	depositRepo *repository.DepositRepository
	addrRepo    *repository.DepositAddressRepository
	adminRepo   *repository.AdminRepository
	pool        *pgxpool.Pool
	log         *zap.Logger

	scanInterval time.Duration
}

func NewWorker(
	builder *Builder,
	broadcaster *Broadcaster,
	state *StateMachine,
	bundleRepo *BundleRepository,
	sweepRepo *repository.SweepLogRepository,
	depositRepo *repository.DepositRepository,
	addrRepo *repository.DepositAddressRepository,
	adminRepo *repository.AdminRepository,
	pool *pgxpool.Pool,
	log *zap.Logger,
) *Worker {
	return &Worker{
		builder:      builder,
		broadcaster:  broadcaster,
		state:        state,
		bundleRepo:   bundleRepo,
		sweepRepo:    sweepRepo,
		depositRepo:  depositRepo,
		addrRepo:     addrRepo,
		adminRepo:    adminRepo,
		pool:         pool,
		log:          log,
		scanInterval: 30 * time.Second,
	}
}

// sweepAdvisoryLockID is a fixed constant for PG advisory lock (ADR-0026 R6).
// Distinct from chain.Monitor's advisoryLockID to avoid conflict.
const sweepAdvisoryLockID = int64(20260720)

// Run starts the periodic sweep worker loop.
// Acquires a PG advisory lock to ensure single-instance execution (R6).
func (w *Worker) Run(ctx context.Context) error {
	conn, err := w.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("sweep worker: acquire advisory lock conn: %w", err)
	}
	defer conn.Release()

	var locked bool
	err = conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", sweepAdvisoryLockID).Scan(&locked)
	if err != nil {
		return fmt.Errorf("sweep worker: advisory lock query: %w", err)
	}
	if !locked {
		w.log.Warn("sweep worker: another instance holds the advisory lock, not starting")
		return nil
	}
	defer func() {
		if _, err := conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", sweepAdvisoryLockID); err != nil {
			w.log.Error("sweep worker: release advisory lock", zap.Error(err))
		}
	}()

	w.log.Info("sweep worker started (advisory lock acquired)")

	scanInterval := w.scanInterval
	if cfg, err := w.adminRepo.GetConfig(ctx, "sweep_scan_interval_seconds"); err == nil && cfg != nil && cfg.Value != "" {
		if n, err := strconv.Atoi(cfg.Value); err == nil && n > 0 {
			scanInterval = time.Duration(n) * time.Second
		}
	}

	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	// Run one cycle immediately on startup for crash recovery.
	if err := w.runCycle(ctx); err != nil {
		w.log.Error("sweep worker: initial cycle", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			w.log.Info("sweep worker stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := w.runCycle(ctx); err != nil {
				w.log.Error("sweep worker: cycle", zap.Error(err))
			}
		}
	}
}

func (w *Worker) runCycle(ctx context.Context) error {
	// 1. Reconfirm SWEEPING legs.
	if err := w.state.ReconfirmSweeping(ctx); err != nil {
		w.log.Error("sweep worker: reconfirm", zap.Error(err))
	}

	// 2. Mark stuck legs.
	if err := w.state.MarkStuckSweeping(ctx); err != nil {
		w.log.Error("sweep worker: mark stuck", zap.Error(err))
	}

	// 3. Expire stale PENDING_SIGN bundles and free their addresses (D1).
	if err := w.expireStalePendingSign(ctx); err != nil {
		w.log.Error("sweep worker: expire stale pending sign", zap.Error(err))
	}

	// 4. Resume broadcasting persisted bundles (crash recovery, Q3).
	if err := w.resumeBroadcasting(ctx); err != nil {
		w.log.Error("sweep worker: resume broadcasting", zap.Error(err))
	}

	// No automatic bundle building (ADR-0026 §10.4).
	// Sweep is admin-manual-trigger only — see ExportUnsignedBundle / ExportBatchUnsignedBundle.

	return nil
}

// expireStalePendingSign marks PENDING_SIGN bundles older than 24h as EXPIRED
// and fails their PENDING legs, freeing addresses for re-sweeping (D1).
// 24h matches the raw_tx expiry window — if cold signing hasn't happened by then,
// the unsigned transactions are no longer valid on chain anyway.
func (w *Worker) expireStalePendingSign(ctx context.Context) error {
	pendingSignExpiry := 24 * time.Hour
	if cfg, err := w.adminRepo.GetConfig(ctx, "sweep_pending_sign_expiry_hours"); err == nil && cfg != nil && cfg.Value != "" {
		if n, err := strconv.Atoi(cfg.Value); err == nil && n > 0 {
			pendingSignExpiry = time.Duration(n) * time.Hour
		}
	}
	batchIDs, err := w.bundleRepo.ExpireStalePendingSign(ctx, pendingSignExpiry)
	if err != nil {
		return fmt.Errorf("expire stale pending sign: %w", err)
	}
	for _, bid := range batchIDs {
		if err := w.sweepRepo.FailLegsByBatch(ctx, bid, "unsigned bundle expired (cold signing window exceeded 24h)"); err != nil {
			w.log.Error("sweep worker: fail legs for expired bundle",
				zap.String("batch_id", bid.String()),
				zap.Error(err))
		}
		w.log.Info("sweep worker: expired stale PENDING_SIGN bundle",
			zap.String("batch_id", bid.String()))
	}
	return nil
}

// resumeBroadcasting reads back BROADCASTING bundles and continues broadcasting
// from the first unconfirmed leg. Re-broadcast needs no private key (Q3).
func (w *Worker) resumeBroadcasting(ctx context.Context) error {
	bundles, err := w.bundleRepo.ListBroadcastingBundles(ctx)
	if err != nil {
		return fmt.Errorf("resume broadcasting: list: %w", err)
	}

	// Load configurable raw_tx expiry for IsExpired check.
	rawTxExpiryHours := 23
	if cfg, err := w.adminRepo.GetConfig(ctx, "sweep_raw_tx_expiry_hours"); err == nil && cfg != nil && cfg.Value != "" {
		if n, err := strconv.Atoi(cfg.Value); err == nil && n > 0 {
			rawTxExpiryHours = n
		}
	}

	for _, bb := range bundles {
		if bb.IsExpired(rawTxExpiryHours) {
			w.log.Warn("sweep worker: bundle expired, marking EXPIRED",
				zap.String("batch_id", bb.BatchID.String()))
			_ = w.bundleRepo.MarkBundleExpired(ctx, bb.BatchID)
			continue
		}

		signed, err := w.bundleRepo.GetBundle(ctx, bb.BatchID)
		if err != nil {
			w.log.Error("sweep worker: get bundle for resume",
				zap.String("batch_id", bb.BatchID.String()),
				zap.Error(err))
			continue
		}

		if err := w.broadcaster.BroadcastBundle(ctx, signed); err != nil {
			if errors.Is(err, ErrManualReview) {
				_ = w.bundleRepo.MarkBundleManualReview(ctx, bb.BatchID)
				w.log.Warn("sweep worker: bundle marked MANUAL_REVIEW",
					zap.String("batch_id", bb.BatchID.String()))
			} else {
				w.log.Error("sweep worker: resume broadcast",
					zap.String("batch_id", bb.BatchID.String()),
					zap.Error(err))
			}
			continue
		}

		_ = w.bundleRepo.MarkBundleDone(ctx, bb.BatchID)
		w.log.Info("sweep worker: resumed bundle complete",
			zap.String("batch_id", bb.BatchID.String()))
	}

	return nil
}
