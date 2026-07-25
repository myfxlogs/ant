package agent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/tools/mql2go"
)

// backtestPipelineResult holds outputs from the backtest+LLM+bridge pipeline (Steps 2-6).
type backtestPipelineResult struct {
	profile       *antv1.StrategyProfile
	analysis      *antv1.BacktestAnalysis
	semanticDiff  *antv1.SemanticDiff
	btProto       *antv1.AgentBacktestResult
	bridgeStatus  string
	bridgedPython string
	bridgeErr     string
}

// runBacktestPipeline executes Steps 2-6: fetch bars → backtest → profile → analysis → bridge.
// Errors from individual steps are logged but do not abort the pipeline; only fetchBars
// failures are returned as an error (caller treats it as non-fatal — strategy is already persisted).
func (s *GatewayServer) runBacktestPipeline(
	ctx context.Context,
	runner *mql2go.VMRunner,
	btCfg *antv1.AgentBacktestConfig,
	params map[string]string,
	userID string,
	strategyID string,
	sourceCode string,
	coverage *mql2go.CoverageResult,
	language string,
) (*backtestPipelineResult, error) {
	// Step 2: Fetch market data bars
	bars, err := s.fetchBars(ctx, btCfg)
	if err != nil {
		return nil, fmt.Errorf("fetch market data: %w", err)
	}
	if len(bars) < 2 {
		return nil, fmt.Errorf("insufficient market data: %d bars (need ≥2)", len(bars))
	}

	result := &backtestPipelineResult{bridgeStatus: "not_attempted"}

	// Step 3: Run backtest via VM + SimBroker
	btResult, btErr := runVMBacktest(ctx, runner, btCfg, bars, params)
	if btErr != nil {
		result.btProto = &antv1.AgentBacktestResult{Success: false, Error: btErr.Error()}
	} else {
		result.btProto = buildBacktestResultProto(btResult)
	}

	// PostBacktest hook (ADR-0025 §8)
	if s.hooks != nil && s.hooks.HasHandlers(HookPostBacktest) {
		uid, _ := uuid.Parse(userID)
		s.hooks.Fire(ctx, &HookContext{
			Event:          HookPostBacktest,
			UserID:         uid,
			StrategyID:     strategyID,
			BacktestResult: result.btProto,
		})
	}

	// Step 4: Generate strategy profile (LLM injection point [1])
	profile, profErr := s.profiler.GenerateProfile(ctx, userID, sourceCode, coverage)
	if profErr != nil {
		s.log.Warn("AgentGateway: profile generation failed", zap.Error(profErr))
	}
	result.profile = profile

	// Step 5: Generate backtest analysis (LLM injection point [4])
	if btErr == nil {
		analysis, analysisErr := s.interpreter.AnalyzeBacktest(ctx, userID, result.btProto, profile)
		if analysisErr != nil {
			s.log.Warn("AgentGateway: analysis generation failed", zap.Error(analysisErr))
		}
		result.analysis = analysis
	}

	// Step 6: Blind-spot bridge (ADR-0024)
	if language != "python" && coverage.Score < 1.0 && len(coverage.BlindSpots) > 0 {
		validateBacktest := func(pyRunner *mql2go.VMRunner) error {
			_, btErr := runVMBacktest(ctx, pyRunner, btCfg, bars, params)
			return btErr
		}

		bridgeResult, bridgeErr := s.bridge.TranslateWithRetry(ctx, userID, sourceCode, coverage, profile, validateBacktest)
		if bridgeErr != nil {
			s.log.Warn("AgentGateway: bridge translation failed", zap.Error(bridgeErr))
			result.bridgeStatus = "bridge_failed"
			result.bridgeErr = bridgeErr.Error()
			result.semanticDiff = buildBridgeFailureReport(coverage)
		} else {
			result.bridgeStatus = bridgeResult.Status
			result.bridgeErr = bridgeResult.CompileError
			if bridgeResult.BacktestError != "" {
				result.bridgeErr = bridgeResult.BacktestError
			}
			if bridgeResult.Status == "success" {
				result.bridgedPython = bridgeResult.PythonSource
				pyCov := bridgeResult.BridgedCov
				s.log.Info("AgentGateway: bridge translation successful",
					zap.Int("attempts", bridgeResult.Attempts),
					zap.Float64("original_coverage", coverage.Score),
					zap.Float64("bridged_coverage", pyCov.Score))
				if pyCov != nil {
					result.semanticDiff = &antv1.SemanticDiff{
						Changes:       buildBridgeChanges(coverage, pyCov),
						EffectSummary: fmt.Sprintf("覆盖率 %.0f%% → %.0f%%", coverage.Score*100, pyCov.Score*100),
					}
				}
			} else {
				s.log.Warn("AgentGateway: bridge failed after retries",
					zap.Int("attempts", bridgeResult.Attempts),
					zap.String("last_error", truncate(result.bridgeErr, 200)))
				result.semanticDiff = buildBridgeFailureReport(coverage)
			}
		}
	}

	return result, nil
}
