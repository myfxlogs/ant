package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TemplateRow struct {
	ID          uuid.UUID
	UserID      *uuid.UUID // nil for system strategies (is_system=true)
	Name        string
	Description string
	Code        string
	Status      string
	Parameters  []byte
	IsPublic    bool
	IsSystem    bool
	Tags        []string
	UseCount    int32
	Flag        string     // "" | "flagged" | "disabled" | "archived"
	FlagReason  string
	FlaggedBy   *uuid.UUID // admin who flagged
	FlaggedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *StrategySvc) ListTemplates(ctx context.Context, userID uuid.UUID) ([]TemplateRow, error) {
	rows, err := s.pg.Query(ctx,
		`SELECT id, user_id, name, description, code, status, parameters, is_public, is_system, tags, use_count, flag, flag_reason, flagged_by, flagged_at, created_at, updated_at
		 FROM strategy_templates WHERE (user_id = $1 OR is_public = true OR is_system = true) AND status != 'canceled' ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()
	return scanTemplateRows(rows)
}

func (s *StrategySvc) GetTemplate(ctx context.Context, id, userID uuid.UUID) (*TemplateRow, error) {
	var t TemplateRow
	err := s.pg.QueryRow(ctx,
		`SELECT id, user_id, name, description, code, status, parameters, is_public, is_system, tags, use_count, flag, flag_reason, flagged_by, flagged_at, created_at, updated_at
		 FROM strategy_templates WHERE id = $1 AND (user_id = $2 OR is_public = true OR is_system = true)`, id, userID,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.Description, &t.Code, &t.Status, &t.Parameters, &t.IsPublic, &t.IsSystem, &t.Tags, &t.UseCount, &t.Flag, &t.FlagReason, &t.FlaggedBy, &t.FlaggedAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("GetTemplate: %w", err)
	}
	return &t, nil
}

func (s *StrategySvc) CreateTemplate(ctx context.Context, t *TemplateRow) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Tags == nil {
		t.Tags = []string{}
	}
	_, err := s.pg.Exec(ctx,
		`INSERT INTO strategy_templates (id, user_id, name, description, code, status, parameters, is_public, is_system, tags, use_count, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		t.ID, t.UserID, t.Name, t.Description, t.Code, t.Status, t.Parameters, t.IsPublic, t.IsSystem, t.Tags, t.UseCount, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("CreateTemplate: %w", err)
	}
	return nil
}

func (s *StrategySvc) UpdateTemplate(ctx context.Context, t *TemplateRow) error {
	t.UpdatedAt = time.Now()
	_, err := s.pg.Exec(ctx,
		`UPDATE strategy_templates SET name=$2, description=$3, code=$4, status=$5, parameters=$6, is_public=$7, tags=$8, updated_at=$9 WHERE id=$1 AND user_id=$10`,
		t.ID, t.Name, t.Description, t.Code, t.Status, t.Parameters, t.IsPublic, t.Tags, t.UpdatedAt, t.UserID)
	if err != nil {
		return fmt.Errorf("UpdateTemplate: %w", err)
	}
	return nil
}

func (s *StrategySvc) DeleteTemplate(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := s.pg.Exec(ctx, `DELETE FROM strategy_templates WHERE id=$1 AND user_id=$2 AND is_system=false`, id, userID)
	if err != nil {
		return fmt.Errorf("DeleteTemplate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (s *StrategySvc) SetTemplateStatus(ctx context.Context, id, userID uuid.UUID, status string) error {
	ct, err := s.pg.Exec(ctx, `UPDATE strategy_templates SET status=$2, updated_at=$3 WHERE id=$1 AND user_id=$4`, id, status, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("SetTemplateStatus: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func scanTemplateRows(rows pgx.Rows) ([]TemplateRow, error) {
	var out []TemplateRow
	for rows.Next() {
		var t TemplateRow
		err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Description, &t.Code, &t.Status, &t.Parameters, &t.IsPublic, &t.IsSystem, &t.Tags, &t.UseCount, &t.Flag, &t.FlagReason, &t.FlaggedBy, &t.FlaggedAt, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan template row: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
