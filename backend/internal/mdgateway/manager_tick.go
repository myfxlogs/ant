package mdgateway

import (
	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// HandleTick processes a tick through the full 6-stage pipeline:
// normalize → quality → dedup → aggregate → publish → enqueue.
func (m *Manager) HandleTick(t *mdtick.Tick) {
	ctx := m.baseContext()

	if m.stuffingDetector != nil {
		if m.stuffingDetector.IsPaused(t.Broker, t.Canonical) {
			return
		}
	}

	_, span1 := m.startTrace(ctx, "normalize")
	t.Canonical = m.normalizer.Resolve(ctx, t.Broker, t.SymbolRaw)
	span1.End()

	_, span2 := m.startTrace(ctx, "quality")
	qr := m.quality.Check(ctx, t)
	span2.End()
	if qr.Dropped {
		return
	}

	if m.stuffingDetector != nil {
		if stuffed, _ := m.stuffingDetector.Observe(t.Broker, t.Canonical); stuffed {
			return
		}
	}

	if qr.SpreadBps > 0 && m.quality != nil {
		key := t.Broker + ":" + t.Canonical
		m.quality.trackSpread(key, qr.SpreadBps)
		z := m.quality.SpreadZscore(key, qr.SpreadBps)
		if z > m.quality.cfg.MaxSpreadZscore {
			RecordSpreadAnomaly()
		}
	}

	_, span3 := m.startTrace(ctx, "dedup")
	seen := m.dedup.Seen(t)
	span3.End()
	if seen {
		return
	}

	m.mu.Lock()
	m.lastTickAt[t.AccountID] = Clk.Now().UnixMilli()
	m.mu.Unlock()

	if m.marketState != nil {
		m.marketState.Update(t)
	}

	_, span4 := m.startTrace(ctx, "aggregate")
	var bars []*mdtick.Bar
	m.aggregator.AddTick(t, func(b *mdtick.Bar) { bars = append(bars, b) })
	span4.End()

	_, span5 := m.startTrace(ctx, "publish")
	if err := m.publisher.PublishTick(ctx, t); err != nil && m.log != nil {
		m.log.Warn("mdgateway: PublishTick failed", zap.String("account", t.AccountID), zap.String("symbol", t.Canonical), zap.Error(err))
	}
	for _, b := range bars {
		if err := m.publisher.PublishBar(ctx, b); err != nil && m.log != nil {
			m.log.Warn("mdgateway: PublishBar failed", zap.String("account", b.AccountID), zap.String("symbol", b.Canonical), zap.Error(err))
		}
		if m.onBar != nil {
			m.onBar(b)
		}
	}
	span5.End()

	_, span6 := m.startTrace(ctx, "enqueue")
	m.chWriter.EnqueueTick(t)
	for _, b := range bars {
		m.chWriter.EnqueueBar(b)
	}
	span6.End()
}
