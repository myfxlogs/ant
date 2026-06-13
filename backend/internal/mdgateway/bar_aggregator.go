package mdgateway

import (
	"strings"
	"sync"

	"github.com/shopspring/decimal"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/repository"
)

var Periods = []struct{ Name string; Ms int64 }{
	{"1m", 60_000}, {"5m", 300_000}, {"15m", 900_000}, {"30m", 1_800_000},
	{"1h", 3_600_000}, {"4h", 14_400_000}, {"1d", 86_400_000}, {"1w", 604_800_000},
}

type BarAggregator struct {
	mu sync.Mutex
	bars map[string]*openBar // key: broker:canonical:period
	finalizedBars map[repository.FinalizedKey]map[int64]struct{}
}

type openBar struct {
	bucket int64
	open, high, low, close decimal.Decimal
	bid, ask               decimal.Decimal
	volume float64
	count  uint32
	startTs, endTs int64
	accountID string
}

func NewBarAggregator() *BarAggregator {
	return &BarAggregator{
		bars:          make(map[string]*openBar),
		finalizedBars: make(map[repository.FinalizedKey]map[int64]struct{}),
	}
}

func (a *BarAggregator) LoadFinalizedBars(closeTsMap map[repository.FinalizedKey][]int64) {
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

func (a *BarAggregator) IngestExternalBar(b *mdtick.Bar) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	fk := repository.FinalizedKey{Broker: b.Broker, Canonical: b.Canonical, Period: b.Period}
	if set, ok := a.finalizedBars[fk]; ok {
		if _, exists := set[b.CloseTsUnixMs]; exists {
			barSkippedFinalized.Add(1)
			return false
		}
	}
	if a.finalizedBars[fk] == nil {
		a.finalizedBars[fk] = make(map[int64]struct{})
	}
	a.finalizedBars[fk][b.CloseTsUnixMs] = struct{}{}
	return true
}

func (a *BarAggregator) AddTick(t *mdtick.Tick, onBar func(*mdtick.Bar)) {
	a.mu.Lock()
	defer a.mu.Unlock()

	mid := t.Bid.Add(t.Ask).Div(decimal.NewFromInt(2))

	for _, p := range Periods {
		key := t.Broker + ":" + t.Canonical + ":" + p.Name
		bucket := t.ArrivedUnixMs / p.Ms

		ob := a.bars[key]
		if ob == nil {
			ob = &openBar{bucket: bucket, open: mid, high: mid, low: mid, close: mid, bid: t.Bid, ask: t.Ask, startTs: t.ArrivedUnixMs, accountID: t.AccountID}
			a.bars[key] = ob
		} else if ob.bucket != bucket {
			bar := &mdtick.Bar{
				AccountID: ob.accountID,
				Broker: t.Broker, Canonical: t.Canonical, Period: p.Name,
				OpenTsUnixMs: ob.startTs, CloseTsUnixMs: ob.endTs,
				Open: ob.open, High: ob.high, Low: ob.low, Close: ob.close,
				Bid: ob.bid, Ask: ob.ask,
				Volume: ob.volume, TickCount: ob.count,
				IsClosed: true,
			}
			fk := repository.FinalizedKey{Broker: t.Broker, Canonical: t.Canonical, Period: p.Name}
			if a.finalizedBars[fk] == nil {
				a.finalizedBars[fk] = make(map[int64]struct{})
			}
			a.finalizedBars[fk][bar.CloseTsUnixMs] = struct{}{}
			onBar(bar)
			ob.bucket = bucket
			ob.open = mid; ob.high = mid; ob.low = mid; ob.close = mid; ob.bid = t.Bid; ob.ask = t.Ask; ob.accountID = t.AccountID
			ob.volume = 0; ob.count = 0
			ob.startTs = t.ArrivedUnixMs
		}
		if mid.Cmp(ob.high) > 0 { ob.high = mid }
		if mid.Cmp(ob.low) < 0 { ob.low = mid }
		ob.close = mid
		ob.bid = t.Bid
		ob.ask = t.Ask
		ob.accountID = t.AccountID
		ob.volume += float64(t.BidVolume + t.AskVolume)
		ob.count++
		ob.endTs = t.ArrivedUnixMs
	}
}

// GetOpenBars returns a snapshot of all currently open bars across all periods.
func (a *BarAggregator) GetOpenBars() []*mdtick.Bar {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []*mdtick.Bar
	for key, ob := range a.bars {
		parts := strings.SplitN(key, ":", 3)
		if len(parts) != 3 {
			continue
		}
		out = append(out, &mdtick.Bar{
			AccountID:     ob.accountID,
			Broker:        parts[0],
			Canonical:     parts[1],
			Period:        parts[2],
			OpenTsUnixMs:  ob.startTs,
			CloseTsUnixMs: ob.endTs,
			Open:          ob.open,
			High:          ob.high,
			Low:           ob.low,
			Close:         ob.close,
			Bid:           ob.bid,
			Ask:           ob.ask,
			Volume:        ob.volume,
			TickCount:     ob.count,
		})
	}
	return out
}
