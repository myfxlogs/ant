package ai

import (
	"testing"
)

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"plain json", `{"key":"val"}`, `{"key":"val"}`},
		{"json with markdown fence", "```json\n{\"key\":\"val\"}\n```", `{"key":"val"}`},
		{"json with generic fence", "```\n{\"key\":\"val\"}\n```", `{"key":"val"}`},
		{"json with surrounding text", "some text {\"key\":\"val\"} more text", `{"key":"val"}`},
		{"empty", "", ""},
		{"nested json", `{"outer":{"inner":"val"}}`, `{"outer":{"inner":"val"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)
			if got != tt.want {
				t.Errorf("extractJSON(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDeduplicateStrings(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"no dups", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"with dups", []string{"a", "b", "a", "c"}, []string{"a", "b", "c"}},
		{"with spaces", []string{" a ", "a", " b "}, []string{"a", "b"}},
		{"empty strings filtered", []string{"", "a", "", "b"}, []string{"a", "b"}},
		{"empty input", []string{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicateStrings(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("deduplicateStrings(%v) len = %d, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("deduplicateStrings(%v)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestStringToStopLossPct(t *testing.T) {
	tests := []struct{ input, want string }{
		{"tight", "0.01"},
		{"medium", "0.03"},
		{"wide", "0.07"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := stringToStopLossPct(tt.input)
		if got != tt.want {
			t.Errorf("stringToStopLossPct(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStringToTakeProfitPct(t *testing.T) {
	tests := []struct{ input, want string }{
		{"tight", "0.02"},
		{"medium", "0.05"},
		{"wide", "0.10"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := stringToTakeProfitPct(tt.input)
		if got != tt.want {
			t.Errorf("stringToTakeProfitPct(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTruncateCode(t *testing.T) {
	tests := []struct {
		code   string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"this is too long", 7, "this is..."},
		{"exact", 5, "exact"},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncateCode(tt.code, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateCode(%q, %d) = %q, want %q", tt.code, tt.maxLen, got, tt.want)
		}
	}
}

func TestToParamMap(t *testing.T) {
	tests := []struct {
		name string
		r    IntentResult
		want map[string]string
	}{
		{
			"full result",
			IntentResult{
				StrategyFamily: "trend_following",
				RiskLevel:      "medium",
				MaxDrawdown:    "0.10",
				TradeDirection: "long",
				HoldingPeriod:  "swing",
			},
			map[string]string{
				"strategy_family": "trend_following",
				"risk_level":      "medium",
				"max_drawdown":    "0.10",
				"trade_direction": "long",
				"holding_period":  "swing",
			},
		},
		{
			"unknown values filtered",
			IntentResult{
				StrategyFamily: "unknown",
				RiskLevel:      "unknown",
				TradeDirection: "unknown",
				HoldingPeriod:  "unknown",
			},
			map[string]string{},
		},
		{
			"empty result",
			IntentResult{},
			map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.ToParamMap()
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("ToParamMap()[%q] = %q, want %q", k, got[k], v)
				}
			}
			for k := range got {
				if _, ok := tt.want[k]; !ok {
					t.Errorf("ToParamMap() unexpected key %q = %q", k, got[k])
				}
			}
		})
	}
}

func TestMatchAnyKeyword(t *testing.T) {
	tests := []struct {
		msg      string
		keywords []string
		want     bool
	}{
		{"i want a trend following strategy", []string{"trend", "momentum"}, true},
		{"no match here", []string{"trend", "momentum"}, false},
		{"case sensitive match", []string{"MATCH"}, true},
		{"empty keywords", []string{}, false},
		{"empty message", []string{"test"}, false},
	}
	for _, tt := range tests {
		got := matchAnyKeyword(tt.msg, tt.keywords)
		if got != tt.want {
			t.Errorf("matchAnyKeyword(%q, %v) = %v, want %v", tt.msg, tt.keywords, got, tt.want)
		}
	}
}

func TestFirstMatch(t *testing.T) {
	tests := []struct {
		msg      string
		keywords []string
		want     string
	}{
		{"trend following strategy", []string{"trend", "mean_reversion"}, "trend"},
		{"mean entry", []string{"trend", "mean"}, "mean"},
		{"no match", []string{"trend", "momentum"}, ""},
		{"empty keywords", []string{}, ""},
	}
	for _, tt := range tests {
		got := firstMatch(tt.msg, tt.keywords)
		if got != tt.want {
			t.Errorf("firstMatch(%q, %v) = %q, want %q", tt.msg, tt.keywords, got, tt.want)
		}
	}
}

func TestBuildIntentUserMessage(t *testing.T) {
	got := buildIntentUserMessage("test message", "EURUSD", "1H")
	if got == "" {
		t.Error("buildIntentUserMessage returned empty")
	}
	if !contains(got, "test message") || !contains(got, "EURUSD") || !contains(got, "1H") {
		t.Errorf("buildIntentUserMessage missing input data: %s", got)
	}
}

func TestBuildFeedbackUserMessage(t *testing.T) {
	got := buildFeedbackUserMessage("too aggressive", "def run(ctx): pass", "sharpe:1.5")
	if got == "" {
		t.Error("buildFeedbackUserMessage returned empty")
	}
	if !contains(got, "too aggressive") || !contains(got, "def run") || !contains(got, "1.5") {
		t.Errorf("buildFeedbackUserMessage missing input data: %s", got)
	}
}

func TestNewIntentAnalyzer(t *testing.T) {
	a := NewIntentAnalyzer(nil)
	if a == nil {
		t.Error("NewIntentAnalyzer returned nil")
	}
}

func TestNewClarificationEngine(t *testing.T) {
	rules := []ClarificationRule{{Keywords: []string{"test"}, Questions: []string{"q1"}}}
	e := NewClarificationEngine(rules)
	if e == nil {
		t.Error("NewClarificationEngine returned nil")
	}
}

func TestClarificationEngine_Check(t *testing.T) {
	rules := []ClarificationRule{
		{
			Keywords:  []string{"trend"},
			Questions: []string{"What timeframe?"},
			ParamMap:  map[string]string{"strategy_family": "trend_following"},
			Priority:  1,
		},
	}
	e := NewClarificationEngine(rules)

	t.Run("match", func(t *testing.T) {
		got := e.Check("i want a trend following strategy")
		if !got.NeedsClarification {
			t.Error("expected NeedsClarification=true")
		}
		if got.MatchedKeyword != "trend" {
			t.Errorf("MatchedKeyword = %q, want %q", got.MatchedKeyword, "trend")
		}
	})

	t.Run("no match", func(t *testing.T) {
		got := e.Check("no keywords here")
		if got != nil {
			t.Error("expected nil for no match")
		}
	})
}

func TestFeedbackMetrics_FormatPromptContext(t *testing.T) {
	m := &FeedbackMetrics{
		SharpeRatio: 1.5, MaxDrawdown: 0.2, WinRate: 0.6,
		ProfitFactor: 1.8, TotalReturn: 0.3, TotalTrades: 50,
		Summary: "good strategy",
	}
	got := m.FormatPromptContext()
	if got == "" {
		t.Error("FormatPromptContext returned empty")
	}
	if !contains(got, "1.50") || !contains(got, "20.0") || !contains(got, "good strategy") {
		t.Errorf("FormatPromptContext missing data: %s", got)
	}
}

func TestFeedbackMetrics_Summarize(t *testing.T) {
	tests := []struct {
		name string
		m    FeedbackMetrics
		want string
	}{
		{"high drawdown", FeedbackMetrics{MaxDrawdown: 0.30, SharpeRatio: 1.0, TotalTrades: 50, TotalReturn: 0.1}, "回撤偏大"},
		{"low sharpe", FeedbackMetrics{MaxDrawdown: 0.10, SharpeRatio: 0.3, TotalTrades: 50, TotalReturn: 0.1}, "Sharpe偏低"},
		{"few trades", FeedbackMetrics{MaxDrawdown: 0.10, SharpeRatio: 1.0, TotalTrades: 5, TotalReturn: 0.1}, "交易次数少"},
		{"negative return", FeedbackMetrics{MaxDrawdown: 0.10, SharpeRatio: 1.0, TotalTrades: 50, TotalReturn: -0.1}, "总收益为负"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.m.Summarize()
			if !contains(got, tt.want) {
				t.Errorf("Summarize() = %q, want containing %q", got, tt.want)
			}
		})
	}
}

func TestFromBacktestResult(t *testing.T) {
	t.Run("nil metrics", func(t *testing.T) {
		got := FromBacktestResult(nil, "code")
		if got == nil || got.Summary != "回测未产生结果" {
			t.Errorf("FromBacktestResult(nil) = %v, want summary '回测未产生结果'", got)
		}
	})
	t.Run("valid metrics", func(t *testing.T) {
		bm := &BacktestMetrics{SharpeRatio: 1.5, MaxDrawdown: 0.15, WinRate: 0.6, ProfitFactor: 1.8, TotalReturn: 0.3, TotalTrades: 50}
		got := FromBacktestResult(bm, "def run(ctx): pass")
		if got.SharpeRatio != 1.5 || got.TotalTrades != 50 {
			t.Errorf("FromBacktestResult: got Sharpe=%.2f Trades=%d", got.SharpeRatio, got.TotalTrades)
		}
	})
}

func TestValidateOutcome(t *testing.T) {
	tests := []struct {
		name       string
		decision   string
		actualRet  float64
		thresholds []float64
		want       bool
	}{
		{"buy win", "BUY", 0.05, nil, true},
		{"buy lose", "BUY", -0.01, nil, false},
		{"sell win", "SELL", -0.05, nil, true},
		{"sell lose", "SELL", 0.01, nil, false},
		{"hold within", "HOLD", 0.03, nil, true},
		{"hold outside", "HOLD", 0.10, nil, false},
		{"custom thresholds", "BUY", 0.03, []float64{0.02, -0.02, 0.05}, true},
		{"unknown decision", "UNKNOWN", 0.05, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateOutcome(tt.decision, tt.actualRet, tt.thresholds...)
			if got != tt.want {
				t.Errorf("ValidateOutcome(%q, %.2f, %v) = %v, want %v",
					tt.decision, tt.actualRet, tt.thresholds, got, tt.want)
			}
		})
	}
}

func TestNewCalibrationService(t *testing.T) {
	s := NewCalibrationService(nil)
	if s == nil {
		t.Error("NewCalibrationService returned nil")
	}
}

func TestNewCalibrationRepository(t *testing.T) {
	r := NewCalibrationRepository(nil)
	if r == nil {
		t.Error("NewCalibrationRepository returned nil")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
