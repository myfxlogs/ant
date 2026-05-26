package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

type TradeRecord struct {
	ID         uuid.UUID `db:"id"`
	AccountID  uuid.UUID `db:"account_id"`
	Symbol     string    `db:"symbol"`
	OrderType  string    `db:"order_type"`
	Volume     float64   `db:"volume"`
	OpenPrice  float64   `db:"open_price"`
	ClosePrice float64   `db:"close_price"`
	Profit     float64   `db:"profit"`
	Swap       float64   `db:"swap"`
	Commission float64   `db:"commission"`
	OpenTime   time.Time `db:"open_time"`
	CloseTime  time.Time `db:"close_time"`
}

func (r *AnalyticsRepository) GetTradeRecords(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*TradeRecord, error) {
	query := `
		SELECT
			id, account_id, symbol, order_type, volume,
			open_price, close_price, profit, swap, commission,
			open_time, close_time
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
		ORDER BY close_time ASC
	`
	rows, err := r.db.Query(ctx, query, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []*TradeRecord
	for rows.Next() {
		rec := &TradeRecord{}
		if err := rows.Scan(&rec.ID, &rec.AccountID, &rec.Symbol, &rec.OrderType, &rec.Volume, &rec.OpenPrice, &rec.ClosePrice, &rec.Profit, &rec.Swap, &rec.Commission, &rec.OpenTime, &rec.CloseTime); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *AnalyticsRepository) GetTradeRecordsByUser(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*TradeRecord, error) {
	query := `
		SELECT
			tr.id, tr.account_id, tr.symbol, tr.order_type, tr.volume,
			tr.open_price, tr.close_price, tr.profit, tr.swap, tr.commission,
			tr.open_time, tr.close_time
		FROM trade_records tr
		JOIN mt_accounts ma ON tr.account_id = ma.id
		WHERE ma.user_id = $1 AND tr.close_time >= $2 AND tr.close_time <= $3
		ORDER BY tr.close_time ASC
	`
	rows, err := r.db.Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []*TradeRecord
	for rows.Next() {
		rec := &TradeRecord{}
		if err := rows.Scan(&rec.ID, &rec.AccountID, &rec.Symbol, &rec.OrderType, &rec.Volume, &rec.OpenPrice, &rec.ClosePrice, &rec.Profit, &rec.Swap, &rec.Commission, &rec.OpenTime, &rec.CloseTime); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *AnalyticsRepository) GetTradeLogsByAccount(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.TradeLog, error) {
	query := `
		SELECT * FROM trade_logs
		WHERE account_id = $1 AND created_at >= $2 AND created_at <= $3
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(ctx, query, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []*model.TradeLog
	for rows.Next() {
		t := &model.TradeLog{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.AccountID, &t.Action, &t.Symbol, &t.OrderType, &t.Volume, &t.Price, &t.Ticket, &t.Profit, &t.Message, &t.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, t)
	}
	return logs, rows.Err()
}

func (r *AnalyticsRepository) GetTradeRecordsWithLimit(ctx context.Context, accountID uuid.UUID, start, end time.Time, limit int) ([]*model.TradeRecord, error) {
	query := `
		SELECT
			id, account_id, ticket, symbol, order_type, volume,
			open_price, close_price, profit, swap, commission,
			open_time, close_time, stop_loss, take_profit, order_comment, magic_number
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
		ORDER BY close_time DESC
	`
	if limit > 0 {
		query += " LIMIT $4"
		rows, err := r.db.Query(ctx, query, accountID, start, end, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var records []*model.TradeRecord
		for rows.Next() {
			rec := &model.TradeRecord{}
			if err := rows.Scan(&rec.ID, &rec.AccountID, &rec.Ticket, &rec.Symbol, &rec.OrderType, &rec.Volume, &rec.OpenPrice, &rec.ClosePrice, &rec.Profit, &rec.Swap, &rec.Commission, &rec.OpenTime, &rec.CloseTime, &rec.StopLoss, &rec.TakeProfit, &rec.OrderComment, &rec.MagicNumber); err != nil {
				return nil, err
			}
			records = append(records, rec)
		}
		return records, rows.Err()
	}
	rows, err := r.db.Query(ctx, query, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []*model.TradeRecord
	for rows.Next() {
		rec := &model.TradeRecord{}
		if err := rows.Scan(&rec.ID, &rec.AccountID, &rec.Ticket, &rec.Symbol, &rec.OrderType, &rec.Volume, &rec.OpenPrice, &rec.ClosePrice, &rec.Profit, &rec.Swap, &rec.Commission, &rec.OpenTime, &rec.CloseTime, &rec.StopLoss, &rec.TakeProfit, &rec.OrderComment, &rec.MagicNumber); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *AnalyticsRepository) GetTradeRecordsCount(ctx context.Context, accountID uuid.UUID, start, end time.Time) (int, error) {
	query := `SELECT COUNT(*) FROM trade_records WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3`
	var total int
	err := r.db.QueryRow(ctx, query, accountID, start, end).Scan(&total)
	return total, err
}

func (r *AnalyticsRepository) GetTradeRecordsPaginated(ctx context.Context, accountID uuid.UUID, start, end time.Time, page, pageSize int) ([]*model.TradeRecord, int, error) {
	countQuery := `SELECT COUNT(*) FROM trade_records WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3`
	var total int
	err := r.db.QueryRow(ctx, countQuery, accountID, start, end).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := `
		SELECT
			id, account_id, ticket, symbol, order_type, volume,
			open_price, close_price, profit, swap, commission,
			open_time, close_time, stop_loss, take_profit, order_comment, magic_number
		FROM trade_records
		WHERE account_id = $1 AND close_time >= $2 AND close_time <= $3
		ORDER BY close_time DESC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.db.Query(ctx, query, accountID, start, end, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var records []*model.TradeRecord
	for rows.Next() {
		rec := &model.TradeRecord{}
		if err := rows.Scan(&rec.ID, &rec.AccountID, &rec.Ticket, &rec.Symbol, &rec.OrderType, &rec.Volume, &rec.OpenPrice, &rec.ClosePrice, &rec.Profit, &rec.Swap, &rec.Commission, &rec.OpenTime, &rec.CloseTime, &rec.StopLoss, &rec.TakeProfit, &rec.OrderComment, &rec.MagicNumber); err != nil {
			return nil, 0, err
		}
		records = append(records, rec)
	}
	return records, total, rows.Err()
}
