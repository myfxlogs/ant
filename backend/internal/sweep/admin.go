package sweep

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// ListPendingSignBundlesForAdmin returns all PENDING_SIGN bundle summaries for the admin RPC.
func (w *Worker) ListPendingSignBundlesForAdmin(ctx context.Context) ([]PendingSignBundleSummary, error) {
	return w.bundleRepo.ListPendingSignBundlesForAdmin(ctx)
}

// ImportSignedBundle imports a SignedSweepBundle from the cold signing machine
// and broadcasts it. This is the admin RPC entry point for the sweep workflow:
// 1. Online worker builds UnsignedSweepBundle → exported to USB.
// 2. Cold signing machine signs → SignedSweepBundle on USB.
// 3. Admin imports SignedSweepBundle via this method → broadcast.
func (w *Worker) ImportSignedBundle(ctx context.Context, signed *antv1.SignedSweepBundle) error {
	batchID, err := uuid.Parse(signed.BundleId)
	if err != nil {
		return fmt.Errorf("sweep worker: import: parse batch_id: %w", err)
	}

	if err := w.bundleRepo.SaveBundle(ctx, batchID, signed, time.Now().UnixMilli()); err != nil {
		return fmt.Errorf("sweep worker: import: save bundle: %w", err)
	}

	if err := w.broadcaster.BroadcastBundle(ctx, signed); err != nil {
		if errors.Is(err, ErrManualReview) {
			_ = w.bundleRepo.MarkBundleManualReview(ctx, batchID)
		}
		return fmt.Errorf("sweep worker: import: broadcast: %w", err)
	}

	_ = w.bundleRepo.MarkBundleDone(ctx, batchID)

	w.log.Info("sweep worker: signed bundle imported and broadcast",
		zap.String("batch_id", signed.BundleId))
	return nil
}

// ExportUnsignedBundle builds an unsigned bundle for a specific address.
// Admin RPC entry point for manual sweep initiation.
func (w *Worker) ExportUnsignedBundle(ctx context.Context, addrID uuid.UUID) (*antv1.UnsignedSweepBundle, error) {
	addr, err := w.addrRepo.GetByID(ctx, addrID)
	if err != nil {
		return nil, fmt.Errorf("sweep worker: export: address lookup: %w", err)
	}
	if addr == nil {
		return nil, fmt.Errorf("sweep worker: export: address not found")
	}

	amount, err := w.depositRepo.GetUnsweptBalance(ctx, addrID)
	if err != nil {
		return nil, fmt.Errorf("sweep worker: export: get unswept balance: %w", err)
	}
	if amount == "" {
		return nil, fmt.Errorf("sweep worker: export: no unswept balance for address")
	}

	hasOutgoing, err := w.state.CheckDoubleSpend(ctx, addr.ID, addr.Address)
	if err != nil {
		return nil, fmt.Errorf("sweep worker: export: double-spend check: %w", err)
	}
	if hasOutgoing {
		return nil, fmt.Errorf("sweep worker: export: outgoing transfer detected (double-spend prevention)")
	}

	bundle, err := w.builder.BuildUnsignedBundle(ctx, addr, amount)
	if err != nil {
		return nil, fmt.Errorf("sweep worker: export: build: %w", err)
	}

	batchID, err := uuid.Parse(bundle.BundleId)
	if err != nil {
		return nil, fmt.Errorf("sweep worker: export: parse batch_id: %w", err)
	}

	if err := w.saveBundleAndLegs(ctx, batchID, addr.ID, bundle, amount); err != nil {
		return nil, fmt.Errorf("sweep worker: export: %w", err)
	}

	return bundle, nil
}

// SweepDashboardEntry is a single row for the admin dashboard (C4).
type SweepDashboardEntry struct {
	AddrID          uuid.UUID
	Address         string
	DerivationIndex int32
	UnsweptAmount   string
	SweepStatus     string
}

// GetSweepDashboard returns addresses with unswept balances sorted descending,
// total unswept, and the current threshold (C4).
func (w *Worker) GetSweepDashboard(ctx context.Context, page, pageSize int) ([]SweepDashboardEntry, int64, string, string, error) {
	rows, total, totalUnswept, err := w.depositRepo.ListSweepDashboard(ctx, page, pageSize)
	if err != nil {
		return nil, 0, "", "", fmt.Errorf("sweep worker: dashboard: %w", err)
	}

	threshold := "0.01"
	if cfg, err := w.adminRepo.GetConfig(ctx, "sweep_threshold"); err == nil && cfg != nil && cfg.Value != "" {
		threshold = cfg.Value
	}

	entries := make([]SweepDashboardEntry, len(rows))
	for i, row := range rows {
		entries[i] = SweepDashboardEntry{
			AddrID:          row.AddrID,
			Address:         row.Address,
			DerivationIndex: row.DerivationIndex,
			UnsweptAmount:   row.UnsweptAmount,
			SweepStatus:     row.SweepStatus,
		}
	}

	return entries, total, totalUnswept, threshold, nil
}

// BuildUndelegateOnlyBundle constructs an undelegate-only bundle for the given
// address IDs (C5: energy recovery from stuck MANUAL_REVIEW addresses).
func (w *Worker) BuildUndelegateOnlyBundle(ctx context.Context, addrIDs []uuid.UUID) (*antv1.UnsignedSweepBundle, error) {
	var entries []BatchSweepEntry
	for _, id := range addrIDs {
		addr, err := w.addrRepo.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("sweep worker: undelegate-only: address lookup %s: %w", id, err)
		}
		if addr == nil {
			return nil, fmt.Errorf("sweep worker: undelegate-only: address not found: %s", id)
		}
		entries = append(entries, BatchSweepEntry{Addr: addr, Amount: "0"})
	}

	bundle, err := w.builder.BuildUndelegateOnlyBundle(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("sweep worker: undelegate-only: build: %w", err)
	}

	return bundle, nil
}
