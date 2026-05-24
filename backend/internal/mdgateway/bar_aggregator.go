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

// finalizedBars stores the MAX close_ts_unix_ms for each (broker,canonical,period)
// that has been committed to CH. Replay/backfill bars with close_ts <= this value
// are skipped (ADR-0009 §2.2 bar finality invariant).
type finalizedKey struct {
	broker, canonical, period string
}

type BarAggregator struct {
	mu sync.Mutex
	bars map[string]*openBar // key: broker:canonical:period

	finalizedBars map[finalizedKey]int64
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
		finalizedBars: make(map[finalizedKey]int64),
	}
}

// LoadFinalizedBars hydrates the finalized-bars map from ClickHouse.
// Call after CH connection is established, before any bar ingestion.
// Query: SELECT broker, canonical, period, MAX(close_ts_unix_ms)
//
//	FROM md_bars GROUP BY broker, canonical, period
func (a *BarAggregator) LoadFinalizedBars(maxCloseTs map[finalizedKey]int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for k, v := range maxCloseTs {
		a.finalizedBars[k] = v
	}
}

// IngestExternalBar handles bars from backfiller or spill_replay (IsReplay=true).
// Returns false if the bar was skipped because its close_ts <= finalizedBars[key]
// (ADR-0009 §2.2 bar finality invariant).
func (a *BarAggregator) IngestExternalBar(b *mdtick.Bar) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	fk := finalizedKey{b.Broker, b.Canonical, b.Period}
	if finalized, ok := a.finalizedBars[fk]; ok && b.CloseTsUnixMs <= finalized {
		// Already finalized — skip.
		barSkippedFinalized.Add(1)
		return false
	}
	// Accept: update finalized ceiling.
	if b.CloseTsUnixMs > a.finalizedBars[fk] {
		a.finalizedBars[fk] = b.CloseTsUnixMs
	}
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
			// ADR-0009 §2.2: real-time bars update finalized ceiling.
			fk := finalizedKey{t.Broker, t.Canonical, p.Name}
			if bar.CloseTsUnixMs > a.finalizedBars[fk] {
				a.finalizedBars[fk] = bar.CloseTsUnixMs
			}
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
