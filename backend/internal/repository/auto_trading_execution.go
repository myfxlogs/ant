package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"anttrader/internal/model"
)

func (r *AutoTradingRepository) CreateExecution(ctx context.Context, execution *model.StrategyExecution) error {
	query := `
			INSERT INTO strategy_executions (
				id, user_id, template_id, schedule_id, account_id, status,
				signals, orders, error_message, started_at, completed_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	if execution.ID == uuid.Nil {
		execution.ID = uuid.New()
	}
	execution.StartedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		execution.ID, execution.UserID, execution.TemplateID, execution.ScheduleID, execution.AccountID,
		execution.Status, execution.Signals, execution.Orders, execution.ErrorMessage,
		execution.StartedAt, execution.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("create execution: %w", err)
	}
	return nil
}

func (r *AutoTradingRepository) GetExecutionByID(ctx context.Context, id uuid.UUID) (*model.StrategyExecution, error) {
	query := `SELECT id, user_id, template_id, schedule_id, account_id, status, signals, orders, error_message, started_at, completed_at FROM strategy_executions WHERE id = $1`
	var execution model.StrategyExecution
	err := r.db.QueryRow(ctx, query, id).Scan(
		&execution.ID, &execution.UserID, &execution.TemplateID, &execution.ScheduleID, &execution.AccountID,
		&execution.Status, &execution.Signals, &execution.Orders, &execution.ErrorMessage,
		&execution.StartedAt, &execution.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrExecutionNotFound
		}
		return nil, err
	}
	return &execution, nil
}

func (r *AutoTradingRepository) GetExecutionsByTemplateID(ctx context.Context, templateID uuid.UUID, limit int) ([]*model.StrategyExecution, error) {
	query := `SELECT id, user_id, template_id, schedule_id, account_id, status, signals, orders, error_message, started_at, completed_at FROM strategy_executions WHERE template_id = $1 ORDER BY started_at DESC LIMIT $2`
	rows, err := r.db.Query(ctx, query, templateID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var executions []*model.StrategyExecution
	for rows.Next() {
		var e model.StrategyExecution
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.TemplateID, &e.ScheduleID, &e.AccountID,
			&e.Status, &e.Signals, &e.Orders, &e.ErrorMessage,
			&e.StartedAt, &e.CompletedAt,
		); err != nil {
			return nil, err
		}
		executions = append(executions, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return executions, nil
}

func (r *AutoTradingRepository) GetExecutionsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*model.StrategyExecution, error) {
	query := `SELECT id, user_id, template_id, schedule_id, account_id, status, signals, orders, error_message, started_at, completed_at FROM strategy_executions WHERE user_id = $1 ORDER BY started_at DESC LIMIT $2`
	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var executions []*model.StrategyExecution
	for rows.Next() {
		var e model.StrategyExecution
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.TemplateID, &e.ScheduleID, &e.AccountID,
			&e.Status, &e.Signals, &e.Orders, &e.ErrorMessage,
			&e.StartedAt, &e.CompletedAt,
		); err != nil {
			return nil, err
		}
		executions = append(executions, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return executions, nil
}

func (r *AutoTradingRepository) GetExecutionsByAccountID(ctx context.Context, accountID uuid.UUID, limit int) ([]*model.StrategyExecution, error) {
	query := `SELECT id, user_id, template_id, schedule_id, account_id, status, signals, orders, error_message, started_at, completed_at FROM strategy_executions WHERE account_id = $1 ORDER BY started_at DESC LIMIT $2`
	rows, err := r.db.Query(ctx, query, accountID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var executions []*model.StrategyExecution
	for rows.Next() {
		var e model.StrategyExecution
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.TemplateID, &e.ScheduleID, &e.AccountID,
			&e.Status, &e.Signals, &e.Orders, &e.ErrorMessage,
			&e.StartedAt, &e.CompletedAt,
		); err != nil {
			return nil, err
		}
		executions = append(executions, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return executions, nil
}

func (r *AutoTradingRepository) UpdateExecution(ctx context.Context, execution *model.StrategyExecution) error {
	query := `
			UPDATE strategy_executions SET
				status = $2, signals = $3, orders = $4, error_message = $5, completed_at = $6
			WHERE id = $1`

	_, err := r.db.Exec(ctx, query,
		execution.ID, execution.Status, execution.Signals, execution.Orders,
		execution.ErrorMessage, execution.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("update execution: %w", err)
	}
	return nil
}

func (r *AutoTradingRepository) GetTodayExecutionCount(ctx context.Context, accountID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM strategy_executions WHERE account_id = $1 AND started_at >= CURRENT_DATE`
	var count int
	err := r.db.QueryRow(ctx, query, accountID).Scan(&count)
	return count, err
}

func (r *AutoTradingRepository) GetTodayProfit(ctx context.Context, accountID uuid.UUID) (float64, error) {
	query := `
			SELECT COALESCE(SUM((orders->>'profit')::float), 0)
			FROM strategy_executions
			WHERE account_id = $1 AND started_at >= CURRENT_DATE AND status = 'completed'`
	var profit float64
	err := r.db.QueryRow(ctx, query, accountID).Scan(&profit)
	return profit, err
}
