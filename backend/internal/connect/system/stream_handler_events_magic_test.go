package system

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

// TestEmitPositionSnapshotMagicNumber verifies FIX-2026-08-28 断裂 2:
// emitPositionSnapshot must populate MagicNumber in OrderUpdateEvent.
// Adversarial proof: remove the MagicNumber assignment in emitPositionSnapshot
// → this test turns RED.
func TestEmitPositionSnapshotMagicNumber(t *testing.T) {
	const magic int32 = -1486243899
	snap := &mthub.PositionSnapshot{
		AccountID: "acc-1",
		Positions: []mthub.PositionSnapshotItem{
			{
				Ticket:    12345,
				Symbol:    "XAUUSD",
				Type:      "buy",
				Magic:     magic,
				Volume:    decimal.NewFromFloat(0.1),
				OpenPrice: decimal.NewFromFloat(2000.5),
				Profit:    decimal.NewFromFloat(5.0),
				OpenTime:  1700000000,
			},
		},
	}
	var captured *antv1.StreamEvent
	sendEvent := func(ev *antv1.StreamEvent) error {
		captured = ev
		return nil
	}
	if err := (&StreamServer{}).emitPositionSnapshot(snap, timestamppb.Now(), sendEvent); err != nil {
		t.Fatalf("emitPositionSnapshot: %v", err)
	}
	if captured == nil || captured.GetPositionSnapshot() == nil {
		t.Fatal("expected PositionSnapshot event")
	}
	positions := captured.GetPositionSnapshot().GetPositions()
	if len(positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(positions))
	}
	if positions[0].MagicNumber != magic {
		t.Fatalf("MagicNumber: expected %d, got %d", magic, positions[0].MagicNumber)
	}
}

// TestOrderRecordToUpdateEventMagic verifies FIX-2026-08-28 断裂 2:
// orderRecordToUpdateEvent must populate MagicNumber from rec.Magic.
// Adversarial proof: remove the MagicNumber assignment → this test turns RED.
func TestOrderRecordToUpdateEventMagic(t *testing.T) {
	const magic int32 = 777
	rec := &mthub.OrderRecord{
		Ticket:    99,
		SymbolRaw: "EURUSD",
		Volume:    decimal.NewFromFloat(0.1),
		OpenPrice: decimal.NewFromFloat(1.1),
		Profit:    decimal.NewFromFloat(5.0),
		Magic:     magic,
		OpenTime:  time.Now(),
	}
	ev := orderRecordToUpdateEvent(rec, "acc-1", "open", rec.Ticket)
	if ev.MagicNumber != magic {
		t.Fatalf("MagicNumber: expected %d, got %d", magic, ev.MagicNumber)
	}
}
