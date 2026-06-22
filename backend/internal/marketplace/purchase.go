package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ── Purchase ──────────────────────────────────────────────────────────────────

// PurchaseStrategy atomically charges the user's wallet, credits the publisher,
// and creates a subscription — all in a single DB transaction with FOR UPDATE
// row locking to prevent races.
func (s *Service) PurchaseStrategy(ctx context.Context, userID, strategyID, idempotencyKey string) (*PurchaseResult, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid user_id: %w", err)
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}

	// 0. Idempotency check — if a subscription already exists with this key, return it.
	if idempotencyKey != "" {
		var existingSubID, existingTxID, existingAmount, existingBalance string
		err := s.pg.QueryRow(ctx,
			`SELECT us.id::text, wt.id::text, ABS(wt.amount)::text, w.balance::text
			 FROM user_subscriptions us
			 JOIN wallet_transactions wt ON wt.user_id = us.subscriber_user_id AND wt.tx_type = $1
			 JOIN user_wallets w ON w.user_id = us.subscriber_user_id
			 WHERE us.idempotency_key = $2 AND us.subscriber_user_id = $3
			 ORDER BY wt.created_at DESC LIMIT 1`,
			TxTypePurchase, idempotencyKey, uid,
		).Scan(&existingSubID, &existingTxID, &existingAmount, &existingBalance)
		if err == nil {
			return &PurchaseResult{
				SubscriptionID: existingSubID,
				TransactionID:  existingTxID,
				AmountCharged:  existingAmount,
				BalanceAfter:   existingBalance,
			}, nil
		}
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("marketplace: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Look up strategy price, publisher, and platform fee (source of truth from DB).
	var priceModel string
	var priceAmount float64
	var strategyTitle string
	var dbPublisherID string
	var platformFeeRate float64
	err = tx.QueryRow(ctx,
		`SELECT price_model, COALESCE(price_amount, 0), title, publisher_id::text,
		        COALESCE(platform_fee_rate, 0)
		 FROM marketplace_strategies WHERE strategy_id = $1`,
		sid,
	).Scan(&priceModel, &priceAmount, &strategyTitle, &dbPublisherID, &platformFeeRate)
	if err != nil {
		return nil, fmt.Errorf("marketplace: strategy not published")
	}
	if (priceModel != PriceModelOnce && priceModel != PriceModelSubscription) || priceAmount <= 0 {
		return nil, fmt.Errorf("marketplace: strategy is not purchasable")
	}

	// Determine subscription kind and expiry.
	subKind := SubKindPurchase
	var expiresAt *string // nil = permanent
	if priceModel == PriceModelSubscription {
		subKind = SubKindSubscription
		expiry := "now() + INTERVAL '30 days'"
		expiresAt = &expiry
	}

	// Use publisher from DB as source of truth (ignore client-supplied value).
	pid, err := uuid.Parse(dbPublisherID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid publisher_id in DB: %w", err)
	}

	// 2. Guard: cannot purchase your own strategy.
	if uid == pid {
		return nil, fmt.Errorf("marketplace: cannot purchase your own strategy")
	}

	// 3. Check for existing active subscription.
	var existing string
	_ = tx.QueryRow(ctx,
		`SELECT id::text FROM user_subscriptions WHERE subscriber_user_id = $1 AND target_strategy_id = $2 AND active = true`,
		uid, sid,
	).Scan(&existing)
	if existing != "" {
		return nil, fmt.Errorf("marketplace: already subscribed")
	}

	// 4. Lock buyer wallet and read balance.
	var buyerWalletID uuid.UUID
	var buyerBalanceBefore string
	err = tx.QueryRow(ctx,
		`SELECT id, balance::text FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
		uid,
	).Scan(&buyerWalletID, &buyerBalanceBefore)
	if err != nil {
		return nil, fmt.Errorf("marketplace: wallet not found")
	}
	if buyerBalanceBefore == "" {
		return nil, fmt.Errorf("marketplace: wallet balance unavailable")
	}

	amountStr := fmt.Sprintf("%.2f", priceAmount)
	negAmountStr := fmt.Sprintf("-%.2f", priceAmount)

	// 5. Deduct buyer balance (DB enforces non-negative via CHECK constraint).
	var buyerBalanceAfter string
	err = tx.QueryRow(ctx,
		`UPDATE user_wallets SET balance = balance - $1::numeric, updated_at = now()
			 WHERE user_id = $2 AND balance >= $1::numeric
			 RETURNING balance::text`,
		amountStr, uid,
	).Scan(&buyerBalanceAfter)
	if err != nil {
		return nil, fmt.Errorf("marketplace: insufficient balance")
	}

	// 6. Record buyer wallet_transaction.
	buyerTxID := uuid.New()
	buyerDesc := fmt.Sprintf("Purchase strategy: %s", strategyTitle)
	err = tx.QueryRow(ctx,
		fmt.Sprintf(`INSERT INTO wallet_transactions (id, wallet_id, user_id, tx_type, amount, balance_before, balance_after, description)
			 VALUES ($1, $2, $3, '%s', $4::numeric, $5::numeric, $6::numeric, $7)
			 RETURNING id`, TxTypePurchase),
		buyerTxID, buyerWalletID, uid, negAmountStr, buyerBalanceBefore, buyerBalanceAfter, buyerDesc,
	).Scan(&buyerTxID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: record buyer transaction: %w", err)
	}

	// 7. Credit publisher wallet (minus platform fee). Use decimal for precise arithmetic.
	priceDec := decimal.NewFromFloat(priceAmount)
	publisherAmount := priceAmount
	platformFee := 0.0
	if platformFeeRate > 0 {
		feeDec := priceDec.Mul(decimal.NewFromFloat(platformFeeRate))
		platformFee, _ = feeDec.Float64()
		publisherDec := priceDec.Sub(feeDec)
		publisherAmount, _ = publisherDec.Float64()
	}
	pubAmountStr := fmt.Sprintf("%.2f", publisherAmount)
	var pubWalletID uuid.UUID
	var pubBalanceBefore string
	err = tx.QueryRow(ctx,
		`SELECT id, balance::text FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
		pid,
	).Scan(&pubWalletID, &pubBalanceBefore)
	if err != nil {
		// Publisher may not have a wallet yet — create one.
		err = tx.QueryRow(ctx,
			`INSERT INTO user_wallets (user_id) VALUES ($1)
			 ON CONFLICT (user_id) DO NOTHING
			 RETURNING id, balance::text`,
			pid,
		).Scan(&pubWalletID, &pubBalanceBefore)
		if err != nil {
			return nil, fmt.Errorf("marketplace: publisher wallet: %w", err)
		}
	}

	var pubBalanceAfter string
	err = tx.QueryRow(ctx,
		`UPDATE user_wallets SET balance = balance + $1::numeric, updated_at = now()
			 WHERE user_id = $2
			 RETURNING balance::text`,
		pubAmountStr, pid,
	).Scan(&pubBalanceAfter)
	if err != nil {
		return nil, fmt.Errorf("marketplace: credit publisher: %w", err)
	}

	// 8. Record publisher wallet_transaction (sale credit).
	pubTxID := uuid.New()
	saleDesc := fmt.Sprintf("Strategy sale: %s", strategyTitle)
	if platformFee > 0 {
		saleDesc += fmt.Sprintf(" (platform fee: %.2f)", platformFee)
	}
	_, err = tx.Exec(ctx,
		fmt.Sprintf(`INSERT INTO wallet_transactions (id, wallet_id, user_id, tx_type, amount, balance_before, balance_after, description)
			 VALUES ($1, $2, $3, '%s', $4::numeric, $5::numeric, $6::numeric, $7)`, TxTypeSale),
		pubTxID, pubWalletID, pid, pubAmountStr, pubBalanceBefore, pubBalanceAfter, saleDesc,
	)
	if err != nil {
		return nil, fmt.Errorf("marketplace: record publisher transaction: %w", err)
	}

	// 9. Insert subscription row (target_user_id = publisher from DB).
	subID := uuid.New()
	if expiresAt != nil {
		err = tx.QueryRow(ctx,
			fmt.Sprintf(`INSERT INTO user_subscriptions (id, subscriber_user_id, target_user_id, target_strategy_id, kind, active, idempotency_key, expires_at)
				 VALUES ($1, $2, $3, $4, '%s', true, $5, %s)
				 RETURNING id`, subKind, *expiresAt),
			subID, uid, pid, sid, idempotencyKey,
		).Scan(&subID)
	} else {
		err = tx.QueryRow(ctx,
			fmt.Sprintf(`INSERT INTO user_subscriptions (id, subscriber_user_id, target_user_id, target_strategy_id, kind, active, idempotency_key)
				 VALUES ($1, $2, $3, $4, '%s', true, $5)
				 RETURNING id`, subKind),
			subID, uid, pid, sid, idempotencyKey,
		).Scan(&subID)
	}
	if err != nil {
		return nil, fmt.Errorf("marketplace: create subscription: %w", err)
	}

	// 10. Increment total_subscribers counter.
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
	publishedCacheClear()

	return &PurchaseResult{
		SubscriptionID: subID.String(),
		TransactionID:  buyerTxID.String(),
		AmountCharged:  amountStr,
		BalanceAfter:   buyerBalanceAfter,
	}, nil
}

// ── Admin Pricing ─────────────────────────────────────────────────────────────

// SetPricing updates the pricing model, amount, and platform fee rate for a published strategy.
func (s *Service) SetPricing(ctx context.Context, strategyID, priceModel string, priceAmount, platformFeeRate float64) error {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_strategies SET price_model=$2, price_amount=$3, platform_fee_rate=$4, updated_at=now() WHERE strategy_id=$1`,
		sid, priceModel, priceAmount, platformFeeRate)
	if err == nil {
		publishedCacheClear()
	}
	return err
}
