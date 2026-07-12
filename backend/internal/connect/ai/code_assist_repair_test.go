package ai

import (
	"strings"
	"testing"
)

func TestExtractCodeFromRepair_FencedCode(t *testing.T) {
	raw := "Here is the fixed code:\n\n```go\npackage main\n\nfunc (s *MyStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {\n    return nil, nil\n}\n```\n\nThis fixes the error."
	code := extractCodeFromRepair(raw, false)
	if code == "" {
		t.Fatal("expected code extraction")
	}
	if code != "package main\n\nfunc (s *MyStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {\n    return nil, nil\n}" {
		t.Errorf("unexpected code: %q", code)
	}
}

func TestExtractCodeFromRepair_Heuristic(t *testing.T) {
	raw := "package main\n\nimport (\n    \"alphaforge/strategy/sdk\"\n)\n\nfunc (s *MyStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {\n    return nil, nil\n}\n\n以上是修正后的代码"
	code := extractCodeFromRepair(raw, false)
	if code == "" {
		t.Fatal("expected heuristic extraction")
	}
}

func TestExtractCodeFromRepair_NoCode(t *testing.T) {
	raw := "这个策略的止损设置合理，但是建议增加止盈条件。可以考虑在盈利达到2%时自动止盈。"
	code := extractCodeFromRepair(raw, false)
	if code != "" {
		t.Errorf("expected empty, got: %q", code)
	}
}

func TestExtractCodeFromRepair_GenericFence(t *testing.T) {
	raw := "Fixed version:\n```\npackage main\n\nfunc (s *MyStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {\n    return nil, nil\n}\n```"
	code := extractCodeFromRepair(raw, false)
	if code == "" {
		t.Fatal("expected extraction from generic fence")
	}
}

func TestExtractCodeFromRepair_MQLFenced(t *testing.T) {
	raw := "Here is the fixed MQL4 code:\n\n```mql4\n#property strict\n\nextern int Period = 14;\n\nint OnInit() {\n    return(INIT_SUCCEEDED);\n}\n\nvoid OnTick() {\n    double ma = iMA(Symbol(), PERIOD_CURRENT, Period, 0, MODE_EMA, PRICE_CLOSE, 0);\n}\n```\n\nThis fixes the compile error."
	code := extractCodeFromRepair(raw, true)
	if code == "" {
		t.Fatal("expected MQL code extraction")
	}
	if !strings.Contains(code, "#property strict") {
		t.Errorf("expected #property strict in extracted code, got: %q", code)
	}
}

func TestExtractCodeFromRepair_MQLHeuristic(t *testing.T) {
	raw := "#property strict\n\nextern int Period = 14;\n\nint OnInit() {\n    return(INIT_SUCCEEDED);\n}\n\n以上是修正后的代码"
	code := extractCodeFromRepair(raw, true)
	if code == "" {
		t.Fatal("expected MQL heuristic extraction")
	}
	if !strings.Contains(code, "#property strict") {
		t.Errorf("expected #property strict in extracted code, got: %q", code)
	}
}
