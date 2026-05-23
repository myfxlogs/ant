package repository

import (
	"context"
	"fmt"

	"anttrader/internal/model"
)

func (r *AdminRepository) ListPositions(ctx context.Context, userID, accountID, symbol string, page, pageSize int) ([]*model.Position, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	countQuery := `SELECT COUNT(*) FROM positions p JOIN mt_accounts ma ON p.mt_account_id = ma.id WHERE 1=1`
	query := `
		SELECT p.* FROM positions p
		JOIN mt_accounts ma ON p.mt_account_id = ma.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if userID != "" {
		countQuery += fmt.Sprintf(" AND ma.user_id = $%d", argIndex)
		query += fmt.Sprintf(" AND ma.user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	if accountID != "" {
		countQuery += fmt.Sprintf(" AND p.mt_account_id = $%d", argIndex)
		query += fmt.Sprintf(" AND p.mt_account_id = $%d", argIndex)
		args = append(args, accountID)
		argIndex++
	}

	if symbol != "" {
		countQuery += fmt.Sprintf(" AND p.symbol = $%d", argIndex)
		query += fmt.Sprintf(" AND p.symbol = $%d", argIndex)
		args = append(args, symbol)
		argIndex++
	}

	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query += fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var positions []*model.Position
	for rows.Next() {
		var p model.Position
		err := rows.Scan(
			&p.ID, &p.MTAccountID, &p.Platform, &p.Ticket, &p.Symbol, &p.OrderType,
			&p.Volume, &p.OpenPrice, &p.CurrentPrice, &p.StopLoss, &p.TakeProfit,
			&p.OpenTime, &p.Profit, &p.Swap, &p.Commission, &p.Fee,
			&p.OrderComment, &p.MagicNumber, &p.CloseReason, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		positions = append(positions, &p)
	}

	return positions, total, nil
}

func (r *AdminRepository) ListOrders(ctx context.Context, userID, accountID, symbol, orderType, status string, page, pageSize int) ([]*model.Order, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	countQuery := `SELECT COUNT(*) FROM orders o JOIN mt_accounts ma ON o.mt_account_id = ma.id WHERE 1=1`
	query := `
		SELECT o.* FROM orders o
		JOIN mt_accounts ma ON o.mt_account_id = ma.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if userID != "" {
		countQuery += fmt.Sprintf(" AND ma.user_id = $%d", argIndex)
		query += fmt.Sprintf(" AND ma.user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	if accountID != "" {
		countQuery += fmt.Sprintf(" AND o.mt_account_id = $%d", argIndex)
		query += fmt.Sprintf(" AND o.mt_account_id = $%d", argIndex)
		args = append(args, accountID)
		argIndex++
	}

	if symbol != "" {
		countQuery += fmt.Sprintf(" AND o.symbol = $%d", argIndex)
		query += fmt.Sprintf(" AND o.symbol = $%d", argIndex)
		args = append(args, symbol)
		argIndex++
	}

	if orderType != "" {
		countQuery += fmt.Sprintf(" AND o.order_type = $%d", argIndex)
		query += fmt.Sprintf(" AND o.order_type = $%d", argIndex)
		args = append(args, orderType)
		argIndex++
	}

	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query += fmt.Sprintf(" ORDER BY o.created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var o model.Order
		err := rows.Scan(
			&o.ID, &o.MTAccountID, &o.Platform, &o.Ticket, &o.Symbol, &o.OrderType,
			&o.Volume, &o.Price, &o.StopLimitPrice, &o.StopLoss, &o.TakeProfit,
			&o.Expiration, &o.ExpirationType, &o.PlacedType,
			&o.OrderComment, &o.MagicNumber, &o.CreatedAt, &o.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, &o)
	}

	return orders, total, nil
}
