package marketplace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

// AdminStrategyRow is a single row in the admin strategy list.
type AdminStrategyRow struct {
	StrategyID       string
	Title            string
	Description      string
	PublisherID      string
	PublisherName    string
	PriceModel       string
	PriceAmount      decimal.Decimal
	AssetClass       string
	Status           string
	TotalSales       int32
	TotalRevenue     decimal.Decimal
	PlatformRevenue  decimal.Decimal
	LastSaleAt       *time.Time
	IsFeatured       bool
	FeaturedPriority int32
	CreatedAt        time.Time
}

// AdminListStrategies lists all marketplace strategies with admin-level detail.
func (s *Service) AdminListStrategies(ctx context.Context, status, keyword string, limit, offset int) ([]AdminStrategyRow, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	if status != "" && status != "all" {
		conditions = append(conditions, fmt.Sprintf("ms.status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	if keyword != "" {
		conditions = append(conditions, fmt.Sprintf("(ms.title ILIKE $%d OR ms.description ILIKE $%d)", argIdx, argIdx+1))
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
		argIdx += 2
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total.
	var total int32
	countQuery := "SELECT COUNT(*) FROM marketplace_strategies ms" + whereClause
	err := s.pg.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: admin list count: %w", err)
	}

	query := `SELECT ms.strategy_id::text, COALESCE(ms.title,''),
	        COALESCE(ms.description,''), COALESCE(ms.publisher_id::text,''),
	        COALESCE(u.email, u.nickname, ms.publisher_id::text),
	        COALESCE(ms.price_model,''), COALESCE(ms.price_amount,0),
	        COALESCE(ms.asset_class,''), ms.status,
	        COALESCE(ms.total_subscribers,0)::int,
	        COALESCE(ms.total_pnl,0),
	        COALESCE(ms.total_pnl,0) * COALESCE(ms.platform_fee_rate,0),
	        NULLIF(ms.updated_at, ms.created_at),
	        COALESCE(ms.is_featured,false), COALESCE(ms.featured_priority,0)::int,
	        ms.created_at
		 FROM marketplace_strategies ms
		 LEFT JOIN users u ON u.id = ms.publisher_id`
	query += whereClause
	query += fmt.Sprintf(" ORDER BY ms.is_featured DESC, ms.featured_priority DESC, ms.created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pg.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: admin list query: %w", err)
	}
	defer rows.Close()

	var result []AdminStrategyRow
	for rows.Next() {
		var r AdminStrategyRow
		var lastSale pgtype.Timestamptz
		if err := rows.Scan(
			&r.StrategyID, &r.Title, &r.Description,
			&r.PublisherID, &r.PublisherName,
			&r.PriceModel, &r.PriceAmount, &r.AssetClass, &r.Status,
			&r.TotalSales, &r.TotalRevenue, &r.PlatformRevenue,
			&lastSale, &r.IsFeatured, &r.FeaturedPriority, &r.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("marketplace: admin list scan: %w", err)
		}
		if lastSale.Valid {
			r.LastSaleAt = &lastSale.Time
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return result, int(total), nil
}

// AdminFeatureStrategy sets or removes featured status for a strategy.
func (s *Service) AdminFeatureStrategy(ctx context.Context, strategyID string, featured bool, priority int32) error {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}

	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_strategies
		 SET is_featured = $2, featured_priority = $3, updated_at = now()
		 WHERE strategy_id = $1`,
		sid, featured, priority)
	if err != nil {
		return fmt.Errorf("marketplace: admin feature strategy: %w", err)
	}
	return nil
}
