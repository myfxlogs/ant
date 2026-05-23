package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSystemAIConfigNotFound = errors.New("system ai config not found")

// SystemAIConfigRow mirrors a row in system_ai_configs. Secret material is
// stored encrypted server-side via the secretbox package; this struct only
// exposes the metadata flag has_secret.
//
// Since 059 系统 AI 配置改为「按用户隔离」：所有读写都必须带 user_id。
type SystemAIConfigRow struct {
	UserID         uuid.UUID
	ProviderID     string
	Name           string
	BaseURL        string
	Organization   string
	Models         []string
	DefaultModel   string
	Temperature    float64
	TimeoutSeconds int
	MaxTokens      int
	Purposes       []string
	PrimaryFor     []string
	Enabled        bool
	HasSecret      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	UpdatedBy      string
}

// SystemAISecret holds raw ciphertext bytes; only the service layer should
// touch these.
type SystemAISecret struct {
	Ciphertext []byte
	Salt       []byte
	Nonce      []byte
}

type SystemAIConfigRepository struct {
	db *pgxpool.Pool
}

func NewSystemAIConfigRepository(db *pgxpool.Pool) *SystemAIConfigRepository {
	return &SystemAIConfigRepository{db: db}
}

const systemAIConfigSelectCols = `
	user_id, provider_id, name, base_url, organization, models, default_model,
	temperature, timeout_seconds, max_tokens, purposes, primary_for,
	enabled, has_secret, created_at, updated_at, updated_by
`

func scanSystemAIConfigRow(scanner pgx.Row) (*SystemAIConfigRow, error) {
	r := &SystemAIConfigRow{}
	err := scanner.Scan(
		&r.UserID, &r.ProviderID, &r.Name, &r.BaseURL, &r.Organization, &r.Models, &r.DefaultModel,
		&r.Temperature, &r.TimeoutSeconds, &r.MaxTokens, &r.Purposes, &r.PrimaryFor,
		&r.Enabled, &r.HasSecret, &r.CreatedAt, &r.UpdatedAt, &r.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSystemAIConfigNotFound
		}
		return nil, err
	}
	return r, nil
}

func (r *SystemAIConfigRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*SystemAIConfigRow, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+systemAIConfigSelectCols+" FROM system_ai_configs WHERE user_id = $1 ORDER BY provider_id", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*SystemAIConfigRow, 0, 7)
	for rows.Next() {
		row, err := scanSystemAIConfigRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *SystemAIConfigRepository) Get(ctx context.Context, userID uuid.UUID, providerID string) (*SystemAIConfigRow, error) {
	return scanSystemAIConfigRow(r.db.QueryRow(ctx,
		"SELECT "+systemAIConfigSelectCols+" FROM system_ai_configs WHERE user_id = $1 AND provider_id = $2",
		userID, providerID))
}

// Upsert inserts a new row or updates an existing one for (user_id, provider_id).
// 用 INSERT ... ON CONFLICT 让首次配置自然 seed，避免上层先 ensure-seed 再 update 的两步操作。
func (r *SystemAIConfigRepository) Upsert(ctx context.Context, row *SystemAIConfigRow, updatedBy string) error {
	models := row.Models
	if models == nil {
		models = []string{}
	}
	purposes := row.Purposes
	if purposes == nil {
		purposes = []string{}
	}
	primaryFor := row.PrimaryFor
	if primaryFor == nil {
		primaryFor = []string{}
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO system_ai_configs (
			user_id, provider_id, name, base_url, organization, models, default_model,
			temperature, timeout_seconds, max_tokens, purposes, primary_for, enabled, updated_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (user_id, provider_id) DO UPDATE SET
			name = EXCLUDED.name,
			base_url = EXCLUDED.base_url,
			organization = EXCLUDED.organization,
			models = EXCLUDED.models,
			default_model = EXCLUDED.default_model,
			temperature = EXCLUDED.temperature,
			timeout_seconds = EXCLUDED.timeout_seconds,
			max_tokens = EXCLUDED.max_tokens,
			purposes = EXCLUDED.purposes,
			primary_for = EXCLUDED.primary_for,
			enabled = EXCLUDED.enabled,
			updated_at = NOW(),
			updated_by = EXCLUDED.updated_by
	`,
		row.UserID, row.ProviderID, row.Name, row.BaseURL, row.Organization, models, row.DefaultModel,
		row.Temperature, row.TimeoutSeconds, row.MaxTokens, purposes, primaryFor, row.Enabled, updatedBy)
	return err
}

func (r *SystemAIConfigRepository) Delete(ctx context.Context, userID uuid.UUID, providerID string) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM system_ai_configs
		WHERE user_id = $1 AND provider_id = $2`, userID, providerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSystemAIConfigNotFound
	}
	return nil
}

// SetSecret stores the encrypted secret bytes. Pass nil to clear.
func (r *SystemAIConfigRepository) SetSecret(ctx context.Context, userID uuid.UUID, providerID string, sec *SystemAISecret, updatedBy string) error {
	if sec == nil {
		tag, err := r.db.Exec(ctx, `
			UPDATE system_ai_configs SET
				secret_ciphertext = NULL, secret_salt = NULL, secret_nonce = NULL,
				has_secret = FALSE, updated_at = NOW(), updated_by = $1
			WHERE user_id = $2 AND provider_id = $3`, updatedBy, userID, providerID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrSystemAIConfigNotFound
		}
		return nil
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE system_ai_configs SET
			secret_ciphertext = $1, secret_salt = $2, secret_nonce = $3,
			has_secret = TRUE, updated_at = NOW(), updated_by = $4
		WHERE user_id = $5 AND provider_id = $6`,
		sec.Ciphertext, sec.Salt, sec.Nonce, updatedBy, userID, providerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSystemAIConfigNotFound
	}
	return nil
}

// GetSecret returns the raw ciphertext blobs, or (nil, nil) if no secret stored.
func (r *SystemAIConfigRepository) GetSecret(ctx context.Context, userID uuid.UUID, providerID string) (*SystemAISecret, error) {
	var ct, salt, nonce []byte
	err := r.db.QueryRow(ctx, `
		SELECT secret_ciphertext, secret_salt, secret_nonce
		FROM system_ai_configs
		WHERE user_id = $1 AND provider_id = $2 AND has_secret = TRUE`, userID, providerID).
		Scan(&ct, &salt, &nonce)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &SystemAISecret{Ciphertext: ct, Salt: salt, Nonce: nonce}, nil
}
