package marketplace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/model"
)

// SettlementResult holds the outcome of a lazy settlement batch.
type SettlementResult struct {
	SettledCount int
	TotalAmount  string
}

// SettleExpired settles all frozen settlements for a provider whose refund window
// has expired. Credits the provider wallet and platform wallet, marks settlements
// as 'settled'. Called lazily — triggered by provider dashboard views, earnings
// queries, or new purchases. No timers, no polling.
func (s *Service) SettleExpired(ctx context.Context, providerID string) (*SettlementResult, error) {
	pid, err := uuid.Parse(providerID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: settle: invalid provider_id: %w", err)
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("marketplace: settle: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Select all frozen settlements past their settles_at for this provider.
	rows, err := tx.Query(ctx,
		`SELECT id, provider_amount, platform_fee
		 FROM marketplace_settlements
		 WHERE provider_id = $1 AND status = 'frozen' AND settles_at <= $2
		 FOR UPDATE SKIP LOCKED`,
		pid, time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("marketplace: settle: query frozen: %w", err)
	}

	type pending struct {
		id          uuid.UUID
		providerAmt string
		platformFee string
	}
	var batch []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.providerAmt, &p.platformFee); err != nil {
			rows.Close()
			return nil, fmt.Errorf("marketplace: settle: scan: %w", err)
		}
		batch = append(batch, p)
	}
	rows.Close()

	if len(batch) == 0 {
		_ = tx.Commit(ctx)
		return &SettlementResult{SettledCount: 0, TotalAmount: "0.00"}, nil
	}

	// 2. Ensure provider + system wallets exist and lock them.
	pubWalletID, sysWalletID, err := s.ensureSettlementWallets(ctx, tx, pid)
	if err != nil {
		return nil, err
	}

	// 4. Credit each settlement individually for idempotency + hash chain integrity.
	settledCount := 0
	totalSettled := decimal.Zero
	for _, p := range batch {
		settleID := p.id.String()

		// Credit provider.
		_, err := s.walletRepo.AdjustBalanceTx(ctx, tx, pubWalletID, pid,
			p.providerAmt, TxTypeSettlement,
			fmt.Sprintf("Settlement for purchase %s", settleID),
			nil, IdemKeySettle+settleID)
		if err != nil && !errors.Is(err, model.ErrIdempotentReplay) {
			s.log.Warn("settle: credit provider failed, skipping",
				zap.String("settlementID", settleID), zap.Error(err))
			continue
		}

		// Credit platform fee.
		if p.platformFee != "0.00" && p.platformFee != "" {
			_, err = s.walletRepo.AdjustBalanceTx(ctx, tx, sysWalletID, SystemUserID,
				p.platformFee, TxTypeFeeSettlement,
				fmt.Sprintf("Platform fee settlement for purchase %s", settleID),
				nil, IdemKeyFeeSettle+settleID)
			if err != nil && !errors.Is(err, model.ErrIdempotentReplay) {
				s.log.Warn("settle: credit platform fee failed, skipping",
					zap.String("settlementID", settleID), zap.Error(err))
				continue
			}
		}

		// Mark settlement as settled.
		_, err = tx.Exec(ctx,
			`UPDATE marketplace_settlements SET status = 'settled', settled_at = $2 WHERE id = $1 AND status = 'frozen'`,
			p.id, time.Now(),
		)
		if err != nil {
			s.log.Warn("settle: mark settled failed",
				zap.String("settlementID", settleID), zap.Error(err))
			continue
		}

		providerAmt, _ := decimal.NewFromString(p.providerAmt)
		totalSettled = totalSettled.Add(providerAmt)
		settledCount++
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("marketplace: settle: commit: %w", err)
	}

	// R3: Lazily retry failed refund reversals for this provider.
	// This runs in a separate transaction after settlement completes,
	// so a retry failure doesn't roll back the settlement.
	if retried, rerr := s.retryFailedReversals(ctx, providerID); rerr != nil {
		s.log.Warn("settle: retry failed reversals error",
			zap.String("providerID", providerID), zap.Error(rerr))
	} else if retried > 0 {
		s.log.Info("settle: retried failed reversals",
			zap.String("providerID", providerID), zap.Int("retried", retried))
	}

	return &SettlementResult{
		SettledCount: settledCount,
		TotalAmount:  totalSettled.StringFixed(2),
	}, nil
}

func (s *Service) ensureSettlementWallets(ctx context.Context, tx pgx.Tx, pid uuid.UUID) (pubWalletID, sysWalletID uuid.UUID, err error) {
	err = tx.QueryRow(ctx,
		`INSERT INTO user_wallets (user_id) VALUES ($1)
		 ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
		 RETURNING id`,
		pid,
	).Scan(&pubWalletID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("marketplace: settle: provider wallet: %w", err)
	}
	err = tx.QueryRow(ctx,
		`INSERT INTO user_wallets (user_id) VALUES ($1)
		 ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
		 RETURNING id`,
		SystemUserID,
	).Scan(&sysWalletID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("marketplace: settle: system wallet: %w", err)
	}
	return pubWalletID, sysWalletID, nil
}

// retryFailedReversals retries debit operations on settlements marked with
// reversal_failed = true for the given provider. Called lazily from SettleExpired.
// R3: Replaces the previous "record but never retry" behavior.
func (s *Service) retryFailedReversals(ctx context.Context, providerID string) (int, error) {
	pid, err := uuid.Parse(providerID)
	if err != nil {
		return 0, fmt.Errorf("marketplace: retry reversals: invalid provider_id: %w", err)
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("marketplace: retry reversals: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Select settlements with failed reversals for this provider.
	// purchase_id is needed to construct the original idem keys for correct idempotency.
	rows, err := tx.Query(ctx,
		`SELECT id, purchase_id, provider_amount::text, platform_fee::text, reversal_failure_note
		 FROM marketplace_settlements
		 WHERE provider_id = $1 AND reversal_failed = true AND status = 'refunded'
		 FOR UPDATE SKIP LOCKED`,
		pid,
	)
	if err != nil {
		return 0, fmt.Errorf("marketplace: retry reversals: query: %w", err)
	}

	type failed struct {
		id          uuid.UUID
		purchaseID  uuid.UUID
		providerAmt string
		platformFee string
		note        string
	}
	var batch []failed
	for rows.Next() {
		var f failed
		if err := rows.Scan(&f.id, &f.purchaseID, &f.providerAmt, &f.platformFee, &f.note); err != nil {
			rows.Close()
			return 0, fmt.Errorf("marketplace: retry reversals: scan: %w", err)
		}
		batch = append(batch, f)
	}
	rows.Close()

	if len(batch) == 0 {
		_ = tx.Commit(ctx)
		return 0, nil
	}

	// Lock provider wallet.
	var pubWalletID uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT id FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
		pid,
	).Scan(&pubWalletID)
	if err != nil {
		return 0, fmt.Errorf("marketplace: retry reversals: provider wallet: %w", err)
	}

	retried := 0
	for _, f := range batch {
		settleID := f.id.String()

		// Retry publisher debit using the ORIGINAL idem key so that
		// already-debited reversals are rejected as idempotent replays.
		purchaseIDStr := f.purchaseID.String()
		negPub := "-" + f.providerAmt
		_, err := s.walletRepo.AdjustBalanceTx(ctx, tx, pubWalletID, pid,
			negPub, TxTypeRefundReversal,
			fmt.Sprintf("Refund reversal retry for settlement %s", settleID),
			nil, IdemKeyRev+purchaseIDStr)
		if err != nil {
			s.log.Warn("retry reversals: publisher debit still failing",
				zap.String("settlementID", settleID), zap.Error(err))
			continue
		}

		// Retry platform fee debit.
		if f.platformFee != "0.00" && f.platformFee != "" {
			var sysWalletID uuid.UUID
			err = tx.QueryRow(ctx,
				`INSERT INTO user_wallets (user_id) VALUES ($1)
				 ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
				 RETURNING id`,
				SystemUserID,
			).Scan(&sysWalletID)
			if err == nil {
				negFee := "-" + f.platformFee
				_, err = s.walletRepo.AdjustBalanceTx(ctx, tx, sysWalletID, SystemUserID,
					negFee, TxTypeRefundReversal,
					fmt.Sprintf("Platform fee reversal retry for settlement %s", settleID),
					nil, IdemKeyFeeRev+purchaseIDStr)
				if err != nil {
					s.log.Warn("retry reversals: platform fee debit still failing",
						zap.String("settlementID", settleID), zap.Error(err))
					// Publisher debit succeeded, but fee failed — partial retry.
					// Keep reversal_failed = true with updated note.
					_, _ = tx.Exec(ctx,
						`UPDATE marketplace_settlements
						 SET reversal_failure_note = $2 WHERE id = $1`,
						f.id, fmt.Sprintf("platform fee retry failed: %s; ", err.Error()))
					continue
				}
			}
		}

		// Both debits succeeded — clear the failure flag.
		_, err = tx.Exec(ctx,
			`UPDATE marketplace_settlements
			 SET reversal_failed = false, reversal_failure_note = NULL
			 WHERE id = $1`,
			f.id,
		)
		if err != nil {
			s.log.Warn("retry reversals: clear flag failed",
				zap.String("settlementID", settleID), zap.Error(err))
			continue
		}
		retried++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("marketplace: retry reversals: commit: %w", err)
	}
	return retried, nil
}

// createFrozenSettlementTx inserts a frozen settlement row within an existing
// transaction. Shared by PurchaseStrategy, PurchaseBundle, and subscription renewal.
// M6: bundleID is set for bundle purchases so refund logic can find the settlement
// regardless of which subscription in the bundle is being refunded.
func (s *Service) createFrozenSettlementTx(ctx context.Context, tx pgx.Tx,
	purchaseID, buyerID, providerID uuid.UUID,
	amount, platformFee, providerAmount string,
	refundWindowDays int,
	bundleID *uuid.UUID,
) error {
	if refundWindowDays <= 0 {
		refundWindowDays = DefaultRefundWindowDays
	}
	now := time.Now()
	settlesAt := now.Add(time.Duration(refundWindowDays) * 24 * time.Hour)
	_, err := tx.Exec(ctx,
		`INSERT INTO marketplace_settlements
		 (purchase_id, buyer_id, provider_id, amount, platform_fee, provider_amount,
		  status, refund_window_days, freezes_at, settles_at, bundle_id)
		 VALUES ($1, $2, $3, $4, $5, $6, 'frozen', $7, $8, $9, $10)`,
		purchaseID, buyerID, providerID, amount, platformFee, providerAmount,
		refundWindowDays, now, settlesAt, bundleID,
	)
	return err
}
