package mql2go

import (
	"context"
	"testing"
)

// vm_lasterror_test.go — QS-1.2a: real lastError builtins.
// MQL4 semantics: _LastError persists across events; GetLastError reads then
// clears; SetUserError writes ERR_USER_ERROR_FIRST + code.

// S4a: SetUserError(5) → GetLastError() = 65541 (ERR_USER_ERROR_FIRST+5).
func TestQS12a_SetUserErrorThenGetLastError(t *testing.T) {
	vmRunner, err := CompileMQL(`int r;
void OnTick() {
    SetUserError(5);
    r = GetLastError();
}`)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	if err := vmRunner.vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick failed: %v", err)
	}
	if got := getGlobalInt(t, vmRunner, "r"); got != 65541 {
		t.Errorf("r = %d, want 65541 (ERR_USER_ERROR_FIRST+5)", got)
	}
}

// S4b: GetLastError reads-then-clears — second call returns 0.
func TestQS12a_GetLastErrorReadsThenClears(t *testing.T) {
	vmRunner, err := CompileMQL(`int a, b;
void OnTick() {
    SetUserError(5);
    a = GetLastError();
    b = GetLastError();
}`)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	if err := vmRunner.vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick failed: %v", err)
	}
	if got := getGlobalInt(t, vmRunner, "a"); got != 65541 {
		t.Errorf("a = %d, want 65541", got)
	}
	if got := getGlobalInt(t, vmRunner, "b"); got != 0 {
		t.Errorf("b = %d, want 0 (GetLastError must clear after read)", got)
	}
}

// S4c: ResetLastError clears — GetLastError after reset returns 0.
func TestQS12a_ResetLastErrorClears(t *testing.T) {
	vmRunner, err := CompileMQL(`int r;
void OnTick() {
    SetUserError(5);
    ResetLastError();
    r = GetLastError();
}`)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	if err := vmRunner.vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick failed: %v", err)
	}
	if got := getGlobalInt(t, vmRunner, "r"); got != 0 {
		t.Errorf("r = %d, want 0 (ResetLastError must clear)", got)
	}
}

// S4d: lastError persists across events — set in event #1, read in event #2.
func TestQS12a_LastErrorPersistsAcrossEvents(t *testing.T) {
	vmRunner, err := CompileMQL(`int n, r;
void OnTick() {
    if (n == 0) {
        SetUserError(7);
    } else {
        r = GetLastError();
    }
    n++;
}`)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	ctx := context.Background()
	if err := vmRunner.vm.RunOnTick(ctx); err != nil {
		t.Fatalf("RunOnTick #1 failed: %v", err)
	}
	if err := vmRunner.vm.RunOnTick(ctx); err != nil {
		t.Fatalf("RunOnTick #2 failed: %v", err)
	}
	if got := getGlobalInt(t, vmRunner, "r"); got != 65543 {
		t.Errorf("r = %d, want 65543 (lastError must persist across events)", got)
	}
}

// S4e: default value — GetLastError with nothing set returns 0.
func TestQS12a_GetLastErrorDefaultZero(t *testing.T) {
	vmRunner, err := CompileMQL(`int r;
void OnTick() {
    r = GetLastError();
}`)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	if err := vmRunner.vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick failed: %v", err)
	}
	if got := getGlobalInt(t, vmRunner, "r"); got != 0 {
		t.Errorf("r = %d, want 0 (default lastError)", got)
	}
}

// S4f: ERR_USER_ERROR_FIRST constant is usable in MQL source.
func TestQS12a_ErrUserErrorFirstConstant(t *testing.T) {
	vmRunner, err := CompileMQL(`int r;
void OnTick() {
    r = ERR_USER_ERROR_FIRST;
}`)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	if err := vmRunner.vm.RunOnTick(context.Background()); err != nil {
		t.Fatalf("RunOnTick failed: %v", err)
	}
	if got := getGlobalInt(t, vmRunner, "r"); got != 65536 {
		t.Errorf("r = %d, want 65536 (ERR_USER_ERROR_FIRST)", got)
	}
}
