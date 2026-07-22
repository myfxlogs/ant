package marketplace

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// ProviderEarningsResult holds the provider's earnings summary.
type ProviderEarningsResult struct {
	TotalEarnings     decimal.Decimal
	AvailableBalance  decimal.Decimal
	PendingWithdrawal decimal.Decimal
	PendingSettlement decimal.Decimal
	LifetimeWithdrawn decimal.Decimal
	TotalSales        int32
	ActiveStrategies  int32
}

// ProviderTxRow represents a row in the provider's transaction history.
type ProviderTxRow struct {
	ID            string
	TxType        string
	Amount        decimal.Decimal
	StrategyTitle string
	BuyerName     string
	CreatedAt     string
}

// GetProviderEarnings computes the provider's earnings summary.
// Triggers lazy settlement of expired frozen balances before computing earnings.
func (s *Service) GetProviderEarnings(ctx context.Context, userID string) (*ProviderEarningsResult, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid user_id: %w", err)
	}

	// Phase 5.4: Lazy settlement — settle expired frozen balances for this provider.
	if _, err := s.SettleExpired(ctx, userID); err != nil {
		s.log.Warn("provider earnings: lazy settlement failed", zap.String("userID", userID), zap.Error(err))
	}

	// Total earnings = sum of all 'settlement' type transactions (Phase 5.4).
	var totalEarnings decimal.Decimal
	err = s.pg.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount),0)
		 FROM wallet_transactions
		 WHERE user_id = $1 AND tx_type = $2`,
		uid, TxTypeSettlement,
	).Scan(&totalEarnings)
	if err != nil {
		return nil, fmt.Errorf("marketplace: provider earnings: %w", err)
	}

	// Available balance from wallet.
	var availBalance decimal.Decimal
	err = s.pg.QueryRow(ctx,
		`SELECT COALESCE(balance,0) FROM user_wallets WHERE user_id = $1`,
		uid,
	).Scan(&availBalance)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("marketplace: provider available balance: %w", err)
	}

	// Pending withdrawal = frozen amount in wallet.
	var pendingWithdrawal decimal.Decimal
	err = s.pg.QueryRow(ctx,
		`SELECT COALESCE(frozen,0) FROM user_wallets WHERE user_id = $1`,
		uid,
	).Scan(&pendingWithdrawal)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("marketplace: provider pending withdrawal: %w", err)
	}

	// Lifetime withdrawn = sum of completed withdrawal transactions.
	var lifetimeWithdrawn decimal.Decimal
	err = s.pg.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount),0)
		 FROM withdrawal_requests
		 WHERE user_id = $1 AND status = 'DONE'`,
		uid,
	).Scan(&lifetimeWithdrawn)
	if err != nil {
		return nil, fmt.Errorf("marketplace: provider lifetime withdrawn: %w", err)
	}

	// Total sales count (Phase 5.4: settlement tx_type replaces sale).
	var totalSales int32
	err = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM wallet_transactions WHERE user_id = $1 AND tx_type = $2`,
		uid, TxTypeSettlement,
	).Scan(&totalSales)
	if err != nil {
		return nil, fmt.Errorf("marketplace: provider total sales: %w", err)
	}

	// Active strategies count.
	var activeStrategies int32
	err = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM marketplace_strategies WHERE publisher_id = $1 AND status = 'published'`,
		uid,
	).Scan(&activeStrategies)
	if err != nil {
		return nil, fmt.Errorf("marketplace: provider active strategies: %w", err)
	}

	// Phase 5.4: Pending settlement balance (frozen provider_amount sum).
	var pendingSettlement decimal.Decimal
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(SUM(provider_amount),0) FROM marketplace_settlements
		 WHERE provider_id = $1 AND status = 'frozen'`,
		uid,
	).Scan(&pendingSettlement)

	return &ProviderEarningsResult{
		TotalEarnings:     totalEarnings,
		AvailableBalance:  availBalance,
		PendingWithdrawal: pendingWithdrawal,
		PendingSettlement: pendingSettlement,
		LifetimeWithdrawn: lifetimeWithdrawn,
		TotalSales:        totalSales,
		ActiveStrategies:  activeStrategies,
	}, nil
}

// ListProviderTransactions returns paginated transaction history for a provider.
func (s *Service) ListProviderTransactions(ctx context.Context, userID string, limit, offset int) ([]ProviderTxRow, int, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: invalid user_id: %w", err)
	}
	if limit <= 0 {
		limit = 20
	}

	var total int
	err = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM wallet_transactions WHERE user_id = $1 AND tx_type IN ('settlement','refund_reversal')`,
		uid,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: count provider txs: %w", err)
	}

	rows, err := s.pg.Query(ctx,
		`SELECT wt.id::text, wt.tx_type, wt.amount,
		        COALESCE(ms.title, ''),
		        COALESCE(buyer.email, buyer.nickname, ''),
		        wt.created_at::text
		 FROM wallet_transactions wt
		 LEFT JOIN user_subscriptions us`+subJoinOnClause+`
		 LEFT JOIN marketplace_strategies ms ON ms.strategy_id = us.target_strategy_id
		 LEFT JOIN users buyer ON buyer.id = us.subscriber_user_id
		 WHERE wt.user_id = $1 AND wt.tx_type IN ('settlement','refund_reversal')
		 ORDER BY wt.created_at DESC LIMIT $2 OFFSET $3`,
		uid, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: list provider txs: %w", err)
	}
	defer rows.Close()

	var result []ProviderTxRow
	for rows.Next() {
		var r ProviderTxRow
		if err := rows.Scan(&r.ID, &r.TxType, &r.Amount, &r.StrategyTitle, &r.BuyerName, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("marketplace: scan provider tx: %w", err)
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return result, total, nil
}
