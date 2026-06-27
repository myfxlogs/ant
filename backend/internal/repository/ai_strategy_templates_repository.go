package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
)

// AIStrategyTemplate is a system-seeded strategy template for AI generation.
type AIStrategyTemplate struct {
	ID             uuid.UUID `db:"id"`
	Category       string    `db:"category"`
	Name           string    `db:"name"`
	DescriptionZh  string    `db:"description_zh"`
	CodeSkeleton string    `db:"code_skeleton"` // Go code skeleton for AI strategy generation
	ParameterSlots []byte    `db:"parameter_slots"` // proto binary TemplateParameterSlots
	RiskLevel      string    `db:"risk_level"`
	IsActive       bool      `db:"is_active"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// ParameterSlotsString returns a human-readable description of the parameter slots
// for AI prompt injection (legacy callers expecting string format).
func (t *AIStrategyTemplate) ParameterSlotsString() string {
	if len(t.ParameterSlots) == 0 {
		return ""
	}
	var slots antv1.TemplateParameterSlots
	if err := proto.Unmarshal(t.ParameterSlots, &slots); err != nil {
		return ""
	}
	var s string
	for _, p := range slots.GetSlots() {
		s += fmt.Sprintf("%s(%s): default=%.1f range=%.1f:%.1f:%.1f %s\n",
			p.GetName(), p.GetType(), p.GetDefaultValue(),
			p.GetMin(), p.GetMax(), p.GetStep(), p.GetDescription())
	}
	return s
}

// AIStrategyTemplatesRepository provides read access to platform strategy templates.
type AIStrategyTemplatesRepository struct {
	db *pgxpool.Pool
}

func NewAIStrategyTemplatesRepository(db *pgxpool.Pool) *AIStrategyTemplatesRepository {
	return &AIStrategyTemplatesRepository{db: db}
}

// ListActive returns all active platform strategies ordered by category.
func (r *AIStrategyTemplatesRepository) ListActive(ctx context.Context) ([]AIStrategyTemplate, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, category, name, description_zh, code_skeleton, parameter_slots, risk_level, is_active, created_at, updated_at
		 FROM ai_strategy_templates WHERE is_active = true ORDER BY category, name`)
	if err != nil {
		return nil, fmt.Errorf("list platform strategies: %w", err)
	}
	defer rows.Close()

	var out []AIStrategyTemplate
	for rows.Next() {
		var s AIStrategyTemplate
		if err := rows.Scan(&s.ID, &s.Category, &s.Name, &s.DescriptionZh,
			&s.CodeSkeleton, &s.ParameterSlots, &s.RiskLevel, &s.IsActive,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan platform strategy: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetByID returns a single platform strategy by ID.
func (r *AIStrategyTemplatesRepository) GetByID(ctx context.Context, id uuid.UUID) (*AIStrategyTemplate, error) {
	var s AIStrategyTemplate
	err := r.db.QueryRow(ctx,
		`SELECT id, category, name, description_zh, code_skeleton, parameter_slots, risk_level, is_active, created_at, updated_at
		 FROM ai_strategy_templates WHERE id = $1 AND is_active = true`, id,
	).Scan(&s.ID, &s.Category, &s.Name, &s.DescriptionZh,
		&s.CodeSkeleton, &s.ParameterSlots, &s.RiskLevel, &s.IsActive,
		&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get platform strategy: %w", err)
	}
	return &s, nil
}

// ListByCategory returns active strategies for a given category.
func (r *AIStrategyTemplatesRepository) ListByCategory(ctx context.Context, category string) ([]AIStrategyTemplate, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, category, name, description_zh, code_skeleton, parameter_slots, risk_level, is_active, created_at, updated_at
		 FROM ai_strategy_templates WHERE category = $1 AND is_active = true ORDER BY name`, category)
	if err != nil {
		return nil, fmt.Errorf("list by category: %w", err)
	}
	defer rows.Close()

	var out []AIStrategyTemplate
	for rows.Next() {
		var s AIStrategyTemplate
		if err := rows.Scan(&s.ID, &s.Category, &s.Name, &s.DescriptionZh,
			&s.CodeSkeleton, &s.ParameterSlots, &s.RiskLevel, &s.IsActive,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
