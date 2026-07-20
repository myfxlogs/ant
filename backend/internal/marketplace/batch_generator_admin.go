package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ListTasks returns tasks filtered by status with pagination.
func (b *BatchGenerator) ListTasks(ctx context.Context, status string, limit, offset int) ([]AutoGenTask, error) {
	rows, err := b.pg.Query(ctx,
		`SELECT id, symbol, timeframe, strategy_type, risk_level, status,
		        strategy_id, quality_passed, error_message, source_code, created_at, started_at, finished_at
		 FROM auto_generation_tasks
		 WHERE ($1 = '' OR status = $1)
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("batch: list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []AutoGenTask
	for rows.Next() {
		var t AutoGenTask
		var strategyID *uuid.UUID
		var qualityPassed *bool
		var errMsg *string
		var sourceCode *string
		var startedAt, finishedAt *time.Time
		if err := rows.Scan(&t.ID, &t.Symbol, &t.Timeframe, &t.StrategyType, &t.RiskLevel, &t.Status,
			&strategyID, &qualityPassed, &errMsg, &sourceCode, &t.CreatedAt, &startedAt, &finishedAt); err != nil {
			return nil, fmt.Errorf("batch: scan task: %w", err)
		}
		t.StrategyID = strategyID
		t.QualityPassed = qualityPassed
		t.ErrorMessage = errMsg
		if sourceCode != nil {
			t.SourceCode = *sourceCode
		}
		t.StartedAt = startedAt
		t.FinishedAt = finishedAt
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// ApproveTask creates a strategy_templates row from the generated source code,
// then publishes it to the marketplace. The strategy_templates.id becomes the
// StrategyID — this satisfies the FK constraint from marketplace_strategies.
func (b *BatchGenerator) ApproveTask(ctx context.Context, taskID uuid.UUID, pub TaskPublisher) error {
	var symbol, timeframe, strategyType, riskLevel, sourceCode string
	var snapshot []byte
	err := b.pg.QueryRow(ctx,
		`SELECT symbol, timeframe, strategy_type, risk_level, source_code, result_backtest_snapshot
		 FROM auto_generation_tasks WHERE id=$1 AND status='awaiting_review'`,
		taskID).Scan(&symbol, &timeframe, &strategyType, &riskLevel, &sourceCode, &snapshot)
	if err != nil {
		return fmt.Errorf("batch: approve: task not found or not in review: %w", err)
	}
	if sourceCode == "" {
		return fmt.Errorf("batch: approve: source code is empty")
	}

	// 1. Persist source code into strategy_templates (user_id=NULL for system-generated).
	var templateID uuid.UUID
	err = b.pg.QueryRow(ctx,
		`INSERT INTO strategy_templates (user_id, name, description, code, is_public, is_system, tags, use_count)
		 VALUES (NULL, $1, $2, $3, true, true, '{}', 0)
		 RETURNING id`,
		fmt.Sprintf("%s %s %s", strategyType, symbol, timeframe),
		fmt.Sprintf("AI-generated %s strategy for %s on %s", strategyType, symbol, timeframe),
		sourceCode,
	).Scan(&templateID)
	if err != nil {
		return fmt.Errorf("batch: approve: create strategy_template: %w", err)
	}

	stratID := templateID.String()

	// 2. Publish to marketplace.
	if pub != nil {
		_, err = pub.Publish(ctx, PublishParams{
			StrategyID:            stratID,
			UserID:                uuid.Nil.String(), // system-generated
			Title:                 fmt.Sprintf("%s %s %s", strategyType, symbol, timeframe),
			Description:           fmt.Sprintf("AI-generated %s strategy for %s on %s", strategyType, symbol, timeframe),
			PriceModel:            PriceModelFree,
			PriceAmount:           "0",
			AssetClass:            "forex",
			Symbols:               []string{symbol},
			Timeframe:             timeframe,
			RiskLevel:             riskLevel,
			BacktestSnapshotProto: snapshot,
		})
		if err != nil {
			return fmt.Errorf("batch: approve: publish failed: %w", err)
		}
	}

	// 3. Mark task as published.
	_, err = b.pg.Exec(ctx,
		`UPDATE auto_generation_tasks
		 SET status='published', strategy_id=$2, finished_at=now()
		 WHERE id=$1`,
		taskID, templateID)
	if err != nil {
		return fmt.Errorf("batch: approve: update task: %w", err)
	}
	return nil
}

// RejectTask marks a task as rejected.
func (b *BatchGenerator) RejectTask(ctx context.Context, taskID uuid.UUID, reason string) error {
	_, err := b.pg.Exec(ctx,
		`UPDATE auto_generation_tasks
		 SET status='rejected', error_message=$2, finished_at=now()
		 WHERE id=$1 AND status='awaiting_review'`,
		taskID, reason)
	if err != nil {
		return fmt.Errorf("batch: reject: %w", err)
	}
	return nil
}
