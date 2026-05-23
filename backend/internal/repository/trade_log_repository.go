package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"anttrader/internal/model"
)

type TradeLogRepository struct {
	db *sqlx.DB
}

func NewTradeLogRepository(db *sqlx.DB) *TradeLogRepository {
	return &TradeLogRepository{db: db}
}

func (r *TradeLogRepository) Create(ctx context.Context, log *model.TradeLog) error {
	query := `
		INSERT INTO trade_logs (id, user_id, account_id, action, symbol, order_type, volume, price, ticket, profit, message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.ExecContext(ctx, query,
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
	return err
}

func (r *TradeLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.TradeLog, error) {
	var log model.TradeLog
	query := `SELECT * FROM trade_logs WHERE id = $1`
	err := r.db.GetContext(ctx, &log, query, id)
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *TradeLogRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.TradeLog, error) {
	var logs []*model.TradeLog
	query := `SELECT * FROM trade_logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	err := r.db.SelectContext(ctx, &logs, query, userID, limit, offset)
	return logs, err
}

func (r *TradeLogRepository) ListByAccountID(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*model.TradeLog, error) {
	var logs []*model.TradeLog
	query := `SELECT * FROM trade_logs WHERE account_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	err := r.db.SelectContext(ctx, &logs, query, accountID, limit, offset)
	return logs, err
}

func (r *TradeLogRepository) ListByDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time, limit, offset int) ([]*model.TradeLog, error) {
	var logs []*model.TradeLog
	query := `SELECT * FROM trade_logs WHERE user_id = $1 AND created_at >= $2 AND created_at <= $3 ORDER BY created_at DESC LIMIT $4 OFFSET $5`
	err := r.db.SelectContext(ctx, &logs, query, userID, start, end, limit, offset)
	return logs, err
}

func (r *TradeLogRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM trade_logs WHERE user_id = $1`
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}
