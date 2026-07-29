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

// ── Refund ─────────────────────────────────────────────────────────────────────

// RefundResult holds the outcome of a refund operation.
type RefundResult struct {
	SubscriptionID string
	RefundTxID     string
	AmountRefunded string
	BalanceAfter   string
}

// RefundPurchase reverses a paid strategy purchase: credits the buyer back,
// handles the settlement (frozen→mark refunded, settled→reverse publisher+platform),
// deactivates the subscription, and decrements the subscriber counter.
// All steps run in a single DB transaction.
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
	subTargetUserID, subStrategyID, subBundleID, err := s.validateRefundSubscription(ctx, tx, uid, sid)
	if err != nil {
		return nil, err
	}

	buyKey := IdemKeyBuy + subTargetUserID
	_ = buyKey // original used idemKey from subscription

	// 2. Find the original purchase transaction by its unique idem_key.
	var purchaseAmount string
	var idemKey string
	err = tx.QueryRow(ctx,
		`SELECT amount::text, idempotency_key FROM wallet_transactions WHERE idem_key = $1`,
		IdemKeyBuy+subTargetUserID,
	).Scan(&purchaseAmount, &idemKey)
	_ = idemKey
	if err != nil {
		return nil, fmt.Errorf("marketplace: original purchase transaction not found: %w", err)
	}

	purchaseDec, err := decimal.NewFromString(purchaseAmount)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid purchase amount: %w", err)
	}
	absAmount := purchaseDec.Abs().StringFixed(2)

	// 3. Refund buyer wallet.
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
		absAmount, TxTypeRefund, refundDesc, nil, IdemKeyRefund+sid.String())
	if err != nil {
		return nil, fmt.Errorf("marketplace: refund buyer: %w", err)
	}
	buyerBalAfter := buyerWallet.Balance
	refundTxID := uuid.Nil
	if buyerWallet.LastTransactionID != nil {
		refundTxID = *buyerWallet.LastTransactionID
	}

	// 4. Handle settlement reversal.
	if err := s.reverseSettlement(ctx, tx, sid, uid, subTargetUserID, subBundleID); err != nil {
		return nil, err
	}

	// 5. Deactivate subscription and decrement counter.
	if err := s.deactivateRefundSubscription(ctx, tx, sid, subStrategyID); err != nil {
		return nil, err
	}

	return &RefundResult{
		SubscriptionID: sid.String(),
		RefundTxID:     refundTxID.String(),
		AmountRefunded: absAmount,
		BalanceAfter:   buyerBalAfter,
	}, nil
}

func (s *Service) validateRefundSubscription(ctx context.Context, tx pgx.Tx, uid, sid uuid.UUID) (subTargetUserID, subStrategyID string, subBundleID *uuid.UUID, err error) {
	var subKind, idemKey string
	var subActive bool
	err = tx.QueryRow(ctx,
		`SELECT target_user_id::text, target_strategy_id::text, kind, active, idempotency_key, bundle_id
		 FROM user_subscriptions WHERE id = $1 AND subscriber_user_id = $2 FOR UPDATE`,
		sid, uid,
	).Scan(&subTargetUserID, &subStrategyID, &subKind, &subActive, &idemKey, &subBundleID)
	if err != nil {
		return "", "", nil, fmt.Errorf("marketplace: subscription not found")
	}
	if !subActive {
		return "", "", nil, fmt.Errorf("marketplace: subscription already inactive")
	}
	if subKind != SubKindPurchase {
		return "", "", nil, fmt.Errorf("marketplace: only purchased subscriptions can be refunded")
	}
	if idemKey == "" {
		return "", "", nil, fmt.Errorf("marketplace: subscription missing idempotency key")
	}

	var activeSchedules int
	err = tx.QueryRow(ctx,
		`SELECT count(*) FROM strategy_schedules
		 WHERE template_id = $1 AND user_id = $2 AND is_active = true`,
		subStrategyID, uid,
	).Scan(&activeSchedules)
	if err != nil {
		return "", "", nil, fmt.Errorf("marketplace: check active schedules: %w", err)
	}
	if activeSchedules > 0 {
		return "", "", nil, fmt.Errorf("marketplace: strategy has active live schedules")
	}
	return subTargetUserID, subStrategyID, subBundleID, nil
}

func (s *Service) reverseSettlement(ctx context.Context, tx pgx.Tx, sid, uid uuid.UUID, subTargetUserID string, subBundleID *uuid.UUID) error {
	var settlementStatus, settlementID, providerAmount, platformFee string
	err := tx.QueryRow(ctx,
		`SELECT status, id::text, provider_amount::text, platform_fee::text
		 FROM marketplace_settlements WHERE purchase_id = $1 FOR UPDATE`,
		sid,
	).Scan(&settlementStatus, &settlementID, &providerAmount, &platformFee)
	if err != nil && subBundleID != nil {
		err = tx.QueryRow(ctx,
			`SELECT status, id::text, provider_amount::text, platform_fee::text
			 FROM marketplace_settlements
			 WHERE bundle_id = $1 AND buyer_id = $2
			 ORDER BY created_at DESC LIMIT 1 FOR UPDATE`,
			*subBundleID, uid,
		).Scan(&settlementStatus, &settlementID, &providerAmount, &platformFee)
	}
	if err != nil {
		s.log.Warn("marketplace: settlement not found for subscription, skipping settlement reversal",
			zap.String("subID", sid.String()), zap.Error(err))
		return nil
	}
	if settlementID == "" {
		return nil
	}

	switch settlementStatus {
	case SettlementStatusFrozen:
		_, err = tx.Exec(ctx,
			`UPDATE marketplace_settlements SET status = 'refunded', refunded_at = $2 WHERE id = $1`,
			settlementID, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("marketplace: mark settlement refunded: %w", err)
		}

	case SettlementStatusSettled:
		var reversalFailed bool
		var reversalNote string

		pubUUID, perr := uuid.Parse(subTargetUserID)
		if perr != nil {
			return fmt.Errorf("marketplace: invalid publisher_id in subscription: %w", perr)
		}
		var pubWalletID uuid.UUID
		err = tx.QueryRow(ctx,
			`SELECT id FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
			pubUUID,
		).Scan(&pubWalletID)
		if err == nil {
			negPub := "-" + providerAmount
			revDesc := fmt.Sprintf("Refund reversal for subscription %s", sid)
			_, err = s.walletRepo.AdjustBalanceTx(ctx, tx, pubWalletID, pubUUID,
				negPub, TxTypeRefundReversal, revDesc, nil, IdemKeyRev+sid.String())
			if err != nil {
				reversalFailed = true
				reversalNote += fmt.Sprintf("publisher reversal failed: %s; ", err.Error())
				s.log.Warn("marketplace: refund reversal failed (insufficient publisher balance)",
					zap.String("subID", sid.String()), zap.Error(err))
			}
		}

		if platformFee != "0.00" && platformFee != "" {
			var sysWalletID uuid.UUID
			err = tx.QueryRow(ctx,
				`INSERT INTO user_wallets (user_id) VALUES ($1)
				 ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
				 RETURNING id`,
				SystemUserID,
			).Scan(&sysWalletID)
			if err == nil {
				negFee := "-" + platformFee
				feeRevDesc := fmt.Sprintf("Platform fee reversal for subscription %s", sid)
				_, err = s.walletRepo.AdjustBalanceTx(ctx, tx, sysWalletID, SystemUserID,
					negFee, TxTypeRefundReversal, feeRevDesc, nil, IdemKeyFeeRev+sid.String())
				if err != nil {
					reversalFailed = true
					reversalNote += fmt.Sprintf("platform fee reversal failed: %s; ", err.Error())
					s.log.Warn("marketplace: platform fee reversal failed",
						zap.String("subID", sid.String()), zap.Error(err))
				}
			}
		}

		if reversalFailed {
			_, err = tx.Exec(ctx,
				`UPDATE marketplace_settlements SET status = 'refunded', refunded_at = $2,
				 reversal_failed = true, reversal_failure_note = $3 WHERE id = $1`,
				settlementID, time.Now(), reversalNote,
			)
		} else {
			_, err = tx.Exec(ctx,
				`UPDATE marketplace_settlements SET status = 'refunded', refunded_at = $2 WHERE id = $1`,
				settlementID, time.Now(),
			)
		}
		if err != nil {
			return fmt.Errorf("marketplace: mark settled refund: %w", err)
		}

	case SettlementStatusRefunded:
		return fmt.Errorf("marketplace: settlement already refunded")
	}
	return nil
}

func (s *Service) deactivateRefundSubscription(ctx context.Context, tx pgx.Tx, sid uuid.UUID, subStrategyID string) error {
	_, err := tx.Exec(ctx,
		`UPDATE user_subscriptions SET active = false WHERE id = $1`,
		sid,
	)
	if err != nil {
		return fmt.Errorf("marketplace: deactivate subscription: %w", err)
	}
	_, err = tx.Exec(ctx,
		`UPDATE marketplace_strategies SET total_subscribers = GREATEST(total_subscribers - 1, 0)
		 WHERE strategy_id = $1`,
		subStrategyID,
	)
	if err != nil {
		return fmt.Errorf("marketplace: decrement subscribers: %w", err)
	}
	return nil
}
