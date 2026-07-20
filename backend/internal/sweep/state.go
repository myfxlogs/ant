package sweep

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/model"
)

// StateMachine handles sweep state transitions and double-spend prevention.
// It runs periodically to:
// 1. Reconfirm SWEEPING legs by checking chain state (ADR §2.7).
// 2. Mark stuck sweeps as FAILED or MANUAL_REVIEW.
// 3. Before any retry, verify no outgoing transfer exists on chain (防双花).
type StateMachine struct {
	tron      TronClientIface
	sweepRepo SweepLogRepoIface
	tronGrid  TronGridIface
	adminRepo AdminRepoIface
	addrRepo  AddrRepoIface
	log       *zap.Logger
}

func NewStateMachine(
	tron TronClientIface,
	sweepRepo SweepLogRepoIface,
	tronGrid TronGridIface,
	adminRepo AdminRepoIface,
	addrRepo AddrRepoIface,
	log *zap.Logger,
) *StateMachine {
	return &StateMachine{
		tron:      tron,
		sweepRepo: sweepRepo,
		tronGrid:  tronGrid,
		adminRepo: adminRepo,
		addrRepo:  addrRepo,
		log:       log,
	}
}

// ReconfirmSweeping checks all SWEEPING and FAILED legs with tx_hash against the chain.
// SUCCESS → DONE, FAILED → MANUAL_REVIEW, still not found → stays in current status.
// For SWEEPING legs: stays SWEEPING (BroadcastBundle will re-broadcast if still BROADCASTING).
// For FAILED legs: stays FAILED (BroadcastBundle will re-broadcast if still BROADCASTING,
// or address becomes re-eligible after 1h cooldown if bundle expired).
func (s *StateMachine) ReconfirmSweeping(ctx context.Context) error {
	legs, err := s.sweepRepo.ListSweepingWithTxHash(ctx)
	if err != nil {
		return fmt.Errorf("sweep state: list sweeping: %w", err)
	}

	for i := range legs {
		leg := &legs[i]
		confirmed, success, energyUsed, err := s.tron.GetTransactionInfo(ctx, leg.TxHash)
		if err != nil {
			s.log.Warn("sweep state: reconfirm error",
				zap.String("leg_type", leg.LegType),
				zap.String("tx_hash", leg.TxHash),
				zap.Error(err))
			continue
		}

		if !confirmed {
			s.log.Debug("sweep state: still unconfirmed",
				zap.String("leg_type", leg.LegType),
				zap.String("tx_hash", leg.TxHash))
			continue
		}

		if success {
			s.markLegDone(ctx, leg)
			s.log.Info("sweep state: leg confirmed DONE",
				zap.String("leg_type", leg.LegType),
				zap.String("tx_hash", leg.TxHash),
				zap.Int64("energy_used", energyUsed))
		} else {
			if err := s.sweepRepo.UpdateToManualReview(ctx, leg.ID,
				fmt.Sprintf("on-chain execution failed (reconfirm, tx=%s, energy=%d)", leg.TxHash, energyUsed)); err != nil {
				s.log.Error("sweep state: update to manual review",
					zap.String("leg_type", leg.LegType),
					zap.Error(err))
			}
			s.log.Warn("sweep state: leg confirmed FAILED → MANUAL_REVIEW",
				zap.String("leg_type", leg.LegType),
				zap.String("tx_hash", leg.TxHash))
		}
	}

	return nil
}

// MarkStuckSweeping transitions stuck SWEEPING/PENDING legs after a timeout.
// - SWEEPING with tx_hash → MANUAL_REVIEW (broadcast succeeded, funds may have moved)
// - SWEEPING without tx_hash → FAILED (stuck before broadcast)
// - PENDING → FAILED (stuck before broadcast)
func (s *StateMachine) MarkStuckSweeping(ctx context.Context) error {
	count, err := s.sweepRepo.MarkStuckSweepingAsFailed(ctx, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("sweep state: mark stuck: %w", err)
	}
	if count > 0 {
		s.log.Info("sweep state: marked stuck legs", zap.Int64("count", count))
	}
	return nil
}

// CheckDoubleSpend verifies that a deposit address has no unaccounted outgoing
// TRC20 transfer to the cold wallet before allowing a re-sweep (ADR §2.7 防双花).
// This prevents double-sweeping when DB write of tx_hash failed but the
// transaction was already broadcast successfully on chain.
//
// If a DONE transfer leg exists for this address, the outgoing transfer is expected
// (previous successful sweep) and does NOT block re-sweeping (ADR §2.7: "归集后地址保留").
// Only flag as double-spend if there's an outgoing transfer with no matching DONE leg.
func (s *StateMachine) CheckDoubleSpend(ctx context.Context, addrID uuid.UUID, fromAddr string) (bool, error) {
	// F5: Admin override — allows operator to bypass TronGrid check during outages.
	// Operator must manually verify no outgoing transfers exist before enabling.
	if cfg, err := s.adminRepo.GetConfig(ctx, "sweep_skip_doublecheck"); err == nil && cfg != nil && cfg.Value == "true" {
		s.log.Warn("sweep state: double-spend check SKIPPED by admin override (sweep_skip_doublecheck=true)",
			zap.String("from", fromAddr))
		return false, nil
	}

	cfg, err := s.adminRepo.GetConfig(ctx, "cold_wallet_address")
	if err != nil || cfg == nil || cfg.Value == "" {
		return false, fmt.Errorf("sweep state: cold_wallet_address not configured")
	}

	usdtCfg, err := s.adminRepo.GetConfig(ctx, "usdt_contract_address")
	if err != nil || usdtCfg == nil || usdtCfg.Value == "" {
		return false, fmt.Errorf("sweep state: usdt_contract_address not configured")
	}

	// Check if there's a DONE transfer leg — means a previous sweep succeeded.
	// The outgoing transfer on chain is expected and should not block re-sweeping.
	doneLeg, err := s.sweepRepo.GetLatestDoneTransferLeg(ctx, addrID)
	if err == nil && doneLeg != nil {
		doneAt := time.Time{}
		if doneLeg.CompletedAt != nil {
			doneAt = *doneLeg.CompletedAt
		}
		s.log.Debug("sweep state: DONE transfer leg exists, outgoing transfer expected",
			zap.String("from", fromAddr),
			zap.Time("done_at", doneAt))
		return false, nil
	}

	hasOutgoing, err := s.tronGrid.HasOutgoingTRC20Transfer(ctx, fromAddr, cfg.Value, usdtCfg.Value)
	if err != nil {
		return false, fmt.Errorf("sweep state: check outgoing transfers: %w", err)
	}

	if hasOutgoing {
		s.log.Warn("sweep state: double-spend prevention — unaccounted outgoing transfer detected",
			zap.String("from", fromAddr),
			zap.String("to", cfg.Value))
		return true, nil
	}

	return false, nil
}

// markLegDone transitions a leg to DONE and, for transfer legs, marks has_received_usdt
// on the deposit address (ADR §2.7 step 7). Shared logic for ReconfirmSweeping.
func (s *StateMachine) markLegDone(ctx context.Context, leg *model.SweepLog) {
	if err := s.sweepRepo.UpdateToDone(ctx, leg.ID); err != nil {
		s.log.Error("sweep state: update to done",
			zap.String("leg_type", leg.LegType),
			zap.Error(err))
		return
	}
	if leg.LegType == "transfer" {
		if err := s.addrRepo.MarkReceivedUSDT(ctx, leg.DepositAddressID); err != nil {
			s.log.Error("sweep state: mark received usdt",
				zap.String("addr_id", leg.DepositAddressID.String()),
				zap.Error(err))
		}
	}
}
