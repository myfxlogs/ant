// Package marketplace — Phase 5.1b: AI strategy optimization.
//
// When decay is detected, CreateOptimizationTask enqueues a task for the
// strategy publisher. The AI generates an optimized version, runs a backtest,
// and stores the suggestion. The publisher can review and publish it via
// PublishOptimization, which creates a new strategy version.
package marketplace

import (
	"context"
	"fmt"
	"time"

	antv1 "alphaforge/gen/proto/ant/v1"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// OptimizationTask represents a strategy optimization task row.
type OptimizationTask struct {
	ID                 string
	StrategyID         string
	PublisherID        string
	Status             string // pending | generating | completed | rejected | published
	TriggerReason      string
	DecayMetrics       []byte // proto-serialized DecayMetrics
	SuggestedCode      string
	SuggestedParams    string
	BacktestSnapshot   []byte
	ChangeSummary      string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CompletedAt        *time.Time
	PublishedVersionID *string
}

// CreateOptimizationTask creates a new optimization task for a strategy.
// Called when decay is detected or when a publisher manually requests optimization.
func (s *Service) CreateOptimizationTask(ctx context.Context, strategyID, publisherID, triggerReason string, decayResult *DecayResult) (string, error) {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return "", fmt.Errorf("marketplace: create opt task: invalid strategy_id: %w", err)
	}
	pid, err := uuid.Parse(publisherID)
	if err != nil {
		return "", fmt.Errorf("marketplace: create opt task: invalid publisher_id: %w", err)
	}

	// Verify ownership and fetch title in a single query.
	var ownerID uuid.UUID
	var title string
	err = s.pg.QueryRow(ctx,
		`SELECT publisher_id, COALESCE(title, '') FROM marketplace_strategies WHERE strategy_id = $1`,
		sid,
	).Scan(&ownerID, &title)
	if err != nil {
		return "", fmt.Errorf("marketplace: create opt task: strategy not found: %w", err)
	}
	if ownerID != pid {
		return "", fmt.Errorf("marketplace: create opt task: not the strategy owner")
	}

	// Serialize decay metrics via proto (no JSON).
	var decayMetricsBytes []byte
	if decayResult != nil {
		dm := &antv1.DecayMetrics{
			DecayScore:        decayResult.DecayScore,
			SharpeDeclinePct:  decayResult.SharpeDeclinePct.String(),
			WinrateDeclinePct: decayResult.WinRateDeclinePct.String(),
			ReturnDelta:       decayResult.ReturnDelta.String(),
			TriggerReason:     decayResult.TriggerReason,
		}
		var err2 error
		decayMetricsBytes, err2 = proto.Marshal(dm)
		if err2 != nil {
			s.log.Warn("create opt task: proto marshal decay metrics", zap.Error(err2))
		}
	}

	id := uuid.New()
	_, err = s.pg.Exec(ctx,
		`INSERT INTO marketplace_strategy_optimization_tasks
		   (id, strategy_id, publisher_id, status, trigger_reason, decay_metrics)
		 VALUES ($1, $2, $3, 'pending', $4, $5)`,
		id, sid, pid, triggerReason, decayMetricsBytes,
	)
	if err != nil {
		return "", fmt.Errorf("marketplace: create opt task: %w", err)
	}

	// Notify the publisher about the optimization suggestion.
	if s.notifSender != nil {
		go func() {
			_, _ = s.notifSender.Send(context.WithoutCancel(ctx), pid, "optimization_suggested",
				"Strategy Optimization Suggested",
				fmt.Sprintf("We detected potential alpha decay in \"%s\". An optimization task has been created.", title),
				nil)
		}()
	}

	// FEAT-5: No auto-start of AI generation. Author must explicitly call
	// InitiateStrategyIteration RPC to trigger credit-billed AI optimization.
	// Manual code edits remain free via the existing version update path.

	return id.String(), nil
}

// InitiateStrategyIteration is the author-initiated entry point for AI iteration.
// It creates an optimization task (if one doesn't already exist for this strategy
// in pending/completed state), then starts credit-billed AI generation.
// The author's credits are billed via PreHold/Settle through the credit service.
func (s *Service) InitiateStrategyIteration(ctx context.Context, strategyID, publisherID string) (string, error) {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return "", fmt.Errorf("marketplace: initiate iteration: invalid strategy_id: %w", err)
	}
	pid, err := uuid.Parse(publisherID)
	if err != nil {
		return "", fmt.Errorf("marketplace: initiate iteration: invalid publisher_id: %w", err)
	}

	// Verify ownership.
	var ownerID uuid.UUID
	err = s.pg.QueryRow(ctx,
		`SELECT publisher_id FROM marketplace_strategies WHERE strategy_id = $1`,
		sid,
	).Scan(&ownerID)
	if err != nil {
		return "", fmt.Errorf("marketplace: initiate iteration: strategy not found: %w", err)
	}
	if ownerID != pid {
		return "", fmt.Errorf("marketplace: initiate iteration: not the strategy owner")
	}

	// Check for existing pending task for this strategy to avoid duplicates.
	var existingTaskID string
	_ = s.pg.QueryRow(ctx,
		`SELECT id::text FROM marketplace_strategy_optimization_tasks
		 WHERE strategy_id = $1 AND publisher_id = $2 AND status IN ('pending', 'generating')
		 ORDER BY created_at DESC LIMIT 1`,
		sid, pid,
	).Scan(&existingTaskID)

	var taskID string
	if existingTaskID != "" {
		taskID = existingTaskID
	} else {
		// Create a new task with trigger_reason "author_initiated".
		id := uuid.New()
		_, err = s.pg.Exec(ctx,
			`INSERT INTO marketplace_strategy_optimization_tasks
			   (id, strategy_id, publisher_id, status, trigger_reason)
			 VALUES ($1, $2, $3, 'pending', 'author_initiated')`,
			id, sid, pid,
		)
		if err != nil {
			return "", fmt.Errorf("marketplace: initiate iteration: create task: %w", err)
		}
		taskID = id.String()
	}

	// Start credit-billed AI generation in background.
	tid, _ := uuid.Parse(taskID)
	go s.runOptimizationGeneration(context.WithoutCancel(ctx), tid, pid)

	return taskID, nil
}

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
