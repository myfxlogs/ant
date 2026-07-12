package backtest

import (
	"math"
	"testing"

	"alphaforge/internal/costsvc"
)

func newTestEngine(bars []Bar, capital float64, cm *costsvc.CostModel) *Engine {
	e := NewEngine(bars, capital, cm)
	e.SetSeed(42)
	return e
}

func TestFillSim_Buy(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	e := newTestEngine(nil, 10000, cm)
	e.SlippagePips = 0.5
	e.PartialFillProb = 0
	e.ContractSize = 100000
	result := e.simulateFill(1, 1.0, 1.0850, 0)

	if result.FilledVolume <= 0 {
		t.Fatal("should fill some volume")
	}
	if result.GrossPrice <= 0 {
		t.Fatal("should have gross price")
	}
	if result.NetFillPrice <= result.GrossPrice {
		t.Fatal("buy net price should be > gross (costs added)")
	}
	if result.TotalCost <= 0 {
		t.Fatal("should have non-zero cost in backtest mode")
	}
}

func TestFillSim_Sell(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	e := newTestEngine(nil, 10000, cm)
	e.SlippagePips = 0.5
	e.ContractSize = 100000
	result := e.simulateFill(-1, 1.0, 1.0850, 0)

	if result.FilledVolume <= 0 {
		t.Fatal("should fill some volume")
	}
	if result.NetFillPrice >= result.GrossPrice {
		t.Fatal("sell net price should be < gross (costs deducted)")
	}
}

func TestFillSim_PartialFill(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	e := newTestEngine(nil, 10000, cm)
	e.SlippagePips = 0
	e.PartialFillProb = 1.0
	e.PartialFillRatio = 0.5
	e.ContractSize = 100000
	result := e.simulateFill(1, 2.0, 1.0850, 0)

	if result.FilledVolume >= 2.0 {
		t.Fatal("partial fill should reduce volume")
	}
	if result.FilledVolume < 0.5 {
		t.Fatal("should still fill some")
	}
}

func TestFillSim_NoCostModel(t *testing.T) {
	t.Parallel()
	e := NewEngine(nil, 10000, nil)
	e.SetSeed(42)
	e.SlippagePips = 0
	e.ContractSize = 100000
	result := e.simulateFill(1, 1.0, 1.0850, 0)

	if result.FilledVolume <= 0 {
		t.Fatal("should fill without cost model")
	}
}

func TestEngine_LongWin(t *testing.T) {
	t.Parallel()
	bars := []Bar{
		{OpenTime: 1, CloseTime: 2, Open: 1.1000, High: 1.1050, Low: 1.0990, Close: 1.1020},
		{OpenTime: 2, CloseTime: 3, Open: 1.1020, High: 1.1080, Low: 1.1010, Close: 1.1060},
		{OpenTime: 3, CloseTime: 4, Open: 1.1060, High: 1.1100, Low: 1.1050, Close: 1.1090},
	}
	cm := costsvc.DefaultForexModel("EURUSD")
	e := newTestEngine(bars, 10000, cm)
	e.SlippagePips = 0
	e.PartialFillProb = 0
	e.ContractSize = 100000
	metrics := e.Run(func(bar Bar, pos float64) (int, float64) {
		if pos == 0 {
			return 1, 1.0 // buy 1 lot
		}
		if bar.Close > 1.1070 {
			return -1, 1.0 // sell to close
		}
		return 1, 0
	})
	if metrics.TotalTrades < 1 {
		t.Fatal("should have at least 1 trade")
	}
	if metrics.NetPnL <= 0 {
		t.Fatal("winning strategy should have positive net PnL")
	}
}

func TestEngine_FlatMarket(t *testing.T) {
	t.Parallel()
	bars := []Bar{
		{OpenTime: 1, CloseTime: 2, Close: 1.1000},
		{OpenTime: 2, CloseTime: 3, Close: 1.1000},
	}
	cm := costsvc.DefaultForexModel("EURUSD")
	e := newTestEngine(bars, 10000, cm)
	e.SlippagePips = 0
	metrics := e.Run(func(bar Bar, pos float64) (int, float64) {
		return 0, 0 // never trade
	})
	if metrics.TotalTrades != 0 {
		t.Fatal("flat strategy should produce no trades")
	}
	if len(e.EquityCurve()) != len(bars) {
		t.Fatalf("equity curve length %d != bars %d", len(e.EquityCurve()), len(bars))
	}
}

func TestEngine_ShortWin(t *testing.T) {
	t.Parallel()
	bars := []Bar{
		{OpenTime: 1, CloseTime: 2, Close: 1.1100},
		{OpenTime: 2, CloseTime: 3, Close: 1.1000},
	}
	cm := costsvc.DefaultForexModel("EURUSD")
	e := newTestEngine(bars, 10000, cm)
	e.SlippagePips = 0
	e.PartialFillProb = 0
	e.ContractSize = 100000
	metrics := e.Run(func(bar Bar, pos float64) (int, float64) {
		if pos == 0 {
			return -1, 1.0
		}
		return 1, 0 // close
	})
	if metrics.TotalTrades < 1 {
		t.Fatal("should have at least 1 trade")
	}
}

func TestEngine_Reproducibility(t *testing.T) {
	t.Parallel()
	bars := []Bar{
		{OpenTime: 1, CloseTime: 2, Close: 1.1050},
		{OpenTime: 2, CloseTime: 3, Close: 1.1020},
	}
	cm := costsvc.DefaultForexModel("EURUSD")
	runAll := func() *Metrics {
		e := newTestEngine(bars, 10000, cm)
		e.SlippagePips = 0.5
		e.PartialFillProb = 0.2
		e.ContractSize = 100000
		return e.Run(func(bar Bar, pos float64) (int, float64) {
			if pos == 0 {
				return 1, 1.0
			}
			return -1, 0
		})
	}
	r1 := runAll()
	r2 := runAll()
	if math.Abs(r1.NetPnL-r2.NetPnL) > 0.0001 {
		t.Fatal("reproducibility: same seed should produce identical results")
	}
}

func TestEngine_ForceClose(t *testing.T) {
	t.Parallel()
	bars := []Bar{
		{OpenTime: 1, CloseTime: 2, Close: 1.1000},
		{OpenTime: 2, CloseTime: 3, Close: 1.1050},
	}
	cm := costsvc.DefaultForexModel("EURUSD")
	e := newTestEngine(bars, 10000, cm)
	e.SlippagePips = 0
	e.PartialFillProb = 0
	e.ContractSize = 100000
	metrics := e.Run(func(bar Bar, pos float64) (int, float64) {
		if pos == 0 {
			return 1, 1.0 // enter but never exit
		}
		return 1, 0
	})
	if metrics.TotalTrades < 1 {
		t.Fatal("force-close at end should produce a trade")
	}
}
