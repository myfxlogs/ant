// Package marketplace — Phase 5.3: Tiered platform fee rates.
//
// Providers get lower platform fees as they accumulate more sales volume.
// Fee tiers are configured in the marketplace_fee_tiers table and applied
// automatically based on the provider's total sales count.
package marketplace

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// FeeTier represents a platform fee tier.
type FeeTier struct {
	ID            int32
	TierName      string
	MinSalesCount int32
	FeeRate       decimal.Decimal
	SortOrder     int32
	Enabled       bool
}

// ProviderFeeTierResult contains the fee tier plus stats for a provider.
type ProviderFeeTierResult struct {
	Tier             *FeeTier
	CurrentSales     int32
	NextTierMinSales int32
}

// GetProviderFeeTierWithStats returns the fee tier plus current sales count
// and the next tier's min_sales_count for upgrade progress display.
func (s *Service) GetProviderFeeTierWithStats(ctx context.Context, publisherID string) (*ProviderFeeTierResult, error) {
	pid, err := uuid.Parse(publisherID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: fee tier: invalid publisher_id: %w", err)
	}

	var totalSales int32
	err = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM wallet_transactions
		 WHERE user_id = $1 AND tx_type = $2`,
		pid, TxTypeSettlement,
	).Scan(&totalSales)
	if err != nil {
		return nil, fmt.Errorf("marketplace: fee tier: count sales: %w", err)
	}

	tier := s.getFeeTierForSalesCount(ctx, totalSales)

	// Find the next tier (first tier with min_sales_count > current sales).
	var nextMinSales int32
	_ = s.pg.QueryRow(ctx,
		`SELECT min_sales_count FROM marketplace_fee_tiers
		 WHERE enabled = true AND min_sales_count > $1
		 ORDER BY min_sales_count ASC LIMIT 1`,
		totalSales,
	).Scan(&nextMinSales)

	return &ProviderFeeTierResult{
		Tier:             tier,
		CurrentSales:     totalSales,
		NextTierMinSales: nextMinSales,
	}, nil
}

// getFeeTierForSalesCount finds the best matching tier for a given sales count.
func (s *Service) getFeeTierForSalesCount(ctx context.Context, salesCount int32) *FeeTier {
	var tier FeeTier
	err := s.pg.QueryRow(ctx,
		`SELECT id, tier_name, min_sales_count, fee_rate, sort_order, enabled
		 FROM marketplace_fee_tiers
		 WHERE enabled = true AND min_sales_count <= $1
		 ORDER BY min_sales_count DESC LIMIT 1`,
		salesCount,
	).Scan(&tier.ID, &tier.TierName, &tier.MinSalesCount, &tier.FeeRate, &tier.SortOrder, &tier.Enabled)
	if err != nil {
		// Fallback to default 10% if no tiers configured.
		return &FeeTier{
			TierName:      "default",
			MinSalesCount: 0,
			FeeRate:       decimal.NewFromFloat(0.10),
		}
	}
	return &tier
}

// ListFeeTiers returns all fee tiers ordered by sort_order.
func (s *Service) ListFeeTiers(ctx context.Context) ([]FeeTier, error) {
	rows, err := s.pg.Query(ctx,
		`SELECT id, tier_name, min_sales_count, fee_rate, sort_order, enabled
		 FROM marketplace_fee_tiers ORDER BY sort_order ASC`)
	if err != nil {
		return nil, fmt.Errorf("marketplace: list fee tiers: %w", err)
	}
	defer rows.Close()

	var tiers []FeeTier
	for rows.Next() {
		var t FeeTier
		if err := rows.Scan(&t.ID, &t.TierName, &t.MinSalesCount, &t.FeeRate, &t.SortOrder, &t.Enabled); err != nil {
			return nil, err
		}
		tiers = append(tiers, t)
	}
	return tiers, rows.Err()
}

// UpdateFeeTier updates an existing fee tier's rate and thresholds.
// Admin-only operation.
func (s *Service) UpdateFeeTier(ctx context.Context, tierID int32, feeRate string, minSalesCount int32, enabled bool) error {
	rate, err := decimal.NewFromString(feeRate)
	if err != nil {
		return fmt.Errorf("marketplace: update fee tier: invalid fee_rate: %w", err)
	}
	if rate.LessThan(decimal.Zero) || rate.GreaterThan(decimal.NewFromInt(1)) {
		return fmt.Errorf("marketplace: update fee tier: fee_rate must be between 0 and 1")
	}

	tag, err := s.pg.Exec(ctx,
		`UPDATE marketplace_fee_tiers
		 SET fee_rate = $2, min_sales_count = $3, enabled = $4
		 WHERE id = $1`,
		tierID, rate, minSalesCount, enabled,
	)
	if err != nil {
		return fmt.Errorf("marketplace: update fee tier: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("marketplace: update fee tier: tier not found")
	}
	return nil
}

// getEffectiveFeeRateTx is the transaction-aware version for use inside purchase flows.
// Reads fee tier and sales count within the same tx snapshot for consistency.
func (s *Service) getEffectiveFeeRateTx(ctx context.Context, tx pgx.Tx, publisherID string) (decimal.Decimal, error) {
	pid, err := uuid.Parse(publisherID)
	if err != nil {
		return decimal.Zero, fmt.Errorf("marketplace: fee tier tx: invalid publisher_id: %w", err)
	}

	var totalSales int32
	err = tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM wallet_transactions
		 WHERE user_id = $1 AND tx_type = $2`,
		pid, TxTypeSettlement,
	).Scan(&totalSales)
	if err != nil {
		return decimal.Zero, fmt.Errorf("marketplace: fee tier tx: count sales: %w", err)
	}

	var feeRate decimal.Decimal
	err = tx.QueryRow(ctx,
		`SELECT fee_rate FROM marketplace_fee_tiers
		 WHERE enabled = true AND min_sales_count <= $1
		 ORDER BY min_sales_count DESC LIMIT 1`,
		totalSales,
	).Scan(&feeRate)
	if err != nil {
		// Fallback to system_config (read via tx for snapshot isolation) then 10% default.
		var configRate string
		_ = tx.QueryRow(ctx,
			`SELECT value FROM system_config WHERE key = 'marketplace.platform_fee_rate' AND enabled = true`,
		).Scan(&configRate)
		rate, _ := decimal.NewFromString(configRate)
		if rate.IsZero() {
			rate = decimal.NewFromFloat(0.10)
		}
		return rate, nil
	}
	return feeRate, nil
}
