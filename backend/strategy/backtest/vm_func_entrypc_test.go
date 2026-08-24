package backtest

import (
	"context"
	"testing"

	mql2go "alphaforge/tools/mql2go"
)

// TestVM_FuncEntryPC_UserFuncCallAfterIntReturn verifies that a void user
// function called inside an if-body after an int-returning user function
// actually executes. This was a critical bug where executeCallUser jumped
// to entryPC+1, but EntryPC pointed at the OP_ENTER_FUNC marker — and
// because Pass 1 emits ALL markers contiguously before Pass 2 compiles any
// body, entryPC+1 landed on the NEXT function's marker, not this function's
// body. The called function silently didn't execute.
//
// Regression: Moving Average EA (backtest 95fbd896) — CalculateCurrentOrders
// returned 0, if(res==0) CheckForOpen() was reached, but CheckForOpen's body
// never executed because OP_CALL_USER jumped to the wrong PC.
//
// Adversarial proof: revert executeCallUser to `vm.pc = entryPC + 1` →
// g_checkForOpenEntered stays 0 → RED.
func TestVM_FuncEntryPC_UserFuncCallAfterIntReturn(t *testing.T) {
	const ea = `#property strict
int g_checkForOpenEntered = 0;
int g_calcEntered = 0;
int g_calcResult = -999;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    if(Volume[0]>1) return;
    if(Bars<10) return;
    int res = CalculateCurrentOrders();
    g_calcResult = res;
    if(res==0) CheckForOpen();
}

int CalculateCurrentOrders() {
    g_calcEntered++;
    return(0);
}

void CheckForOpen() {
    g_checkForOpenEntered++;
}`
	bars := d3GenerateBars(20)
	runner, err := mql2go.CompileMQL(ea)
	if err != nil {
		t.Fatalf("CompileMQL: %v", err)
	}

	cfg := d3MakeConfig()
	engine := New(cfg, runner, bars)
	_, err = engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	calcEntered, ok := runner.GetGlobal("g_calcEntered")
	if !ok || calcEntered.ToInt() == 0 {
		t.Fatal("CalculateCurrentOrders never executed")
	}
	checkForOpenEntered, ok := runner.GetGlobal("g_checkForOpenEntered")
	if !ok {
		t.Fatal("g_checkForOpenEntered global not found")
	}
	if checkForOpenEntered.ToInt() == 0 {
		t.Fatal("CheckForOpen never executed — OP_CALL_USER jumped to wrong PC " +
			"(entryPC+1 landed on next function's marker instead of body)")
	}
	t.Logf("CalculateCurrentOrders called %d times, CheckForOpen called %d times",
		calcEntered.ToInt(), checkForOpenEntered.ToInt())
}
