// Package marketplace — Phase 5.2: Bundle purchase operations.
package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// PurchaseBundle purchases a bundle, creating subscriptions for all strategies.
// The buyer is charged the bundle price (not individual strategy prices).
// Revenue split: publisher gets (1 - fee_rate) * price, platform gets fee_rate * price.
// The entire operation (bundle lookup, wallet ops, subscription creation) is wrapped
// in a single transaction with FOR UPDATE row locking for atomicity and consistency.
func (s *Service) PurchaseBundle(ctx context.Context, userID, bundleID, idempotencyKey string) (*PurchaseResult, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: purchase bundle: invalid user_id: %w", err)
	}
	bid, err := uuid.Parse(bundleID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: purchase bundle: invalid bundle_id: %w", err)
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("marketplace: purchase bundle: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Idempotency check: return existing purchase if already processed.
	if idempotencyKey != "" {
		var txID, amount, balance string
		err := tx.QueryRow(ctx,
			`SELECT wt.id::text, ABS(wt.amount)::text, w.balance::text
			 FROM wallet_transactions wt
			 JOIN user_wallets w ON w.user_id = wt.user_id
			 WHERE wt.idem_key = $1 AND wt.user_id = $2`,
			IdemKeyBuy+idempotencyKey, uid,
		).Scan(&txID, &amount, &balance)
		if err == nil {
			_ = tx.Rollback(ctx)
			return &PurchaseResult{
				SubscriptionID: bundleID,
				TransactionID:  txID,
				AmountCharged:  amount,
				BalanceAfter:   balance,
			}, nil
		}
		// For free bundles (no wallet transaction), check subscriptions.
		var subID string
		_ = tx.QueryRow(ctx,
			`SELECT id::text FROM user_subscriptions
			 WHERE subscriber_user_id = $1 AND idempotency_key LIKE $2 || '-%'
			 LIMIT 1`,
			uid, idempotencyKey,
		).Scan(&subID)
		if subID != "" {
			_ = tx.Rollback(ctx)
			return &PurchaseResult{
				SubscriptionID: bundleID,
				AmountCharged:  "0",
			}, nil
		}
	}

	// Read bundle details inside the transaction with row lock to prevent TOCTOU.
	var publisherID uuid.UUID
	var title, priceModel string
	var priceAmount decimal.Decimal
	err = tx.QueryRow(ctx,
		`SELECT publisher_id, title, price_model, price_amount
		 FROM marketplace_bundles WHERE id = $1 AND status = 'published' FOR UPDATE`,
		bid,
	).Scan(&publisherID, &title, &priceModel, &priceAmount)
	if err != nil {
		return nil, fmt.Errorf("marketplace: purchase bundle: not found: %w", err)
	}

	// Guard: cannot purchase your own bundle.
	if uid == publisherID {
		return nil, fmt.Errorf("marketplace: cannot purchase your own bundle")
	}

	// Read items inside the transaction for snapshot consistency.
	itemRows, err := tx.Query(ctx,
		`SELECT bi.strategy_id::text
		 FROM marketplace_bundle_items bi
		 WHERE bi.bundle_id = $1
		 ORDER BY bi.sort_order ASC`,
		bid)
	if err != nil {
		return nil, fmt.Errorf("marketplace: purchase bundle: get items: %w", err)
	}

	var items []string
	for itemRows.Next() {
		var sid string
		if err := itemRows.Scan(&sid); err != nil {
			itemRows.Close()
			return nil, fmt.Errorf("marketplace: purchase bundle: scan item: %w", err)
		}
		items = append(items, sid)
	}
	itemRows.Close()

	isFree := priceModel == PriceModelFree || !priceAmount.IsPositive()

	// Paid path: charge buyer, credit publisher and platform.
	var amountStr, balanceAfter, txIDStr, feeStr, pubAmountStr string
	if !isFree {
		feeRate, err := s.getEffectiveFeeRateTx(ctx, tx, publisherID.String())
		if err != nil {
			return nil, fmt.Errorf("marketplace: purchase bundle: get effective fee rate: %w", err)
		}

		// Lock buyer wallet.
		var buyerWalletID uuid.UUID
		err = tx.QueryRow(ctx,
			`SELECT id FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
			uid,
		).Scan(&buyerWalletID)
		if err != nil {
			return nil, fmt.Errorf("marketplace: purchase bundle: buyer wallet: %w", err)
		}

		// Charge buyer.
		amountStr = priceAmount.StringFixed(2)
		negAmountStr := "-" + amountStr
		buyerDesc := fmt.Sprintf("Bundle purchase: %s", title)
		buyerWallet, err := s.walletRepo.AdjustBalanceTx(ctx, tx, buyerWalletID, uid,
			negAmountStr, TxTypePurchase, buyerDesc, nil, IdemKeyBuy+idempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("marketplace: purchase bundle: charge buyer: %w", err)
		}
		balanceAfter = buyerWallet.Balance
		if buyerWallet.LastTransactionID != nil {
			txIDStr = buyerWallet.LastTransactionID.String()
		}

		// Credit publisher and platform via frozen settlement (Phase 5.4).
		feeDec := priceAmount.Mul(feeRate)
		pubDec := priceAmount.Sub(feeDec)
		pubAmountStr = pubDec.StringFixed(2)
		feeStr = feeDec.StringFixed(2)
	}

	// Create subscriptions for all strategies in the bundle.
	var firstSubID uuid.UUID
	for _, sidStr := range items {
		sid, _ := uuid.Parse(sidStr)
		subID := uuid.New()
		tag, err := tx.Exec(ctx,
			`INSERT INTO user_subscriptions (id, subscriber_user_id, target_user_id, target_strategy_id, kind, active, idempotency_key)
			 VALUES ($1, $2, $3, $4, 'purchase', true, $5)
			 ON CONFLICT DO NOTHING`,
			subID, uid, publisherID, sid, idempotencyKey+"-"+sidStr,
		)
		if err != nil {
			s.log.Warn("bundle: subscription creation failed",
				zap.String("strategy_id", sidStr), zap.Error(err))
			continue
		}
		if tag.RowsAffected() == 0 {
			continue // already exists, skip counter increment
		}
		if firstSubID == uuid.Nil {
			firstSubID = subID
		}
		if _, err := tx.Exec(ctx,
			`UPDATE marketplace_strategies SET total_subscribers = total_subscribers + 1 WHERE strategy_id = $1`,
			sid); err != nil {
			s.log.Warn("bundle: increment subscriber count failed",
				zap.String("strategy_id", sidStr), zap.Error(err))
		}
	}

	// Create frozen settlement — purchase_id must be a user_subscriptions.id
	// for subJoinOnClause and refund logic to work correctly.
	if !isFree && firstSubID != uuid.Nil {
		err = s.createFrozenSettlementTx(ctx, tx, firstSubID, uid, publisherID, amountStr, feeStr, pubAmountStr, DefaultRefundWindowDays)
		if err != nil {
			return nil, fmt.Errorf("marketplace: purchase bundle: create settlement: %w", err)
		}
	}

	// Increment bundle purchase count.
	_, err = tx.Exec(ctx,
		`UPDATE marketplace_bundles SET total_purchases = total_purchases + 1 WHERE id = $1`,
		bid)
	if err != nil {
		return nil, fmt.Errorf("marketplace: purchase bundle: increment count: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("marketplace: purchase bundle: commit: %w", err)
	}

	result := &PurchaseResult{
		SubscriptionID: bid.String(),
		TransactionID:  txIDStr,
		AmountCharged:  amountStr,
		BalanceAfter:   balanceAfter,
	}
	if isFree {
		result.AmountCharged = "0"
	}
	return result, nil
}

// DeleteBundle hides a bundle (soft delete by setting status to 'hidden').
// Only the bundle publisher or an admin can delete.
func (s *Service) DeleteBundle(ctx context.Context, bundleID, userID string, isAdmin bool) error {
	bid, err := uuid.Parse(bundleID)
	if err != nil {
		return fmt.Errorf("marketplace: delete bundle: invalid bundle_id: %w", err)
	}

	if !isAdmin {
		var publisherID uuid.UUID
		err = s.pg.QueryRow(ctx,
			`SELECT publisher_id FROM marketplace_bundles WHERE id = $1`,
			bid,
		).Scan(&publisherID)
		if err != nil {
			return fmt.Errorf("marketplace: delete bundle: not found: %w", err)
		}
		uid, err := uuid.Parse(userID)
		if err != nil || uid != publisherID {
			return fmt.Errorf("marketplace: delete bundle: not the owner")
		}
	}

	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_bundles SET status = 'hidden', updated_at = now() WHERE id = $1`,
		bid,
	)
	if err != nil {
		return fmt.Errorf("marketplace: delete bundle: %w", err)
	}
	return nil
}
