package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetTemplateDetail fetches a template with user email (admin bypass — no ownership check).
func (s *StrategySvc) GetTemplateDetail(ctx context.Context, id uuid.UUID) (*TemplateRow, string, error) {
	var t TemplateRow
	var userEmail string
	err := s.pg.QueryRow(ctx,
		`SELECT st.id, st.user_id, st.name, st.description, st.code, st.status, st.parameters, st.is_public, st.is_system,
		        st.tags, st.use_count, st.flag, st.flag_reason, st.flagged_by, st.flagged_at, st.created_at, st.updated_at,
		        COALESCE(u.email, '')
		 FROM strategy_templates st
		 LEFT JOIN users u ON st.user_id = u.id
		 WHERE st.id = $1`, id,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.Description, &t.Code, &t.Status, &t.Parameters,
		&t.IsPublic, &t.IsSystem, &t.Tags, &t.UseCount,
		&t.Flag, &t.FlagReason, &t.FlaggedBy, &t.FlaggedAt, &t.CreatedAt, &t.UpdatedAt,
		&userEmail)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrTemplateNotFound
		}
		return nil, "", fmt.Errorf("GetTemplateDetail: %w", err)
	}
	return &t, userEmail, nil
}

// ── System strategy CRUD ──

// SystemStrategyRow is a lightweight row for admin system strategy management.
type SystemStrategyRow struct {
	ID          uuid.UUID
	Name        string
	Description string
	Code        string
	Tags        []string
	UseCount    int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *StrategySvc) ListSystemStrategies(ctx context.Context) ([]SystemStrategyRow, error) {
	rows, err := s.pg.Query(ctx,
		`SELECT id, name, description, code, tags, use_count, created_at, updated_at
		 FROM strategy_templates WHERE is_system = true AND status != 'canceled' ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list system strategies: %w", err)
	}
	defer rows.Close()
	var out []SystemStrategyRow
	for rows.Next() {
		var r SystemStrategyRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Code, &r.Tags, &r.UseCount, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan system strategy: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *StrategySvc) CreateSystemStrategy(ctx context.Context, name, description, code string, tags []string) (*TemplateRow, error) {
	now := time.Now()
	if tags == nil {
		tags = []string{}
	}
	t := &TemplateRow{
		ID:          uuid.New(),
		UserID:      nil, // system strategy
		Name:        name,
		Description: description,
		Code:        code,
		Status:      "published",
		IsPublic:    true,
		IsSystem:    true,
		Tags:        tags,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := s.pg.Exec(ctx,
		`INSERT INTO strategy_templates (id, user_id, name, description, code, status, parameters, i18n, is_public, is_system, tags, use_count, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		t.ID, t.UserID, t.Name, t.Description, t.Code, t.Status, []byte("[]"), nil, t.IsPublic, t.IsSystem, t.Tags, t.UseCount, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create system strategy: %w", err)
	}
	return t, nil
}

func (s *StrategySvc) UpdateSystemStrategy(ctx context.Context, id uuid.UUID, name, description, code *string, tags []string) (*TemplateRow, error) {
	// Fetch existing
	var t TemplateRow
	err := s.pg.QueryRow(ctx,
		`SELECT id, user_id, name, description, code, status, parameters, i18n, is_public, is_system, tags, use_count, flag, flag_reason, flagged_by, flagged_at, created_at, updated_at
		 FROM strategy_templates WHERE id = $1 AND is_system = true`, id,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.Description, &t.Code, &t.Status, &t.Parameters, &t.I18n, &t.IsPublic, &t.IsSystem, &t.Tags, &t.UseCount, &t.Flag, &t.FlagReason, &t.FlaggedBy, &t.FlaggedAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("get system strategy: %w", err)
	}
	if name != nil {
		t.Name = *name
	}
	if description != nil {
		t.Description = *description
	}
	if code != nil {
		t.Code = *code
	}
	if tags != nil {
		t.Tags = tags
	}
	t.UpdatedAt = time.Now()
	_, err = s.pg.Exec(ctx,
		`UPDATE strategy_templates SET name=$2, description=$3, code=$4, tags=$5, updated_at=$6 WHERE id=$1 AND is_system=true`,
		t.ID, t.Name, t.Description, t.Code, t.Tags, t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update system strategy: %w", err)
	}
	return &t, nil
}

func (s *StrategySvc) DeleteSystemStrategy(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pg.Exec(ctx,
		`UPDATE strategy_templates SET status='canceled', updated_at=$2 WHERE id=$1 AND is_system=true AND status!='canceled'`,
		id, time.Now())
	if err != nil {
		return fmt.Errorf("delete system strategy: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

// ── Admin oversight (all strategies) ──

// AllStrategyRow is a flattened row for admin strategy oversight.
type AllStrategyRow struct {
	ID            uuid.UUID
	Name          string
	UserID        *uuid.UUID
	UserEmail     string
	Status        string
	IsSystem      bool
	IsPublic      bool
	Flag          string
	FlagReason    string
	FlaggedBy     *uuid.UUID
	ScheduleCount int32
	UseCount      int32
	Tags          []string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ListAllStrategiesParams struct {
	Page     int32
	PageSize int32
	Search   *string
	UserID   *uuid.UUID
	Flag     *string // "flagged" | "disabled" | "archived" | "" (all active)
}

func (s *StrategySvc) ListAllStrategies(ctx context.Context, params ListAllStrategiesParams) ([]AllStrategyRow, int32, error) {
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	// Build WHERE clauses
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if params.Search != nil && *params.Search != "" {
		where += fmt.Sprintf(" AND (st.name ILIKE $%d OR st.description ILIKE $%d)", argIdx, argIdx+1)
		search := "%" + *params.Search + "%"
		args = append(args, search, search)
		argIdx += 2
	}
	if params.UserID != nil {
		where += fmt.Sprintf(" AND st.user_id = $%d", argIdx)
		args = append(args, *params.UserID)
		argIdx++
	}
	if params.Flag != nil {
		if *params.Flag == "" {
			where += " AND st.flag = ''"
		} else {
			where += fmt.Sprintf(" AND st.flag = $%d", argIdx)
			args = append(args, *params.Flag)
			argIdx++
		}
	}

	// Count
	var total int32
	countQuery := fmt.Sprintf("SELECT count(*) FROM strategy_templates st %s", where)
	if err := s.pg.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count all strategies: %w", err)
	}

	// Query
	offset := (params.Page - 1) * params.PageSize
	query := fmt.Sprintf(
		`SELECT st.id, st.name, st.user_id, COALESCE(u.email, ''), st.status, st.is_system, st.is_public,
		        st.flag, st.flag_reason, st.flagged_by, st.use_count, st.tags, st.created_at, st.updated_at,
		        COALESCE(ss.cnt, 0) as schedule_count
		 FROM strategy_templates st
		 LEFT JOIN users u ON st.user_id = u.id
		 LEFT JOIN (SELECT template_id, count(*) as cnt FROM strategy_schedules GROUP BY template_id) ss ON ss.template_id = st.id
		 %s ORDER BY st.created_at DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, params.PageSize, offset)

	rows, err := s.pg.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list all strategies: %w", err)
	}
	defer rows.Close()

	var out []AllStrategyRow
	for rows.Next() {
		var r AllStrategyRow
		if err := rows.Scan(&r.ID, &r.Name, &r.UserID, &r.UserEmail, &r.Status, &r.IsSystem, &r.IsPublic,
			&r.Flag, &r.FlagReason, &r.FlaggedBy, &r.UseCount, &r.Tags, &r.CreatedAt, &r.UpdatedAt,
			&r.ScheduleCount); err != nil {
			return nil, 0, fmt.Errorf("scan all strategy: %w", err)
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

// ── Compliance actions ──

func (s *StrategySvc) FlagTemplate(ctx context.Context, id uuid.UUID, reason string, adminID uuid.UUID) error {
	now := time.Now()
	ct, err := s.pg.Exec(ctx,
		`UPDATE strategy_templates SET flag='flagged', flag_reason=$2, flagged_by=$3, flagged_at=$4 WHERE id=$1`,
		id, reason, adminID, now)
	if err != nil {
		return fmt.Errorf("flag template: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (s *StrategySvc) UnflagTemplate(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pg.Exec(ctx,
		`UPDATE strategy_templates SET flag='', flag_reason='', flagged_by=NULL, flagged_at=NULL WHERE id=$1 AND flag='flagged'`,
		id)
	if err != nil {
		return fmt.Errorf("unflag template: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (s *StrategySvc) UnpublishTemplate(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pg.Exec(ctx,
		`UPDATE strategy_templates SET is_public=false WHERE id=$1 AND is_system=false`, id)
	if err != nil {
		return fmt.Errorf("unpublish template: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (s *StrategySvc) PublishTemplate(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pg.Exec(ctx,
		`UPDATE strategy_templates SET is_public=true WHERE id=$1 AND is_system=false`, id)
	if err != nil {
		return fmt.Errorf("publish template: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (s *StrategySvc) DisableTemplate(ctx context.Context, id uuid.UUID) error {
	// Stop all active schedules for this template, then flag as disabled.
	_, err := s.pg.Exec(ctx,
		`UPDATE strategy_schedules SET is_active = false, updated_at = $2 WHERE template_id = $1 AND is_active = true`, id, time.Now())
	if err != nil {
		return fmt.Errorf("disable schedules: %w", err)
	}
	ct, err := s.pg.Exec(ctx,
		`UPDATE strategy_templates SET flag='disabled' WHERE id=$1 AND is_system=false`, id)
	if err != nil {
		return fmt.Errorf("disable template: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (s *StrategySvc) EnableTemplate(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pg.Exec(ctx,
		`UPDATE strategy_templates SET flag='', flag_reason='', flagged_by=NULL, flagged_at=NULL WHERE id=$1 AND is_system=false AND flag='disabled'`, id)
	if err != nil {
		return fmt.Errorf("enable template: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (s *StrategySvc) ArchiveTemplate(ctx context.Context, id uuid.UUID) error {
	// Soft-delete: stop schedules + mark as archived.
	_, err := s.pg.Exec(ctx,
		`UPDATE strategy_schedules SET is_active = false, updated_at = $2 WHERE template_id = $1 AND is_active = true`, id, time.Now())
	if err != nil {
		return fmt.Errorf("archive schedules: %w", err)
	}
	ct, err := s.pg.Exec(ctx,
		`UPDATE strategy_templates SET flag='archived', status='canceled' WHERE id=$1 AND is_system=false`, id)
	if err != nil {
		return fmt.Errorf("archive template: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}
