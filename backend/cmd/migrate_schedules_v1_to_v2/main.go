package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"anttrader/internal/config"
	"anttrader/internal/repository"
	"anttrader/pkg/logger"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(&logger.Config{Level: cfg.Log.Level, Format: cfg.Log.Format, Output: cfg.Log.Output}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	sqlxDB, err := repository.NewSQLXDB(&cfg.Database)
	if err != nil {
		logger.Fatal("failed to connect db", zap.Error(err))
	}
	defer sqlxDB.Close()

	ctx := context.Background()

	result, err := migrateSchedulesV1ToV2(ctx, sqlxDB)
	if err != nil {
		logger.Fatal("migration failed", zap.Error(err))
	}

	fmt.Printf("Migration completed. scanned=%d inserted=%d skipped=%d\n", result.Scanned, result.Inserted, result.Skipped)
}

type migrateResult struct {
	Scanned  int
	Inserted int
	Skipped  int
}

func migrateSchedulesV1ToV2(ctx context.Context, db *sqlx.DB) (*migrateResult, error) {
	srcTable := "strategy_schedules"
	if tableExists(ctx, db, "strategy_schedules_legacy") {
		srcTable = "strategy_schedules_legacy"
	}
	dstTable := "strategy_schedules_v2"
	if tableExists(ctx, db, "strategy_schedules") {
		dstTable = "strategy_schedules"
	}
	cols, err := loadTableColumns(ctx, db, srcTable)
	if err != nil {
		return nil, err
	}
	useNewSchema := cols["user_id"] && cols["template_id"]

	var query string
	if useNewSchema {
		query = `
			WITH inserted AS (
				INSERT INTO ` + dstTable + ` (
					id, user_id, template_id, account_id, name, symbol, timeframe,
					parameters, schedule_type, schedule_config,
					backtest_metrics, risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
					is_active, last_run_at, next_run_at, run_count, last_error,
					created_at, updated_at
				)
				SELECT
					s.id,
					s.user_id,
					s.template_id,
					s.account_id,
					s.name,
					s.symbol,
					COALESCE(NULLIF(s.timeframe, ''), 'H1'),
					COALESCE(s.parameters, '{}'::jsonb),
					COALESCE(NULLIF(s.schedule_type, ''), 'interval'),
					COALESCE(s.schedule_config, '{}'::jsonb),
					NULL::jsonb,
					NULL::int,
					'unknown',
					'[]'::jsonb,
					'[]'::jsonb,
					NULL::timestamp,
					COALESCE(s.is_active, false),
					s.last_run_at,
					s.next_run_at,
					COALESCE(s.run_count, 0),
					s.last_error,
					s.created_at,
					s.updated_at
				FROM ` + srcTable + ` s
				ON CONFLICT (id) DO NOTHING
				RETURNING 1
			)
			SELECT
				(SELECT COUNT(*) FROM ` + srcTable + `) AS scanned,
				(SELECT COUNT(*) FROM inserted) AS inserted
		`
	} else {
		query = `
			WITH inserted AS (
				INSERT INTO ` + dstTable + ` (
					id, user_id, template_id, account_id, name, symbol, timeframe,
					parameters, schedule_type, schedule_config,
					backtest_metrics, risk_score, risk_level, risk_reasons, risk_warnings, last_backtest_at,
					is_active, last_run_at, next_run_at, run_count, last_error,
					created_at, updated_at
				)
				SELECT
					s.id,
					st.user_id,
					s.strategy_id,
					s.account_id,
					st.name,
					st.symbol,
					'H1',
					'{}'::jsonb,
					COALESCE(NULLIF(s.schedule_type, ''), 'interval'),
					COALESCE(s.schedule_config, '{}'::jsonb),
					NULL::jsonb,
					NULL::int,
					'unknown',
					'[]'::jsonb,
					'[]'::jsonb,
					NULL::timestamp,
					COALESCE(s.is_active, false),
					s.last_run_at,
					s.next_run_at,
					COALESCE(s.run_count, 0),
					s.last_error,
					s.created_at,
					s.updated_at
				FROM ` + srcTable + ` s
				JOIN strategies st ON st.id = s.strategy_id
				ON CONFLICT (id) DO NOTHING
				RETURNING 1
			)
			SELECT
				(SELECT COUNT(*) FROM ` + srcTable + `) AS scanned,
				(SELECT COUNT(*) FROM inserted) AS inserted
		`
	}

	var out struct {
		Scanned  int `db:"scanned"`
		Inserted int `db:"inserted"`
	}
	if err := db.GetContext(ctx, &out, query); err != nil {
		return nil, err
	}
	return &migrateResult{Scanned: out.Scanned, Inserted: out.Inserted, Skipped: out.Scanned - out.Inserted}, nil
}

func loadTableColumns(ctx context.Context, db *sqlx.DB, table string) (map[string]bool, error) {
	query := `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1`
	var names []string
	if err := db.SelectContext(ctx, &names, query, table); err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[strings.ToLower(strings.TrimSpace(n))] = true
	}
	return out, nil
}

func tableExists(ctx context.Context, db *sqlx.DB, table string) bool {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)`
	var ok bool
	if err := db.GetContext(ctx, &ok, query, table); err != nil {
		return false
	}
	return ok
}
