package strategy

import (
	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/backtest"
)

// Thin wrappers delegating to the shared backtest package (ADR-0028 Defense Line B).
// REUSE: backtest.CheckVolumeInvariant @ strategy/backtest/invariants.go

func checkVolumeInvariant(trades []backtest.Trade) *antv1.BlindSpot {
	return backtest.CheckVolumeInvariant(trades)
}

func checkPricePositive(result *backtest.Result) *antv1.BlindSpot {
	return backtest.CheckPricePositive(result)
}

func checkSideValid(result *backtest.Result) *antv1.BlindSpot {
	return backtest.CheckSideValid(result)
}

func checkTimeOrder(result *backtest.Result) *antv1.BlindSpot {
	return backtest.CheckTimeOrder(result)
}

func checkCapitalConservation(result *backtest.Result) *antv1.BlindSpot {
	return backtest.CheckCapitalConservation(result)
}
