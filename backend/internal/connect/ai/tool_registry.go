package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
	"anttrader/internal/repository"
)

// ── Tool interface ──

// ToolInput is passed to each tool by the execution engine.
type ToolInput struct {
	Code      string
	Symbol    string
	Timeframe string
	UserID    uuid.UUID
}

// ToolOutput is the structured result returned by each tool.
type ToolOutput struct {
	Success bool
	Output  any    // tool-specific struct (will be JSON-marshalled)
	Error   string
}

// Tool is a single step in the execution pipeline.
type Tool interface {
	Name() string
	Run(ctx context.Context, in ToolInput) ToolOutput
}

// ── Tool Registry ──

// ToolRegistry holds the ordered list of tools to execute after code generation.
type ToolRegistry struct {
	preTools []Tool     // tools the AI can request during planning/generation
	tools    []Tool     // auto-run tools after code generation
}

// NewToolRegistry creates a registry with the standard tool set.
func NewToolRegistry(backtestRepo *repository.BacktestRunRepository, store repository.MarketDataStore) *ToolRegistry {
	return &ToolRegistry{
		preTools: []Tool{
			&readKlineTool{repo: store},
			&readBacktestLogTool{repo: backtestRepo},
		},
		tools: []Tool{
			&complianceTool{},
			&backtestTool{repo: backtestRepo},
		},
	}
}


// WireMemoryDB wires the PG pool for memory tools.
func (r *ToolRegistry) WireMemoryDB(execFn func(ctx context.Context, sql string, args ...any) error, queryFn func(ctx context.Context, sql string, args ...any) (string, error)) {
	rem := &rememberTool{execFn: execFn}
	rec := &recallTool{queryFn: queryFn}
	ls := &listStrategiesTool{queryFn: queryFn}
	sv := &saveStrategyTool{execFn: execFn}
	ld := &loadStrategyTool{queryFn: queryFn}
	r.preTools = append(r.preTools, rem, rec, ls, sv, ld)
}

// PreToolNames returns the names of pre-execution tools the AI can request.
func (r *ToolRegistry) PreToolNames() []string {
	var names []string
	for _, t := range r.preTools {
		names = append(names, t.Name())
	}
	return names
}

// FindPreTool looks up a pre-execution tool by name. Returns nil if not found.
func (r *ToolRegistry) FindPreTool(name string) Tool {
	for _, t := range r.preTools {
		if t.Name() == name {
			return t
		}
	}
	return nil
}

// Execute runs all registered tools in order, streaming results via the callback.
func (r *ToolRegistry) Execute(ctx context.Context, in ToolInput, send func(*antv1.ExecutePlanChunk) error) {
	for _, t := range r.tools {
		callID := "call_" + t.Name()
		_ = send(&antv1.ExecutePlanChunk{
			Phase: "tool_call",
			ToolCall: &antv1.ToolCall{
				CallId: callID, Name: t.Name(), ParamsJson: `{}`,
			},
		})

		out := t.Run(ctx, in)
		outJSON, _ := json.Marshal(out.Output)
		_ = send(&antv1.ExecutePlanChunk{
			Phase: "tool_result",
			ToolResult: &antv1.ToolResult{
				CallId: callID, Name: t.Name(),
				Success: out.Success, OutputJson: string(outJSON), Error: out.Error,
			},
		})
	}
}

// ── compliance_check tool ──

type complianceTool struct{}

func (t *complianceTool) Name() string { return "compliance_check" }

func (t *complianceTool) Run(_ context.Context, in ToolInput) ToolOutput {
	blocks, warns := ai.NewCodeComplianceScanner().Scan(in.Code)
	allIssues := append(blocks, warns...)
	var issueProtos []*antv1.ComplianceIssue
	for _, iss := range allIssues {
		issueProtos = append(issueProtos, &antv1.ComplianceIssue{
			Rule: iss.RuleName, Message: iss.Message, Severity: iss.Severity, Line: int32(iss.Line),
		})
	}
	return ToolOutput{
		Success: len(blocks) == 0,
		Output:  &antv1.ComplianceResult{Passed: len(blocks) == 0, Issues: issueProtos},
	}
}


// ── detect_regime tool ──

type detectRegimeTool struct{ store repository.MarketDataStore }

func (t *detectRegimeTool) Name() string { return "detect_regime" }
func (t *detectRegimeTool) Run(ctx context.Context, in ToolInput) ToolOutput {
	if t.store == nil {
		return ToolOutput{Success: false, Error: "market data store not wired"}
	}
	bars, err := t.store.GetKlines(ctx, in.Symbol, "", in.Timeframe, nil, nil, 200)
	if err != nil || len(bars) < 20 {
		return ToolOutput{Success: false, Error: fmt.Sprintf("insufficient kline data: %d bars", len(bars))}
	}
	regime := detectRegimeFromBars(bars)
	return ToolOutput{Success: true, Output: map[string]string{
		"symbol": in.Symbol, "timeframe": in.Timeframe,
		"regime": regime, "bars": fmt.Sprintf("%d", len(bars)),
	}}
}

func detectRegimeFromBars(bars []repository.KlineBar) string {
	if len(bars) < 20 { return "unknown" }
	closes := make([]float64, len(bars))
	for i, b := range bars { closes[i] = b.Close }

	// Simple trend detection: linear regression slope
	n := float64(len(closes))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i, y := range closes {
		x := float64(i)
		sumX += x; sumY += y; sumXY += x*y; sumX2 += x*x
	}
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	avgPrice := sumY / n
	normalizedSlope := slope / avgPrice * 100

	// Volatility
	var variance float64
	for _, y := range closes {
		variance += (y - avgPrice) * (y - avgPrice)
	}
	vol := variance / n
	avgVol := avgPrice * 0.02 // 2% baseline

	switch {
	case normalizedSlope > 0.5 && vol > avgVol:
		return "bull_trend_volatile"
	case normalizedSlope > 0.3:
		return "bull_trend"
	case normalizedSlope < -0.5 && vol > avgVol:
		return "bear_trend_volatile"
	case normalizedSlope < -0.3:
		return "bear_trend"
	case vol > avgVol*2:
		return "high_volatility"
	case vol < avgVol*0.5:
		return "range_compression"
	default:
		return "ranging"
	}
}

// ── read_kline tool ──

type readKlineTool struct{ repo repository.MarketDataStore }

func (t *readKlineTool) Name() string { return "read_kline" }
func (t *readKlineTool) Run(_ context.Context, in ToolInput) ToolOutput {
	bars, err := t.repo.GetKlines(context.Background(), in.Symbol, "", in.Timeframe, nil, nil, 2000)
	if err != nil {
		return ToolOutput{Success: false, Error: err.Error()}
	}
	if len(bars) == 0 {
		return ToolOutput{Success: true, Output: map[string]any{"bars": 0, "message": "no data for this symbol/timeframe"}}
	}
	return ToolOutput{
		Success: true,
		Output: map[string]any{
			"symbol": in.Symbol, "timeframe": in.Timeframe,
			"bars":   len(bars),
			"first":  bars[0].CloseTsUnixMs,
			"last":   bars[len(bars)-1].CloseTsUnixMs,
		},
	}
}

// ── read_backtest_log tool ──

type readBacktestLogTool struct{ repo *repository.BacktestRunRepository }

func (t *readBacktestLogTool) Name() string { return "read_backtest_log" }
func (t *readBacktestLogTool) Run(ctx context.Context, in ToolInput) ToolOutput {
	// Use the most recent backtest run for this code hash
	runs, err := t.repo.ListByUser(ctx, in.UserID, nil, nil, 1, 0)
	if err != nil || len(runs) == 0 {
		return ToolOutput{Success: false, Error: "no recent backtest runs found"}
	}
	run := runs[0]
	out := map[string]any{
		"run_id": run.ID.String(), "symbol": run.Symbol, "timeframe": run.Timeframe,
		"status": run.Status,
	}
	if run.Error != "" {
		out["error"] = run.Error
	}
	return ToolOutput{Success: true, Output: out}
}


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
