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

	if idempotencyKey != "" {
		existing, err := s.lookupExistingPurchase(ctx, tx, idempotencyKey, uid)
		if err == nil {
			_ = tx.Rollback(ctx)
			return existing, nil
		}
	}

	stratInfo, err := s.fetchStrategyForPurchase(ctx, tx, sid)
	if err != nil {
		return nil, err
	}

	if _, err := s.SettleExpired(ctx, stratInfo.publisherID.String()); err != nil {
		s.log.Warn("purchase: lazy settlement failed for publisher",
			zap.String("publisherID", stratInfo.publisherID.String()), zap.Error(err))
	}

	feeRateDec, err := s.getEffectiveFeeRateTx(ctx, tx, stratInfo.publisherID.String())
	if err != nil {
		return nil, fmt.Errorf("marketplace: get effective fee rate: %w", err)
	}
	if (stratInfo.priceModel != PriceModelOnce && stratInfo.priceModel != PriceModelSubscription) || !stratInfo.price.IsPositive() {
		return nil, fmt.Errorf("marketplace: strategy is not purchasable")
	}

	finalAmount, couponID, err := s.applyCoupon(ctx, tx, couponCode, sid, stratInfo.price)
	if err != nil {
		return nil, err
	}

	subKind := SubKindPurchase
	var expiresAt *time.Time
	if stratInfo.priceModel == PriceModelSubscription {
		subKind = SubKindSubscription
		exp := time.Now().Add(30 * 24 * time.Hour)
		expiresAt = &exp
	}

	if uid == stratInfo.publisherID {
		return nil, fmt.Errorf("marketplace: cannot purchase your own strategy")
	}

	var existing string
	_ = tx.QueryRow(ctx,
		`SELECT id::text FROM user_subscriptions WHERE subscriber_user_id = $1 AND target_strategy_id = $2 AND active = true`,
		uid, sid,
	).Scan(&existing)
	if existing != "" {
		return nil, fmt.Errorf("marketplace: already subscribed")
	}

	amountStr, _, buyerBalanceAfter, buyerTxID, feeStr, pubAmountStr, err := s.chargeBuyerAndCalcFees(ctx, tx, uid, finalAmount, feeRateDec, stratInfo.title, idempotencyKey)
	if err != nil {
		return nil, err
	}

	subID, err := s.insertSubscription(ctx, tx, uid, stratInfo.publisherID, sid, subKind, idempotencyKey, expiresAt)
	if err != nil {
		if idempotencyKey != "" && isUniqueViolation(err) {
			_ = tx.Rollback(ctx)
			if dup, err2 := s.lookupExistingPurchase(ctx, s.pg, idempotencyKey, uid); err2 == nil {
				return dup, nil
			}
		}
		return nil, fmt.Errorf("marketplace: create subscription: %w", err)
	}

	err = s.createFrozenSettlementTx(ctx, tx, subID, uid, stratInfo.publisherID, amountStr, feeStr, pubAmountStr, stratInfo.refundWindowDays, nil)
	if err != nil {
		return nil, fmt.Errorf("marketplace: create settlement: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE marketplace_strategies SET total_subscribers = total_subscribers + 1 WHERE strategy_id = $1`,
		sid,
	)
	if err != nil {
		return nil, fmt.Errorf("marketplace: update subscriber count: %w", err)
	}

	if err := s.consumeCoupon(ctx, tx, couponID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("marketplace: commit purchase: %w", err)
	}
	s.pubCache.clear()

	return &PurchaseResult{
		SubscriptionID: subID.String(),
		TransactionID:  buyerTxID,
		AmountCharged:  amountStr,
		BalanceAfter:   buyerBalanceAfter,
	}, nil
}

func (s *Service) chargeBuyerAndCalcFees(ctx context.Context, tx pgx.Tx, uid uuid.UUID, finalAmount, feeRateDec decimal.Decimal, title, idempotencyKey string) (amountStr, negAmountStr, balanceAfter, txIDStr, feeStr, pubAmountStr string, err error) {
	var buyerWalletID uuid.UUID
	var buyerBalanceBefore string
	err = tx.QueryRow(ctx,
		`SELECT id, balance::text FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
		uid,
	).Scan(&buyerWalletID, &buyerBalanceBefore)
	if err != nil {
		err = fmt.Errorf("marketplace: wallet not found")
		return
	}
	if buyerBalanceBefore == "" {
		err = fmt.Errorf("marketplace: wallet balance unavailable")
		return
	}

	amountStr = finalAmount.StringFixed(2)
	negAmountStr = "-" + amountStr

	buyerDesc := fmt.Sprintf("Purchase strategy: %s", title)
	buyerWallet, e := s.walletRepo.AdjustBalanceTx(ctx, tx, buyerWalletID, uid,
		negAmountStr, TxTypePurchase, buyerDesc, nil, IdemKeyBuy+idempotencyKey)
	if e != nil {
		err = fmt.Errorf("marketplace: charge buyer: %w", e)
		return
	}
	balanceAfter = buyerWallet.Balance
	buyerTxID := uuid.Nil
	if buyerWallet.LastTransactionID != nil {
		buyerTxID = *buyerWallet.LastTransactionID
	}
	txIDStr = buyerTxID.String()

	feeDec := finalAmount.Mul(feeRateDec)
	pubDec := finalAmount.Sub(feeDec)
	pubAmountStr = pubDec.StringFixed(2)
	feeStr = feeDec.StringFixed(2)
	return
}

type strategyPurchaseInfo struct {
	priceModel        string
	price             decimal.Decimal
	title             string
	publisherID       uuid.UUID
	refundWindowDays  int
}

func (s *Service) fetchStrategyForPurchase(ctx context.Context, tx pgx.Tx, sid uuid.UUID) (*strategyPurchaseInfo, error) {
	var priceModel, priceAmountStr, title, dbPublisherID string
	var refundWindowDays int
	err := tx.QueryRow(ctx,
		`SELECT price_model, COALESCE(price_amount::text, '0'), title, publisher_id::text, COALESCE(refund_window_days, 7)
		 FROM marketplace_strategies WHERE strategy_id = $1 AND status = 'published'`,
		sid,
	).Scan(&priceModel, &priceAmountStr, &title, &dbPublisherID, &refundWindowDays)
	if err != nil {
		return nil, fmt.Errorf("marketplace: strategy not published")
	}
	priceDec, err := decimal.NewFromString(priceAmountStr)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid price_amount in DB: %w", err)
	}
	pid, err := uuid.Parse(dbPublisherID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid publisher_id in DB: %w", err)
	}
	return &strategyPurchaseInfo{
		priceModel:       priceModel,
		price:            priceDec,
		title:            title,
		publisherID:      pid,
		refundWindowDays: refundWindowDays,
	}, nil
}

func (s *Service) applyCoupon(ctx context.Context, tx pgx.Tx, couponCode string, sid uuid.UUID, price decimal.Decimal) (decimal.Decimal, string, error) {
	if couponCode == "" {
		return price, "", nil
	}
	cp, err := s.validateCouponTx(ctx, tx, couponCode, sid, price)
	if err != nil {
		return decimal.Zero, "", fmt.Errorf("marketplace: validate coupon: %w", err)
	}
	if !cp.Valid {
		return decimal.Zero, "", fmt.Errorf("marketplace: %s", cp.ErrorMessage)
	}
	return cp.FinalAmount, cp.ID, nil
}

func (s *Service) insertSubscription(ctx context.Context, tx pgx.Tx, uid, pid, sid uuid.UUID, subKind, idempotencyKey string, expiresAt *time.Time) (uuid.UUID, error) {
	subID := uuid.New()
	var idemKey *string
	if idempotencyKey != "" {
		idemKey = &idempotencyKey
	}
	err := tx.QueryRow(ctx,
		`INSERT INTO user_subscriptions (id, subscriber_user_id, target_user_id, target_strategy_id, kind, active, idempotency_key, expires_at)
		 VALUES ($1, $2, $3, $4, $5, true, $6, $7)
		 RETURNING id`,
		subID, uid, pid, sid, subKind, idemKey, expiresAt,
	).Scan(&subID)
	return subID, err
}

func (s *Service) consumeCoupon(ctx context.Context, tx pgx.Tx, couponID string) error {
	if couponID == "" {
		return nil
	}
	var consumed bool
	err := tx.QueryRow(ctx,
		`UPDATE marketplace_coupons
		 SET used_count = used_count + 1
		 WHERE id = $1 AND (max_uses = 0 OR used_count < max_uses)
		 RETURNING true`,
		couponID,
	).Scan(&consumed)
	if err != nil {
		return fmt.Errorf("marketplace: coupon exhausted: %w", err)
	}
	return nil
}

// ── Admin Pricing ─────────────────────────────────────────────────────────────

// SetPricing updates the pricing model, amount, and platform fee rate for a published strategy.
// M3: Verifies caller is the strategy publisher (defense-in-depth, handler already checks admin).
func (s *Service) SetPricing(ctx context.Context, userID, strategyID, priceModel, priceAmount, platformFeeRate string) error {
	switch priceModel {
	case PriceModelFree, PriceModelOnce, PriceModelSubscription:
	default:
		return fmt.Errorf("marketplace: unsupported price_model %q", priceModel)
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}

	// M3: Verify ownership — only the publisher can change pricing.
	var dbPublisherID string
	err = s.pg.QueryRow(ctx,
		`SELECT publisher_id::text FROM marketplace_strategies WHERE strategy_id = $1`,
		sid,
	).Scan(&dbPublisherID)
	if err != nil {
		return fmt.Errorf("marketplace: strategy not found: %w", err)
	}
	if dbPublisherID != userID {
		return fmt.Errorf("marketplace: not the strategy owner")
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
