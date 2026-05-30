package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"anttrader/internal/model"
)

type TradeRecordRepository struct {
	db *pgxpool.Pool
}

func NewTradeRecordRepository(db *pgxpool.Pool) *TradeRecordRepository {
	return &TradeRecordRepository{db: db}
}

func (r *TradeRecordRepository) Create(ctx context.Context, record *model.TradeRecord) error {
	query := `
		INSERT INTO trade_records (
			schedule_id, account_id, ticket, symbol, order_type, volume,
			open_price, close_price, profit, swap, commission,
			open_time, close_time, stop_loss, take_profit,
			order_comment, magic_number, platform
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		) ON CONFLICT (account_id, ticket, close_time) DO UPDATE SET
			schedule_id = COALESCE(EXCLUDED.schedule_id, trade_records.schedule_id),
			profit = EXCLUDED.profit,
			swap = EXCLUDED.swap,
			commission = EXCLUDED.commission,
			close_price = EXCLUDED.close_price,
			platform = EXCLUDED.platform,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`
	return r.db.QueryRow(ctx, query,
		record.ScheduleID, record.AccountID, record.Ticket, record.Symbol, record.OrderType, record.Volume,
		record.OpenPrice, record.ClosePrice, record.Profit, record.Swap, record.Commission,
		record.OpenTime, record.CloseTime, record.StopLoss, record.TakeProfit,
		record.OrderComment, record.MagicNumber, record.Platform,
	).Scan(&record.ID)
}

const maxBatchSize = 500

func (r *TradeRecordRepository) BatchCreate(ctx context.Context, records []*model.TradeRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Split into chunks of maxBatchSize to avoid oversized transactions
	// and excessive lock contention.
	for start := 0; start < len(records); start += maxBatchSize {
		end := start + maxBatchSize
		if end > len(records) {
			end = len(records)
		}
		if err := r.batchCreateChunk(ctx, records[start:end]); err != nil {
			return fmt.Errorf("batch create trade record chunk [%d:%d]: %w", start, end, err)
		}
	}
	return nil
}

func (r *TradeRecordRepository) batchCreateChunk(ctx context.Context, records []*model.TradeRecord) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("batch create trade record: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO trade_records (
			schedule_id, account_id, ticket, symbol, order_type, volume,
			open_price, close_price, profit, swap, commission,
			open_time, close_time, stop_loss, take_profit,
			order_comment, magic_number, platform
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		) ON CONFLICT (account_id, ticket, close_time) DO UPDATE SET
			schedule_id = COALESCE(EXCLUDED.schedule_id, trade_records.schedule_id),
			profit = EXCLUDED.profit,
			swap = EXCLUDED.swap,
			commission = EXCLUDED.commission,
			close_price = EXCLUDED.close_price,
			platform = EXCLUDED.platform,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`

	for _, record := range records {
		var returnedID uuid.UUID
		if err := tx.QueryRow(ctx, query,
			record.ScheduleID, record.AccountID, record.Ticket, record.Symbol, record.OrderType, record.Volume,
			record.OpenPrice, record.ClosePrice, record.Profit, record.Swap, record.Commission,
			record.OpenTime, record.CloseTime, record.StopLoss, record.TakeProfit,
			record.OrderComment, record.MagicNumber, record.Platform,
		).Scan(&returnedID); err != nil {
			return fmt.Errorf("batch create trade record ticket=%d: %w", record.Ticket, err)
		}
		record.ID = returnedID
	}

	return tx.Commit(ctx)
}

func (r *TradeRecordRepository) GetByAccountID(ctx context.Context, accountID uuid.UUID, start, end time.Time, limit int) ([]*model.TradeRecord, error) {
	query := `
		SELECT
			id, schedule_id, account_id, ticket, symbol, order_type, volume,
			open_price, close_price, profit, swap, commission,
			open_time, close_time, stop_loss, take_profit, order_comment, magic_number, platform,
			created_at, updated_at
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
		ORDER BY close_time DESC
	`
	args := []interface{}{accountID, start, end}

	if limit > 0 {
		query += " LIMIT $4"
		args = append(args, limit)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*model.TradeRecord
	for rows.Next() {
		var rec model.TradeRecord
		if err := rows.Scan(
			&rec.ID, &rec.ScheduleID, &rec.AccountID, &rec.Ticket, &rec.Symbol, &rec.OrderType,
			&rec.Volume, &rec.OpenPrice, &rec.ClosePrice, &rec.Profit, &rec.Swap, &rec.Commission,
			&rec.OpenTime, &rec.CloseTime, &rec.StopLoss, &rec.TakeProfit,
			&rec.OrderComment, &rec.MagicNumber, &rec.Platform,
			&rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, &rec)
	}
	return records, rows.Err()
}

func (r *TradeRecordRepository) GetLastSyncTime(ctx context.Context, accountID uuid.UUID) (*time.Time, error) {
	query := `
		SELECT MAX(close_time) FROM trade_records WHERE account_id = $1
	`
	var lastTime *time.Time
	err := r.db.QueryRow(ctx, query, accountID).Scan(&lastTime)
	if err != nil {
		return nil, err
	}
	return lastTime, nil
}

func (r *TradeRecordRepository) CountByAccount(ctx context.Context, accountID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM trade_records WHERE account_id = $1`
	var count int
	err := r.db.QueryRow(ctx, query, accountID).Scan(&count)
	return count, err
}

func (r *TradeRecordRepository) DeleteByAccount(ctx context.Context, accountID uuid.UUID) error {
	query := `DELETE FROM trade_records WHERE account_id = $1`
	_, err := r.db.Exec(ctx, query, accountID)
	if err != nil {
		return fmt.Errorf("delete trade records by account: %w", err)
	}
	return nil
}
