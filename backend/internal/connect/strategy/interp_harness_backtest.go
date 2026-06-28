package strategy

// generateInterpBacktestHarness generates a backtest harness for the MQL
// interpreter path. The harness calls interp.WASMRunSetup() to read
// serialized IR from stdin and create an Interpreter instance.
// No user strategy code needed — the IR IS the strategy.
func generateInterpBacktestHarness() string {
	return generateBacktestHarnessBase(
		"strategy := interp.WASMRunSetup()",
		`"anttrader/tools/mql2go/interp"`,
	)
}
