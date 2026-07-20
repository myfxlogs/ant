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

// Worker runs the periodic sweep lifecycle:
// 1. Reconfirm SWEEPING legs (state machine).
// 2. Mark stuck legs as FAILED/MANUAL_REVIEW.
// 3. Resume broadcasting any persisted BROADCASTING bundles (crash recovery, Q3).
// 4. Build unsigned bundles for new unswept addresses (export for cold signing).
//
// The worker does NOT sign transactions — signing is done on the air-gapped
// coldsign machine. The worker only builds, exports, and broadcasts.
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

	ticker := time.NewTicker(w.scanInterval)
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

	// 5. Build unsigned bundles for unswept addresses (export for cold signing).
	if err := w.buildPendingBundles(ctx); err != nil {
		w.log.Error("sweep worker: build pending", zap.Error(err))
	}

	return nil
}

// expireStalePendingSign marks PENDING_SIGN bundles older than 24h as EXPIRED
// and fails their PENDING legs, freeing addresses for re-sweeping (D1).
// 24h matches the raw_tx expiry window — if cold signing hasn't happened by then,
// the unsigned transactions are no longer valid on chain anyway.
func (w *Worker) expireStalePendingSign(ctx context.Context) error {
	batchIDs, err := w.bundleRepo.ExpireStalePendingSign(ctx, 24*time.Hour)
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

	for _, bb := range bundles {
		if bb.IsExpired() {
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

// buildPendingBundles finds unswept addresses and builds unsigned bundles
// for export to the cold signing machine. Does NOT broadcast — requires cold signing first.
// Uses batch builder to group all eligible addresses into a single cold-sign session (ADR §2.7).
func (w *Worker) buildPendingBundles(ctx context.Context) error {
	threshold := "0.01"
	batchSize := 10

	if cfg, err := w.adminRepo.GetConfig(ctx, "sweep_threshold"); err == nil && cfg != nil && cfg.Value != "" {
		threshold = cfg.Value
	}
	if cfg, err := w.adminRepo.GetConfig(ctx, "sweep_batch_size"); err == nil && cfg != nil && cfg.Value != "" {
		if n, err := strconv.Atoi(cfg.Value); err == nil && n > 0 {
			batchSize = n
		}
	}

	unswept, err := w.depositRepo.ListUnsweptAddresses(ctx, threshold, batchSize)
	if err != nil {
		return fmt.Errorf("build pending: list unswept: %w", err)
	}
	if len(unswept) == 0 {
		return nil
	}

	// D4: Skip addresses that already have a PENDING_SIGN bundle awaiting cold signing.
	pendingSignAddrs, err := w.bundleRepo.ListPendingSignAddrIDs(ctx)
	if err != nil {
		w.log.Warn("sweep worker: list pending sign addr ids, proceeding without skip", zap.Error(err))
		pendingSignAddrs = make(map[uuid.UUID]bool)
	}

	// Build entries for batch bundle, filtering out skipped addresses and checking double-spend.
	var entries []BatchSweepEntry
	for _, u := range unswept {
		if pendingSignAddrs[u.AddrID] {
			w.log.Debug("sweep worker: skipping address — PENDING_SIGN bundle exists",
				zap.String("addr_id", u.AddrID.String()))
			continue
		}

		addr, err := w.addrRepo.GetByID(ctx, u.AddrID)
		if err != nil {
			w.log.Error("sweep worker: get address for build",
				zap.String("addr_id", u.AddrID.String()),
				zap.Error(err))
			continue
		}
		if addr == nil {
			w.log.Warn("sweep worker: address not found",
				zap.String("addr_id", u.AddrID.String()))
			continue
		}

		hasOutgoing, err := w.state.CheckDoubleSpend(ctx, addr.ID, addr.Address)
		if err != nil {
			w.log.Error("sweep worker: double-spend check",
				zap.String("address", addr.Address),
				zap.Error(err))
			continue
		}
		if hasOutgoing {
			w.log.Warn("sweep worker: skipping address — outgoing transfer detected (double-spend prevention)",
				zap.String("address", addr.Address))
			continue
		}

		entries = append(entries, BatchSweepEntry{Addr: addr, Amount: u.Amount})
	}

	if len(entries) == 0 {
		return nil
	}

	// Build a single batch bundle for all eligible addresses (ADR §2.7: "批量归集可一次委托整批").
	bundle, err := w.builder.BuildBatchUnsignedBundle(ctx, entries)
	if err != nil {
		w.log.Error("sweep worker: batch build failed, falling back to individual bundles",
			zap.Error(err))
		return w.buildIndividualBundles(ctx, entries)
	}

	batchID, err := uuid.Parse(bundle.BundleId)
	if err != nil {
		return fmt.Errorf("sweep worker: parse batch_id from bundle: %w", err)
	}
	if err := w.saveBatchBundleAndLegs(ctx, batchID, entries, bundle); err != nil {
		w.log.Error("sweep worker: save batch bundle and legs",
			zap.String("batch_id", bundle.BundleId),
			zap.Error(err))
		return err
	}

	w.log.Info("sweep worker: batch unsigned bundle ready for cold signing",
		zap.String("batch_id", bundle.BundleId),
		zap.Int("addresses", len(entries)),
		zap.Int("txs", len(bundle.Txs)))

	return nil
}

// buildIndividualBundles is a fallback when batch build fails — builds one bundle per address.
func (w *Worker) buildIndividualBundles(ctx context.Context, entries []BatchSweepEntry) error {
	for _, e := range entries {
		bundle, err := w.builder.BuildUnsignedBundle(ctx, e.Addr, e.Amount)
		if err != nil {
			w.log.Error("sweep worker: individual build",
				zap.String("address", e.Addr.Address),
				zap.Error(err))
			continue
		}

		batchID, perr := uuid.Parse(bundle.BundleId)
		if perr != nil {
			w.log.Error("sweep worker: parse batch_id from individual bundle",
				zap.String("bundle_id", bundle.BundleId), zap.Error(perr))
			continue
		}
		if err := w.saveBundleAndLegs(ctx, batchID, e.Addr.ID, bundle, e.Amount); err != nil {
			w.log.Error("sweep worker: save individual bundle and legs",
				zap.String("batch_id", bundle.BundleId),
				zap.Error(err))
			continue
		}

		w.log.Info("sweep worker: unsigned bundle ready for cold signing",
			zap.String("batch_id", bundle.BundleId),
			zap.String("address", e.Addr.Address),
			zap.String("amount", e.Amount))
	}
	return nil
}

// saveBatchBundleAndLegs creates sweep_log legs for all addresses in a batch and persists
// the unsigned bundle in a single DB transaction (D3, ADR Q3).
func (w *Worker) saveBatchBundleAndLegs(ctx context.Context, batchID uuid.UUID, entries []BatchSweepEntry, bundle *antv1.UnsignedSweepBundle) error {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("save batch bundle and legs: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

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

	bundleRepoTx := NewBundleRepository(tx)
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
