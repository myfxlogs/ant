package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GateEvaluation is the persisted result of a 7-gate pipeline run + marketplace quality preview.
type GateEvaluation struct {
	ID              uuid.UUID `db:"id"`
	BacktestRunID   uuid.UUID `db:"backtest_run_id"`
	UserID          uuid.UUID `db:"user_id"`
	GateResult      []byte    `db:"gate_result"`   // GateEvaluationUpdate (summary only)
	GateResults     []byte    `db:"gate_results"`  // GateResultList (individual gates)
	QualityPreview  []byte    `db:"quality_preview"`
	Passed          bool      `db:"passed"`
	FirstFail       string    `db:"first_fail"`
	Summary         string    `db:"summary"`
	Publishable     bool      `db:"publishable"`
	CreatedAt       time.Time `db:"created_at"`
}

type GateEvaluationRepository struct {
	db *pgxpool.Pool
}

func NewGateEvaluationRepository(db *pgxpool.Pool) *GateEvaluationRepository {
	return &GateEvaluationRepository{db: db}
}

// Upsert inserts or updates a gate evaluation for a backtest run.
func (r *GateEvaluationRepository) Upsert(ctx context.Context, userID, runID uuid.UUID, gateResult, gateResults, qualityPreview []byte, passed bool, firstFail, summary string, publishable bool) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO gate_evaluations (backtest_run_id, user_id, gate_result, gate_results, quality_preview, passed, first_fail, summary, publishable)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (backtest_run_id) DO UPDATE SET
			gate_result = EXCLUDED.gate_result,
			gate_results = EXCLUDED.gate_results,
			quality_preview = EXCLUDED.quality_preview,
			passed = EXCLUDED.passed,
			first_fail = EXCLUDED.first_fail,
			summary = EXCLUDED.summary,
			publishable = EXCLUDED.publishable,
			created_at = now()
	`, runID, userID, gateResult, gateResults, qualityPreview, passed, firstFail, summary, publishable)
	return err
}

// GetByRunID retrieves the gate evaluation for a backtest run, or nil if not found.
func (r *GateEvaluationRepository) GetByRunID(ctx context.Context, runID uuid.UUID) (*GateEvaluation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, backtest_run_id, user_id, gate_result, gate_results, quality_preview, passed, first_fail, summary, publishable, created_at
		FROM gate_evaluations WHERE backtest_run_id = $1
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	var ge GateEvaluation
	if err := rows.Scan(&ge.ID, &ge.BacktestRunID, &ge.UserID, &ge.GateResult, &ge.GateResults, &ge.QualityPreview, &ge.Passed, &ge.FirstFail, &ge.Summary, &ge.Publishable, &ge.CreatedAt); err != nil {
		return nil, err
	}
	return &ge, nil
}

// CountByUserAndTemplate counts completed backtest runs for a user+template (for DSR NumAttempts).
func (r *GateEvaluationRepository) CountBacktestAttempts(ctx context.Context, userID uuid.UUID, templateID *uuid.UUID) (int, error) {
	if templateID == nil {
		var count int
		err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM backtest_runs WHERE user_id = $1 AND status = 'SUCCEEDED'`, userID).Scan(&count)
		return count, err
	}
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM backtest_runs WHERE user_id = $1 AND template_id = $2 AND status = 'SUCCEEDED'`, userID, *templateID).Scan(&count)
	return count, err
}

// ExistingSignal represents a signal from a live strategy run for correlation gate.
type ExistingSignal struct {
	StrategyID string
	Timestamp  int64
	Direction  float64
}

// GetExistingSignals retrieves signals from the user's running live strategies for correlation gate.
// Returns signals grouped by strategy_id.
func (r *GateEvaluationRepository) GetExistingSignals(ctx context.Context, userID uuid.UUID) (map[string][]ExistingSignal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(ss.strategy_id::text, ''), ss.signal_type, EXTRACT(EPOCH FROM ss.created_at)::bigint * 1000
		FROM strategy_signals ss
		JOIN strategy_runs sr ON ss.run_id = sr.id
		WHERE sr.user_id = $1 AND sr.status = 'running' AND ss.signal_type IN ('buy', 'sell')
		ORDER BY ss.created_at
		LIMIT 500
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string][]ExistingSignal)
	for rows.Next() {
		var sid, sigType string
		var ts int64
		if err := rows.Scan(&sid, &sigType, &ts); err != nil {
			return nil, err
		}
		dir := 1.0
		if sigType == "sell" {
			dir = -1.0
		}
		result[sid] = append(result[sid], ExistingSignal{StrategyID: sid, Timestamp: ts, Direction: dir})
	}
	return result, nil
}
