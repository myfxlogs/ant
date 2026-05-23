package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type BacktestDatasetRepository struct {
	db *sqlx.DB
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

func NewBacktestDatasetRepository(db *sqlx.DB) *BacktestDatasetRepository {
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
	err := r.db.QueryRowxContext(ctx, query,
		id, ds.UserID, ds.AccountID, ds.Symbol, ds.Timeframe, ds.FromTime, ds.ToTime, ds.Count, ds.Frozen, ds.CostModelSnapshot,
	).Scan(&out)
	return out, err
}

func (r *BacktestDatasetRepository) GetByID(ctx context.Context, id uuid.UUID) (*BacktestDataset, error) {
	query := `
		SELECT id, user_id, account_id, symbol, timeframe, from_time, to_time, count, frozen, cost_model_snapshot, created_at, updated_at
		FROM backtest_datasets
		WHERE id = $1
	`
	var ds BacktestDataset
	if err := r.db.GetContext(ctx, &ds, query, id); err != nil {
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

	var rows []*BacktestDataset
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	return rows, nil
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

func (r *BacktestDatasetRepository) SetFrozen(ctx context.Context, id uuid.UUID, frozen bool) error {
	query := `UPDATE backtest_datasets SET frozen = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, frozen)
	return err
}

func (r *BacktestDatasetRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error) {
	query := `DELETE FROM backtest_datasets WHERE id = $1 AND user_id = $2`
	res, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
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
	return withTx(ctx, r.db, func(tx *sqlx.Tx) error {
		for _, b := range bars {
			if b == nil {
				continue
			}
			if _, err := tx.ExecContext(ctx, query,
				b.DatasetID, b.Symbol, b.Timeframe, b.OpenTime, b.CloseTime,
				b.OpenPrice, b.HighPrice, b.LowPrice, b.ClosePrice, b.TickVolume,
			); err != nil {
				return err
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
	var rows []*BacktestDatasetBar
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	return rows, nil
}

func withTx(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
