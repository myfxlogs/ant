package oms

import (
	"math"
	"testing"

	"anttrader/internal/costsvc"
)

func TestFillModel_Buy_NetHigherThanGross(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)

	result := fm.Compute(1.0850, costsvc.EstimateParams{
		Side: "buy", Lots: 1.0, Price: 1.0850, ContractSize: 100000, HoldingDays: 0,
	}, false)

	if result.NetFillPrice <= result.GrossPrice {
		t.Fatalf("buy: net fill %.6f should be > gross %.6f", result.NetFillPrice, result.GrossPrice)
	}
	if result.Commission <= 0 {
		t.Fatalf("commission should be > 0, got %.4f", result.Commission)
	}
}

func TestFillModel_Sell_NetLowerThanGross(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)

	result := fm.Compute(1.0850, costsvc.EstimateParams{
		Side: "sell", Lots: 1.0, Price: 1.0850, ContractSize: 100000, HoldingDays: 0,
	}, false)

	if result.NetFillPrice >= result.GrossPrice {
		t.Fatalf("sell: net fill %.6f should be < gross %.6f", result.NetFillPrice, result.GrossPrice)
	}
}

func TestFillModel_Backtest_EnforcesNonZeroCosts(t *testing.T) {
	cm := &costsvc.CostModel{
		Symbol: "TEST", PipSize: 0.0001, PipValue: 10,
		SpreadPips: 0, CommissionPerLot: 0, CommissionBps: 0, SlippageBps: 0,
	}
	fm := NewFillModel(cm)

	result := fm.Compute(1.0850, costsvc.EstimateParams{
		Side: "buy", Lots: 1.0, Price: 1.0850, ContractSize: 100000, HoldingDays: 0,
	}, true) // isBacktest=true

	if result.SpreadCost <= 0 {
		t.Fatalf("backtest should enforce non-zero spread, got %.4f", result.SpreadCost)
	}
	if result.SlippageCost <= 0 {
		t.Fatalf("backtest should enforce non-zero slippage, got %.4f", result.SlippageCost)
	}
	if result.Commission <= 0 {
		t.Fatalf("backtest should enforce non-zero commission, got %.4f", result.Commission)
	}
	if result.NetFillPrice <= result.GrossPrice {
		t.Fatalf("backtest buy: net %.6f > gross %.6f", result.NetFillPrice, result.GrossPrice)
	}
	t.Logf("Backtest fill: gross=%.6f net=%.6f costs: spread=%.4f comm=%.4f slip=%.4f",
		result.GrossPrice, result.NetFillPrice, result.SpreadCost, result.Commission, result.SlippageCost)
}

func TestFillModel_ComputeNet(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)

	net := fm.ComputeNet(1.0850, costsvc.EstimateParams{
		Side: "buy", Lots: 1.0, Price: 1.0850, ContractSize: 100000, HoldingDays: 0,
	}, false)

	if net <= 1.0850 {
		t.Fatalf("net should be > gross for buy")
	}
}

func TestFillModel_ZeroVolume(t *testing.T) {
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := NewFillModel(cm)

	result := fm.Compute(1.0850, costsvc.EstimateParams{
		Side: "buy", Lots: 0, Price: 1.0850, ContractSize: 100000, HoldingDays: 0,
	}, false)

	if math.Abs(result.NetFillPrice-1.0850) > 0.0001 {
		t.Fatalf("zero volume: net should equal gross, got %.6f", result.NetFillPrice)
	}
}
