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
	ErrStrategyAssetNotFound      = errors.New("strategy asset not found")
	ErrStrategyAssetCloneNotFound = errors.New("strategy asset clone not found")
)

type StrategyAsset struct {
	ID               uuid.UUID `db:"id"`
	OwnerUserID      uuid.UUID `db:"owner_user_id"`
	SourceTemplateID uuid.UUID `db:"source_template_id"`
	SourceVersion    int       `db:"source_version"`
	Name             string    `db:"name"`
	Description      string    `db:"description"`
	Visibility       string    `db:"visibility"`
	ReviewStatus     string    `db:"review_status"`
	RatingSummary    string    `db:"rating_summary"`
	CloneCount       int       `db:"clone_count"`
	LatestVersion    int       `db:"latest_version"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

type StrategyAssetClone struct {
	ID               uuid.UUID  `db:"id"`
	AssetID          uuid.UUID  `db:"asset_id"`
	UserID           uuid.UUID  `db:"user_id"`
	ClonedTemplateID uuid.UUID  `db:"cloned_template_id"`
	SourceVersion    int        `db:"source_version"`
	SyncAvailable    bool       `db:"sync_available"`
	LastSyncCheckAt  *time.Time `db:"last_sync_check_at"`
	CreatedAt        time.Time  `db:"created_at"`
}

type StrategyAssetRepository struct{ db *pgxpool.Pool }

func NewStrategyAssetRepository(db *pgxpool.Pool) *StrategyAssetRepository {
	return &StrategyAssetRepository{db: db}
}

func (r *StrategyAssetRepository) Create(ctx context.Context, row *StrategyAsset) error {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	now := time.Now().UTC()
	row.CreatedAt = now
	row.UpdatedAt = now
	if row.Visibility == "" {
		row.Visibility = "private"
	}
	if row.ReviewStatus == "" {
		row.ReviewStatus = "pending"
	}
	if row.SourceVersion <= 0 {
		row.SourceVersion = 1
	}
	if row.LatestVersion <= 0 {
		row.LatestVersion = row.SourceVersion
	}
	_, err := r.db.Exec(ctx, `INSERT INTO strategy_assets (id,owner_user_id,source_template_id,source_version,name,description,visibility,review_status,rating_summary,clone_count,latest_version,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, row.ID, row.OwnerUserID, row.SourceTemplateID, row.SourceVersion, row.Name, row.Description, row.Visibility, row.ReviewStatus, row.RatingSummary, row.CloneCount, row.LatestVersion, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create asset: %w", err)
	}
	return nil
}

func (r *StrategyAssetRepository) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]StrategyAsset, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	queryRows, err := r.db.Query(ctx, `SELECT * FROM strategy_assets WHERE owner_user_id = $1 OR (visibility = 'public' AND review_status = 'approved') ORDER BY updated_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer queryRows.Close()
	var result []StrategyAsset
	for queryRows.Next() {
		var row StrategyAsset
		if err := queryRows.Scan(&row.ID, &row.OwnerUserID, &row.SourceTemplateID, &row.SourceVersion, &row.Name, &row.Description, &row.Visibility, &row.ReviewStatus, &row.RatingSummary, &row.CloneCount, &row.LatestVersion, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	if err := queryRows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *StrategyAssetRepository) Get(ctx context.Context, userID, id uuid.UUID) (*StrategyAsset, error) {
	var row StrategyAsset
	err := r.db.QueryRow(ctx, `SELECT * FROM strategy_assets WHERE id = $1 AND (owner_user_id = $2 OR (visibility = 'public' AND review_status = 'approved'))`, id, userID).Scan(&row.ID, &row.OwnerUserID, &row.SourceTemplateID, &row.SourceVersion, &row.Name, &row.Description, &row.Visibility, &row.ReviewStatus, &row.RatingSummary, &row.CloneCount, &row.LatestVersion, &row.CreatedAt, &row.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStrategyAssetNotFound
	}
	return &row, err
}

func (r *StrategyAssetRepository) Review(ctx context.Context, id uuid.UUID, status, summary string) (*StrategyAsset, error) {
	var row StrategyAsset
	err := r.db.QueryRow(ctx, `UPDATE strategy_assets SET review_status = $2, rating_summary = $3, updated_at = now() WHERE id = $1 RETURNING *`, id, status, summary).Scan(&row.ID, &row.OwnerUserID, &row.SourceTemplateID, &row.SourceVersion, &row.Name, &row.Description, &row.Visibility, &row.ReviewStatus, &row.RatingSummary, &row.CloneCount, &row.LatestVersion, &row.CreatedAt, &row.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStrategyAssetNotFound
	}
	return &row, err
}

func (r *StrategyAssetRepository) CreateClone(ctx context.Context, row *StrategyAssetClone) error {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.Exec(ctx, `INSERT INTO strategy_asset_clones (id,asset_id,user_id,cloned_template_id,source_version,sync_available,last_sync_check_at,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, row.ID, row.AssetID, row.UserID, row.ClonedTemplateID, row.SourceVersion, row.SyncAvailable, row.LastSyncCheckAt, row.CreatedAt)
	if err == nil {
		_, _ = r.db.Exec(ctx, `UPDATE strategy_assets SET clone_count = clone_count + 1 WHERE id = $1`, row.AssetID)
	}
	if err != nil {
		return fmt.Errorf("create asset clone: %w", err)
	}
	return nil
}

func (r *StrategyAssetRepository) GetClone(ctx context.Context, userID, id uuid.UUID) (*StrategyAssetClone, error) {
	var row StrategyAssetClone
	err := r.db.QueryRow(ctx, `SELECT * FROM strategy_asset_clones WHERE id = $1 AND user_id = $2`, id, userID).Scan(&row.ID, &row.AssetID, &row.UserID, &row.ClonedTemplateID, &row.SourceVersion, &row.SyncAvailable, &row.LastSyncCheckAt, &row.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStrategyAssetCloneNotFound
	}
	return &row, err
}

func (r *StrategyAssetRepository) ListClones(ctx context.Context, userID, assetID uuid.UUID) ([]StrategyAssetClone, error) {
	queryRows, err := r.db.Query(ctx, `SELECT c.* FROM strategy_asset_clones c JOIN strategy_assets a ON a.id = c.asset_id WHERE c.asset_id = $1 AND (c.user_id = $2 OR a.owner_user_id = $2) ORDER BY c.created_at DESC`, assetID, userID)
	if err != nil {
		return nil, err
	}
	defer queryRows.Close()
	var result []StrategyAssetClone
	for queryRows.Next() {
		var row StrategyAssetClone
		if err := queryRows.Scan(&row.ID, &row.AssetID, &row.UserID, &row.ClonedTemplateID, &row.SourceVersion, &row.SyncAvailable, &row.LastSyncCheckAt, &row.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	if err := queryRows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *StrategyAssetRepository) UpdateCloneSync(ctx context.Context, id uuid.UUID, sourceVersion int, syncAvailable bool) (*StrategyAssetClone, error) {
	now := time.Now().UTC()
	var row StrategyAssetClone
	err := r.db.QueryRow(ctx, `UPDATE strategy_asset_clones SET source_version = $2, sync_available = $3, last_sync_check_at = $4 WHERE id = $1 RETURNING *`, id, sourceVersion, syncAvailable, now).Scan(&row.ID, &row.AssetID, &row.UserID, &row.ClonedTemplateID, &row.SourceVersion, &row.SyncAvailable, &row.LastSyncCheckAt, &row.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStrategyAssetCloneNotFound
	}
	return &row, err
}
