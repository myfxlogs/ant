package agent

import (
	"testing"

	antv1 "anttrader/gen/proto/ant/v1"
)

// TestBuildGeneratePrompt verifies the prompt construction for first-attempt generation.
func TestBuildGeneratePrompt(t *testing.T) {
	msg := &antv1.AgentGenerateStrategyRequest{
		Message:   "EURUSD H1, EMA10 crosses above EMA30, buy",
		Symbol:    "EURUSD",
		Timeframe: "H1",
	}

	sysPrompt, userPrompt := buildGeneratePrompt(msg, nil, nil)

	if sysPrompt == "" {
		t.Error("system prompt should not be empty")
	}
	if userPrompt == "" {
		t.Error("user prompt should not be empty")
	}

	// System prompt should contain Python subset rules
	if !contains(sysPrompt, "IRON RULES") {
		t.Error("system prompt should contain Python subset rules")
	}
	if !contains(sysPrompt, "ctx.bars().close(0)") {
		t.Error("system prompt should contain SDK API mapping")
	}

	// User prompt should contain the strategy description
	if !contains(userPrompt, "EURUSD") {
		t.Error("user prompt should contain symbol")
	}
	if !contains(userPrompt, "EMA10") {
		t.Error("user prompt should contain strategy description")
	}
	if !contains(userPrompt, "H1") {
		t.Error("user prompt should contain timeframe")
	}
}

// TestBuildGenerateRetryPrompt verifies the retry prompt includes error context.
func TestBuildGenerateRetryPrompt(t *testing.T) {
	msg := &antv1.AgentGenerateStrategyRequest{
		Message: "RSI oversold bounce strategy",
		Symbol:  "GBPUSD",
	}

	sysPrompt, userPrompt := buildGenerateRetryPrompt(msg, "class S:\n    pass", "syntax error at line 3", "", nil, nil)

	if sysPrompt == "" {
		t.Error("system prompt should not be empty")
	}
	if !contains(userPrompt, "syntax error at line 3") {
		t.Error("retry prompt should contain compile error")
	}
	if !contains(userPrompt, "class S:") {
		t.Error("retry prompt should contain previous code")
	}
	if !contains(userPrompt, "RSI oversold") {
		t.Error("retry prompt should contain original description")
	}
}

// TestBuildGenerateRetryPrompt_BacktestError verifies retry with backtest error.
func TestBuildGenerateRetryPrompt_BacktestError(t *testing.T) {
	msg := &antv1.AgentGenerateStrategyRequest{
		Message: "Bollinger Band mean reversion",
	}

	_, userPrompt := buildGenerateRetryPrompt(msg, "class S:\n    pass", "", "backtest: no trades executed", nil, nil)

	if !contains(userPrompt, "no trades executed") {
		t.Error("retry prompt should contain backtest error")
	}
}

// TestBuildGeneratePrompt_WithParams verifies params are included in prompt.
func TestBuildGeneratePrompt_WithParams(t *testing.T) {
	msg := &antv1.AgentGenerateStrategyRequest{
		Message: "MACD divergence strategy",
		Symbol:  "USDJPY",
		Params:  map[string]string{"fast": "12", "slow": "26", "signal": "9"},
	}

	_, userPrompt := buildGeneratePrompt(msg, nil, nil)

	if !contains(userPrompt, "fast: 12") {
		t.Error("user prompt should contain param overrides")
	}
	if !contains(userPrompt, "signal: 9") {
		t.Error("user prompt should contain param overrides")
	}
}

// TestBuildGeneratePrompt_WithProfile verifies profile is injected into the prompt.
func TestBuildGeneratePrompt_WithProfile(t *testing.T) {
	msg := &antv1.AgentGenerateStrategyRequest{
		Message: "EMA crossover strategy",
	}
	profile := &antv1.StrategyProfile{
		StrategyType:    "trend_following",
		Description:     "EMA10/EMA30 crossover",
		IndicatorsUsed:  []string{"EMA"},
		EntryLogic:      "EMA10 crosses above EMA30",
		ExitLogic:       "EMA10 crosses below EMA30",
		RiskManagement:  "fixed stop-loss 50 pips",
	}

	_, userPrompt := buildGeneratePrompt(msg, profile, nil)

	if !contains(userPrompt, "trend_following") {
		t.Error("user prompt should contain profile strategy type")
	}
	if !contains(userPrompt, "EMA10 crosses above EMA30") {
		t.Error("user prompt should contain profile entry logic")
	}
	if !contains(userPrompt, "fixed stop-loss 50 pips") {
		t.Error("user prompt should contain profile risk management")
	}
}

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
