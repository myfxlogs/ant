package agent

import (
	_ "embed"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/service/systemai"
	"alphaforge/tools/mql2go"
)

//go:embed prompts/bridge_user.prompt
var bridgeUserPromptTmpl string

//go:embed prompts/bridge_retry_user.prompt
var bridgeRetryUserPromptTmpl string

// Bridge orchestrates the blind-spot bridging Agent loop.
// ADR-0024: MQL blind-spot EA → LLM translates to Python subset → compile_py → VM backtest.
//
// The bridge calls LLM to translate MQL (with blind spots) into Python subset code,
// then verifies via compile + coverage check + VM backtest (BacktestValidator).
type Bridge struct {
	aiSvc *systemai.Service
	log   *zap.Logger
}

// NewBridge creates the blind-spot bridging orchestrator.
func NewBridge(aiSvc *systemai.Service, log *zap.Logger, _ *LLCache) *Bridge {
	return &Bridge{aiSvc: aiSvc, log: log}
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

	// Unified retry loop: initial call + retries in one path.
	// ADR-0024 §5.4: max 3 retries total (3 attempts = 1 initial + 2 retries).
	// LLM transient failures are also retried, not fatal.
	var result *BridgeResult
	var lastCompileError string
	for attempt := 1; attempt <= maxBridgeRetries; attempt++ {
		// Build prompt: original for attempt 1, error-feedback for retries
		var userPrompt string
		if attempt == 1 {
			userPrompt = buildBridgeUserPrompt(mqlSource, coverage, profile)
		} else {
			userPrompt = buildBridgeRetryPrompt(mqlSource, result.PythonSource, lastCompileError, profile)
		}

		// LLM call with timeout (30s per ADR §5.3)
		llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		resp, llmErr := b.aiSvc.ChatCompletion(llmCtx, uid, []systemai.ChatMessage{
			{Role: "system", Content: bridgeSystemPrompt},
			{Role: "user", Content: userPrompt},
		})
		cancel()
		if llmErr != nil {
			b.log.Warn("bridge LLM call failed, retrying",
				zap.Int("attempt", attempt),
				zap.Error(llmErr))
			continue // transient LLM failure → next attempt, not fatal
		}

		result = parseBridgeResponse(resp)
		result.Attempts = attempt
		lastCompileError = result.CompileError
		if result.BacktestError != "" {
			lastCompileError = result.BacktestError
		}

		// Verify: compile + coverage + backtest
		if b.verifyBridgeResult(result, coverage, validateBacktest) {
			return result, nil
		}

		b.log.Info("bridge retry",
			zap.Int("attempt", attempt),
			zap.String("error", truncate(lastCompileError, 200)))
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
	data := promptData{
		MQLSource:  sanitizeInput(mqlSource),
		PrevPython: sanitizeInput(prevPython),
		ErrorMsg:   errorMsg,
	}
	if profile != nil {
		var sb strings.Builder
		sb.WriteString("## Strategy Profile (for context)\n")
		fmt.Fprintf(&sb, "Type: %s\n", profile.StrategyType)
		fmt.Fprintf(&sb, "Indicators: %s\n", strings.Join(profile.IndicatorsUsed, ", "))
		fmt.Fprintf(&sb, "Entry: %s\n", profile.EntryLogic)
		fmt.Fprintf(&sb, "Exit: %s\n", profile.ExitLogic)
		data.ProfileBlock = sb.String()
	}
	userPrompt, err := renderPrompt("bridge_retry_user", bridgeRetryUserPromptTmpl, data)
	if err != nil {
		return fallbackBridgeRetryPrompt(mqlSource, prevPython, errorMsg, profile)
	}
	return userPrompt
}

func buildBridgeUserPrompt(mqlSource string, coverage *mql2go.CoverageResult, profile *antv1.StrategyProfile) string {
	data := promptData{
		MQLSource: sanitizeInput(mqlSource),
	}
	if profile != nil {
		var sb strings.Builder
		writeProfileToPrompt(&sb, profile, "## Strategy Profile\n")
		data.ProfileBlock = sb.String()
	}
	var covSB strings.Builder
	fmt.Fprintf(&covSB, "Coverage score: %.0f%%\n", coverage.Score*100)
	if len(coverage.BlindSpots) > 0 {
		covSB.WriteString("Blind spots:\n")
		for _, bs := range coverage.BlindSpots {
			fmt.Fprintf(&covSB, "- %s (severity: %s, count: %d)\n", bs.Builtin, bs.Severity, bs.Count)
		}
	}
	data.CoverageBlock = covSB.String()
	userPrompt, err := renderPrompt("bridge_user", bridgeUserPromptTmpl, data)
	if err != nil {
		return fallbackBridgeUserPrompt(mqlSource, coverage, profile)
	}
	return userPrompt
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

func fallbackBridgeRetryPrompt(mqlSource, prevPython, errorMsg string, profile *antv1.StrategyProfile) string {
	var sb strings.Builder
	sb.WriteString("## Previous Attempt (failed validation)\n```\n")
	sb.WriteString(prevPython)
	sb.WriteString("\n```\n\n")
	sb.WriteString("## Error\n")
	sb.WriteString(errorMsg)
	if profile != nil {
		sb.WriteString("\n\n## Strategy Profile (for context)\n")
		fmt.Fprintf(&sb, "Type: %s\n", profile.StrategyType)
		fmt.Fprintf(&sb, "Indicators: %s\n", strings.Join(profile.IndicatorsUsed, ", "))
		fmt.Fprintf(&sb, "Entry: %s\n", profile.EntryLogic)
		fmt.Fprintf(&sb, "Exit: %s\n", profile.ExitLogic)
	}
	sb.WriteString("\n\n## Original MQL Source\n```\n")
	sb.WriteString(mqlSource)
	sb.WriteString("\n```\n\n")
	sb.WriteString("## Task\n")
	sb.WriteString("The previous Python translation failed validation (compile error, coverage regression, or backtest failure). Fix the issue above and output the corrected Python source code.\n")
	sb.WriteString("Output ONLY the Python source code.")
	return sb.String()
}

func fallbackBridgeUserPrompt(mqlSource string, coverage *mql2go.CoverageResult, profile *antv1.StrategyProfile) string {
	var sb strings.Builder
	sb.WriteString("## MQL Source Code (has blind spots)\n```\n")
	sb.WriteString(mqlSource)
	sb.WriteString("\n```\n\n")
	writeProfileToPrompt(&sb, profile, "## Strategy Profile\n")
	if profile != nil {
		sb.WriteString("\n")
	}
	sb.WriteString("## Coverage Report\n")
	fmt.Fprintf(&sb, "Coverage score: %.0f%%\n", coverage.Score*100)
	if len(coverage.BlindSpots) > 0 {
		sb.WriteString("Blind spots:\n")
		for _, bs := range coverage.BlindSpots {
			fmt.Fprintf(&sb, "- %s (severity: %s, count: %d)\n", bs.Builtin, bs.Severity, bs.Count)
		}
	}
	sb.WriteString("\n## Task\n")
	sb.WriteString("Translate this MQL strategy into a Python subset strategy that avoids the blind spots.\n")
	sb.WriteString("Use the strategy profile to guide semantic translation of custom indicators.\n")
	sb.WriteString("Map MQL trading functions to the Python SDK API shown above.\n")
	sb.WriteString("Output ONLY the Python source code.\n")
	return sb.String()
}
