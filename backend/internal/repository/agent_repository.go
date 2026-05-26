package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var ErrAgentTokenNotFound = errors.New("agent token not found")

type AgentToken struct {
	ID                 uuid.UUID  `db:"id"`
	UserID             uuid.UUID  `db:"user_id"`
	Name               string     `db:"name"`
	TokenPrefix        string     `db:"token_prefix"`
	TokenHash          string     `db:"token_hash"`
	Scopes             []string   `db:"scopes"`
	AccountAllowlist   []string   `db:"account_allowlist"`
	SymbolAllowlist    []string   `db:"symbol_allowlist"`
	PaperOnly          bool       `db:"paper_only"`
	RateLimitPerMinute int        `db:"rate_limit_per_min"`
	ExpiresAt          *time.Time `db:"expires_at"`
	Status             string     `db:"status"`
	LastUsedAt         *time.Time `db:"last_used_at"`
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
}

type AgentAuditLog struct {
	ID              uuid.UUID `db:"id"`
	UserID          uuid.UUID `db:"user_id"`
	AgentTokenID    uuid.UUID `db:"agent_token_id"`
	AgentName       string    `db:"agent_name"`
	RPCService      string    `db:"rpc_service"`
	RPCMethod       string    `db:"rpc_method"`
	Scope           string    `db:"scope"`
	StatusCode      string    `db:"status_code"`
	IdempotencyKey  string    `db:"idempotency_key"`
	RiskDecision    string    `db:"risk_decision"`
	RequestSummary  string    `db:"request_summary"`
	ResponseSummary string    `db:"response_summary"`
	DurationMS      int64     `db:"duration_ms"`
	CreatedAt       time.Time `db:"created_at"`
}

type AgentRepository struct {
	db *sqlx.DB
}

func NewAgentRepository(db *sqlx.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

func NewAgentPlaintextToken() (string, string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", "", err
	}
	plain := "agt_" + base64.RawURLEncoding.EncodeToString(buf)
	prefix := plain
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}
	sum := sha256.Sum256([]byte(plain))
	return plain, prefix, hex.EncodeToString(sum[:]), nil
}

func (r *AgentRepository) CreateToken(ctx context.Context, token *AgentToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	now := time.Now().UTC()
	if token.CreatedAt.IsZero() {
		token.CreatedAt = now
	}
	if token.UpdatedAt.IsZero() {
		token.UpdatedAt = now
	}
	if token.Status == "" {
		token.Status = "active"
	}
	if token.RateLimitPerMinute <= 0 {
		token.RateLimitPerMinute = 60
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_tokens (id, user_id, name, token_prefix, token_hash, scopes, account_allowlist, symbol_allowlist, paper_only, rate_limit_per_min, expires_at, status, last_used_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
	`, token.ID, token.UserID, token.Name, token.TokenPrefix, token.TokenHash, token.Scopes, token.AccountAllowlist, token.SymbolAllowlist, token.PaperOnly, token.RateLimitPerMinute, token.ExpiresAt, token.Status, token.LastUsedAt, token.CreatedAt, token.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create agent token: %w", err)
	}
	return nil
}

func (r *AgentRepository) ListTokens(ctx context.Context, userID uuid.UUID) ([]AgentToken, error) {
	var tokens []AgentToken
	err := r.db.SelectContext(ctx, &tokens, `SELECT * FROM agent_tokens WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	return tokens, err
}

func (r *AgentRepository) RevokeToken(ctx context.Context, userID, tokenID uuid.UUID) (*AgentToken, error) {
	var token AgentToken
	err := r.db.GetContext(ctx, &token, `UPDATE agent_tokens SET status = 'revoked', updated_at = now() WHERE id = $1 AND user_id = $2 RETURNING *`, tokenID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAgentTokenNotFound
	}
	return &token, err
}

func (r *AgentRepository) ListAudit(ctx context.Context, userID uuid.UUID, tokenID *uuid.UUID, limit, offset int) ([]AgentAuditLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var rows []AgentAuditLog
	if tokenID != nil {
		err := r.db.SelectContext(ctx, &rows, `SELECT * FROM agent_audit_logs WHERE user_id = $1 AND agent_token_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`, userID, *tokenID, limit, offset)
		return rows, err
	}
	err := r.db.SelectContext(ctx, &rows, `SELECT * FROM agent_audit_logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	return rows, err
}

func NormalizeAgentScopes(scopes []string) []string {
	out := make([]string, 0, len(scopes))
	seen := map[string]bool{}
	for _, scope := range scopes {
		s := strings.ToUpper(strings.TrimSpace(scope))
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
