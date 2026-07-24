package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

type TemplateRow struct {
	ID          uuid.UUID
	UserID      *uuid.UUID // nil for system strategies (is_system=true)
	Name        string
	Description string
	Code        string // MQL source code (ADR-0023: single source of truth)
	StrategyID  *uuid.UUID // FK to imported_strategies.id (nullable for legacy/system templates)
	Status      string
	Parameters  []byte
	IsPublic    bool
	IsSystem    bool
	Tags        []string
	UseCount    int32
	I18n        []byte     // JSONB — parameter label translations
	Flag        string     // "" | "flagged" | "disabled" | "archived"
	FlagReason  string
	FlaggedBy   *uuid.UUID // admin who flagged
	FlaggedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *StrategySvc) ListTemplates(ctx context.Context, userID uuid.UUID) ([]TemplateRow, error) {
	rows, err := s.pg.Query(ctx,
		`SELECT id, user_id, name, COALESCE(description, ''), COALESCE(code, ''), strategy_id, COALESCE(status, ''), parameters, i18n, is_public, is_system, tags, use_count, COALESCE(flag, ''), COALESCE(flag_reason, ''), flagged_by, flagged_at, created_at, updated_at
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
		`SELECT id, user_id, name, COALESCE(description, ''), COALESCE(code, ''), strategy_id, COALESCE(status, ''), parameters, i18n, is_public, is_system, tags, use_count, COALESCE(flag, ''), COALESCE(flag_reason, ''), flagged_by, flagged_at, created_at, updated_at
		 FROM strategy_templates WHERE id = $1 AND (user_id = $2 OR is_public = true OR is_system = true)`, id, userID,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.Description, &t.Code, &t.StrategyID, &t.Status, &t.Parameters, &t.I18n, &t.IsPublic, &t.IsSystem, &t.Tags, &t.UseCount, &t.Flag, &t.FlagReason, &t.FlaggedBy, &t.FlaggedAt, &t.CreatedAt, &t.UpdatedAt)
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
	// Empty byte slice is not valid JSON; nil lets the column default ('{}'::jsonb) take effect.
	if len(t.I18n) == 0 {
		t.I18n = nil
	}
	_, err := s.pg.Exec(ctx,
		`INSERT INTO strategy_templates (id, user_id, name, description, code, strategy_id, status, parameters, i18n, is_public, is_system, tags, use_count, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		t.ID, t.UserID, t.Name, t.Description, t.Code, t.StrategyID, t.Status, t.Parameters, t.I18n, t.IsPublic, t.IsSystem, t.Tags, t.UseCount, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("CreateTemplate: %w", err)
	}
	return nil
}

func (s *StrategySvc) UpdateTemplate(ctx context.Context, t *TemplateRow) error {
	t.UpdatedAt = time.Now()
	// Empty byte slice is not valid JSON; nil lets the column default ('{}'::jsonb) take effect.
	if len(t.I18n) == 0 {
		t.I18n = nil
	}
	_, err := s.pg.Exec(ctx,
		`UPDATE strategy_templates SET name=$2, description=$3, code=$4, strategy_id=$5, status=$6, parameters=$7, i18n=$8, is_public=$9, tags=$10, updated_at=$11 WHERE id=$1 AND user_id=$12`,
		t.ID, t.Name, t.Description, t.Code, t.StrategyID, t.Status, t.Parameters, t.I18n, t.IsPublic, t.Tags, t.UpdatedAt, t.UserID)
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

func (s *StrategySvc) UnpublishUserTemplate(ctx context.Context, id, userID uuid.UUID) error {
	ct, err := s.pg.Exec(ctx,
		`UPDATE strategy_templates SET is_public=false, updated_at=$3 WHERE id=$1 AND user_id=$2 AND is_system=false`,
		id, userID, time.Now())
	if err != nil {
		return fmt.Errorf("UnpublishUserTemplate: %w", err)
	}
	if ct.RowsAffected() == 0 {
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

// StrategyCardRow is a denormalized row for Gallery card display (ADR-0027).
type StrategyCardRow struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Name            string
	Description     string
	Tags            []string
	IsSystem        bool
	IsPublic        bool
	UseCount        int32
	CreatedAt       time.Time
	Sparkline       []string // equity curve from latest successful backtest
	WinRate         string
	MaxDrawdown     string
	ProfitFactor    string
	SharpeRatio     string
	RunningSchedules int32
	BacktestRunID   *uuid.UUID
	IsMarketplacePublished bool // H3: true if has active marketplace listing
}

// ListStrategyCardsParams controls filtering, sorting, and searching.
type ListStrategyCardsParams struct {
	Filter string // "all" | "mine" | "preset"
	Sort   string // "recent" | "return" | "risk" | "usage"
	Search string // name/description ILIKE
	Limit  int    // page size (default 50, max 200)
	Offset int    // page offset
}

// ListStrategyCards returns templates with aggregated backtest KPIs and schedule counts.
// Uses 3 batched queries to avoid N+1 (ADR-0027 §4.7). Protobuf parsing in Go.
func (s *StrategySvc) ListStrategyCards(ctx context.Context, userID uuid.UUID, params ListStrategyCardsParams) ([]StrategyCardRow, int, error) {
	// Apply pagination defaults
	if params.Limit <= 0 {
		params.Limit = 50
	}
	if params.Limit > 200 {
		params.Limit = 200
	}
	// Build WHERE clause
	where := "WHERE status != 'canceled'"
	args := []any{userID}
	argIdx := 2
	switch params.Filter {
	case "mine":
		where += " AND user_id = $1 AND NOT is_system"
	case "preset":
		where += " AND is_system = true"
	default: // "all"
		where += " AND (user_id = $1 OR is_public = true OR is_system = true)"
	}
	if params.Search != "" {
		where += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+params.Search+"%")
		argIdx++
	}
	// Build ORDER BY clause (SQL-side for recent/usage; Go-side for return/risk after KPI parse)
	sqlOrder := "created_at DESC"
	switch params.Sort {
	case "usage":
		sqlOrder = "use_count DESC"
	}
	// Count total matching rows (for pagination)
	var total int
	countQ := `SELECT count(*) FROM strategy_templates ` + where
	if err := s.pg.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("ListStrategyCards count: %w", err)
	}
	// Add LIMIT/OFFSET
	limitIdx := argIdx
	offsetIdx := argIdx + 1
	args = append(args, params.Limit, params.Offset)
	rows, err := s.pg.Query(ctx,
		`SELECT id, user_id, name, COALESCE(description, ''), tags, is_system, is_public, use_count, created_at
		 FROM strategy_templates
		 `+where+`
		 ORDER BY `+sqlOrder+`
		 LIMIT $`+strconv.Itoa(limitIdx)+` OFFSET $`+strconv.Itoa(offsetIdx), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListStrategyCards templates: %w", err)
	}
	defer rows.Close()
	var out []StrategyCardRow
	tids := make([]uuid.UUID, 0)
	for rows.Next() {
		var r StrategyCardRow
		if err := rows.Scan(&r.ID, &r.UserID, &r.Name, &r.Description, &r.Tags, &r.IsSystem, &r.IsPublic, &r.UseCount, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("ListStrategyCards scan: %w", err)
		}
		out = append(out, r)
		tids = append(tids, r.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(out) == 0 {
		return out, total, nil
	}
	// Batch: latest successful backtest per template
	btRows, err := s.pg.Query(ctx,
		`SELECT DISTINCT ON (template_id) template_id, id, proto_response
		 FROM backtest_runs WHERE template_id = ANY($1) AND status = 'succeeded'
		 ORDER BY template_id, created_at DESC`, tids)
	if err != nil {
		return nil, 0, fmt.Errorf("ListStrategyCards backtests: %w", err)
	}
	defer btRows.Close()
	type btInfo struct{ runID uuid.UUID; raw []byte }
	btMap := make(map[uuid.UUID]btInfo)
	for btRows.Next() {
		var tid uuid.UUID
		var info btInfo
		if err := btRows.Scan(&tid, &info.runID, &info.raw); err != nil {
			return nil, 0, fmt.Errorf("ListStrategyCards bt scan: %w", err)
		}
		btMap[tid] = info
	}
	if err := btRows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListStrategyCards backtests: %w", err)
	}
	// Batch: active schedule counts
	scRows, err := s.pg.Query(ctx,
		`SELECT template_id, COUNT(*)::int FROM strategy_schedules
		 WHERE template_id = ANY($1) AND is_active = true GROUP BY template_id`, tids)
	if err != nil {
		return nil, 0, fmt.Errorf("ListStrategyCards schedules: %w", err)
	}
	defer scRows.Close()
	schedMap := make(map[uuid.UUID]int32)
	for scRows.Next() {
		var tid uuid.UUID
		var n int32
		if err := scRows.Scan(&tid, &n); err != nil {
			return nil, 0, fmt.Errorf("ListStrategyCards sched scan: %w", err)
		}
		schedMap[tid] = n
	}
	if err := scRows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListStrategyCards schedules: %w", err)
	}
	// Batch: marketplace published status (H3)
	marketRows, err := s.pg.Query(ctx,
		`SELECT strategy_id FROM marketplace_strategies
		 WHERE strategy_id = ANY($1) AND status = 'published'`, tids)
	if err != nil {
		return nil, 0, fmt.Errorf("ListStrategyCards marketplace: %w", err)
	}
	defer marketRows.Close()
	marketMap := make(map[uuid.UUID]bool)
	for marketRows.Next() {
		var tid uuid.UUID
		if err := marketRows.Scan(&tid); err != nil {
			return nil, 0, fmt.Errorf("ListStrategyCards market scan: %w", err)
		}
		marketMap[tid] = true
	}
	if err := marketRows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListStrategyCards marketplace: %w", err)
	}
	// Assemble: parse proto_response for KPIs + sparkline
	for i := range out {
		tid := out[i].ID
		out[i].RunningSchedules = schedMap[tid]
		out[i].IsMarketplacePublished = marketMap[tid]
		info, ok := btMap[tid]
		if !ok || len(info.raw) == 0 {
			continue
		}
		out[i].BacktestRunID = &info.runID
		var resp antv1.ExecuteBacktestResponse
		if err := proto.Unmarshal(info.raw, &resp); err != nil {
			continue
		}
		out[i].Sparkline = resp.GetEquityCurve()
		m := resp.GetMetrics()
		out[i].WinRate = m.GetWinRate()
		out[i].MaxDrawdown = m.GetMaxDrawdown()
		out[i].ProfitFactor = m.GetProfitFactor()
		out[i].SharpeRatio = m.GetSharpeRatio()
	}
	// Go-side sort for KPI-based orderings (values come from proto_response, not SQL columns)
	switch params.Sort {
	case "return":
		sort.SliceStable(out, func(i, j int) bool {
			return parseCardFloat(out[i].ProfitFactor) > parseCardFloat(out[j].ProfitFactor)
		})
	case "risk":
		sort.SliceStable(out, func(i, j int) bool {
			return parseCardFloat(out[i].MaxDrawdown) < parseCardFloat(out[j].MaxDrawdown)
		})
	}
	return out, total, nil
}

func parseCardFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func scanTemplateRows(rows pgx.Rows) ([]TemplateRow, error) {
	var out []TemplateRow
	for rows.Next() {
		var t TemplateRow
		err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Description, &t.Code, &t.StrategyID, &t.Status, &t.Parameters, &t.I18n, &t.IsPublic, &t.IsSystem, &t.Tags, &t.UseCount, &t.Flag, &t.FlagReason, &t.FlaggedBy, &t.FlaggedAt, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan template row: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
