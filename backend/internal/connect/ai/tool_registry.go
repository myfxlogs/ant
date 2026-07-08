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
	RawArgs   map[string]any // full LLM arguments — tools read what they need
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

type ReadKlineTool struct{ repo repository.MarketDataStore }

// NewReadKlineTool creates a read_kline tool with the given market data store.
func NewReadKlineTool(repo repository.MarketDataStore) *ReadKlineTool {
	return &ReadKlineTool{repo: repo}
}

func (t *ReadKlineTool) Name() string { return "read_kline" }
func (t *ReadKlineTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name:        "read_kline",
			Description: "读取K线数据并返回市场分析（bar数/日期/价格/EMA/趋势/波动率/近期OHLC）。当你需要了解当前市场状况、分析行情趋势、查看价格形态、或写策略前确认数据可用性时调用此工具。用户问'市场怎么样/什么形态/趋势如何'时必须调用。",
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
func (t *ReadKlineTool) Run(ctx context.Context, in ToolInput) ToolOutput {
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

	// ── Build market analysis ──
	// Last 10 bars for the LLM to "see" recent price action.
	type barSummary struct {
		T int64   `json:"t"`
		O float64 `json:"o"`
		H float64 `json:"h"`
		L float64 `json:"l"`
		C float64 `json:"c"`
	}
	recent := bars
	if len(recent) > 10 {
		recent = recent[len(recent)-10:]
	}
	recentBars := make([]barSummary, len(recent))
	for i, b := range recent {
		recentBars[i] = barSummary{
			T: int64(b.CloseTsUnixMs),
			O: b.Open.InexactFloat64(),
			H: b.High.InexactFloat64(),
			L: b.Low.InexactFloat64(),
			C: b.Close.InexactFloat64(),
		}
	}

	// Compute EMA(20) and EMA(50) for trend detection.
	ema20 := ema(bars, 20)
	ema50 := ema(bars, 50)
	currentPrice := recentBars[len(recentBars)-1].C

	// Trend detection: EMA position + recent slope.
	trend := "ranging"
	trendStrength := "neutral"
	if ema20 > 0 && ema50 > 0 {
		if ema20 > ema50 && currentPrice > ema20 {
			trend = "上升趋势 (uptrend)"
			trendStrength = "bullish"
		} else if ema20 < ema50 && currentPrice < ema20 {
			trend = "下降趋势 (downtrend)"
			trendStrength = "bearish"
		}
	}

	// Recent price range (last 50 bars).
	lookback := bars
	if len(lookback) > 50 {
		lookback = lookback[len(lookback)-50:]
	}
	high, low := lookback[0].High.InexactFloat64(), lookback[0].Low.InexactFloat64()
	for _, b := range lookback {
		if h := b.High.InexactFloat64(); h > high {
			high = h
		}
		if l := b.Low.InexactFloat64(); l < low {
			low = l
		}
	}
	rangePct := (high - low) / low * 100

	// Volatility (mean absolute return %, last 20 bars).
	volLookback := bars
	if len(volLookback) > 20 {
		volLookback = volLookback[len(volLookback)-20:]
	}
	var sumAbsReturn float64
	for i := 1; i < len(volLookback); i++ {
		r := (volLookback[i].Close.InexactFloat64() - volLookback[i-1].Close.InexactFloat64()) / volLookback[i-1].Close.InexactFloat64() * 100
		if r < 0 {
			r = -r
		}
		sumAbsReturn += r
	}
	meanVol := sumAbsReturn / float64(len(volLookback)-1)

	return ToolOutput{
		Success: true,
		Output: map[string]any{
			"symbol":          in.Symbol,
			"timeframe":       in.Timeframe,
			"total_bars":      len(bars),
			"date_from":       time.UnixMilli(first).UTC().Format("2006-01-02"),
			"date_to":         time.UnixMilli(last).UTC().Format("2006-01-02"),
			"current_price":   fmt.Sprintf("%.5f", currentPrice),
			"ema_20":          fmt.Sprintf("%.5f", ema20),
			"ema_50":          fmt.Sprintf("%.5f", ema50),
			"trend":           trend,
			"trend_strength":  trendStrength,
			"recent_high":     fmt.Sprintf("%.5f", high),
			"recent_low":      fmt.Sprintf("%.5f", low),
			"recent_range_pct": fmt.Sprintf("%.2f", rangePct),
			"volatility_pct":  fmt.Sprintf("%.3f", meanVol),
			"recent_bars":     recentBars,
		},
	}
}

// ema computes the Exponential Moving Average for the last N bars.
func ema(bars []repository.KlineBar, period int) float64 {
	if len(bars) < period {
		return 0
	}
	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close.InexactFloat64()
	}
	k := 2.0 / float64(period+1)
	result := closes[0]
	for i := 1; i < len(closes); i++ {
		result = closes[i]*k + result*(1-k)
	}
	return result
}

// ── read_backtest_log tool ──

type ReadBacktestLogTool struct{ repo *repository.BacktestRunRepository }

func NewReadBacktestLogTool(repo *repository.BacktestRunRepository) *ReadBacktestLogTool {
	return &ReadBacktestLogTool{repo: repo}
}

func (t *ReadBacktestLogTool) Name() string { return "read_backtest_log" }
func (t *ReadBacktestLogTool) Schema() systemai.ToolDefinition {
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
func (t *ReadBacktestLogTool) Run(ctx context.Context, in ToolInput) ToolOutput {
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
