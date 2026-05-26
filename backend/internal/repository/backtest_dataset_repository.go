package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BacktestDatasetRepository struct {
	db *pgxpool.Pool
}

type BacktestDataset struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	AccountID uuid.UUID  `db:"account_id"`
	Symbol    string     `db:"symbol"`
	Timeframe string     `db:"timeframe"`
	FromTime  *time.Time `db:"from_time"`
	ToTime    *time.Time `db:"to_time"`
	Count     int        `db:"count"`
	Frozen    bool       `db:"frozen"`
	CostModelSnapshot []byte `db:"cost_model_snapshot"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
}

type BacktestDatasetBar struct {
	DatasetID  uuid.UUID `db:"dataset_id"`
	Symbol     string    `db:"symbol"`
	Timeframe  string    `db:"timeframe"`
	OpenTime   time.Time `db:"open_time"`
	CloseTime  time.Time `db:"close_time"`
	OpenPrice  float64   `db:"open_price"`
	HighPrice  float64   `db:"high_price"`
	LowPrice   float64   `db:"low_price"`
	ClosePrice float64   `db:"close_price"`
	TickVolume int64     `db:"tick_volume"`
	CreatedAt  time.Time `db:"created_at"`
}

func NewBacktestDatasetRepository(db *pgxpool.Pool) *BacktestDatasetRepository {
	return &BacktestDatasetRepository{db: db}
}

func (r *BacktestDatasetRepository) Create(ctx context.Context, ds *BacktestDataset) (uuid.UUID, error) {
	query := `
		INSERT INTO backtest_datasets (id, user_id, account_id, symbol, timeframe, from_time, to_time, count, frozen, cost_model_snapshot, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`
	id := ds.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	var out uuid.UUID
	err := r.db.QueryRow(ctx, query,
		id, ds.UserID, ds.AccountID, ds.Symbol, ds.Timeframe, ds.FromTime, ds.ToTime, ds.Count, ds.Frozen, ds.CostModelSnapshot,
	).Scan(&out)
	return out, err
}

func (r *BacktestDatasetRepository) GetByID(ctx context.Context, id uuid.UUID) (*BacktestDataset, error) {
	var ds BacktestDataset
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, account_id, symbol, timeframe, from_time, to_time, count, frozen, cost_model_snapshot, created_at, updated_at
		FROM backtest_datasets
		WHERE id = $1`, id,
	).Scan(
		&ds.ID, &ds.UserID, &ds.AccountID, &ds.Symbol, &ds.Timeframe,
		&ds.FromTime, &ds.ToTime, &ds.Count, &ds.Frozen, &ds.CostModelSnapshot,
		&ds.CreatedAt, &ds.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &ds, nil
}

func (r *BacktestDatasetRepository) List(ctx context.Context, userID uuid.UUID, accountID *uuid.UUID, symbol *string, timeframe *string, limit int, offset int) ([]*BacktestDataset, error) {
	query := `
		SELECT id, user_id, account_id, symbol, timeframe, from_time, to_time, count, frozen, cost_model_snapshot, created_at, updated_at
		FROM backtest_datasets
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	idx := 2
	if accountID != nil && *accountID != uuid.Nil {
		query += " AND account_id = $" + itoa(idx)
		args = append(args, *accountID)
		idx++
	}
	if symbol != nil && *symbol != "" {
		query += " AND symbol = $" + itoa(idx)
		args = append(args, *symbol)
		idx++
	}
	if timeframe != nil && *timeframe != "" {
		query += " AND timeframe = $" + itoa(idx)
		args = append(args, *timeframe)
		idx++
	}
	query += " ORDER BY created_at DESC"
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	query += " LIMIT $" + itoa(idx)
	args = append(args, limit)
	idx++
	query += " OFFSET $" + itoa(idx)
	args = append(args, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var datasets []*BacktestDataset
	for rows.Next() {
		var ds BacktestDataset
		if err := rows.Scan(
			&ds.ID, &ds.UserID, &ds.AccountID, &ds.Symbol, &ds.Timeframe,
			&ds.FromTime, &ds.ToTime, &ds.Count, &ds.Frozen, &ds.CostModelSnapshot,
			&ds.CreatedAt, &ds.UpdatedAt,
		); err != nil {
			return nil, err
		}
		datasets = append(datasets, &ds)
	}
	return datasets, rows.Err()
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

func (r *BacktestDatasetRepository) SetFrozen(ctx context.Context, id uuid.UUID, frozen bool) error {
	query := `UPDATE backtest_datasets SET frozen = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, frozen)
	if err != nil {
		return fmt.Errorf("set dataset frozen: %w", err)
	}
	return nil
}

func (r *BacktestDatasetRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error) {
	query := `DELETE FROM backtest_datasets WHERE id = $1 AND user_id = $2`
	ct, err := r.db.Exec(ctx, query, id, userID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

func (r *BacktestDatasetRepository) BatchInsertBars(ctx context.Context, bars []*BacktestDatasetBar) error {
	if len(bars) == 0 {
		return nil
	}
	query := `
		INSERT INTO backtest_dataset_bars (
			dataset_id, symbol, timeframe, open_time, close_time,
			open_price, high_price, low_price, close_price, tick_volume
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT DO NOTHING
	`
	return withTx(ctx, r.db, func(tx pgx.Tx) error {
		for _, b := range bars {
			if b == nil {
				continue
			}
			if _, err := tx.Exec(ctx, query,
				b.DatasetID, b.Symbol, b.Timeframe, b.OpenTime, b.CloseTime,
				b.OpenPrice, b.HighPrice, b.LowPrice, b.ClosePrice, b.TickVolume,
			); err != nil {
				return fmt.Errorf("insert dataset bar: %w", err)
			}
		}
		return nil
	})
}

func (r *BacktestDatasetRepository) ListBars(ctx context.Context, datasetID uuid.UUID, limit int) ([]*BacktestDatasetBar, error) {
	query := `
		SELECT dataset_id, symbol, timeframe, open_time, close_time, open_price, high_price, low_price, close_price, tick_volume, created_at
		FROM backtest_dataset_bars
		WHERE dataset_id = $1
		ORDER BY open_time ASC
	`
	args := []interface{}{datasetID}
	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bars []*BacktestDatasetBar
	for rows.Next() {
		var b BacktestDatasetBar
		if err := rows.Scan(
			&b.DatasetID, &b.Symbol, &b.Timeframe,
			&b.OpenTime, &b.CloseTime,
			&b.OpenPrice, &b.HighPrice, &b.LowPrice, &b.ClosePrice,
			&b.TickVolume, &b.CreatedAt,
		); err != nil {
			return nil, err
		}
		bars = append(bars, &b)
	}
	return bars, rows.Err()
}

func withTx(ctx context.Context, db *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("list dataset bars: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		if err != nil {
			return fmt.Errorf("list dataset bars: %w", err)
		}
		return nil
	}
	return tx.Commit(ctx)
}
