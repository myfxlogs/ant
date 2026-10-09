package runner

import "alphaforge/internal/clock"

// Clk is the package-level clock used for all time operations.
// Defaults to real wall clock. Set to clock.SimulatedClock for deterministic tests.
// (forbidigo M10-BASE-A5: time.Now 直用禁令——所有取时走 Clk。)
var Clk clock.Clock = clock.NewRealClock()
