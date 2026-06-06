package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	StrategyCode    string     `db:"strategy_code"`
	Symbol          string     `db:"symbol"`
	Timeframe       string     `db:"timeframe"`
	FromTsUnixMs    int64      `db:"from_ts_unix_ms"`
	ToTsUnixMs      int64      `db:"to_ts_unix_ms"`
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
	// OOS validation fields (nil when window too short or not in top-K)
	OOSScore        *float64 `db:"oos_score"`
	OOSTotalReturn  *float64 `db:"oos_total_return"`
	OOSSharpeRatio  *float64 `db:"oos_sharpe_ratio"`
	DegradationPct  *float64 `db:"degradation_pct"`
	IsOverfit       bool     `db:"is_overfit"`
}

type StrategyExperimentRepository struct {
	db *pgxpool.Pool
}

func NewStrategyExperimentRepository(db *pgxpool.Pool) *StrategyExperimentRepository {
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
		exp.Status = "PENDING"
	}
	if exp.SearchMethod == "" {
		exp.SearchMethod = "grid"
	}
	if exp.MaxCandidates <= 0 {
		exp.MaxCandidates = 20
	}
	if len(exp.ParameterSpace) == 0 {
		exp.ParameterSpace = nil
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO strategy_experiments (id,user_id,base_template_id,status,parameter_space,search_method,max_candidates,objective,market_regime_ref,best_candidate_id,job_id,strategy_code,symbol,timeframe,from_ts_unix_ms,to_ts_unix_ms,created_at,finished_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
	`, exp.ID, exp.UserID, exp.BaseTemplateID, exp.Status, exp.ParameterSpace, exp.SearchMethod, exp.MaxCandidates, exp.Objective, exp.MarketRegimeRef, exp.BestCandidateID, exp.JobID, exp.StrategyCode, exp.Symbol, exp.Timeframe, exp.FromTsUnixMs, exp.ToTsUnixMs, exp.CreatedAt, exp.FinishedAt)
	if err != nil {
		return fmt.Errorf("create experiment: %w", err)
	}
	return nil
}

func (r *StrategyExperimentRepository) Get(ctx context.Context, userID, id uuid.UUID) (*StrategyExperiment, error) {
	var exp StrategyExperiment
	err := r.db.QueryRow(ctx, `SELECT id,user_id,base_template_id,status,parameter_space,search_method,max_candidates,objective,market_regime_ref,best_candidate_id,job_id,strategy_code,symbol,timeframe,from_ts_unix_ms,to_ts_unix_ms,created_at,finished_at FROM strategy_experiments WHERE id = $1 AND user_id = $2`, id, userID).Scan(
		&exp.ID, &exp.UserID, &exp.BaseTemplateID, &exp.Status, &exp.ParameterSpace, &exp.SearchMethod, &exp.MaxCandidates, &exp.Objective, &exp.MarketRegimeRef, &exp.BestCandidateID, &exp.JobID, &exp.StrategyCode, &exp.Symbol, &exp.Timeframe, &exp.FromTsUnixMs, &exp.ToTsUnixMs, &exp.CreatedAt, &exp.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
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
	rows, err := r.db.Query(ctx, `SELECT id,user_id,base_template_id,status,parameter_space,search_method,max_candidates,objective,market_regime_ref,best_candidate_id,job_id,strategy_code,symbol,timeframe,from_ts_unix_ms,to_ts_unix_ms,created_at,finished_at FROM strategy_experiments WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []StrategyExperiment
	for rows.Next() {
		var exp StrategyExperiment
		if err := rows.Scan(
			&exp.ID, &exp.UserID, &exp.BaseTemplateID, &exp.Status, &exp.ParameterSpace, &exp.SearchMethod, &exp.MaxCandidates, &exp.Objective, &exp.MarketRegimeRef, &exp.BestCandidateID, &exp.JobID, &exp.StrategyCode, &exp.Symbol, &exp.Timeframe, &exp.FromTsUnixMs, &exp.ToTsUnixMs, &exp.CreatedAt, &exp.FinishedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, exp)
	}
	return result, rows.Err()
}

func (r *StrategyExperimentRepository) Cancel(ctx context.Context, userID, id uuid.UUID) (*StrategyExperiment, error) {
	var exp StrategyExperiment
	err := r.db.QueryRow(ctx, `UPDATE strategy_experiments SET status = 'CANCELLED', finished_at = COALESCE(finished_at, now()) WHERE id = $1 AND user_id = $2 AND status IN ('QUEUED','RUNNING') RETURNING id,user_id,base_template_id,status,parameter_space,search_method,max_candidates,objective,market_regime_ref,best_candidate_id,job_id,strategy_code,symbol,timeframe,from_ts_unix_ms,to_ts_unix_ms,created_at,finished_at`, id, userID).Scan(
		&exp.ID, &exp.UserID, &exp.BaseTemplateID, &exp.Status, &exp.ParameterSpace, &exp.SearchMethod, &exp.MaxCandidates, &exp.Objective, &exp.MarketRegimeRef, &exp.BestCandidateID, &exp.JobID, &exp.StrategyCode, &exp.Symbol, &exp.Timeframe, &exp.FromTsUnixMs, &exp.ToTsUnixMs, &exp.CreatedAt, &exp.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
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
		candidate.Parameters = nil
	}
	if len(candidate.ScoreComponents) == 0 {
		candidate.ScoreComponents = nil
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO strategy_experiment_candidates (id,experiment_id,parameters,draft_code_ref,backtest_run_id,score,grade,score_components,rank,summary,recommendation,created_at,oos_score,oos_total_return,oos_sharpe_ratio,degradation_pct,is_overfit)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
	`, candidate.ID, candidate.ExperimentID, candidate.Parameters, candidate.DraftCodeRef, candidate.BacktestRunID, candidate.Score, candidate.Grade, candidate.ScoreComponents, candidate.Rank, candidate.Summary, candidate.Recommendation, candidate.CreatedAt, candidate.OOSScore, candidate.OOSTotalReturn, candidate.OOSSharpeRatio, candidate.DegradationPct, candidate.IsOverfit)
	if err != nil {
		return fmt.Errorf("create experiment candidate: %w", err)
	}
	return nil
}

func (r *StrategyExperimentRepository) ListCandidates(ctx context.Context, userID, experimentID uuid.UUID) ([]StrategyExperimentCandidate, error) {
	if _, err := r.Get(ctx, userID, experimentID); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT * FROM strategy_experiment_candidates WHERE experiment_id = $1 ORDER BY rank ASC, created_at ASC`, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []StrategyExperimentCandidate
	for rows.Next() {
		var c StrategyExperimentCandidate
		if err := rows.Scan(&c.ID, &c.ExperimentID, &c.Parameters, &c.DraftCodeRef, &c.BacktestRunID, &c.Score, &c.Grade, &c.ScoreComponents, &c.Rank, &c.Summary, &c.Recommendation, &c.CreatedAt, &c.OOSScore, &c.OOSTotalReturn, &c.OOSSharpeRatio, &c.DegradationPct, &c.IsOverfit); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *StrategyExperimentRepository) GetCandidate(ctx context.Context, userID, candidateID uuid.UUID) (*StrategyExperimentCandidate, error) {
	var row StrategyExperimentCandidate
	err := r.db.QueryRow(ctx, `
		SELECT c.* FROM strategy_experiment_candidates c
		JOIN strategy_experiments e ON e.id = c.experiment_id
		WHERE c.id = $1 AND e.user_id = $2
	`, candidateID, userID).Scan(&row.ID, &row.ExperimentID, &row.Parameters, &row.DraftCodeRef, &row.BacktestRunID, &row.Score, &row.Grade, &row.ScoreComponents, &row.Rank, &row.Summary, &row.Recommendation, &row.CreatedAt, &row.OOSScore, &row.OOSTotalReturn, &row.OOSSharpeRatio, &row.DegradationPct, &row.IsOverfit)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExperimentCandidateNotFound
	}
	return &row, err
}

// ClaimPendingExperiment atomically claims the oldest PENDING experiment using
// FOR UPDATE SKIP LOCKED to prevent concurrent workers from claiming the same row.
// Returns nil if no PENDING experiments exist.
func (r *StrategyExperimentRepository) ClaimPendingExperiment(ctx context.Context) (*StrategyExperiment, error) {
	var e StrategyExperiment
	err := r.db.QueryRow(ctx,
		`WITH candidate AS (
			SELECT id FROM strategy_experiments
			WHERE status = 'PENDING'
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE strategy_experiments s
		SET status = 'PROCESSING'
		FROM candidate c
		WHERE s.id = c.id
		RETURNING
			s.id, s.user_id, s.base_template_id, s.status, s.parameter_space, s.search_method,
			s.max_candidates, s.objective, s.market_regime_ref, s.best_candidate_id, s.job_id,
			s.strategy_code, s.symbol, s.timeframe, s.from_ts_unix_ms, s.to_ts_unix_ms, s.created_at, s.finished_at`,
	).Scan(
		&e.ID, &e.UserID, &e.BaseTemplateID, &e.Status, &e.ParameterSpace,
		&e.SearchMethod, &e.MaxCandidates, &e.Objective, &e.MarketRegimeRef,
		&e.BestCandidateID, &e.JobID, &e.StrategyCode, &e.Symbol, &e.Timeframe, &e.FromTsUnixMs, &e.ToTsUnixMs, &e.CreatedAt, &e.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("claim pending: %w", err)
	}
	return &e, nil
}

func (r *StrategyExperimentRepository) UpdateExperimentStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE strategy_experiments SET status = $2, finished_at = NOW() WHERE id = $1`, id, status)
	return err
}

// UpdateMarketRegime persists the detected market regime on the experiment.
func (r *StrategyExperimentRepository) UpdateMarketRegime(ctx context.Context, id uuid.UUID, regime string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE strategy_experiments SET market_regime_ref = $2 WHERE id = $1`,
		id, regime)
	return err
}

func (r *StrategyExperimentRepository) SetBestCandidate(ctx context.Context, experimentID, candidateID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE strategy_experiments SET best_candidate_id = $2 WHERE id = $1`, experimentID, candidateID)
	if err != nil {
		return fmt.Errorf("set best candidate: %w", err)
	}
	return nil
}
