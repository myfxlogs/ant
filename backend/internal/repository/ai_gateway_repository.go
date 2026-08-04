package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"alphaforge/internal/pkg/secretbox"
)

// ── System AI Provider ──

type SystemAIProvider struct {
	ID              uuid.UUID `db:"id"`
	ProviderID      string    `db:"provider_id"`
	Name            string    `db:"name"`
	BaseURL         string    `db:"base_url"`
	APIKeyEncrypted []byte    `db:"api_key_encrypted"`
	Enabled         bool      `db:"enabled"`
	DefaultModel    string    `db:"default_model"`
	Models          []string  `db:"-"` // populated from ai_models join
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// SealAPIKey encrypts a plaintext API key and returns the encrypted blob.
// Format: 2-byte salt-len + salt + 2-byte nonce-len + nonce + ciphertext.
func SealAPIKey(plaintext string, box *secretbox.Box) ([]byte, error) {
	ct, salt, nonce, err := box.Seal([]byte(plaintext))
	if err != nil {
		return nil, err
	}
	out := make([]byte, 2+len(salt)+2+len(nonce)+len(ct))
	out[0] = byte(len(salt) >> 8)
	out[1] = byte(len(salt))
	copy(out[2:], salt)
	off := 2 + len(salt)
	out[off] = byte(len(nonce) >> 8)
	out[off+1] = byte(len(nonce))
	copy(out[off+2:], nonce)
	copy(out[off+2+len(nonce):], ct)
	return out, nil
}

// OpenAPIKey decrypts an encrypted API key blob.
func OpenAPIKey(encrypted []byte, box *secretbox.Box) (string, error) {
	if len(encrypted) < 4 {
		return "", fmt.Errorf("invalid encrypted key")
	}
	saltLen := int(encrypted[0])<<8 | int(encrypted[1])
	if 2+saltLen+2 > len(encrypted) {
		return "", fmt.Errorf("invalid encrypted key")
	}
	salt := encrypted[2 : 2+saltLen]
	off := 2 + saltLen
	nonceLen := int(encrypted[off])<<8 | int(encrypted[off+1])
	if off+2+nonceLen > len(encrypted) {
		return "", fmt.Errorf("invalid encrypted key")
	}
	nonce := encrypted[off+2 : off+2+nonceLen]
	ct := encrypted[off+2+nonceLen:]
	pt, err := box.Open(ct, salt, nonce)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

type SystemAIProviderRepository struct {
	db *pgxpool.Pool
}

func NewSystemAIProviderRepository(db *pgxpool.Pool) *SystemAIProviderRepository {
	return &SystemAIProviderRepository{db: db}
}

func (r *SystemAIProviderRepository) ListEnabled(ctx context.Context) ([]*SystemAIProvider, error) {
	rows, err := r.db.Query(ctx,
		`SELECT p.id, p.provider_id, p.name, p.base_url, p.api_key_encrypted, p.enabled,
		        COALESCE(p.default_model, '') AS default_model,
		        p.created_at, p.updated_at,
		        COALESCE(array_agg(m.model_name ORDER BY m.sort_order) FILTER (WHERE m.enabled), '{}') AS models
		 FROM system_ai_providers p
		 LEFT JOIN ai_models m ON m.provider_id = p.id
		 WHERE p.enabled = true
		 GROUP BY p.id
		 ORDER BY p.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*SystemAIProvider
	for rows.Next() {
		var p SystemAIProvider
		var models []string
		if err := rows.Scan(&p.ID, &p.ProviderID, &p.Name, &p.BaseURL, &p.APIKeyEncrypted, &p.Enabled, &p.DefaultModel, &p.CreatedAt, &p.UpdatedAt, &models); err != nil {
			return nil, err
		}
		p.Models = models
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (r *SystemAIProviderRepository) ListAll(ctx context.Context) ([]*SystemAIProvider, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, provider_id, name, base_url, api_key_encrypted, enabled, created_at, updated_at
		 FROM system_ai_providers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAIProviders(rows)
}

func scanAIProviders(rows interface{ Next() bool; Scan(...interface{}) error; Err() error }) ([]*SystemAIProvider, error) {
	var out []*SystemAIProvider
	for rows.Next() {
		var p SystemAIProvider
		if err := rows.Scan(&p.ID, &p.ProviderID, &p.Name, &p.BaseURL, &p.APIKeyEncrypted, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// ── AI Model ──

type AIModel struct {
	ID              uuid.UUID `db:"id"`
	ProviderID      uuid.UUID `db:"provider_id"`
	ModelName       string    `db:"model_name"`
	DisplayName     string    `db:"display_name"`
	PricePer1MInput  string   `db:"price_per_1m_input"`
	PricePer1MOutput string   `db:"price_per_1m_output"`
	MarkupRate      string    `db:"markup_rate"`
	ModelTier       string    `db:"model_tier"`
	Enabled         bool      `db:"enabled"`
	SortOrder       int       `db:"sort_order"`
	CreatedAt       time.Time `db:"created_at"`
}

type AIModelRepository struct {
	db *pgxpool.Pool
}

func NewAIModelRepository(db *pgxpool.Pool) *AIModelRepository {
	return &AIModelRepository{db: db}
}

func (r *SystemAIProviderRepository) Create(ctx context.Context, p *SystemAIProvider) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO system_ai_providers (id, provider_id, name, base_url, api_key_encrypted, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.ProviderID, p.Name, p.BaseURL, p.APIKeyEncrypted, p.Enabled)
	return err
}

func (r *SystemAIProviderRepository) Update(ctx context.Context, p *SystemAIProvider) error {
	_, err := r.db.Exec(ctx,
		`UPDATE system_ai_providers SET name=$1, base_url=$2, api_key_encrypted=$3, enabled=$4, updated_at=NOW()
		 WHERE id=$5`,
		p.Name, p.BaseURL, p.APIKeyEncrypted, p.Enabled, p.ID)
	return err
}

func (r *SystemAIProviderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM system_ai_providers WHERE id=$1`, id)
	return err
}

func (r *SystemAIProviderRepository) GetByID(ctx context.Context, id uuid.UUID) (*SystemAIProvider, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, provider_id, name, base_url, api_key_encrypted, enabled, created_at, updated_at
		 FROM system_ai_providers WHERE id=$1`, id)
	var p SystemAIProvider
	err := row.Scan(&p.ID, &p.ProviderID, &p.Name, &p.BaseURL, &p.APIKeyEncrypted, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ── AI Model CRUD ──

func (r *AIModelRepository) ListEnabled(ctx context.Context) ([]*AIModel, error) {
	rows, err := r.db.Query(ctx,
		`SELECT m.id, m.provider_id, m.model_name, m.display_name, m.price_per_1m_input, m.price_per_1m_output,
		        m.markup_rate, m.model_tier, m.enabled, m.sort_order, m.created_at
		 FROM ai_models m JOIN system_ai_providers p ON m.provider_id = p.id
		 WHERE m.enabled = true AND p.enabled = true ORDER BY p.name, m.sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAIModelRows(rows)
}

func (r *AIModelRepository) ListAll(ctx context.Context) ([]*AIModel, error) {
	rows, err := r.db.Query(ctx,
		`SELECT m.id, m.provider_id, m.model_name, m.display_name, m.price_per_1m_input, m.price_per_1m_output,
		        m.markup_rate, m.model_tier, m.enabled, m.sort_order, m.created_at
		 FROM ai_models m ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAIModelRows(rows)
}

func scanAIModelRows(rows interface{ Next() bool; Scan(...interface{}) error; Err() error }) ([]*AIModel, error) {
	var out []*AIModel
	for rows.Next() {
		var m AIModel
		var dn *string
		if err := rows.Scan(&m.ID, &m.ProviderID, &m.ModelName, &dn, &m.PricePer1MInput, &m.PricePer1MOutput, &m.MarkupRate, &m.ModelTier, &m.Enabled, &m.SortOrder, &m.CreatedAt); err != nil {
			return nil, err
		}
		if dn != nil {
			m.DisplayName = *dn
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (r *AIModelRepository) Upsert(ctx context.Context, m *AIModel) (uuid.UUID, error) {
	if m.MarkupRate == "" {
		m.MarkupRate = "1.5"
	}
	if m.ModelTier == "" {
		m.ModelTier = "flagship"
	}
	if m.ID != uuid.Nil {
		_, err := r.db.Exec(ctx,
			`UPDATE ai_models SET model_name=$1, display_name=$2, price_per_1m_input=$3, price_per_1m_output=$4,
			 markup_rate=$5, model_tier=$6, enabled=$7, sort_order=$8 WHERE id=$9`,
			m.ModelName, m.DisplayName, m.PricePer1MInput, m.PricePer1MOutput, m.MarkupRate, m.ModelTier, m.Enabled, m.SortOrder, m.ID)
		return m.ID, err
	}
	id := uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO ai_models (id, provider_id, model_name, display_name, price_per_1m_input, price_per_1m_output, markup_rate, model_tier, enabled, sort_order)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		id, m.ProviderID, m.ModelName, m.DisplayName, m.PricePer1MInput, m.PricePer1MOutput, m.MarkupRate, m.ModelTier, m.Enabled, m.SortOrder)
	return id, err
}

func (r *AIModelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM ai_models WHERE id=$1`, id)
	return err
}

func (r *AIModelRepository) GetByProviderAndModel(ctx context.Context, providerID, modelName string) (*AIModel, error) {
	m := &AIModel{}
	err := r.db.QueryRow(ctx,
		`SELECT m.id, m.provider_id, m.model_name, m.display_name, m.price_per_1m_input, m.price_per_1m_output,
		        m.markup_rate, m.model_tier, m.enabled, m.sort_order, m.created_at
		 FROM ai_models m JOIN system_ai_providers p ON m.provider_id = p.id
		 WHERE p.provider_id = $1 AND m.model_name = $2`, providerID, modelName,
	).Scan(&m.ID, &m.ProviderID, &m.ModelName, &m.DisplayName,
		&m.PricePer1MInput, &m.PricePer1MOutput, &m.MarkupRate, &m.ModelTier,
		&m.Enabled, &m.SortOrder, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *AIModelRepository) ListByProvider(ctx context.Context, providerID uuid.UUID) ([]*AIModel, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, provider_id, model_name, display_name, price_per_1m_input, price_per_1m_output, markup_rate, model_tier, enabled, sort_order, created_at
		 FROM ai_models WHERE provider_id=$1 ORDER BY sort_order`, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAIModelRows(rows)
}

// ── AI Token Usage ──

type AITokenUsage struct {
	ID                  uuid.UUID  `db:"id"`
	UserID              uuid.UUID  `db:"user_id"`
	WalletTransactionID *uuid.UUID `db:"wallet_transaction_id"`
	PaidBy              string     `db:"paid_by"`
	ProviderID          string     `db:"provider_id"`
	ModelName           string     `db:"model_name"`
	Feature             string     `db:"feature"`
	InputTokens         int        `db:"input_tokens"`
	OutputTokens        int        `db:"output_tokens"`
	Cost                string     `db:"cost"`
	SessionID           *uuid.UUID `db:"session_id"`
	CreatedAt           time.Time  `db:"created_at"`
}

type AITokenUsageRepository struct {
	db *pgxpool.Pool
}

func NewAITokenUsageRepository(db *pgxpool.Pool) *AITokenUsageRepository {
	return &AITokenUsageRepository{db: db}
}

func (r *AITokenUsageRepository) Insert(ctx context.Context, u *AITokenUsage) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO ai_token_usage (id, user_id, wallet_transaction_id, paid_by, provider_id, model_name, feature, input_tokens, output_tokens, cost, session_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		u.ID, u.UserID, u.WalletTransactionID, u.PaidBy, u.ProviderID, u.ModelName, u.Feature, u.InputTokens, u.OutputTokens, u.Cost, u.SessionID)
	return err
}

func (r *AITokenUsageRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*AITokenUsage, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, wallet_transaction_id, paid_by, provider_id, model_name, feature, input_tokens, output_tokens, cost, created_at
		 FROM ai_token_usage WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTokenUsageRows(rows)
}

func (r *AITokenUsageRepository) MonthlyCost(ctx context.Context, userID uuid.UUID) (string, error) {
	var cost string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(cost::numeric), 0)::text
		 FROM ai_token_usage
		 WHERE user_id = $1 AND created_at >= date_trunc('month', NOW())`, userID).Scan(&cost)
	return cost, err
}

func (r *AITokenUsageRepository) DailyTokenUsage(ctx context.Context, userID uuid.UUID) (int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(input_tokens + output_tokens), 0)::int
		 FROM ai_token_usage
		 WHERE user_id = $1 AND created_at >= date_trunc('day', NOW())`, userID).Scan(&total)
	return total, err
}

func (r *AITokenUsageRepository) DailySessionCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT session_id)::int
		 FROM ai_token_usage
		 WHERE user_id = $1 AND created_at >= date_trunc('day', NOW())
		   AND session_id IS NOT NULL`, userID).Scan(&count)
	return count, err
}

func (r *AITokenUsageRepository) DailyPlatformCost(ctx context.Context) (string, error) {
	var cost string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(cost::numeric), 0)::text
		 FROM ai_token_usage
		 WHERE paid_by = 'system' AND created_at >= date_trunc('day', NOW())`).Scan(&cost)
	return cost, err
}

func (r *AITokenUsageRepository) MonthlySummary(ctx context.Context, userID uuid.UUID) (map[string]int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT feature, SUM(input_tokens + output_tokens)::int AS total
		 FROM ai_token_usage
		 WHERE user_id = $1 AND created_at >= date_trunc('month', NOW())
		 GROUP BY feature ORDER BY total DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int)
	for rows.Next() {
		var feat string
		var total int
		if err := rows.Scan(&feat, &total); err != nil {
			return nil, err
		}
		out[feat] = total
	}
	return out, rows.Err()
}

func scanTokenUsageRows(rows interface{ Next() bool; Scan(...interface{}) error; Err() error }) ([]*AITokenUsage, error) {
	var out []*AITokenUsage
	for rows.Next() {
		var u AITokenUsage
		if err := rows.Scan(&u.ID, &u.UserID, &u.WalletTransactionID, &u.PaidBy, &u.ProviderID, &u.ModelName, &u.Feature, &u.InputTokens, &u.OutputTokens, &u.Cost, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}
