package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// notifyTrialExpiring sends a notification for trials expiring within 24h.
// Called lazily from HasActiveTrial — only fires once per trial (uses notified_expiring flag).
// This is the push-first approach: no cron, no timer — triggered when the user accesses code.
func (s *Service) notifyTrialExpiring(ctx context.Context, userID uuid.UUID, strategyID uuid.UUID) {
	if s.notifSender == nil {
		return
	}

	// Atomically claim trials expiring within 24h that haven't been notified.
	// This prevents duplicate notifications from concurrent goroutines.
	rows, err := s.pg.Query(ctx,
		`UPDATE marketplace_trials
		 SET notified_expiring = true
		 WHERE user_id = $1 AND strategy_id = $2 AND status = 'active'
		   AND expires_at < now() + INTERVAL '24 hours' AND expires_at > now()
		   AND notified_expiring = false
		 RETURNING id::text, expires_at`,
		userID, strategyID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var trialID string
		var expiresAt time.Time
		if err := rows.Scan(&trialID, &expiresAt); err != nil {
			continue
		}

		hoursLeft := time.Until(expiresAt).Hours()
		_, _ = s.notifSender.Send(ctx, userID, "trial_expiring",
			"Free Trial Expiring Soon",
			fmt.Sprintf("Your free trial expires in %.0f hours. Subscribe to keep access.", hoursLeft),
			nil)
	}
}

// notifyNewStrategy notifies all users who have matching asset class preferences
// when a new strategy is published. Called from Publish.
func (s *Service) notifyNewStrategy(ctx context.Context, strategyID uuid.UUID, title, assetClass string) {
	if s.notifSender == nil {
		return
	}

	// Notify users who have new_strategy_enabled=true (defaults to true if no prefs row).
	rows, err := s.pg.Query(ctx,
		`SELECT u.id FROM users u
			 LEFT JOIN marketplace_notification_prefs p ON p.user_id = u.id
			 WHERE COALESCE(p.new_strategy_enabled, true) = true`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			continue
		}
		_, _ = s.notifSender.Send(ctx, uid, "new_strategy",
			"New Strategy Published",
			fmt.Sprintf("A new strategy \"%s\" (%s) is now available on the marketplace.", title, assetClass),
			nil)
	}
}

// notifyPriceChange notifies all active subscribers of a strategy when its price changes.
// Called from SetPricing.
func (s *Service) notifyPriceChange(ctx context.Context, strategyID uuid.UUID, strategyTitle, oldPrice, newPrice string) {
	if s.notifSender == nil {
		return
	}

	rows, err := s.pg.Query(ctx,
		`SELECT us.subscriber_user_id
		 FROM user_subscriptions us
		 LEFT JOIN marketplace_notification_prefs p ON p.user_id = us.subscriber_user_id
		 WHERE us.target_strategy_id = $1 AND us.active = true AND COALESCE(p.price_change_enabled, true) = true`,
		strategyID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			continue
		}
		_, _ = s.notifSender.Send(ctx, uid, "price_change",
			"Strategy Price Updated",
			fmt.Sprintf("\"%s\" price changed from %s to %s.", strategyTitle, oldPrice, newPrice),
			nil)
	}
}

// notifyNewRating notifies the strategy publisher when a new rating or comment is received.
// Called from Rate and Comment.
func (s *Service) notifyNewRating(ctx context.Context, strategyID uuid.UUID, strategyTitle string, rating int32) {
	if s.notifSender == nil {
		return
	}

	// Get the strategy publisher.
	var publisherID uuid.UUID
	err := s.pg.QueryRow(ctx,
		`SELECT usp.user_id
		 FROM user_strategy_publishes usp
		 WHERE usp.platform_strategy_id = $1`,
		strategyID,
	).Scan(&publisherID)
	if err != nil {
		return
	}

	// Check publisher's preference (defaults to true if no prefs row).
	var enabled *bool
	_ = s.pg.QueryRow(ctx,
		`SELECT new_rating_enabled FROM marketplace_notification_prefs WHERE user_id = $1`,
		publisherID,
	).Scan(&enabled)
	if enabled != nil && !*enabled {
		return
	}

	_, _ = s.notifSender.Send(ctx, publisherID, "new_rating",
		"New Rating Received",
		fmt.Sprintf("Your strategy \"%s\" received a %d-star rating.", strategyTitle, rating),
		nil)
}

// notifySubExpiring notifies users whose subscriptions are expiring within 3 days.
// Called lazily from ListSubscriptions (when user views their purchases).
func (s *Service) notifySubExpiring(ctx context.Context, userID uuid.UUID) {
	if s.notifSender == nil {
		return
	}

	// Check preference (defaults to true if no prefs row).
	var enabled bool
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(p.sub_expiring_enabled, true) FROM users u
		 LEFT JOIN marketplace_notification_prefs p ON p.user_id = u.id
		 WHERE u.id = $1`,
		userID,
	).Scan(&enabled)
	if !enabled {
		return
	}

	// Atomically claim subscriptions expiring within 3 days that haven't been notified.
	rows, err := s.pg.Query(ctx,
		`UPDATE user_subscriptions
		 SET notified_expiring = true
		 WHERE subscriber_user_id = $1 AND active = true
		   AND expires_at IS NOT NULL
		   AND expires_at < now() + INTERVAL '3 days' AND expires_at > now()
		   AND notified_expiring = false
		 RETURNING id::text, expires_at, target_strategy_id`,
		userID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var subID string
		var expiresAt time.Time
		var strategyID uuid.UUID
		if err := rows.Scan(&subID, &expiresAt, &strategyID); err != nil {
			continue
		}

		var title string
		_ = s.pg.QueryRow(ctx, `SELECT COALESCE(title,'') FROM marketplace_strategies WHERE strategy_id = $1`, strategyID).Scan(&title)

		daysLeft := time.Until(expiresAt).Hours() / 24
		_, _ = s.notifSender.Send(ctx, userID, "sub_expiring",
			"Subscription Expiring Soon",
			fmt.Sprintf("Your subscription to \"%s\" expires in %.1f days.", title, daysLeft),
			nil)
	}
}

// notifyPerformanceAnomaly notifies all active subscribers when a strategy's
// daily drawdown exceeds 10% or daily return exceeds 20% (anomaly thresholds).
// Called from UpsertDailyPerformance after the daily record is written.
func (s *Service) notifyPerformanceAnomaly(ctx context.Context, strategyID uuid.UUID, strategyTitle string, dailyReturn, drawdown decimal.Decimal) {
	if s.notifSender == nil {
		return
	}

	// Anomaly thresholds: daily drawdown > 10% or daily return > 20%.
	threshold := decimal.NewFromFloat(0.10)
	returnThreshold := decimal.NewFromFloat(0.20)

	isDrawdownAnomaly := drawdown.GreaterThan(threshold)
	isReturnAnomaly := dailyReturn.Abs().GreaterThan(returnThreshold)
	if !isDrawdownAnomaly && !isReturnAnomaly {
		return
	}

	// Notify all active subscribers with performance_alert_enabled (defaults to true if no prefs row).
	rows, err := s.pg.Query(ctx,
		`SELECT us.subscriber_user_id
		 FROM user_subscriptions us
		 LEFT JOIN marketplace_notification_prefs p ON p.user_id = us.subscriber_user_id
		 WHERE us.target_strategy_id = $1 AND us.active = true AND COALESCE(p.performance_alert_enabled, true) = true`,
		strategyID)
	if err != nil {
		return
	}
	defer rows.Close()

	var alertType, detail string
	if isDrawdownAnomaly {
		alertType = "Performance Alert: Large Drawdown"
		detail = fmt.Sprintf("\"%s\" experienced a %.1f%% drawdown today.", strategyTitle, drawdown.InexactFloat64()*100)
	} else {
		alertType = "Performance Alert: Large Return"
		detail = fmt.Sprintf("\"%s\" had a %.1f%% daily return today.", strategyTitle, dailyReturn.InexactFloat64()*100)
	}

	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			continue
		}
		_, _ = s.notifSender.Send(ctx, uid, "performance_alert", alertType, detail, nil)
	}
}
