package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UpdateOptimizationTaskResult stores the AI-generated optimization result.
// Called by the AI generator after it produces optimized code + backtest.
func (s *Service) UpdateOptimizationTaskResult(ctx context.Context, taskID, suggestedCode, suggestedParams, changeSummary string, backtestSnapshot []byte) error {
	tid, err := uuid.Parse(taskID)
	if err != nil {
		return fmt.Errorf("marketplace: update opt task: invalid task_id: %w", err)
	}
	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_strategy_optimization_tasks
		 SET status = 'completed', suggested_code = $2, suggested_params = $3,
		     change_summary = $4, backtest_snapshot = $5,
		     completed_at = now(), updated_at = now()
		 WHERE id = $1 AND status IN ('pending', 'generating')`,
		tid, suggestedCode, suggestedParams, changeSummary, backtestSnapshot,
	)
	if err != nil {
		return fmt.Errorf("marketplace: update opt task: %w", err)
	}
	return nil
}

// RejectOptimizationTask marks a task as rejected by the publisher.
func (s *Service) RejectOptimizationTask(ctx context.Context, taskID, publisherID string) error {
	tid, err := uuid.Parse(taskID)
	if err != nil {
		return fmt.Errorf("marketplace: reject opt task: invalid task_id: %w", err)
	}
	pid, err := uuid.Parse(publisherID)
	if err != nil {
		return fmt.Errorf("marketplace: reject opt task: invalid publisher_id: %w", err)
	}
	tag, err := s.pg.Exec(ctx,
		`UPDATE marketplace_strategy_optimization_tasks
		 SET status = 'rejected', updated_at = now()
		 WHERE id = $1 AND publisher_id = $2 AND status = 'completed'`,
		tid, pid,
	)
	if err != nil {
		return fmt.Errorf("marketplace: reject opt task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("marketplace: reject opt task: task not found or not in completed state")
	}
	return nil
}

// PublishOptimization publishes the optimized version as a new strategy version.
// It updates the strategy's source code and creates a version snapshot.
// The publisher must own the task and the task must be in 'completed' state.
// After publishing, all active subscribers are notified about the new version.
func (s *Service) PublishOptimization(ctx context.Context, taskID, publisherID string) (string, error) {
	tid, err := uuid.Parse(taskID)
	if err != nil {
		return "", fmt.Errorf("marketplace: publish opt: invalid task_id: %w", err)
	}
	pid, err := uuid.Parse(publisherID)
	if err != nil {
		return "", fmt.Errorf("marketplace: publish opt: invalid publisher_id: %w", err)
	}

	// Fetch the task and verify ownership.
	var t OptimizationTask
	var completedAt *time.Time
	var pubVerID *string
	err = s.pg.QueryRow(ctx,
		`SELECT id::text, strategy_id::text, publisher_id::text, status,
		        COALESCE(suggested_code, ''), COALESCE(change_summary, ''),
		        COALESCE(backtest_snapshot, NULL), completed_at, published_version_id::text
		 FROM marketplace_strategy_optimization_tasks
		 WHERE id = $1 AND publisher_id = $2`,
		tid, pid,
	).Scan(&t.ID, &t.StrategyID, &t.PublisherID, &t.Status,
		&t.SuggestedCode, &t.ChangeSummary, &t.BacktestSnapshot,
		&completedAt, &pubVerID)
	if err != nil {
		return "", fmt.Errorf("marketplace: publish opt: task not found: %w", err)
	}
	if t.Status != "completed" {
		return "", fmt.Errorf("marketplace: publish opt: task must be in completed state, current: %s", t.Status)
	}
	if t.SuggestedCode == "" {
		return "", fmt.Errorf("marketplace: publish opt: no suggested code available")
	}

	// Update the strategy source code via the version repository.
	// This creates a new version snapshot atomically.
	sid, err := uuid.Parse(t.StrategyID)
	if err != nil {
		return "", fmt.Errorf("marketplace: publish opt: invalid strategy_id: %w", err)
	}

	// Use a transaction to update code + mark task as published.
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("marketplace: publish opt: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Update source code in imported_strategies.
	var sourceLang string
	err = tx.QueryRow(ctx,
		`UPDATE imported_strategies SET source_code = $2, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1 RETURNING source_lang`,
		sid, t.SuggestedCode,
	).Scan(&sourceLang)
	if err != nil {
		return "", fmt.Errorf("marketplace: publish opt: update strategy code: %w", err)
	}

	// Create version snapshot.
	versionID := uuid.New()
	_, err = tx.Exec(ctx,
		`INSERT INTO strategy_versions (id, strategy_id, user_id, version_number, source_code, source_lang, change_summary, code_hash)
		 VALUES ($1, $2, $3, COALESCE((SELECT MAX(version_number) FROM strategy_versions WHERE strategy_id = $2), 0) + 1,
		         $4, $5, $6, $7)`,
		versionID, sid, pid, t.SuggestedCode, sourceLang, t.ChangeSummary,
		hashCodeStr(t.SuggestedCode),
	)
	if err != nil {
		return "", fmt.Errorf("marketplace: publish opt: create version: %w", err)
	}

	// Update backtest snapshot on the marketplace strategy if available.
	if len(t.BacktestSnapshot) > 0 {
		_, err = tx.Exec(ctx,
			`UPDATE marketplace_strategies SET backtest_snapshot = $2, updated_at = now()
			 WHERE strategy_id = $1`,
			sid, t.BacktestSnapshot,
		)
		if err != nil {
			return "", fmt.Errorf("marketplace: publish opt: update backtest snapshot: %w", err)
		}
	}

	// Mark task as published.
	_, err = tx.Exec(ctx,
		`UPDATE marketplace_strategy_optimization_tasks
		 SET status = 'published', published_version_id = $2, updated_at = now()
		 WHERE id = $1`,
		tid, versionID,
	)
	if err != nil {
		return "", fmt.Errorf("marketplace: publish opt: mark published: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("marketplace: publish opt: commit: %w", err)
	}

	// Notify all active subscribers about the new version.
	go s.notifyVersionUpdate(context.WithoutCancel(ctx), sid, t.ChangeSummary)

	return versionID.String(), nil
}
