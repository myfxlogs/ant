package mdgateway

import (
	"strings"
	"sync"

	"github.com/shopspring/decimal"
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/repository"
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
	volume                 decimal.Decimal
	count                  uint32
	startTs, endTs         int64
	accountID              string
	symbolRaw              string
}

func (ob *openBar) reset(bucket int64, mid decimal.Decimal, t *mdtick.Tick, periodMs int64) {
	ob.bucket = bucket
	ob.open, ob.high, ob.low, ob.close = mid, mid, mid, mid
	ob.bid, ob.ask = t.Bid, t.Ask
	ob.volume = decimal.Zero
	ob.count = 0
	ob.accountID = t.AccountID
	ob.symbolRaw = t.SymbolRaw
	ob.startTs = bucket * periodMs
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
			ob = &openBar{}
			ob.reset(bucket, mid, t, p.Ms)
			a.bars[key] = ob
		} else if ob.bucket != bucket {
			bar := &mdtick.Bar{
				AccountID: ob.accountID,
				Broker: t.Broker, SymbolRaw: t.SymbolRaw, Canonical: t.Canonical, Period: p.Name,
				OpenTsUnixMs: ob.bucket * p.Ms, CloseTsUnixMs: (ob.bucket + 1) * p.Ms,
				Open: ob.open, High: ob.high, Low: ob.low, Close: ob.close,
				Bid: ob.bid, Ask: ob.ask,
				Volume: ob.volume.InexactFloat64(), TickCount: ob.count,
				IsClosed: true,
				IsReplay: t.IsReplay,
			}
			fk := repository.FinalizedKey{Broker: t.Broker, Canonical: t.Canonical, Period: p.Name}
			if a.finalizedBars[fk] == nil {
				a.finalizedBars[fk] = make(map[int64]struct{})
			}
			a.finalizedBars[fk][bar.CloseTsUnixMs] = struct{}{}
			onBar(bar)
			ob.reset(bucket, mid, t, p.Ms)
		}
		if mid.Cmp(ob.high) > 0 { ob.high = mid }
		if mid.Cmp(ob.low) < 0 { ob.low = mid }
		ob.close = mid
		ob.bid = t.Bid
		ob.ask = t.Ask
		ob.accountID = t.AccountID
		ob.symbolRaw = t.SymbolRaw
		ob.volume = ob.volume.Add(decimal.NewFromFloat(float64(t.BidVolume + t.AskVolume)))
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
			SymbolRaw:     ob.symbolRaw,
			Canonical:     parts[1],
			Period:        parts[2],
			OpenTsUnixMs:  ob.bucket * mdtick.PeriodMs(parts[2]),
			CloseTsUnixMs: (ob.bucket + 1) * mdtick.PeriodMs(parts[2]),
			Open:          ob.open,
			High:          ob.high,
			Low:           ob.low,
			Close:         ob.close,
			Bid:           ob.bid,
			Ask:           ob.ask,
			Volume:        ob.volume.InexactFloat64(),
			TickCount:     ob.count,
		})
	}
	return out
}

// RestoreOpenBars restores in-progress bar state after a process restart.
// For each latest finalized bar, if the current time bucket is the one immediately
// after the finalized bar's bucket, an openBar is recreated using the finalized
// bar's close as the initial OHLC. This minimizes the data gap caused by restart.
// Subsequent ticks will update the restored bar normally.
func (a *BarAggregator) RestoreOpenBars(bars []repository.KlineBar, nowMs int64) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	restored := 0
	for _, b := range bars {
		for _, p := range Periods {
			if p.Name != b.Period {
				continue
			}
			finalizedBucket := int64(b.CloseTsUnixMs) / p.Ms
			currentBucket := nowMs / p.Ms
			// Only restore if the current bucket is exactly the next one after
			// the finalized bar. If more buckets have passed, the gap is too large
			// and the next tick will start a fresh bar anyway.
			if currentBucket != finalizedBucket+1 {
				continue
			}
			key := b.Broker + ":" + b.Canonical + ":" + p.Name
			if a.bars[key] != nil {
				continue // already has an open bar (e.g. from a prior restore)
			}
			a.bars[key] = &openBar{
				bucket:    currentBucket,
				open:      b.Close,
				high:      b.Close,
				low:       b.Close,
				close:     b.Close,
				startTs:   currentBucket * p.Ms,
				endTs:     currentBucket * p.Ms,
				accountID: "", // will be updated by the first tick
			}
			restored++
		}
	}
	return restored
}
