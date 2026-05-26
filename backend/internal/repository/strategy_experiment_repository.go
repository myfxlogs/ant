package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrStrategyExperimentNotFound  = errors.New("strategy experiment not found")
	ErrExperimentCandidateNotFound = errors.New("strategy experiment candidate not found")
)

type StrategyExperiment struct {
	ID              uuid.UUID  `db:"id"`
	UserID          uuid.UUID  `db:"user_id"`
	BaseTemplateID  *uuid.UUID `db:"base_template_id"`
	Status          string     `db:"status"`
	ParameterSpace  []byte     `db:"parameter_space"`
	SearchMethod    string     `db:"search_method"`
	MaxCandidates   int        `db:"max_candidates"`
	Objective       string     `db:"objective"`
	MarketRegimeRef string     `db:"market_regime_ref"`
	BestCandidateID *uuid.UUID `db:"best_candidate_id"`
	JobID           *uuid.UUID `db:"job_id"`
	CreatedAt       time.Time  `db:"created_at"`
	FinishedAt      *time.Time `db:"finished_at"`
}

type StrategyExperimentCandidate struct {
	ID              uuid.UUID  `db:"id"`
	ExperimentID    uuid.UUID  `db:"experiment_id"`
	Parameters      []byte     `db:"parameters"`
	DraftCodeRef    string     `db:"draft_code_ref"`
	BacktestRunID   *uuid.UUID `db:"backtest_run_id"`
	Score           float64    `db:"score"`
	Grade           string     `db:"grade"`
	ScoreComponents []byte     `db:"score_components"`
	Rank            int        `db:"rank"`
	Summary         string     `db:"summary"`
	Recommendation  string     `db:"recommendation"`
	CreatedAt       time.Time  `db:"created_at"`
}

type StrategyExperimentRepository struct {
	db *sqlx.DB
}

func NewStrategyExperimentRepository(db *sqlx.DB) *StrategyExperimentRepository {
	return &StrategyExperimentRepository{db: db}
}

func (r *StrategyExperimentRepository) Create(ctx context.Context, exp *StrategyExperiment) error {
	if exp.ID == uuid.Nil {
		exp.ID = uuid.New()
	}
	if exp.CreatedAt.IsZero() {
		exp.CreatedAt = time.Now().UTC()
	}
	if exp.Status == "" {
		exp.Status = "SUCCEEDED"
	}
	if exp.SearchMethod == "" {
		exp.SearchMethod = "grid"
	}
	if exp.MaxCandidates <= 0 {
		exp.MaxCandidates = 20
	}
	if len(exp.ParameterSpace) == 0 {
		exp.ParameterSpace = []byte(`{}`)
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO strategy_experiments (id,user_id,base_template_id,status,parameter_space,search_method,max_candidates,objective,market_regime_ref,best_candidate_id,job_id,created_at,finished_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, exp.ID, exp.UserID, exp.BaseTemplateID, exp.Status, exp.ParameterSpace, exp.SearchMethod, exp.MaxCandidates, exp.Objective, exp.MarketRegimeRef, exp.BestCandidateID, exp.JobID, exp.CreatedAt, exp.FinishedAt)
	return fmt.Errorf("create experiment: %w", err)
}

func (r *StrategyExperimentRepository) Get(ctx context.Context, userID, id uuid.UUID) (*StrategyExperiment, error) {
	var exp StrategyExperiment
	err := r.db.GetContext(ctx, &exp, `SELECT * FROM strategy_experiments WHERE id = $1 AND user_id = $2`, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrStrategyExperimentNotFound
	}
	return &exp, err
}

func (r *StrategyExperimentRepository) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]StrategyExperiment, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var rows []StrategyExperiment
	err := r.db.SelectContext(ctx, &rows, `SELECT * FROM strategy_experiments WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	return rows, err
}

func (r *StrategyExperimentRepository) Cancel(ctx context.Context, userID, id uuid.UUID) (*StrategyExperiment, error) {
	var exp StrategyExperiment
	err := r.db.GetContext(ctx, &exp, `UPDATE strategy_experiments SET status = 'CANCELLED', finished_at = COALESCE(finished_at, now()) WHERE id = $1 AND user_id = $2 AND status IN ('QUEUED','RUNNING') RETURNING *`, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return r.Get(ctx, userID, id)
	}
	return &exp, err
}

func (r *StrategyExperimentRepository) CreateCandidate(ctx context.Context, candidate *StrategyExperimentCandidate) error {
	if candidate.ID == uuid.Nil {
		candidate.ID = uuid.New()
	}
	if candidate.CreatedAt.IsZero() {
		candidate.CreatedAt = time.Now().UTC()
	}
	if len(candidate.Parameters) == 0 {
		candidate.Parameters = []byte(`{}`)
	}
	if len(candidate.ScoreComponents) == 0 {
		candidate.ScoreComponents = []byte(`{}`)
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO strategy_experiment_candidates (id,experiment_id,parameters,draft_code_ref,backtest_run_id,score,grade,score_components,rank,summary,recommendation,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, candidate.ID, candidate.ExperimentID, candidate.Parameters, candidate.DraftCodeRef, candidate.BacktestRunID, candidate.Score, candidate.Grade, candidate.ScoreComponents, candidate.Rank, candidate.Summary, candidate.Recommendation, candidate.CreatedAt)
	return fmt.Errorf("create experiment candidate: %w", err)
}

func (r *StrategyExperimentRepository) ListCandidates(ctx context.Context, userID, experimentID uuid.UUID) ([]StrategyExperimentCandidate, error) {
	if _, err := r.Get(ctx, userID, experimentID); err != nil {
		return nil, err
	}
	var rows []StrategyExperimentCandidate
	err := r.db.SelectContext(ctx, &rows, `SELECT * FROM strategy_experiment_candidates WHERE experiment_id = $1 ORDER BY rank ASC, created_at ASC`, experimentID)
	return rows, err
}

func (r *StrategyExperimentRepository) GetCandidate(ctx context.Context, userID, candidateID uuid.UUID) (*StrategyExperimentCandidate, error) {
	var row StrategyExperimentCandidate
	err := r.db.GetContext(ctx, &row, `
		SELECT c.* FROM strategy_experiment_candidates c
		JOIN strategy_experiments e ON e.id = c.experiment_id
		WHERE c.id = $1 AND e.user_id = $2
	`, candidateID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrExperimentCandidateNotFound
	}
	return &row, err
}

func (r *StrategyExperimentRepository) SetBestCandidate(ctx context.Context, experimentID, candidateID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE strategy_experiments SET best_candidate_id = $2 WHERE id = $1`, experimentID, candidateID)
	if err != nil {
		return fmt.Errorf("set best candidate: %w", err)
	}
	return nil
}
