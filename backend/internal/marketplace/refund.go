package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
// debits the publisher and platform fee, deactivates the subscription, and
// decrements the subscriber counter. All steps run in a single DB transaction.
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
	defer func() { _ = tx.Rollback(ctx) }()

	result, err := s.refundPurchaseTx(ctx, tx, uid, sid)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("marketplace: commit refund: %w", err)
	}
	s.pubCache.clear()

	return result, nil
}

func (s *Service) refundPurchaseTx(ctx context.Context, tx pgx.Tx, uid, sid uuid.UUID) (*RefundResult, error) {
	// 1. Look up subscription — must be an active purchase belonging to this user.
	var subTargetUserID, subStrategyID, subKind, idemKey string
	var subActive bool
	err := tx.QueryRow(ctx,
		`SELECT target_user_id::text, target_strategy_id::text, kind, active, idempotency_key
		 FROM user_subscriptions WHERE id = $1 AND subscriber_user_id = $2 FOR UPDATE`,
		sid, uid,
	).Scan(&subTargetUserID, &subStrategyID, &subKind, &subActive, &idemKey)
	if err != nil {
		return nil, fmt.Errorf("marketplace: subscription not found")
	}
	if !subActive {
		return nil, fmt.Errorf("marketplace: subscription already inactive")
	}
	if subKind != SubKindPurchase {
		return nil, fmt.Errorf("marketplace: only purchased subscriptions can be refunded")
	}
	if idemKey == "" {
		return nil, fmt.Errorf("marketplace: subscription missing idempotency key")
	}

	buyKey := "mkt-buy-" + idemKey
	saleKey := "mkt-sale-" + idemKey
	feeKey := "mkt-fee-" + idemKey

	// 2. Find the original purchase transaction by its unique idem_key.
	var purchaseAmount string
	err = tx.QueryRow(ctx,
		`SELECT amount::text FROM wallet_transactions WHERE idem_key = $1`,
		buyKey,
	).Scan(&purchaseAmount)
	if err != nil {
		return nil, fmt.Errorf("marketplace: original purchase transaction not found: %w", err)
	}

	// amount is negative in the transaction; use absolute value for credit.
	absAmount := purchaseAmount
	if len(absAmount) > 0 && absAmount[0] == '-' {
		absAmount = absAmount[1:]
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

	refundDesc := fmt.Sprintf("Refund for subscription %s", sid)
	buyerWallet, err := s.walletRepo.AdjustBalanceTx(ctx, tx, buyerWalletID, uid,
		absAmount, TxTypeRefund, refundDesc, nil, "mkt-refund-"+sid.String())
	if err != nil {
		return nil, fmt.Errorf("marketplace: refund buyer: %w", err)
	}
	buyerBalAfter := buyerWallet.Balance
	refundTxID := uuid.Nil
	if buyerWallet.LastTransactionID != nil {
		refundTxID = *buyerWallet.LastTransactionID
	}

	// 4. Find publisher's original sale transaction by idem_key.
	var pubNetReceived string
	err = tx.QueryRow(ctx,
		`SELECT amount::text FROM wallet_transactions WHERE idem_key = $1`,
		saleKey,
	).Scan(&pubNetReceived)
	if err != nil {
		pubNetReceived = "0"
	}
	if len(pubNetReceived) > 0 && pubNetReceived[0] == '-' {
		pubNetReceived = pubNetReceived[1:]
	}

	// 5. Debit publisher by the net amount they actually received.
	if pubNetReceived != "0" {
		pubUUID, _ := uuid.Parse(subTargetUserID)
		var pubWalletID uuid.UUID
		err = tx.QueryRow(ctx,
			`SELECT id FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
			pubUUID,
		).Scan(&pubWalletID)
		if err == nil {
			negNet := "-" + pubNetReceived
			revDesc := fmt.Sprintf("Refund reversal for subscription %s", sid)
			_, err = s.walletRepo.AdjustBalanceTx(ctx, tx, pubWalletID, pubUUID,
				negNet, TxTypeRefundReversal, revDesc, nil, "mkt-rev-"+sid.String())
			if err != nil {
				s.log.Warn("marketplace: refund reversal failed (insufficient publisher balance)",
					zap.String("subID", sid.String()), zap.Error(err))
			}
		}
	}

	// 6. Reverse the platform fee credited to the system wallet at purchase time.
	var feeReceived string
	_ = tx.QueryRow(ctx,
		`SELECT amount::text FROM wallet_transactions WHERE idem_key = $1`,
		feeKey,
	).Scan(&feeReceived)
	if len(feeReceived) > 0 && feeReceived[0] == '-' {
		feeReceived = feeReceived[1:]
	}
	if feeReceived != "" && feeReceived != "0" {
		var sysWalletID uuid.UUID
		err = tx.QueryRow(ctx,
			`INSERT INTO user_wallets (user_id) VALUES ($1)
			 ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
			 RETURNING id`,
			SystemUserID,
		).Scan(&sysWalletID)
		if err == nil {
			negFee := "-" + feeReceived
			feeRevDesc := fmt.Sprintf("Platform fee reversal for subscription %s", sid)
			_, err = s.walletRepo.AdjustBalanceTx(ctx, tx, sysWalletID, SystemUserID,
				negFee, TxTypePlatformFee, feeRevDesc, nil, "mkt-fee-rev-"+sid.String())
			if err != nil {
				s.log.Warn("marketplace: platform fee reversal failed (insufficient system wallet)",
					zap.String("subID", sid.String()), zap.Error(err))
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

	return &RefundResult{
		SubscriptionID: sid.String(),
		RefundTxID:     refundTxID.String(),
		AmountRefunded: absAmount,
		BalanceAfter:   buyerBalAfter,
	}, nil
}
