package mql2go

import (
	"context"
	"testing"
)

func TestVM_Audit_SingleStatementLoops(t *testing.T) {
	const source = `
int g_value = 0;
int OnInit() { return 0; }
void OnTick() {
    int i = 0;
    while (i < 2) i++;
    for (int j = 0; j < 2; j++) g_value++;
    do g_value++; while (g_value < 3);
}
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	if err := runner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit: %v", err)
	}
	if err := runner.vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick: %v", err)
	}
	value, ok := runner.GetGlobal("g_value")
	if !ok || value.ToInt() != 3 {
		t.Fatalf("single-statement loops value = %v, want 3", value)
	}
}

func TestVM_Audit_SwitchNoMatchDoesNotEnterFirstCase(t *testing.T) {
	const source = `
int g_value = 9;
int OnInit() { return 0; }
void OnTick() {
    switch (g_value) {
    case 1:
        g_value = 2;
    }
}
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	if err := runner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit: %v", err)
	}
	if err := runner.vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick: %v", err)
	}
	value, ok := runner.GetGlobal("g_value")
	if !ok || value.ToInt() != 9 {
		t.Fatalf("switch no-match value = %v, want 9", value)
	}
}

func TestVM_Audit_SwitchFallthrough(t *testing.T) {
	const source = `
int g_value = 1;
int OnInit() { return 0; }
void OnTick() {
    switch (g_value) {
    case 1:
        g_value = 2;
    case 2:
        g_value = 3;
        break;
    default:
        g_value = 4;
    }
}
`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	if err := runner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit: %v", err)
	}
	if err := runner.vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick: %v", err)
	}
	value, ok := runner.GetGlobal("g_value")
	if !ok || value.ToInt() != 3 {
		t.Fatalf("switch fallthrough value = %v, want 3", value)
	}
	// Stack must be empty after switch — break must consume the switch value,
	// not leave it on the stack (VM-COMPILER-SEMANTICS-3 / VM-TEST-EVIDENCE-3).
	if len(runner.vm.stack) != 0 {
		t.Fatalf("stack depth = %d after switch with break, want 0 (switch value not consumed)", len(runner.vm.stack))
	}
}

// TestVM_Audit_SwitchDefaultBeforeCase verifies that default can appear
// before regular cases in source order, and fallthrough from default
// continues to the next case in source order (MQL/C semantics).
func TestVM_Audit_SwitchDefaultBeforeCase(t *testing.T) {
	const source = `
int g_value = 0;
int OnInit() { return 0; }
void OnTick() {
    g_value = 0;
    switch (g_value) {
    default:
        g_value = 10;
        // fallthrough to case 1
    case 1:
        g_value = 20;
        break;
    case 2:
        g_value = 30;
        break;
    }
}`
	runner, err := CompileMQL(source)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}
	if err := runner.vm.RunOnInit(context.Background()); err != nil {
		t.Fatalf("RunOnInit: %v", err)
	}
	if err := runner.vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick: %v", err)
	}
	value, ok := runner.GetGlobal("g_value")
	if !ok || value.ToInt() != 20 {
		t.Fatalf("default-before-case fallthrough value = %v, want 20 (default falls through to case 1)", value)
	}
	if len(runner.vm.stack) != 0 {
		t.Fatalf("stack depth = %d after switch, want 0", len(runner.vm.stack))
	}
}
