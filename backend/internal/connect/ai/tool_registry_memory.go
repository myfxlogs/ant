package ai

import (
	"context"
	"strings"

	"anttrader/internal/repository"
)

// ── remember tool ──

type rememberTool struct{ execFn func(ctx context.Context, sql string, args ...any) error }

func (t *rememberTool) Name() string { return "remember" }
func (t *rememberTool) Run(ctx context.Context, in ToolInput) ToolOutput {
	if t.execFn == nil { return ToolOutput{Success: false, Error: "db not wired"} }
	parts := strings.SplitN(in.Symbol, " ", 2) // abuse Symbol field for "key value"
	key := in.Symbol
	val := in.Timeframe
	if len(parts) >= 2 {
		key = parts[0]
		val = strings.Join(parts[1:], " ")
	}
	err := t.execFn(ctx,
		"INSERT INTO ai_memory (user_id, key, value, updated_at) VALUES ($1,$2,$3,NOW()) ON CONFLICT (user_id,key) DO UPDATE SET value=$3, updated_at=NOW()",
		in.UserID, key, val)
	if err != nil {
		return ToolOutput{Success: false, Error: err.Error()}
	}
	return ToolOutput{Success: true, Output: map[string]string{"key": key, "value": val}}
}

// ── recall tool ──

type recallTool struct{ queryFn func(ctx context.Context, sql string, args ...any) (string, error) }

func (t *recallTool) Name() string { return "recall" }
func (t *recallTool) Run(ctx context.Context, in ToolInput) ToolOutput {
	if t.queryFn == nil { return ToolOutput{Success: false, Error: "db not wired"} }
	key := in.Symbol
	val, err := t.queryFn(ctx, "SELECT value FROM ai_memory WHERE user_id=$1 AND key=$2 ORDER BY updated_at DESC LIMIT 1", in.UserID, key)
	if err != nil || val == "" {
		return ToolOutput{Success: false, Error: "not found"}
	}
	return ToolOutput{Success: true, Output: map[string]string{"key": key, "value": val}}
}

// ── list_strategies tool ──

type listStrategiesTool struct{ queryFn func(ctx context.Context, sql string, args ...any) (string, error) }

func (t *listStrategiesTool) Name() string { return "list_strategies" }
func (t *listStrategiesTool) Run(ctx context.Context, in ToolInput) ToolOutput {
	code, err := t.queryFn(ctx,
		`SELECT json_agg(s ORDER BY s.created_at DESC) FROM (
		 SELECT t.name, t.created_at,
		   COALESCE((SELECT status FROM backtest_runs WHERE strategy_code_hash=md5(t.code)::text ORDER BY created_at DESC LIMIT 1), 'not_run') as bt_status
		 FROM strategy_templates t WHERE t.user_id=$1 LIMIT 20) s`, in.UserID)
	if err != nil || code == "" || code == "[null]" {
		return ToolOutput{Success: true, Output: map[string]string{"strategies": "none"}}
	}
	return ToolOutput{Success: true, Output: map[string]string{"strategies": code}}
}

// ── save_strategy tool ──

type saveStrategyTool struct{ execFn func(ctx context.Context, sql string, args ...any) error }

func (t *saveStrategyTool) Name() string { return "save_strategy" }
func (t *saveStrategyTool) Run(ctx context.Context, in ToolInput) ToolOutput {
	name := in.Symbol
	code := in.Code
	if name == "" || code == "" {
		return ToolOutput{Success: false, Error: "用法: [TOOL: save_strategy 策略名称]. 例如: [TOOL: save_strategy BTCUSD均线策略]"}
	}
	err := t.execFn(ctx,
		"INSERT INTO strategy_templates (user_id, name, code) VALUES ($1,$2,$3)",
		in.UserID, name, code)
	if err != nil {
		return ToolOutput{Success: false, Error: err.Error()}
	}
	return ToolOutput{Success: true, Output: map[string]string{"name": name, "message": "策略已保存到模板库，可在Workspace中加载"}}
}

// ── load_strategy tool ──

type loadStrategyTool struct{ queryFn func(ctx context.Context, sql string, args ...any) (string, error) }

func (t *loadStrategyTool) Name() string { return "load_strategy" }
func (t *loadStrategyTool) Run(ctx context.Context, in ToolInput) ToolOutput {
	name := in.Symbol
	if name == "" {
		return ToolOutput{Success: false, Error: "用法: [TOOL: load_strategy 策略名称]"}
	}
	code, err := t.queryFn(ctx,
		"SELECT code FROM strategy_templates WHERE user_id=$1 AND name=$2 ORDER BY created_at DESC LIMIT 1",
		in.UserID, name)
	if err != nil || code == "" {
		return ToolOutput{Success: false, Error: "未找到策略: " + name}
	}
	return ToolOutput{Success: true, Output: map[string]string{"name": name, "code": code}}
}

// ── backtest tool ──

type backtestTool struct{ repo *repository.BacktestRunRepository }

func (t *backtestTool) Name() string { return "backtest" }

func (t *backtestTool) Run(ctx context.Context, in ToolInput) ToolOutput {
	runID, err := CreateBacktestRun(ctx, t.repo, in.UserID, in.Code, in.Symbol, in.Timeframe)
	if err != nil {
		return ToolOutput{Success: false, Error: err.Error()}
	}
	return ToolOutput{
		Success: true,
		Output:  map[string]string{"run_id": runID},
	}
}
