package strategy

import (
	"fmt"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go"
	"alphaforge/tools/mql2go/interp"
)

// runDiagnostics runs the diagnostic rule engine and collects coverage/runtime blind spots.
func runDiagnostics(params backtestParams, vmRunner *mql2go.VMRunner, totalTrades int) ([]mql2go.DiagnosticFinding, []mql2go.CoverageBlindSpot, []mql2go.RuntimeBlindSpot) {
	cov := vmRunner.GetCoverage()
	var covBlindSpots []mql2go.CoverageBlindSpot
	if cov != nil {
		for _, bs := range cov.BlindSpots {
			covBlindSpots = append(covBlindSpots, mql2go.CoverageBlindSpot{
				Builtin:  bs,
				Severity: interp.SeverityForBuiltin(bs),
				Source:   "compile",
			})
		}
	}
	var runtimeBlinds []mql2go.RuntimeBlindSpot
	for _, rbs := range vmRunner.GetRuntimeBlindSpots() {
		runtimeBlinds = append(runtimeBlinds, mql2go.RuntimeBlindSpot{
			Builtin:  rbs.Builtin,
			Severity: rbs.Severity,
			Count:    rbs.Count,
		})
	}
	var ruleFindings []mql2go.DiagnosticFinding
	if params.code != "" {
		engine := mql2go.NewRuleEngine()
		ruleFindings = engine.Run(mql2go.RuleInput{
			Source:        params.code,
			HasOnTick:     vmRunner.HasOnTick(),
			Coverage:      cov,
			BlindSpots:    covBlindSpots,
			TotalTrades:   totalTrades,
			RuntimeBlinds: runtimeBlinds,
		})
	}
	return ruleFindings, covBlindSpots, runtimeBlinds
}

func buildBacktestResponse(result *backtest.Result, cfg backtest.Config, params backtestParams, vmRunner *mql2go.VMRunner) (*antv1.ExecuteBacktestResponse, []mql2go.DiagnosticFinding, []mql2go.CoverageBlindSpot, []mql2go.RuntimeBlindSpot) {
	totalTrades := 0
	if result.Metrics != nil {
		totalTrades = int(result.Metrics.TotalTrades)
	}

	ruleFindings, covBlindSpots, runtimeBlinds := runDiagnostics(params, vmRunner, totalTrades)
	cov := vmRunner.GetCoverage()

	resp := &antv1.ExecuteBacktestResponse{
		Success: true,
		Metrics: &antv1.ExecuteBacktestMetrics{
			TotalReturn:   result.Metrics.TotalReturn,
			AnnualReturn:  result.Metrics.AnnualReturn,
			MaxDrawdown:   result.Metrics.MaxDrawdown,
			SharpeRatio:   result.Metrics.SharpeRatio,
			WinRate:       result.Metrics.WinRate,
			ProfitFactor:  result.Metrics.ProfitFactor,
			TotalTrades:   result.Metrics.TotalTrades,
			WinningTrades: result.Metrics.WinningTrades,
			LosingTrades:  result.Metrics.LosingTrades,
		},
	}
	if result.Metrics != nil {
		totalReturn, err := decimal.NewFromString(result.Metrics.TotalReturn)
		if err != nil {
			totalReturn = decimal.Zero
		}
		totalPnl := cfg.InitialCapital.Mul(totalReturn)
		resp.Metrics.TotalPnlAbsolute = totalPnl.String()
	}
	for _, ep := range result.Equity {
		resp.EquityCurve = append(resp.EquityCurve, ep.Equity.String())
		resp.EquityTimesMs = append(resp.EquityTimesMs, ep.Time.UnixMilli())
	}
	for i, t := range result.Trades {
		side := "BUY"
		if t.Side == sdk.SideSell {
			side = "SELL"
		}
		resp.Trades = append(resp.Trades, &antv1.ExecuteBacktestTrade{
			Ticket:     int64(i + 1),
			Side:       side,
			Volume:     t.Volume.String(),
			OpenTsMs:   t.EntryTime.UnixMilli(),
			OpenPrice:  t.EntryPrice.String(),
			CloseTsMs:  t.ExitTime.UnixMilli(),
			ClosePrice: t.ExitPrice.String(),
			Pnl:        t.Profit.String(),
			Commission: t.Commission.String(),
			Reason:     t.Comment,
		})
	}

	resp.ExecutionAssumptions = &antv1.ExecutionAssumptions{
		SimulationMode:   cfg.SimulationMode,
		SignalTiming:     cfg.SignalTiming,
		FillRule:         cfg.FillRule,
		ActualCommission: cfg.Commission.String(),
		ActualSlippage:   cfg.Slippage.String(),
		ActualLeverage:   fmt.Sprintf("%d", cfg.Leverage),
		TradeDirection:   tradeDirectionToString(params.tradeDir),
	}

	resp.Risk = assessRisk(result.Metrics)
	resp.BlindSpots = attachBlindSpots(cov, vmRunner, ruleFindings)

	// MQL-HONESTY-3: Fatal coverage/runtime blind spots (e.g. unimplemented
	// indicator silently returning 0) make the backtest result unreliable.
	// Only SeverityFatal affects correctness — warning/info are advisory.
	for _, bs := range resp.BlindSpots {
		if bs.Severity == interp.SeverityFatal {
			resp.Risk.IsReliable = false
			break
		}
	}

	// P0 invariant: every trade must have Volume > 0.
	// If violated, the backtest result is unreliable (ADR-0028 §4.2 防线 B).
	if bs := checkVolumeInvariant(result.Trades); bs != nil {
		resp.Risk.IsReliable = false
		resp.BlindSpots = append(resp.BlindSpots, bs)
	}

	// P0 invariant: capital conservation — 期末净值 must equal 本金 + ΣProfit − ΣCommission − ΣSwap.
	// If violated, the backtest result is unreliable (ADR-0028 §4.2 防线 B).
	if bs := checkCapitalConservation(result); bs != nil {
		resp.Risk.IsReliable = false
		resp.BlindSpots = append(resp.BlindSpots, bs)
	}

	// P0 invariant: trade field integrity — prices, side, time order.
	// If violated, the backtest result is unreliable (ADR-0028 §4.2 防线 B).
	if bs := checkPricePositive(result); bs != nil {
		resp.Risk.IsReliable = false
		resp.BlindSpots = append(resp.BlindSpots, bs)
	}
	if bs := checkSideValid(result); bs != nil {
		resp.Risk.IsReliable = false
		resp.BlindSpots = append(resp.BlindSpots, bs)
	}
	if bs := checkTimeOrder(result); bs != nil {
		resp.Risk.IsReliable = false
		resp.BlindSpots = append(resp.BlindSpots, bs)
	}

	// ADR-0028 §4.2 statistical-class hints: advisory only, do NOT affect IsReliable.
	resp.BlindSpots = append(resp.BlindSpots, backtest.CheckStatisticalHints(result)...)

	// ADR-0028 §4.1 Defense A: post-parse validation violations.
	// Fatal structural issues (param name collision, no entry point, etc.)
	// make the backtest result unreliable.
	for _, dv := range vmRunner.GetDefenseAViolations() {
		resp.Risk.IsReliable = false
		resp.BlindSpots = append(resp.BlindSpots, &antv1.BlindSpot{
			Id:          "defense_a_" + dv.Rule,
			Category:    "defense_a",
			Severity:    dv.Severity,
			Description: dv.Message,
			Location:    dv.Identifier,
		})
	}

	return resp, ruleFindings, covBlindSpots, runtimeBlinds
}

func attachBlindSpots(cov *mql2go.CoverageReport, vmRunner *mql2go.VMRunner, ruleFindings []mql2go.DiagnosticFinding) []*antv1.BlindSpot {
	var spots []*antv1.BlindSpot
	// MQL-HONESTY-3: Use CoverageResult (pre-classified severity) instead of
	// raw CoverageReport strings. CoverageResult.BlindSpots have correct
	// severity from static analysis (e.g. iXxx → SeverityFatal), while
	// CoverageReport.BlindSpots are raw strings like "unknown function: iXxx"
	// that SeverityForBuiltin can't classify correctly.
	covResult := vmRunner.GetCoverageResult()
	if covResult != nil {
		for _, bs := range covResult.BlindSpots {
			spots = append(spots, &antv1.BlindSpot{
				Id:          bs.Builtin,
				Severity:    bs.Severity,
				Description: bs.Builtin + " is not fully supported",
			})
		}
	} else if cov != nil {
		// Fallback for cached bytecode without CoverageResult.
		for _, bs := range cov.BlindSpots {
			spots = append(spots, &antv1.BlindSpot{
				Id:          bs,
				Severity:    interp.SeverityForBuiltin(bs),
				Description: bs + " is not fully supported",
			})
		}
	}
	for _, rbs := range vmRunner.GetRuntimeBlindSpots() {
		spots = append(spots, &antv1.BlindSpot{
			Id:          rbs.Builtin,
			Severity:    rbs.Severity,
			Description: fmt.Sprintf("%s hit %d time(s) at runtime", rbs.Builtin, rbs.Count),
		})
	}
	for _, f := range ruleFindings {
		spots = append(spots, &antv1.BlindSpot{
			Id:          f.RuleID,
			Severity:    interp.EnglishToChineseSeverity(f.Severity),
			Description: f.Title + ": " + f.Detail,
		})
	}
	return spots
}
