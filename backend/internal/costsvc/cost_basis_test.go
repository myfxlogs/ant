package costsvc

import (
	"math"
	"testing"
	"time"
)

func makeTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func closeEnough(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}

// ---- FIFO Tests ----

func TestFIFO_SingleOpenFullClose(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(FIFO)
	tracker.AddOpening("O1", makeTime("2025-01-01T10:00:00Z"), 1.0, 100.0, "buy")

	lots := tracker.Match("C1", 1.0, 110.0, "sell")

	if len(lots) != 1 {
		t.Fatalf("expected 1 lot, got %d", len(lots))
	}
	if lots[0].OpeningTicket != "O1" {
		t.Errorf("OpeningTicket = %s, want O1", lots[0].OpeningTicket)
	}
	if lots[0].Volume != 1.0 {
		t.Errorf("Volume = %f, want 1.0", lots[0].Volume)
	}
	if !closeEnough(lots[0].RealizedPnL, 10.0) {
		t.Errorf("RealizedPnL = %f, want 10.0", lots[0].RealizedPnL)
	}
	if tracker.RemainingVolume() != 0 {
		t.Errorf("RemainingVolume = %f, want 0", tracker.RemainingVolume())
	}
}

func TestFIFO_MultiOpenFIFOOrder(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(FIFO)
	tracker.AddOpening("O1", makeTime("2025-01-01T10:00:00Z"), 1.0, 100.0, "buy")
	tracker.AddOpening("O2", makeTime("2025-01-02T10:00:00Z"), 1.0, 105.0, "buy")
	tracker.AddOpening("O3", makeTime("2025-01-03T10:00:00Z"), 1.0, 110.0, "buy")

	lots := tracker.Match("C1", 2.0, 115.0, "sell")

	if len(lots) != 2 {
		t.Fatalf("expected 2 lots, got %d", len(lots))
	}
	// FIFO: oldest first → O1, then O2
	if lots[0].OpeningTicket != "O1" {
		t.Errorf("first lot OpeningTicket = %s, want O1", lots[0].OpeningTicket)
	}
	if lots[1].OpeningTicket != "O2" {
		t.Errorf("second lot OpeningTicket = %s, want O2", lots[1].OpeningTicket)
	}
	if !closeEnough(lots[0].RealizedPnL, 15.0) { // 115-100
		t.Errorf("O1 RealizedPnL = %f, want 15.0", lots[0].RealizedPnL)
	}
	if !closeEnough(lots[1].RealizedPnL, 10.0) { // 115-105
		t.Errorf("O2 RealizedPnL = %f, want 10.0", lots[1].RealizedPnL)
	}
	// O3 still has volume
	if tracker.RemainingVolume() != 1.0 {
		t.Errorf("RemainingVolume = %f, want 1.0", tracker.RemainingVolume())
	}
}

func TestFIFO_PartialCloseThenRemaining(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(FIFO)
	tracker.AddOpening("O1", makeTime("2025-01-01T10:00:00Z"), 2.0, 100.0, "buy")

	lots := tracker.Match("C1", 0.5, 110.0, "sell")

	if len(lots) != 1 {
		t.Fatalf("expected 1 lot, got %d", len(lots))
	}
	if lots[0].Volume != 0.5 {
		t.Errorf("Volume = %f, want 0.5", lots[0].Volume)
	}
	if tracker.RemainingVolume() != 1.5 {
		t.Errorf("RemainingVolume = %f, want 1.5", tracker.RemainingVolume())
	}

	// Close the rest
	lots2 := tracker.Match("C2", 2.0, 120.0, "sell")
	if len(lots2) != 1 {
		t.Fatalf("expected 1 lot, got %d", len(lots2))
	}
	if lots2[0].Volume != 1.5 {
		t.Errorf("Volume = %f, want 1.5", lots2[0].Volume)
	}
	if tracker.RemainingVolume() != 0 {
		t.Errorf("RemainingVolume = %f, want 0", tracker.RemainingVolume())
	}
}

func TestFIFO_CloseMoreThanOpen(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(FIFO)
	tracker.AddOpening("O1", makeTime("2025-01-01T10:00:00Z"), 1.0, 100.0, "buy")

	// Request more volume than available — matched to what's open
	lots := tracker.Match("C1", 5.0, 110.0, "sell")

	if len(lots) != 1 {
		t.Fatalf("expected 1 lot, got %d", len(lots))
	}
	if lots[0].Volume != 1.0 {
		t.Errorf("Volume = %f, want 1.0", lots[0].Volume)
	}
	if tracker.RemainingVolume() != 0 {
		t.Errorf("RemainingVolume = %f, want 0", tracker.RemainingVolume())
	}
}

func TestFIFO_ShortPositions(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(FIFO)
	tracker.AddOpening("O1", makeTime("2025-01-01T10:00:00Z"), 1.0, 100.0, "sell")

	lots := tracker.Match("C1", 1.0, 90.0, "buy")

	if len(lots) != 1 {
		t.Fatalf("expected 1 lot, got %d", len(lots))
	}
	// Short: P&L = open - close = 100 - 90 = 10 (profit)
	if !closeEnough(lots[0].RealizedPnL, 10.0) {
		t.Errorf("RealizedPnL = %f, want 10.0", lots[0].RealizedPnL)
	}
}

func TestFIFO_MixedSidesDontCross(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(FIFO)
	tracker.AddOpening("O1", makeTime("2025-01-01T10:00:00Z"), 1.0, 100.0, "buy")
	tracker.AddOpening("O2", makeTime("2025-01-02T10:00:00Z"), 1.0, 95.0, "sell")

	// Close the long
	lots := tracker.Match("C1", 1.0, 110.0, "sell")
	if len(lots) != 1 || lots[0].OpeningTicket != "O1" {
		t.Fatalf("expected O1 only, got %v", lots)
	}

	// Close the short
	lots2 := tracker.Match("C2", 1.0, 90.0, "buy")
	if len(lots2) != 1 || lots2[0].OpeningTicket != "O2" {
		t.Fatalf("expected O2 only, got %v", lots2)
	}
}

func TestFIFO_EmptyOpenings(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(FIFO)
	lots := tracker.Match("C1", 1.0, 100.0, "sell")
	if len(lots) != 0 {
		t.Errorf("expected 0 lots, got %d", len(lots))
	}
}

func TestFIFO_RealizedPnLMatchesVolume(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(FIFO)
	tracker.AddOpening("O1", makeTime("2025-01-01T10:00:00Z"), 3.0, 200.0, "buy")

	lots := tracker.Match("C1", 3.0, 210.0, "sell")

	if len(lots) != 1 {
		t.Fatalf("expected 1 lot, got %d", len(lots))
	}
	expectedPnL := (210.0 - 200.0) * 3.0 // 30.0
	if !closeEnough(lots[0].RealizedPnL, expectedPnL) {
		t.Errorf("RealizedPnL = %f, want %f", lots[0].RealizedPnL, expectedPnL)
	}
}

// ---- LIFO Tests ----

func TestLIFO_NewestMatchedFirst(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(LIFO)
	tracker.AddOpening("O1", makeTime("2025-01-01T10:00:00Z"), 1.0, 100.0, "buy")
	tracker.AddOpening("O2", makeTime("2025-01-02T10:00:00Z"), 1.0, 105.0, "buy")
	tracker.AddOpening("O3", makeTime("2025-01-03T10:00:00Z"), 1.0, 110.0, "buy")

	lots := tracker.Match("C1", 2.0, 115.0, "sell")

	if len(lots) != 2 {
		t.Fatalf("expected 2 lots, got %d", len(lots))
	}
	// LIFO: newest first → O3, then O2
	if lots[0].OpeningTicket != "O3" {
		t.Errorf("first lot OpeningTicket = %s, want O3", lots[0].OpeningTicket)
	}
	if lots[1].OpeningTicket != "O2" {
		t.Errorf("second lot OpeningTicket = %s, want O2", lots[1].OpeningTicket)
	}
	if !closeEnough(lots[0].RealizedPnL, 5.0) { // 115-110
		t.Errorf("O3 RealizedPnL = %f, want 5.0", lots[0].RealizedPnL)
	}
	if !closeEnough(lots[1].RealizedPnL, 10.0) { // 115-105
		t.Errorf("O2 RealizedPnL = %f, want 10.0", lots[1].RealizedPnL)
	}
}

func TestLIFO_PartialCloseAcrossMultiple(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(LIFO)
	tracker.AddOpening("O1", makeTime("2025-01-01T10:00:00Z"), 1.0, 100.0, "buy")
	tracker.AddOpening("O2", makeTime("2025-01-02T10:00:00Z"), 1.0, 105.0, "buy")

	// Close 0.7 — should hit O2 (newest) partially
	lots := tracker.Match("C1", 0.7, 110.0, "sell")
	if len(lots) != 1 || lots[0].OpeningTicket != "O2" {
		t.Fatalf("expected O2, got %v", lots)
	}
	if !closeEnough(lots[0].Volume, 0.7) {
		t.Errorf("Volume = %f, want 0.7", lots[0].Volume)
	}

	// Close 1.5 more — O2 remaining 0.3, then O1
	lots2 := tracker.Match("C2", 1.5, 112.0, "sell")
	if len(lots2) != 2 {
		t.Fatalf("expected 2 lots, got %d", len(lots2))
	}
	if lots2[0].OpeningTicket != "O2" || !closeEnough(lots2[0].Volume, 0.3) {
		t.Errorf("first lot expected O2 0.3, got %s %f", lots2[0].OpeningTicket, lots2[0].Volume)
	}
	if lots2[1].OpeningTicket != "O1" || !closeEnough(lots2[1].Volume, 1.0) {
		t.Errorf("second lot expected O1 1.0, got %s %f", lots2[1].OpeningTicket, lots2[1].Volume)
	}
}

// ---- HIFO Tests ----

func TestHIFO_LongHighestCostFirst(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(HIFO)
	tracker.AddOpening("O_low", makeTime("2025-01-01T10:00:00Z"), 1.0, 100.0, "buy")
	tracker.AddOpening("O_mid", makeTime("2025-01-02T10:00:00Z"), 1.0, 105.0, "buy")
	tracker.AddOpening("O_high", makeTime("2025-01-03T10:00:00Z"), 1.0, 110.0, "buy")

	lots := tracker.Match("C1", 2.0, 115.0, "sell")

	if len(lots) != 2 {
		t.Fatalf("expected 2 lots, got %d", len(lots))
	}
	// HIFO long: highest price first → O_high (110), then O_mid (105)
	if lots[0].OpeningTicket != "O_high" {
		t.Errorf("first lot OpeningTicket = %s, want O_high", lots[0].OpeningTicket)
	}
	if lots[1].OpeningTicket != "O_mid" {
		t.Errorf("second lot OpeningTicket = %s, want O_mid", lots[1].OpeningTicket)
	}
	// O_high: (115-110)=5, O_mid: (115-105)=10
	if !closeEnough(lots[0].RealizedPnL, 5.0) {
		t.Errorf("O_high RealizedPnL = %f, want 5.0", lots[0].RealizedPnL)
	}
	if !closeEnough(lots[1].RealizedPnL, 10.0) {
		t.Errorf("O_mid RealizedPnL = %f, want 10.0", lots[1].RealizedPnL)
	}
}

func TestHIFO_ShortLowestPriceFirst(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(HIFO)
	tracker.AddOpening("S_high", makeTime("2025-01-01T10:00:00Z"), 1.0, 110.0, "sell")
	tracker.AddOpening("S_mid", makeTime("2025-01-02T10:00:00Z"), 1.0, 105.0, "sell")
	tracker.AddOpening("S_low", makeTime("2025-01-03T10:00:00Z"), 1.0, 100.0, "sell")

	lots := tracker.Match("C1", 2.0, 95.0, "buy")

	if len(lots) != 2 {
		t.Fatalf("expected 2 lots, got %d", len(lots))
	}
	// HIFO short: lowest open price = highest cost basis → S_low (100), then S_mid (105)
	if lots[0].OpeningTicket != "S_low" {
		t.Errorf("first lot OpeningTicket = %s, want S_low", lots[0].OpeningTicket)
	}
	if lots[1].OpeningTicket != "S_mid" {
		t.Errorf("second lot OpeningTicket = %s, want S_mid", lots[1].OpeningTicket)
	}
	// Short P&L = open - close: S_low (100-95)=5, S_mid (105-95)=10
	if !closeEnough(lots[0].RealizedPnL, 5.0) {
		t.Errorf("S_low RealizedPnL = %f, want 5.0", lots[0].RealizedPnL)
	}
	if !closeEnough(lots[1].RealizedPnL, 10.0) {
		t.Errorf("S_mid RealizedPnL = %f, want 10.0", lots[1].RealizedPnL)
	}
}

func TestHIFO_SamePriceFallsBackToFIFO(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(HIFO)
	tracker.AddOpening("O1", makeTime("2025-01-01T10:00:00Z"), 1.0, 100.0, "buy")
	tracker.AddOpening("O2", makeTime("2025-01-02T10:00:00Z"), 1.0, 100.0, "buy")

	lots := tracker.Match("C1", 1.5, 110.0, "sell")

	if len(lots) != 2 {
		t.Fatalf("expected 2 lots, got %d", len(lots))
	}
	// Same price, stable sort preserves insertion order
	if lots[0].OpeningTicket != "O1" {
		t.Errorf("first lot OpeningTicket = %s, want O1 (stable)", lots[0].OpeningTicket)
	}
}

// ---- Cross-Method Consistency ----

func TestAllMethods_FullCloseSameResult(t *testing.T) {
	t.Parallel()
	// Full close of a single opening produces the same P&L regardless of method.
	openings := []struct {
		ticket string
		ts     time.Time
		vol    float64
		price  float64
		side   string
	}{
		{"O1", makeTime("2025-01-01T10:00:00Z"), 2.0, 100.0, "buy"},
	}

	for _, method := range []CostBasisMethod{FIFO, LIFO, HIFO} {
		tracker := NewCostBasisTracker(method)
		for _, o := range openings {
			tracker.AddOpening(o.ticket, o.ts, o.vol, o.price, o.side)
		}
		lots := tracker.Match("C1", 2.0, 110.0, "sell")
		totalPnL := 0.0
		for _, l := range lots {
			totalPnL += l.RealizedPnL
		}
		expected := 20.0 // (110-100)*2
		if !closeEnough(totalPnL, expected) {
			t.Errorf("%s: total RealizedPnL = %f, want %f", method, totalPnL, expected)
		}
	}
}

func TestAllMethods_DifferentOrdering(t *testing.T) {
	t.Parallel()
	opens := []struct {
		ticket string
		ts     time.Time
		price  float64
	}{
		{"O1", makeTime("2025-01-01T10:00:00Z"), 100.0},
		{"O2", makeTime("2025-01-02T10:00:00Z"), 110.0},
		{"O3", makeTime("2025-01-03T10:00:00Z"), 105.0},
	}

	t.Run("FIFO", func(t *testing.T) {
		tracker := NewCostBasisTracker(FIFO)
		for _, o := range opens {
			tracker.AddOpening(o.ticket, o.ts, 1.0, o.price, "buy")
		}
		lots := tracker.Match("C1", 1.0, 120.0, "sell")
		if lots[0].OpeningTicket != "O1" {
			t.Errorf("FIFO: first = %s, want O1", lots[0].OpeningTicket)
		}
	})

	t.Run("LIFO", func(t *testing.T) {
		tracker := NewCostBasisTracker(LIFO)
		for _, o := range opens {
			tracker.AddOpening(o.ticket, o.ts, 1.0, o.price, "buy")
		}
		lots := tracker.Match("C1", 1.0, 120.0, "sell")
		if lots[0].OpeningTicket != "O3" {
			t.Errorf("LIFO: first = %s, want O3", lots[0].OpeningTicket)
		}
	})

	t.Run("HIFO", func(t *testing.T) {
		tracker := NewCostBasisTracker(HIFO)
		for _, o := range opens {
			tracker.AddOpening(o.ticket, o.ts, 1.0, o.price, "buy")
		}
		lots := tracker.Match("C1", 1.0, 120.0, "sell")
		// HIFO long: highest price first → O2 (110)
		if lots[0].OpeningTicket != "O2" {
			t.Errorf("HIFO: first = %s, want O2", lots[0].OpeningTicket)
		}
	})
}

func TestOpenPositionCount(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(FIFO)
	if tracker.OpenPositionCount() != 0 {
		t.Errorf("OpenPositionCount = %d, want 0", tracker.OpenPositionCount())
	}

	tracker.AddOpening("O1", makeTime("2025-01-01T10:00:00Z"), 2.0, 100.0, "buy")
	tracker.AddOpening("O2", makeTime("2025-01-02T10:00:00Z"), 1.0, 105.0, "buy")
	if tracker.OpenPositionCount() != 2 {
		t.Errorf("OpenPositionCount = %d, want 2", tracker.OpenPositionCount())
	}

	tracker.Match("C1", 2.0, 110.0, "sell")
	// O1 fully closed (vol=2), O2 still has 1.0
	if tracker.OpenPositionCount() != 1 {
		t.Errorf("OpenPositionCount = %d, want 1", tracker.OpenPositionCount())
	}
}

func TestCloseLoss(t *testing.T) {
	t.Parallel()
	tracker := NewCostBasisTracker(FIFO)
	tracker.AddOpening("O1", makeTime("2025-01-01T10:00:00Z"), 1.0, 100.0, "buy")

	lots := tracker.Match("C1", 1.0, 90.0, "sell")

	if !closeEnough(lots[0].RealizedPnL, -10.0) {
		t.Errorf("RealizedPnL = %f, want -10.0", lots[0].RealizedPnL)
	}
}
