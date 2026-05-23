package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"anttrader/internal/model"
)

var (
	ErrTemplateNotFound    = errors.New("strategy template not found")
	ErrTemplateIsSystem    = errors.New("system template cannot be modified or deleted")
)

type StrategyTemplateRepository struct {
	db *sqlx.DB
}

func NewStrategyTemplateRepository(db *sqlx.DB) *StrategyTemplateRepository {
	return &StrategyTemplateRepository{db: db}
}

func (r *StrategyTemplateRepository) SetStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE strategy_templates SET status = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, status, time.Now())
	return err
}

func (r *StrategyTemplateRepository) Create(ctx context.Context, template *model.StrategyTemplate) error {
	query := `
		INSERT INTO strategy_templates (
			id, user_id, name, description, code, status, parameters,
			i18n, is_public, is_system, tags, use_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	now := time.Now()
	if template.ID == uuid.Nil {
		template.ID = uuid.New()
	}
	template.CreatedAt = now
	template.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		template.ID, template.UserID, template.Name, template.Description,
		template.Code, template.Status, template.Parameters, template.I18n,
		template.IsPublic, template.IsSystem, template.Tags,
		template.UseCount, template.CreatedAt, template.UpdatedAt,
	)

	return err
}

func (r *StrategyTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.StrategyTemplate, error) {
	query := `SELECT * FROM strategy_templates WHERE id = $1`
	var template model.StrategyTemplate
	err := r.db.GetContext(ctx, &template, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}
	return &template, nil
}

func (r *StrategyTemplateRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.StrategyTemplate, error) {
	query := `SELECT * FROM strategy_templates WHERE user_id = $1 AND status <> $2 ORDER BY created_at DESC`
	var templates []*model.StrategyTemplate
	err := r.db.SelectContext(ctx, &templates, query, userID, model.StrategyTemplateStatusCanceled)
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *StrategyTemplateRepository) GetPublicTemplates(ctx context.Context, limit, offset int) ([]*model.StrategyTemplate, error) {
	query := `
		SELECT * FROM strategy_templates 
		WHERE is_public = true AND status = $3
		ORDER BY use_count DESC, created_at DESC 
		LIMIT $1 OFFSET $2`
	var templates []*model.StrategyTemplate
	err := r.db.SelectContext(ctx, &templates, query, limit, offset, model.StrategyTemplateStatusPublished)
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *StrategyTemplateRepository) Search(ctx context.Context, userID uuid.UUID, keyword string) ([]*model.StrategyTemplate, error) {
	query := `
		SELECT * FROM strategy_templates 
		WHERE user_id = $1 AND (name ILIKE $2 OR description ILIKE $2)
		ORDER BY created_at DESC`
	var templates []*model.StrategyTemplate
	err := r.db.SelectContext(ctx, &templates, query, userID, "%"+keyword+"%")
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *StrategyTemplateRepository) Update(ctx context.Context, template *model.StrategyTemplate) error {
	// NOTE: is_system is intentionally NOT updated here — it is controlled by
	// the seeder only. Normal user updates must never flip a row's is_system.
	query := `
		UPDATE strategy_templates SET
			name = $2, description = $3, code = $4, status = $5, parameters = $6,
			i18n = $7, is_public = $8, tags = $9, updated_at = $10
		WHERE id = $1`

	template.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		template.ID, template.Name, template.Description, template.Code, template.Status,
		template.Parameters, template.I18n, template.IsPublic, template.Tags, template.UpdatedAt,
	)

	return err
}

// Delete removes a user-owned template. System templates (is_system = true)
// are protected: attempts to delete them return ErrTemplateIsSystem so the
// caller can surface a clean error message to the user.
func (r *StrategyTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Use a single conditional DELETE that refuses to touch system rows. We
	// then disambiguate "not found" vs "is system" with a follow-up existence
	// check only when nothing was deleted, to keep the happy path a single RTT.
	query := `DELETE FROM strategy_templates WHERE id = $1 AND is_system = FALSE`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		var isSystem bool
		if err2 := r.db.GetContext(ctx, &isSystem,
			`SELECT is_system FROM strategy_templates WHERE id = $1`, id); err2 != nil {
			if errors.Is(err2, sql.ErrNoRows) {
				return ErrTemplateNotFound
			}
			return err2
		}
		if isSystem {
			return ErrTemplateIsSystem
		}
		return ErrTemplateNotFound
	}
	return nil
}

func (r *StrategyTemplateRepository) IncrementUseCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE strategy_templates SET use_count = use_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *StrategyTemplateRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM strategy_templates WHERE user_id = $1`
	var count int
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}
