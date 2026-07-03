package agent

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service/systemai"
	"anttrader/tools/mql2go"
)

// Bridge orchestrates the blind-spot bridging Agent loop.
// ADR-0024 Phase 2: MQL blind-spot EA → LLM translates to Python subset → compile_py → VM backtest.
//
// The bridge does NOT execute strategies itself — it calls LLM to translate MQL
// (with blind spots) into Python subset code, then submits the Python code
// through the normal GatewayServer.SubmitStrategy path.
type Bridge struct {
	aiSvc *systemai.Service
	log   *zap.Logger
	cache *LLCache
}

// NewBridge creates the blind-spot bridging orchestrator.
func NewBridge(aiSvc *systemai.Service, log *zap.Logger, cache *LLCache) *Bridge {
	return &Bridge{aiSvc: aiSvc, log: log, cache: cache}
}

const bridgeSystemPrompt = `You are a quantitative strategy translator. Your task is to translate an MQL trading strategy with blind spots into an equivalent Python subset strategy.

## Python Subset Rules
- Class-based: class MyStrategy: with methods on_init, on_bar, on_tick, on_timer, on_deinit
- __init__ params become strategy parameters with type annotations
- All methods must have return type annotations (-> None, -> int, etc.)
- Allowed import: ONLY "from decimal import Decimal"
- NO: list comprehensions, lambda, try/except, with, yield, decorators, async/await
- NO: exec, eval, open, print, len, sorted, sum, enumerate, zip, range (outside for-loops)
- NO: f-strings, walrus operator (:=), global/nonlocal, del, assert, raise
- NO: slicing, tuple unpacking, *args, **kwargs

## SDK API Mapping (MQL → Python subset)
- Close[0] → ctx.bars().close(0)
- Open[0] → ctx.bars().open(0)
- High[0] → ctx.bars().high(0)
- Low[0] → ctx.bars().low(0)
- Volume[0] → ctx.bars().volume(0)
- iMA(symbol, period, shift) → ctx.indicators().ima(ctx.symbol(), period, shift)
- iRSI(symbol, period, shift) → ctx.indicators().irsi(ctx.symbol(), period, shift)
- iATR(symbol, period, shift) → ctx.indicators().iatr(ctx.symbol(), period, shift)
- OrderSend(...) → ctx.broker().buy(lot=Decimal("0.1")) or ctx.broker().sell(lot=Decimal("0.1"))
- OrderClose(...) → ctx.broker().close(ticket)
- OrderModify(...) → ctx.broker().modify(ticket, sl, tp)
- PositionsTotal() → ctx.positions().count()
- Bid → ctx.bars().bid()
- Ask → ctx.bars().ask()
- Point → ctx.point()
- Symbol() → ctx.symbol()

## Output Format
Output ONLY the Python source code, no markdown fences, no explanations.
The code must be a complete, compilable Python subset strategy.

## Blind Spot Handling
- iCustom → replace with equivalent standard indicator or comment as limitation
- ObjectCreate/ObjectDelete → remove (UI operations not relevant to backtest)
- WebRequest → remove (network calls not allowed in VM)
- FileOpen/FileWrite → remove (file I/O not allowed in VM)
- EventSetTimer → map to on_timer method
- OnTradeTransaction → map to on_trade_transaction method`

// BridgeResult holds the output of a bridge translation attempt.
type BridgeResult struct {
	PythonSource  string
	SemanticDiff  *antv1.SemanticDiff
	CompileError  string
	Translated    bool
	Status        string // "success" | "bridge_failed"
	Attempts      int
}

// TranslateBlindSpot calls LLM to translate MQL with blind spots into Python subset.
// ADR-0024 Phase 2: "盲区桥接 prompt + Agent 循环编排"
func (b *Bridge) TranslateBlindSpot(
	ctx context.Context,
	userID string,
	mqlSource string,
	coverage *mql2go.CoverageResult,
) (*BridgeResult, error) {
	userPrompt := buildBridgeUserPrompt(mqlSource, coverage)

	// Check cache
	if b.cache != nil {
		if cached, ok := b.cache.Get(mqlSource, userPrompt); ok {
			return parseBridgeResponse(cached), nil
		}
	}

	uid, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	resp, err := b.aiSvc.ChatCompletion(ctx, uid, []systemai.ChatMessage{
		{Role: "system", Content: bridgeSystemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return nil, fmt.Errorf("bridge LLM call: %w", err)
	}

	if b.cache != nil {
		b.cache.Set(mqlSource, userPrompt, resp)
	}

	return parseBridgeResponse(resp), nil
}

const maxBridgeRetries = 3

// TranslateWithRetry runs the bridge Agent loop: LLM translate → compile → error feedback → retry.
// ADR-0024 §5.4: max 3 retries with compile error feedback to LLM.
func (b *Bridge) TranslateWithRetry(
	ctx context.Context,
	userID string,
	mqlSource string,
	coverage *mql2go.CoverageResult,
) (*BridgeResult, error) {
	result, err := b.TranslateBlindSpot(ctx, userID, mqlSource, coverage)
	if err != nil {
		return nil, err
	}
	result.Attempts = 1

	// Try to compile the translated Python
	_, _, pyErr := mql2go.CompilePythonWithCoverage(result.PythonSource)
	if pyErr == nil {
		result.Status = "success"
		return result, nil
	}
	result.CompileError = pyErr.Error()

	// Retry loop: feed compile error back to LLM
	for attempt := 2; attempt <= maxBridgeRetries; attempt++ {
		result.Attempts = attempt
		b.log.Info("bridge retry",
			zap.Int("attempt", attempt),
			zap.String("compile_error", truncate(pyErr.Error(), 200)))

		retryPrompt := buildBridgeRetryPrompt(mqlSource, result.PythonSource, pyErr.Error())
		uid, uidErr := parseUUID(userID)
		if uidErr != nil {
			return nil, fmt.Errorf("parse user id: %w", uidErr)
		}
		resp, llmErr := b.aiSvc.ChatCompletion(ctx, uid, []systemai.ChatMessage{
			{Role: "system", Content: bridgeSystemPrompt},
			{Role: "user", Content: retryPrompt},
		})
		if llmErr != nil {
			return nil, fmt.Errorf("bridge LLM retry %d: %w", attempt, llmErr)
		}
		result = parseBridgeResponse(resp)
		result.Attempts = attempt

		_, _, pyErr = mql2go.CompilePythonWithCoverage(result.PythonSource)
		if pyErr == nil {
			result.Status = "success"
			result.CompileError = ""
			return result, nil
		}
		result.CompileError = pyErr.Error()
	}

	result.Status = "bridge_failed"
	return result, nil
}

func buildBridgeRetryPrompt(mqlSource, prevPython, compileError string) string {
	var sb strings.Builder
	sb.WriteString("## Previous Attempt (failed to compile)\n```\n")
	sb.WriteString(prevPython)
	sb.WriteString("\n```\n\n")
	sb.WriteString("## Compile Error\n")
	sb.WriteString(compileError)
	sb.WriteString("\n\n## Original MQL Source\n```\n")
	sb.WriteString(mqlSource)
	sb.WriteString("\n```\n\n")
	sb.WriteString("## Task\n")
	sb.WriteString("The previous Python translation failed to compile. Fix the error above and output the corrected Python source code.\n")
	sb.WriteString("Output ONLY the Python source code.")
	return sb.String()
}

func buildBridgeUserPrompt(mqlSource string, coverage *mql2go.CoverageResult) string {
	var sb strings.Builder
	sb.WriteString("## MQL Source Code (has blind spots)\n```\n")
	sb.WriteString(mqlSource)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Coverage Report\n")
	sb.WriteString(fmt.Sprintf("Coverage score: %.0f%%\n", coverage.Score*100))
	if len(coverage.BlindSpots) > 0 {
		sb.WriteString("Blind spots:\n")
		for _, bs := range coverage.BlindSpots {
			sb.WriteString(fmt.Sprintf("- %s (severity: %s, count: %d)\n", bs.Builtin, bs.Severity, bs.Count))
		}
	}

	sb.WriteString("\n## Task\n")
	sb.WriteString("Translate this MQL strategy into a Python subset strategy that avoids the blind spots.\n")
	sb.WriteString("Map MQL trading functions to the Python SDK API shown above.\n")
	sb.WriteString("Output ONLY the Python source code.\n")

	return sb.String()
}

// parseBridgeResponse extracts the Python source from the LLM response.
// The LLM may wrap code in markdown fences despite instructions — strip them.
func parseBridgeResponse(resp string) *BridgeResult {
	python := stripMarkdownFences(resp)
	python = strings.TrimSpace(python)

	return &BridgeResult{
		PythonSource: python,
		Translated:   python != "",
	}
}

// stripMarkdownFences removes ```python ... ``` or ``` ... ``` fences if present.
func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove opening fence line
		idx := strings.Index(s, "\n")
		if idx < 0 {
			return s
		}
		s = s[idx+1:]
	}
	if strings.HasSuffix(s, "```") {
		s = s[:len(s)-3]
	}
	return strings.TrimSpace(s)
}
