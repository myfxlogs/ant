// Package marketplace — Phase 5.2: Bundle purchase operations.
package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/jackc/pgx/v5"
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

	if result, found := s.checkBundleIdempotency(ctx, tx, uid, bid, idempotencyKey); found {
		return result, nil
	}

	publisherID, title, priceModel, priceAmount, err := s.loadBundleForPurchase(ctx, tx, bid)
	if err != nil {
		return nil, err
	}
	if uid == publisherID {
		return nil, fmt.Errorf("marketplace: cannot purchase your own bundle")
	}

	items, totalItemCount, err := s.loadBundleItems(ctx, tx, bid)
	if err != nil {
		return nil, err
	}

	items = s.filterOwnedItems(ctx, tx, uid, items)

	if len(items) == 0 {
		_ = tx.Rollback(ctx)
		return &PurchaseResult{
			SubscriptionID: bid.String(),
			AmountCharged:  "0",
		}, nil
	}

	isFree := priceModel == PriceModelFree || !priceAmount.IsPositive()
	effectivePrice := priceAmount
	if !isFree && totalItemCount > 0 && len(items) < totalItemCount {
		ratio := decimal.NewFromInt(int64(len(items))).Div(decimal.NewFromInt(int64(totalItemCount)))
		effectivePrice = priceAmount.Mul(ratio)
	}

	amountStr, balanceAfter, txIDStr, feeStr, pubAmountStr, err := s.chargeBundleBuyer(ctx, tx, uid, publisherID, title, effectivePrice, isFree, idempotencyKey)
	if err != nil {
		return nil, err
	}

	firstSubID, err := s.createBundleSubscriptions(ctx, tx, uid, publisherID, bid, items, idempotencyKey)
	if err != nil {
		return nil, err
	}

	if !isFree && firstSubID != uuid.Nil {
		err = s.createFrozenSettlementTx(ctx, tx, firstSubID, uid, publisherID, amountStr, feeStr, pubAmountStr, DefaultRefundWindowDays, &bid)
		if err != nil {
			return nil, fmt.Errorf("marketplace: purchase bundle: create settlement: %w", err)
		}
	}

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

func (s *Service) checkBundleIdempotency(ctx context.Context, tx pgx.Tx, uid uuid.UUID, bid uuid.UUID, idempotencyKey string) (*PurchaseResult, bool) {
	if idempotencyKey == "" {
		return nil, false
	}
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
			SubscriptionID: bid.String(),
			TransactionID:  txID,
			AmountCharged:  amount,
			BalanceAfter:   balance,
		}, true
	}
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
			SubscriptionID: bid.String(),
			AmountCharged:  "0",
		}, true
	}
	return nil, false
}

func (s *Service) loadBundleForPurchase(ctx context.Context, tx pgx.Tx, bid uuid.UUID) (publisherID uuid.UUID, title, priceModel string, priceAmount decimal.Decimal, err error) {
	err = tx.QueryRow(ctx,
		`SELECT publisher_id, title, price_model, price_amount
		 FROM marketplace_bundles WHERE id = $1 AND status = 'published' FOR UPDATE`,
		bid,
	).Scan(&publisherID, &title, &priceModel, &priceAmount)
	if err != nil {
		err = fmt.Errorf("marketplace: purchase bundle: not found: %w", err)
	}
	return
}

func (s *Service) loadBundleItems(ctx context.Context, tx pgx.Tx, bid uuid.UUID) ([]string, int, error) {
	itemRows, err := tx.Query(ctx,
		`SELECT bi.strategy_id::text
		 FROM marketplace_bundle_items bi
		 WHERE bi.bundle_id = $1
		 ORDER BY bi.sort_order ASC`,
		bid)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: purchase bundle: get items: %w", err)
	}
	var items []string
	for itemRows.Next() {
		var sid string
		if err := itemRows.Scan(&sid); err != nil {
			itemRows.Close()
			return nil, 0, fmt.Errorf("marketplace: purchase bundle: scan item: %w", err)
		}
		items = append(items, sid)
	}
	itemRows.Close()
	return items, len(items), nil
}

func (s *Service) filterOwnedItems(ctx context.Context, tx pgx.Tx, uid uuid.UUID, items []string) []string {
	if len(items) == 0 {
		return items
	}
	parsedItemIDs := make([]uuid.UUID, 0, len(items))
	for _, sidStr := range items {
		parsedID, perr := uuid.Parse(sidStr)
		if perr == nil {
			parsedItemIDs = append(parsedItemIDs, parsedID)
		}
	}
	existingRows, eerr := tx.Query(ctx,
		`SELECT target_strategy_id::text FROM user_subscriptions
		 WHERE subscriber_user_id = $1 AND target_strategy_id = ANY($2) AND active = true`,
		uid, parsedItemIDs)
	if eerr != nil {
		return items
	}
	owned := make(map[string]bool)
	for existingRows.Next() {
		var ownedSID string
		if err := existingRows.Scan(&ownedSID); err == nil {
			owned[ownedSID] = true
		}
	}
	existingRows.Close()
	if len(owned) == 0 {
		return items
	}
	filtered := items[:0]
	for _, sidStr := range items {
		if !owned[sidStr] {
			filtered = append(filtered, sidStr)
		}
	}
	return filtered
}

func (s *Service) chargeBundleBuyer(ctx context.Context, tx pgx.Tx, uid, publisherID uuid.UUID, title string, effectivePrice decimal.Decimal, isFree bool, idempotencyKey string) (amountStr, balanceAfter, txIDStr, feeStr, pubAmountStr string, err error) {
	if isFree {
		return "", "", "", "", "", nil
	}
	feeRate, err := s.getEffectiveFeeRateTx(ctx, tx, publisherID.String())
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("marketplace: purchase bundle: get effective fee rate: %w", err)
	}
	var buyerWalletID uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT id FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
		uid,
	).Scan(&buyerWalletID)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("marketplace: purchase bundle: buyer wallet: %w", err)
	}
	amountStr = effectivePrice.StringFixed(2)
	negAmountStr := "-" + amountStr
	buyerDesc := fmt.Sprintf("Bundle purchase: %s", title)
	buyerWallet, err := s.walletRepo.AdjustBalanceTx(ctx, tx, buyerWalletID, uid,
		negAmountStr, TxTypePurchase, buyerDesc, nil, IdemKeyBuy+idempotencyKey)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("marketplace: purchase bundle: charge buyer: %w", err)
	}
	balanceAfter = buyerWallet.Balance
	if buyerWallet.LastTransactionID != nil {
		txIDStr = buyerWallet.LastTransactionID.String()
	}
	feeDec := effectivePrice.Mul(feeRate)
	pubDec := effectivePrice.Sub(feeDec)
	pubAmountStr = pubDec.StringFixed(2)
	feeStr = feeDec.StringFixed(2)
	return
}

func (s *Service) createBundleSubscriptions(ctx context.Context, tx pgx.Tx, uid, publisherID, bid uuid.UUID, items []string, idempotencyKey string) (uuid.UUID, error) {
	var firstSubID uuid.UUID
	for _, sidStr := range items {
		sid, _ := uuid.Parse(sidStr)
		subID := uuid.New()
		tag, err := tx.Exec(ctx,
			`INSERT INTO user_subscriptions (id, subscriber_user_id, target_user_id, target_strategy_id, kind, active, idempotency_key, bundle_id)
			 VALUES ($1, $2, $3, $4, 'purchase', true, $5, $6)
			 ON CONFLICT DO NOTHING`,
			subID, uid, publisherID, sid, idempotencyKey+"-"+sidStr, bid,
		)
		if err != nil {
			s.log.Warn("bundle: subscription creation failed",
				zap.String("strategy_id", sidStr), zap.Error(err))
			continue
		}
		if tag.RowsAffected() == 0 {
			continue
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
	return firstSubID, nil
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
