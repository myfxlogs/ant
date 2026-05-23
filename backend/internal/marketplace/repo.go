// Package marketplace — strategy marketplace API (M5-1).
// Provides CRUD for marketplace_strategies table.
package marketplace

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

// Publish creates a new marketplace listing.
func (r *Repo) Publish(ctx context.Context, s *Strategy) error {
	s.ID = uuid.New()
	s.CreatedAt = time.Now()
	s.UpdatedAt = s.CreatedAt
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO marketplace_strategies (id, strategy_id, publisher_id, title, description,
			price_model, price_amount, asset_class, symbols, timeframe, risk_level, tags, status)
		VALUES (:id, :strategy_id, :publisher_id, :title, :description,
			:price_model, :price_amount, :asset_class, :symbols, :timeframe, :risk_level, :tags, :status)
	`, s)
	return err
}

// ListPublished returns all published strategies, ordered by recency.
func (r *Repo) ListPublished(ctx context.Context, assetClass string, limit, offset int) ([]Strategy, error) {
	var rows []Strategy
	query := `SELECT * FROM marketplace_strategies WHERE status = 'published'`
	args := []interface{}{}
	if assetClass != "" {
		query += ` AND asset_class = $1`
		args = append(args, assetClass)
	}
	query += ` ORDER BY created_at DESC LIMIT $` + string(rune(len(args)+1)) + ` OFFSET $` + string(rune(len(args)+2))
	args = append(args, limit, offset)
	err := r.db.SelectContext(ctx, &rows, r.db.Rebind(query), args...)
	return rows, err
}
