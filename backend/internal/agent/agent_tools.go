package agent

import (
	antv1 "alphaforge/gen/proto/ant/v1"
	connectai "alphaforge/internal/connect/ai"
	"alphaforge/internal/repository"
)

// buildPythonToolRegistry creates a ToolRegistry with all agent tools.
// NEW: mkt+cfg injected for write_strategy real backtest (I2).
func buildPythonToolRegistry(result *generateState, mkt repository.MarketDataStore, cfg *antv1.AgentBacktestConfig) *connectai.ToolRegistry {
	reg := connectai.NewEmptyToolRegistry()
	// PRIMARY: write_strategy — the only way to submit final code (I1).
	// Compile + real backtest happen inside this tool (I2).
	reg.AddPreTool(&writeStrategyTool{result: result, mkt: mkt, cfg: cfg})
	// Support tools.
	reg.AddPreTool(&readCurrentCodeTool{result: result})
	reg.AddPreTool(&editCodeTool{result: result})
	reg.AddPreTool(&updatePlanTool{result: result})
	// Market data: let the LLM inspect current bars before writing strategy code.
	// Needed for "what's the market pattern?" / "check kline" queries.
	reg.AddPreTool(connectai.NewReadKlineTool(mkt))
	// compile_python removed: write_strategy already does compile + backtest.
	return reg
}
