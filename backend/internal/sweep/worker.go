package sweep

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
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

// ExportBatchUnsignedBundle builds a batch unsigned bundle for multiple addresses.
// Admin RPC entry point for manual batch sweep initiation (ADR §2.7: "批量归集可一次委托整批").
// Each address is double-spend checked; eligible addresses are grouped into a single
// cold-sign session to minimize USB round-trips.
func (w *Worker) ExportBatchUnsignedBundle(ctx context.Context, addrIDs []uuid.UUID) (*antv1.UnsignedSweepBundle, error) {
	if len(addrIDs) == 0 {
		return nil, fmt.Errorf("sweep worker: batch export: empty address list")
	}

	var entries []BatchSweepEntry
	var checkedIDs []uuid.UUID
	for _, id := range addrIDs {
		addr, err := w.addrRepo.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("sweep worker: batch export: address lookup %s: %w", id, err)
		}
		if addr == nil {
			return nil, fmt.Errorf("sweep worker: batch export: address not found: %s", id)
		}

		amount, err := w.depositRepo.GetUnsweptBalance(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("sweep worker: batch export: get unswept balance %s: %w", id, err)
		}
		if amount == "" {
			w.log.Warn("sweep worker: batch export: skipping address — no unswept balance",
				zap.String("addr_id", id.String()))
			continue
		}

		checkedIDs = append(checkedIDs, id)
		entries = append(entries, BatchSweepEntry{Addr: addr, Amount: amount})
	}

	// D4 pre-filter: skip addresses with existing PENDING_SIGN bundles to avoid
	// unnecessary tx building. The authoritative check is inside saveBatchBundleAndLegs
	// transaction to eliminate TOCTOU race.
	if len(checkedIDs) > 0 {
		pendingIDs, err := w.bundleRepo.HasPendingSignBundle(ctx, checkedIDs)
		if err != nil {
			return nil, fmt.Errorf("sweep worker: batch export: check pending sign: %w", err)
		}
		if len(pendingIDs) > 0 {
			pendingSet := make(map[uuid.UUID]bool, len(pendingIDs))
			for _, pid := range pendingIDs {
				pendingSet[pid] = true
			}
			filtered := entries[:0]
			for _, e := range entries {
				if pendingSet[e.Addr.ID] {
					w.log.Warn("sweep worker: batch export: skipping address — PENDING_SIGN bundle exists",
						zap.String("addr_id", e.Addr.ID.String()))
					continue
				}
				filtered = append(filtered, e)
			}
			entries = filtered
		}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("sweep worker: batch export: no eligible addresses after filtering")
	}

	// Double-spend check for each eligible address — filter out flagged addresses.
	dsFiltered := entries[:0]
	for _, e := range entries {
		hasOutgoing, err := w.state.CheckDoubleSpend(ctx, e.Addr.ID, e.Addr.Address)
		if err != nil {
			return nil, fmt.Errorf("sweep worker: batch export: double-spend check %s: %w", e.Addr.ID, err)
		}
		if hasOutgoing {
			w.log.Warn("sweep worker: batch export: skipping address — outgoing transfer detected",
				zap.String("addr_id", e.Addr.ID.String()),
				zap.String("address", e.Addr.Address))
			continue
		}
		dsFiltered = append(dsFiltered, e)
	}
	entries = dsFiltered

	if len(entries) == 0 {
		return nil, fmt.Errorf("sweep worker: batch export: no eligible addresses after double-spend filter")
	}

	bundle, err := w.builder.BuildBatchUnsignedBundle(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("sweep worker: batch export: build: %w", err)
	}

	batchID, err := uuid.Parse(bundle.BundleId)
	if err != nil {
		return nil, fmt.Errorf("sweep worker: batch export: parse batch_id: %w", err)
	}

	if err := w.saveBatchBundleAndLegs(ctx, batchID, entries, bundle); err != nil {
		return nil, fmt.Errorf("sweep worker: batch export: save: %w", err)
	}

	w.log.Info("sweep worker: batch unsigned bundle ready for cold signing",
		zap.String("batch_id", bundle.BundleId),
		zap.Int("addresses", len(entries)),
		zap.Int("txs", len(bundle.Txs)))

	return bundle, nil
}

// saveBatchBundleAndLegs creates sweep_log legs for all addresses in a batch and persists
// the unsigned bundle in a single DB transaction (D3, ADR Q3).
func (w *Worker) saveBatchBundleAndLegs(ctx context.Context, batchID uuid.UUID, entries []BatchSweepEntry, bundle *antv1.UnsignedSweepBundle) error {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("save batch bundle and legs: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// D4 (authoritative): Check for PENDING_SIGN bundles inside the transaction
	// to eliminate TOCTOU race — two concurrent admin RPCs could both pass a
	// pre-tx check and create duplicate PENDING_SIGN bundles for the same address.
	bundleRepoTx := NewBundleRepository(tx)
	addrIDs := make([]uuid.UUID, len(entries))
	for i, e := range entries {
		addrIDs[i] = e.Addr.ID
	}
	pendingIDs, err := bundleRepoTx.HasPendingSignBundle(ctx, addrIDs)
	if err != nil {
		return fmt.Errorf("save batch bundle and legs: check pending sign: %w", err)
	}
	if len(pendingIDs) > 0 {
		idStrs := make([]string, len(pendingIDs))
		for i, id := range pendingIDs {
			idStrs[i] = id.String()
		}
		return fmt.Errorf("save batch bundle and legs: addresses already have PENDING_SIGN bundles: %v", idStrs)
	}

	sweepRepoTx := repository.NewSweepLogRepository(tx)
	legSeq := 0
	for _, e := range entries {
		legTypes := []string{"delegate", "transfer", "undelegate"}
		for _, legType := range legTypes {
			legAmount := "0"
			if legType == "transfer" {
				legAmount = e.Amount
			}
			if _, err := sweepRepoTx.CreateLeg(ctx, batchID, e.Addr.ID, legType, legSeq, legAmount); err != nil {
				return fmt.Errorf("save batch bundle and legs: create leg %s seq %d: %w", legType, legSeq, err)
			}
			legSeq++
		}
	}

	// Batch bundles have no single deposit_address_id — pass first entry's addr for compatibility.
	var firstAddrID uuid.UUID
	if len(entries) > 0 {
		firstAddrID = entries[0].Addr.ID
	}
	if err := bundleRepoTx.SaveUnsignedBundle(ctx, batchID, firstAddrID, bundle, time.Now().UnixMilli()); err != nil {
		return fmt.Errorf("save batch bundle and legs: save unsigned bundle: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("save batch bundle and legs: commit: %w", err)
	}
	return nil
}

// saveBundleAndLegs creates sweep_log legs and persists the unsigned bundle in a single
// DB transaction for atomicity (D3, ADR Q3). It is the single-address case of
// saveBatchBundleAndLegs (N=1) — delegates to avoid duplicating transaction logic.
func (w *Worker) saveBundleAndLegs(ctx context.Context, batchID, addrID uuid.UUID, bundle *antv1.UnsignedSweepBundle, amount string) error {
	entries := []BatchSweepEntry{
		{Addr: &model.DepositAddress{ID: addrID}, Amount: amount},
	}
	return w.saveBatchBundleAndLegs(ctx, batchID, entries, bundle)
}

// saveUndelegateOnlyBundleAndLegs persists an undelegate-only bundle with one leg per address.
// Unlike saveBatchBundleAndLegs which creates 3 legs (delegate/transfer/undelegate) per address,
// this creates only 1 undelegate leg per address — matching the bundle's tx count.
// No D4 PENDING_SIGN check: undelegate is chain-idempotent (second undelegate fails harmlessly)
// and does not transfer USDT, so duplicate bundles pose no double-spend risk.
func (w *Worker) saveUndelegateOnlyBundleAndLegs(ctx context.Context, batchID uuid.UUID, entries []BatchSweepEntry, bundle *antv1.UnsignedSweepBundle) error {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("save undelegate-only: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sweepRepoTx := repository.NewSweepLogRepository(tx)
	legSeq := 0
	for _, e := range entries {
		if _, err := sweepRepoTx.CreateLeg(ctx, batchID, e.Addr.ID, "undelegate", legSeq, "0"); err != nil {
			return fmt.Errorf("save undelegate-only: create leg seq %d: %w", legSeq, err)
		}
		legSeq++
	}

	bundleRepoTx := NewBundleRepository(tx)
	var firstAddrID uuid.UUID
	if len(entries) > 0 {
		firstAddrID = entries[0].Addr.ID
	}
	if err := bundleRepoTx.SaveUnsignedBundle(ctx, batchID, firstAddrID, bundle, time.Now().UnixMilli()); err != nil {
		return fmt.Errorf("save undelegate-only: save bundle: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("save undelegate-only: commit: %w", err)
	}
	return nil
}
