package marketplace

import (
	"context"
	"fmt"
	"time"

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

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("marketplace: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 0. Idempotency check inside transaction — serialized to prevent races.
	if idempotencyKey != "" {
		var existingSubID, existingTxID, existingAmount, existingBalance string
		err := tx.QueryRow(ctx,
			`SELECT us.id::text, wt.id::text, ABS(wt.amount)::text, w.balance::text
			 FROM user_subscriptions us
			 JOIN wallet_transactions wt ON wt.user_id = us.subscriber_user_id AND wt.tx_type = $1
			 JOIN user_wallets w ON w.user_id = us.subscriber_user_id
			 WHERE us.idempotency_key = $2 AND us.subscriber_user_id = $3
			 ORDER BY wt.created_at DESC LIMIT 1`,
			TxTypePurchase, idempotencyKey, uid,
		).Scan(&existingSubID, &existingTxID, &existingAmount, &existingBalance)
		if err == nil {
			tx.Rollback(ctx)
			return &PurchaseResult{
				SubscriptionID: existingSubID,
				TransactionID:  existingTxID,
				AmountCharged:  existingAmount,
				BalanceAfter:   existingBalance,
			}, nil
		}
	}

	// 1. Look up strategy price, publisher, and platform fee (source of truth from DB).
	var priceModel string
	var priceAmountStr string
	var strategyTitle string
	var dbPublisherID string
	var platformFeeRateStr string
	err = tx.QueryRow(ctx,
		`SELECT price_model, COALESCE(price_amount::text, '0'), title, publisher_id::text,
		        COALESCE(platform_fee_rate::text, '0')
		 FROM marketplace_strategies WHERE strategy_id = $1 AND status = 'published'`,
		sid,
	).Scan(&priceModel, &priceAmountStr, &strategyTitle, &dbPublisherID, &platformFeeRateStr)
	if err != nil {
		return nil, fmt.Errorf("marketplace: strategy not published")
	}
	priceDec, err := decimal.NewFromString(priceAmountStr)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid price_amount in DB: %w", err)
	}
	feeRateDec, err := decimal.NewFromString(platformFeeRateStr)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid platform_fee_rate in DB: %w", err)
	}
	if (priceModel != PriceModelOnce && priceModel != PriceModelSubscription) || !priceDec.IsPositive() {
		return nil, fmt.Errorf("marketplace: strategy is not purchasable")
	}

	// Determine subscription kind and expiry.
	subKind := SubKindPurchase
	var expiresAt *time.Time // nil = permanent
	if priceModel == PriceModelSubscription {
		subKind = SubKindSubscription
		exp := time.Now().Add(30 * 24 * time.Hour)
		expiresAt = &exp
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

	amountStr := priceDec.StringFixed(2)
	negAmountStr := "-" + amountStr

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
		`INSERT INTO wallet_transactions (id, wallet_id, user_id, tx_type, amount, balance_before, balance_after, description)
			 VALUES ($1, $2, $3, $4, $5::numeric, $6::numeric, $7::numeric, $8)
			 RETURNING id`,
		buyerTxID, buyerWalletID, uid, TxTypePurchase, negAmountStr, buyerBalanceBefore, buyerBalanceAfter, buyerDesc,
	).Scan(&buyerTxID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: record buyer transaction: %w", err)
	}

	// 7. Credit publisher wallet (minus platform fee). Use decimal for precise arithmetic.
	feeDec := priceDec.Mul(feeRateDec)
	pubDec := priceDec.Sub(feeDec)
	pubAmountStr := pubDec.StringFixed(2)
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
	if feeDec.GreaterThan(decimal.Zero) {
		saleDesc += fmt.Sprintf(" (platform fee: %s)", feeDec.StringFixed(2))
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO wallet_transactions (id, wallet_id, user_id, tx_type, amount, balance_before, balance_after, description)
			 VALUES ($1, $2, $3, $4, $5::numeric, $6::numeric, $7::numeric, $8)`,
		pubTxID, pubWalletID, pid, TxTypeSale, pubAmountStr, pubBalanceBefore, pubBalanceAfter, saleDesc,
	)
	if err != nil {
		return nil, fmt.Errorf("marketplace: record publisher transaction: %w", err)
	}

	// 8b. Credit platform fee to system wallet with full balance tracking.
	if feeDec.GreaterThan(decimal.Zero) {
		feeStr := feeDec.StringFixed(2)

		// Ensure system wallet exists.
		var sysWalletID uuid.UUID
		var sysBalBefore string
		err = tx.QueryRow(ctx,
			`INSERT INTO user_wallets (user_id) VALUES ($1)
			 ON CONFLICT (user_id) DO UPDATE SET user_id = $1
			 RETURNING id, balance::text`,
			SystemUserID,
		).Scan(&sysWalletID, &sysBalBefore)
		if err != nil {
			return nil, fmt.Errorf("marketplace: system wallet: %w", err)
		}

		// Credit system wallet.
		var sysBalAfter string
		err = tx.QueryRow(ctx,
			`UPDATE user_wallets SET balance = balance + $1::numeric, updated_at = now()
			 WHERE id = $2 RETURNING balance::text`,
			feeStr, sysWalletID,
		).Scan(&sysBalAfter)
		if err != nil {
			return nil, fmt.Errorf("marketplace: credit system wallet: %w", err)
		}

		// Record platform fee transaction.
		feeTxID := uuid.New()
		feeDesc := fmt.Sprintf("Platform fee from strategy sale: %s", strategyTitle)
		_, err = tx.Exec(ctx,
			`INSERT INTO wallet_transactions (id, wallet_id, user_id, tx_type, amount, balance_before, balance_after, description)
				 VALUES ($1, $2, $3, $4, $5::numeric, $6::numeric, $7::numeric, $8)`,
			feeTxID, sysWalletID, SystemUserID, TxTypePlatformFee, feeStr, sysBalBefore, sysBalAfter, feeDesc,
		)
		if err != nil {
			return nil, fmt.Errorf("marketplace: record platform fee: %w", err)
		}
	}

	// 9. Insert subscription row (target_user_id = publisher from DB).
	subID := uuid.New()
	err = tx.QueryRow(ctx,
		fmt.Sprintf(`INSERT INTO user_subscriptions (id, subscriber_user_id, target_user_id, target_strategy_id, kind, active, idempotency_key, expires_at)
			 VALUES ($1, $2, $3, $4, '%s', true, $5, $6)
			 RETURNING id`, subKind),
		subID, uid, pid, sid, idempotencyKey, expiresAt,
	).Scan(&subID)
	if err != nil {
		// If unique violation on idempotency_key, another request won the race.
		// Return the now-existing subscription gracefully.
		if idempotencyKey != "" && isUniqueViolation(err) {
			tx.Rollback(ctx)
			var dupSubID, dupTxID, dupAmount, dupBalance string
			if err2 := s.pg.QueryRow(ctx,
				`SELECT us.id::text, wt.id::text, ABS(wt.amount)::text, w.balance::text
				 FROM user_subscriptions us
				 JOIN wallet_transactions wt ON wt.user_id = us.subscriber_user_id AND wt.tx_type = $1
				 JOIN user_wallets w ON w.user_id = us.subscriber_user_id
				 WHERE us.idempotency_key = $2 AND us.subscriber_user_id = $3
				 ORDER BY wt.created_at DESC LIMIT 1`,
				TxTypePurchase, idempotencyKey, uid,
			).Scan(&dupSubID, &dupTxID, &dupAmount, &dupBalance); err2 == nil {
				return &PurchaseResult{
					SubscriptionID: dupSubID, TransactionID: dupTxID,
					AmountCharged: dupAmount, BalanceAfter: dupBalance,
				}, nil
			}
		}
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
func (s *Service) SetPricing(ctx context.Context, strategyID, priceModel, priceAmount, platformFeeRate string) error {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_strategies SET price_model=$2, price_amount=$3::numeric, platform_fee_rate=$4::numeric, updated_at=now() WHERE strategy_id=$1`,
		sid, priceModel, priceAmount, platformFeeRate)
	if err == nil {
		publishedCacheClear()
	}
	return err
}
