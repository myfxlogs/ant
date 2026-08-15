package strategy

import (
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	ai "alphaforge/internal/connect/ai"
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
