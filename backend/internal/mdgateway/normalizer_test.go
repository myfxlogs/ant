package mdgateway

import (
	"testing"
)

func TestCanonicalize_Basic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"EURUSD", "EURUSD"},
		{"eurusd", "EURUSD"},
		{"EURUSD.ecn", "EURUSD"},
		{"EURUSD.raw", "EURUSD"},
		{"EURUSD.pro", "EURUSD"},
		{"EURUSD.stp", "EURUSD"},
		{"EURUSD.m", "EURUSD"},
		{"EURUSD.i", "EURUSD"},
		{"EURUSD.x", "EURUSD"},
		{"EURUSDM", "EURUSD"},
		{"EURUSDI", "EURUSD"},
		{"EURUSD#", "EURUSD"},
		{"EURUSD!", "EURUSD"},
		{"EURUSDc", "EURUSDC"}, // .c NOT stripped
	}

	for _, tt := range tests {
		got := Canonicalize(tt.input)
		if got != tt.expected {
			t.Errorf("Canonicalize(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCanonicalize_ShortSymbols(t *testing.T) {
	// Very short symbols should not have suffixes stripped
	if Canonicalize("AU") != "AU" {
		t.Error("AU should remain AU")
	}
}

func TestCanonicalize_US30(t *testing.T) {
	// US30M is too short (5 chars) and has digit before M → not stripped
	if Canonicalize("US30M") != "US30M" {
		t.Error("US30M should remain US30M (too short + digit before suffix)")
	}
}

func TestMapResolver(t *testing.T) {
	r := NewMapResolver().(*mapResolver)
	c := r.Resolve("broker1", "EURUSD.ecn")
	if c != "EURUSD" {
		t.Fatalf("expected EURUSD, got %s", c)
	}
	// Second call should hit cache
	c2 := r.Resolve("broker1", "EURUSD.ecn")
	if c2 != "EURUSD" {
		t.Fatalf("expected EURUSD from cache, got %s", c2)
	}
}

func TestMapResolver_Load(t *testing.T) {
	r := NewMapResolver().(*mapResolver)
	r.Load("broker1", "XAUUSDm", "XAUUSD")
	c := r.Resolve("broker1", "XAUUSDm")
	if c != "XAUUSD" {
		t.Fatalf("expected XAUUSD, got %s", c)
	}
}

func TestNormalizer_Tick(t *testing.T) {
	n := NewNormalizer(nil)
	tick := n.Tick("u1", "broker1", "EURUSD", 1000, "1.1000", "1.1005")
	if tick.UserID != "u1" {
		t.Fatalf("expected u1, got %s", tick.UserID)
	}
	if tick.Canonical != "EURUSD" {
		t.Fatalf("expected EURUSD, got %s", tick.Canonical)
	}
	if tick.Bid.GetValue() != "1.1000" {
		t.Fatalf("expected 1.1000, got %s", tick.Bid.GetValue())
	}
}

func TestNormalizer_WithResolver(t *testing.T) {
	r := NewMapResolver()
	n := NewNormalizer(r)
	tick := n.Tick("u1", "broker1", "EURUSD.ecn", 1000, "1.1000", "1.1005")
	if tick.Canonical != "EURUSD" {
		t.Fatalf("expected EURUSD from resolver, got %s", tick.Canonical)
	}
}
