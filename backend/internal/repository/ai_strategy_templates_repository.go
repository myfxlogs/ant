package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AIStrategyTemplate is a system-seeded strategy template for AI generation.
type AIStrategyTemplate struct {
	ID             uuid.UUID       `db:"id"`
	Category       string          `db:"category"`
	Name           string          `db:"name"`
	DescriptionZh  string          `db:"description_zh"`
	PythonSkeleton string          `db:"python_skeleton"`
	ParameterSlots json.RawMessage `db:"parameter_slots"`
	RiskLevel      string          `db:"risk_level"`
	IsActive       bool            `db:"is_active"`
	CreatedAt      time.Time       `db:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at"`
}

// ParameterSlot describes a tunable parameter in a strategy template.
type ParameterSlot struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Default       float64 `json:"default"`
	Min           float64 `json:"min"`
	Max           float64 `json:"max"`
	Step          float64 `json:"step"`
	DescriptionZh string  `json:"description_zh"`
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
		`SELECT id, category, name, description_zh, python_skeleton, parameter_slots, risk_level, is_active, created_at, updated_at
		 FROM ai_strategy_templates WHERE is_active = true ORDER BY category, name`)
	if err != nil {
		return nil, fmt.Errorf("list platform strategies: %w", err)
	}
	defer rows.Close()

	var out []AIStrategyTemplate
	for rows.Next() {
		var s AIStrategyTemplate
		if err := rows.Scan(&s.ID, &s.Category, &s.Name, &s.DescriptionZh,
			&s.PythonSkeleton, &s.ParameterSlots, &s.RiskLevel, &s.IsActive,
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
		`SELECT id, category, name, description_zh, python_skeleton, parameter_slots, risk_level, is_active, created_at, updated_at
		 FROM ai_strategy_templates WHERE id = $1 AND is_active = true`, id,
	).Scan(&s.ID, &s.Category, &s.Name, &s.DescriptionZh,
		&s.PythonSkeleton, &s.ParameterSlots, &s.RiskLevel, &s.IsActive,
		&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get platform strategy: %w", err)
	}
	return &s, nil
}

// ListByCategory returns active strategies for a given category.
func (r *AIStrategyTemplatesRepository) ListByCategory(ctx context.Context, category string) ([]AIStrategyTemplate, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, category, name, description_zh, python_skeleton, parameter_slots, risk_level, is_active, created_at, updated_at
		 FROM ai_strategy_templates WHERE category = $1 AND is_active = true ORDER BY name`, category)
	if err != nil {
		return nil, fmt.Errorf("list by category: %w", err)
	}
	defer rows.Close()

	var out []AIStrategyTemplate
	for rows.Next() {
		var s AIStrategyTemplate
		if err := rows.Scan(&s.ID, &s.Category, &s.Name, &s.DescriptionZh,
			&s.PythonSkeleton, &s.ParameterSlots, &s.RiskLevel, &s.IsActive,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
