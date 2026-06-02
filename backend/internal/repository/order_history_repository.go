package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
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
func (r *LogRepository) UpdateOrderHistoryClose(ctx context.Context, userID, accountID, scheduleID uuid.UUID, ticket int64, closePrice, profit, swap, commission float64, closeTime time.Time) (int64, error) {
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

func (r *LogRepository) GetOrderHistory(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.OrderHistory, int, error) {
	baseQuery := `FROM order_history WHERE user_id = $1`
	args := []interface{}{userID}
	argIndex := 2

	if params != nil {
		if params.ScheduleID != "" {
			baseQuery += fmt.Sprintf(` AND schedule_id = $%d`, argIndex)
			scheduleID, _ := uuid.Parse(params.ScheduleID)
			args = append(args, scheduleID)
			argIndex++
		}
		if params.AccountID != "" {
			baseQuery += fmt.Sprintf(` AND account_id = $%d`, argIndex)
			accountID, _ := uuid.Parse(params.AccountID)
			args = append(args, accountID)
			argIndex++
		}
		if params.Symbol != "" {
			baseQuery += fmt.Sprintf(` AND symbol = $%d`, argIndex)
			args = append(args, params.Symbol)
			argIndex++
		}
		if params.Type != "" {
			baseQuery += fmt.Sprintf(` AND order_type = $%d`, argIndex)
			args = append(args, params.Type)
			argIndex++
		}
		if params.StartDate != "" {
			baseQuery += fmt.Sprintf(` AND open_time >= $%d`, argIndex)
			args = append(args, params.StartDate)
			argIndex++
		}
		if params.EndDate != "" {
			baseQuery += fmt.Sprintf(` AND open_time <= $%d`, argIndex)
			args = append(args, params.EndDate)
			argIndex++
		}
	}

	countQuery := `SELECT COUNT(*) ` + baseQuery
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	page := 1
	pageSize := 20
	if params != nil {
		if params.Page > 0 {
			page = params.Page
		}
		if params.PageSize > 0 {
			pageSize = params.PageSize
		}
	}

	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf(`SELECT * %s ORDER BY open_time DESC LIMIT $%d OFFSET $%d`, baseQuery, argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	queryRows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer queryRows.Close()
	var orders []*model.OrderHistory
	for queryRows.Next() {
		var order model.OrderHistory
		if err := queryRows.Scan(&order.ID, &order.UserID, &order.AccountID, &order.Ticket, &order.OrderType, &order.Symbol, &order.Volume, &order.OpenPrice, &order.ClosePrice, &order.OpenTime, &order.CloseTime, &order.StopLoss, &order.TakeProfit, &order.Profit, &order.Commission, &order.Swap, &order.Comment, &order.MagicNumber, &order.IsAutoTrade, &order.ScheduleID, &order.CreatedAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, &order)
	}
	if err := queryRows.Err(); err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}
