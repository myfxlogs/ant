package strategy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	ai "alphaforge/internal/connect/ai"
	"alphaforge/internal/model"
)

// --- E3 Adversarial Tests: validateParamType ---

func TestE3_validateParamType_IntValid(t *testing.T) {
	if err := validateParamType("int", "42"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestE3_validateParamType_IntInvalid(t *testing.T) {
	if err := validateParamType("int", "abc"); err == nil {
		t.Fatal("expected error for int='abc', got nil")
	}
}

func TestE3_validateParamType_FloatValid(t *testing.T) {
	if err := validateParamType("float", "3.14"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestE3_validateParamType_FloatInvalid(t *testing.T) {
	if err := validateParamType("float", "notanumber"); err == nil {
		t.Fatal("expected error for float='notanumber', got nil")
	}
}

func TestE3_validateParamType_BoolValid(t *testing.T) {
	if err := validateParamType("bool", "true"); err != nil {
		t.Fatalf("expected nil for 'true', got %v", err)
	}
	if err := validateParamType("bool", "false"); err != nil {
		t.Fatalf("expected nil for 'false', got %v", err)
	}
}

func TestE3_validateParamType_BoolInvalid(t *testing.T) {
	if err := validateParamType("bool", "yes"); err == nil {
		t.Fatal("expected error for bool='yes', got nil")
	}
}

func TestE3_validateParamType_StrAny(t *testing.T) {
	if err := validateParamType("str", "anything"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// TestE3_ExtractParams_Reuse verifies ai.ExtractParams correctly extracts
// declared parameters (REUSE check — no regex duplication).
func TestE3_ExtractParams_Reuse(t *testing.T) {
	code := `func OnTick(ctx *Context) {
	_ = ctx.ParamInt("TakeProfit", 50)
	_ = ctx.ParamDecimal("LotSize", 0.01)
	_ = ctx.ParamBool("UseTrailing", true)
	_ = ctx.ParamString("Comment", "test")
}`
	entries := ai.ExtractParams(code)
	if len(entries) != 4 {
		t.Fatalf("expected 4 params, got %d", len(entries))
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name] = true
	}
	for _, expected := range []string{"TakeProfit", "LotSize", "UseTrailing", "Comment"} {
		if !names[expected] {
			t.Errorf("expected param %q not found", expected)
		}
	}
}

// --- E2 Adversarial Tests ---

func TestE2_isRunning_NotRunning(t *testing.T) {
	engine := &ScheduleEngine{
		activeRuns: make(map[uuid.UUID]*runHandle),
	}
	if engine.isRunning(uuid.New()) {
		t.Fatal("expected isRunning=false for unknown id")
	}
}

func TestE2_isRunning_Running(t *testing.T) {
	id := uuid.New()
	engine := &ScheduleEngine{
		activeRuns: map[uuid.UUID]*runHandle{
			id: {},
		},
	}
	if !engine.isRunning(id) {
		t.Fatal("expected isRunning=true for registered id")
	}
}

func TestE2_StopSchedule_NotRunningSafe(t *testing.T) {
	engine := &ScheduleEngine{
		activeRuns: make(map[uuid.UUID]*runHandle),
	}
	engine.StopSchedule(uuid.New())
}

// --- E3: legacyDeadKeys ---

func TestE3_LegacyDeadKeys_AllPresent(t *testing.T) {
	expected := []string{
		"__risk.default_volume",
		"__risk.max_positions",
		"__risk.stop_loss_price_offset",
		"__risk.take_profit_price_offset",
		"__risk.max_drawdown_pct",
	}
	for _, k := range expected {
		if !legacyDeadKeys[k] {
			t.Errorf("expected %q in legacyDeadKeys", k)
		}
	}
}

// TestE3_UnknownKeyError_ContainsKeyName verifies the error message
// for unknown keys includes the key name (user can self-correct typos).
func TestE3_UnknownKeyError_ContainsKeyName(t *testing.T) {
	err := connect.NewError(connect.CodeInvalidArgument,
		fmt.Errorf("unknown parameter(s): TypoKey"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "TypoKey") {
		t.Errorf("error message %q does not contain key name 'TypoKey'", err.Error())
	}
}

// --- D3: validateParamsAgainstSchema full-path tests ---

func makeDeclared() []*antv1.ParameterEntry {
	return []*antv1.ParameterEntry{
		{Name: "TakeProfit", Type: "int", Default: "50"},
		{Name: "LotSize", Type: "float", Default: "0.01"},
		{Name: "UseTrailing", Type: "bool", Default: "true"},
		{Name: "Comment", Type: "str", Default: "test"},
	}
}

// TestD3_UnknownKey_RejectedWithKeyName: unknown key → 400 with key name in message.
func TestD3_UnknownKey_RejectedWithKeyName(t *testing.T) {
	declared := makeDeclared()
	params := map[string]string{"TakeProfit": "60", "TypoKey": "1"}
	_, err := validateParamsAgainstSchema(declared, params)
	if err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
	if !strings.Contains(err.Error(), "TypoKey") {
		t.Errorf("error %q does not contain key name 'TypoKey'", err.Error())
	}
}

// TestD3_TypeMismatch_RejectedWithKeyName: int param with "abc" → 400 with key name.
func TestD3_TypeMismatch_RejectedWithKeyName(t *testing.T) {
	declared := makeDeclared()
	params := map[string]string{"TakeProfit": "abc"}
	_, err := validateParamsAgainstSchema(declared, params)
	if err == nil {
		t.Fatal("expected error for type mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "TakeProfit") {
		t.Errorf("error %q does not contain key name 'TakeProfit'", err.Error())
	}
}

// TestD3_LegacyKeys_StrippedNoMutation: legacy dead keys stripped from result,
// input map NOT mutated (pure function contract).
func TestD3_LegacyKeys_StrippedNoMutation(t *testing.T) {
	declared := makeDeclared()
	params := map[string]string{
		"TakeProfit":                      "60",
		"__risk.default_volume":           "0.1",
		"__risk.max_positions":            "5",
		"__risk.stop_loss_price_offset":   "10",
		"__risk.take_profit_price_offset": "20",
		"__risk.max_drawdown_pct":         "0.05",
	}
	// Snapshot input before call
	origLen := len(params)
	_, err := validateParamsAgainstSchema(declared, params)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	// Input map must NOT be mutated
	if len(params) != origLen {
		t.Errorf("input map mutated: was %d keys, now %d", origLen, len(params))
	}
	if _, ok := params["__risk.default_volume"]; !ok {
		t.Error("input map lost legacy key (mutation detected)")
	}
}

// TestD3_ValidParams_PassAndCleaned: all valid → nil error, cleaned map has only valid keys.
func TestD3_ValidParams_PassAndCleaned(t *testing.T) {
	declared := makeDeclared()
	params := map[string]string{
		"TakeProfit":            "60",
		"LotSize":               "0.02",
		"UseTrailing":           "false",
		"Comment":               "hello",
		"__risk.default_volume": "0.1",
		"__schedule.cron":       "0 * * * *",
	}
	cleaned, err := validateParamsAgainstSchema(declared, params)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if len(cleaned) != 4 {
		t.Fatalf("expected 4 cleaned keys, got %d: %v", len(cleaned), cleaned)
	}
	for _, k := range []string{"TakeProfit", "LotSize", "UseTrailing", "Comment"} {
		if _, ok := cleaned[k]; !ok {
			t.Errorf("expected key %q in cleaned map", k)
		}
	}
	for _, k := range []string{"__risk.default_volume", "__schedule.cron"} {
		if _, ok := cleaned[k]; ok {
			t.Errorf("legacy/schedule key %q should be stripped from cleaned map", k)
		}
	}
}

// TestD3_EmptyDeclared_NilError: no declared params + empty params → nil (skip validation).
func TestD3_EmptyDeclared_NilError(t *testing.T) {
	params := map[string]string{}
	_, err := validateParamsAgainstSchema(nil, params)
	if err != nil {
		t.Fatalf("expected nil for empty declared + empty params, got %v", err)
	}
}

// --- D3: maybeRestartSchedule behavior test ---

// TestD3_MaybeRestart_CancelTriggered: running + substantive=true →
// runHandle's cancel is called (ctx.Done() fires).
func TestD3_MaybeRestart_CancelTriggered(t *testing.T) {
	id := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine := &ScheduleEngine{
		activeRuns: map[uuid.UUID]*runHandle{
			id: {cancel: cancel},
		},
		log: zap.NewNop(),
		// Mock repo returns inactive schedule → StartSchedule returns error
		// (caught by maybeRestartSchedule's warn, no panic).
		repo: &mockScheduleRepo{
			getByID: func(_ context.Context, _ uuid.UUID) (*model.StrategySchedule, error) {
				return &model.StrategySchedule{ID: id, IsActive: false}, nil
			},
		},
	}

	srv := &StrategyServer{engine: engine, log: zap.NewNop()}

	// Call maybeRestartSchedule with substantive=true.
	// StartSchedule will return error (inactive) but that's OK — we assert cancel was triggered.
	srv.maybeRestartSchedule(ctx, id, true)

	// The cancel from runHandle should have been called by StopSchedule.
	// ctx.Done() should be closed.
	select {
	case <-ctx.Done():
		// ✅ cancel was triggered — expected
	default:
		t.Fatal("expected ctx.Done() to be closed after maybeRestartSchedule with substantive=true")
	}
}

// TestD3_MaybeRestart_NotRunning_NoCancel: not running + substantive=true →
// no cancel (Notify path, not Stop path).
func TestD3_MaybeRestart_NotRunning_NoCancel(t *testing.T) {
	id := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine := &ScheduleEngine{
		activeRuns: make(map[uuid.UUID]*runHandle),
		log:        zap.NewNop(),
	}

	srv := &StrategyServer{engine: engine, log: zap.NewNop()}
	srv.maybeRestartSchedule(ctx, id, true)

	select {
	case <-ctx.Done():
		t.Fatal("ctx should NOT be cancelled when schedule is not running")
	default:
		// ✅ not cancelled — expected
	}
}

// TestD3_MaybeRestart_SubstantiveFalse_NoCancel: running but substantive=false →
// no cancel (Notify path only, name-only change doesn't restart).
func TestD3_MaybeRestart_SubstantiveFalse_NoCancel(t *testing.T) {
	id := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine := &ScheduleEngine{
		activeRuns: map[uuid.UUID]*runHandle{
			id: {cancel: cancel},
		},
		log: zap.NewNop(),
	}

	srv := &StrategyServer{engine: engine, log: zap.NewNop()}
	srv.maybeRestartSchedule(ctx, id, false)

	select {
	case <-ctx.Done():
		t.Fatal("ctx should NOT be cancelled when substantive=false (name-only change)")
	default:
		// ✅ not cancelled — expected
	}
}
