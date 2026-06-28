package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CountActiveSchedules returns the number of active strategy schedules for a user.
func (r *AutoTradingRepository) CountActiveSchedules(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM strategy_schedules WHERE user_id = $1 AND is_active = true`
	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active schedules: %w", err)
	}
	return count, nil
}

// CountPendingExecutions returns the count of in-progress strategy executions for a user.
func (r *AutoTradingRepository) CountPendingExecutions(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM strategy_executions WHERE user_id = $1 AND status = 'running'`
	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending executions: %w", err)
	}
	return count, nil
}

// CountTodayExecutionsByUser returns today's execution count across all user accounts.
func (r *AutoTradingRepository) CountTodayExecutionsByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM strategy_executions WHERE user_id = $1 AND started_at >= CURRENT_DATE`
	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count today executions: %w", err)
	}
	return count, nil
}

// GetTodayProfitByUser returns today's profit sum across all user accounts.
func (r *AutoTradingRepository) GetTodayProfitByUser(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM((orders->>'profit')::numeric), 0)
		FROM strategy_executions
		WHERE user_id = $1 AND started_at >= CURRENT_DATE AND status = 'completed'`
	var profit decimal.Decimal
	err := r.db.QueryRow(ctx, query, userID).Scan(&profit)
	if err != nil {
		return decimal.Zero, fmt.Errorf("get today profit: %w", err)
	}
	return profit, nil
}
