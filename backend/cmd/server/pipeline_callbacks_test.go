package main

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
)

// stubScheduleResolver captures the magic passed to ResolveScheduleIDByMagic
// and returns a deterministic schedule ID for non-zero magic.
type stubScheduleResolver struct {
	id    *uuid.UUID
	err   error
	calls int
	last  int32
}

func (s *stubScheduleResolver) ResolveScheduleIDByMagic(_ context.Context, _ uuid.UUID, magic int32) (*uuid.UUID, error) {
	s.calls++
	s.last = magic
	if magic == 0 {
		return nil, nil
	}
	return s.id, s.err
}

func closedOrderUpdate(magic int32) *mdtick.OrderUpdate {
	return &mdtick.OrderUpdate{
		Platform:         "mt5",
		UpdateType:       "close",
		UpdateTicket:     12345,
		UpdateSymbol:     "EURUSD",
		UpdateOrderType:  "buy",
		UpdateVolume:     decimal.NewFromFloat(0.1),
		UpdateOpenPrice:  decimal.NewFromFloat(1.1),
		UpdateClosePrice: decimal.NewFromFloat(1.105),
		UpdateProfit:     decimal.NewFromFloat(5.0),
		UpdateOpenTime:   1700000000,
		UpdateCloseTime:  1700003600,
		UpdateMagic:      magic,
	}
}

// TestBuildClosedTradeRecordMagicAttribution verifies FIX-2026-08-27 修复 B:
// a closed strategy order (magic != 0) must carry MagicNumber and ScheduleID.
// Adversarial proof: remove the MagicNumber/ScheduleID assignment lines in
// buildClosedTradeRecord and this test turns RED.
func TestBuildClosedTradeRecordMagicAttribution(t *testing.T) {
	sid := uuid.New()
	resolver := &stubScheduleResolver{id: &sid}
	accID := uuid.New()
	userID := uuid.New()
	const magic int32 = 777

	rec := buildClosedTradeRecord(zap.NewNop(), resolver, context.Background(),
		accID.String(), userID.String(), closedOrderUpdate(magic))
	if rec == nil {
		t.Fatal("expected non-nil record for close event")
	}
	if rec.MagicNumber != int(magic) {
		t.Fatalf("MagicNumber: expected %d, got %d", magic, rec.MagicNumber)
	}
	if rec.ScheduleID == nil || *rec.ScheduleID != sid {
		t.Fatalf("ScheduleID: expected %s, got %v", sid, rec.ScheduleID)
	}
	if resolver.calls != 1 {
		t.Fatalf("resolver should be called once for magic!=0, calls=%d", resolver.calls)
	}
	if resolver.last != magic {
		t.Fatalf("resolver magic arg: expected %d, got %d", magic, resolver.last)
	}
}

// TestBuildClosedTradeRecordManualOrderNoSchedule covers the manual-trade
// branch: magic=0 must yield MagicNumber=0 and ScheduleID=nil, and the
// resolver must NOT be consulted (ResolveScheduleID short-circuits).
func TestBuildClosedTradeRecordManualOrderNoSchedule(t *testing.T) {
	resolver := &stubScheduleResolver{id: nil}
	accID := uuid.New()
	userID := uuid.New()

	rec := buildClosedTradeRecord(zap.NewNop(), resolver, context.Background(),
		accID.String(), userID.String(), closedOrderUpdate(0))
	if rec == nil {
		t.Fatal("expected non-nil record for close event")
	}
	if rec.MagicNumber != 0 {
		t.Fatalf("MagicNumber: expected 0 for manual order, got %d", rec.MagicNumber)
	}
	if rec.ScheduleID != nil {
		t.Fatalf("ScheduleID: expected nil for magic=0, got %v", rec.ScheduleID)
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver must not be called for magic=0, calls=%d", resolver.calls)
	}
}

// TestBuildClosedTradeRecordNilResolver verifies best-effort attribution:
// with no resolver injected, MagicNumber is still set and ScheduleID is nil.
func TestBuildClosedTradeRecordNilResolver(t *testing.T) {
	accID := uuid.New()
	userID := uuid.New()
	const magic int32 = 42

	rec := buildClosedTradeRecord(zap.NewNop(), nil, context.Background(),
		accID.String(), userID.String(), closedOrderUpdate(magic))
	if rec == nil {
		t.Fatal("expected non-nil record for close event")
	}
	if rec.MagicNumber != int(magic) {
		t.Fatalf("MagicNumber: expected %d, got %d", magic, rec.MagicNumber)
	}
	if rec.ScheduleID != nil {
		t.Fatalf("ScheduleID: expected nil for nil resolver, got %v", rec.ScheduleID)
	}
}

// TestBuildClosedTradeRecordSkipsNonClose verifies the guard: non-close
// events and invalid IDs return nil without constructing a record.
func TestBuildClosedTradeRecordSkipsNonClose(t *testing.T) {
	resolver := &stubScheduleResolver{}
	accID := uuid.New()
	userID := uuid.New()

	// Non-close event.
	o := closedOrderUpdate(777)
	o.UpdateType = "modify"
	if rec := buildClosedTradeRecord(zap.NewNop(), resolver, context.Background(),
		accID.String(), userID.String(), o); rec != nil {
		t.Fatal("expected nil for non-close event")
	}
	// Close event with zero close time.
	o = closedOrderUpdate(777)
	o.UpdateCloseTime = 0
	if rec := buildClosedTradeRecord(zap.NewNop(), resolver, context.Background(),
		accID.String(), userID.String(), o); rec != nil {
		t.Fatal("expected nil for close with zero close_time")
	}
	// Invalid account ID.
	if rec := buildClosedTradeRecord(zap.NewNop(), resolver, context.Background(),
		"not-a-uuid", userID.String(), closedOrderUpdate(777)); rec != nil {
		t.Fatal("expected nil for invalid account ID")
	}
	// Invalid user ID.
	if rec := buildClosedTradeRecord(zap.NewNop(), resolver, context.Background(),
		accID.String(), "not-a-uuid", closedOrderUpdate(777)); rec != nil {
		t.Fatal("expected nil for invalid user ID")
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver must not be called for skipped records, calls=%d", resolver.calls)
	}
}

// Compile-time guard: stubScheduleResolver satisfies mthub.ScheduleResolver.
var _ mthub.ScheduleResolver = (*stubScheduleResolver)(nil)
