package agent

import (
	antv1 "anttrader/gen/proto/ant/v1"
	connectai "anttrader/internal/connect/ai"
	"anttrader/internal/repository"
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
	reg.AddPreTool(&updatePlanTool{})
	// compile_python removed: write_strategy already does compile + backtest.
	// Its presence confuses LLM into picking it over write_strategy (native function calling).
	return reg
}
