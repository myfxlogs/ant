package mthub

import (
	"context"
	"fmt"

	antv1 "alphaforge/gen/proto/ant/v1"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/usermgr"
)

// CloseOrder closes an existing position.
// Gate order matches PlaceOrder: killSwitch → ownership → idempotency → reconcile → rateLimit.
func (s *MtHubService) CloseOrder(ctx context.Context, accountID string, ticket int64, lots decimal.Decimal) error {
	if err := s.preCloseChecks(ctx, accountID, ticket, lots); err != nil {
		return err
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
		return fmt.Errorf("%w: %s", ErrReconciling, accountID)
	}
	if s.userLimiter != nil {
		uid := usermgr.GetUserID(ctx)
		if uid != "" && !s.userLimiter.AllowOrder(uid) {
			return ErrRateLimited
		}
	}

	if err := s.evaluateCloseGate(ctx, accountID, ticket, lots); err != nil {
		return err
	}

	exec := s.hub.Get(accountID)
	if exec == nil {
		if s.logger != nil {
			s.logger.Warn("CloseOrder: session not found", zap.String("accountID", accountID), zap.Int64("ticket", ticket))
		}
		return ErrSessionNotFound
	}

	if s.logger != nil {
		s.logger.Info("CloseOrder: calling executor", zap.String("accountID", accountID), zap.Int64("ticket", ticket), zap.String("lots", lots.String()))
	}

	if s.omsWriter != nil {
		pf := platform(accountID, s.hub)
		if err := s.omsWriter.InsertOrder(ctx, closeOrderID, accountID, pf, "",
			int16(OrderMarket), lots, decimal.Zero, decimal.Zero, decimal.Zero); err != nil {
			if s.logger != nil {
				s.logger.Warn("CloseOrder: OMS insert skipped", zap.Error(err))
			}
		} else {
			s.omsTransition(ctx, closeOrderID, accountID, OMSStateNew, OMSStateValidated)
			s.omsTransition(ctx, closeOrderID, accountID, OMSStateValidated, OMSStateRiskApproved)
			s.omsTransition(ctx, closeOrderID, accountID, OMSStateRiskApproved, OMSStateSubmitted)
		}
	}

	if err := exec.CloseOrder(ctx, ticket, lots); err != nil {
		s.postCloseFailure(ctx, closeOrderID, accountID, ticket, err)
		return err
	}

	s.postCloseSuccess(ctx, closeOrderID, accountID, ticket)
	return nil
}

func (s *MtHubService) preCloseChecks(ctx context.Context, accountID string, ticket int64, lots decimal.Decimal) error {
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
	if s.gate == nil || s.accountStateProvider == nil {
		return nil
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
		return fmt.Errorf("gate rejected close: %s", decision.GetReason())
	}
	return nil
}

func (s *MtHubService) postCloseFailure(ctx context.Context, closeOrderID, accountID string, ticket int64, err error) {
	if s.logger != nil {
		s.logger.Error("CloseOrder: executor failed", zap.Error(err), zap.String("accountID", accountID), zap.Int64("ticket", ticket))
	}
	if s.omsWriter != nil {
		s.omsTransition(ctx, closeOrderID, accountID, OMSStateSubmitted, OMSStateFailed)
	}
}

func (s *MtHubService) postCloseSuccess(ctx context.Context, closeOrderID, accountID string, ticket int64) {
	if s.logger != nil {
		s.logger.Info("CloseOrder: executor success", zap.String("accountID", accountID), zap.Int64("ticket", ticket))
	}
	if s.omsWriter != nil {
		s.omsTransition(ctx, closeOrderID, accountID, OMSStateSubmitted, OMSStateFilled)
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
