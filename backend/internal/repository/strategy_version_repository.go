package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrStrategyVersionNotFound = errors.New("strategy version not found")

// StrategyVersion is a snapshot of strategy source code at a point in time.
type StrategyVersion struct {
	ID            uuid.UUID `db:"id"`
	StrategyID    uuid.UUID `db:"strategy_id"`
	UserID        uuid.UUID `db:"user_id"`
	VersionNumber int       `db:"version_number"`
	SourceCode    string    `db:"source_code"`
	SourceLang    string    `db:"source_lang"`
	ChangeSummary string    `db:"change_summary"`
	CodeHash      string    `db:"code_hash"`
	CreatedAt     time.Time `db:"created_at"`
}

// StrategyVersionMeta is a lightweight version entry without source_code.
// Used for listing version history without transferring large payloads.
type StrategyVersionMeta struct {
	ID            uuid.UUID `db:"id"`
	StrategyID    uuid.UUID `db:"strategy_id"`
	UserID        uuid.UUID `db:"user_id"`
	VersionNumber int       `db:"version_number"`
	SourceLang    string    `db:"source_lang"`
	ChangeSummary string    `db:"change_summary"`
	CodeHash      string    `db:"code_hash"`
	CreatedAt     time.Time `db:"created_at"`
}

type StrategyVersionRepository struct{ db *pgxpool.Pool }

func NewStrategyVersionRepository(db *pgxpool.Pool) *StrategyVersionRepository {
	return &StrategyVersionRepository{db: db}
}

// createVersionInTx inserts a new version snapshot within a transaction.
// If the latest version has the same code_hash, it is skipped (dedup).
// Used by CreateVersion, RollbackToVersion, and UpdateStrategyCode.
func createVersionInTx(ctx context.Context, tx pgx.Tx, strategyID, userID uuid.UUID, sourceCode, sourceLang, changeSummary string) (*StrategyVersion, error) {
	hash := hashCode(sourceCode)

	// Dedup: skip if latest version has the same code_hash
	var latestHash string
	err := tx.QueryRow(ctx,
		`SELECT code_hash FROM strategy_versions WHERE strategy_id = $1 ORDER BY version_number DESC LIMIT 1`,
		strategyID).Scan(&latestHash)
	if err == nil && latestHash == hash {
		// Same code as latest version — return the existing latest version
		var v StrategyVersion
		err := tx.QueryRow(ctx,
			`SELECT id, strategy_id, user_id, version_number, source_code, source_lang, change_summary, code_hash, created_at
			 FROM strategy_versions WHERE strategy_id = $1 AND user_id = $2 ORDER BY version_number DESC LIMIT 1`,
			strategyID, userID).
			Scan(&v.ID, &v.StrategyID, &v.UserID, &v.VersionNumber,
				&v.SourceCode, &v.SourceLang, &v.ChangeSummary, &v.CodeHash, &v.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("create version: fetch existing: %w", err)
		}
		return &v, nil
	}
	// err == pgx.ErrNoRows is fine — no existing versions, proceed to insert

	var version StrategyVersion
	err = tx.QueryRow(ctx,
		`INSERT INTO strategy_versions (strategy_id, user_id, version_number, source_code, source_lang, change_summary, code_hash)
		 VALUES ($1, $2, COALESCE((SELECT MAX(version_number) FROM strategy_versions WHERE strategy_id = $1), 0) + 1, $3, $4, $5, $6)
		 RETURNING id, strategy_id, user_id, version_number, source_code, source_lang, change_summary, code_hash, created_at`,
		strategyID, userID, sourceCode, sourceLang, changeSummary, hash).
		Scan(&version.ID, &version.StrategyID, &version.UserID, &version.VersionNumber,
			&version.SourceCode, &version.SourceLang, &version.ChangeSummary, &version.CodeHash, &version.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create version: %w", err)
	}
	return &version, nil
}

// CreateVersion snapshots the current strategy code as a new version.
// version_number is auto-incremented per strategy_id.
// If the latest version has the same code_hash, no new version is created.
// Uses a transaction for atomic dedup (SELECT + INSERT).
func (r *StrategyVersionRepository) CreateVersion(ctx context.Context, strategyID, userID uuid.UUID, sourceCode, sourceLang, changeSummary string) (*StrategyVersion, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("create strategy version: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	v, err := createVersionInTx(ctx, tx, strategyID, userID, sourceCode, sourceLang, changeSummary)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("create strategy version: commit: %w", err)
	}
	return v, nil
}

// ListVersions returns version metadata for a strategy, newest first.
// Does NOT include source_code — use GetVersion for full content.
func (r *StrategyVersionRepository) ListVersions(ctx context.Context, strategyID, userID uuid.UUID, limit, offset int) ([]StrategyVersionMeta, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, strategy_id, user_id, version_number, source_lang, change_summary, code_hash, created_at
		 FROM strategy_versions WHERE strategy_id = $1 AND user_id = $2
		 ORDER BY version_number DESC LIMIT $3 OFFSET $4`,
		strategyID, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []StrategyVersionMeta
	for rows.Next() {
		var v StrategyVersionMeta
		if err := rows.Scan(&v.ID, &v.StrategyID, &v.UserID, &v.VersionNumber,
			&v.SourceLang, &v.ChangeSummary, &v.CodeHash, &v.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

// GetVersion retrieves a specific version by version_number.
func (r *StrategyVersionRepository) GetVersion(ctx context.Context, strategyID, userID uuid.UUID, versionNumber int) (*StrategyVersion, error) {
	var v StrategyVersion
	err := r.db.QueryRow(ctx,
		`SELECT id, strategy_id, user_id, version_number, source_code, source_lang, change_summary, code_hash, created_at
		 FROM strategy_versions WHERE strategy_id = $1 AND user_id = $2 AND version_number = $3`,
		strategyID, userID, versionNumber).
		Scan(&v.ID, &v.StrategyID, &v.UserID, &v.VersionNumber,
			&v.SourceCode, &v.SourceLang, &v.ChangeSummary, &v.CodeHash, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStrategyVersionNotFound
	}
	return &v, err
}

// GetLatestVersion returns the most recent version for a strategy.
func (r *StrategyVersionRepository) GetLatestVersion(ctx context.Context, strategyID, userID uuid.UUID) (*StrategyVersion, error) {
	var v StrategyVersion
	err := r.db.QueryRow(ctx,
		`SELECT id, strategy_id, user_id, version_number, source_code, source_lang, change_summary, code_hash, created_at
		 FROM strategy_versions WHERE strategy_id = $1 AND user_id = $2
		 ORDER BY version_number DESC LIMIT 1`,
		strategyID, userID).
		Scan(&v.ID, &v.StrategyID, &v.UserID, &v.VersionNumber,
			&v.SourceCode, &v.SourceLang, &v.ChangeSummary, &v.CodeHash, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStrategyVersionNotFound
	}
	return &v, err
}

// RollbackToVersion restores strategy source_code from a specific version.
// It also creates a new version snapshot of the restored code.
// Both operations are wrapped in a single transaction for atomicity.
func (r *StrategyVersionRepository) RollbackToVersion(ctx context.Context, strategyID, userID uuid.UUID, versionNumber int) (*StrategyVersion, error) {
	target, err := r.GetVersion(ctx, strategyID, userID, versionNumber)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("rollback: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Update imported_strategies with the old code
	tag, err := tx.Exec(ctx,
		`UPDATE imported_strategies SET source_code = $2, source_lang = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND user_id = $4`,
		strategyID, target.SourceCode, target.SourceLang, userID)
	if err != nil {
		return nil, fmt.Errorf("rollback: update imported_strategies: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrImportedStrategyNotFound
	}

	// Create a new version snapshot of the restored code
	version, err := createVersionInTx(ctx, tx, strategyID, userID, target.SourceCode, target.SourceLang,
		fmt.Sprintf("Rollback to version %d", versionNumber))
	if err != nil {
		return nil, fmt.Errorf("rollback: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("rollback: commit: %w", err)
	}
	return version, nil
}

// UpdateStrategyCode updates the source code of an existing strategy and creates a version snapshot.
// source_lang is derived from the existing record via UPDATE RETURNING (it doesn't change on code update).
// Both operations are wrapped in a single transaction for atomicity.
func (r *StrategyVersionRepository) UpdateStrategyCode(ctx context.Context, strategyID, userID uuid.UUID, sourceCode, changeSummary string) (*StrategyVersion, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("update strategy code: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Update imported_strategies with the new code, returning source_lang for the version snapshot
	var sourceLang string
	err = tx.QueryRow(ctx,
		`UPDATE imported_strategies SET source_code = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND user_id = $3 RETURNING source_lang`,
		strategyID, sourceCode, userID).Scan(&sourceLang)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrImportedStrategyNotFound
		}
		return nil, fmt.Errorf("update strategy code: update imported_strategies: %w", err)
	}

	// Create a new version snapshot
	version, err := createVersionInTx(ctx, tx, strategyID, userID, sourceCode, sourceLang, changeSummary)
	if err != nil {
		return nil, fmt.Errorf("update strategy code: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("update strategy code: commit: %w", err)
	}
	return version, nil
}

// DiffVersions returns the two version snapshots for client-side diffing.
func (r *StrategyVersionRepository) DiffVersions(ctx context.Context, strategyID, userID uuid.UUID, fromVersion, toVersion int) (*StrategyVersion, *StrategyVersion, error) {
	from, err := r.GetVersion(ctx, strategyID, userID, fromVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("diff: from version: %w", err)
	}
	to, err := r.GetVersion(ctx, strategyID, userID, toVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("diff: to version: %w", err)
	}
	return from, to, nil
}

// CountVersions returns the total number of versions for a strategy.
func (r *StrategyVersionRepository) CountVersions(ctx context.Context, strategyID, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM strategy_versions WHERE strategy_id = $1 AND user_id = $2`,
		strategyID, userID).Scan(&count)
	return count, err
}

func hashCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}
