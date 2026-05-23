package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type TickDatasetRepository struct {
	db *sqlx.DB
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

func NewTickDatasetRepository(db *sqlx.DB) *TickDatasetRepository {
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
	err := r.db.QueryRowxContext(ctx, query, id, ds.UserID, ds.AccountID, ds.Symbol, ds.FromTime, ds.ToTime, ds.Frozen).Scan(&out)
	return out, err
}

func (r *TickDatasetRepository) GetByID(ctx context.Context, id uuid.UUID) (*TickDataset, error) {
	query := `
		SELECT id, user_id, account_id, symbol, from_time, to_time, frozen, created_at, updated_at
		FROM tick_datasets
		WHERE id = $1
	`
	var ds TickDataset
	if err := r.db.GetContext(ctx, &ds, query, id); err != nil {
		return nil, err
	}
	return &ds, nil
}

func (r *TickDatasetRepository) SetFrozen(ctx context.Context, id uuid.UUID, frozen bool) error {
	query := `UPDATE tick_datasets SET frozen = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, frozen)
	return err
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
	return withTx(ctx, r.db, func(tx *sqlx.Tx) error {
		for _, t := range ticks {
			if t == nil {
				continue
			}
			if _, err := tx.ExecContext(ctx, query, t.DatasetID, t.Time, t.Bid, t.Ask); err != nil {
				return err
			}
		}
		return nil
	})
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
	var rows []*TickDatasetTick
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	return rows, nil
}
