package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AIAgentDefinitionRow 自 060 起 user-scoped：去掉 profile_id 与 locator 字符串
// model_profile_id，改为直接保存 (provider_id, model_override)，
// provider_id 直连 system_ai_configs.provider_id。
type AIAgentDefinitionRow struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	AgentKey      string
	Type          string
	Name          string
	Identity      string
	InputHint     string
	Enabled       bool
	Position      int32
	ProviderID    string // 空 = 由调用方按角色 fallback 取一个 SystemAI provider
	ModelOverride string // 空 = 用 provider 的 default_model
}

type AIAgentDefinitionRepository struct {
	db *pgxpool.Pool
}

func NewAIAgentDefinitionRepository(db *pgxpool.Pool) *AIAgentDefinitionRepository {
	return &AIAgentDefinitionRepository{db: db}
}

const aiAgentSelectCols = `
	id, user_id, agent_key, type, name, identity, input_hint, enabled,
	position, provider_id, model_override
`

// ListByUser 返回用户的所有 Agent，按 (position, name) 排序。
func (r *AIAgentDefinitionRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*AIAgentDefinitionRow, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+aiAgentSelectCols+`
		   FROM ai_agent_definitions
		  WHERE user_id = $1
		  ORDER BY position ASC, name ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*AIAgentDefinitionRow, 0, 8)
	for rows.Next() {
		row := &AIAgentDefinitionRow{}
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.AgentKey, &row.Type, &row.Name,
			&row.Identity, &row.InputHint, &row.Enabled, &row.Position,
			&row.ProviderID, &row.ModelOverride,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ReplaceByUser 事务式 DELETE + INSERT，用提交的 list 整体覆盖该用户的 agents。
// 调用方需保证 agents 中每行 UserID 已设置正确。
func (r *AIAgentDefinitionRepository) ReplaceByUser(ctx context.Context, userID uuid.UUID, agents []*AIAgentDefinitionRow) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM ai_agent_definitions WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, a := range agents {
		id := a.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO ai_agent_definitions (
				id, user_id, agent_key, type, name, identity, input_hint,
				enabled, position, provider_id, model_override
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			id, userID, a.AgentKey, a.Type, a.Name, a.Identity, a.InputHint,
			a.Enabled, a.Position, a.ProviderID, a.ModelOverride,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
