package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"anttrader/internal/model"
)

type TradeLogRepository struct {
	db *pgxpool.Pool
}

func NewTradeLogRepository(db *pgxpool.Pool) *TradeLogRepository {
	return &TradeLogRepository{db: db}
}

func (r *TradeLogRepository) Create(ctx context.Context, log *model.TradeLog) error {
	query := `
		INSERT INTO trade_logs (id, user_id, account_id, action, symbol, order_type, volume, price, ticket, profit, message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.Exec(ctx, query,
		log.ID,
		log.UserID,
		log.AccountID,
		log.Action,
		log.Symbol,
		log.OrderType,
		log.Volume,
		log.Price,
		log.Ticket,
		log.Profit,
		log.Message,
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create trade log: %w", err)
	}
	return nil
}

func (r *TradeLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.TradeLog, error) {
	var log model.TradeLog
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, account_id, action, symbol, order_type, volume, price, ticket, profit, message, created_at
		FROM trade_logs WHERE id = $1`, id,
	).Scan(
		&log.ID, &log.UserID, &log.AccountID, &log.Action,
		&log.Symbol, &log.OrderType, &log.Volume, &log.Price,
		&log.Ticket, &log.Profit, &log.Message, &log.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *TradeLogRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.TradeLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, account_id, action, symbol, order_type, volume, price, ticket, profit, message, created_at
		FROM trade_logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.TradeLog
	for rows.Next() {
		var l model.TradeLog
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.AccountID, &l.Action,
			&l.Symbol, &l.OrderType, &l.Volume, &l.Price,
			&l.Ticket, &l.Profit, &l.Message, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	return logs, rows.Err()
}

func (r *TradeLogRepository) ListByAccountID(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*model.TradeLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, account_id, action, symbol, order_type, volume, price, ticket, profit, message, created_at
		FROM trade_logs WHERE account_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.TradeLog
	for rows.Next() {
		var l model.TradeLog
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.AccountID, &l.Action,
			&l.Symbol, &l.OrderType, &l.Volume, &l.Price,
			&l.Ticket, &l.Profit, &l.Message, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	return logs, rows.Err()
}

func (r *TradeLogRepository) ListByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time, limit, offset int) ([]*model.TradeLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, account_id, action, symbol, order_type, volume, price, ticket, profit, message, created_at
		FROM trade_logs WHERE user_id = $1 AND created_at >= $2 AND created_at <= $3 ORDER BY created_at DESC LIMIT $4 OFFSET $5`,
		userID, start, end, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.TradeLog
	for rows.Next() {
		var l model.TradeLog
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.AccountID, &l.Action,
			&l.Symbol, &l.OrderType, &l.Volume, &l.Price,
			&l.Ticket, &l.Profit, &l.Message, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	return logs, rows.Err()
}

func (r *TradeLogRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM trade_logs WHERE user_id = $1`
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}
