package mt5

import "anttrader/internal/clock"

// Clk is the package-level clock used for all time operations.
var Clk clock.Clock = clock.NewRealClock()
