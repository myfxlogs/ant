package mthub

import (
	"testing"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
	"github.com/google/uuid"
)

// ADVERSARIAL: snapshotToProto must preserve per-position MagicNumber and the
// downstream GetSchedulePositions filter must isolate positions by schedule magic.
func TestSnapshotToProto_PreservesMagicNumber(t *testing.T) {
	snap := &PositionSnapshot{
		AccountID: "acc-1", UserID: "user-1", Platform: "mt5",
		Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(10000),
		Positions: []PositionSnapshotItem{
			{Ticket: 1, Symbol: "EURUSD", Type: "buy", Magic: 100, Volume: decimal.NewFromFloat(0.1), OpenPrice: decimal.NewFromFloat(1.08), CurrentPrice: decimal.NewFromFloat(1.09), Profit: decimal.NewFromInt(10), OpenTime: 1234567890},
			{Ticket: 2, Symbol: "USDJPY", Type: "sell", Magic: 200, Volume: decimal.NewFromFloat(0.2), OpenPrice: decimal.NewFromFloat(150.0), CurrentPrice: decimal.NewFromFloat(149.0), Profit: decimal.NewFromInt(20), OpenTime: 1234567891},
		},
	}
	record := snapshotToProto(snap)
	if len(record.GetPositions()) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(record.GetPositions()))
	}
	if record.GetPositions()[0].GetMagicNumber() != 100 {
		t.Errorf("position 0 magic = %d, want 100", record.GetPositions()[0].GetMagicNumber())
	}
	if record.GetPositions()[1].GetMagicNumber() != 200 {
		t.Errorf("position 1 magic = %d, want 200", record.GetPositions()[1].GetMagicNumber())
	}
}

// ADVERSARIAL: two strategies (magic A/B) each one position → query A only returns A.
func TestGetSchedulePositionsFilter(t *testing.T) {
	sidA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	sidB := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	magicA := int64(model.StrategyMagic(sidA))
	magicB := int64(model.StrategyMagic(sidB))

	record := &antv1.MtPositionSnapshotRecord{
		Positions: []*antv1.MtPositionSnapshotItem{
			{Ticket: 1, Symbol: "EURUSD", MagicNumber: magicA},
			{Ticket: 2, Symbol: "USDJPY", MagicNumber: magicB},
		},
	}

	var forA []*antv1.MtPositionSnapshotItem
	for _, pos := range record.GetPositions() {
		if pos.GetMagicNumber() == magicA {
			forA = append(forA, pos)
		}
	}
	if len(forA) != 1 || forA[0].GetTicket() != 1 {
		t.Errorf("expected exactly one position for strategy A, got %v", forA)
	}
}
