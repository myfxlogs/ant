// Package ai — E1 LookAhead Scanner tests.
package ai

import "testing"

func TestLookAheadScanner_CleanExpression(t *testing.T) {
	t.Parallel()
	s := NewLookAheadScanner()
	result := s.Scan("sma(close, 5) > ema(close, 20)")
	if !result.Passed {
		t.Fatalf("clean expression should pass, got violations: %+v", result.Violations)
	}
}

func TestLookAheadScanner_ExplicitFutureIndex(t *testing.T) {
	t.Parallel()
	s := NewLookAheadScanner()
	result := s.Scan("close[t+1] > open")
	if result.Passed {
		t.Fatal("close[t+1] should be detected as lookahead")
	}
	if len(result.Violations) == 0 {
		t.Fatal("should have violations")
	}
	t.Logf("Violation: %s", result.Violations[0].Message)
}

func TestLookAheadScanner_NegativeRefOffset(t *testing.T) {
	t.Parallel()
	s := NewLookAheadScanner()
	result := s.Scan("ref(close, -1) > open")
	if result.Passed {
		t.Fatal("ref(close, -1) should be detected as lookahead (negative offset = future peek)")
	}
}

func TestLookAheadScanner_FutureHighReference(t *testing.T) {
	t.Parallel()
	s := NewLookAheadScanner()
	result := s.Scan("high[t+1] > high")
	if result.Passed {
		t.Fatal("high[t+1] should be detected as lookahead")
	}
}

func TestLookAheadScanner_PositiveRefOK(t *testing.T) {
	t.Parallel()
	s := NewLookAheadScanner()
	// ref(close, 1) with positive offset = looking at past value (legitimate).
	result := s.Scan("ref(close, 1) > open")
	if !result.Passed {
		t.Fatalf("ref(close, 1) is a past reference, should pass: %+v", result.Violations)
	}
}

func TestLookAheadScanner_ComplexExpression(t *testing.T) {
	t.Parallel()
	s := NewLookAheadScanner()
	// Contains close[t+2] lookahead.
	result := s.Scan("close[t+2] > sma(close, 5) && volume > 1000")
	if result.Passed {
		t.Fatal("close[t+2] should be flagged")
	}
}

func TestHasLookahead(t *testing.T) {
	t.Parallel()
	if !HasLookahead("close[t+1] > open") {
		t.Fatal("HasLookahead should detect future reference")
	}
	if HasLookahead("sma(close, 5) > ema(close, 20)") {
		t.Fatal("clean expression should not have lookahead")
	}
}
