package strategy

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestInterpHarness_ParsesAsValidGo(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"interp backtest harness", generateInterpBacktestHarness()},
		{"interp live harness", generateInterpLiveHarness()},
		{"compiled backtest harness", generateBacktestHarness("MyStrategy")},
		{"compiled live harness", generateLiveHarness("MyStrategy")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			_, err := parser.ParseFile(fset, "harness.go", tt.code, 0)
			if err != nil {
				t.Fatalf("parse harness: %v", err)
			}
		})
	}
}

func TestInterpHarness_UsesWASMRunSetup(t *testing.T) {
	bt := generateInterpBacktestHarness()
	if !strings.Contains(bt, "interp.WASMRunSetup()") {
		t.Error("interp backtest harness should call interp.WASMRunSetup()")
	}
	live := generateInterpLiveHarness()
	if !strings.Contains(live, "interp.WASMRunSetup()") {
		t.Error("interp live harness should call interp.WASMRunSetup()")
	}
}

func TestCompiledHarness_InstantiatesType(t *testing.T) {
	bt := generateBacktestHarness("MyStrategy")
	if !strings.Contains(bt, "&MyStrategy{}") {
		t.Error("compiled backtest harness should instantiate &MyStrategy{}")
	}
	live := generateLiveHarness("MyStrategy")
	if !strings.Contains(live, "&MyStrategy{}") {
		t.Error("compiled live harness should instantiate &MyStrategy{}")
	}
}

func TestIRLengthPrefix(t *testing.T) {
	ir := []byte{0x01, 0x02, 0x03}
	prefixed := irLengthPrefix(ir)
	if len(prefixed) != 4+3 {
		t.Fatalf("expected 7 bytes, got %d", len(prefixed))
	}
	if prefixed[0] != 3 || prefixed[1] != 0 || prefixed[2] != 0 || prefixed[3] != 0 {
		t.Fatalf("expected LE u32 = 3, got %v", prefixed[:4])
	}
	if prefixed[4] != 0x01 || prefixed[5] != 0x02 || prefixed[6] != 0x03 {
		t.Fatalf("IR data mismatch: %v", prefixed[4:])
	}
}

func TestIsMQLStrategy(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"", false},
		{"package main\nimport \"anttrader/strategy/sdk\"\nfunc OnInit() {}", false},
		{"int OnInit() { return 0; }\nvoid OnBar() {}", true},
		{"void OnTick() { }", true},
		{"void OnTimer() { }", true},
		{"void OnDeinit() { }", true},
		{"hello world", false},
	}
	for _, tt := range tests {
		got := isMQLStrategy(tt.code)
		if got != tt.want {
			t.Errorf("isMQLStrategy(%q) = %v, want %v", tt.code, got, tt.want)
		}
	}
}
