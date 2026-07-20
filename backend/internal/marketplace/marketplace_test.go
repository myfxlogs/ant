package marketplace

import (
	"testing"
)

// ── parseJSONStringArray + splitJSONArray tests ──

func TestParseJSONStringArray(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty string", "", nil},
		{"null literal", "null", nil},
		{"empty brackets", "[]", nil},
		{"single element", `["EURUSD"]`, []string{"EURUSD"}},
		{"two elements", `["EURUSD","GBPUSD"]`, []string{"EURUSD", "GBPUSD"}},
		{"three elements", `["EURUSD","GBPUSD","USDJPY"]`, []string{"EURUSD", "GBPUSD", "USDJPY"}},
		{"no spaces", `["a","b","c"]`, []string{"a", "b", "c"}},
		{"empty element filtered", `["a","","b"]`, []string{"a", "b"}},
		{"element with comma in quotes", `["hello, world","bar"]`, []string{"hello, world", "bar"}},
		{"single element no brackets", `"EURUSD"`, []string{"EURUSD"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJSONStringArray(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("parseJSONStringArray(%q) = %v (len=%d), want %v (len=%d)", tt.raw, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSplitJSONArray(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", []string{}},
		{"single unquoted", "abc", []string{"abc"}},
		{"two unquoted", "a,b", []string{"a", "b"}},
		{"two quoted", `"a","b"`, []string{`"a"`, `"b"`}},
		{"quoted with comma", `"hello, world","bar"`, []string{`"hello, world"`, `"bar"`}},
		{"mixed", `"a",b,"c"`, []string{`"a"`, `b`, `"c"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitJSONArray(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("splitJSONArray(%q) = %v (len=%d), want %v (len=%d)", tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
