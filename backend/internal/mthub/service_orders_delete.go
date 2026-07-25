package mthub

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"alphaforge/internal/usermgr"
)

// DeleteOrder cancels a pending order.
// Gate order matches CloseOrder: killSwitch → ownership → rateLimit.
// Idempotency and reconcile gates are skipped — cancel is safe to retry.
func (s *MtHubService) DeleteOrder(ctx context.Context, accountID string, ticket int64) error {
	if s.killSwitch != nil && s.killSwitch.IsEngaged() {
		return ErrKillSwitchEngaged
	}
	if s.accountOwnerVerifier != nil {
		uid := usermgr.GetUserID(ctx)
		if uid != "" {
			owns, err := s.accountOwnerVerifier(ctx, uid, accountID)
			if err != nil {
				return fmt.Errorf("account ownership check: %w", err)
			}
			if !owns {
				return fmt.Errorf("%w: %s", ErrAccountNotOwned, accountID)
			}
		}
	}

	if s.userLimiter != nil {
		uid := usermgr.GetUserID(ctx)
		if uid != "" && !s.userLimiter.AllowOrder(uid) {
			return ErrRateLimited
		}
	}

	exec := s.hub.Get(accountID)
	if exec == nil {
		if s.logger != nil {
			s.logger.Warn("DeleteOrder: session not found", zap.String("accountID", accountID), zap.Int64("ticket", ticket))
		}
		return ErrSessionNotFound
	}

	if s.logger != nil {
		s.logger.Info("DeleteOrder: calling executor", zap.String("accountID", accountID), zap.Int64("ticket", ticket))
	}

	err := exec.DeleteOrder(ctx, ticket)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("DeleteOrder: executor failed", zap.Error(err), zap.String("accountID", accountID), zap.Int64("ticket", ticket))
		}
		return err
	}

	if s.logger != nil {
		s.logger.Info("DeleteOrder: executor success", zap.String("accountID", accountID), zap.Int64("ticket", ticket))
	}
	return nil
}
