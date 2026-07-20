package sweep

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
)

// BundleRepository persists signed sweep bundles for crash recovery (ADR §2.3, Q3).
// Signed transactions contain no private keys — safe to persist.
// On restart, the worker reads back BROADCASTING bundles and resumes from
// the first unconfirmed leg (re-broadcast needs no private key).
type BundleRepository struct {
	db repository.DBTX
}

func NewBundleRepository(db repository.DBTX) *BundleRepository {
	return &BundleRepository{db: db}
}

// SaveUnsignedBundle persists an UnsignedSweepBundle with status PENDING_SIGN.
// This ensures crash recovery: if the server dies after creating sweep_log legs
// but before cold signing, the unsigned bundle can be re-exported on restart.
func (r *BundleRepository) SaveUnsignedBundle(ctx context.Context, batchID, addrID uuid.UUID, unsigned *antv1.UnsignedSweepBundle, builtAtMs int64) error {
	data, err := proto.Marshal(unsigned)
	if err != nil {
		return fmt.Errorf("sweep bundle repo: marshal unsigned: %w", err)
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO sweep_bundles (batch_id, deposit_address_id, unsigned_bundle, built_at_ms, status)
		VALUES ($1, $2, $3, $4, 'PENDING_SIGN')
		ON CONFLICT (batch_id) DO UPDATE SET unsigned_bundle = $3, updated_at = NOW()
	`, batchID, addrID, data, builtAtMs)
	if err != nil {
		return fmt.Errorf("sweep bundle repo: save unsigned: %w", err)
	}
	return nil
}

// ListPendingSignAddrIDs returns the set of deposit_address_ids that have
// a PENDING_SIGN bundle. Used by buildPendingBundles to skip addresses
// already awaiting cold signing (D4: prevent duplicate bundle creation).
// JOINs sweep_logs because batch bundles may have NULL deposit_address_id
// in sweep_bundles — the legs always carry the correct addr ID.
func (r *BundleRepository) ListPendingSignAddrIDs(ctx context.Context) (map[uuid.UUID]bool, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT sl.deposit_address_id
		FROM sweep_bundles sb
		JOIN sweep_logs sl ON sl.batch_id = sb.batch_id
		WHERE sb.status = 'PENDING_SIGN'
	`)
	if err != nil {
		return nil, fmt.Errorf("sweep bundle repo: list pending sign addr ids: %w", err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID]bool)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("sweep bundle repo: scan addr id: %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

// ExpireStalePendingSign marks PENDING_SIGN bundles older than maxAge as EXPIRED.
// Returns the batch_ids that were expired so the caller can mark their legs FAILED.
// D1: prevents orphaned PENDING_SIGN bundles from blocking addresses forever.
func (r *BundleRepository) ExpireStalePendingSign(ctx context.Context, maxAge time.Duration) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE sweep_bundles
		SET status = 'EXPIRED', updated_at = NOW()
		WHERE status = 'PENDING_SIGN' AND created_at < NOW() - make_interval(secs => $1)
		RETURNING batch_id
	`, int64(maxAge.Seconds()))
	if err != nil {
		return nil, fmt.Errorf("sweep bundle repo: expire stale pending sign: %w", err)
	}
	defer rows.Close()

	var batchIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("sweep bundle repo: scan expired batch_id: %w", err)
		}
		batchIDs = append(batchIDs, id)
	}
	return batchIDs, rows.Err()
}

// PendingSignBundleSummary is a lightweight view of a PENDING_SIGN bundle for admin listing.
type PendingSignBundleSummary struct {
	BatchID          uuid.UUID
	DepositAddressID *uuid.UUID // first address; NULL for batch bundles
	BuiltAtMs        int64
	Status           string
}

// ListPendingSignBundlesForAdmin returns all PENDING_SIGN bundle summaries for the admin RPC.
func (r *BundleRepository) ListPendingSignBundlesForAdmin(ctx context.Context) ([]PendingSignBundleSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT sb.batch_id, sb.deposit_address_id, sb.built_at_ms, sb.status
		FROM sweep_bundles sb
		WHERE sb.status = 'PENDING_SIGN'
		ORDER BY sb.created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("sweep bundle repo: list pending sign for admin: %w", err)
	}
	defer rows.Close()

	var out []PendingSignBundleSummary
	for rows.Next() {
		var s PendingSignBundleSummary
		if err := rows.Scan(&s.BatchID, &s.DepositAddressID, &s.BuiltAtMs, &s.Status); err != nil {
			return nil, fmt.Errorf("sweep bundle repo: scan pending sign summary: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SaveBundle persists a SignedSweepBundle with its batch_id.
func (r *BundleRepository) SaveBundle(ctx context.Context, batchID uuid.UUID, signed *antv1.SignedSweepBundle, builtAtMs int64) error {
	data, err := proto.Marshal(signed)
	if err != nil {
		return fmt.Errorf("sweep bundle repo: marshal: %w", err)
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO sweep_bundles (batch_id, signed_bundle, built_at_ms, status)
		VALUES ($1, $2, $3, 'BROADCASTING')
		ON CONFLICT (batch_id) DO UPDATE SET signed_bundle = $2, status = 'BROADCASTING', updated_at = NOW()
	`, batchID, data, builtAtMs)
	if err != nil {
		return fmt.Errorf("sweep bundle repo: save: %w", err)
	}
	return nil
}

// GetBundle retrieves a signed bundle by batch_id.
func (r *BundleRepository) GetBundle(ctx context.Context, batchID uuid.UUID) (*antv1.SignedSweepBundle, error) {
	var data []byte
	err := r.db.QueryRow(ctx, `
		SELECT signed_bundle FROM sweep_bundles WHERE batch_id = $1
	`, batchID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("sweep bundle repo: get: %w", err)
	}

	bundle := &antv1.SignedSweepBundle{}
	if err := proto.Unmarshal(data, bundle); err != nil {
		return nil, fmt.Errorf("sweep bundle repo: unmarshal: %w", err)
	}
	return bundle, nil
}

// ListBroadcastingBundles returns all bundles with status='BROADCASTING'.
// Used on restart to resume broadcasting from the first unconfirmed leg.
func (r *BundleRepository) ListBroadcastingBundles(ctx context.Context) ([]BroadcastingBundle, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, batch_id, built_at_ms, status
		FROM sweep_bundles
		WHERE status = 'BROADCASTING'
		ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("sweep bundle repo: list broadcasting: %w", err)
	}
	defer rows.Close()

	var out []BroadcastingBundle
	for rows.Next() {
		var b BroadcastingBundle
		if err := rows.Scan(&b.ID, &b.BatchID, &b.BuiltAtMs, &b.Status); err != nil {
			return nil, fmt.Errorf("sweep bundle repo: scan: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// MarkBundleDone transitions a bundle to DONE status.
func (r *BundleRepository) MarkBundleDone(ctx context.Context, batchID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sweep_bundles SET status = 'DONE', updated_at = NOW()
		WHERE batch_id = $1
	`, batchID)
	if err != nil {
		return fmt.Errorf("sweep bundle repo: mark done: %w", err)
	}
	return nil
}

// MarkBundleManualReview transitions a bundle to MANUAL_REVIEW status (D14).
// Called when BroadcastBundle encounters a MANUAL_REVIEW leg — stops the
// 30s retry cycle that would otherwise persist for 23h until expiry.
func (r *BundleRepository) MarkBundleManualReview(ctx context.Context, batchID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sweep_bundles SET status = 'MANUAL_REVIEW', updated_at = NOW()
		WHERE batch_id = $1
	`, batchID)
	if err != nil {
		return fmt.Errorf("sweep bundle repo: mark manual review: %w", err)
	}
	return nil
}

// MarkBundleExpired transitions a bundle to EXPIRED status.
// Called when the raw_tx expiry has passed and the bundle was not fully broadcast.
func (r *BundleRepository) MarkBundleExpired(ctx context.Context, batchID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sweep_bundles SET status = 'EXPIRED', updated_at = NOW()
		WHERE batch_id = $1
	`, batchID)
	if err != nil {
		return fmt.Errorf("sweep bundle repo: mark expired: %w", err)
	}
	return nil
}

// BroadcastingBundle is a persisted signed bundle row.
type BroadcastingBundle struct {
	ID        uuid.UUID
	BatchID   uuid.UUID
	BuiltAtMs int64
	Status    string
}

// IsExpired checks if the bundle's raw_tx has expired (built_at_ms + ~24h).
func (b *BroadcastingBundle) IsExpired() bool {
	expiryMs := b.BuiltAtMs + (23 * time.Hour).Milliseconds()
	return time.Now().UnixMilli() > expiryMs
}
