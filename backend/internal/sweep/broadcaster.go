package sweep

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
)

// ErrManualReview indicates a leg is in MANUAL_REVIEW state — the bundle
// should be marked MANUAL_REVIEW to stop retry cycles (D14).
var ErrManualReview = errors.New("MANUAL_REVIEW")

// Broadcaster broadcasts signed sweep bundles in leg order:
// delegate → confirm → transfer → confirm → undelegate → confirm.
// Each leg's status is tracked in sweep_logs for crash recovery (Q3).
type Broadcaster struct {
	tron           TronClientIface
	sweepRepo      SweepLogRepoIface
	addrRepo       AddrRepoIface
	adminRepo      AdminRepoIface
	log            *zap.Logger
	confirmTimeout time.Duration
}

func NewBroadcaster(
	tron TronClientIface,
	sweepRepo SweepLogRepoIface,
	addrRepo AddrRepoIface,
	adminRepo AdminRepoIface,
	log *zap.Logger,
) *Broadcaster {
	return &Broadcaster{
		tron:           tron,
		sweepRepo:      sweepRepo,
		addrRepo:       addrRepo,
		adminRepo:      adminRepo,
		log:            log,
		confirmTimeout: 120 * time.Second,
	}
}

// BroadcastBundle broadcasts a SignedSweepBundle in tx order: delegate→transfer→undelegate
// per address. Each tx is broadcast, then confirmed on chain before proceeding to the next.
// If a leg fails, the bundle stops and the remaining legs are left PENDING.
// The state machine (state.go) handles retry/recovery for failed or stuck legs.
//
// For single-address bundles: 3 txs (delegate, transfer, undelegate).
// For batch bundles: 3N txs (delegate0, transfer0, undelegate0, delegate1, ...).
// signed.Txs are matched to sweep_logs by leg_seq order (both are ordered identically).
func (b *Broadcaster) BroadcastBundle(ctx context.Context, signed *antv1.SignedSweepBundle) error {
	batchID, err := uuid.Parse(signed.BundleId)
	if err != nil {
		return fmt.Errorf("sweep broadcaster: parse batch_id: %w", err)
	}

	// F4: Load configurable confirmation timeout (default 120s).
	confirmTimeout := b.confirmTimeout
	if b.adminRepo != nil {
		if cfg, err := b.adminRepo.GetConfig(ctx, "sweep_confirm_timeout_seconds"); err == nil && cfg != nil && cfg.Value != "" {
			if n, err := strconv.Atoi(cfg.Value); err == nil && n > 0 {
				confirmTimeout = time.Duration(n) * time.Second
			}
		}
	}

	legs, err := b.sweepRepo.ListBatchLegs(ctx, batchID)
	if err != nil {
		return fmt.Errorf("sweep broadcaster: list batch legs: %w", err)
	}

	if len(signed.Txs) != len(legs) {
		return fmt.Errorf("sweep broadcaster: tx count mismatch: signed=%d legs=%d",
			len(signed.Txs), len(legs))
	}

	for i, signedTx := range signed.Txs {
		leg := &legs[i]

		if leg.Status == "DONE" {
			b.log.Info("sweep broadcaster: leg already DONE, skipping",
				zap.String("batch_id", batchID.String()),
				zap.Int("leg_seq", leg.LegSeq),
				zap.String("leg_type", leg.LegType))
			continue
		}

		// ADR §2.7: MANUAL_REVIEW means broadcast succeeded but final state is
		// unknown. Never re-broadcast — funds may have already moved.
		if leg.Status == "MANUAL_REVIEW" {
			b.log.Warn("sweep broadcaster: leg in MANUAL_REVIEW, halting bundle",
				zap.String("batch_id", batchID.String()),
				zap.Int("leg_seq", leg.LegSeq),
				zap.String("leg_type", leg.LegType),
				zap.String("tx_hash", leg.TxHash))
			return fmt.Errorf("sweep broadcaster: leg %s (seq %d) is MANUAL_REVIEW — requires human intervention: %w",
				leg.LegType, leg.LegSeq, ErrManualReview)
		}

		// ADR §2.7: "绝不盲目重新广播, 避免双花"
		// SWEEPING or FAILED legs with tx_hash must be chain-checked before re-broadcast.
		// SWEEPING: broadcast succeeded, confirmation timed out — tx may be on chain.
		// FAILED+tx_hash: on-chain execution appeared to fail, but TronGrid may have been
		// inconsistent — tx may actually be confirmed. Re-broadcasting without checking = double-spend.
		if (leg.Status == "SWEEPING" || leg.Status == "FAILED") && leg.TxHash != "" {
			confirmed, success, energyUsed, err := b.tron.GetTransactionInfo(ctx, leg.TxHash)
			if err != nil {
				b.log.Warn("sweep broadcaster: chain check error, skipping leg",
					zap.String("leg_type", leg.LegType),
					zap.String("tx_hash", leg.TxHash),
					zap.Error(err))
				return fmt.Errorf("sweep broadcaster: leg %s: chain check: %w", leg.LegType, err)
			}
			if confirmed {
				if success {
					b.markLegDone(ctx, leg)
					b.log.Info("sweep broadcaster: leg already confirmed on chain, marking DONE",
						zap.String("leg_type", leg.LegType),
						zap.String("tx_hash", leg.TxHash),
						zap.Int64("energy_used", energyUsed))
					continue
				}
				// Chain-confirmed FAILED: transition to MANUAL_REVIEW, not FAILED.
				// This stops the infinite retry loop (D6 halts on MANUAL_REVIEW),
				// prevents energy leak (operator manually undelegates), and
				// prevents double-sweep (D7 excludes MANUAL_REVIEW addresses).
				_ = b.sweepRepo.UpdateToManualReview(ctx, leg.ID,
					fmt.Sprintf("on-chain execution failed (tx=%s, energy=%d)", leg.TxHash, energyUsed))
				b.log.Warn("sweep broadcaster: leg confirmed FAILED on chain → MANUAL_REVIEW",
					zap.String("leg_type", leg.LegType),
					zap.String("tx_hash", leg.TxHash),
					zap.Int64("energy_used", energyUsed))
				return fmt.Errorf("sweep broadcaster: leg %s failed on chain → MANUAL_REVIEW: %w",
					leg.LegType, ErrManualReview)
			}
			// Not found on chain — safe to re-broadcast (tx expired or never accepted).
			b.log.Info("sweep broadcaster: tx not on chain, re-broadcasting",
				zap.String("leg_type", leg.LegType),
				zap.String("tx_hash", leg.TxHash))
		}

		if err := b.broadcastLeg(ctx, leg, signedTx, confirmTimeout); err != nil {
			return fmt.Errorf("sweep broadcaster: leg %s (seq %d): %w", leg.LegType, leg.LegSeq, err)
		}
	}

	b.log.Info("sweep bundle broadcast complete",
		zap.String("batch_id", batchID.String()),
		zap.Int("legs", len(signed.Txs)))

	return nil
}

func (b *Broadcaster) broadcastLeg(ctx context.Context, leg *model.SweepLog, signedTx *antv1.SignedTx, confirmTimeout time.Duration) error {
	// Transition to SWEEPING with tx_hash.
	// On recovery, leg may already be SWEEPING — use UpdateTxHash instead.
	if leg.Status == "SWEEPING" {
		if err := b.sweepRepo.UpdateTxHash(ctx, leg.ID, signedTx.TxHash); err != nil {
			return fmt.Errorf("update tx_hash: %w", err)
		}
	} else {
		if err := b.sweepRepo.UpdateToSweeping(ctx, leg.ID, signedTx.TxHash, 0); err != nil {
			return fmt.Errorf("update to sweeping: %w", err)
		}
	}

	// Broadcast the signed transaction.
	txid, err := b.tron.BroadcastSignedTx(ctx, signedTx.SignedTxData)
	if err != nil {
		_ = b.sweepRepo.UpdateToFailed(ctx, leg.ID, err.Error())
		return fmt.Errorf("broadcast: %w", err)
	}

	// D13: Verify the returned txid matches the expected hash from the builder.
	// If they differ, update the DB with the actual txid to ensure ReconfirmSweeping
	// queries the correct transaction on chain.
	if txid != signedTx.TxHash && txid != "" {
		b.log.Warn("sweep broadcaster: txid mismatch, updating DB with actual txid",
			zap.String("leg_type", leg.LegType),
			zap.String("expected_txid", signedTx.TxHash),
			zap.String("actual_txid", txid))
		if err := b.sweepRepo.UpdateTxHash(ctx, leg.ID, txid); err != nil {
			b.log.Error("sweep broadcaster: update tx_hash after mismatch", zap.Error(err))
		}
	}

	b.log.Info("sweep broadcaster: leg broadcast",
		zap.String("leg_type", leg.LegType),
		zap.String("txid", txid))

	// Wait for confirmation (configurable, default 120s per leg).
	confirmCtx, cancel := context.WithTimeout(ctx, confirmTimeout)
	defer cancel()

	success, energyUsed, err := b.tron.WaitForConfirmation(confirmCtx, txid, 3*time.Second)
	if err != nil {
		// Timeout — leave as SWEEPING for reconfirmation checker.
		b.log.Warn("sweep broadcaster: confirmation timeout, leaving SWEEPING",
			zap.String("leg_type", leg.LegType),
			zap.String("txid", txid),
			zap.Error(err))
		return fmt.Errorf("confirm timeout: %w", err)
	}

	if !success {
		// Chain-confirmed failure: go directly to MANUAL_REVIEW (not FAILED).
		// The tx is on chain with tx_hash set — re-broadcasting risks double-spend.
		// Operator must investigate (e.g. manually undelegate if delegate succeeded but transfer failed).
		_ = b.sweepRepo.UpdateToManualReview(ctx, leg.ID,
			fmt.Sprintf("on-chain execution failed (tx=%s)", txid))
		return fmt.Errorf("on-chain execution failed for %s → MANUAL_REVIEW: %w",
			leg.LegType, ErrManualReview)
	}

	// Mark DONE + MarkReceivedUSDT for transfer legs (ADR §2.7 step 7).
	b.markLegDone(ctx, leg)

	b.log.Info("sweep broadcaster: leg confirmed",
		zap.String("leg_type", leg.LegType),
		zap.String("txid", txid),
		zap.Int64("energy_used", energyUsed))

	return nil
}

// markLegDone transitions a leg to DONE and, for transfer legs, marks has_received_usdt
// on the deposit address (ADR §2.7 step 7). Centralizes the pattern that was
// duplicated across BroadcastBundle, broadcastLeg, and ReconfirmSweeping.
func (b *Broadcaster) markLegDone(ctx context.Context, leg *model.SweepLog) {
	if err := b.sweepRepo.UpdateToDone(ctx, leg.ID); err != nil {
		b.log.Error("sweep broadcaster: update to done",
			zap.String("leg_type", leg.LegType),
			zap.Error(err))
		return
	}
	if leg.LegType == "transfer" {
		if err := b.addrRepo.MarkReceivedUSDT(ctx, leg.DepositAddressID); err != nil {
			b.log.Error("sweep broadcaster: mark received usdt",
				zap.String("addr_id", leg.DepositAddressID.String()),
				zap.Error(err))
		}
	}
}
