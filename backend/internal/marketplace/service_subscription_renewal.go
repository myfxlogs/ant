// service_subscription_renewal.go — Renewal processing extracted from service_subscription.go.
package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func (s *Service) RenewSubscriptions(ctx context.Context) (renewed, failed int, err error) {
	rows, qErr := s.pg.Query(ctx, `
		SELECT us.id, us.subscriber_user_id::text, us.target_user_id::text,
		       us.target_strategy_id::text, ms.price_amount::text, ms.title,
		       ms.platform_fee_rate::text, COALESCE(ms.refund_window_days, 7)
		FROM user_subscriptions us
		JOIN marketplace_strategies ms ON ms.strategy_id = us.target_strategy_id
		WHERE us.kind = $1 AND us.active = true
		  AND us.expires_at IS NOT NULL AND us.expires_at <= now()
		LIMIT 100`, SubKindSubscription)
	if qErr != nil {
		return 0, 0, qErr
	}
	defer rows.Close()

	var renewals []renewalItem
	for rows.Next() {
		var r renewalItem
		if err := rows.Scan(&r.subID, &r.userID, &r.publisherID, &r.strategyID, &r.priceAmount, &r.title, &r.platformFeeRate, &r.refundWindowDays); err != nil {
			continue
		}
		renewals = append(renewals, r)
	}
	rows.Close()

	for _, p := range renewals {
		r, f := s.processRenewal(ctx, p)
		renewed += r
		failed += f
	}

	return renewed, failed, nil
}

func (s *Service) processRenewal(ctx context.Context, p renewalItem) (renewed, failed int) {
	tx, txErr := s.pg.Begin(ctx)
	if txErr != nil {
		return 0, 1
	}

	priceDec, decErr := decimal.NewFromString(p.priceAmount)
	if decErr != nil {
		_ = tx.Rollback(ctx)
		return 0, 1
	}
	feeRateDec, err := s.getEffectiveFeeRateTx(ctx, tx, p.publisherID)
	if err != nil {
		s.log.Warn("renewal: get effective fee rate failed", zap.String("subID", p.subID), zap.Error(err))
		_ = tx.Rollback(ctx)
		return 0, 1
	}
	feeDec := priceDec.Mul(feeRateDec)
	pubDec := priceDec.Sub(feeDec)

	amountStr := priceDec.StringFixed(2)
	negAmountStr := "-" + amountStr

	uid, err := uuid.Parse(p.userID)
	if err != nil {
		s.log.Warn("renewal: invalid user_id", zap.String("subID", p.subID), zap.String("userID", p.userID), zap.Error(err))
		_ = tx.Rollback(ctx)
		return 0, 1
	}
	var buyerWalletID uuid.UUID
	if err := tx.QueryRow(ctx,
		`SELECT id FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
		uid,
	).Scan(&buyerWalletID); err != nil {
		_ = tx.Rollback(ctx)
		return 0, 1
	}

	buyerDesc := fmt.Sprintf("Subscription renewal: %s", p.title)
	_, chargeErr := s.walletRepo.AdjustBalanceTx(ctx, tx, buyerWalletID, uid,
		negAmountStr, TxTypePurchase, buyerDesc, nil, IdemKeyRenewBuy+p.subID)

	if chargeErr != nil {
		if _, dErr := tx.Exec(ctx, `UPDATE user_subscriptions SET active = false WHERE id = $1`, p.subID); dErr != nil {
			s.log.Warn("renewal: deactivate failed", zap.String("subID", p.subID), zap.Error(dErr))
			_ = tx.Rollback(ctx)
			return 0, 1
		}
		if cErr := tx.Commit(ctx); cErr != nil {
			s.log.Warn("renewal: deactivate commit failed", zap.String("subID", p.subID), zap.Error(cErr))
		}
		return 0, 1
	}

	pubID, err := uuid.Parse(p.publisherID)
	if err != nil {
		s.log.Warn("renewal: invalid publisher_id", zap.String("subID", p.subID), zap.String("publisherID", p.publisherID), zap.Error(err))
		_ = tx.Rollback(ctx)
		return 0, 1
	}
	pubAmountStr := pubDec.StringFixed(2)
	feeStr := feeDec.StringFixed(2)
	subUUID, err := uuid.Parse(p.subID)
	if err != nil {
		s.log.Warn("renewal: invalid sub_id", zap.String("subID", p.subID), zap.Error(err))
		_ = tx.Rollback(ctx)
		return 0, 1
	}
	if err := s.createFrozenSettlementTx(ctx, tx, subUUID, uid, pubID, amountStr, feeStr, pubAmountStr, p.refundWindowDays, nil); err != nil {
		s.log.Warn("renewal: create settlement failed", zap.String("subID", p.subID), zap.Error(err))
		_ = tx.Rollback(ctx)
		return 0, 1
	}

	if _, eErr := tx.Exec(ctx,
		`UPDATE user_subscriptions SET expires_at = now() + INTERVAL '30 days' WHERE id = $1`,
		p.subID,
	); eErr != nil {
		s.log.Warn("renewal: extend failed", zap.String("subID", p.subID), zap.Error(eErr))
		_ = tx.Rollback(ctx)
		return 0, 1
	}

	if cErr := tx.Commit(ctx); cErr != nil {
		s.log.Warn("renewal: commit failed", zap.String("subID", p.subID), zap.Error(cErr))
		return 0, 1
	}
	return 1, 0
}

// StartRenewalLoop runs a daily subscription renewal ticker in a background
// goroutine. Call during server startup.
