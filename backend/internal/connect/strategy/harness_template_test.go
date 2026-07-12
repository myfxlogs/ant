package strategy

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestCompiledHarness_ParsesAsValidGo(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
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

func TestIsMQLStrategy(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"", false},
		{"package main\nimport \"alphaforge/strategy/sdk\"\nfunc OnInit() {}", false},
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
