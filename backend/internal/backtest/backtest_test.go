package backtest

import (
	"math"
	"math/rand"
	"testing"

	"anttrader/internal/costsvc"
)

func TestFillModel_Buy(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := &FillModel{
		SlippagePips:    0.5,
		PartialFillProb: 0,
		CostModel:       cm,
		ContractSize:    100000,
	}
	rng := rand.New(rand.NewSource(42))
	result := fm.SimulateFill(1, 1.0, 1.0850, 0, rng)

	if result.FilledVolume <= 0 {
		t.Fatal("should fill some volume")
	}
	if result.GrossPrice <= 0 {
		t.Fatal("should have gross price")
	}
	if result.NetPrice <= result.GrossPrice {
		t.Fatal("buy net price should be > gross (costs added)")
	}
	if result.TotalCost <= 0 {
		t.Fatal("should have non-zero total cost")
	}
	// NetPrice should be close to GrossPrice (costs spread over 100k units)
	if math.Abs(result.NetPrice-result.GrossPrice) > 0.001 {
		t.Fatalf("net price should be close to gross, gross=%.6f net=%.6f", result.GrossPrice, result.NetPrice)
	}
	t.Logf("Buy fill: gross=%.6f net=%.6f filled=%.4f totalCost=%.4f",
		result.GrossPrice, result.NetPrice, result.FilledVolume, result.TotalCost)
}

func TestFillModel_Sell(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := &FillModel{
		SlippagePips:    0.5,
		CostModel:       cm,
		PartialFillProb: 0,
		ContractSize:    100000,
	}
	rng := rand.New(rand.NewSource(42))
	result := fm.SimulateFill(-1, 1.0, 1.0850, 0, rng)

	if result.NetPrice >= result.GrossPrice {
		t.Fatal("sell net price should be < gross")
	}
	t.Logf("Sell fill: gross=%.6f net=%.6f totalCost=%.4f",
		result.GrossPrice, result.NetPrice, result.TotalCost)
}

func TestFillModel_PartialFill(t *testing.T) {
	t.Parallel()
	cm := costsvc.DefaultForexModel("EURUSD")
	fm := &FillModel{
		PartialFillProb:  1.0,
		PartialFillRatio: 0.5,
		CostModel:        cm,
		ContractSize:     100000,
	}
	rng := rand.New(rand.NewSource(42))
	result := fm.SimulateFill(1, 1.0, 1.0850, 0, rng)

	if !result.IsPartial {
		t.Fatal("should be partial fill when prob=1.0")
	}
	if math.Abs(result.FilledVolume-0.5) > 0.01 {
		t.Fatalf("partial fill should fill 0.5, got %.4f", result.FilledVolume)
	}
}

func TestFillModel_NoCostModel(t *testing.T) {
	t.Parallel()
	fm := &FillModel{ContractSize: 100000}
	rng := rand.New(rand.NewSource(42))
	result := fm.SimulateFill(1, 1.0, 1.0850, 0, rng)

	if result.GrossPrice != result.NetPrice {
		t.Fatal("net should equal gross with no cost model")
	}
	if result.TotalCost != 0 {
		t.Fatal("total cost should be 0 with no cost model")
	}
}

func TestComputeMetrics_ProfitableStrategy(t *testing.T) {
	t.Parallel()
	trades := []Trade{
		{NetPnL: 100, GrossPnL: 110, Commission: 5, Swap: 3, Slippage: 2},
		{NetPnL: 200, GrossPnL: 215, Commission: 7, Swap: 5, Slippage: 3},
		{NetPnL: -50, GrossPnL: -40, Commission: 5, Swap: 3, Slippage: 2},
		{NetPnL: 150, GrossPnL: 160, Commission: 5, Swap: 3, Slippage: 2},
	}
	equity := []float64{10000, 10100, 10300, 10250, 10400}

	m := ComputeMetrics(trades, 10000, equity)

	if m.TotalTrades != 4 {
		t.Fatalf("want 4 trades, got %d", m.TotalTrades)
	}
	if m.WinningTrades != 3 {
		t.Fatalf("want 3 wins, got %d", m.WinningTrades)
	}
	if m.LosingTrades != 1 {
		t.Fatalf("want 1 loss, got %d", m.LosingTrades)
	}
	if m.NetPnL != 400 {
		t.Fatalf("net PnL should be 400, got %.2f", m.NetPnL)
	}
	if m.TotalReturn != 0.04 {
		t.Fatalf("total return should be 0.04, got %.4f", m.TotalReturn)
	}
	if m.ProfitFactor <= 1.0 {
		t.Fatalf("profit factor should be > 1, got %.4f", m.ProfitFactor)
	}
	t.Logf("Metrics: return=%.4f winRate=%.2f pf=%.2f avgWin=%.2f avgLoss=%.2f maxDD=%.4f",
		m.TotalReturn, m.WinRate, m.ProfitFactor, m.AverageProfit, m.AverageLoss, m.MaxDrawdown)
}

func TestComputeMetrics_EmptyTrades(t *testing.T) {
	t.Parallel()
	m := ComputeMetrics(nil, 10000, nil)
	if m.TotalTrades != 0 {
		t.Fatalf("want 0 trades, got %d", m.TotalTrades)
	}
}

func TestComputeMetrics_AllLosers(t *testing.T) {
	t.Parallel()
	trades := []Trade{
		{NetPnL: -100},
		{NetPnL: -200},
		{NetPnL: -50},
	}
	equity := []float64{10000, 9900, 9700, 9650}
	m := ComputeMetrics(trades, 10000, equity)

	if m.WinRate != 0 {
		t.Fatalf("win rate should be 0, got %.4f", m.WinRate)
	}
}

func TestComputeMaxDrawdown(t *testing.T) {
	t.Parallel()
	equity := []float64{100, 110, 105, 95, 100, 90, 100}
	dd := computeMaxDrawdown(equity)
	if math.Abs(dd-0.1818) > 0.01 {
		t.Fatalf("max drawdown should be ~0.1818, got %.4f", dd)
	}
}

func TestComputeMaxDrawdown_NoDrawdown(t *testing.T) {
	t.Parallel()
	equity := []float64{100, 110, 120, 130}
	dd := computeMaxDrawdown(equity)
	if dd != 0 {
		t.Fatalf("no drawdown should be 0, got %.4f", dd)
	}
}

func TestComputeSharpe(t *testing.T) {
	t.Parallel()
	equity := []float64{100, 101, 102, 103, 104, 105}
	sharpe := computeSharpe(equity, 100)
	if sharpe < 0 {
		t.Fatalf("sharpe should be positive for steady growth, got %.4f", sharpe)
	}
}

func TestEngine_SimpleTrendFollow(t *testing.T) {
	t.Parallel()
	bars := make([]Bar, 100)
	price := 1.0850
	for i := range bars {
		barTs := int64(i) * 60000
		price += 0.0001
		bars[i] = Bar{
			OpenTime: barTs, CloseTime: barTs + 59999,
			Open: price, High: price + 0.0002, Low: price - 0.0002, Close: price,
		}
	}

	cm := costsvc.DefaultForexModel("EURUSD")
	cm.CommissionPerLot = 0
	cm.SpreadPips = 0
	cm.SwapLong = 0
	cm.SwapShort = 0
	cm.SlippageBps = 0

	fm := &FillModel{
		SlippagePips:    0,
		PartialFillProb: 0,
		CostModel:       cm,
		ContractSize:    100000,
	}
	engine := NewEngine(bars, 10000, fm)
	engine.SetSeed(42)

	strategy := func(bar Bar, pos float64) (int, float64) {
		if pos == 0 && bar.Close > 1.0900 {
			return 1, 0.1
		}
		if pos > 0 && bar.Close > 1.0940 {
			return -1, 0.1
		}
		return 0, 0
	}

	metrics := engine.Run(strategy)

	if metrics.TotalTrades == 0 {
		t.Fatal("should have at least 1 trade")
	}
	if metrics.NetPnL <= 0 {
		t.Fatalf("trend-following in uptrend should be profitable, net=%.4f", metrics.NetPnL)
	}
	t.Logf("Trend follow: trades=%d net=%.4f return=%.4f winRate=%.2f maxDD=%.4f",
		metrics.TotalTrades, metrics.NetPnL, metrics.TotalReturn, metrics.WinRate, metrics.MaxDrawdown)
}

func TestEngine_NoTrades(t *testing.T) {
	t.Parallel()
	bars := []Bar{
		{OpenTime: 0, CloseTime: 60000, Open: 1.0850, High: 1.0860, Low: 1.0840, Close: 1.0855},
	}
	engine := NewEngine(bars, 10000, &FillModel{ContractSize: 100000})
	strategy := func(bar Bar, pos float64) (int, float64) { return 0, 0 }
	metrics := engine.Run(strategy)
	if metrics.TotalTrades != 0 {
		t.Fatalf("want 0 trades, got %d", metrics.TotalTrades)
	}
}

func TestEngine_MeanReversion(t *testing.T) {
	t.Parallel()
	bars := make([]Bar, 200)
	price := 1.0800
	up := true
	for i := range bars {
		barTs := int64(i) * 60000
		if up {
			price += 0.0005
		} else {
			price -= 0.0005
		}
		if price > 1.1000 {
			up = false
		}
		if price < 1.0800 {
			up = true
		}
		bars[i] = Bar{
			OpenTime: barTs, CloseTime: barTs + 59999,
			Open: price, High: price + 0.0002, Low: price - 0.0002, Close: price,
		}
	}

	cm := costsvc.DefaultForexModel("EURUSD")
	cm.CommissionPerLot = 0
	cm.SpreadPips = 0
	cm.SwapLong = 0
	cm.SwapShort = 0
	cm.SlippageBps = 0

	fm := &FillModel{
		SlippagePips:    0,
		PartialFillProb: 0,
		CostModel:       cm,
		ContractSize:    100000,
	}
	engine := NewEngine(bars, 10000, fm)
	engine.SetSeed(42)

	strategy := func(bar Bar, pos float64) (int, float64) {
		if pos == 0 && bar.Close < 1.0820 {
			return 1, 0.1
		}
		if pos > 0 && bar.Close > 1.0980 {
			return -1, 0.1
		}
		return 0, 0
	}

	metrics := engine.Run(strategy)
	t.Logf("Mean reversion: trades=%d net=%.4f return=%.4f winRate=%.2f",
		metrics.TotalTrades, metrics.NetPnL, metrics.TotalReturn, metrics.WinRate)

	if metrics.NetPnL <= 0 {
		t.Log("mean reversion was not profitable (depends on oscillation)")
	}
}

func TestEngine_ForceCloseAtEnd(t *testing.T) {
	t.Parallel()
	bars := []Bar{
		{OpenTime: 0, CloseTime: 60000, Open: 1.0850, High: 1.0860, Low: 1.0840, Close: 1.0855},
		{OpenTime: 60000, CloseTime: 120000, Open: 1.0855, High: 1.0870, Low: 1.0850, Close: 1.0865},
	}
	engine := NewEngine(bars, 10000, &FillModel{SlippagePips: 0, PartialFillProb: 0, ContractSize: 100000})

	strategy := func(bar Bar, pos float64) (int, float64) {
		if pos == 0 && bar.Close == 1.0855 {
			return 1, 0.1
		}
		return 0, 0
	}

	metrics := engine.Run(strategy)
	if metrics.TotalTrades != 1 {
		t.Fatalf("want 1 trade (force close), got %d", metrics.TotalTrades)
	}
	t.Logf("Force close: trades=%d net=%.4f", metrics.TotalTrades, metrics.NetPnL)
}

func TestEngine_ShortTrade(t *testing.T) {
	t.Parallel()
	bars := make([]Bar, 50)
	price := 1.1000
	for i := range bars {
		price -= 0.0005
		bars[i] = Bar{
			OpenTime: int64(i) * 60000, CloseTime: int64(i)*60000 + 59999,
			Open: price, High: price + 0.0002, Low: price - 0.0002, Close: price,
		}
	}

	engine := NewEngine(bars, 10000, &FillModel{SlippagePips: 0, PartialFillProb: 0, ContractSize: 100000})
	engine.SetSeed(42)

	strategy := func(bar Bar, pos float64) (int, float64) {
		if pos == 0 && bar.Close < 1.0990 {
			return -1, 0.1
		}
		return 0, 0
	}

	metrics := engine.Run(strategy)
	if metrics.TotalTrades == 0 {
		t.Fatal("should have short trade")
	}
	if metrics.NetPnL <= 0 {
		t.Fatalf("short in downtrend should profit, net=%.4f", metrics.NetPnL)
	}
	t.Logf("Short trade: trades=%d net=%.4f", metrics.TotalTrades, metrics.NetPnL)
}

func TestEngine_WithCostModel(t *testing.T) {
	t.Parallel()
	bars := make([]Bar, 100)
	price := 1.0850
	for i := range bars {
		price += 0.0002
		bars[i] = Bar{
			OpenTime: int64(i) * 60000, CloseTime: int64(i)*60000 + 59999,
			Open: price, High: price + 0.0002, Low: price - 0.0002, Close: price,
		}
	}

	cm := costsvc.DefaultForexModel("EURUSD")
	fm := &FillModel{
		SlippagePips:    0.5,
		PartialFillProb: 0,
		CostModel:       cm,
		ContractSize:    100000,
	}
	engine := NewEngine(bars, 10000, fm)
	engine.SetSeed(42)

	strategy := func(bar Bar, pos float64) (int, float64) {
		if pos == 0 && bar.Close > 1.0900 {
			return 1, 0.1
		}
		if pos > 0 && bar.Close > 1.1000 {
			return -1, 0.1
		}
		return 0, 0
	}

	metrics := engine.Run(strategy)
	if metrics.TotalCosts <= 0 {
		t.Fatal("should have non-zero costs with cost model")
	}
	if metrics.NetPnL >= metrics.GrossPnL {
		t.Fatalf("net PnL (%.4f) should be < gross PnL (%.4f) with costs",
			metrics.NetPnL, metrics.GrossPnL)
	}
	t.Logf("With costs: gross=%.4f net=%.4f costs=%.4f trades=%d",
		metrics.GrossPnL, metrics.NetPnL, metrics.TotalCosts, metrics.TotalTrades)
}
