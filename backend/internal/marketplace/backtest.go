package marketplace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"alphaforge/internal/backtest"
	"alphaforge/internal/repository"
)

// StartMarketBacktest creates a pending backtest run for a marketplace-published
// strategy. It verifies access rights, looks up the strategy code from
// strategy_templates, and inserts a pending backtest run for the worker to pick up.
func (s *Service) StartMarketBacktest(ctx context.Context, params StartBacktestParams) (string, error) {
	// 1. Permission check.
	allowed, err := s.CanAccessCode(ctx, params.UserID, params.StrategyID)
	if err != nil || !allowed {
		return "", fmt.Errorf("marketplace: access denied for strategy backtest")
	}

	// 2. Look up strategy code.
	var strategyCode string
	err = s.pg.QueryRow(ctx,
		`SELECT code FROM strategy_templates WHERE id::text = $1`,
		params.StrategyID,
	).Scan(&strategyCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("marketplace: strategy not found")
		}
		return "", fmt.Errorf("marketplace: lookup strategy: %w", err)
	}
	if strategyCode == "" {
		return "", fmt.Errorf("marketplace: strategy has no code")
	}

	// 3. Build and apply defaults via shared package.
	runID := uuid.New()
	templateUUID, _ := uuid.Parse(params.StrategyID)
	userUUID, _ := uuid.Parse(params.UserID)
	strictTrue := true

	var fromTs, toTs *time.Time
	if params.StartDateMs > 0 {
		t := time.UnixMilli(params.StartDateMs).UTC()
		fromTs = &t
	}
	if params.EndDateMs > 0 {
		t := time.UnixMilli(params.EndDateMs).UTC()
		toTs = &t
	}

	run := &repository.BacktestRun{
		ID:             runID,
		UserID:         userUUID,
		Symbol:         params.Symbol,
		Timeframe:      params.Timeframe,
		Mode:           backtest.DefaultMode,
		Status:         "PENDING",
		StrategyCode:   &strategyCode,
		TemplateID:     &templateUUID,
		Commission:     &params.Commission,
		Slippage:       &params.Slippage,
		Leverage:       &params.Leverage,
		InitialCapital: &params.InitialCapital,
		TradeDirection: &params.TradeDirection,
		StrictMode:     &strictTrue,
		FromTs:         fromTs,
		ToTs:           toTs,
	}
	backtest.ApplyDefaults(run)

	_, err = s.pg.Exec(ctx, `
		INSERT INTO backtest_runs (
			id, user_id, account_id, symbol, timeframe,
			mode, from_ts, to_ts,
			strategy_code, template_id, initial_capital,
			commission, slippage, leverage, trade_direction, strict_mode,
			status, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,CURRENT_TIMESTAMP)`,
		run.ID, run.UserID, uuid.Nil, run.Symbol, run.Timeframe,
		run.Mode, run.FromTs, run.ToTs,
		strategyCode, templateUUID, *run.InitialCapital,
		*run.Commission, *run.Slippage, *run.Leverage, *run.TradeDirection, *run.StrictMode,
		run.Status,
	)
	if err != nil {
		return "", fmt.Errorf("marketplace: create backtest run: %w", err)
	}

	return runID.String(), nil
}

// QueryBacktestRun reads a lightweight snapshot of a backtest run for
// server-streamed progress updates.
func (s *Service) QueryBacktestRun(ctx context.Context, runID uuid.UUID) (*BacktestRunSnapshot, error) {
	var snap BacktestRunSnapshot
	err := s.pg.QueryRow(ctx,
		`SELECT status, COALESCE(error,''), COALESCE(symbol,''), COALESCE(timeframe,''),
		        proto_response, started_at, finished_at, template_id
		 FROM backtest_runs WHERE id = $1`,
		runID,
	).Scan(&snap.Status, &snap.Error, &snap.Symbol, &snap.Timeframe,
		&snap.ProtoResponse, &snap.StartedAt, &snap.FinishedAt, &snap.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: query backtest run %s: %w", runID, err)
	}
	return &snap, nil
}
