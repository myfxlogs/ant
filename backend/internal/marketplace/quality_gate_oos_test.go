// quality_gate_oos_test.go — LOWPRI-SWEEP-1 S6/T6（TUNING-OVERFIT-2）(2026-09-19).
//
// The OOS degradation gate was a silent fail-open: thresholds were armed in
// system_config, but production BacktestSnapshots never carried OOS fields,
// so the gate never fired and nobody could tell. evaluateOosGate now logs a
// warning in that state. These tests use a zap observer to capture the log.
//
// Adversarial (T6 mutation): delete the warn block in evaluateOosGate →
// no log entry → RED.
package marketplace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// newObservedService builds a Service whose logger writes to a zap observer
// (pg left nil — evaluateOosGate does not touch the DB).
func newObservedService() (*Service, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.WarnLevel)
	return &Service{log: zap.New(core)}, logs
}

func armedGates() qualityGates {
	return qualityGates{
		MaxIsOosDegradation: decimal.NewFromFloat(0.5),
	}
}

// TestEvaluateOosGate_ArmedButNoOosDataWarns — T6: gate armed + snapshot
// without OOS data → exactly one warning naming the gate.
//
// Adversarial: delete the warn block → zero entries → RED.
func TestEvaluateOosGate_ArmedButNoOosDataWarns(t *testing.T) {
	s, logs := newObservedService()
	snap := &antv1.BacktestSnapshot{
		SharpeRatio: "1.0",
		TotalReturn: "0.3",
		// OosSharpeRatio/OosTotalReturn empty → gate skipped with warning.
	}

	violations := s.evaluateOosGate("strategy-1", armedGates(), snap)

	assert.Empty(t, violations, "no OOS data = gate skipped, not a violation")
	matching := logs.FilterMessage("OOS degradation gate armed but snapshot lacks OOS data; gate skipped")
	require.Equal(t, 1, matching.Len(), "expected exactly one stall warning, got %d", matching.Len())
	entry := matching.All()[0]
	assert.Equal(t, "max_is_oos_degradation", contextValue(t, entry, "gate"))
}

// TestEvaluateOosGate_OosDataPresentNoWarn — with OOS data present the gate
// evaluates normally (in-bounds) and stays silent.
func TestEvaluateOosGate_OosDataPresentNoWarn(t *testing.T) {
	s, logs := newObservedService()
	snap := &antv1.BacktestSnapshot{
		SharpeRatio:    "2.0",
		OosSharpeRatio: "1.8", // degradation 0.1 < 0.5
		TotalReturn:    "0.4",
		OosTotalReturn: "0.35",
	}

	violations := s.evaluateOosGate("strategy-1", armedGates(), snap)

	assert.Empty(t, violations, "in-bounds OOS must not violate")
	assert.Zero(t, logs.Len(), "no warnings expected for healthy OOS data")
}

// TestEvaluateOosGate_OosDegradationViolates — out-of-bounds OOS data
// produces violations (sharpe + return) — the gate actually fires when fed.
func TestEvaluateOosGate_OosDegradationViolates(t *testing.T) {
	s, logs := newObservedService()
	snap := &antv1.BacktestSnapshot{
		SharpeRatio:    "2.0",
		OosSharpeRatio: "0.4", // degradation 0.8 > 0.5
		TotalReturn:    "0.4",
		OosTotalReturn: "0.1", // degradation 0.75 > 0.5
	}

	violations := s.evaluateOosGate("strategy-1", armedGates(), snap)

	require.Len(t, violations, 2, "both sharpe and return degradation must violate")
	for _, v := range violations {
		assert.Contains(t, v.Metric, "is_oos")
	}
	assert.Zero(t, logs.Len(), "real violations are not warnings")
}

// TestEvaluateOosGate_DisabledGateSilent — gate not armed → fully silent.
func TestEvaluateOosGate_DisabledGateSilent(t *testing.T) {
	s, logs := newObservedService()
	gates := qualityGates{} // MaxIsOosDegradation zero = not armed

	violations := s.evaluateOosGate("strategy-1", gates, &antv1.BacktestSnapshot{})

	assert.Empty(t, violations)
	assert.Zero(t, logs.Len(), "disabled gate must be fully silent")
}

// contextValue extracts a structured context field from a zap entry.
func contextValue(t *testing.T, e observer.LoggedEntry, key string) string {
	t.Helper()
	for _, f := range e.Context {
		if f.Key == key {
			return f.String
		}
	}
	t.Fatalf("context field %q not found", key)
	return ""
}
