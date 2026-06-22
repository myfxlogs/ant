package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ── Refund ─────────────────────────────────────────────────────────────────────

// RefundResult holds the outcome of a refund operation.
type RefundResult struct {
	SubscriptionID  string
	RefundTxID      string
	AmountRefunded  string
	BalanceAfter    string
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
	if subKind != "purchase" {
		return nil, fmt.Errorf("marketplace: only purchased subscriptions can be refunded")
	}

	// 2. Find the original purchase transaction (buyer's debit).
	var purchaseAmount string
	var purchaseTxID string
	err = tx.QueryRow(ctx,
		`SELECT amount::text, id::text FROM wallet_transactions
		 WHERE user_id = $1 AND tx_type = 'purchase'
		 ORDER BY created_at DESC LIMIT 1`,
		uid,
	).Scan(&purchaseAmount, &purchaseTxID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: original purchase transaction not found")
	}

	// 3. Refund buyer wallet.
	var buyerWalletID uuid.UUID
	var buyerBalBefore, buyerBalAfter string
	err = tx.QueryRow(ctx,
		`SELECT id, balance::text FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
		uid,
	).Scan(&buyerWalletID, &buyerBalBefore)
	if err != nil {
		return nil, fmt.Errorf("marketplace: buyer wallet not found")
	}

	// amount is negative in the transaction; use absolute value for credit.
	absAmount := purchaseAmount
	if len(absAmount) > 0 && absAmount[0] == '-' {
		absAmount = absAmount[1:]
	}

	err = tx.QueryRow(ctx,
		`UPDATE user_wallets SET balance = balance + $1::numeric, updated_at = now()
		 WHERE user_id = $2 RETURNING balance::text`,
		absAmount, uid,
	).Scan(&buyerBalAfter)
	if err != nil {
		return nil, fmt.Errorf("marketplace: refund buyer: %w", err)
	}

	// 4. Record refund transaction for buyer.
	refundTxID := uuid.New()
	refundDesc := fmt.Sprintf("Refund for subscription %s", subscriptionID)
	err = tx.QueryRow(ctx,
		`INSERT INTO wallet_transactions (id, wallet_id, user_id, tx_type, amount, balance_before, balance_after, description)
		 VALUES ($1, $2, $3, 'refund', $4::numeric, $5::numeric, $6::numeric, $7) RETURNING id`,
		refundTxID, buyerWalletID, uid, absAmount, buyerBalBefore, buyerBalAfter, refundDesc,
	).Scan(&refundTxID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: record refund: %w", err)
	}

	// 5. Debit publisher (reverse the sale).
	pubUUID, _ := uuid.Parse(subTargetUserID)
	var pubWalletID uuid.UUID
	var pubBalBefore, pubBalAfter string
	err = tx.QueryRow(ctx,
		`SELECT id, balance::text FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
		pubUUID,
	).Scan(&pubWalletID, &pubBalBefore)
	if err != nil {
		// Publisher may no longer have a wallet — skip debit but log.
		pubBalBefore = "0"
	} else {
		err = tx.QueryRow(ctx,
			`UPDATE user_wallets SET balance = balance - $1::numeric, updated_at = now()
			 WHERE user_id = $2 AND balance >= $1::numeric
			 RETURNING balance::text`,
			absAmount, pubUUID,
		).Scan(&pubBalAfter)
		if err != nil {
			// Publisher has insufficient balance — still proceed with refund.
			pubBalAfter = pubBalBefore
		}

		// Record reversal transaction for publisher.
		revTxID := uuid.New()
		revDesc := fmt.Sprintf("Refund reversal for subscription %s", subscriptionID)
		negAbs := "-" + absAmount
		_, _ = tx.Exec(ctx,
			`INSERT INTO wallet_transactions (id, wallet_id, user_id, tx_type, amount, balance_before, balance_after, description)
			 VALUES ($1, $2, $3, 'refund_reversal', $4::numeric, $5::numeric, $6::numeric, $7)`,
			revTxID, pubWalletID, pubUUID, negAbs, pubBalBefore, pubBalAfter, revDesc,
		)
	}

	// 6. Deactivate subscription.
	_, err = tx.Exec(ctx,
		`UPDATE user_subscriptions SET active = false WHERE id = $1`,
		sid,
	)
	if err != nil {
		return nil, fmt.Errorf("marketplace: deactivate subscription: %w", err)
	}

	// 7. Decrement subscriber counter (floor at 0).
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

	return &RefundResult{
		SubscriptionID: subscriptionID,
		RefundTxID:     refundTxID.String(),
		AmountRefunded: absAmount,
		BalanceAfter:   buyerBalAfter,
	}, nil
}
