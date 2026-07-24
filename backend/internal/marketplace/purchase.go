package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// ── Purchase ──────────────────────────────────────────────────────────────────

// queryRower is the minimal interface for executing a QueryRow, satisfied by
// both pgx.Tx and *pgxpool.Pool.
type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// lookupExistingPurchase reads an already-completed purchase by idempotency key.
// Used for idempotency check and duplicate race fallback. Caller must handle
// pgx.ErrNoRows (no existing purchase found).
func (s *Service) lookupExistingPurchase(ctx context.Context, q queryRower, idempotencyKey string, uid uuid.UUID) (*PurchaseResult, error) {
	var subID, txID, amount, balance string
	err := q.QueryRow(ctx,
		`SELECT us.id::text, wt.id::text, ABS(wt.amount)::text, w.balance::text
		 FROM user_subscriptions us
		 LEFT JOIN wallet_transactions wt ON wt.idem_key = $3 || us.idempotency_key
		 JOIN user_wallets w ON w.user_id = us.subscriber_user_id
		 WHERE us.idempotency_key = $1 AND us.subscriber_user_id = $2`,
		idempotencyKey, uid, IdemKeyBuy,
	).Scan(&subID, &txID, &amount, &balance)
	if err != nil {
		return nil, err
	}
	return &PurchaseResult{
		SubscriptionID: subID,
		TransactionID:  txID,
		AmountCharged:  amount,
		BalanceAfter:   balance,
	}, nil
}

// PurchaseStrategy atomically charges the user's wallet, credits the publisher,
// and creates a subscription — all in a single DB transaction with FOR UPDATE
// row locking to prevent races.
func (s *Service) PurchaseStrategy(ctx context.Context, userID, strategyID, couponCode, idempotencyKey string) (*PurchaseResult, error) {
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
	defer func() { _ = tx.Rollback(ctx) }()

	// 0. Idempotency check inside transaction — serialized to prevent races.
	if idempotencyKey != "" {
		existing, err := s.lookupExistingPurchase(ctx, tx, idempotencyKey, uid)
		if err == nil {
			_ = tx.Rollback(ctx)
			return existing, nil
		}
	}

	// 1. Look up strategy price, publisher, platform fee, and refund window (source of truth from DB).
	var priceModel string
	var priceAmountStr string
	var strategyTitle string
	var dbPublisherID string
	var dbRefundWindowDays int
	err = tx.QueryRow(ctx,
		`SELECT price_model, COALESCE(price_amount::text, '0'), title, publisher_id::text, COALESCE(refund_window_days, 7)
		 FROM marketplace_strategies WHERE strategy_id = $1 AND status = 'published'`,
		sid,
	).Scan(&priceModel, &priceAmountStr, &strategyTitle, &dbPublisherID, &dbRefundWindowDays)
	if err != nil {
		return nil, fmt.Errorf("marketplace: strategy not published")
	}
	priceDec, err := decimal.NewFromString(priceAmountStr)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid price_amount in DB: %w", err)
	}
	// Use tiered fee rate based on publisher's sales volume.
	// Read inside the transaction for snapshot consistency.
	pid, err := uuid.Parse(dbPublisherID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid publisher_id in DB: %w", err)
	}

	// Phase 5.4: Lazy settlement — settle publisher's expired frozen balances
	// before fee tier calculation so settled sales count is up-to-date.
	if _, err := s.SettleExpired(ctx, pid.String()); err != nil {
		s.log.Warn("purchase: lazy settlement failed for publisher",
			zap.String("publisherID", pid.String()), zap.Error(err))
	}

	feeRateDec, err := s.getEffectiveFeeRateTx(ctx, tx, pid.String())
	if err != nil {
		return nil, fmt.Errorf("marketplace: get effective fee rate: %w", err)
	}
	if (priceModel != PriceModelOnce && priceModel != PriceModelSubscription) || !priceDec.IsPositive() {
		return nil, fmt.Errorf("marketplace: strategy is not purchasable")
	}

	// Apply coupon if provided (validated inside the transaction to prevent TOCTOU race).
	finalAmount := priceDec
	var couponID string
	if couponCode != "" {
		cp, err := s.validateCouponTx(ctx, tx, couponCode, sid, priceDec)
		if err != nil {
			return nil, fmt.Errorf("marketplace: validate coupon: %w", err)
		}
		if !cp.Valid {
			return nil, fmt.Errorf("marketplace: %s", cp.ErrorMessage)
		}
		finalAmount = cp.FinalAmount
		couponID = cp.ID
	}

	// Determine subscription kind and expiry.
	subKind := SubKindPurchase
	var expiresAt *time.Time // nil = permanent
	if priceModel == PriceModelSubscription {
		subKind = SubKindSubscription
		exp := time.Now().Add(30 * 24 * time.Hour)
		expiresAt = &exp
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

	amountStr := finalAmount.StringFixed(2)
	negAmountStr := "-" + amountStr

	// 5. Deduct buyer balance via AdjustBalanceTx (hash chain + idempotency + ledger_outbox).
	buyerDesc := fmt.Sprintf("Purchase strategy: %s", strategyTitle)
	buyerWallet, err := s.walletRepo.AdjustBalanceTx(ctx, tx, buyerWalletID, uid,
		negAmountStr, TxTypePurchase, buyerDesc, nil, IdemKeyBuy+idempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("marketplace: charge buyer: %w", err)
	}
	buyerBalanceAfter := buyerWallet.Balance
	buyerTxID := uuid.Nil
	if buyerWallet.LastTransactionID != nil {
		buyerTxID = *buyerWallet.LastTransactionID
	}

	// 6. Compute fee split — publisher and platform amounts are recorded in a
	//    frozen settlement row, NOT credited to wallets yet. Settlement happens
	//    lazily after the refund window expires (see settlement.go SettleExpired).
	feeDec := finalAmount.Mul(feeRateDec)
	pubDec := finalAmount.Sub(feeDec)
	pubAmountStr := pubDec.StringFixed(2)
	feeStr := feeDec.StringFixed(2)

	// 7. Insert subscription row (target_user_id = publisher from DB).
	subID := uuid.New()
	err = tx.QueryRow(ctx,
		`INSERT INTO user_subscriptions (id, subscriber_user_id, target_user_id, target_strategy_id, kind, active, idempotency_key, expires_at)
			 VALUES ($1, $2, $3, $4, $5, true, $6, $7)
			 RETURNING id`,
		subID, uid, pid, sid, subKind, idempotencyKey, expiresAt,
	).Scan(&subID)
	if err != nil {
		// If unique violation on idempotency_key, another request won the race.
		// Return the now-existing subscription gracefully.
		if idempotencyKey != "" && isUniqueViolation(err) {
			_ = tx.Rollback(ctx)
			if dup, err2 := s.lookupExistingPurchase(ctx, s.pg, idempotencyKey, uid); err2 == nil {
				return dup, nil
			}
		}
		return nil, fmt.Errorf("marketplace: create subscription: %w", err)
	}

	// 8. Create frozen settlement record — replaces direct publisher/platform credits.
	err = s.createFrozenSettlementTx(ctx, tx, subID, uid, pid, amountStr, feeStr, pubAmountStr, dbRefundWindowDays)
	if err != nil {
		return nil, fmt.Errorf("marketplace: create settlement: %w", err)
	}

	// 9. Increment total_subscribers counter.
	_, err = tx.Exec(ctx,
		`UPDATE marketplace_strategies SET total_subscribers = total_subscribers + 1 WHERE strategy_id = $1`,
		sid,
	)
	if err != nil {
		return nil, fmt.Errorf("marketplace: update subscriber count: %w", err)
	}

	// 10. If a coupon was applied, atomically consume it.
	if couponID != "" {
		var consumed bool
		err = tx.QueryRow(ctx,
			`UPDATE marketplace_coupons
			 SET used_count = used_count + 1
			 WHERE id = $1 AND (max_uses = 0 OR used_count < max_uses)
			 RETURNING true`,
			couponID,
		).Scan(&consumed)
		if err != nil {
			return nil, fmt.Errorf("marketplace: coupon exhausted: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("marketplace: commit purchase: %w", err)
	}
	s.pubCache.clear()

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
	switch priceModel {
	case PriceModelFree, PriceModelOnce, PriceModelSubscription:
	default:
		return fmt.Errorf("marketplace: unsupported price_model %q", priceModel)
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}

	// Read old price for notification.
	var oldPrice, title string
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(price_amount::text,'0'), COALESCE(title,'') FROM marketplace_strategies WHERE strategy_id = $1`,
		sid,
	).Scan(&oldPrice, &title)

	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_strategies SET price_model=$2, price_amount=$3::numeric, platform_fee_rate=$4::numeric, updated_at=now() WHERE strategy_id=$1`,
		sid, priceModel, priceAmount, platformFeeRate)
	if err == nil {
		s.pubCache.clear()
		// Notify subscribers of price change.
		if oldPrice != priceAmount {
			go s.notifyPriceChange(context.WithoutCancel(ctx), sid, title, oldPrice, priceAmount)
		}
	}
	return err
}
