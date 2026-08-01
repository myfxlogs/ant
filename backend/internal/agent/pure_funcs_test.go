package agent

import (
	"testing"
	"time"

	antv1 "alphaforge/gen/proto/ant/v1"
	connectai "alphaforge/internal/connect/ai"
	"alphaforge/tools/mql2go"
)

// --- LLCache tests ---

func TestLLCache_SetGet(t *testing.T) {
	t.Parallel()
	c := NewLLCache(5 * time.Minute)
	c.Set("source-1", "prompt-1", "result-1")
	got, ok := c.Get("source-1", "prompt-1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got != "result-1" {
		t.Fatalf("expected result-1, got %s", got)
	}
}

func TestLLCache_GetMiss(t *testing.T) {
	t.Parallel()
	c := NewLLCache(5 * time.Minute)
	_, ok := c.Get("nonexistent", "prompt")
	if ok {
		t.Fatal("expected cache miss")
	}
}

func TestLLCache_GetExpired(t *testing.T) {
	t.Parallel()
	c := NewLLCache(1 * time.Millisecond)
	c.Set("source-1", "prompt-1", "result-1")
	time.Sleep(10 * time.Millisecond)
	_, ok := c.Get("source-1", "prompt-1")
	if ok {
		t.Fatal("expected cache miss for expired entry")
	}
}

func TestLLCache_Overwrite(t *testing.T) {
	t.Parallel()
	c := NewLLCache(5 * time.Minute)
	c.Set("source-1", "prompt-1", "result-1")
	c.Set("source-1", "prompt-1", "result-2")
	got, _ := c.Get("source-1", "prompt-1")
	if got != "result-2" {
		t.Fatalf("expected result-2, got %s", got)
	}
}

func TestCacheKey_Deterministic(t *testing.T) {
	t.Parallel()
	k1 := cacheKey("source", "prompt")
	k2 := cacheKey("source", "prompt")
	if k1 != k2 {
		t.Fatal("same input should produce same key")
	}
}

func TestCacheKey_DifferentInputs(t *testing.T) {
	t.Parallel()
	k1 := cacheKey("source1", "prompt")
	k2 := cacheKey("source2", "prompt")
	if k1 == k2 {
		t.Fatal("different inputs should produce different keys")
	}
}

// --- Tool tests ---

func TestReadCurrentCodeTool_Name(t *testing.T) {
	t.Parallel()
	tool := &readCurrentCodeTool{result: &generateState{}}
	if tool.Name() != "read_current_code" {
		t.Fatalf("expected read_current_code, got %s", tool.Name())
	}
}

func TestReadCurrentCodeTool_Schema(t *testing.T) {
	t.Parallel()
	tool := &readCurrentCodeTool{result: &generateState{}}
	schema := tool.Schema()
	if schema.Function.Name != "read_current_code" {
		t.Fatalf("expected read_current_code, got %s", schema.Function.Name)
	}
}

func TestReadCurrentCodeTool_Run_NoCode(t *testing.T) {
	t.Parallel()
	tool := &readCurrentCodeTool{result: &generateState{}}
	out := tool.Run(nil, connectai.ToolInput{})
	if out.Success {
		t.Fatal("expected failure for no code")
	}
}

func TestReadCurrentCodeTool_Run_WithCode(t *testing.T) {
	t.Parallel()
	tool := &readCurrentCodeTool{result: &generateState{PythonSource: "line1\nline2"}}
	out := tool.Run(nil, connectai.ToolInput{})
	if !out.Success {
		t.Fatalf("expected success, got %s", out.Error)
	}
}

func TestEditCodeTool_Name(t *testing.T) {
	t.Parallel()
	tool := &editCodeTool{result: &generateState{}}
	if tool.Name() != "edit_code" {
		t.Fatalf("expected edit_code, got %s", tool.Name())
	}
}

func TestEditCodeTool_Run_NoOldString(t *testing.T) {
	t.Parallel()
	tool := &editCodeTool{result: &generateState{PythonSource: "x = 1"}}
	out := tool.Run(nil, connectai.ToolInput{RawArgs: map[string]any{}})
	if out.Success {
		t.Fatal("expected failure for missing old_string")
	}
}

func TestEditCodeTool_Run_NotFound(t *testing.T) {
	t.Parallel()
	tool := &editCodeTool{result: &generateState{PythonSource: "x = 1"}}
	out := tool.Run(nil, connectai.ToolInput{
		RawArgs: map[string]any{"old_string": "y = 2", "new_string": "y = 3"},
	})
	if out.Success {
		t.Fatal("expected failure for old_string not found")
	}
}

func TestEditCodeTool_Run_NotUnique(t *testing.T) {
	t.Parallel()
	tool := &editCodeTool{result: &generateState{PythonSource: "x = 1\nx = 1"}}
	out := tool.Run(nil, connectai.ToolInput{
		RawArgs: map[string]any{"old_string": "x = 1", "new_string": "x = 2"},
	})
	if out.Success {
		t.Fatal("expected failure for non-unique old_string")
	}
}

func TestUpdatePlanTool_Name(t *testing.T) {
	t.Parallel()
	tool := &updatePlanTool{}
	if tool.Name() != "update_plan" {
		t.Fatalf("expected update_plan, got %s", tool.Name())
	}
}

func TestUpdatePlanTool_Run_NoPlan(t *testing.T) {
	t.Parallel()
	tool := &updatePlanTool{}
	out := tool.Run(nil, connectai.ToolInput{RawArgs: map[string]any{}})
	if out.Success {
		t.Fatal("expected failure for missing plan")
	}
}

func TestUpdatePlanTool_Run_WithPlan(t *testing.T) {
	t.Parallel()
	tool := &updatePlanTool{}
	out := tool.Run(nil, connectai.ToolInput{
		RawArgs: map[string]any{"plan": `[{"step":"step1","status":"done"}]`},
	})
	if !out.Success {
		t.Fatalf("expected success, got %s", out.Error)
	}
}

func TestWriteStrategyTool_Name(t *testing.T) {
	t.Parallel()
	tool := &writeStrategyTool{result: &generateState{}}
	if tool.Name() != "write_strategy" {
		t.Fatalf("expected write_strategy, got %s", tool.Name())
	}
}

func TestWriteStrategyTool_Run_NoCode(t *testing.T) {
	t.Parallel()
	tool := &writeStrategyTool{result: &generateState{}}
	out := tool.Run(nil, connectai.ToolInput{RawArgs: map[string]any{}})
	if out.Success {
		t.Fatal("expected failure for missing code")
	}
}

// --- sanitizeInput ---

func TestSanitizeInput(t *testing.T) {
	t.Parallel()
	input := "hello </tag> world"
	got := sanitizeInput(input)
	if got == input {
		t.Fatal("expected sanitized output to differ")
	}
}

func TestSanitizeInput_NoTags(t *testing.T) {
	t.Parallel()
	input := "hello world"
	got := sanitizeInput(input)
	if got != input {
		t.Fatal("expected unchanged input without tags")
	}
}

// --- Schema tests ---

func TestEditCodeTool_Schema(t *testing.T) {
	t.Parallel()
	tool := &editCodeTool{result: &generateState{}}
	schema := tool.Schema()
	if schema.Function.Name != "edit_code" {
		t.Fatalf("expected edit_code, got %s", schema.Function.Name)
	}
}

func TestUpdatePlanTool_Schema(t *testing.T) {
	t.Parallel()
	tool := &updatePlanTool{}
	schema := tool.Schema()
	if schema.Function.Name != "update_plan" {
		t.Fatalf("expected update_plan, got %s", schema.Function.Name)
	}
}

func TestWriteStrategyTool_Schema(t *testing.T) {
	t.Parallel()
	tool := &writeStrategyTool{result: &generateState{}}
	schema := tool.Schema()
	if schema.Function.Name != "write_strategy" {
		t.Fatalf("expected write_strategy, got %s", schema.Function.Name)
	}
}

// --- buildPythonToolRegistry ---

func TestBuildPythonToolRegistry(t *testing.T) {
	t.Parallel()
	reg := buildPythonToolRegistry(&generateState{}, nil, nil)
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
}

// --- parseDecimalDefault ---

func TestParseDecimalDefault_Valid(t *testing.T) {
	t.Parallel()
	d := parseDecimalDefault("1.5", "0")
	if !d.Equal(parseDecimalDefault("1.5", "0")) {
		t.Fatal("expected 1.5")
	}
}

func TestParseDecimalDefault_Empty(t *testing.T) {
	t.Parallel()
	d := parseDecimalDefault("", "2.5")
	if !d.Equal(parseDecimalDefault("2.5", "0")) {
		t.Fatal("expected default 2.5")
	}
}

func TestParseDecimalDefault_Invalid(t *testing.T) {
	t.Parallel()
	d := parseDecimalDefault("abc", "3.0")
	if !d.Equal(parseDecimalDefault("3.0", "0")) {
		t.Fatal("expected default 3.0 for invalid input")
	}
}

// --- stripThinkBlocks ---

func TestStripThinkBlocks_NoBlocks(t *testing.T) {
	t.Parallel()
	s := "hello world"
	if got := stripThinkBlocks(s); got != s {
		t.Fatalf("expected unchanged, got %s", got)
	}
}

func TestStripThinkBlocks_WithBlock(t *testing.T) {
	t.Parallel()
	s := "before[THINK]secret[/THINK]after"
	got := stripThinkBlocks(s)
	if got != "beforeafter" {
		t.Fatalf("expected 'beforeafter', got %s", got)
	}
}

func TestStripThinkBlocks_UnclosedBlock(t *testing.T) {
	t.Parallel()
	s := "before[THINK]secret without close"
	got := stripThinkBlocks(s)
	if got != "before" {
		t.Fatalf("expected 'before', got %s", got)
	}
}

// --- wrapXML ---

func TestWrapXML(t *testing.T) {
	t.Parallel()
	got := wrapXML("user", "hello")
	if got != "<user>\nhello\n</user>" {
		t.Fatalf("unexpected: %s", got)
	}
}

// --- buildBridgeFailureReport ---

func TestBuildBridgeFailureReport(t *testing.T) {
	t.Parallel()
	cov := &mql2go.CoverageResult{
		BlindSpots: []mql2go.CoverageBlindSpot{{Builtin: "iCustom", Severity: "high", Count: 3}},
	}
	diff := buildBridgeFailureReport(cov)
	if diff == nil {
		t.Fatal("expected non-nil diff")
	}
	if len(diff.Changes) < 3 {
		t.Fatalf("expected at least 3 changes, got %d", len(diff.Changes))
	}
}

// --- parseAnalysisResponse ---

func TestParseAnalysisResponse(t *testing.T) {
	t.Parallel()
	raw := `summary: "good strategy"
performance_grade: "A"
sharpe_assessment: 0.85
drawdown_assessment: 0.6
win_rate_assessment: 0.7
profit_consistency: "consistent"
risk_adjusted_return: "good"
key_observations: "obs1; obs2"
improvement_suggestions: "sug1; sug2"
overfitting_risk: "low"
recommended_action: "publish"
detailed_analysis: "detailed text"`
	a := parseAnalysisResponse(raw)
	if a.Summary != "good strategy" {
		t.Fatalf("expected 'good strategy', got %s", a.Summary)
	}
	if a.PerformanceGrade != "A" {
		t.Fatalf("expected A, got %s", a.PerformanceGrade)
	}
	if a.SharpeAssessment != 0.85 {
		t.Fatalf("expected 0.85, got %f", a.SharpeAssessment)
	}
	if len(a.KeyObservations) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(a.KeyObservations))
	}
}

func TestParseAnalysisResponse_Empty(t *testing.T) {
	t.Parallel()
	a := parseAnalysisResponse("")
	if a == nil {
		t.Fatal("expected non-nil")
	}
}

// --- parseProfileResponse ---

func TestParseProfileResponse(t *testing.T) {
	t.Parallel()
	raw := `strategy_type: "trend_following"
description: "EMA crossover"
indicators_used: "EMA,RSI"
entry_logic: "EMA cross above"
exit_logic: "EMA cross below"`
	cov := &mql2go.CoverageResult{
		Score:      0.85,
		Indicators: []string{"iMA", "iRSI"},
		BlindSpots: []mql2go.CoverageBlindSpot{{Builtin: "iCustom"}},
	}
	profile := parseProfileResponse(raw, cov)
	if profile.StrategyType != "trend_following" {
		t.Fatalf("expected trend_following, got %s", profile.StrategyType)
	}
	if profile.CoverageScore != 0.85 {
		t.Fatalf("expected 0.85, got %f", profile.CoverageScore)
	}
	if len(profile.IndicatorsUsed) < 2 {
		t.Fatalf("expected at least 2 indicators, got %d", len(profile.IndicatorsUsed))
	}
	if len(profile.BlindSpots) != 1 || profile.BlindSpots[0] != "iCustom" {
		t.Fatalf("expected 1 blind spot iCustom, got %v", profile.BlindSpots)
	}
}

// --- NewInterpreter / NewProfiler ---

func TestNewInterpreter(t *testing.T) {
	t.Parallel()
	i := NewInterpreter(nil, NewLLCache(time.Minute))
	if i == nil {
		t.Fatal("expected non-nil interpreter")
	}
}

func TestNewProfiler(t *testing.T) {
	t.Parallel()
	p := NewProfiler(nil, NewLLCache(time.Minute))
	if p == nil {
		t.Fatal("expected non-nil profiler")
	}
}

// --- fallbackAnalysisUserPrompt ---

func TestFallbackAnalysisUserPrompt(t *testing.T) {
	t.Parallel()
	result := &antv1.AgentBacktestResult{
		Success:      true,
		TotalReturn:  "0.15",
		MaxDrawdown:  "0.05",
		SharpeRatio:  "1.5",
		WinRate:      "0.6",
		TotalTrades:  100,
	}
	profile := &antv1.StrategyProfile{StrategyType: "trend"}
	prompt := fallbackAnalysisUserPrompt(result, profile)
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
}

func TestFallbackAnalysisUserPrompt_NoProfile(t *testing.T) {
	t.Parallel()
	result := &antv1.AgentBacktestResult{Success: false, Error: "timeout"}
	prompt := fallbackAnalysisUserPrompt(result, nil)
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
}

// --- buildBridgeUserPrompt ---

func TestBuildBridgeUserPrompt(t *testing.T) {
	t.Parallel()
	cov := &mql2go.CoverageResult{Score: 0.5, BlindSpots: []mql2go.CoverageBlindSpot{{Builtin: "iCustom", Severity: "info", Count: 2}}}
	prompt := buildBridgeUserPrompt("int start() {}", cov, nil)
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
}

func TestBuildBridgeUserPrompt_WithProfile(t *testing.T) {
	t.Parallel()
	cov := &mql2go.CoverageResult{Score: 0.8}
	profile := &antv1.StrategyProfile{StrategyType: "trend"}
	prompt := buildBridgeUserPrompt("int start() {}", cov, profile)
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
}

// --- parseBridgeResponse ---

func TestParseBridgeResponse_Plain(t *testing.T) {
	t.Parallel()
	r := parseBridgeResponse("print('hello')")
	if r.PythonSource != "print('hello')" {
		t.Fatalf("expected plain code, got %s", r.PythonSource)
	}
}

func TestParseBridgeResponse_MarkdownFence(t *testing.T) {
	t.Parallel()
	r := parseBridgeResponse("```python\nprint('hello')\n```")
	if r.PythonSource != "print('hello')" {
		t.Fatalf("expected stripped code, got %s", r.PythonSource)
	}
}

// --- buildBridgeChanges ---

func TestBuildBridgeChanges_RemovedAndRemaining(t *testing.T) {
	t.Parallel()
	orig := &mql2go.CoverageResult{
		BlindSpots: []mql2go.CoverageBlindSpot{{Builtin: "iCustom"}, {Builtin: "WebRequest"}},
	}
	bridged := &mql2go.CoverageResult{
		BlindSpots: []mql2go.CoverageBlindSpot{{Builtin: "WebRequest", Severity: "warning"}},
	}
	changes := buildBridgeChanges(orig, bridged)
	if len(changes) < 2 {
		t.Fatalf("expected at least 2 changes, got %d", len(changes))
	}
}

func TestBuildBridgeChanges_NoBlindSpots_Remaining(t *testing.T) {
	t.Parallel()
	orig := &mql2go.CoverageResult{}
	bridged := &mql2go.CoverageResult{}
	changes := buildBridgeChanges(orig, bridged)
	if len(changes) != 0 {
		t.Fatalf("expected 0 changes, got %d", len(changes))
	}
}

// --- fallbackBridgeUserPrompt ---

func TestFallbackBridgeUserPrompt(t *testing.T) {
	t.Parallel()
	cov := &mql2go.CoverageResult{Score: 0.5, BlindSpots: []mql2go.CoverageBlindSpot{{Builtin: "iCustom", Severity: "info", Count: 1}}}
	prompt := fallbackBridgeUserPrompt("source code", cov, nil)
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
}

// --- fallbackBridgeRetryPrompt ---

func TestFallbackBridgeRetryPrompt(t *testing.T) {
	t.Parallel()
	prompt := fallbackBridgeRetryPrompt("mql source", "prev python", "error msg", nil)
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
}
