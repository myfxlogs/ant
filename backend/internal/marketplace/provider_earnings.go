package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ProviderEarningsResult holds the provider's earnings summary.
type ProviderEarningsResult struct {
	TotalEarnings     decimal.Decimal
	AvailableBalance  decimal.Decimal
	PendingWithdrawal decimal.Decimal
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
	Status        string
}

// GetProviderEarnings computes the provider's earnings summary.
func (s *Service) GetProviderEarnings(ctx context.Context, userID string) (*ProviderEarningsResult, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid user_id: %w", err)
	}

	// Total earnings = sum of all 'sale' type transactions.
	var totalEarnings decimal.Decimal
	err = s.pg.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount),0)
		 FROM wallet_transactions
		 WHERE user_id = $1 AND tx_type = $2`,
		uid, TxTypeSale,
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
	if err != nil {
		availBalance = decimal.Zero
	}

	// Pending withdrawal = frozen amount in wallet.
	var pendingWithdrawal decimal.Decimal
	err = s.pg.QueryRow(ctx,
		`SELECT COALESCE(frozen,0) FROM user_wallets WHERE user_id = $1`,
		uid,
	).Scan(&pendingWithdrawal)
	if err != nil {
		pendingWithdrawal = decimal.Zero
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
		lifetimeWithdrawn = decimal.Zero
	}

	// Total sales count.
	var totalSales int32
	err = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM wallet_transactions WHERE user_id = $1 AND tx_type = $2`,
		uid, TxTypeSale,
	).Scan(&totalSales)
	if err != nil {
		totalSales = 0
	}

	// Active strategies count.
	var activeStrategies int32
	err = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM marketplace_strategies WHERE publisher_id = $1 AND status = 'published'`,
		uid,
	).Scan(&activeStrategies)
	if err != nil {
		activeStrategies = 0
	}

	return &ProviderEarningsResult{
		TotalEarnings:     totalEarnings,
		AvailableBalance:  availBalance,
		PendingWithdrawal: pendingWithdrawal,
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
		`SELECT COUNT(*) FROM wallet_transactions WHERE user_id = $1 AND tx_type IN ('sale','refund_reversal')`,
		uid,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: count provider txs: %w", err)
	}

	rows, err := s.pg.Query(ctx,
		`SELECT wt.id::text, wt.tx_type, wt.amount,
		        COALESCE(ms.title, ''),
		        COALESCE(u.email, u.nickname, ''),
		        wt.created_at::text,
		        COALESCE(wr.status, '')
		 FROM wallet_transactions wt
		 LEFT JOIN user_subscriptions us ON us.subscriber_user_id = wt.user_id
		 LEFT JOIN marketplace_strategies ms ON ms.strategy_id = us.target_strategy_id
		 LEFT JOIN users u ON u.id = wt.user_id
		 LEFT JOIN withdrawal_requests wr ON wr.user_id = wt.user_id
		 WHERE wt.user_id = $1 AND wt.tx_type IN ('sale','refund_reversal')
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
		if err := rows.Scan(&r.ID, &r.TxType, &r.Amount, &r.StrategyTitle, &r.BuyerName, &r.CreatedAt, &r.Status); err != nil {
			return nil, 0, fmt.Errorf("marketplace: scan provider tx: %w", err)
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return result, total, nil
}
