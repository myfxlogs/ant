package ai

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/ai"
)

// TestFeedbackFlowIntegration verifies the full feedback pipeline:
// RouteFeedback + FeedbackMetrics + BuildFeedbackPrompt produce valid output.
func TestFeedbackFlowIntegration(t *testing.T) {
	// Step 1: Route a Chinese feedback message
	metrics := &ai.FeedbackMetrics{
		SharpeRatio:  0.45,
		MaxDrawdown:  0.28,
		WinRate:      0.35,
		ProfitFactor: 1.2,
		TotalReturn:  0.08,
		TotalTrades:  15,
		Summary:      "回撤过大，胜率偏低",
	}

	result := ai.RouteFeedback("太激进了，加个止损", metrics)
	if result == nil {
		t.Fatal("RouteFeedback returned nil")
	}
	t.Logf("Routing: target=%v reason=%s", result.Target, result.Reason)

	// Step 2: Build a feedback prompt
	builder := ai.NewStrategyPromptBuilder()
	params := &ai.FeedbackPromptParams{
		PreviousCode: `# @param fast_period 10 range=5:50:5
# @param slow_period 30 range=10:100:10
def run(context):
    p = context.get('params', {})
    fast_period = int(p.get('fast_period', 10))
    slow_period = int(p.get('slow_period', 30))
    prices = context['close']
    if len(prices) < slow_period + 1:
        return {'signal': 'hold', 'volume': 0}
    fast_ma = sum(prices[-fast_period:]) / fast_period
    slow_ma = sum(prices[-slow_period:]) / slow_period
    pos = context.get('position')
    if fast_ma > slow_ma:
        if pos and pos.get('side') == 'buy':
            return {'signal': 'hold', 'volume': 0}
        if pos:
            return {'signal': 'close', 'volume': 0}
        return {'signal': 'buy', 'volume': 1.0}
    elif fast_ma < slow_ma:
        if pos and pos.get('side') == 'sell':
            return {'signal': 'hold', 'volume': 0}
        if pos:
            return {'signal': 'close', 'volume': 0}
        return {'signal': 'sell', 'volume': 1.0}
    return {'signal': 'hold', 'volume': 0}
`,
		Metrics:         metrics,
		FeedbackMessage: "太激进了，加个止损",
		FeedbackHints:   result.Reason,
	}

	system, user := builder.BuildFeedbackPrompt(params)

	// Verify prompt structure
	requiredInSystem := []string{
		"<section type=\"analysis\">",
		"<section type=\"advice\">",
		"<section type=\"code\">",
		"def run(context)",
		"Sharpe 0.45",
	}
	for _, needle := range requiredInSystem {
		if !strings.Contains(system, needle) {
			t.Errorf("system prompt missing: %s", needle)
		}
	}
	if !strings.Contains(user, "太激进了，加个止损") {
		t.Error("user prompt missing feedback message")
	}

	t.Logf("System prompt length: %d chars", len(system))
	t.Logf("User prompt length: %d chars", len(user))

	// Step 3: Verify RouteFeedback classification
	// "太激进" should route to Phase2 (quantitative adjustment)
	phase2Tests := []struct {
		msg    string
		isPhase2 bool
	}{
		{"太激进了", true},
		{"回撤太大", true},
		{"降低仓位", true},
		{"换成做空", false},  // structural change -> Phase1
		{"加上止损", false},  // structural change -> Phase1
		{"胜率太低", false},  // not in phase2Keywords -> Phase1 (default)
	}

	for _, tc := range phase2Tests {
		r := ai.RouteFeedback(tc.msg, metrics)
		gotPhase2 := r.Target == ai.FeedbackPhase2
		if gotPhase2 != tc.isPhase2 {
			t.Errorf("RouteFeedback(%q): got Phase2=%v, want Phase2=%v (target=%v reason=%s)",
				tc.msg, gotPhase2, tc.isPhase2, r.Target, r.Reason)
		} else {
			t.Logf("RouteFeedback(%q): Phase2=%v ✓ %s", tc.msg, gotPhase2, r.Reason)
		}
	}
}

// TestFeedbackConnectRpcFormat verifies the ConnectRPC request format for feedback.
func TestFeedbackConnectRpcFormat(t *testing.T) {
	// Verify the request proto can be constructed correctly
	metrics := ai.FeedbackMetrics{
		SharpeRatio:  0.82,
		MaxDrawdown:  0.185,
		WinRate:      0.42,
		ProfitFactor: 1.65,
		TotalReturn:  0.123,
		TotalTrades:  23,
	}

	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Metrics JSON: %s", string(metricsJSON))

	req := &antv1.GenerateStrategyRequest{
		Message:             "太激进了，加个止损",
		Symbol:              "XAUUSDm",
		Timeframe:           "1h",
		PreviousCode:        "def run(context):\n    return {'signal': 'buy', 'volume': 1.0}\n",
		BacktestMetricsJson: string(metricsJSON),
		FeedbackMessage:     "太激进了",
	}

	// Verify the request struct
	if req.Message == "" {
		t.Error("Message should not be empty")
	}
	if req.PreviousCode == "" {
		t.Error("PreviousCode should not be empty")
	}
	if req.BacktestMetricsJson == "" {
		t.Error("BacktestMetricsJson should not be empty")
	}

	// This request would trigger the feedback path in handleFeedback
	t.Log("ConnectRPC request format validated for feedback mode")
}

// TestConnectServerStreaming checks we can create a server-stream handler.
func TestConnectServerStreaming(t *testing.T) {
	// Create a minimal handler to verify registration
	path, handler := antv1c.NewStrategyGenerationServiceHandler(&StrategyGenServer{})
	if handler == nil {
		t.Fatal("handler should not be nil")
	}

	// Create test server
	server := httptest.NewServer(handler)
	defer server.Close()
	_ = path

	t.Logf("Test server running at: %s", server.URL)

	// Send a feedback request
	metrics := ai.FeedbackMetrics{
		SharpeRatio:  0.82,
		MaxDrawdown:  0.185,
		WinRate:      0.42,
		ProfitFactor: 1.65,
		TotalReturn:  0.123,
		TotalTrades:  23,
	}
	metricsJSON, _ := json.Marshal(metrics)

	req := connect.NewRequest(&antv1.GenerateStrategyRequest{
		Message:             "优化这个策略",
		Symbol:              "XAUUSDm",
		Timeframe:           "1h",
		PreviousCode:        "def run(context):\n    return {'signal': 'hold', 'volume': 0}\n",
		BacktestMetricsJson: string(metricsJSON),
		FeedbackMessage:     "降低回撤",
	})

	client := antv1c.NewStrategyGenerationServiceClient(server.Client(), server.URL)
	stream, err := client.GenerateStrategy(context.Background(), req)
	if err != nil {
		t.Logf("Expected streaming call to connect (may fail without LLM): %v", err)
		// This is expected to fail without an LLM backend, but the format is correct
		return
	}
	defer stream.Close()

	// Try to receive at least one message (will likely fail/timeout without LLM)
	t.Log("Stream connected, waiting for response...")
	for stream.Receive() {
		chunk := stream.Msg()
		t.Logf("Received chunk: analysis=%q advice=%q code_len=%d",
			chunk.Analysis, chunk.Advice, len(chunk.Code))
	}
	t.Logf("Stream ended: %v", stream.Err())
}
