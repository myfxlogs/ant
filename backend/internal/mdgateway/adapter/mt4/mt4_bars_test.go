package mt4

import (
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt4"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConvertMT4Bars_Empty(t *testing.T) {
	t.Parallel()
	bars := convertMT4Bars(nil, "acct-1", "1h")
	if len(bars) != 0 {
		t.Errorf("expected 0 bars from nil, got %d", len(bars))
	}
	bars = convertMT4Bars([]*pb.Bar{}, "acct-1", "1h")
	if len(bars) != 0 {
		t.Errorf("expected 0 bars from empty, got %d", len(bars))
	}
}

func TestConvertMT4Bars_WithData(t *testing.T) {
	t.Parallel()
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	pbBars := []*pb.Bar{
		{Time: ts, Open: 1.1000, High: 1.1050, Low: 1.0990, Close: 1.1020, Volume: 100},
		{Time: ts, Open: 1.1020, High: 1.1080, Low: 1.1010, Close: 1.1060, Volume: 200},
	}
	bars := convertMT4Bars(pbBars, "acct-1", "1h")
	if len(bars) != 2 {
		t.Fatalf("expected 2 bars, got %d", len(bars))
	}
	if bars[0].AccountID != "acct-1" {
		t.Errorf("AccountID = %q, want acct-1", bars[0].AccountID)
	}
	if bars[0].Period != "1h" {
		t.Errorf("Period = %q, want 1h", bars[0].Period)
	}
	if !bars[0].Open.Equal(decimal.NewFromFloat(1.1000)) {
		t.Errorf("Open = %s, want 1.1000", bars[0].Open)
	}
	if !bars[0].High.Equal(decimal.NewFromFloat(1.1050)) {
		t.Errorf("High = %s, want 1.1050", bars[0].High)
	}
	if !bars[0].Close.Equal(decimal.NewFromFloat(1.1020)) {
		t.Errorf("Close = %s, want 1.1020", bars[0].Close)
	}
	if bars[0].Volume != 100 {
		t.Errorf("Volume = %f, want 100", bars[0].Volume)
	}
}

// TestBARALIGN_ConvertMT4Bars_SubSecondAlignment verifies BAR-ALIGN:
// mtapi bar Time with sub-second precision must be floored to period boundary.
//
// Adversarial proof: Remove the `openMs -= openMs % pm` alignment line →
// openMs retains sub-second offset → open_ts % periodMs != 0 (RED).

// TestBARALIGN_ConvertMT4Bars_SubSecondAlignment verifies BAR-ALIGN:
// mtapi bar Time with sub-second precision must be floored to period boundary.
//
// Adversarial proof: Remove the `openMs -= openMs % pm` alignment line →
// openMs retains sub-second offset → open_ts % periodMs != 0 (RED).
func TestBARALIGN_ConvertMT4Bars_SubSecondAlignment(t *testing.T) {
	t.Parallel()
	// 2026-01-15 10:00:00.385 UTC → UnixMilli = 1784978400385 (not 5m-aligned, offset 385ms)
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 385_000_000, time.UTC))
	pbBars := []*pb.Bar{
		{Time: ts, Open: 1.1000, High: 1.1050, Low: 1.0990, Close: 1.1020, Volume: 100},
	}
	bars := convertMT4Bars(pbBars, "acct-1", "5m")
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
