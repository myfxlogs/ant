package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

func scanTokenUsageRows(rows interface {
	Next() bool
	Scan(...interface{}) error
	Err() error
}) ([]*AITokenUsage, error) {
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
