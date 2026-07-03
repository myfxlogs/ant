package agent

import (
	"testing"

	"anttrader/tools/mql2go"
)

// TestBridge_TranslateBlindSpot verifies the bridge LLM prompt construction and response parsing.
// This is a unit test for the bridge logic — it does not call the actual LLM.
func TestBridge_TranslateBlindSpot(t *testing.T) {
	// Test stripMarkdownFences
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no fences", "x = 1", "x = 1"},
		{"python fences", "```python\nx = 1\n```", "x = 1"},
		{"plain fences", "```\nx = 1\n```", "x = 1"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMarkdownFences(tt.input)
			if got != tt.want {
				t.Errorf("stripMarkdownFences(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestBridge_ParseBridgeResponse verifies that LLM response parsing works correctly.
func TestBridge_ParseBridgeResponse(t *testing.T) {
	tests := []struct {
		name   string
		resp   string
		wantPy string
	}{
		{
			name:   "plain python code",
			resp:   "from decimal import Decimal\nclass S:\n    def on_bar(self) -> None:\n        x = 0",
			wantPy: "from decimal import Decimal\nclass S:\n    def on_bar(self) -> None:\n        x = 0",
		},
		{
			name:   "markdown fenced",
			resp:   "```python\nfrom decimal import Decimal\nclass S:\n    def on_bar(self) -> None:\n        x = 0\n```",
			wantPy: "from decimal import Decimal\nclass S:\n    def on_bar(self) -> None:\n        x = 0",
		},
		{
			name:   "empty response",
			resp:   "",
			wantPy: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBridgeResponse(tt.resp)
			if result.PythonSource != tt.wantPy {
				t.Errorf("PythonSource = %q, want %q", result.PythonSource, tt.wantPy)
			}
		})
	}
}

// TestBuildBridgeChanges verifies semantic diff generation from coverage improvement.
func TestBuildBridgeChanges(t *testing.T) {
	orig := &mql2go.CoverageResult{
		Score: 0.6,
		BlindSpots: []mql2go.CoverageBlindSpot{
			{Builtin: "iCustom", Severity: "fatal", Count: 2},
			{Builtin: "ObjectCreate", Severity: "warning", Count: 5},
		},
	}
	bridged := &mql2go.CoverageResult{
		Score: 0.9,
		BlindSpots: []mql2go.CoverageBlindSpot{
			{Builtin: "ObjectCreate", Severity: "warning", Count: 5},
		},
	}

	changes := buildBridgeChanges(orig, bridged)
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}

	// iCustom should be "removed" (resolved)
	foundRemoved := false
	foundRemaining := false
	for _, c := range changes {
		if c.Kind == "removed" && c.Description != "" {
			foundRemoved = true
		}
		if c.Kind == "remaining" && c.Description != "" {
			foundRemaining = true
		}
	}
	if !foundRemoved {
		t.Error("expected a 'removed' change for resolved blind spot")
	}
	if !foundRemaining {
		t.Error("expected a 'remaining' change for remaining blind spot")
	}
}

// TestTruncate verifies the truncate helper.
func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("truncate('hello', 10) = %q, want 'hello'", got)
	}
	if got := truncate("hello world", 5); got != "hello..." {
		t.Errorf("truncate('hello world', 5) = %q, want 'hello...'", got)
	}
}
