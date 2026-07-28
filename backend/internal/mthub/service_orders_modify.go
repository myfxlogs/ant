// service_orders_modify.go — ModifyOrder wraps the MT gateway OrderModify RPC.

package mthub

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/usermgr"
)

// ModifyOrder modifies stop-loss and/or take-profit on an open position
// via the MT gateway's OrderModify RPC.
// price is the new limit price for pending orders (use decimal.Zero if N/A).
func (s *MtHubService) ModifyOrder(ctx context.Context, accountID string, ticket int64, sl, tp, price decimal.Decimal) error {
	if s.killSwitch != nil && s.killSwitch.IsEngaged() {
		return ErrKillSwitchEngaged
	}
	if s.accountOwnerVerifier != nil {
		uid := usermgr.GetUserID(ctx)
		if uid == "" {
			return fmt.Errorf("unauthenticated: user ID required for order modification")
		}
		owns, err := s.accountOwnerVerifier(ctx, uid, accountID)
		if err != nil {
			return fmt.Errorf("account ownership check: %w", err)
		}
		if !owns {
			return fmt.Errorf("%w: %s", ErrAccountNotOwned, accountID)
		}
	}

	modifyID := fmt.Sprintf("modify-%s-%d", accountID, ticket)
	if s.idem != nil {
		isDup, _, err := s.idem.CheckAndSet(ctx, accountID, modifyID, ticket)
		if err != nil {
			return fmt.Errorf("idempotency check: %w", err)
		}
		if isDup {
			return nil
		}
		defer s.idem.DeleteKey(ctx, accountID, modifyID)
	}

	// D6-A: gate intentionally NOT applied to ModifyOrder.
	// Gate rules (MaxLotSize, MaxPositionCount, MaxExposure, etc.) are
	// designed for position OPENING — they evaluate exposure, margin,
	// and concentration risk.  SL/TP modification does not change any
	// of these.

	exec := s.hub.Get(accountID)
	if exec == nil {
		if s.logger != nil {
			s.logger.Warn("ModifyOrder: session not found", zap.String("accountID", accountID), zap.Int64("ticket", ticket))
		}
		return ErrSessionNotFound
	}

	_ = price // passed to executor, used when modifying pending order limit price

	if s.logger != nil {
		s.logger.Info("ModifyOrder: calling executor",
			zap.String("accountID", accountID), zap.Int64("ticket", ticket),
			zap.String("sl", sl.String()), zap.String("tp", tp.String()))
	}

	if err := exec.ModifyOrder(ctx, ticket, sl, tp, price); err != nil {
		if s.logger != nil {
			s.logger.Error("ModifyOrder: executor failed", zap.Error(err),
				zap.String("accountID", accountID), zap.Int64("ticket", ticket))
		}
		return fmt.Errorf("modify order: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("ModifyOrder: executor success",
			zap.String("accountID", accountID), zap.Int64("ticket", ticket))
	}
	return nil
}
