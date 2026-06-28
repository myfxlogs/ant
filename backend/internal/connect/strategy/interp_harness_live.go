package strategy

// generateInterpLiveHarness generates a live harness for the MQL
// interpreter path. The harness calls interp.WASMRunSetup() to read
// serialized IR from stdin and create an Interpreter instance.
// No user strategy code needed — the IR IS the strategy.
func generateInterpLiveHarness() string {
	return generateLiveHarnessBase(
		"strategy := interp.WASMRunSetup()",
		`"anttrader/tools/mql2go/interp"`,
	)
}
