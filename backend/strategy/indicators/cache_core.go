package indicators

// SeriesCache lazily computes and incrementally updates indicator series.
// It turns O(n) per-call indicator calculations into O(1) amortized by
// maintaining incremental state across bar updates.
//
// Usage: create once with a BarSource, call EnsureUpdated() before any
// query. Series are created lazily on first access per (indicator, params) key.
type SeriesCache struct {
	src BarSource
	n   int // number of bars processed

	// lastRev tracks the Revision() of a RevisionedBarSource. hasRev=false
	// means no revision has been recorded yet (first call or after reset).
	// For non-revisioned sources, hasRev stays false and lastRev stays 0.
	lastRev uint64
	hasRev  bool

	ema     map[int]*emaSeries
	smma    map[int]*emaSeries // SMMA uses same incremental pattern with different alpha
	sma     map[int]*smaSeries
	lwma    map[int]*lwmaSeries
	rsi     map[int]*rsiSeries
	atr     map[int]*atrSeries
	adx     map[int]*adxSeries
	macd    map[string]*macdSeries
	chaikin map[string]*chaikinSeries
	ad      *adSeries
	obv     *obvSeries
	sar     map[string]*sarSeries
	force   map[string]*forceSeries
	ama     map[string]*amaSeries
	dema    map[int]*demaSeries
	tema    map[int]*temaSeries
}

// NewSeriesCache creates a cache backed by the given BarSource.
// For append-only sources (backtest), incremental updates work via Len() growth.
// For RevisionedBarSource sources (live rolling window), any Revision() change
// triggers a full reset + lazy rebuild — content may change at constant Len().
func NewSeriesCache(src BarSource) *SeriesCache {
	return &SeriesCache{
		src:     src,
		ema:     make(map[int]*emaSeries),
		smma:    make(map[int]*emaSeries),
		sma:     make(map[int]*smaSeries),
		lwma:    make(map[int]*lwmaSeries),
		rsi:     make(map[int]*rsiSeries),
		atr:     make(map[int]*atrSeries),
		adx:     make(map[int]*adxSeries),
		macd:    make(map[string]*macdSeries),
		chaikin: make(map[string]*chaikinSeries),
		sar:     make(map[string]*sarSeries),
		force:   make(map[string]*forceSeries),
		ama:     make(map[string]*amaSeries),
		dema:    make(map[int]*demaSeries),
		tema:    make(map[int]*temaSeries),
	}
}

// EnsureUpdated processes any new bars since the last update.
// Must be called before any query method.
//
// For RevisionedBarSource sources (e.g. live rolling window), any revision
// change triggers a full reset — the content has changed even if Len() is
// unchanged, so incremental state is invalid. After reset, series are rebuilt
// lazily on first query. For non-revisioned sources (backtest append-only),
// the existing n>c.n incremental / n<c.n reset path is preserved.
func (c *SeriesCache) EnsureUpdated() {
	// Revisioned source: detect content change via Revision().
	// Must check before the Len()-based logic — a rolling window keeps
	// constant length while content changes, so Len() alone cannot detect it.
	if rs, ok := c.src.(RevisionedBarSource); ok {
		rev := rs.Revision()
		if c.hasRev && rev != c.lastRev {
			c.reset()
		}
		c.lastRev = rev
		c.hasRev = true
	}

	n := c.src.Len()
	if n < c.n {
		// BarSource shrank (non-revisioned fallback) — reset everything.
		c.reset()
	}

	// Process new bars in chronological order (oldest first).
	// BarSource index 0 = newest, n-1 = oldest.
	// Chronological order: n-1 (oldest) down to 0 (newest).
	// We already processed c.n bars (indices n-1 down to n-c.n).
	// New bars are at indices n-c.n-1 down to 0.
	for i := n - c.n - 1; i >= 0; i-- {
		c.processBar(i)
	}
	c.n = n
}

// reset clears all cached series state so the next query rebuilds from scratch.
// This is the single source of truth for what state must be cleared — having
// two reset lists would drift as new series are added.
func (c *SeriesCache) reset() {
	c.n = 0
	c.lastRev = 0
	c.hasRev = false
	c.ema = make(map[int]*emaSeries)
	c.smma = make(map[int]*emaSeries)
	c.sma = make(map[int]*smaSeries)
	c.lwma = make(map[int]*lwmaSeries)
	c.rsi = make(map[int]*rsiSeries)
	c.atr = make(map[int]*atrSeries)
	c.adx = make(map[int]*adxSeries)
	c.macd = make(map[string]*macdSeries)
	c.chaikin = make(map[string]*chaikinSeries)
	c.ad = nil
	c.obv = nil
	c.sar = make(map[string]*sarSeries)
	c.force = make(map[string]*forceSeries)
	c.ama = make(map[string]*amaSeries)
	c.dema = make(map[int]*demaSeries)
	c.tema = make(map[int]*temaSeries)
}

func (c *SeriesCache) processBar(bsIdx int) {
	close, _ := c.src.Close(bsIdx).Float64()
	high, _ := c.src.High(bsIdx).Float64()
	low, _ := c.src.Low(bsIdx).Float64()
	vol := float64(c.src.Volume(bsIdx))

	for _, s := range c.ema {
		s.update(close)
	}
	for _, s := range c.smma {
		s.update(close)
	}
	for _, s := range c.sma {
		s.update(close)
	}
	for _, s := range c.lwma {
		s.update(close)
	}
	for _, s := range c.rsi {
		s.update(close)
	}
	for _, s := range c.atr {
		s.update(high, low, close)
	}
	for _, s := range c.adx {
		s.update(high, low, close)
	}
	for _, s := range c.macd {
		s.update(close)
	}
	for _, s := range c.chaikin {
		s.update(high, low, close, vol)
	}
	if c.ad != nil {
		c.ad.update(high, low, close, vol)
	}
	if c.obv != nil {
		c.obv.update(close, vol)
	}
	for _, s := range c.sar {
		s.update(high, low)
	}
	for _, s := range c.force {
		s.update(close, vol)
	}
	for _, s := range c.ama {
		s.update(close)
	}
	for _, s := range c.dema {
		s.update(close)
	}
	for _, s := range c.tema {
		s.update(close)
	}
}

// ── Rebuild helpers (lazy init from full history) ──────────────────

type barUpdater func(close float64)

func (c *SeriesCache) rebuild(upd barUpdater, _ int) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		close, _ := c.src.Close(i).Float64()
		upd(close)
	}
}
