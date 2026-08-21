package mthub

import (
	"context"
	"fmt"

	antv1 "alphaforge/gen/proto/ant/v1"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/usermgr"
)

// closeOMSOrderID derives a deterministic UUID for a close order's OMS row from
// (accountID, ticket): the same pair always maps to the same row, so an
// idempotent re-close of the same ticket upserts the same row
// (CLOSE-ORDER-UUID fix; extracted so tests exercise the production derivation).
func closeOMSOrderID(closeOrderID string) string {
	return uuid.NewMD5(uuid.NameSpaceOID, []byte(closeOrderID)).String()
}

// CloseOrder closes an existing position.
// Gate order matches PlaceOrder: killSwitch → ownership → idempotency → reconcile → rateLimit.
func (s *MtHubService) CloseOrder(ctx context.Context, accountID string, ticket int64, lots decimal.Decimal) error {
	if err := s.preCloseChecks(ctx, accountID); err != nil {
		return preBrokerError(err)
	}

	closeOrderID := fmt.Sprintf("close-%s-%d", accountID, ticket)
	if s.idem != nil {
		isDup, _, err := s.idem.CheckAndSet(ctx, accountID, closeOrderID, ticket)
		if err != nil {
			return fmt.Errorf("idempotency check: %w", err)
		}
		if isDup {
			return nil
		}
		defer s.idem.DeleteKey(ctx, accountID, closeOrderID)
	}

	if s.reconcileGate != nil && !s.reconcileGate.CanAccept(accountID) {
		return preBrokerError(fmt.Errorf("%w: %s", ErrReconciling, accountID))
	}
	if s.userLimiter != nil {
		uid := usermgr.GetUserID(ctx)
		if uid != "" && !s.userLimiter.AllowOrder(uid) {
			return preBrokerError(ErrRateLimited)
		}
	}

	if err := s.evaluateCloseGate(ctx, accountID, ticket, lots); err != nil {
		return preBrokerError(err)
	}

	exec := s.hub.Get(accountID)
	if exec == nil {
		if s.logger != nil {
			s.logger.Warn("CloseOrder: session not found", zap.String("accountID", accountID), zap.Int64("ticket", ticket))
		}
		return preBrokerError(ErrSessionNotFound)
	}

	if s.logger != nil {
		s.logger.Info("CloseOrder: calling executor", zap.String("accountID", accountID), zap.Int64("ticket", ticket), zap.String("lots", lots.String()))
	}

	if s.omsWriter != nil {
		pf := platform(accountID, s.hub)
		omsOrderID := closeOMSOrderID(closeOrderID)
		if err := s.omsWriter.InsertOrder(ctx, omsOrderID, accountID, pf, "",
			int16(OrderMarket), lots, decimal.Zero, decimal.Zero, decimal.Zero, 0); err != nil {
			if s.logger != nil {
				s.logger.Warn("CloseOrder: OMS insert skipped", zap.Error(err))
			}
		} else {
			s.omsTransition(ctx, omsOrderID, accountID, OMSStateNew, OMSStateValidated)
			s.omsTransition(ctx, omsOrderID, accountID, OMSStateValidated, OMSStateRiskApproved)
			s.omsTransition(ctx, omsOrderID, accountID, OMSStateRiskApproved, OMSStateSubmitted)
		}
	}

	if err := exec.CloseOrder(ctx, ticket, lots); err != nil {
		s.postCloseFailure(ctx, closeOrderID, accountID, ticket, err)
		return brokerError(err)
	}

	s.postCloseSuccess(ctx, closeOrderID, accountID, ticket)
	return nil
}

func (s *MtHubService) preCloseChecks(ctx context.Context, accountID string) error {
	if s.killSwitch != nil && s.killSwitch.IsEngaged() {
		return ErrKillSwitchEngaged
	}
	if s.accountOwnerVerifier != nil {
		uid := usermgr.GetUserID(ctx)
		if uid == "" {
			return fmt.Errorf("unauthenticated: user ID required for order close")
		}
		owns, err := s.accountOwnerVerifier(ctx, uid, accountID)
		if err != nil {
			return fmt.Errorf("account ownership check: %w", err)
		}
		if !owns {
			return fmt.Errorf("%w: %s", ErrAccountNotOwned, accountID)
		}
	}
	return nil
}

func (s *MtHubService) evaluateCloseGate(ctx context.Context, accountID string, ticket int64, lots decimal.Decimal) error {
	if s.gate == nil {
		return wrapGateError("gate not configured: order rejected (fail-closed)")
	}
	if s.accountStateProvider == nil {
		return wrapGateError("account state provider not configured: order rejected (fail-closed)")
	}
	closeIntent := &antv1.OrderIntent{
		AccountId: accountID,
		Type:      "close",
		Volume:    lots.String(),
		Magic:     ticket,
		UserId:    usermgr.GetUserID(ctx),
		Source:    antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
	}
	state, stateErr := s.accountStateProvider(ctx, accountID)
	if stateErr != nil && s.logger != nil {
		s.logger.Warn("gate: account state fetch failed for close — fail-closed",
			zap.String("accountID", accountID), zap.Error(stateErr))
	}
	decision := s.gate.Evaluate(ctx, closeIntent, state)
	if !decision.GetAllow() {
		return wrapGateError("gate rejected close: " + decision.GetReason())
	}
	return nil
}

func (s *MtHubService) postCloseFailure(ctx context.Context, closeOrderID, accountID string, ticket int64, err error) {
	if s.logger != nil {
		s.logger.Error("CloseOrder: executor failed", zap.Error(err), zap.String("accountID", accountID), zap.Int64("ticket", ticket))
	}
	if s.omsWriter != nil {
		omsOrderID := closeOMSOrderID(closeOrderID)
		s.omsTransition(ctx, omsOrderID, accountID, OMSStateSubmitted, OMSStateFailed)
	}
}

func (s *MtHubService) postCloseSuccess(ctx context.Context, closeOrderID, accountID string, ticket int64) {
	if s.logger != nil {
		s.logger.Info("CloseOrder: executor success", zap.String("accountID", accountID), zap.Int64("ticket", ticket))
	}
	if s.omsWriter != nil {
		omsOrderID := closeOMSOrderID(closeOrderID)
		s.omsTransition(ctx, omsOrderID, accountID, OMSStateSubmitted, OMSStateFilled)
	}
	if s.eventStore != nil {
		ev := &TradeEvent{
			EventID:   fmt.Sprintf("close-%s-%d", accountID, ticket),
			EventType: TradeEventOrderFilled,
			AccountID: accountID, Ticket: ticket,
			ToState: string(OMSStateFilled), FromState: string(OMSStateSubmitted),
			Timestamp: Clk.Now(), Version: 1,
		}
		if err := s.eventStore.Publish(ctx, ev); err != nil && s.logger != nil {
			s.logger.Error("close order event publish failed", zap.Error(err),
				zap.String("accountID", accountID), zap.Int64("ticket", ticket))
		}
	}
}
