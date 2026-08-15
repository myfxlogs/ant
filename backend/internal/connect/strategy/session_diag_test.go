package strategy

import (
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestSessionDiag_RecordEval(t *testing.T) {
	d := newSessionDiag()
	for i := 0; i < 5; i++ {
		d.RecordEval(evalKindBar)
	}
	for i := 0; i < 3; i++ {
		d.RecordEval(evalKindTick)
	}
	d.RecordEval(evalKindTrade)

	snap := d.SnapshotDiag()
	if snap.EvalCount != 9 {
		t.Errorf("evalCount = %d, want 9", snap.EvalCount)
	}
	if snap.BarCount != 5 {
		t.Errorf("barCount = %d, want 5", snap.BarCount)
	}
	if snap.TickCount != 3 {
		t.Errorf("tickCount = %d, want 3", snap.TickCount)
	}
	if snap.LastEvalAt == 0 {
		t.Error("lastEvalAt should be non-zero after RecordEval")
	}
}

func TestSessionDiag_RecordWindow(t *testing.T) {
	d := newSessionDiag()
	d.RecordWindow(100)
	snap := d.SnapshotDiag()
	if snap.WindowBars != 100 {
		t.Errorf("windowBars = %d, want 100", snap.WindowBars)
	}
}

func TestSessionDiag_RecordIndicators_Throttling(t *testing.T) {
	d := newSessionDiag()
	vals := map[string]decimal.Decimal{
		"iRSI[14,0]": decimal.NewFromFloat(55.5),
	}
	d.RecordIndicators(vals, 2)
	snap := d.SnapshotDiag()
	if snap.OrdersTotalSeen != 2 {
		t.Errorf("ordersTotal = %d, want 2", snap.OrdersTotalSeen)
	}
	if len(snap.Indicators) != 1 {
		t.Errorf("indicators count = %d, want 1", len(snap.Indicators))
	}

	// Second call within 5s should be throttled.
	vals2 := map[string]decimal.Decimal{
		"iRSI[14,0]": decimal.NewFromFloat(60.0),
	}
	d.RecordIndicators(vals2, 3)
	snap2 := d.SnapshotDiag()
	if snap2.OrdersTotalSeen != 2 {
		t.Errorf("ordersTotal after throttle = %d, want 2 (throttled)", snap2.OrdersTotalSeen)
	}
	rsi := snap2.Indicators["iRSI[14,0]"]
	if len(rsi) != 1 || !rsi[0].Equal(decimal.NewFromFloat(55.5)) {
		t.Errorf("indicator value should be unchanged after throttle, got %v", rsi)
	}
}

func TestSessionDiag_RecordIndicators_AfterInterval(t *testing.T) {
	d := newSessionDiag()
	vals := map[string]decimal.Decimal{
		"iMA[14,sma,0]": decimal.NewFromFloat(1.234),
	}
	d.RecordIndicators(vals, 1)

	// Manually advance lastWriteAt to simulate passage of time.
	d.mu.Lock()
	d.lastWriteAt = time.Now().Add(-10 * time.Second).UnixMilli()
	d.mu.Unlock()

	vals2 := map[string]decimal.Decimal{
		"iMA[14,sma,0]": decimal.NewFromFloat(2.345),
	}
	d.RecordIndicators(vals2, 5)
	snap := d.SnapshotDiag()
	if snap.OrdersTotalSeen != 5 {
		t.Errorf("ordersTotal = %d, want 5", snap.OrdersTotalSeen)
	}
	ma := snap.Indicators["iMA[14,sma,0]"]
	if len(ma) != 2 {
		t.Errorf("indicator ring len = %d, want 2", len(ma))
	}
	if !ma[1].Equal(decimal.NewFromFloat(2.345)) {
		t.Errorf("latest indicator = %s, want 2.345", ma[1].String())
	}
}

func TestSessionDiag_IndicatorRingCap(t *testing.T) {
	d := newSessionDiag()
	d.mu.Lock()
	d.lastWriteAt = 0
	d.mu.Unlock()

	for i := 0; i < indicatorRingCap+10; i++ {
		d.mu.Lock()
		d.lastWriteAt = 0
		d.mu.Unlock()
		d.RecordIndicators(map[string]decimal.Decimal{
			"iATR[14]": decimal.NewFromInt(int64(i)),
		}, 0)
	}
	snap := d.SnapshotDiag()
	atr := snap.Indicators["iATR[14]"]
	if len(atr) > indicatorRingCap {
		t.Errorf("ring len = %d, max %d", len(atr), indicatorRingCap)
	}
	if len(atr) != indicatorRingCap {
		t.Errorf("ring len = %d, want %d", len(atr), indicatorRingCap)
	}
}

func TestSessionDiag_MaxIndicatorKeys(t *testing.T) {
	d := newSessionDiag()
	d.mu.Lock()
	d.lastWriteAt = 0
	d.mu.Unlock()

	for i := 0; i < maxIndicatorKeys+10; i++ {
		d.mu.Lock()
		d.lastWriteAt = 0
		d.mu.Unlock()
		d.RecordIndicators(map[string]decimal.Decimal{
			"key_" + string(rune('a'+i)): decimal.Zero,
		}, 0)
	}
	snap := d.SnapshotDiag()
	if len(snap.Indicators) > maxIndicatorKeys {
		t.Errorf("indicator keys = %d, max %d", len(snap.Indicators), maxIndicatorKeys)
	}
}

func TestSessionDiag_Concurrent(t *testing.T) {
	d := newSessionDiag()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			d.RecordEval(evalKindBar)
			d.RecordWindow(n)
			d.RecordIndicators(map[string]decimal.Decimal{
				"iRSI[14,0]": decimal.NewFromInt(int64(n)),
			}, n)
		}(i)
	}
	wg.Wait()
	snap := d.SnapshotDiag()
	if snap.EvalCount != 100 {
		t.Errorf("evalCount = %d, want 100", snap.EvalCount)
	}
}

func TestSessionDiag_SnapshotDeepCopy(t *testing.T) {
	d := newSessionDiag()
	d.mu.Lock()
	d.lastWriteAt = 0
	d.mu.Unlock()
	d.RecordIndicators(map[string]decimal.Decimal{
		"iRSI[14,0]": decimal.NewFromFloat(50.0),
	}, 1)

	snap := d.SnapshotDiag()
	snap.Indicators["iRSI[14,0]"][0] = decimal.NewFromFloat(999.0)

	snap2 := d.SnapshotDiag()
	val := snap2.Indicators["iRSI[14,0]"][0]
	if val.Equal(decimal.NewFromFloat(999.0)) {
		t.Error("snapshot should be a deep copy — modifying it should not affect the original")
	}
}
