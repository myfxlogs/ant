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

var ErrImportedStrategyNotFound = errors.New("imported strategy not found")

// ImportedStrategy is a user-imported MQL strategy stored as raw source.
// source_code is the single source of truth for execution.
type ImportedStrategy struct {
	ID            uuid.UUID `db:"id"`
	UserID        uuid.UUID `db:"user_id"`
	Name          string    `db:"name"`
	SourceLang    string    `db:"source_lang"`
	SourceCode    string    `db:"source_code"`
	Params        []byte    `db:"params"`
	CoverageScore float64   `db:"coverage_score"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type ImportedStrategyRepository struct{ db *pgxpool.Pool }

func NewImportedStrategyRepository(db *pgxpool.Pool) *ImportedStrategyRepository {
	return &ImportedStrategyRepository{db: db}
}

func (r *ImportedStrategyRepository) Create(ctx context.Context, row *ImportedStrategy) error {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	now := time.Now().UTC()
	row.CreatedAt = now
	row.UpdatedAt = now
	if row.SourceLang == "" {
		row.SourceLang = "mql4"
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO imported_strategies (id,user_id,name,source_lang,source_code,params,coverage_score,created_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		row.ID, row.UserID, row.Name, row.SourceLang, row.SourceCode, row.Params, row.CoverageScore, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create imported strategy: %w", err)
	}
	return nil
}

func (r *ImportedStrategyRepository) GetByID(ctx context.Context, id uuid.UUID) (*ImportedStrategy, error) {
	var row ImportedStrategy
	err := r.db.QueryRow(ctx,
		`SELECT id,user_id,name,source_lang,source_code,params,coverage_score,created_at,updated_at
		 FROM imported_strategies WHERE id = $1`, id).
		Scan(&row.ID, &row.UserID, &row.Name, &row.SourceLang, &row.SourceCode, &row.Params, &row.CoverageScore, &row.CreatedAt, &row.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrImportedStrategyNotFound
	}
	return &row, err
}

func (r *ImportedStrategyRepository) GetByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*ImportedStrategy, error) {
	var row ImportedStrategy
	err := r.db.QueryRow(ctx,
		`SELECT id,user_id,name,source_lang,source_code,params,coverage_score,created_at,updated_at
		 FROM imported_strategies WHERE id = $1 AND user_id = $2`, id, userID).
		Scan(&row.ID, &row.UserID, &row.Name, &row.SourceLang, &row.SourceCode, &row.Params, &row.CoverageScore, &row.CreatedAt, &row.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrImportedStrategyNotFound
	}
	return &row, err
}

func (r *ImportedStrategyRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]ImportedStrategy, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.db.Query(ctx,
		`SELECT id,user_id,name,source_lang,source_code,params,coverage_score,created_at,updated_at
		 FROM imported_strategies WHERE user_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ImportedStrategy
	for rows.Next() {
		var row ImportedStrategy
		if err := rows.Scan(&row.ID, &row.UserID, &row.Name, &row.SourceLang, &row.SourceCode, &row.Params, &row.CoverageScore, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *ImportedStrategyRepository) UpdateCoverage(ctx context.Context, id uuid.UUID, coverage float64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE imported_strategies SET coverage_score = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		id, coverage)
	return err
}

func (r *ImportedStrategyRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	ct, err := r.db.Exec(ctx, `DELETE FROM imported_strategies WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrImportedStrategyNotFound
	}
	return nil
}
