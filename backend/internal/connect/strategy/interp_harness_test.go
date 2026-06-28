package strategy

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestInterpHarness_ParsesAsValidGo(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"backtest harness", generateInterpBacktestHarness()},
		{"live harness", generateInterpLiveHarness()},
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

func TestIRLengthPrefix(t *testing.T) {
	ir := []byte{0x01, 0x02, 0x03}
	prefixed := irLengthPrefix(ir)
	if len(prefixed) != 4+3 {
		t.Fatalf("expected 7 bytes, got %d", len(prefixed))
	}
	// u32 LE length = 3
	if prefixed[0] != 3 || prefixed[1] != 0 || prefixed[2] != 0 || prefixed[3] != 0 {
		t.Fatalf("expected LE u32 = 3, got %v", prefixed[:4])
	}
	// IR data follows
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
		{"package main\nimport \"anttrader/strategy/sdk\"\nfunc OnInit() {}", false}, // Go strategy
		{"int OnInit() { return 0; }\nvoid OnBar() {}", true},                       // MQL4
		{"void OnTick() { }", true},                                                   // MQL
		{"void OnTimer() { }", true},                                                  // MQL
		{"hello world", false},                                                        // not MQL
	}
	for _, tt := range tests {
		got := isMQLStrategy(tt.code)
		if got != tt.want {
			t.Errorf("isMQLStrategy(%q) = %v, want %v", tt.code, got, tt.want)
		}
	}
}
