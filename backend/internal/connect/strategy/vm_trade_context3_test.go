package strategy

import (
	"testing"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/sdk"
)

// ── VM-TRADE-CONTEXT-3/4 behavior tests ──────────────────────────────

// TestVM_SignalToProto_OppositeTicket verifies that vmSignalToProto
// preserves OppositeTicket for CloseBy signals.
// VM-TRADE-CONTEXT-3: previously OppositeTicket was lost in proto conversion.
func TestVM_SignalToProto_OppositeTicket(t *testing.T) {
	sig := &sdk.Signal{
		Action:         sdk.ActionClose,
		OrderTicket:    100,
		OppositeTicket: 200,
	}
	ss := vmSignalToProto(sig, "EURUSD")
	if ss == nil {
		t.Fatal("vmSignalToProto returned nil")
	}
	if ss.OppositeTicket != 200 {
		t.Errorf("OppositeTicket=%d, want 200 (CloseBy must preserve both tickets)", ss.OppositeTicket)
	}
	if ss.ExecutedTicket != 100 {
		t.Errorf("ExecutedTicket=%d, want 100", ss.ExecutedTicket)
	}
}

// TestVM_SignalToProto_MagicAndDeviation verifies that vmSignalToProto
// preserves EA-configured Magic and Deviation.
// VM-TRADE-CONTEXT-4: previously Magic/Deviation were lost in proto conversion.
func TestVM_SignalToProto_MagicAndDeviation(t *testing.T) {
	sig := &sdk.Signal{
		Action:    sdk.ActionBuy,
		Magic:     50001,
		Deviation: 20,
	}
	ss := vmSignalToProto(sig, "EURUSD")
	if ss == nil {
		t.Fatal("vmSignalToProto returned nil")
	}
	if ss.Magic != 50001 {
		t.Errorf("Magic=%d, want 50001 (EA magic must be preserved)", ss.Magic)
	}
	if ss.Deviation != 20 {
		t.Errorf("Deviation=%d, want 20 (EA deviation must be preserved)", ss.Deviation)
	}
}

// TestVM_SignalToProto_ZeroMagicKeepsZero verifies that zero magic (legacy
// callers) is preserved as zero, not overwritten.
func TestVM_SignalToProto_ZeroMagicKeepsZero(t *testing.T) {
	sig := &sdk.Signal{
		Action: sdk.ActionBuy,
		Magic:  0, // legacy — no EA magic configured
	}
	ss := vmSignalToProto(sig, "EURUSD")
	if ss.Magic != 0 {
		t.Errorf("Magic=%d, want 0 (zero magic should be preserved for legacy callers)", ss.Magic)
	}
}

// unused import guard
var _ antv1.StrategySignal
