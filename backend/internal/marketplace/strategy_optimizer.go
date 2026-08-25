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
