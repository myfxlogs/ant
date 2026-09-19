package runner

// VM-LIVE-PARITY-F1 passthrough test (T5): brokerImpl.OrderSend must return
// the executor's broker fill facts — never echo the request values.
// Mutation M1: restore `Volume: req.Volume, Price: req.Price` fabrication
// → RED (request values deliberately differ from executor facts).

import (
	"testing"

	"alphaforge/strategy/sdk"

	"github.com/shopspring/decimal"
)

func TestOrderSend_PassthroughBrokerFacts(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{}
	r.broker.executor = exec

	res, err := r.broker.OrderSend(sdk.OrderRequest{
		Symbol:    "BTCUSDm",
		Side:      sdk.SideBuy,
		Type:      sdk.OrderMarket,
		Volume:    decimal.NewFromFloat(0.015), // request volume — broker normalizes
		Price:     decimal.Zero,                // market request — no price
		Comment:   "PARITY-X",
		Magic:     42,
		Deviation: 3,
	})
	if err != nil {
		t.Fatalf("OrderSend: %v", err)
	}
	if res.RetCode != sdk.RetDone {
		t.Fatalf("RetCode = %s, want done", res.RetCode)
	}
	// Executor returns {Price: 81262.24, Volume: 0.02} — request was 0/0.015.
	if !res.Price.Equal(decimal.RequireFromString("81262.24")) {
		t.Errorf("Price = %s, want executor fill fact 81262.24 (request was 0)", res.Price)
	}
	if !res.Volume.Equal(decimal.RequireFromString("0.02")) {
		t.Errorf("Volume = %s, want executor fill fact 0.02 (request was 0.015)", res.Volume)
	}
	if res.Ticket != 42 {
		t.Errorf("Ticket = %d, want 42", res.Ticket)
	}
}

// T5b: deviation rides the passthrough — the executor receives what the
// strategy requested.
func TestOrderSend_PassthroughDeviation(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{}
	r.broker.executor = exec

	if _, err := r.broker.OrderSend(sdk.OrderRequest{
		Symbol: "BTCUSDm", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: decimal.NewFromFloat(0.1), Deviation: 7,
	}); err != nil {
		t.Fatalf("OrderSend: %v", err)
	}
	if exec.lastPlace.Deviation != 7 {
		t.Errorf("executor saw Deviation = %d, want 7 (request deviation must reach the executor)", exec.lastPlace.Deviation)
	}
	if exec.lastPlace.Comment != "" || exec.lastPlace.Magic != 0 {
		t.Errorf("unexpected comment/magic: %q/%d", exec.lastPlace.Comment, exec.lastPlace.Magic)
	}
}
