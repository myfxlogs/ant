package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// FailureSignatureRepository persists backtest failure signatures and repro packages.
type FailureSignatureRepository struct {
	db *pgxpool.Pool
}

func NewFailureSignatureRepository(db *pgxpool.Pool) *FailureSignatureRepository {
	return &FailureSignatureRepository{db: db}
}

// SaveFailureSignature persists a failure signature with its repro package.
// If the same signature_hash already exists, it returns the existing ID (dedup).
func (r *FailureSignatureRepository) SaveFailureSignature(ctx context.Context, pkg *antv1.ReproPackage, backtestRunID *uuid.UUID) (int64, error) {
	sig := pkg.GetSignature()
	if sig == nil {
		return 0, nil
	}

	// Check for existing (dedup by hash)
	var existingID int64
	err := r.db.QueryRow(ctx,
		`SELECT id FROM failure_signatures WHERE signature_hash = $1 LIMIT 1`,
		sig.Hash,
	).Scan(&existingID)
	if err == nil {
		return existingID, nil
	}

	// Serialize findings as proto bytes for BYTEA storage
	findingsProto := &antv1.DiagnosticFindingList{Findings: pkg.GetFindings()}
	findingsBytes, err := proto.Marshal(findingsProto)
	if err != nil {
		return 0, err
	}

	var id int64
	err = r.db.QueryRow(ctx, `
		INSERT INTO failure_signatures
			(signature_hash, source_hash, rule_ids, blind_spots, total_trades,
			 symbol, timeframe, source_preview, findings, backtest_run_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
		sig.Hash,
		sig.SourceHash,
		sig.RuleIds,
		sig.BlindSpots,
		sig.TotalTrades,
		pkg.Symbol,
		pkg.Timeframe,
		pkg.SourcePreview,
		findingsBytes,
		backtestRunID,
		time.UnixMilli(pkg.CreatedAtMs).UTC(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetBySourceHash returns all failure signatures for a given source hash.
func (r *FailureSignatureRepository) GetBySourceHash(ctx context.Context, sourceHash string) ([]*antv1.FailureSignature, error) {
	rows, err := r.db.Query(ctx, `
		SELECT signature_hash, source_hash, rule_ids, blind_spots, total_trades, created_at
		FROM failure_signatures WHERE source_hash = $1 ORDER BY created_at DESC`,
		sourceHash,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*antv1.FailureSignature
	for rows.Next() {
		sig := &antv1.FailureSignature{}
		var ruleIDs, blindSpots []string
		var createdAt time.Time
		if err := rows.Scan(&sig.Hash, &sig.SourceHash, &ruleIDs, &blindSpots, &sig.TotalTrades, &createdAt); err != nil {
			return nil, err
		}
		sig.RuleIds = ruleIDs
		sig.BlindSpots = blindSpots
		sig.CreatedAtMs = createdAt.UnixMilli()
		results = append(results, sig)
	}
	return results, rows.Err()
}

// GetRecent returns the most recent N failure signatures.
func (r *FailureSignatureRepository) GetRecent(ctx context.Context, limit int) ([]*antv1.ReproPackage, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, signature_hash, source_hash, rule_ids, blind_spots, total_trades,
		       symbol, timeframe, source_preview, findings, created_at
		FROM failure_signatures ORDER BY created_at DESC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*antv1.ReproPackage
	for rows.Next() {
		pkg := &antv1.ReproPackage{
			Signature: &antv1.FailureSignature{},
		}
		var ruleIDs, blindSpots []string
		var createdAt time.Time
		var id int64
		var findingsBytes []byte
		if err := rows.Scan(
			&id,
			&pkg.Signature.Hash, &pkg.Signature.SourceHash,
			&ruleIDs, &blindSpots,
			&pkg.Signature.TotalTrades,
			&pkg.Symbol, &pkg.Timeframe, &pkg.SourcePreview,
			&findingsBytes,
			&createdAt,
		); err != nil {
			return nil, err
		}
		pkg.Signature.RuleIds = ruleIDs
		pkg.Signature.BlindSpots = blindSpots
		pkg.Signature.CreatedAtMs = createdAt.UnixMilli()
		pkg.CreatedAtMs = createdAt.UnixMilli()

		if len(findingsBytes) > 0 {
			var fl antv1.DiagnosticFindingList
			if err := proto.Unmarshal(findingsBytes, &fl); err == nil {
				pkg.Findings = fl.Findings
			}
		}

		results = append(results, pkg)
	}
	return results, rows.Err()
}
