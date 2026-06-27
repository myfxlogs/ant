package repository

import (
	"context"
	"fmt"

	"anttrader/internal/model"
)

func (r *AdminRepository) ListPositions(ctx context.Context, userID, accountID, symbol string, page, pageSize int) ([]*model.Position, int64, error) {
	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 20 }
	if pageSize > 100 { pageSize = 100 }
	offset := (page - 1) * pageSize

	countQ, query, args := buildPositionFilters(userID, accountID, symbol)
	var total int64
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil { return nil, 0, err }
	query += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var positions []*model.Position
	for rows.Next() {
		var p model.Position
		if err := rows.Scan(&p.ID, &p.MTAccountID, &p.Platform, &p.Ticket, &p.Symbol, &p.OrderType,
			&p.Volume, &p.OpenPrice, &p.CurrentPrice, &p.StopLoss, &p.TakeProfit,
			&p.OpenTime, &p.Profit, &p.Swap, &p.Commission, &p.Fee,
			&p.OrderComment, &p.MagicNumber, &p.CloseReason, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		positions = append(positions, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return positions, total, nil
}

func buildPositionFilters(userID, accountID, symbol string) (countQ, query string, args []interface{}) {
	countQ = `SELECT COUNT(*) FROM positions p JOIN mt_accounts ma ON p.mt_account_id = ma.id WHERE 1=1`
	query = `SELECT p.* FROM positions p JOIN mt_accounts ma ON p.mt_account_id = ma.id WHERE 1=1`
	addFilter := func(cond, val string) {
		countQ += fmt.Sprintf(" AND %s = $%d", cond, len(args)+1)
		query += fmt.Sprintf(" AND %s = $%d", cond, len(args)+1)
		args = append(args, val)
	}
	if userID != "" { addFilter("ma.user_id", userID) }
	if accountID != "" { addFilter("p.mt_account_id", accountID) }
	if symbol != "" { addFilter("p.symbol", symbol) }
	return
}

func (r *AdminRepository) ListOrders(ctx context.Context, userID, accountID, symbol, orderType, status string, page, pageSize int) ([]*model.Order, int64, error) {
	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 20 }
	if pageSize > 100 { pageSize = 100 }
	offset := (page - 1) * pageSize

	countQ, query, args := buildOrderFilters(userID, accountID, symbol, orderType)
	var total int64
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil { return nil, 0, err }
	query += fmt.Sprintf(" ORDER BY o.created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var orders []*model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.MTAccountID, &o.Platform, &o.Ticket, &o.Symbol, &o.OrderType,
			&o.Volume, &o.Price, &o.StopLimitPrice, &o.StopLoss, &o.TakeProfit,
			&o.Expiration, &o.ExpirationType, &o.PlacedType,
			&o.OrderComment, &o.MagicNumber, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, &o)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func buildOrderFilters(userID, accountID, symbol, orderType string) (countQ, query string, args []interface{}) {
	countQ = `SELECT COUNT(*) FROM orders o JOIN mt_accounts ma ON o.mt_account_id = ma.id WHERE 1=1`
	query = `SELECT o.* FROM orders o JOIN mt_accounts ma ON o.mt_account_id = ma.id WHERE 1=1`
	addFilter := func(cond, val string) {
		countQ += fmt.Sprintf(" AND %s = $%d", cond, len(args)+1)
		query += fmt.Sprintf(" AND %s = $%d", cond, len(args)+1)
		args = append(args, val)
	}
	if userID != "" { addFilter("ma.user_id", userID) }
	if accountID != "" { addFilter("o.mt_account_id", accountID) }
	if symbol != "" { addFilter("o.symbol", symbol) }
	if orderType != "" { addFilter("o.order_type", orderType) }
	return
}
