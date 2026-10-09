package mt5

import (
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt5"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConvertMT5Bars_Empty(t *testing.T) {
	t.Parallel()
	bars := convertMT5Bars(nil, "acct-5", "1h")
	if len(bars) != 0 {
		t.Errorf("expected 0 bars from nil, got %d", len(bars))
	}
	bars = convertMT5Bars([]*pb.Bar{}, "acct-5", "1h")
	if len(bars) != 0 {
		t.Errorf("expected 0 bars from empty, got %d", len(bars))
	}
}

func TestConvertMT5Bars_WithData(t *testing.T) {
	t.Parallel()
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	pbBars := []*pb.Bar{
		{Time: ts, OpenPrice: 1.1000, HighPrice: 1.1050, LowPrice: 1.0990, ClosePrice: 1.1020, Volume: 100, TickVolume: 50},
		{Time: ts, OpenPrice: 1.1020, HighPrice: 1.1080, LowPrice: 1.1010, ClosePrice: 1.1060, Volume: 200, TickVolume: 80},
	}
	bars := convertMT5Bars(pbBars, "acct-5", "1h")
	if len(bars) != 2 {
		t.Fatalf("expected 2 bars, got %d", len(bars))
	}
	if bars[0].AccountID != "acct-5" {
		t.Errorf("AccountID = %q, want acct-5", bars[0].AccountID)
	}
	if bars[0].Period != "1h" {
		t.Errorf("Period = %q, want 1h", bars[0].Period)
	}
	if !bars[0].Open.Equal(decimal.NewFromFloat(1.1000)) {
		t.Errorf("Open = %s, want 1.1000", bars[0].Open)
	}
	if bars[0].Volume != 100 {
		t.Errorf("Volume = %f, want 100", bars[0].Volume)
	}
	if bars[0].TickCount != 50 {
		t.Errorf("TickCount = %d, want 50", bars[0].TickCount)
	}
}

// TestBARALIGN_ConvertMT5Bars_SubSecondAlignment verifies BAR-ALIGN:
// mtapi bar Time with sub-second precision must be floored to period boundary.
//
// Adversarial proof: Remove the `openMs -= openMs % pm` alignment line →
// openMs retains sub-second offset → open_ts % periodMs != 0 (RED).

// TestBARALIGN_ConvertMT5Bars_SubSecondAlignment verifies BAR-ALIGN:
// mtapi bar Time with sub-second precision must be floored to period boundary.
//
// Adversarial proof: Remove the `openMs -= openMs % pm` alignment line →
// openMs retains sub-second offset → open_ts % periodMs != 0 (RED).
func TestBARALIGN_ConvertMT5Bars_SubSecondAlignment(t *testing.T) {
	t.Parallel()
	// 2026-01-15 10:00:00.385 UTC → UnixMilli = 1784978400385 (not 5m-aligned, offset 385ms)
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 385_000_000, time.UTC))
	pbBars := []*pb.Bar{
		{Time: ts, OpenPrice: 1.1000, HighPrice: 1.1050, LowPrice: 1.0990, ClosePrice: 1.1020, Volume: 100, TickVolume: 50},
	}
	bars := convertMT5Bars(pbBars, "acct-5", "5m")
	if len(bars) != 1 {
		t.Fatalf("expected 1 bar, got %d", len(bars))
	}
	pm := mdtick.PeriodMs("5m") // 300000
	if bars[0].OpenTsUnixMs%pm != 0 {
		t.Fatalf("BAR-ALIGN: open_ts %d not aligned to 5m boundary (remainder %d) — RED: sub-second offset not floored",
			bars[0].OpenTsUnixMs, bars[0].OpenTsUnixMs%pm)
	}
	wantOpen := int64(1768471200000) // 2026-01-15 10:00:00.000 UTC, floored to 5m
	if bars[0].OpenTsUnixMs != wantOpen {
		t.Fatalf("open_ts = %d, want %d (floored to 5m)", bars[0].OpenTsUnixMs, wantOpen)
	}
	if bars[0].CloseTsUnixMs != wantOpen+pm {
		t.Fatalf("close_ts = %d, want %d (open+periodMs)", bars[0].CloseTsUnixMs, wantOpen+pm)
	}
}
