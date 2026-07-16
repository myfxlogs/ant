package oms

import (
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/internal/costsvc"
)

func decF(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

func TestFillModel_Buy_NetHigherThanGross(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)

	result := fm.Compute(decF(1.0850), costsvc.EstimateParams{
		Side: "buy", Lots: decF(1.0), Price: decF(1.0850), ContractSize: decF(100000), HoldingDays: decimal.Zero,
	}, false)

	if !result.NetFillPrice.GreaterThan(result.GrossPrice) {
		t.Fatalf("buy: net fill %s should be > gross %s", result.NetFillPrice.String(), result.GrossPrice.String())
	}
	if !result.Commission.GreaterThan(decimal.Zero) {
		t.Fatalf("commission should be > 0, got %s", result.Commission.String())
	}
}

func TestFillModel_Sell_NetLowerThanGross(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)

	result := fm.Compute(decF(1.0850), costsvc.EstimateParams{
		Side: "sell", Lots: decF(1.0), Price: decF(1.0850), ContractSize: decF(100000), HoldingDays: decimal.Zero,
	}, false)

	if !result.NetFillPrice.LessThan(result.GrossPrice) {
		t.Fatalf("sell: net fill %s should be < gross %s", result.NetFillPrice.String(), result.GrossPrice.String())
	}
}

func TestFillModel_Backtest_EnforcesNonZeroCosts(t *testing.T) {
	t.Parallel()
	cm := &costsvc.CostModel{
		Symbol: "TEST", PipSize: decF(0.0001), PipValue: decF(10),
		SpreadPips: decimal.Zero, CommissionPerLot: decimal.Zero, CommissionBps: decimal.Zero, SlippageBps: decimal.Zero,
	}
	fm := NewFillModel(cm)

	result := fm.Compute(decF(1.0850), costsvc.EstimateParams{
		Side: "buy", Lots: decF(1.0), Price: decF(1.0850), ContractSize: decF(100000), HoldingDays: decimal.Zero,
	}, true) // isBacktest=true

	if !result.SpreadCost.GreaterThan(decimal.Zero) {
		t.Fatalf("backtest should enforce non-zero spread, got %s", result.SpreadCost.String())
	}
	if !result.SlippageCost.GreaterThan(decimal.Zero) {
		t.Fatalf("backtest should enforce non-zero slippage, got %s", result.SlippageCost.String())
	}
	if !result.Commission.GreaterThan(decimal.Zero) {
		t.Fatalf("backtest should enforce non-zero commission, got %s", result.Commission.String())
	}
	if !result.NetFillPrice.GreaterThan(result.GrossPrice) {
		t.Fatalf("backtest buy: net %s > gross %s", result.NetFillPrice.String(), result.GrossPrice.String())
	}
	t.Logf("Backtest fill: gross=%s net=%s costs: spread=%s comm=%s slip=%s",
		result.GrossPrice.String(), result.NetFillPrice.String(), result.SpreadCost.String(), result.Commission.String(), result.SlippageCost.String())
}

func TestFillModel_ComputeNet(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)

	net := fm.ComputeNet(decF(1.0850), costsvc.EstimateParams{
		Side: "buy", Lots: decF(1.0), Price: decF(1.0850), ContractSize: decF(100000), HoldingDays: decimal.Zero,
	}, false)

	if !net.GreaterThan(decF(1.0850)) {
		t.Fatalf("net should be > gross for buy")
	}
}

func TestFillModel_ZeroVolume(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)

	result := fm.Compute(decF(1.0850), costsvc.EstimateParams{
		Side: "buy", Lots: decimal.Zero, Price: decF(1.0850), ContractSize: decF(100000), HoldingDays: decimal.Zero,
	}, false)

	if result.NetFillPrice.Sub(decF(1.0850)).Abs().GreaterThanOrEqual(decF(0.0001)) {
		t.Fatalf("zero volume: net should equal gross, got %s", result.NetFillPrice.String())
	}
}
