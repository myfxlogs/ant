package strategy

import (
	"context"

	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/ai"
	"alphaforge/internal/marketplace"
	"alphaforge/internal/repository"
	"alphaforge/tools/mql2go"
)

// sendAutoGateUpdate computes 7-gate results and marketplace quality preview
// for a successful backtest run with auto_gate=true, then sends them as
// BacktestRunUpdate messages on the same stream (§15.7 E1).
// Results are persisted to gate_evaluations table for reconnection recovery.
func sendAutoGateUpdate(ctx context.Context, s *StrategyExecutionServer, run *repository.BacktestRun, stream *connect.ServerStream[antv1.BacktestRunUpdate]) {
	var qualityPreview *antv1.MarketplaceQualityPreview
	var pipelineResult *ai.PipelineResult

	dailyReturns := ai.EquityCurveToDailyReturns(run.ProtoResponse)
	if len(dailyReturns) >= 10 {
		input := buildPipelineInput(ctx, s, run, dailyReturns)
		result := ai.Pipeline(input)
		pipelineResult = &result
	}

	if s.qualityValidator != nil && len(run.BacktestSnapshot) > 0 {
		strategyID := ""
		if run.TemplateID != nil {
			strategyID = run.TemplateID.String()
		}
		violations, err := s.qualityValidator.ValidateBacktestQuality(ctx, run.BacktestSnapshot, strategyID)
		if err != nil {
			s.log.Warn("auto_gate: marketplace quality preview failed", zap.Error(err))
		} else {
			qualityPreview = ViolationsToPreview(violations)
		}
	}

	if pipelineResult == nil && qualityPreview == nil {
		return
	}

	bp := parseBacktestResult(run.ProtoResponse)

	// Send one update per gate result (matching GateEvaluationUpdate streaming design).
	if pipelineResult != nil {
		for _, g := range pipelineResult.Gates {
			if err := stream.Send(&antv1.BacktestRunUpdate{
				Run:                  toProtoBacktestRun(run),
				Metrics:              bp.Metrics,
				EquityCurve:          bp.EquityCurve,
				Risk:                 bp.Risk,
				ExecutionAssumptions: bp.ExecutionAssumptions,
				BlindSpots:           bp.BlindSpots,
				GateUpdate: &antv1.GateEvaluationUpdate{
					Gate: GateResultToProto(g),
				},
			}); err != nil {
				s.log.Warn("auto_gate: failed to send gate result", zap.Error(err))
				return
			}
		}
	}

	// Build final summary update.
	gateSummary := BuildGateSummaryProto(pipelineResult)

	// #4: Unify publishability — requires BOTH 7-gate pass AND no marketplace violations.
	if qualityPreview != nil && pipelineResult != nil && !pipelineResult.Passed {
		qualityPreview.Publishable = false
	}

	// #1: Persist to DB for reconnection recovery.
	persistGateEvaluation(ctx, s, run, gateSummary, pipelineResult, qualityPreview)

	if err := stream.Send(&antv1.BacktestRunUpdate{
		Run:                  toProtoBacktestRun(run),
		Metrics:              bp.Metrics,
		EquityCurve:          bp.EquityCurve,
		Risk:                 bp.Risk,
		ExecutionAssumptions: bp.ExecutionAssumptions,
		BlindSpots:           bp.BlindSpots,
		GateUpdate:           gateSummary,
		QualityPreview:       qualityPreview,
	}); err != nil {
		s.log.Warn("auto_gate: failed to send gate/quality update", zap.Error(err))
	}
}

func ViolationsToPreview(violations []marketplace.QualityViolation) *antv1.MarketplaceQualityPreview {
	preview := &antv1.MarketplaceQualityPreview{
		Publishable: len(violations) == 0,
	}
	for _, v := range violations {
		preview.Violations = append(preview.Violations, &antv1.QualityViolation{
			Metric:    v.Metric,
			Actual:    v.Actual,
			Threshold: v.Threshold,
		})
	}
	return preview
}

// GateResultToProto converts an ai.GateStatus to its proto representation.
func GateResultToProto(g ai.GateStatus) *antv1.GateResult {
	return &antv1.GateResult{
		Gate:       string(g.Gate),
		Passed:     g.Passed,
		Reason:     g.Reason,
		Score:      g.Score,
		DurationMs: g.Duration,
		Skipped:    g.Skipped,
	}
}

// BuildGateSummaryProto builds a GateEvaluationUpdate containing the pipeline summary.
func BuildGateSummaryProto(result *ai.PipelineResult) *antv1.GateEvaluationUpdate {
	if result == nil {
		return nil
	}
	return &antv1.GateEvaluationUpdate{
		Completed: &antv1.GatePipelineSummary{
			Passed:          result.Passed,
			FirstFail:       string(result.FirstFail),
			Summary:         result.Summary,
			TotalDurationMs: result.TotalDuration,
		},
	}
}

// BuildGateListProto builds a GateResultList from individual gate results.
func BuildGateListProto(result *ai.PipelineResult) *antv1.GateResultList {
	if result == nil || len(result.Gates) == 0 {
		return nil
	}
	list := &antv1.GateResultList{}
	for _, g := range result.Gates {
		list.Gates = append(list.Gates, GateResultToProto(g))
	}
	return list
}

// buildPipelineInput constructs a PipelineInput with all available data:
// - DailyReturns from the backtest equity curve
// - NumAttempts from historical backtest count (#5)
// - NewSignals from backtest trades (#3)
// - ExistingSignals from live strategy runs (#3)
func buildPipelineInput(ctx context.Context, s *StrategyExecutionServer, run *repository.BacktestRun, dailyReturns []float64) ai.PipelineInput {
	return BuildPipelineInputFromRepo(ctx, s.gateEvalRepo, run, dailyReturns)
}

// buildPipelineInputFromRepo is the repo-level variant of buildPipelineInput.
// Shared by sendAutoGateUpdate (via buildPipelineInput) and onBacktestComplete (in handlers_strategy.go).
func BuildPipelineInputFromRepo(ctx context.Context, repo *repository.GateEvaluationRepository, run *repository.BacktestRun, dailyReturns []float64) ai.PipelineInput {
	input := ai.PipelineInput{
		DailyReturns: dailyReturns,
		NumAttempts:  1,
	}

	// Detect lookahead violations from strategy source via IR analysis.
	// This replaces the legacy regex-based DSL scanner for MQL/Python strategies.
	if run.StrategyCode != nil && *run.StrategyCode != "" {
		rawViolations := mql2go.DetectLookaheadFromSource(*run.StrategyCode)
		if len(rawViolations) > 0 {
			input.LookaheadViolations = make([]ai.LookaheadViolation, len(rawViolations))
			for i, v := range rawViolations {
				input.LookaheadViolations[i] = ai.LookaheadViolation{
					Function:  v.Function,
					ShiftExpr: v.ShiftExpr,
					ShiftVal:  v.ShiftVal,
					IsLiteral: v.IsLiteral,
					Severity:  v.Severity,
					Message:   v.Message,
				}
			}
		}
	}

	if repo != nil {
		if count, err := repo.CountBacktestAttempts(ctx, run.UserID, run.TemplateID); err == nil && count > 0 {
			input.NumAttempts = count
		}
	}

	input.NewSignals = extractSignalsFromTrades(run.ProtoResponse)

	if repo != nil {
		if existing, err := repo.GetExistingSignals(ctx, run.UserID); err == nil && len(existing) > 0 {
			input.ExistingSignals = make(map[string][]ai.SignalDirection, len(existing))
			for sid, signals := range existing {
				dirs := make([]ai.SignalDirection, len(signals))
				for i, sig := range signals {
					dirs[i] = ai.SignalDirection{Timestamp: sig.Timestamp, Direction: sig.Direction}
				}
				input.ExistingSignals[sid] = dirs
			}
		}
	}

	return input
}

// extractSignalsFromTrades parses ExecuteBacktestResponse proto and converts
// trades to SignalDirection entries (buy=+1, sell=-1, timestamp=open_ts_ms).
func extractSignalsFromTrades(protoResp []byte) []ai.SignalDirection {
	if len(protoResp) == 0 {
		return nil
	}
	var resp antv1.ExecuteBacktestResponse
	if err := proto.Unmarshal(protoResp, &resp); err != nil {
		return nil
	}
	signals := make([]ai.SignalDirection, 0, len(resp.Trades))
	for _, tr := range resp.Trades {
		dir := 1.0
		if tr.Side == "sell" {
			dir = -1.0
		}
		signals = append(signals, ai.SignalDirection{Timestamp: tr.OpenTsMs, Direction: dir})
	}
	return signals
}

// persistGateEvaluation saves gate results + quality preview to DB (#1).
func persistGateEvaluation(ctx context.Context, s *StrategyExecutionServer, run *repository.BacktestRun, gateSummary *antv1.GateEvaluationUpdate, pipelineResult *ai.PipelineResult, qualityPreview *antv1.MarketplaceQualityPreview) {
	if s.gateEvalRepo == nil {
		return
	}
	var gateBytes []byte
	if gateSummary != nil {
		if b, err := proto.Marshal(gateSummary); err == nil {
			gateBytes = b
		}
	}
	var gateResultsBytes []byte
	if gateList := BuildGateListProto(pipelineResult); gateList != nil {
		if b, err := proto.Marshal(gateList); err == nil {
			gateResultsBytes = b
		}
	}
	var qualityBytes []byte
	if qualityPreview != nil {
		if b, err := proto.Marshal(qualityPreview); err == nil {
			qualityBytes = b
		}
	}
	passed := gateSummary != nil && gateSummary.Completed != nil && gateSummary.Completed.Passed
	firstFail := ""
	summary := ""
	if gateSummary != nil && gateSummary.Completed != nil {
		firstFail = gateSummary.Completed.FirstFail
		summary = gateSummary.Completed.Summary
	}
	publishable := qualityPreview != nil && qualityPreview.Publishable
	if err := s.gateEvalRepo.Upsert(ctx, run.UserID, run.ID, gateBytes, gateResultsBytes, qualityBytes, passed, firstFail, summary, publishable); err != nil {
		s.log.Warn("auto_gate: failed to persist gate evaluation", zap.Error(err))
	}
}

// restoreGateEvaluation sends persisted gate results on stream reconnection (#1).
// Falls back to fresh computation if no persisted result exists.
func restoreGateEvaluation(ctx context.Context, s *StrategyExecutionServer, run *repository.BacktestRun, stream *connect.ServerStream[antv1.BacktestRunUpdate]) {
	if s.gateEvalRepo == nil {
		sendAutoGateUpdate(ctx, s, run, stream)
		return
	}
	ge, err := s.gateEvalRepo.GetByRunID(ctx, run.ID)
	if err != nil || ge == nil {
		sendAutoGateUpdate(ctx, s, run, stream)
		return
	}

	bp := parseBacktestResult(run.ProtoResponse)

	var gateSummary *antv1.GateEvaluationUpdate
	if len(ge.GateResult) > 0 {
		gateSummary = &antv1.GateEvaluationUpdate{}
		if err := proto.Unmarshal(ge.GateResult, gateSummary); err != nil {
			gateSummary = nil
		}
	}

	var qualityPreview *antv1.MarketplaceQualityPreview
	if len(ge.QualityPreview) > 0 {
		qualityPreview = &antv1.MarketplaceQualityPreview{}
		if err := proto.Unmarshal(ge.QualityPreview, qualityPreview); err != nil {
			qualityPreview = nil
		}
	}

	// If quality preview not persisted (e.g. older runs before quality preview
	// persistence was added), compute it on the fly from the backtest snapshot.
	if qualityPreview == nil && s.qualityValidator != nil && len(run.BacktestSnapshot) > 0 {
		strategyID := ""
		if run.TemplateID != nil {
			strategyID = run.TemplateID.String()
		}
		violations, err := s.qualityValidator.ValidateBacktestQuality(ctx, run.BacktestSnapshot, strategyID)
		if err != nil {
			s.log.Warn("auto_gate: marketplace quality preview failed on restore", zap.Error(err))
		} else {
			qualityPreview = ViolationsToPreview(violations)
			// Unify publishability with gate results.
			if gateSummary != nil && gateSummary.Completed != nil && !gateSummary.Completed.Passed {
				qualityPreview.Publishable = false
			}
		}
	}

	if gateSummary == nil && qualityPreview == nil {
		return
	}

	// Replay individual gate results first (matching live streaming behavior).
	if len(ge.GateResults) > 0 {
		gateList := &antv1.GateResultList{}
		if err := proto.Unmarshal(ge.GateResults, gateList); err == nil {
			for _, g := range gateList.Gates {
				if err := stream.Send(&antv1.BacktestRunUpdate{
					Run:                  toProtoBacktestRun(run),
					Metrics:              bp.Metrics,
					EquityCurve:          bp.EquityCurve,
					Risk:                 bp.Risk,
					ExecutionAssumptions: bp.ExecutionAssumptions,
					BlindSpots:           bp.BlindSpots,
					GateUpdate: &antv1.GateEvaluationUpdate{
						Gate: g,
					},
				}); err != nil {
					s.log.Warn("auto_gate: failed to replay gate result", zap.Error(err))
					return
				}
			}
		}
	}

	if err := stream.Send(&antv1.BacktestRunUpdate{
		Run:                  toProtoBacktestRun(run),
		Metrics:              bp.Metrics,
		EquityCurve:          bp.EquityCurve,
		Risk:                 bp.Risk,
		ExecutionAssumptions: bp.ExecutionAssumptions,
		BlindSpots:           bp.BlindSpots,
		GateUpdate:           gateSummary,
		QualityPreview:       qualityPreview,
	}); err != nil {
		s.log.Warn("auto_gate: failed to restore gate evaluation", zap.Error(err))
	}
}
