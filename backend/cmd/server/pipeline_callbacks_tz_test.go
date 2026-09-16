package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TestBuildClosedTradeRecord_UTCTimes pins TZ-MIXED-ENCODING-1: live-stream
// trade records must carry UTC-located times. trade_records.open_time /
// close_time are `timestamp` columns whose other writers (history sync,
// order import) store UTC wall clock; pgx encodes a `timestamp` param by
// wall-clock components, so a Local (CST) time would persist +8h skewed.
// Mutation proof: dropping .UTC() in buildClosedTradeRecord turns this RED.
func TestBuildClosedTradeRecord_UTCTimes(t *testing.T) {
	rec := buildClosedTradeRecord(zap.NewNop(), nil, nil, context.Background(),
		uuid.New().String(), uuid.New().String(), closedOrderUpdate(777))
	if rec == nil {
		t.Fatal("expected non-nil record for close event")
	}
	if loc := rec.OpenTime.Location(); loc != time.UTC {
		t.Fatalf("OpenTime location = %v, want time.UTC", loc)
	}
	if loc := rec.CloseTime.Location(); loc != time.UTC {
		t.Fatalf("CloseTime location = %v, want time.UTC", loc)
	}
	// Wall-clock check (what pgx will encode): must equal the epoch instant
	// expressed in UTC, not in time.Local.
	wantOpen := time.Unix(1700000000, 0).UTC()
	wantClose := time.Unix(1700003600, 0).UTC()
	if !rec.OpenTime.Equal(wantOpen) || rec.OpenTime.Hour() != wantOpen.Hour() {
		t.Fatalf("OpenTime = %v, want UTC wall clock of %v", rec.OpenTime, wantOpen)
	}
	if !rec.CloseTime.Equal(wantClose) || rec.CloseTime.Hour() != wantClose.Hour() {
		t.Fatalf("CloseTime = %v, want UTC wall clock of %v", rec.CloseTime, wantClose)
	}
}
