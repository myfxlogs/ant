package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"anttrader/internal/model"
)

type KlineRepository struct {
	db *sqlx.DB
}

func NewKlineRepository(db *sqlx.DB) *KlineRepository {
	return &KlineRepository{db: db}
}

func (r *KlineRepository) Create(ctx context.Context, kline *model.KlineData) error {
	query := `
		INSERT INTO kline_data (id, symbol, timeframe, open_time, close_time, kline_date, open_price, high_price, low_price, close_price, tick_volume, real_volume, spread, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query,
		kline.ID, kline.Symbol, kline.Timeframe, kline.OpenTime, kline.CloseTime, kline.KlineDate,
		kline.OpenPrice, kline.HighPrice, kline.LowPrice, kline.ClosePrice,
		kline.TickVolume, kline.RealVolume, kline.Spread, kline.CreatedAt, kline.UpdatedAt,
	)
	return err
}

func (r *KlineRepository) BatchCreate(ctx context.Context, klines []*model.KlineData) error {
	if len(klines) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO kline_data (id, symbol, timeframe, open_time, close_time, kline_date, open_price, high_price, low_price, close_price, tick_volume, real_volume, spread, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT DO NOTHING
	`

	for _, kline := range klines {
		_, err := tx.ExecContext(ctx, query,
			kline.ID, kline.Symbol, kline.Timeframe, kline.OpenTime, kline.CloseTime, kline.KlineDate,
			kline.OpenPrice, kline.HighPrice, kline.LowPrice, kline.ClosePrice,
			kline.TickVolume, kline.RealVolume, kline.Spread, kline.CreatedAt, kline.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *KlineRepository) GetBySymbolAndTimeframe(ctx context.Context, symbol, timeframe string, from, to time.Time, limit int) ([]*model.KlineData, error) {
	query := `
		SELECT id, symbol, timeframe, open_time, close_time, open_price, high_price, low_price, close_price, tick_volume, real_volume, spread, created_at, updated_at
		FROM kline_data
		WHERE symbol = $1 AND timeframe = $2 AND open_time >= $3 AND open_time < $4
		ORDER BY open_time ASC
	`
	args := []interface{}{symbol, timeframe, from, to}

	if limit > 0 {
		query += " LIMIT $5"
		args = append(args, limit)
	}

	var klines []*model.KlineData
	err := r.db.SelectContext(ctx, &klines, query, args...)
	return klines, err
}

func (r *KlineRepository) GetLatest(ctx context.Context, symbol, timeframe string) (*model.KlineData, error) {
	query := `
		SELECT id, symbol, timeframe, open_time, close_time, open_price, high_price, low_price, close_price, tick_volume, real_volume, spread, created_at, updated_at
		FROM kline_data
		WHERE symbol = $1 AND timeframe = $2
		ORDER BY open_time DESC
		LIMIT 1
	`
	var kline model.KlineData
	err := r.db.GetContext(ctx, &kline, query, symbol, timeframe)
	if err != nil {
		return nil, err
	}
	return &kline, nil
}

func (r *KlineRepository) DeleteOlderThan(ctx context.Context, olderThan time.Time) error {
	query := `DELETE FROM kline_data WHERE open_time < $1`
	_, err := r.db.ExecContext(ctx, query, olderThan)
	return err
}
