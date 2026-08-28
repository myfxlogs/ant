package system

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
)

// TestOrderHistoryToProtoMagicNumber verifies FIX-2026-08-27 修复 A:
// orderHistoryToProto must populate MagicNumber in the proto response.
// Adversarial proof: remove the MagicNumber assignment in orderHistoryToProto
// → this test turns RED.
func TestOrderHistoryToProtoMagicNumber(t *testing.T) {
	o := &model.OrderHistory{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		AccountID:   uuid.New(),
		Ticket:      12345,
		OrderType:   model.OrderHistoryTypeBuy,
		Symbol:      "EURUSD",
		Volume:      decimal.NewFromFloat(0.1),
		OpenPrice:   decimal.NewFromFloat(1.1),
		ClosePrice:  decimal.NewFromFloat(1.105),
		Profit:      decimal.NewFromFloat(5.0),
		OpenTime:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CloseTime:   &[]time.Time{time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)}[0],
		MagicNumber: 777,
	}
	r := orderHistoryToProto(o)
	if r.MagicNumber != 777 {
		t.Fatalf("MagicNumber: expected 777, got %d", r.MagicNumber)
	}
}

// TestOrderHistoryToProtoMagicNumberZero verifies that magic=0 (manual trade)
// is faithfully passed through as 0, not masked.
func TestOrderHistoryToProtoMagicNumberZero(t *testing.T) {
	o := &model.OrderHistory{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		AccountID: uuid.New(),
		Ticket:    99,
		OrderType: model.OrderHistoryTypeSell,
		Symbol:    "GBPUSD",
		Volume:    decimal.NewFromFloat(0.2),
		OpenPrice: decimal.NewFromFloat(1.2),
		ClosePrice: decimal.NewFromFloat(1.21),
		Profit:    decimal.NewFromFloat(-3.0),
		OpenTime:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	r := orderHistoryToProto(o)
	if r.MagicNumber != 0 {
		t.Fatalf("MagicNumber: expected 0 for manual trade, got %d", r.MagicNumber)
	}
}

// TestOrderHistoryToProtoAllFields verifies the full field mapping from
// model.OrderHistory to antv1.OrderHistoryRecord.
func TestOrderHistoryToProtoAllFields(t *testing.T) {
	id := uuid.New()
	accID := uuid.New()
	schedID := uuid.New()
	closeT := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	o := &model.OrderHistory{
		ID:          id,
		UserID:      uuid.New(),
		AccountID:   accID,
		ScheduleID:  schedID,
		Ticket:      42,
		OrderType:   model.OrderHistoryTypeBuy,
		Symbol:      "XAUUSD",
		Volume:      decimal.NewFromFloat(0.05),
		OpenPrice:   decimal.NewFromFloat(2000.5),
		ClosePrice:  decimal.NewFromFloat(2010.0),
		Profit:      decimal.NewFromFloat(50.0),
		OpenTime:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CloseTime:   &closeT,
		MagicNumber: 123456,
	}
	r := orderHistoryToProto(o)
	if r.Id != id.String() {
		t.Fatalf("Id: expected %s, got %s", id, r.Id)
	}
	if r.AccountId != accID.String() {
		t.Fatalf("AccountId: expected %s, got %s", accID, r.AccountId)
	}
	if r.ScheduleId != schedID.String() {
		t.Fatalf("ScheduleId: expected %s, got %s", schedID, r.ScheduleId)
	}
	if r.Ticket != 42 {
		t.Fatalf("Ticket: expected 42, got %d", r.Ticket)
	}
	if r.Symbol != "XAUUSD" {
		t.Fatalf("Symbol: expected XAUUSD, got %s", r.Symbol)
	}
	if r.OrderType != "buy" {
		t.Fatalf("OrderType: expected buy, got %s", r.OrderType)
	}
	if r.Lots != "0.05" {
		t.Fatalf("Lots: expected 0.05, got %s", r.Lots)
	}
	if r.OpenPrice != "2000.5" {
		t.Fatalf("OpenPrice: expected 2000.5, got %s", r.OpenPrice)
	}
	if r.ClosePrice != "2010" {
		t.Fatalf("ClosePrice: expected 2010, got %s", r.ClosePrice)
	}
	if r.Profit != "50" {
		t.Fatalf("Profit: expected 50, got %s", r.Profit)
	}
	if r.MagicNumber != 123456 {
		t.Fatalf("MagicNumber: expected 123456, got %d", r.MagicNumber)
	}
	if r.OpenTime == nil {
		t.Fatal("OpenTime should not be nil")
	}
	if r.CloseTime == nil {
		t.Fatal("CloseTime should not be nil")
	}
}

// Compile-time guard: ensure orderHistoryToProto returns the right type.
var _ *antv1.OrderHistoryRecord = orderHistoryToProto(&model.OrderHistory{})
