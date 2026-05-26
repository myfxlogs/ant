package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrMarketRegimeNotFound = errors.New("market regime not found")

type MarketRegime struct {
	ID               uuid.UUID  `db:"id"`
	UserID           uuid.UUID  `db:"user_id"`
	AccountID        uuid.UUID  `db:"account_id"`
	Symbol           string     `db:"symbol"`
	Timeframe        string     `db:"timeframe"`
	Regime           string     `db:"regime"`
	Confidence       float64    `db:"confidence"`
	Features         []byte     `db:"features"`
	Segments         []byte     `db:"segments"`
	StrategyFamilies []string   `db:"strategy_families"`
	FromTime         *time.Time `db:"from_time"`
	ToTime           *time.Time `db:"to_time"`
	ModelVersion     string     `db:"model_version"`
	CreatedAt        time.Time  `db:"created_at"`
}

type MarketRegimeRepository struct {
	db *pgxpool.Pool
}

func NewMarketRegimeRepository(db *pgxpool.Pool) *MarketRegimeRepository {
	return &MarketRegimeRepository{db: db}
}

func (r *MarketRegimeRepository) Create(ctx context.Context, row *MarketRegime) error {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	if len(row.Features) == 0 {
		row.Features = []byte(`{}`)
	}
	if len(row.Segments) == 0 {
		row.Segments = []byte(`[]`)
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO market_regimes (id,user_id,account_id,symbol,timeframe,regime,confidence,features,segments,strategy_families,from_time,to_time,model_version,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`, row.ID, row.UserID, row.AccountID, row.Symbol, row.Timeframe, row.Regime, row.Confidence, row.Features, row.Segments, row.StrategyFamilies, row.FromTime, row.ToTime, row.ModelVersion, row.CreatedAt)
	if err != nil {
		return fmt.Errorf("create market regime: %w", err)
	}
	return nil
}

func (r *MarketRegimeRepository) Get(ctx context.Context, userID, id uuid.UUID) (*MarketRegime, error) {
	var row MarketRegime
	err := r.db.QueryRow(ctx, `SELECT * FROM market_regimes WHERE id = $1 AND user_id = $2`, id, userID).Scan(&row.ID, &row.UserID, &row.AccountID, &row.Symbol, &row.Timeframe, &row.Regime, &row.Confidence, &row.Features, &row.Segments, &row.StrategyFamilies, &row.FromTime, &row.ToTime, &row.ModelVersion, &row.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMarketRegimeNotFound
	}
	return &row, err
}

func (r *MarketRegimeRepository) GetByID(ctx context.Context, id uuid.UUID) (*MarketRegime, error) {
	var row MarketRegime
	err := r.db.QueryRow(ctx, `SELECT * FROM market_regimes WHERE id = $1`, id).Scan(&row.ID, &row.UserID, &row.AccountID, &row.Symbol, &row.Timeframe, &row.Regime, &row.Confidence, &row.Features, &row.Segments, &row.StrategyFamilies, &row.FromTime, &row.ToTime, &row.ModelVersion, &row.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMarketRegimeNotFound
	}
	return &row, err
}
