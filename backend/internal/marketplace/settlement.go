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

	// 2. Ensure provider wallet exists and lock it.
	var pubWalletID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO user_wallets (user_id) VALUES ($1)
		 ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
		 RETURNING id`,
		pid,
	).Scan(&pubWalletID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: settle: provider wallet: %w", err)
	}

	// 3. Ensure system wallet exists.
	var sysWalletID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO user_wallets (user_id) VALUES ($1)
		 ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
		 RETURNING id`,
		SystemUserID,
	).Scan(&sysWalletID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: settle: system wallet: %w", err)
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

	return &SettlementResult{
		SettledCount: settledCount,
		TotalAmount:  totalSettled.StringFixed(2),
	}, nil
}

// createFrozenSettlementTx inserts a frozen settlement row within an existing
// transaction. Shared by PurchaseStrategy, PurchaseBundle, and subscription renewal.
func (s *Service) createFrozenSettlementTx(ctx context.Context, tx pgx.Tx,
	purchaseID, buyerID, providerID uuid.UUID,
	amount, platformFee, providerAmount string,
	refundWindowDays int,
) error {
	if refundWindowDays <= 0 {
		refundWindowDays = DefaultRefundWindowDays
	}
	now := time.Now()
	settlesAt := now.Add(time.Duration(refundWindowDays) * 24 * time.Hour)
	_, err := tx.Exec(ctx,
		`INSERT INTO marketplace_settlements
		 (purchase_id, buyer_id, provider_id, amount, platform_fee, provider_amount,
		  status, refund_window_days, freezes_at, settles_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 'frozen', $7, $8, $9)`,
		purchaseID, buyerID, providerID, amount, platformFee, providerAmount,
		refundWindowDays, now, settlesAt,
	)
	return err
}
