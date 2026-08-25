// service_subscription_loop.go — Renewal loop and access check extracted from service_subscription.go.
package marketplace

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (s *Service) StartRenewalLoop(ctx context.Context, log *zap.Logger) {
	go func() {
		// D5: Align first tick to next midnight UTC + jitter (0-5min) to avoid
		// thundering herd on startup and make renewal timing deterministic.
		now := time.Now().UTC()
		nextMidnight := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
		initialDelay := time.Until(nextMidnight) + time.Duration(time.Now().UnixNano()%int64(5*time.Minute))
		if initialDelay < 0 {
			initialDelay = 0
		}

		// Run once at startup to catch overdue renewals.
		renewed, failed, _ := s.RenewSubscriptions(ctx)
		if renewed+failed > 0 {
			log.Info("subscription renewal startup run", zap.Int("renewed", renewed), zap.Int("failed", failed))
		}

		// Wait until first aligned tick, then switch to 24h ticker.
		timer := time.NewTimer(initialDelay)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				renewed, failed, err := s.RenewSubscriptions(runCtx)
				cancel()
				if err != nil {
					log.Error("subscription renewal failed", zap.Error(err))
				} else if renewed+failed > 0 {
					log.Info("subscription renewal complete", zap.Int("renewed", renewed), zap.Int("failed", failed))
				}
				timer.Reset(24 * time.Hour)
			}
		}
	}()
}

// CanAccessCode returns true if the user can view the full strategy code.
// Access is granted if the user is the template owner, has an active
// subscription/purchase to the published strategy, or has an active free trial.
func (s *Service) CanAccessCode(ctx context.Context, userID, strategyID string) (bool, error) {
	// M8: Parse UUIDs for index-friendly queries.
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, nil
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return false, nil
	}

	// Check if user is the template owner.
	var ownerID string
	err = s.pg.QueryRow(ctx,
		`SELECT user_id::text FROM strategy_templates WHERE id = $1`,
		sid,
	).Scan(&ownerID)
	if err != nil {
		return false, nil // template not found → deny
	}
	if ownerID == userID {
		return true, nil // owner always has access
	}

	// Check if user has an active, non-expired subscription.
	var exists bool
	err = s.pg.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_subscriptions
		 WHERE subscriber_user_id = $1 AND target_strategy_id = $2 AND active = true
		   AND (expires_at IS NULL OR expires_at > now()))`,
		uid, sid,
	).Scan(&exists)
	if err != nil {
		return false, nil
	}
	if exists {
		return true, nil
	}

	// Check if user has an active free trial.
	hasTrial, _ := s.HasActiveTrial(ctx, userID, strategyID)
	return hasTrial, nil
}
