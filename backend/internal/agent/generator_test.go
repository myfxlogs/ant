package agent

import (
	"testing"
)

// TestParseProfileResponseNL verifies parsing of LLM profile output from NL.
func TestParseProfileResponseNL(t *testing.T) {
	raw := `strategy_type: "mean_reversion"
description: "RSI oversold bounce"
indicators_used: "RSI,EMA"
entry_logic: "RSI below 30 then crosses back above"
exit_logic: "RSI above 70"
risk_management: "stop-loss 30 pips"
timeframe_preference: "M15"
market_regime: "ranging"
strengths: "simple, clear rules"
weaknesses: "may underperform in trends"`

	profile := parseProfileLines(raw)

	if profile.StrategyType != "mean_reversion" {
		t.Errorf("strategy_type = %q, want 'mean_reversion'", profile.StrategyType)
	}
	if profile.Description != "RSI oversold bounce" {
		t.Errorf("description = %q, want 'RSI oversold bounce'", profile.Description)
	}
	if len(profile.IndicatorsUsed) != 2 {
		t.Errorf("indicators_used len = %d, want 2", len(profile.IndicatorsUsed))
	}
	if profile.EntryLogic != "RSI below 30 then crosses back above" {
		t.Errorf("entry_logic = %q", profile.EntryLogic)
	}
	if profile.RiskManagement != "stop-loss 30 pips" {
		t.Errorf("risk_management = %q", profile.RiskManagement)
	}
	if len(profile.Strengths) != 2 {
		t.Errorf("strengths len = %d, want 2", len(profile.Strengths))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
