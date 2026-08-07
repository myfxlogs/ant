// LIVE-1 regression: the live strategy runner must execute OnBar on finalized
// bars only. Open/in-progress bars (chart-feed snapshots) are skipped so live
// semantics match closed-bar backtest. This test locks shouldRunOnBar — the
// guard that prevents open bars from reaching handleBar.
package strategy

import (
	"testing"

	"alphaforge/internal/mthub"
)

func TestShouldRunOnBar(t *testing.T) {
	const sym, tf = "EURUSD", "1h"
	cases := []struct {
		name string
		bar  *mthub.BarUpdate
		want bool
	}{
		{"closed matching", &mthub.BarUpdate{Symbol: sym, Period: tf, Closed: true}, true},
		{"open matching — LIVE-1 must skip", &mthub.BarUpdate{Symbol: sym, Period: tf, Closed: false}, false},
		{"closed wrong symbol", &mthub.BarUpdate{Symbol: "GBPUSD", Period: tf, Closed: true}, false},
		{"closed wrong timeframe", &mthub.BarUpdate{Symbol: sym, Period: "15m", Closed: true}, false},
		{"open wrong symbol", &mthub.BarUpdate{Symbol: "GBPUSD", Period: tf, Closed: false}, false},
		{"zero value bar (nothing set)", &mthub.BarUpdate{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldRunOnBar(c.bar, sym, tf); got != c.want {
				t.Errorf("shouldRunOnBar(%+v) = %v, want %v", c.bar, got, c.want)
			}
		})
	}
}
