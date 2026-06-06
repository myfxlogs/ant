package ai

import (
	"testing"

	"anttrader/internal/ai"
)

// TestParseSections_Complete verifies all three sections are extracted correctly.
func TestParseSections_Complete(t *testing.T) {
	raw := `<section type="analysis">
Sharpe 0.45 偏低，最大回撤 28% 超过风控线。
</section>
<section type="advice">
建议将 fast_period 从 5 调整到 10，加入 1% 止损。
</section>
<section type="code">
` + "```python" + `
def run(context):
    return {'signal': 'buy', 'volume': 1.0}
` + "```" + `
</section>`

	sections := parseSections(raw)

	if sections.Analysis == "" {
		t.Error("expected analysis section to be extracted")
	}
	if sections.Advice == "" {
		t.Error("expected advice section to be extracted")
	}
	if sections.Code == "" {
		t.Error("expected code section to be extracted")
	}
	t.Logf("analysis: %s", sections.Analysis)
	t.Logf("advice: %s", sections.Advice)
	t.Logf("code: %s", sections.Code)
}

// TestParseSections_MissingSections verifies graceful handling of missing sections.
func TestParseSections_MissingSections(t *testing.T) {
	raw := `<section type="code">
def run(context):
    return {'signal': 'sell', 'volume': 0.5}
</section>`

	sections := parseSections(raw)

	if sections.Analysis != "" {
		t.Error("expected empty analysis when section missing")
	}
	if sections.Advice != "" {
		t.Error("expected empty advice when section missing")
	}
	if sections.Code == "" {
		t.Error("expected code section to be extracted when present")
	}
}

// TestParseSections_Empty verifies handling of completely empty input.
func TestParseSections_Empty(t *testing.T) {
	sections := parseSections("")

	if sections.Analysis != "" || sections.Advice != "" || sections.Code != "" {
		t.Error("expected all sections empty for empty input")
	}
}

// TestParseSections_PartialStream verifies progressive parsing during streaming.
func TestParseSections_PartialStream(t *testing.T) {
	// Simulate streaming chunks arriving one at a time
	chunks := []string{
		"<section type=\"analysis\">\n",
		"Sharpe 0.45 偏低。",
		"\n</section>\n",
		"<section type=\"advice\">\n",
		"建议加入止损。",
	}

	var cumulative string
	for i, chunk := range chunks {
		cumulative += chunk
		sections := parseSections(cumulative)

		switch {
		case i < 2:
			// Analysis section not yet closed — should be empty (partial tag)
			// Actually, since we use .*? (non-greedy), it won't match until </section>
		case i >= 2 && i < 4:
			if sections.Analysis == "" {
				t.Errorf("chunk %d: expected analysis to be extracted once </section> arrives", i)
			}
		}
	}

	// After all chunks: analysis should be complete, advice not yet (no </section>)
	sections := parseSections(cumulative)
	if sections.Analysis == "" {
		t.Error("expected analysis to be complete after all chunks")
	}
	if sections.Advice != "" {
		t.Error("advice should not be extracted before </section> arrives")
	}
}

// TestBuildFeedbackPrompt verifies the feedback prompt includes all required sections.
func TestBuildFeedbackPrompt(t *testing.T) {
	builder := ai.NewStrategyPromptBuilder()
	metrics := &ai.FeedbackMetrics{
		SharpeRatio:  0.82,
		MaxDrawdown:  0.185,
		WinRate:      0.42,
		ProfitFactor: 1.65,
		TotalReturn:  0.123,
		TotalTrades:  23,
		Summary:      "策略方向正确但回撤偏大",
	}
	params := &ai.FeedbackPromptParams{
		PreviousCode:    "def run(context):\n    return {'signal': 'buy', 'volume': 1.0}\n",
		Metrics:         metrics,
		FeedbackMessage: "太激进了，加个止损",
		FeedbackHints:   "matched keyword: 太激进",
	}

	system, user := builder.BuildFeedbackPrompt(params)

	// System prompt must include required elements
	if system == "" {
		t.Fatal("system prompt is empty")
	}
	if user == "" {
		t.Fatal("user prompt is empty")
	}

	checks := []struct{ name, needle string }{
		{"section tags", "<section type=\"analysis\">"},
		{"section tags", "<section type=\"advice\">"},
		{"section tags", "<section type=\"code\">"},
		{"contract text", "def run(context)"},
		{"forbidden patterns", "eval()"},
		{"previous code", "def run(context)"},
		{"backtest metrics", "Sharpe 0.82"},
		{"backtest metrics", "最大回撤 18.5"},
		{"feedback hints", "太激进"},
	}

	for _, c := range checks {
		if !contains(system, c.needle) {
			t.Errorf("system prompt missing %q: %s", c.needle, c.name)
		}
	}

	// User prompt must include the feedback message
	if !contains(user, "太激进了，加个止损") {
		t.Error("user prompt missing feedback message")
	}

	t.Logf("=== SYSTEM PROMPT ===\n%s", system)
	t.Logf("=== USER PROMPT ===\n%s", user)
}

// TestBuildFeedbackPrompt_NilMetrics verifies prompt builder handles nil metrics gracefully.
func TestBuildFeedbackPrompt_NilMetrics(t *testing.T) {
	builder := ai.NewStrategyPromptBuilder()
	params := &ai.FeedbackPromptParams{
		PreviousCode:    "def run(context):\n    return {'signal': 'buy'}\n",
		Metrics:         nil,
		FeedbackMessage: "换个方向",
		FeedbackHints:   "",
	}

	system, user := builder.BuildFeedbackPrompt(params)

	if system == "" {
		t.Fatal("system prompt is empty")
	}
	// Should not crash, and should include default hints
	if !contains(system, "def run(context)") {
		t.Error("missing previous code in prompt")
	}
	if user == "" {
		t.Fatal("user prompt is empty")
	}
}

// TestSectionRegexCodeBlock verifies code extraction handles fenced code blocks inside sections.
func TestParseSections_FencedCodeBlock(t *testing.T) {
	raw := `<section type="analysis">Test analysis.</section>
<section type="code">
` + "```python" + `
import numpy as np

# @param fast_period 10 range=5:50:5
def run(context):
    close = np.array(context['close'])
    return {'signal': 'buy', 'volume': 1.0}
` + "```" + `
</section>`

	sections := parseSections(raw)

	if sections.Analysis == "" {
		t.Error("expected analysis")
	}
	if sections.Code == "" {
		t.Error("expected code with fenced block")
	}
	t.Logf("code: %s", sections.Code)
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) &&
		searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
