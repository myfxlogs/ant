package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ── Rating ────────────────────────────────────────────────────────────────────

// Rate inserts or updates a user's rating for a strategy and returns the new
// average and count (matching the RateStrategyResponse proto shape).
func (s *Service) Rate(ctx context.Context, userID, strategyID string, rating int32) (float64, int32, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return 0, 0, fmt.Errorf("marketplace: invalid user_id: %w", err)
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return 0, 0, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	_, err = s.pg.Exec(ctx,
		`INSERT INTO marketplace_ratings (strategy_id, user_id, rating)
			 VALUES ($1,$2,$3) ON CONFLICT (strategy_id, user_id) DO UPDATE SET rating=$3`,
		sid, uid, rating)
	if err != nil {
		return 0, 0, fmt.Errorf("marketplace: rate: %w", err)
	}
	var avg float64
	var count int32
	err = s.pg.QueryRow(ctx,
		`SELECT COALESCE(AVG(rating),0), COUNT(*) FROM marketplace_ratings WHERE strategy_id=$1`, sid,
	).Scan(&avg, &count)
	return avg, count, err
}

// ListRatings returns all ratings for a strategy with average and count.
func (s *Service) ListRatings(ctx context.Context, strategyID string) ([]RatingItem, float64, int32, error) {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	rows, err := s.pg.Query(ctx,
		`SELECT id, user_id, rating, created_at FROM marketplace_ratings WHERE strategy_id=$1 ORDER BY created_at DESC`, sid)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()
	var items []RatingItem
	for rows.Next() {
		var r RatingItem
		if err := rows.Scan(&r.ID, &r.UserID, &r.Rating, &r.CreatedAt); err != nil {
			return nil, 0, 0, err
		}
		items = append(items, r)
	}
	var avg float64
	var count int32
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(AVG(rating),0), COUNT(*) FROM marketplace_ratings WHERE strategy_id=$1`, sid,
	).Scan(&avg, &count)
	return items, avg, count, rows.Err()
}

// ── Comment ───────────────────────────────────────────────────────────────────

// Comment adds a comment to a strategy and returns the new comment ID.
func (s *Service) Comment(ctx context.Context, userID, strategyID, content string) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("marketplace: invalid user_id: %w", err)
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return "", fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	var id uuid.UUID
	err = s.pg.QueryRow(ctx,
		`INSERT INTO marketplace_comments (strategy_id, user_id, content)
			 VALUES ($1,$2,$3) RETURNING id`, sid, uid, content).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("marketplace: comment: %w", err)
	}
	return id.String(), nil
}

// ListComments returns paginated comments for a strategy, with total count.
func (s *Service) ListComments(ctx context.Context, strategyID string, limit, offset int32) ([]CommentItem, int32, error) {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	var total int32
	_ = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM marketplace_comments WHERE strategy_id=$1`, sid).Scan(&total)
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pg.Query(ctx,
		`SELECT c.id, c.user_id, COALESCE(u.nickname,u.email,''), c.content, c.created_at
			 FROM marketplace_comments c LEFT JOIN users u ON u.id=c.user_id
			 WHERE c.strategy_id=$1 ORDER BY c.created_at ASC LIMIT $2 OFFSET $3`,
		sid, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []CommentItem
	for rows.Next() {
		var c CommentItem
		if err := rows.Scan(&c.ID, &c.UserID, &c.UserName, &c.Content, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, c)
	}
	return items, total, rows.Err()
}

// ── Purchase ──────────────────────────────────────────────────────────────────

// PurchaseStrategy atomically charges the user's wallet and creates a subscription
// for a one-time purchase strategy. All steps run in a single DB transaction
// with FOR UPDATE row locking to prevent races.
func (s *Service) PurchaseStrategy(ctx context.Context, userID, strategyID, publisherUserID string) (*PurchaseResult, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid user_id: %w", err)
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	pid, err := uuid.Parse(publisherUserID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid publisher_user_id: %w", err)
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("marketplace: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Look up strategy price.
	var priceModel string
	var priceAmount float64
	var strategyTitle string
	err = tx.QueryRow(ctx,
		`SELECT price_model, COALESCE(price_amount, 0), title FROM marketplace_strategies WHERE strategy_id = $1`,
		sid,
	).Scan(&priceModel, &priceAmount, &strategyTitle)
	if err != nil {
		return nil, fmt.Errorf("marketplace: strategy not published")
	}
	if priceModel != "once" || priceAmount <= 0 {
		return nil, fmt.Errorf("marketplace: strategy is not purchasable")
	}

	// 2. Check for existing active subscription.
	var existing string
	_ = tx.QueryRow(ctx,
		`SELECT id::text FROM user_subscriptions WHERE subscriber_user_id = $1 AND target_strategy_id = $2 AND active = true`,
		uid, sid,
	).Scan(&existing)
	if existing != "" {
		return nil, fmt.Errorf("marketplace: already subscribed")
	}

	// 3. Lock wallet and read balance.
	var walletID uuid.UUID
	var balanceBefore string
	err = tx.QueryRow(ctx,
		`SELECT id, balance::text FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
		uid,
	).Scan(&walletID, &balanceBefore)
	if err != nil {
		return nil, fmt.Errorf("marketplace: wallet not found")
	}

	if balanceBefore == "" {
		return nil, fmt.Errorf("marketplace: wallet balance unavailable")
	}

	amountStr := fmt.Sprintf("%.2f", priceAmount)
	negAmountStr := fmt.Sprintf("-%.2f", priceAmount)

	// 4. Deduct balance (DB enforces non-negative via CHECK constraint).
	var balanceAfter string
	err = tx.QueryRow(ctx,
		`UPDATE user_wallets SET balance = balance - $1::numeric, updated_at = now()
			 WHERE user_id = $2 AND balance >= $1::numeric
			 RETURNING balance::text`,
		amountStr, uid,
	).Scan(&balanceAfter)
	if err != nil {
		return nil, fmt.Errorf("marketplace: insufficient balance")
	}

	// 5. Insert wallet_transactions row.
	var txID uuid.UUID
	desc := fmt.Sprintf("Purchase strategy: %s", strategyTitle)
	err = tx.QueryRow(ctx,
		`INSERT INTO wallet_transactions (id, wallet_id, user_id, tx_type, amount, balance_before, balance_after, description)
			 VALUES (uuid_generate_v4(), $1, $2, 'purchase', $3::numeric, $4::numeric, $5::numeric, $6)
			 RETURNING id`,
		walletID, uid, negAmountStr, balanceBefore, balanceAfter, desc,
	).Scan(&txID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: record transaction: %w", err)
	}

	// 6. Insert subscription row.
	var subID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO user_subscriptions (id, subscriber_user_id, target_user_id, target_strategy_id, kind, active)
			 VALUES (uuid_generate_v4(), $1, $2, $3, 'purchase', true)
			 RETURNING id`,
		uid, pid, sid,
	).Scan(&subID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: create subscription: %w", err)
	}

	// 7. Increment total_subscribers counter.
	_, err = tx.Exec(ctx,
		`UPDATE marketplace_strategies SET total_subscribers = total_subscribers + 1 WHERE strategy_id = $1`,
		sid,
	)
	if err != nil {
		return nil, fmt.Errorf("marketplace: update subscriber count: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("marketplace: commit purchase: %w", err)
	}

	return &PurchaseResult{
		SubscriptionID: subID.String(),
		TransactionID:  txID.String(),
		AmountCharged:  amountStr,
		BalanceAfter:   balanceAfter,
	}, nil
}

// ── Admin Pricing ─────────────────────────────────────────────────────────────

// SetPricing updates the pricing model and amount for a published strategy.
func (s *Service) SetPricing(ctx context.Context, strategyID, priceModel string, priceAmount float64) error {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_strategies SET price_model=$2, price_amount=$3, updated_at=now() WHERE strategy_id=$1`,
		sid, priceModel, priceAmount)
	return err
}
