package mdgateway

import (
	"sync"

	"github.com/shopspring/decimal"
	"anttrader/internal/mdgateway/adapter/mdtick"
)

var Periods = []struct{ Name string; Ms int64 }{
	{"1m", 60_000}, {"5m", 300_000}, {"15m", 900_000},
	{"1h", 3_600_000}, {"4h", 14_400_000}, {"1d", 86_400_000},
}

// finalizedBars stores the set of close_ts_unix_ms values that have been
// committed to CH for each (broker,canonical,period). Replay/backfill bars
// with an exact-match close_ts are skipped (ADR-0009 §2.2 + M10.5-3d fix:
// changed from MAX-based comparison to exact-match dedup).
type finalizedKey struct {
	broker, canonical, period string
}

type BarAggregator struct {
	mu sync.Mutex
	bars map[string]*openBar // key: broker:canonical:period

	finalizedBars map[finalizedKey]map[int64]struct{} // set of close_ts values per key
}

type openBar struct {
	bucket int64
	open, high, low, close decimal.Decimal
	volume float64
	count  uint32
	startTs, endTs int64
}

func NewBarAggregator() *BarAggregator {
	return &BarAggregator{
		bars:          make(map[string]*openBar),
		finalizedBars: make(map[finalizedKey]map[int64]struct{}),
	}
}

// LoadFinalizedBars hydrates the finalized-bars map from ClickHouse.
// Call after CH connection is established, before any bar ingestion.
// Query: SELECT broker, canonical, period, MAX(close_ts_unix_ms)
//
//	FROM md_bars GROUP BY broker, canonical, period
// LoadFinalizedBars hydrates the finalized set from CH close_ts values.
// Call after CH connection is established, before any bar ingestion.
func (a *BarAggregator) LoadFinalizedBars(closeTsMap map[finalizedKey][]int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for k, vals := range closeTsMap {
		if a.finalizedBars[k] == nil {
			a.finalizedBars[k] = make(map[int64]struct{})
		}
		for _, v := range vals {
			a.finalizedBars[k][v] = struct{}{}
		}
	}
}

// IngestExternalBar handles bars from backfiller or spill_replay (IsReplay=true).
// Returns false if the bar was skipped because its close_ts already exists in
// the finalized set (ADR-0009 §2.2 + M10.5-3d: exact-match dedup).
func (a *BarAggregator) IngestExternalBar(b *mdtick.Bar) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	fk := finalizedKey{b.Broker, b.Canonical, b.Period}
	if set, ok := a.finalizedBars[fk]; ok {
		if _, exists := set[b.CloseTsUnixMs]; exists {
			barSkippedFinalized.Add(1)
			return false
		}
	}
	// Accept: add to finalized set.
	if a.finalizedBars[fk] == nil {
		a.finalizedBars[fk] = make(map[int64]struct{})
	}
	a.finalizedBars[fk][b.CloseTsUnixMs] = struct{}{}
	return true
}

// AddTick processes a tick; emits completed bars via onBar.
// ADR-0008 §2.2 + ADR-0009 §2.2: Uses ArrivedUnixMs for bucketing
// (local clock — the only system-clock source; QUIRK Q-001).
func (a *BarAggregator) AddTick(t *mdtick.Tick, onBar func(*mdtick.Bar)) {
	a.mu.Lock()
	defer a.mu.Unlock()

	mid := t.Bid.Add(t.Ask).Div(decimal.NewFromInt(2))

	for _, p := range Periods {
		key := t.Broker + ":" + t.Canonical + ":" + p.Name
		bucket := t.ArrivedUnixMs / p.Ms

		ob := a.bars[key]
		if ob == nil {
			ob = &openBar{bucket: bucket, open: mid, high: mid, low: mid, close: mid, startTs: t.ArrivedUnixMs}
			a.bars[key] = ob
		} else if ob.bucket != bucket {
			bar := &mdtick.Bar{
				Broker: t.Broker, Canonical: t.Canonical, Period: p.Name,
				OpenTsUnixMs: ob.startTs, CloseTsUnixMs: ob.endTs,
				Open: ob.open, High: ob.high, Low: ob.low, Close: ob.close,
				Volume: ob.volume, TickCount: ob.count,
			}
			// ADR-0009 §2.2 + M10.5-3d: real-time bars add to finalized set.
			fk := finalizedKey{t.Broker, t.Canonical, p.Name}
			if a.finalizedBars[fk] == nil {
				a.finalizedBars[fk] = make(map[int64]struct{})
			}
			a.finalizedBars[fk][bar.CloseTsUnixMs] = struct{}{}
			onBar(bar)
			ob.bucket = bucket
			ob.open = mid; ob.high = mid; ob.low = mid; ob.close = mid
			ob.volume = 0; ob.count = 0
			ob.startTs = t.ArrivedUnixMs
		}
		if mid.Cmp(ob.high) > 0 { ob.high = mid }
		if mid.Cmp(ob.low) < 0 { ob.low = mid }
		ob.close = mid
		ob.volume += float64(t.BidVolume + t.AskVolume)
		ob.count++
		ob.endTs = t.ArrivedUnixMs
	}
}
