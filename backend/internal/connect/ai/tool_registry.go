package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
	"anttrader/tools/mql2go"
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
	// Schema returns the JSON Schema definition for this tool (OpenAI function calling format).
	Schema() systemai.ToolDefinition
}

// ── Tool Registry ──

// ToolRegistry holds the ordered list of tools the AI agent can request.
type ToolRegistry struct {
	preTools []Tool
}

// NewEmptyToolRegistry creates a registry with no pre-loaded tools.
// Callers add tools via AddPreTool.
func NewEmptyToolRegistry() *ToolRegistry {
	return &ToolRegistry{}
}

// AddPreTool appends a tool to the pre-execution tool set.
func (r *ToolRegistry) AddPreTool(t Tool) {
	r.preTools = append(r.preTools, t)
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

// BuildToolSchemas returns JSON Schema definitions for all pre-execution tools.
// These are injected into LLM requests so the model can use native tool_use.
func (r *ToolRegistry) BuildToolSchemas() []systemai.ToolDefinition {
	schemas := make([]systemai.ToolDefinition, len(r.preTools))
	for i, t := range r.preTools {
		schemas[i] = t.Schema()
	}
	return schemas
}

// FindPreTool looks up a tool by name. Returns nil if not found.
func (r *ToolRegistry) FindPreTool(name string) Tool {
	for _, t := range r.preTools {
		if t.Name() == name {
			return t
		}
	}
	return nil
}

// ── read_kline tool ──

type readKlineTool struct{ repo repository.MarketDataStore }

func (t *readKlineTool) Name() string { return "read_kline" }
func (t *readKlineTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name:        "read_kline",
			Description: "K线数据统计。返回指定品种和时间周期的bar数量、数据起止日期。用于在生成代码前检查数据是否充足，或回测失败时排查数据问题。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"symbol":    map[string]any{"type": "string", "description": "交易品种代码，例如 BTCUSDm, XAUUSDm"},
					"timeframe": map[string]any{"type": "string", "enum": []string{"1m", "5m", "15m", "30m", "1h", "4h", "1d", "1w"}},
				},
				"required": []string{"symbol", "timeframe"},
			},
		},
	}
}
func (t *readKlineTool) Run(ctx context.Context, in ToolInput) ToolOutput {
	bars, err := t.repo.GetKlines(ctx, in.Symbol, "", in.Timeframe, nil, nil, 2000)
	if err != nil {
		return ToolOutput{Success: false, Error: err.Error()}
	}
	if len(bars) == 0 {
		return ToolOutput{Success: true, Output: map[string]any{
			"bars":    0,
			"message": fmt.Sprintf("数据库中无 %s %s 的数据。请如实告诉用户：该品种暂无K线数据。不要编造任何日期或数量。", in.Symbol, in.Timeframe),
		}}
	}
	first := int64(bars[0].CloseTsUnixMs)
	last := int64(bars[len(bars)-1].CloseTsUnixMs)
	return ToolOutput{
		Success: true,
		Output: map[string]any{
			"symbol":    in.Symbol,
			"timeframe": in.Timeframe,
			"bars":      len(bars),
			"first_ms":  first,
			"last_ms":   last,
			"first_utc": time.UnixMilli(first).UTC().Format("2006-01-02 15:04:05"),
			"last_utc":  time.UnixMilli(last).UTC().Format("2006-01-02 15:04:05"),
		},
	}
}

// ── read_backtest_log tool ──

type readBacktestLogTool struct{ repo *repository.BacktestRunRepository }

func (t *readBacktestLogTool) Name() string { return "read_backtest_log" }
func (t *readBacktestLogTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name:        "read_backtest_log",
			Description: "读取最近一次回测的状态和错误信息。用于回测失败后查看具体原因。无需参数。",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}
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


// ── compile_python tool ──

type compilePythonChatTool struct{}

func (t *compilePythonChatTool) Name() string { return "compile_python" }
func (t *compilePythonChatTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name:        "compile_python",
			Description: "编译当前的 Python 策略代码。代码必须是符合 Python 子集规范的完整策略。编译成功返回覆盖度评分；编译失败返回具体错误信息。",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}
func (t *compilePythonChatTool) Run(_ context.Context, in ToolInput) ToolOutput {
	if in.Code == "" {
		return ToolOutput{Success: false, Error: "no Python code to compile"}
	}
	_, coverage, err := mql2go.CompilePythonWithCoverage(in.Code)
	if err != nil {
		return ToolOutput{Success: false, Error: err.Error()}
	}
	return ToolOutput{
		Success: true,
		Output: map[string]any{
			"compiles":       true,
			"coverage_score": coverage.Score,
		},
	}
}
