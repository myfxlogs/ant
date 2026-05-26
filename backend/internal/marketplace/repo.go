// Package marketplace — strategy marketplace API (M5-1).
// Provides CRUD for marketplace_strategies table.
package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Strategy is a published marketplace entry.
type Strategy struct {
	ID               uuid.UUID  `db:"id" json:"id"`
	StrategyID       uuid.UUID  `db:"strategy_id" json:"strategy_id"`
	PublisherID      uuid.UUID  `db:"publisher_id" json:"publisher_id"`
	Title            string     `db:"title" json:"title"`
	Description      string     `db:"description" json:"description"`
	PriceModel       string     `db:"price_model" json:"price_model"`
	PriceAmount      *float64   `db:"price_amount" json:"price_amount,omitempty"`
	AssetClass       string     `db:"asset_class" json:"asset_class"`
	Symbols          []string   `db:"symbols" json:"symbols"`
	Timeframe        *string    `db:"timeframe" json:"timeframe,omitempty"`
	RiskLevel        string     `db:"risk_level" json:"risk_level"`
	Tags             []string   `db:"tags" json:"tags,omitempty"`
	TotalSubscribers int        `db:"total_subscribers" json:"total_subscribers"`
	TotalPnL         *float64   `db:"total_pnl" json:"total_pnl,omitempty"`
	WinRate          *float64   `db:"win_rate" json:"win_rate,omitempty"`
	Status           string     `db:"status" json:"status"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
}

// Repo handles marketplace_strategies persistence.
type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

// Publish creates a new marketplace listing.
func (r *Repo) Publish(ctx context.Context, s *Strategy) error {
	s.ID = uuid.New()
	s.CreatedAt = time.Now()
	s.UpdatedAt = s.CreatedAt
	_, err := r.db.Exec(ctx, `
			INSERT INTO marketplace_strategies (id, strategy_id, publisher_id, title, description,
				price_model, price_amount, asset_class, symbols, timeframe, risk_level, tags, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, s.ID, s.StrategyID, s.PublisherID, s.Title, s.Description,
		s.PriceModel, s.PriceAmount, s.AssetClass, s.Symbols, s.Timeframe, s.RiskLevel, s.Tags, s.Status)
	if err != nil {
		return fmt.Errorf("marketplace: publish strategy: %w", err)
	}
	return nil
}

// ListPublished returns all published strategies, ordered by recency.
func (r *Repo) ListPublished(ctx context.Context, assetClass string, limit, offset int) ([]Strategy, error) {
	query := `SELECT id, strategy_id, publisher_id, title, description, price_model, price_amount, asset_class, symbols, timeframe, risk_level, tags, total_subscribers, total_pnl, win_rate, status, created_at, updated_at FROM marketplace_strategies WHERE status = 'published'`
	args := []interface{}{}
	if assetClass != "" {
		query += ` AND asset_class = $1`
		args = append(args, assetClass)
	}
	query += ` ORDER BY created_at DESC LIMIT $` + string(rune(len(args)+1)) + ` OFFSET $` + string(rune(len(args)+2))
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strategies []Strategy
	for rows.Next() {
		var s Strategy
		if err := rows.Scan(
			&s.ID, &s.StrategyID, &s.PublisherID, &s.Title, &s.Description,
			&s.PriceModel, &s.PriceAmount, &s.AssetClass, &s.Symbols, &s.Timeframe,
			&s.RiskLevel, &s.Tags, &s.TotalSubscribers, &s.TotalPnL, &s.WinRate,
			&s.Status, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		strategies = append(strategies, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return strategies, nil
}
