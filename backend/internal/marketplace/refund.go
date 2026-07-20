package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── Refund ─────────────────────────────────────────────────────────────────────

// RefundResult holds the outcome of a refund operation.
type RefundResult struct {
	SubscriptionID string
	RefundTxID     string
	AmountRefunded string
	BalanceAfter   string
}

// RefundPurchase reverses a paid strategy purchase: credits the buyer back,
// debits the publisher, deactivates the subscription, and decrements the
// subscriber counter. All steps run in a single DB transaction.
func (s *Service) RefundPurchase(ctx context.Context, userID, subscriptionID string) (*RefundResult, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid user_id: %w", err)
	}
	sid, err := uuid.Parse(subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid subscription_id: %w", err)
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("marketplace: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Look up subscription — must be an active purchase belonging to this user.
	var subTargetUserID, subStrategyID, subKind string
	var subActive bool
	err = tx.QueryRow(ctx,
		`SELECT target_user_id::text, target_strategy_id::text, kind, active
		 FROM user_subscriptions WHERE id = $1 AND subscriber_user_id = $2 FOR UPDATE`,
		sid, uid,
	).Scan(&subTargetUserID, &subStrategyID, &subKind, &subActive)
	if err != nil {
		return nil, fmt.Errorf("marketplace: subscription not found")
	}
	if !subActive {
		return nil, fmt.Errorf("marketplace: subscription already inactive")
	}
	if subKind != SubKindPurchase {
		return nil, fmt.Errorf("marketplace: only purchased subscriptions can be refunded")
	}

	// 2. Find the original purchase transaction (buyer's debit).
	var purchaseAmount string
	err = tx.QueryRow(ctx,
		`SELECT amount::text FROM wallet_transactions
		 WHERE user_id = $1 AND tx_type = $2
		 ORDER BY created_at DESC LIMIT 1`,
		uid, TxTypePurchase,
	).Scan(&purchaseAmount)
	if err != nil {
		return nil, fmt.Errorf("marketplace: original purchase transaction not found")
	}

	// 3. Refund buyer wallet via AdjustBalanceTx (hash chain + idempotency).
	var buyerWalletID uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT id FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
		uid,
	).Scan(&buyerWalletID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: buyer wallet not found")
	}

	// amount is negative in the transaction; use absolute value for credit.
	absAmount := purchaseAmount
	if len(absAmount) > 0 && absAmount[0] == '-' {
		absAmount = absAmount[1:]
	}

	refundDesc := fmt.Sprintf("Refund for subscription %s", subscriptionID)
	buyerWallet, err := s.walletRepo.AdjustBalanceTx(ctx, tx, buyerWalletID, uid,
		absAmount, TxTypeRefund, refundDesc, nil, "mkt-refund-"+subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: refund buyer: %w", err)
	}
	buyerBalAfter := buyerWallet.Balance
	refundTxID := uuid.Nil
	if buyerWallet.LastTransactionID != nil {
		refundTxID = *buyerWallet.LastTransactionID
	}

	// 5. Find publisher's original sale transaction to get the net amount
	//    they actually received (after platform fee deduction).
	var pubNetReceived string
	err = tx.QueryRow(ctx,
		`SELECT amount::text FROM wallet_transactions
		 WHERE user_id = $1 AND tx_type = $2
		 ORDER BY created_at DESC LIMIT 1`,
		subTargetUserID, TxTypeSale,
	).Scan(&pubNetReceived)
	if err != nil {
		// No sale transaction found — skip publisher debit.
		pubNetReceived = "0"
	}
	// Normalize: remove leading sign if present.
	if len(pubNetReceived) > 0 && pubNetReceived[0] == '-' {
		pubNetReceived = pubNetReceived[1:]
	}

	// 6. Debit publisher by the net amount they actually received.
	if pubNetReceived != "0" {
		pubUUID, _ := uuid.Parse(subTargetUserID)
		var pubWalletID uuid.UUID
		err = tx.QueryRow(ctx,
			`SELECT id FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
			pubUUID,
		).Scan(&pubWalletID)
		if err == nil {
			negNet := "-" + pubNetReceived
			revDesc := fmt.Sprintf("Refund reversal for subscription %s", subscriptionID)
			_, err = s.walletRepo.AdjustBalanceTx(ctx, tx, pubWalletID, pubUUID,
				negNet, TxTypeRefundReversal, revDesc, nil, "mkt-rev-"+subscriptionID)
			if err != nil {
				// Publisher has insufficient balance — still proceed with refund.
				s.log.Warn("marketplace: refund reversal failed (insufficient publisher balance)",
					zap.String("subID", subscriptionID), zap.Error(err))
			}
		}
	}

	// 7. Deactivate subscription.
	_, err = tx.Exec(ctx,
		`UPDATE user_subscriptions SET active = false WHERE id = $1`,
		sid,
	)
	if err != nil {
		return nil, fmt.Errorf("marketplace: deactivate subscription: %w", err)
	}

	// 8. Decrement subscriber counter (floor at 0).
	_, err = tx.Exec(ctx,
		`UPDATE marketplace_strategies SET total_subscribers = GREATEST(total_subscribers - 1, 0)
		 WHERE strategy_id = $1`,
		subStrategyID,
	)
	if err != nil {
		return nil, fmt.Errorf("marketplace: decrement subscribers: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("marketplace: commit refund: %w", err)
	}
	publishedCacheClear()

	return &RefundResult{
		SubscriptionID: subscriptionID,
		RefundTxID:     refundTxID.String(),
		AmountRefunded: absAmount,
		BalanceAfter:   buyerBalAfter,
	}, nil
}
