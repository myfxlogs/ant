package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
	db *sqlx.DB
}

func NewAIWorkflowRepository(db *sqlx.DB) *AIWorkflowRepository {
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
	_, err := r.db.ExecContext(ctx,
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
	// authorize run ownership
	var exists int
	err := r.db.GetContext(ctx, &exists, `SELECT 1 FROM ai_workflow_runs WHERE id = $1 AND user_id = $2`, runID, userID)
	if err != nil {
		return nil, ErrAIWorkflowRunNotFound
	}

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

	tx, txErr := r.db.BeginTxx(ctx, nil)
	if txErr != nil {
		return nil, txErr
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
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
	if status == "done" && key == "code" {
		newRunStatus = "succeeded"
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE ai_workflow_runs
		 SET updated_at = $1,
		     status = CASE WHEN status = 'running' THEN $2 ELSE status END
		 WHERE id = $3 AND user_id = $4`,
		now, newRunStatus, runID, userID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
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

	var runs []AIWorkflowRun
	err := r.db.SelectContext(ctx, &runs,
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
	return runs, nil
}

func (r *AIWorkflowRepository) GetRun(ctx context.Context, userID, runID uuid.UUID) (*AIWorkflowRun, []AIWorkflowStep, error) {
	var run AIWorkflowRun
	err := r.db.GetContext(ctx, &run,
		`SELECT id, user_id, title, status, context_json, created_at, updated_at
		 FROM ai_workflow_runs WHERE id = $1 AND user_id = $2`,
		runID, userID,
	)
	if err != nil {
		return nil, nil, ErrAIWorkflowRunNotFound
	}

	var steps []AIWorkflowStep
	err = r.db.SelectContext(ctx, &steps,
		`SELECT id, run_id, step_key, title, status, input, output, error, duration_ms, created_at
		 FROM ai_workflow_steps WHERE run_id = $1 ORDER BY created_at ASC`,
		runID,
	)
	if err != nil {
		return nil, nil, err
	}

	run.StepCount = len(steps)
	return &run, steps, nil
}
