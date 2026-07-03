package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service/systemai"
	"anttrader/tools/mql2go"
)

// Bridge orchestrates the blind-spot bridging Agent loop.
// ADR-0024: MQL blind-spot EA → LLM translates to Python subset → compile_py → VM backtest.
//
// The bridge calls LLM to translate MQL (with blind spots) into Python subset code,
// then verifies via compile + coverage check + VM backtest (BacktestValidator).
type Bridge struct {
	aiSvc *systemai.Service
	log   *zap.Logger
	cache *LLCache
}

// NewBridge creates the blind-spot bridging orchestrator.
func NewBridge(aiSvc *systemai.Service, log *zap.Logger, cache *LLCache) *Bridge {
	return &Bridge{aiSvc: aiSvc, log: log, cache: cache}
}

// BridgeResult holds the output of a bridge translation attempt.
type BridgeResult struct {
	PythonSource  string
	CompileError  string
	Status        string // "success" | "bridge_failed"
	Attempts      int
	BridgedCov    *mql2go.CoverageResult
	BacktestError string
}

// translateBlindSpot calls LLM to translate MQL with blind spots into Python subset.
// ADR-0024: "盲区桥接 prompt + Agent 循环编排"
func (b *Bridge) translateBlindSpot(
	ctx context.Context,
	uid uuid.UUID,
	mqlSource string,
	coverage *mql2go.CoverageResult,
	profile *antv1.StrategyProfile,
) (*BridgeResult, error) {
	userPrompt := buildBridgeUserPrompt(mqlSource, coverage, profile)

	// Check cache
	if b.cache != nil {
		if cached, ok := b.cache.Get(mqlSource, userPrompt); ok {
			return parseBridgeResponse(cached), nil
		}
	}

	// LLM call timeout (per ADR §5.3: 30s)
	llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := b.aiSvc.ChatCompletion(llmCtx, uid, []systemai.ChatMessage{
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

// BacktestValidator runs a backtest on the already-compiled Python runner to verify runtime correctness.
// The runner is produced by verifyBridgeResult via CompilePythonWithCoverage — no recompilation needed.
// Returns nil if the backtest succeeds, or an error describing the failure.
type BacktestValidator func(runner *mql2go.VMRunner) error

// TranslateWithRetry runs the bridge Agent loop: LLM translate → compile → coverage check → backtest → error feedback → retry.
// ADR-0024 §5.4: max 3 retries with compile error feedback to LLM.
// Also runs backtest to verify runtime correctness.
// Parses UUID once outside the retry loop.
func (b *Bridge) TranslateWithRetry(
	ctx context.Context,
	userID string,
	mqlSource string,
	coverage *mql2go.CoverageResult,
	profile *antv1.StrategyProfile,
	validateBacktest BacktestValidator,
) (*BridgeResult, error) {
	// Parse UUID once outside the retry loop.
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	result, err := b.translateBlindSpot(ctx, uid, mqlSource, coverage, profile)
	if err != nil {
		return nil, err
	}
	result.Attempts = 1

	// Verify: compile + coverage + backtest
	if ok := b.verifyBridgeResult(result, coverage, validateBacktest); ok {
		return result, nil
	}

	// Retry loop: feed compile/backtest error back to LLM
	for attempt := 2; attempt <= maxBridgeRetries; attempt++ {
		result.Attempts = attempt
		errorMsg := result.CompileError
		if result.BacktestError != "" {
			errorMsg = result.BacktestError
		}
		b.log.Info("bridge retry",
			zap.Int("attempt", attempt),
			zap.String("error", truncate(errorMsg, 200)))

		retryPrompt := buildBridgeRetryPrompt(mqlSource, result.PythonSource, errorMsg, profile)
		// LLM call timeout
		llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		resp, llmErr := b.aiSvc.ChatCompletion(llmCtx, uid, []systemai.ChatMessage{
			{Role: "system", Content: bridgeSystemPrompt},
			{Role: "user", Content: retryPrompt},
		})
		cancel()
		if llmErr != nil {
			return nil, fmt.Errorf("bridge LLM retry %d: %w", attempt, llmErr)
		}
		result = parseBridgeResponse(resp)
		result.Attempts = attempt

		if b.verifyBridgeResult(result, coverage, validateBacktest) {
			return result, nil
		}
	}

	result.Status = "bridge_failed"
	return result, nil
}

// verifyBridgeResult checks compilation, coverage improvement, and backtest success.
// Returns true if all checks pass, false otherwise (with error details set on result).
func (b *Bridge) verifyBridgeResult(
	result *BridgeResult,
	origCoverage *mql2go.CoverageResult,
	validateBacktest BacktestValidator,
) bool {
	result.CompileError = ""
	result.BacktestError = ""
	result.Status = ""
	result.BridgedCov = nil

	// Step 1: Compile (single compilation — runner reused for backtest)
	pyRunner, pyCov, pyErr := mql2go.CompilePythonWithCoverage(result.PythonSource)
	if pyErr != nil {
		result.CompileError = pyErr.Error()
		return false
	}
	result.BridgedCov = pyCov

	// Step 2: Coverage improvement check (ADR §5.4)
	if pyCov.Score < origCoverage.Score {
		result.CompileError = fmt.Sprintf(
			"coverage regression: %.0f%% → %.0f%% (must not decrease)",
			origCoverage.Score*100, pyCov.Score*100)
		return false
	}

	// Step 3: Backtest validation — reuse compiled runner, no recompilation
	if validateBacktest != nil {
		if btErr := validateBacktest(pyRunner); btErr != nil {
			result.BacktestError = btErr.Error()
			return false
		}
	}

	result.Status = "success"
	return true
}

func buildBridgeRetryPrompt(mqlSource, prevPython, errorMsg string, profile *antv1.StrategyProfile) string {
	var sb strings.Builder
	sb.WriteString("## Previous Attempt (failed validation)\n```\n")
	sb.WriteString(prevPython)
	sb.WriteString("\n```\n\n")
	sb.WriteString("## Error\n")
	sb.WriteString(errorMsg)
	if profile != nil {
		sb.WriteString("\n\n## Strategy Profile (for context)\n")
		sb.WriteString(fmt.Sprintf("Type: %s\n", profile.StrategyType))
		sb.WriteString(fmt.Sprintf("Indicators: %s\n", strings.Join(profile.IndicatorsUsed, ", ")))
		sb.WriteString(fmt.Sprintf("Entry: %s\n", profile.EntryLogic))
		sb.WriteString(fmt.Sprintf("Exit: %s\n", profile.ExitLogic))
	}
	sb.WriteString("\n\n## Original MQL Source\n```\n")
	sb.WriteString(mqlSource)
	sb.WriteString("\n```\n\n")
	sb.WriteString("## Task\n")
	sb.WriteString("The previous Python translation failed validation (compile error, coverage regression, or backtest failure). Fix the issue above and output the corrected Python source code.\n")
	sb.WriteString("Output ONLY the Python source code.")
	return sb.String()
}

func buildBridgeUserPrompt(mqlSource string, coverage *mql2go.CoverageResult, profile *antv1.StrategyProfile) string {
	var sb strings.Builder
	sb.WriteString("## MQL Source Code (has blind spots)\n```\n")
	sb.WriteString(mqlSource)
	sb.WriteString("\n```\n\n")

	writeProfileToPrompt(&sb, profile, "## Strategy Profile\n")
	if profile != nil {
		sb.WriteString("\n")
	}

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
	sb.WriteString("Use the strategy profile to guide semantic translation of custom indicators.\n")
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
	}
}
