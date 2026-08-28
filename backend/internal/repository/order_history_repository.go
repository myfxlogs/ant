package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"alphaforge/internal/model"
)

func (r *LogRepository) CreateOrderHistory(ctx context.Context, order *model.OrderHistory) error {
	query := `
		INSERT INTO order_history (
			id, user_id, account_id, ticket, order_type, symbol, volume,
			open_price, close_price, open_time, close_time, stop_loss, take_profit,
			profit, commission, swap, comment, magic_number, is_auto_trade, schedule_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`

	_, err := r.db.Exec(ctx, query,
		order.ID, order.UserID, order.AccountID, order.Ticket, order.OrderType, order.Symbol, order.Volume,
		order.OpenPrice, order.ClosePrice, order.OpenTime, order.CloseTime, order.StopLoss, order.TakeProfit,
		order.Profit, order.Commission, order.Swap, order.Comment, order.MagicNumber, order.IsAutoTrade, order.ScheduleID, order.CreatedAt)
	return fmt.Errorf("create order history: %w", err)
}

// UpdateOrderHistoryClose fills close_* / PnL on a row previously inserted for this schedule ticket (first close only).
func (r *LogRepository) UpdateOrderHistoryClose(ctx context.Context, userID, accountID, scheduleID uuid.UUID, ticket int64, closePrice, profit, swap, commission decimal.Decimal, closeTime time.Time) (int64, error) {
	const q = `
		UPDATE order_history
		SET close_price = $5,
			close_time = $6,
			profit = $7,
			swap = $8,
			commission = $9
		WHERE user_id = $1 AND account_id = $2 AND schedule_id = $3 AND ticket = $4
		  AND close_time IS NULL`
	res, err := r.db.Exec(ctx, q, userID, accountID, scheduleID, ticket, closePrice, closeTime, profit, swap, commission)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}

// FIX-2026-08-27-ORDER-HISTORY-MAGIC-ATTRIBUTION (修复 A): GetOrderHistory
// queries trade_records (the live write target) instead of the dead
// order_history table. trade_records carries magic_number + schedule_id
// (populated by writeClosedTradeRecord and SyncOrderHistory), so the
// frontend Order Logs tab and Magic column render real data.
func (r *LogRepository) GetOrderHistory(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.OrderHistory, int, error) {
	baseQ, args, idx := buildOrderHistoryFilters(userID, params)
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) `+baseQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	page, pageSize := normalizeOpLogPagination(params)
	dataQ := fmt.Sprintf(`SELECT id, user_id, account_id, schedule_id, ticket, symbol, order_type, volume, open_price, close_price, profit, open_time, close_time, magic_number %s ORDER BY open_time DESC LIMIT $%d OFFSET $%d`, baseQ, idx, idx+1)
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var orders []*model.OrderHistory
	for rows.Next() {
		var o model.OrderHistory
		var closeTime time.Time
		var scheduleID uuid.NullUUID
		if err := rows.Scan(&o.ID, &o.UserID, &o.AccountID, &scheduleID, &o.Ticket, &o.Symbol, &o.OrderType, &o.Volume, &o.OpenPrice, &o.ClosePrice, &o.Profit, &o.OpenTime, &closeTime, &o.MagicNumber); err != nil {
			return nil, 0, err
		}
		o.CloseTime = &closeTime
		if scheduleID.Valid {
			o.ScheduleID = scheduleID.UUID
		}
		orders = append(orders, &o)
	}
	return orders, total, rows.Err()
}

func buildOrderHistoryFilters(userID uuid.UUID, params *model.LogQueryParams) (baseQ string, args []interface{}, idx int) {
	baseQ = `FROM trade_records WHERE user_id = $1`
	args = []interface{}{userID}
	idx = 2
	if params == nil {
		return
	}
	addFilter := func(col string, val interface{}) {
		baseQ += fmt.Sprintf(` AND %s = $%d`, col, idx)
		args = append(args, val)
		idx++
	}
	if params.ScheduleID != "" {
		if sid, err := uuid.Parse(params.ScheduleID); err == nil {
			addFilter("schedule_id", sid)
		}
	}
	if params.AccountID != "" {
		if aid, err := uuid.Parse(params.AccountID); err == nil {
			addFilter("account_id", aid)
		}
	}
	if params.Symbol != "" {
		addFilter("symbol", params.Symbol)
	}
	if params.Type != "" {
		addFilter("order_type", params.Type)
	}
	if params.StartDate != "" {
		baseQ += fmt.Sprintf(` AND open_time >= $%d`, idx)
		args = append(args, params.StartDate)
		idx++
	}
	if params.EndDate != "" {
		baseQ += fmt.Sprintf(` AND open_time <= $%d`, idx)
		args = append(args, params.EndDate)
		idx++
	}
	return
}
