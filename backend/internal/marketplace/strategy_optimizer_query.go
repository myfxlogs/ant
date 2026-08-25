package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ListOptimizationTasks returns optimization tasks for a publisher.
func (s *Service) ListOptimizationTasks(ctx context.Context, publisherID, status string, limit, offset int) ([]OptimizationTask, int, error) {
	pid, err := uuid.Parse(publisherID)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: list opt tasks: invalid publisher_id: %w", err)
	}
	if limit <= 0 {
		limit = 20
	}

	// Build query with optional status filter.
	statusFilter := ""
	args := []any{pid, limit, offset}
	if status != "" {
		statusFilter = " AND status = $4"
		args = []any{pid, limit, offset, status}
	}

	var total int
	countArgs := []any{pid}
	countQuery := `SELECT COUNT(*) FROM marketplace_strategy_optimization_tasks WHERE publisher_id = $1`
	if status != "" {
		countQuery += " AND status = $2"
		countArgs = []any{pid, status}
	}
	err = s.pg.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: list opt tasks: count: %w", err)
	}

	rows, err := s.pg.Query(ctx,
		`SELECT id::text, strategy_id::text, publisher_id::text, status, trigger_reason,
		        COALESCE(decay_metrics, ''::bytea), COALESCE(suggested_code, ''),
		        COALESCE(suggested_params, ''), COALESCE(backtest_snapshot, ''::bytea),
		        COALESCE(change_summary, ''), created_at, updated_at, completed_at,
		        published_version_id::text
		 FROM marketplace_strategy_optimization_tasks
		 WHERE publisher_id = $1`+statusFilter+`
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: list opt tasks: query: %w", err)
	}
	defer rows.Close()

	var tasks []OptimizationTask
	for rows.Next() {
		var t OptimizationTask
		var completedAt *time.Time
		var pubVerID *string
		if err := rows.Scan(&t.ID, &t.StrategyID, &t.PublisherID, &t.Status, &t.TriggerReason,
			&t.DecayMetrics, &t.SuggestedCode, &t.SuggestedParams, &t.BacktestSnapshot,
			&t.ChangeSummary, &t.CreatedAt, &t.UpdatedAt, &completedAt, &pubVerID); err != nil {
			return nil, 0, fmt.Errorf("marketplace: list opt tasks: scan: %w", err)
		}
		t.CompletedAt = completedAt
		t.PublishedVersionID = pubVerID
		tasks = append(tasks, t)
	}
	return tasks, total, rows.Err()
}

// GetOptimizationTask returns a single optimization task by ID.
func (s *Service) GetOptimizationTask(ctx context.Context, taskID, publisherID string) (*OptimizationTask, error) {
	tid, err := uuid.Parse(taskID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: get opt task: invalid task_id: %w", err)
	}
	pid, err := uuid.Parse(publisherID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: get opt task: invalid publisher_id: %w", err)
	}

	var t OptimizationTask
	var completedAt *time.Time
	var pubVerID *string
	err = s.pg.QueryRow(ctx,
		`SELECT id::text, strategy_id::text, publisher_id::text, status, trigger_reason,
		        COALESCE(decay_metrics, ''::bytea), COALESCE(suggested_code, ''),
		        COALESCE(suggested_params, ''), COALESCE(backtest_snapshot, ''::bytea),
		        COALESCE(change_summary, ''), created_at, updated_at, completed_at,
		        published_version_id::text
		 FROM marketplace_strategy_optimization_tasks
		 WHERE id = $1 AND publisher_id = $2`,
		tid, pid,
	).Scan(&t.ID, &t.StrategyID, &t.PublisherID, &t.Status, &t.TriggerReason,
		&t.DecayMetrics, &t.SuggestedCode, &t.SuggestedParams, &t.BacktestSnapshot,
		&t.ChangeSummary, &t.CreatedAt, &t.UpdatedAt, &completedAt, &pubVerID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: get opt task: %w", err)
	}
	t.CompletedAt = completedAt
	t.PublishedVersionID = pubVerID
	return &t, nil
}
