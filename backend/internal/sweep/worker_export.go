package sweep

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
	"alphaforge/internal/repository"
)

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
