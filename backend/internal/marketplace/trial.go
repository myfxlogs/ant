package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const trialDuration = 7 * 24 * time.Hour

// StartTrial creates a 7-day free trial for a strategy.
// One trial per user per strategy — if already trialed, returns already_tried=true.
func (s *Service) StartTrial(ctx context.Context, userID, strategyID string) (trialID string, expiresAt time.Time, alreadyTried bool, err error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", time.Time{}, false, fmt.Errorf("marketplace: invalid user_id: %w", err)
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return "", time.Time{}, false, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}

	// Verify strategy is published and paid (free strategies don't need trials).
	var priceModel string
	err = s.pg.QueryRow(ctx,
		`SELECT price_model FROM marketplace_strategies WHERE strategy_id = $1 AND status = 'published'`,
		sid,
	).Scan(&priceModel)
	if err != nil {
		return "", time.Time{}, false, fmt.Errorf("marketplace: strategy not found or not published")
	}
	if priceModel == PriceModelFree {
		return "", time.Time{}, false, fmt.Errorf("marketplace: free strategies do not offer trials")
	}

	now := time.Now().UTC()
	exp := now.Add(trialDuration)

	// Atomic insert with ON CONFLICT to handle race condition (unique index on user_id + strategy_id).
	newID := uuid.New()
	var existingID string
	err = s.pg.QueryRow(ctx,
		`INSERT INTO marketplace_trials (id, user_id, strategy_id, started_at, expires_at, status)
		 VALUES ($1, $2, $3, $4, $5, 'active')
		 ON CONFLICT (user_id, strategy_id) DO UPDATE SET started_at = marketplace_trials.started_at
		 RETURNING id::text`,
		newID, uid, sid, now, exp,
	).Scan(&existingID)
	if err != nil {
		return "", time.Time{}, false, fmt.Errorf("marketplace: create trial: %w", err)
	}

	// If the returned ID is not the one we generated, a trial already existed.
	if existingID != newID.String() {
		return existingID, time.Time{}, true, nil
	}

	return newID.String(), exp, false, nil
}

// HasActiveTrial checks if a user has an active (non-expired) trial for a strategy.
// Expired trials are lazily marked as expired.
func (s *Service) HasActiveTrial(ctx context.Context, userID, strategyID string) (bool, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, nil
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return false, nil
	}

	// Check for active trial and lazily expire old ones.
	now := time.Now().UTC()
	_, _ = s.pg.Exec(ctx,
		`UPDATE marketplace_trials SET status = 'expired'
		 WHERE user_id = $1 AND strategy_id = $2 AND status = 'active' AND expires_at < $3`,
		uid, sid, now,
	)

	var exists bool
	err = s.pg.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM marketplace_trials
		 WHERE user_id = $1 AND strategy_id = $2 AND status = 'active' AND expires_at > $3)`,
		uid, sid, now,
	).Scan(&exists)
	if err != nil {
		return false, nil
	}

	// Lazily notify trials expiring within 24h (push-first, no cron).
	if exists {
		go s.notifyTrialExpiring(context.Background(), uid, sid)
	}

	return exists, nil
}
