package strategy

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
)

// ── Worker fallback derivation tests (buildBacktestConfig) ──────────

func TestBuildBacktestConfig_ExplicitSignalTimingWins(t *testing.T) {
	srv := &StrategyExecutionServer{}
	run := &repository.BacktestRun{Symbol: "EURUSD", Timeframe: "H1"}
	params := backtestParams{
		initialCapital: "10000",
		leverage:       "100",
		signalTiming:   "same_bar_close",
		strictMode:     true, // should be ignored — explicit signalTiming wins
	}
	cfg := srv.buildBacktestConfig(params, run)
	if cfg.SignalTiming != "same_bar_close" {
		t.Fatalf("explicit signalTiming should win: got %q, want same_bar_close", cfg.SignalTiming)
	}
	if cfg.StrictMode {
		t.Fatal("StrictMode should be false when signalTiming=same_bar_close")
	}
}

func TestBuildBacktestConfig_FallbackFromStrictMode_True(t *testing.T) {
	srv := &StrategyExecutionServer{}
	run := &repository.BacktestRun{Symbol: "EURUSD", Timeframe: "H1"}
	params := backtestParams{
		initialCapital: "10000",
		leverage:       "100",
		signalTiming:   "", // empty → derive from strictMode
		strictMode:     true,
	}
	cfg := srv.buildBacktestConfig(params, run)
	if cfg.SignalTiming != "next_bar_open" {
		t.Fatalf("strictMode=true should derive next_bar_open: got %q", cfg.SignalTiming)
	}
	if !cfg.StrictMode {
		t.Fatal("StrictMode should be true when signalTiming=next_bar_open")
	}
}

func TestBuildBacktestConfig_FallbackFromStrictMode_False(t *testing.T) {
	srv := &StrategyExecutionServer{}
	run := &repository.BacktestRun{Symbol: "EURUSD", Timeframe: "H1"}
	params := backtestParams{
		initialCapital: "10000",
		leverage:       "100",
		signalTiming:   "", // empty → derive from strictMode
		strictMode:     false,
	}
	cfg := srv.buildBacktestConfig(params, run)
	if cfg.SignalTiming != "same_bar_close" {
		t.Fatalf("strictMode=false should derive same_bar_close: got %q", cfg.SignalTiming)
	}
	if cfg.StrictMode {
		t.Fatal("StrictMode should be false when signalTiming=same_bar_close")
	}
}

func TestBuildBacktestConfig_FallbackNeitherSet_DefaultsNextBarOpen(t *testing.T) {
	srv := &StrategyExecutionServer{}
	run := &repository.BacktestRun{Symbol: "EURUSD", Timeframe: "H1"}
	params := backtestParams{
		initialCapital: "10000",
		leverage:       "100",
		signalTiming:   "",    // empty
		strictMode:     false, // false → same_bar_close, not next_bar_open
	}
	cfg := srv.buildBacktestConfig(params, run)
	// strictMode=false → same_bar_close (not the "neither" case)
	if cfg.SignalTiming != "same_bar_close" {
		t.Fatalf("strictMode=false should derive same_bar_close: got %q", cfg.SignalTiming)
	}
}

func TestBuildBacktestConfig_DefaultFillRule(t *testing.T) {
	srv := &StrategyExecutionServer{}
	run := &repository.BacktestRun{Symbol: "EURUSD", Timeframe: "H1"}
	params := backtestParams{
		initialCapital: "10000",
		leverage:       "100",
		fillRule:       "", // empty → default
	}
	cfg := srv.buildBacktestConfig(params, run)
	if cfg.FillRule != "bar_close" {
		t.Fatalf("empty fillRule should default to bar_close: got %q", cfg.FillRule)
	}
}

func TestBuildBacktestConfig_DefaultSimulationMode(t *testing.T) {
	srv := &StrategyExecutionServer{}
	run := &repository.BacktestRun{Symbol: "EURUSD", Timeframe: "H1"}
	params := backtestParams{
		initialCapital: "10000",
		leverage:       "100",
		simulationMode: "", // empty → default
	}
	cfg := srv.buildBacktestConfig(params, run)
	if cfg.SimulationMode != "KLINE_RANGE" {
		t.Fatalf("empty simulationMode should default to KLINE_RANGE: got %q", cfg.SimulationMode)
	}
}

// ── Validate rejection tests (validateBacktestRequest) ──────────────

func TestValidateBacktestRequest_AcceptFillRuleLimit(t *testing.T) {
	srv := &StrategyExecutionServer{marketDataRepo: nil}
	req := connect.NewRequest(&antv1.StartBacktestRunRequest{
		Code:      "//test",
		Symbol:    "EURUSD",
		Timeframe: "H1",
		AccountId: uuid.New().String(),
		ExecutionConfig: &antv1.BacktestExecutionConfig{
			FillRule:       "limit",
			SimulationMode: "KLINE_RANGE",
			SignalTiming:   "next_bar_open",
		},
	})
	err := srv.validateBacktestRequest(context.Background(), req)
	// fill_rule=limit is now a valid whitelist value — should NOT be rejected.
	if err != nil {
		ce, ok := err.(*connect.Error)
		if ok && ce.Code() == connect.CodeInvalidArgument {
			t.Fatalf("fill_rule=limit should be accepted (whitelist): got %v", err)
		}
	}
}

func TestValidateBacktestRequest_RejectSimulationModeDataset(t *testing.T) {
	srv := &StrategyExecutionServer{marketDataRepo: nil}
	req := connect.NewRequest(&antv1.StartBacktestRunRequest{
		Code:      "//test",
		Symbol:    "EURUSD",
		Timeframe: "H1",
		AccountId: uuid.New().String(),
		ExecutionConfig: &antv1.BacktestExecutionConfig{
			FillRule:       "bar_close",
			SimulationMode: "DATASET",
			SignalTiming:   "next_bar_open",
		},
	})
	err := srv.validateBacktestRequest(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for simulation_mode=DATASET, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}
	if ce.Code() != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", ce.Code())
	}
	// Error message should mention OHLC_PATH as the replacement.
	if !strings.Contains(err.Error(), "OHLC_PATH") {
		t.Errorf("error should mention OHLC_PATH as replacement: got %v", err)
	}
}

func TestValidateBacktestRequest_AcceptValidExecConfig(t *testing.T) {
	srv := &StrategyExecutionServer{marketDataRepo: nil}
	req := connect.NewRequest(&antv1.StartBacktestRunRequest{
		Code:      "//test",
		Symbol:    "EURUSD",
		Timeframe: "H1",
		AccountId: uuid.New().String(),
		ExecutionConfig: &antv1.BacktestExecutionConfig{
			FillRule:       "bar_close",
			SimulationMode: "KLINE_RANGE",
			SignalTiming:   "next_bar_open",
		},
	})
	err := srv.validateBacktestRequest(context.Background(), req)
	// With marketDataRepo=nil, validate returns nil after exec config check.
	if err != nil {
		t.Fatalf("valid exec config should not be rejected: got %v", err)
	}
}

func TestValidateBacktestRequest_RejectUnknownFillRule(t *testing.T) {
	srv := &StrategyExecutionServer{marketDataRepo: nil}
	req := connect.NewRequest(&antv1.StartBacktestRunRequest{
		Code:      "//test",
		Symbol:    "EURUSD",
		Timeframe: "H1",
		AccountId: uuid.New().String(),
		ExecutionConfig: &antv1.BacktestExecutionConfig{
			FillRule: "FOO",
		},
	})
	err := srv.validateBacktestRequest(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for unknown fill_rule=FOO, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", err)
	}
}

func TestValidateBacktestRequest_RejectUnknownSimulationMode(t *testing.T) {
	srv := &StrategyExecutionServer{marketDataRepo: nil}
	req := connect.NewRequest(&antv1.StartBacktestRunRequest{
		Code:      "//test",
		Symbol:    "EURUSD",
		Timeframe: "H1",
		AccountId: uuid.New().String(),
		ExecutionConfig: &antv1.BacktestExecutionConfig{
			SimulationMode: "FOO",
		},
	})
	err := srv.validateBacktestRequest(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for unknown simulation_mode=FOO, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", err)
	}
}

func TestValidateBacktestRequest_AcceptOHLCPath(t *testing.T) {
	srv := &StrategyExecutionServer{marketDataRepo: nil}
	req := connect.NewRequest(&antv1.StartBacktestRunRequest{
		Code:      "//test",
		Symbol:    "EURUSD",
		Timeframe: "H1",
		AccountId: uuid.New().String(),
		ExecutionConfig: &antv1.BacktestExecutionConfig{
			FillRule:       "bar_close",
			SimulationMode: "OHLC_PATH",
			SignalTiming:   "next_bar_open",
		},
	})
	err := srv.validateBacktestRequest(context.Background(), req)
	if err != nil {
		ce, ok := err.(*connect.Error)
		if ok && ce.Code() == connect.CodeInvalidArgument {
			t.Fatalf("simulation_mode=OHLC_PATH should be accepted: got %v", err)
		}
	}
}

func TestValidateBacktestRequest_NilExecutionConfigNoPanic(t *testing.T) {
	srv := &StrategyExecutionServer{marketDataRepo: nil}
	req := connect.NewRequest(&antv1.StartBacktestRunRequest{
		Code:      "//test",
		Symbol:    "EURUSD",
		Timeframe: "H1",
		AccountId: uuid.New().String(),
		// ExecutionConfig is nil — must not panic
	})
	err := srv.validateBacktestRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("nil executionConfig should not cause error when marketDataRepo=nil: got %v", err)
	}
}

func TestValidateBacktestRequest_EmptySymbolRejected(t *testing.T) {
	srv := &StrategyExecutionServer{marketDataRepo: nil}
	req := connect.NewRequest(&antv1.StartBacktestRunRequest{
		Code:      "//test",
		Symbol:    "", // empty
		Timeframe: "H1",
		AccountId: uuid.New().String(),
	})
	err := srv.validateBacktestRequest(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty symbol, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}
	if ce.Code() != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", ce.Code())
	}
}

// ── ExtractBacktestParams round-trip test (ConfigSnapshot → params) ──

func TestExtractBacktestParams_ConfigSnapshotRoundTrip(t *testing.T) {
	ec := &antv1.BacktestExecutionConfig{
		SignalTiming:   "same_bar_close",
		FillRule:       "market",
		SimulationMode: "KLINE_RANGE",
		StrictMode:     false,
	}
	snapshot, err := proto.Marshal(ec)
	if err != nil {
		t.Fatalf("proto.Marshal failed: %v", err)
	}

	code := "//test"
	run := &repository.BacktestRun{
		StrategyCode:   &code,
		ConfigSnapshot: snapshot,
	}

	params, err := extractBacktestParams(run)
	if err != nil {
		t.Fatalf("extractBacktestParams failed: %v", err)
	}
	if params.signalTiming != "same_bar_close" {
		t.Fatalf("signalTiming: got %q, want same_bar_close", params.signalTiming)
	}
	if params.fillRule != "market" {
		t.Fatalf("fillRule: got %q, want market", params.fillRule)
	}
	if params.simulationMode != "KLINE_RANGE" {
		t.Fatalf("simulationMode: got %q, want KLINE_RANGE", params.simulationMode)
	}
}

func TestExtractBacktestParams_OldSnapshotNoExecFields(t *testing.T) {
	// Old snapshot with only legacy fields (no signal_timing/fill_rule/simulation_mode).
	// DiscardUnknown=true should handle this gracefully.
	ec := &antv1.BacktestExecutionConfig{
		StrictMode: true,
		// SignalTiming/FillRule/SimulationMode are zero values (empty strings)
	}
	snapshot, err := proto.Marshal(ec)
	if err != nil {
		t.Fatalf("proto.Marshal failed: %v", err)
	}

	code := "//test"
	run := &repository.BacktestRun{
		StrategyCode:   &code,
		StrictMode:     &[]bool{true}[0],
		ConfigSnapshot: snapshot,
	}

	params, err := extractBacktestParams(run)
	if err != nil {
		t.Fatalf("extractBacktestParams failed: %v", err)
	}
	// signalTiming should be empty → buildBacktestConfig will derive from strictMode
	if params.signalTiming != "" {
		t.Fatalf("old snapshot should have empty signalTiming: got %q", params.signalTiming)
	}
	if !params.strictMode {
		t.Fatal("strictMode should be true from run.StrictMode")
	}
}

func TestExtractBacktestParams_EmptyConfigSnapshot(t *testing.T) {
	code := "//test"
	run := &repository.BacktestRun{
		StrategyCode: &code,
		// ConfigSnapshot is nil/empty
	}

	params, err := extractBacktestParams(run)
	if err != nil {
		t.Fatalf("extractBacktestParams failed: %v", err)
	}
	// Defaults: strictMode=true (from extractBacktestParams default)
	if !params.strictMode {
		t.Fatal("default strictMode should be true")
	}
	if params.signalTiming != "" {
		t.Fatalf("empty config should have empty signalTiming: got %q", params.signalTiming)
	}
}

// ── End-to-end: extract → buildConfig fallback chain ────────────────

func TestExtractAndBuildConfig_OldSnapshotStrictFalse(t *testing.T) {
	// Simulates an old run with strict_mode=false and no signal_timing in snapshot.
	// The fallback should derive same_bar_close, NOT next_bar_open.
	ec := &antv1.BacktestExecutionConfig{
		StrictMode: false,
	}
	snapshot, err := proto.Marshal(ec)
	if err != nil {
		t.Fatalf("proto.Marshal failed: %v", err)
	}

	code := "//test"
	run := &repository.BacktestRun{
		StrategyCode:   &code,
		StrictMode:     &[]bool{false}[0],
		ConfigSnapshot: snapshot,
	}

	params, err := extractBacktestParams(run)
	if err != nil {
		t.Fatalf("extractBacktestParams failed: %v", err)
	}

	srv := &StrategyExecutionServer{}
	cfg := srv.buildBacktestConfig(params, run)
	if cfg.SignalTiming != "same_bar_close" {
		t.Fatalf("old strict=false run should derive same_bar_close: got %q", cfg.SignalTiming)
	}
	if cfg.StrictMode {
		t.Fatal("StrictMode should be false for same_bar_close")
	}
}

func TestExtractAndBuildConfig_OldSnapshotStrictTrue(t *testing.T) {
	// Simulates an old run with strict_mode=true and no signal_timing in snapshot.
	// The fallback should derive next_bar_open.
	ec := &antv1.BacktestExecutionConfig{
		StrictMode: true,
	}
	snapshot, err := proto.Marshal(ec)
	if err != nil {
		t.Fatalf("proto.Marshal failed: %v", err)
	}

	code := "//test"
	run := &repository.BacktestRun{
		StrategyCode:   &code,
		StrictMode:     &[]bool{true}[0],
		ConfigSnapshot: snapshot,
	}

	params, err := extractBacktestParams(run)
	if err != nil {
		t.Fatalf("extractBacktestParams failed: %v", err)
	}

	srv := &StrategyExecutionServer{}
	cfg := srv.buildBacktestConfig(params, run)
	if cfg.SignalTiming != "next_bar_open" {
		t.Fatalf("old strict=true run should derive next_bar_open: got %q", cfg.SignalTiming)
	}
	if !cfg.StrictMode {
		t.Fatal("StrictMode should be true for next_bar_open")
	}
}

// ── Adversarial: case sensitivity ───────────────────────────────────

func TestValidateBacktestRequest_CaseSensitiveFillRule(t *testing.T) {
	srv := &StrategyExecutionServer{marketDataRepo: nil}
	// "LIMIT" (uppercase) is NOT in the whitelist — should be rejected.
	req := connect.NewRequest(&antv1.StartBacktestRunRequest{
		Code:      "//test",
		Symbol:    "EURUSD",
		Timeframe: "H1",
		AccountId: uuid.New().String(),
		ExecutionConfig: &antv1.BacktestExecutionConfig{
			FillRule: "LIMIT", // uppercase — not in whitelist
		},
	})
	err := srv.validateBacktestRequest(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for uppercase LIMIT, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}
	if ce.Code() != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", ce.Code())
	}
}

// ── Adversarial: initial_capital validation ─────────────────────────

func TestValidateBacktestRequest_InvalidInitialCapital(t *testing.T) {
	srv := &StrategyExecutionServer{marketDataRepo: nil}
	req := connect.NewRequest(&antv1.StartBacktestRunRequest{
		Code:           "//test",
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		AccountId:      uuid.New().String(),
		InitialCapital: "not-a-number",
	})
	err := srv.validateBacktestRequest(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid initial_capital, got nil")
	}
	ce, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}
	if ce.Code() != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", ce.Code())
	}
}

// suppress unused import (decimal is used by other tests in the package)
var _ = decimal.NewFromInt
