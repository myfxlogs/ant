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

type BarAggregator struct {
	mu sync.Mutex
	bars map[string]*openBar // key: broker:canonical:period
}

type openBar struct {
	bucket int64
	open, high, low, close decimal.Decimal
	volume float64
	count  uint32
	startTs, endTs int64
}

func NewBarAggregator() *BarAggregator {
	return &BarAggregator{bars: make(map[string]*openBar)}
}

// AddTick processes a tick; emits completed bars via onBar.
// Uses ArrivedUnixMs for bucketing (QUIRK Q-001).
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
			onBar(&mdtick.Bar{
				Broker: t.Broker, Canonical: t.Canonical, Period: p.Name,
				OpenTsUnixMs: ob.startTs, CloseTsUnixMs: ob.endTs,
				Open: ob.open, High: ob.high, Low: ob.low, Close: ob.close,
				Volume: ob.volume, TickCount: ob.count,
			})
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
