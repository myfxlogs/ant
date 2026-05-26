package mthub

import "anttrader/internal/clock"

// Clk is the package-level clock used for all time operations.
// Defaults to real wall clock. Set to clock.SimulatedClock for deterministic backtests.
var Clk clock.Clock = clock.NewRealClock()
