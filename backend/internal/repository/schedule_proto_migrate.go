package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// MigrateScheduleProtoColumns scans strategy_schedules rows that still contain
// legacy JSON text in BYTEA columns and re-encodes them as proto binary.
// Runs once at startup; idempotent — rows already in proto format are skipped.
func MigrateScheduleProtoColumns(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `
		SELECT id, parameters, schedule_config, backtest_metrics, risk_reasons, risk_warnings
		FROM strategy_schedules`)
	if err != nil {
		return fmt.Errorf("migrate schedule proto: query: %w", err)
	}
	defer rows.Close()

	type updateRow struct {
		id              uuid.UUID
		parameters      []byte
		scheduleConfig  []byte
		backtestMetrics []byte
		riskReasons     []byte
		riskWarnings    []byte
	}

	var updates []updateRow
	for rows.Next() {
		var r updateRow
		if err := rows.Scan(&r.id, &r.parameters, &r.scheduleConfig, &r.backtestMetrics, &r.riskReasons, &r.riskWarnings); err != nil {
			return fmt.Errorf("migrate schedule proto: scan: %w", err)
		}
		updates = append(updates, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("migrate schedule proto: rows: %w", err)
	}

	for _, r := range updates {
		changed := false

		if len(r.parameters) > 0 {
			if migrated, ok := migrateParameters(r.parameters); ok {
				r.parameters = migrated
				changed = true
			}
		}
		if len(r.scheduleConfig) > 0 {
			if migrated, ok := migrateScheduleConfig(r.scheduleConfig); ok {
				r.scheduleConfig = migrated
				changed = true
			}
		}
		if len(r.backtestMetrics) > 0 {
			if migrated, ok := migrateBacktestMetrics(r.backtestMetrics); ok {
				r.backtestMetrics = migrated
				changed = true
			}
		}
		if len(r.riskReasons) > 0 {
			if migrated, ok := migrateStringList(r.riskReasons); ok {
				r.riskReasons = migrated
				changed = true
			}
		}
		if len(r.riskWarnings) > 0 {
			if migrated, ok := migrateStringList(r.riskWarnings); ok {
				r.riskWarnings = migrated
				changed = true
			}
		}

		if !changed {
			continue
		}

		_, err := pool.Exec(ctx, `
			UPDATE strategy_schedules SET
				parameters = $2, schedule_config = $3, backtest_metrics = $4,
				risk_reasons = $5, risk_warnings = $6, updated_at = $7
			WHERE id = $1`,
			r.id, r.parameters, r.scheduleConfig, r.backtestMetrics, r.riskReasons, r.riskWarnings, time.Now())
		if err != nil {
			return fmt.Errorf("migrate schedule proto: update %s: %w", r.id, err)
		}
	}

	return nil
}

// migrateParameters converts legacy JSON map[string]string to proto StrategyParams.
func migrateParameters(data []byte) ([]byte, bool) {
	var p antv1.StrategyParams
	if proto.Unmarshal(data, &p) == nil {
		return nil, false
	}
	var legacy map[string]string
	if json.Unmarshal(data, &legacy) != nil {
		return nil, false
	}
	out, err := proto.Marshal(&antv1.StrategyParams{Values: legacy})
	if err != nil {
		return nil, false
	}
	return out, true
}

// migrateScheduleConfig converts legacy JSON to proto ScheduleConfig.
func migrateScheduleConfig(data []byte) ([]byte, bool) {
	var cfg antv1.ScheduleConfig
	if proto.Unmarshal(data, &cfg) == nil {
		return nil, false
	}
	var legacy struct {
		CronExpression            string `json:"cron_expression"`
		IntervalMs               int64  `json:"interval_ms"`
		EventTrigger             string `json:"event_trigger"`
		TriggerMode              string `json:"trigger_mode"`
		StableOverrideIntervalMs int64  `json:"stable_override_interval_ms"`
		HfCooldownMs             int64  `json:"hf_cooldown_ms"`
	}
	if json.Unmarshal(data, &legacy) != nil {
		return nil, false
	}
	out, err := proto.Marshal(&antv1.ScheduleConfig{
		CronExpression:            legacy.CronExpression,
		IntervalMs:               legacy.IntervalMs,
		EventTrigger:             legacy.EventTrigger,
		TriggerMode:              legacy.TriggerMode,
		StableOverrideIntervalMs: legacy.StableOverrideIntervalMs,
		HfCooldownMs:             legacy.HfCooldownMs,
	})
	if err != nil {
		return nil, false
	}
	return out, true
}

// migrateBacktestMetrics converts legacy JSON to proto BacktestMetrics.
func migrateBacktestMetrics(data []byte) ([]byte, bool) {
	var m antv1.BacktestMetrics
	if proto.Unmarshal(data, &m) == nil {
		return nil, false
	}
	var legacy struct {
		TotalReturn   string `json:"total_return"`
		AnnualReturn  string `json:"annual_return"`
		MaxDrawdown   string `json:"max_drawdown"`
		SharpeRatio   string `json:"sharpe_ratio"`
		WinRate       string `json:"win_rate"`
		ProfitFactor  string `json:"profit_factor"`
		TotalTrades   int32  `json:"total_trades"`
		WinningTrades int32  `json:"winning_trades"`
		LosingTrades  int32  `json:"losing_trades"`
		AverageProfit string `json:"average_profit"`
		AverageLoss   string `json:"average_loss"`
	}
	if json.Unmarshal(data, &legacy) != nil {
		return nil, false
	}
	out, err := proto.Marshal(&antv1.BacktestMetrics{
		TotalReturn:   legacy.TotalReturn,
		AnnualReturn:  legacy.AnnualReturn,
		MaxDrawdown:   legacy.MaxDrawdown,
		SharpeRatio:   legacy.SharpeRatio,
		WinRate:       legacy.WinRate,
		ProfitFactor:  legacy.ProfitFactor,
		TotalTrades:   legacy.TotalTrades,
		WinningTrades: legacy.WinningTrades,
		LosingTrades:  legacy.LosingTrades,
		AverageProfit: legacy.AverageProfit,
		AverageLoss:   legacy.AverageLoss,
	})
	if err != nil {
		return nil, false
	}
	return out, true
}

// migrateStringList converts legacy JSON []string to proto BacktestRisk.
func migrateStringList(data []byte) ([]byte, bool) {
	var lst antv1.BacktestRisk
	if proto.Unmarshal(data, &lst) == nil {
		return nil, false
	}
	var legacy []string
	if json.Unmarshal(data, &legacy) != nil {
		return nil, false
	}
	out, err := proto.Marshal(&antv1.BacktestRisk{Reasons: legacy})
	if err != nil {
		return nil, false
	}
	return out, true
}
