package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TickDatasetRepository struct {
	db *pgxpool.Pool
}

type TickDataset struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	AccountID uuid.UUID `db:"account_id"`
	Symbol    string    `db:"symbol"`
	FromTime  time.Time `db:"from_time"`
	ToTime    time.Time `db:"to_time"`
	Frozen    bool      `db:"frozen"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type TickDatasetTick struct {
	DatasetID uuid.UUID `db:"dataset_id"`
	Time      time.Time `db:"time"`
	Bid       float64   `db:"bid"`
	Ask       float64   `db:"ask"`
	CreatedAt time.Time `db:"created_at"`
}

func NewTickDatasetRepository(db *pgxpool.Pool) *TickDatasetRepository {
	return &TickDatasetRepository{db: db}
}

func (r *TickDatasetRepository) Create(ctx context.Context, ds *TickDataset) (uuid.UUID, error) {
	query := `
			INSERT INTO tick_datasets (id, user_id, account_id, symbol, from_time, to_time, frozen, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			RETURNING id
		`
	id := ds.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	var out uuid.UUID
	err := r.db.QueryRow(ctx, query, id, ds.UserID, ds.AccountID, ds.Symbol, ds.FromTime, ds.ToTime, ds.Frozen).Scan(&out)
	return out, err
}

func (r *TickDatasetRepository) GetByID(ctx context.Context, id uuid.UUID) (*TickDataset, error) {
	query := `
			SELECT id, user_id, account_id, symbol, from_time, to_time, frozen, created_at, updated_at
			FROM tick_datasets
			WHERE id = $1
		`
	var ds TickDataset
	err := r.db.QueryRow(ctx, query, id).Scan(
		&ds.ID, &ds.UserID, &ds.AccountID, &ds.Symbol, &ds.FromTime, &ds.ToTime, &ds.Frozen, &ds.CreatedAt, &ds.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &ds, nil
}

func (r *TickDatasetRepository) SetFrozen(ctx context.Context, id uuid.UUID, frozen bool) error {
	query := `UPDATE tick_datasets SET frozen = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, frozen)
	if err != nil {
		return fmt.Errorf("set tick dataset frozen: %w", err)
	}
	return nil
}

func (r *TickDatasetRepository) BatchInsertTicks(ctx context.Context, ticks []*TickDatasetTick) error {
	if len(ticks) == 0 {
		return nil
	}
	query := `
			INSERT INTO tick_dataset_ticks (dataset_id, time, bid, ask)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT DO NOTHING
		`
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, t := range ticks {
		if t == nil {
			continue
		}
		if _, err := tx.Exec(ctx, query, t.DatasetID, t.Time, t.Bid, t.Ask); err != nil {
			if err != nil {
				return fmt.Errorf("insert tick: %w", err)
			}
			return nil
		}
	}
	return tx.Commit(ctx)
}

func (r *TickDatasetRepository) ListTicks(ctx context.Context, datasetID uuid.UUID, limit int) ([]*TickDatasetTick, error) {
	query := `
			SELECT dataset_id, time, bid, ask, created_at
			FROM tick_dataset_ticks
			WHERE dataset_id = $1
			ORDER BY time ASC
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
	var result []*TickDatasetTick
	for rows.Next() {
		var t TickDatasetTick
		if err := rows.Scan(&t.DatasetID, &t.Time, &t.Bid, &t.Ask, &t.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
