package ai

import (
	"strings"
	"testing"
)

func TestParseTextToolCalls_Basic(t *testing.T) {
	t.Parallel()
	calls := parseTextToolCalls(`[TOOL: write_strategy code="def run(ctx): pass" name="test"]`)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "write_strategy" {
		t.Fatalf("expected write_strategy, got %s", calls[0].Name)
	}
	if !strings.Contains(calls[0].ArgsJSON, `"code":"def run(ctx): pass"`) {
		t.Fatalf("code arg truncated: %s", calls[0].ArgsJSON)
	}
	if !strings.Contains(calls[0].ArgsJSON, `"name":"test"`) {
		t.Fatalf("name arg missing: %s", calls[0].ArgsJSON)
	}
}

func TestParseTextToolCalls_MultiLineCode(t *testing.T) {
	t.Parallel()
	calls := parseTextToolCalls(`[TOOL: write_strategy code="def run(ctx):
    if ctx.bar.close > ctx.sma(20):
        ctx.buy(0.1)"]`)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if !strings.Contains(calls[0].ArgsJSON, "def run(ctx):") {
		t.Fatalf("multi-line code truncated: %s", calls[0].ArgsJSON)
	}
}

func TestParseTextToolCalls_MultipleCalls(t *testing.T) {
	t.Parallel()
	calls := parseTextToolCalls(`[TOOL: read_kline symbol="EURUSD" timeframe="H1"]
[TOOL: write_strategy code="strategy here"]`)
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0].Name != "read_kline" {
		t.Fatalf("expected read_kline, got %s", calls[0].Name)
	}
	if calls[1].Name != "write_strategy" {
		t.Fatalf("expected write_strategy, got %s", calls[1].Name)
	}
}

func TestParseTextToolCalls_PositionalFallback(t *testing.T) {
	t.Parallel()
	calls := parseTextToolCalls("[TOOL: read_kline EURUSD H1]")
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "read_kline" {
		t.Fatalf("expected read_kline, got %s", calls[0].Name)
	}
	if !strings.Contains(calls[0].ArgsJSON, `"symbol":"EURUSD"`) {
		t.Fatalf("positional symbol missing: %s", calls[0].ArgsJSON)
	}
}

func TestParseTextToolCalls_NoTool(t *testing.T) {
	t.Parallel()
	calls := parseTextToolCalls("just some text without tools")
	if len(calls) != 0 {
		t.Fatalf("expected 0 calls, got %d", len(calls))
	}
}

func TestParseTextToolCalls_NoArgs(t *testing.T) {
	t.Parallel()
	calls := parseTextToolCalls("[TOOL: confirm]")
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "confirm" {
		t.Fatalf("expected confirm, got %s", calls[0].Name)
	}
}

func TestParseTextToolCalls_EscapedQuote(t *testing.T) {
	t.Parallel()
	calls := parseTextToolCalls(`[TOOL: write_strategy code="def run(ctx):\n    ctx.log(\"hello\")"]`)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if !strings.Contains(calls[0].ArgsJSON, `ctx.log(\"hello\")`) {
		t.Fatalf("escaped quotes not preserved: %s", calls[0].ArgsJSON)
	}
}
