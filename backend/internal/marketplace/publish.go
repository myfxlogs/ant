package marketplace

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// Unpublish hides a strategy from the marketplace by setting its status to "hidden".
// The strategy still exists in the database but no longer appears in listings.
func (s *Service) Unpublish(ctx context.Context, strategyID, userID string, isAdmin bool) error {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}

	// Verify ownership unless admin.
	if !isAdmin {
		var publisherID string
		err := s.pg.QueryRow(ctx,
			`SELECT publisher_id::text FROM marketplace_strategies WHERE strategy_id = $1`,
			sid,
		).Scan(&publisherID)
		if err != nil {
			return fmt.Errorf("marketplace: strategy not found")
		}
		if publisherID != userID {
			return fmt.Errorf("marketplace: only the publisher can unpublish this strategy")
		}
	}

	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_strategies SET status = 'hidden', updated_at = now() WHERE strategy_id = $1`,
		sid,
	)
	if err != nil {
		return fmt.Errorf("marketplace: unpublish: %w", err)
	}
	s.pubCache.clear()
	return nil
}

// GetPublisherStats returns aggregated dashboard statistics for a publisher.
// Triggers lazy settlement of expired frozen balances before computing stats.
func (s *Service) GetPublisherStats(ctx context.Context, userID string) (*PublisherStats, error) {
	// Phase 5.4: Lazy settlement — settle expired frozen balances for this provider.
	if _, err := s.SettleExpired(ctx, userID); err != nil {
		s.log.Warn("publisher stats: lazy settlement failed", zap.String("userID", userID), zap.Error(err))
	}

	var stats PublisherStats
	err := s.pg.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(total_subscribers),0)
		 FROM marketplace_strategies WHERE publisher_id::text = $1 AND status = 'published'`,
		userID,
	).Scan(&stats.TotalPublished, &stats.TotalSubscribers)
	if err != nil {
		return nil, err
	}

	// Total revenue from settled transactions (Phase 5.4: settlement tx_type replaces sale).
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount::numeric),0)::text FROM wallet_transactions
		 WHERE user_id::text = $1 AND tx_type = 'settlement'`,
		userID,
	).Scan(&stats.TotalRevenue)

	// Monthly revenue (last 30 days) — settled only.
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount::numeric),0)::text FROM wallet_transactions
		 WHERE user_id::text = $1 AND tx_type = 'settlement'
		   AND created_at > now() - INTERVAL '30 days'`,
		userID,
	).Scan(&stats.MonthlyRevenue)

	// Phase 5.4: Pending settlement balance (frozen provider_amount sum).
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(SUM(provider_amount),0)::text FROM marketplace_settlements
		 WHERE provider_id::text = $1 AND status = 'frozen'`,
		userID,
	).Scan(&stats.PendingSettlement)

	// Phase 5.4: Earliest next settlement date among frozen rows.
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(MIN(settles_at)::text, '') FROM marketplace_settlements
		 WHERE provider_id::text = $1 AND status = 'frozen'`,
		userID,
	).Scan(&stats.NextSettlementDate)

	// Average rating across all published strategies.
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(AVG(r.avg_rating),0)
		 FROM marketplace_strategies ms
		 LEFT JOIN (SELECT strategy_id, AVG(rating) AS avg_rating FROM marketplace_ratings GROUP BY strategy_id) r ON r.strategy_id = ms.strategy_id
		 WHERE ms.publisher_id::text = $1 AND ms.status = 'published'`,
		userID,
	).Scan(&stats.AvgRating)

	// Top strategy by subscribers.
	_ = s.pg.QueryRow(ctx,
		`SELECT strategy_id::text, COALESCE(title,''), total_subscribers
		 FROM marketplace_strategies
		 WHERE publisher_id::text = $1 AND status = 'published'
		 ORDER BY total_subscribers DESC LIMIT 1`,
		userID,
	).Scan(&stats.TopStrategyID, &stats.TopStrategyTitle, &stats.TopStrategySubs)

	// Phase 2.4: Enhanced analytics
	stats.RevenueTrend, _ = s.getRevenueTrend(ctx, userID)
	stats.SubscriberTrend, _ = s.getSubscriberTrend(ctx, userID)
	stats.StrategyBreakdown, _ = s.getStrategyBreakdown(ctx, userID)

	return &stats, nil
}

// Publish adds a strategy to the marketplace. Writes to both
// user_strategy_publishes (ownership tracking) and marketplace_strategies
// (rich listing metadata). Uses a transaction for atomicity.
func (s *Service) Publish(ctx context.Context, params PublishParams) (string, error) {
	switch params.PriceModel {
	case PriceModelFree, PriceModelOnce, PriceModelSubscription:
	default:
		return "", fmt.Errorf("marketplace: unsupported price_model %q", params.PriceModel)
	}

	violations, err := s.ValidateBacktestQuality(ctx, params.BacktestSnapshotProto, params.StrategyID)
	if err != nil {
		return "", fmt.Errorf("marketplace: quality gate check: %w", err)
	}
	if len(violations) > 0 {
		msgs := make([]string, len(violations))
		for i, v := range violations {
			msgs[i] = v.String()
		}
		return "", fmt.Errorf("marketplace: backtest quality gate failed: %s", strings.Join(msgs, "; "))
	}

	// Verify the caller owns the strategy template before publishing.
	var templateOwnerID string
	err = s.pg.QueryRow(ctx,
		`SELECT user_id::text FROM strategy_templates WHERE id = $1`,
		params.StrategyID,
	).Scan(&templateOwnerID)
	if err != nil {
		return "", fmt.Errorf("marketplace: strategy template not found: %w", err)
	}
	if templateOwnerID != params.UserID {
		return "", fmt.Errorf("marketplace: not the strategy owner")
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("marketplace: publish begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	publishID, stratID := uuid.New(), uuid.New()
	_, err = tx.Exec(ctx, `INSERT INTO user_strategy_publishes (id, user_id, platform_strategy_id, published_at) VALUES ($1,$2,$3,now())`,
		publishID, params.UserID, params.StrategyID)
	if err != nil {
		return "", fmt.Errorf("marketplace: insert publish: %w", err)
	}

	// H1: ON CONFLICT — if a published row already exists for this strategy_id,
	// do nothing and return the existing publish ID.
	var existingPublishID string
	refundWindowDays := params.RefundWindowDays
	if refundWindowDays <= 0 {
		refundWindowDays = DefaultRefundWindowDays
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO marketplace_strategies (id, strategy_id, publisher_id, title, description, price_model, price_amount, asset_class, symbols, timeframe, risk_level, tags, code_snippet, backtest_snapshot, platform_fee_rate, disclaimer, trial_days, refund_window_days, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7::numeric,$8,$9,$10,$11,$12,$13,$14,$15::numeric,$16,$17,$18,'published',now(),now())
		ON CONFLICT (strategy_id) WHERE status = 'published' DO NOTHING
		RETURNING id`,
		stratID, params.StrategyID, params.UserID, params.Title, params.Description,
		params.PriceModel, params.PriceAmount, params.AssetClass,
		params.Symbols, params.Timeframe, params.RiskLevel, params.Tags,
		params.CodeSnippet, params.BacktestSnapshotProto, params.PlatformFeeRate, params.Disclaimer,
		params.TrialDays, refundWindowDays,
	).Scan(&existingPublishID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Conflict — row already exists. Fetch the existing publish ID.
			err = tx.QueryRow(ctx,
				`SELECT id::text FROM marketplace_strategies WHERE strategy_id = $1 AND status = 'published'`,
				params.StrategyID,
			).Scan(&existingPublishID)
			if err != nil {
				return "", fmt.Errorf("marketplace: fetch existing listing: %w", err)
			}
			// Roll back the user_strategy_publishes insert since this is a no-op.
			_ = tx.Rollback(ctx)
			s.pubCache.clear()
			return existingPublishID, nil
		}
		return "", fmt.Errorf("marketplace: insert listing: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("marketplace: publish commit: %w", err)
	}
	s.pubCache.clear()

	// Notify users who opted into new strategy notifications.
	go s.notifyNewStrategy(context.WithoutCancel(ctx), params.Title, params.AssetClass)

	return publishID.String(), nil
}
