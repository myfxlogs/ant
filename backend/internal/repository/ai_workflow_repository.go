package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAIWorkflowRunNotFound = errors.New("ai workflow run not found")

type AIWorkflowRun struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	Title     string    `db:"title"`
	Status    string    `db:"status"`
	Context   string    `db:"context_json"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	StepCount int       `db:"step_count"`
}

type AIWorkflowStep struct {
	ID        uuid.UUID `db:"id"`
	RunID     uuid.UUID `db:"run_id"`
	Key       string    `db:"step_key"`
	Title     string    `db:"title"`
	Status    string    `db:"status"`
	Input     string    `db:"input"`
	Output    string    `db:"output"`
	Error     string    `db:"error"`
	Duration  int64     `db:"duration_ms"`
	CreatedAt time.Time `db:"created_at"`
}

type AIWorkflowRepository struct {
	db *pgxpool.Pool
}

func NewAIWorkflowRepository(db *pgxpool.Pool) *AIWorkflowRepository {
	return &AIWorkflowRepository{db: db}
}

func (r *AIWorkflowRepository) CreateRun(ctx context.Context, userID uuid.UUID, title, contextJSON string) (*AIWorkflowRun, error) {
	if title == "" {
		title = "AI 工作流"
	}
	now := time.Now()
	run := &AIWorkflowRun{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     title,
		Status:    "running",
		Context:   contextJSON,
		CreatedAt: now,
		UpdatedAt: now,
		StepCount: 0,
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO ai_workflow_runs (id, user_id, title, status, context_json, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		run.ID, run.UserID, run.Title, run.Status, run.Context, run.CreatedAt, run.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (r *AIWorkflowRepository) AppendStep(ctx context.Context, userID, runID uuid.UUID, key, title, status, input, output, stepErr string, durationMs int64) (*AIWorkflowStep, error) {
	if status == "" {
		status = "done"
	}
	step := &AIWorkflowStep{
		ID:        uuid.New(),
		RunID:     runID,
		Key:       key,
		Title:     title,
		Status:    status,
		Input:     input,
		Output:    output,
		Error:     stepErr,
		Duration:  durationMs,
		CreatedAt: time.Now(),
	}

	tx, txErr := r.db.Begin(ctx)
	if txErr != nil {
		return nil, txErr
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// authorize run ownership inside transaction
	var owner uuid.UUID
	err := tx.QueryRow(ctx, `SELECT user_id FROM ai_workflow_runs WHERE id=$1`, runID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = tx.Rollback(ctx)
		return nil, ErrAIWorkflowRunNotFound
	}
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	if owner != userID {
		_ = tx.Rollback(ctx)
		return nil, ErrAIWorkflowRunNotFound
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO ai_workflow_steps (id, run_id, step_key, title, status, input, output, error, duration_ms, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		step.ID, step.RunID, step.Key, step.Title, step.Status, step.Input, step.Output, step.Error, step.Duration, step.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	newRunStatus := "running"
	if status == "error" {
		newRunStatus = "failed"
	}
	// Mark the run succeeded when a terminal step completes.
	// Terminal keys include "code", "final", "complete", "done", "review", or any key
	// prefixed with "final_" (e.g. "final_review").
	if status == "done" && isTerminalStepKey(key) {
		newRunStatus = "succeeded"
	}
	_, err = tx.Exec(ctx,
		`UPDATE ai_workflow_runs
			 SET updated_at = $1,
			     status = CASE WHEN status = 'running' THEN $2 ELSE status END
			 WHERE id = $3 AND user_id = $4`,
		now, newRunStatus, runID, userID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return step, nil
}

func (r *AIWorkflowRepository) ListRuns(ctx context.Context, userID uuid.UUID, limit, offset int) ([]AIWorkflowRun, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.db.Query(ctx,
		`SELECT r.id, r.user_id, r.title, r.status, r.context_json, r.created_at, r.updated_at,
			        COALESCE(s.cnt, 0) AS step_count
			 FROM ai_workflow_runs r
			 LEFT JOIN (SELECT run_id, COUNT(*) AS cnt FROM ai_workflow_steps GROUP BY run_id) s
			   ON s.run_id = r.id
			 WHERE r.user_id = $1
			 ORDER BY r.updated_at DESC
			 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []AIWorkflowRun
	for rows.Next() {
		var run AIWorkflowRun
		if err := rows.Scan(&run.ID, &run.UserID, &run.Title, &run.Status, &run.Context, &run.CreatedAt, &run.UpdatedAt, &run.StepCount); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return runs, nil
}

func (r *AIWorkflowRepository) GetRun(ctx context.Context, userID, runID uuid.UUID) (*AIWorkflowRun, []AIWorkflowStep, error) {
	var run AIWorkflowRun
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, title, status, context_json, created_at, updated_at
			 FROM ai_workflow_runs WHERE id = $1 AND user_id = $2`,
		runID, userID,
	).Scan(&run.ID, &run.UserID, &run.Title, &run.Status, &run.Context, &run.CreatedAt, &run.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrAIWorkflowRunNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, run_id, step_key, title, status, input, output, error, duration_ms, created_at
			 FROM ai_workflow_steps WHERE run_id = $1 ORDER BY created_at ASC`,
		runID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var steps []AIWorkflowStep
	for rows.Next() {
		var s AIWorkflowStep
		if err := rows.Scan(&s.ID, &s.RunID, &s.Key, &s.Title, &s.Status, &s.Input, &s.Output, &s.Error, &s.Duration, &s.CreatedAt); err != nil {
			return nil, nil, err
		}
		steps = append(steps, s)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	run.StepCount = len(steps)
	return &run, steps, nil
}

// isTerminalStepKey returns true for step keys that mark the end of a workflow.
// This avoids the previous hardcoded "key == 'code'" assumption that blocked
// workflows whose final step was named differently.
func isTerminalStepKey(key string) bool {
	switch key {
	case "code", "final", "complete", "done", "review", "report", "deploy", "summarize":
		return true
	default:
		return strings.HasPrefix(key, "final_")
	}
}
