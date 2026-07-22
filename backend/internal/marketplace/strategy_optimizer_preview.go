// Package marketplace — Phase 5.1: Optimization preview and publish helpers.
package marketplace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PreviewOptimizationResult contains the comparison data for an optimization task.
type PreviewOptimizationResult struct {
	Task              *OptimizationTask
	OriginalBacktest  []byte // proto-serialized BacktestSnapshot from the original strategy
	OptimizedBacktest []byte // proto-serialized BacktestSnapshot from the AI optimization
	DecayMetrics      []byte // proto-serialized DecayMetrics
	ChangeSummary     string
	SuggestedCode     string
}

// PreviewOptimization returns a side-by-side comparison of the original vs optimized
// strategy backtest, plus decay metrics and suggested code preview.
// The task must be in 'completed' state and owned by the requesting publisher.
func (s *Service) PreviewOptimization(ctx context.Context, taskID, publisherID string) (*PreviewOptimizationResult, error) {
	tid, err := uuid.Parse(taskID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: preview opt: invalid task_id: %w", err)
	}
	pid, err := uuid.Parse(publisherID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: preview opt: invalid publisher_id: %w", err)
	}

	// Fetch the optimization task.
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
		return nil, fmt.Errorf("marketplace: preview opt: %w", err)
	}
	t.CompletedAt = completedAt
	t.PublishedVersionID = pubVerID

	if t.Status != "completed" {
		return nil, fmt.Errorf("marketplace: preview opt: task must be in completed state, current: %s", t.Status)
	}

	// Fetch the original strategy's backtest snapshot.
	var originalSnapshot []byte
	sid, _ := uuid.Parse(t.StrategyID)
	err = s.pg.QueryRow(ctx,
		`SELECT backtest_snapshot FROM marketplace_strategies WHERE strategy_id = $1`,
		sid,
	).Scan(&originalSnapshot)
	if err != nil {
		originalSnapshot = nil // strategy may not have a snapshot
	}

	return &PreviewOptimizationResult{
		Task:              &t,
		OriginalBacktest:  originalSnapshot,
		OptimizedBacktest: t.BacktestSnapshot,
		DecayMetrics:      t.DecayMetrics,
		ChangeSummary:     t.ChangeSummary,
		SuggestedCode:     t.SuggestedCode,
	}, nil
}

// notifyVersionUpdate notifies all active subscribers that a strategy has a new version.
func (s *Service) notifyVersionUpdate(ctx context.Context, strategyID uuid.UUID, changeSummary string) {
	if s.notifSender == nil {
		return
	}

	var title string
	_ = s.pg.QueryRow(ctx, `SELECT COALESCE(title,'') FROM marketplace_strategies WHERE strategy_id = $1`, strategyID).Scan(&title)

	rows, err := s.pg.Query(ctx,
		`SELECT subscriber_user_id FROM user_subscriptions
		 WHERE target_strategy_id = $1 AND active = true`,
		strategyID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			continue
		}
		_, _ = s.notifSender.Send(ctx, uid, "strategy_version_update",
			"Strategy Updated",
			fmt.Sprintf("\"%s\" has been updated. %s", title, changeSummary),
			nil)
	}
}

// hashCodeStr computes a SHA-256 hex hash of source code.
// Named hashCodeStr to avoid collision with repository.hashCode.
func hashCodeStr(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
