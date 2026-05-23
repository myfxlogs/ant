package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"anttrader/internal/model"
)

type TradeRecordRepository struct {
	db *sqlx.DB
}

func NewTradeRecordRepository(db *sqlx.DB) *TradeRecordRepository {
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
	return r.db.QueryRowxContext(ctx, query,
		record.ScheduleID, record.AccountID, record.Ticket, record.Symbol, record.OrderType, record.Volume,
		record.OpenPrice, record.ClosePrice, record.Profit, record.Swap, record.Commission,
		record.OpenTime, record.CloseTime, record.StopLoss, record.TakeProfit,
		record.OrderComment, record.MagicNumber, record.Platform,
	).Scan(&record.ID)
}

func (r *TradeRecordRepository) BatchCreate(ctx context.Context, records []*model.TradeRecord) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

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
	`

	stmt, err := tx.PreparexContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, record := range records {
		_, err := stmt.ExecContext(ctx,
			record.ScheduleID, record.AccountID, record.Ticket, record.Symbol, record.OrderType, record.Volume,
			record.OpenPrice, record.ClosePrice, record.Profit, record.Swap, record.Commission,
			record.OpenTime, record.CloseTime, record.StopLoss, record.TakeProfit,
			record.OrderComment, record.MagicNumber, record.Platform,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *TradeRecordRepository) GetByAccountID(ctx context.Context, accountID uuid.UUID, start, end time.Time, limit int) ([]*model.TradeRecord, error) {
	query := `
		SELECT 
			id, account_id, ticket, symbol, order_type, volume,
			open_price, close_price, profit, swap, commission,
			open_time, close_time, stop_loss, take_profit, order_comment, magic_number, platform
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
		ORDER BY close_time DESC
	`
	args := []interface{}{accountID, start, end}

	if limit > 0 {
		query += " LIMIT $4"
		args = append(args, limit)
	}

	var records []*model.TradeRecord
	err := r.db.SelectContext(ctx, &records, query, args...)
	return records, err
}

func (r *TradeRecordRepository) GetLastSyncTime(ctx context.Context, accountID uuid.UUID) (*time.Time, error) {
	query := `
		SELECT MAX(close_time) FROM trade_records WHERE account_id = $1
	`
	var lastTime *time.Time
	err := r.db.QueryRowxContext(ctx, query, accountID).Scan(&lastTime)
	if err != nil {
		return nil, err
	}
	return lastTime, nil
}

func (r *TradeRecordRepository) CountByAccount(ctx context.Context, accountID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM trade_records WHERE account_id = $1`
	var count int
	err := r.db.QueryRowxContext(ctx, query, accountID).Scan(&count)
	return count, err
}

func (r *TradeRecordRepository) DeleteByAccount(ctx context.Context, accountID uuid.UUID) error {
	query := `DELETE FROM trade_records WHERE account_id = $1`
	_, err := r.db.ExecContext(ctx, query, accountID)
	return err
}
