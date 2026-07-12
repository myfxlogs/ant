package indicator

import (
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"github.com/shopspring/decimal"
)

var DefBollinger = Def{
	ID:   "BOLL",
	Name: "Bollinger Bands",
	Kind: KindOverlay,
	Params: []Param{
		{Key: "length", Label: "Period", Type: "int", Default: 20, Min: 1, Max: 500, Step: 1},
		{Key: "mult", Label: "Multiplier", Type: "float", Default: 2, Min: 0.1, Max: 10, Step: 0.1},
	},
	Defaults: map[string]float64{"length": 20, "mult": 2},
}

func computeBollinger(bars []mdtick.Bar, params map[string]float64) (*Result, error) {
	n := int(getParam(params, "length", 20))
	mult := decimal.NewFromFloat(getParam(params, "mult", 2))

	closes := closeSeries(bars)
	middle := sma(closes, n)
	upper := make([]decimal.Decimal, len(bars))
	lower := make([]decimal.Decimal, len(bars))
	std := stddev(closes, n)

	for i := range bars {
		if i >= n-1 && !std[i].IsZero() {
			band := mult.Mul(std[i])
			upper[i] = middle[i].Add(band)
			lower[i] = middle[i].Sub(band)
		}
	}
	return &Result{
		DefID: "BOLL",
		Lines: map[string][]decimal.Decimal{
			"middle": middle,
			"upper":  upper,
			"lower":  lower,
		},
	}, nil
}

var DefRSI = Def{
	ID:   "RSI",
	Name: "Relative Strength Index",
	Kind: KindSubPane,
	Params: []Param{
		{Key: "length", Label: "Period", Type: "int", Default: 14, Min: 1, Max: 500, Step: 1},
	},
	Defaults: map[string]float64{"length": 14},
}

func computeRSI(bars []mdtick.Bar, params map[string]float64) (*Result, error) {
	n := int(getParam(params, "length", 14))
	closes := closeSeries(bars)
	rsi := make([]decimal.Decimal, len(bars))

	if len(bars) <= n {
		return &Result{DefID: "RSI", Lines: map[string][]decimal.Decimal{"rsi": rsi}}, nil
	}

	var avgGain, avgLoss decimal.Decimal
	for i := 1; i <= n; i++ {
		diff := closes[i].Sub(closes[i-1])
		if diff.GreaterThan(decimal.Zero) {
			avgGain = avgGain.Add(diff)
		} else {
			avgLoss = avgLoss.Sub(diff)
		}
	}
	avgGain = avgGain.Div(decimal.NewFromInt(int64(n)))
	avgLoss = avgLoss.Div(decimal.NewFromInt(int64(n)))

	for i := n; i < len(bars); i++ {
		if !avgLoss.IsZero() {
			rs := avgGain.Div(avgLoss)
			rsi[i] = decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)))
		} else {
			rsi[i] = decimal.NewFromInt(100)
		}
		diff := closes[i].Sub(closes[i-1])
		var gain, loss decimal.Decimal
		if diff.GreaterThan(decimal.Zero) {
			gain = diff
		} else {
			loss = diff.Abs()
		}
		avgGain = avgGain.Mul(decimal.NewFromFloat(float64(n-1))).Add(gain).Div(decimal.NewFromInt(int64(n)))
		avgLoss = avgLoss.Mul(decimal.NewFromFloat(float64(n-1))).Add(loss).Div(decimal.NewFromInt(int64(n)))
	}
	return &Result{DefID: "RSI", Lines: map[string][]decimal.Decimal{"rsi": rsi}}, nil
}

var DefMACD = Def{
	ID:   "MACD",
	Name: "MACD",
	Kind: KindSubPane,
	Params: []Param{
		{Key: "fast", Label: "Fast Period", Type: "int", Default: 12, Min: 1, Max: 500, Step: 1},
		{Key: "slow", Label: "Slow Period", Type: "int", Default: 26, Min: 1, Max: 500, Step: 1},
		{Key: "signal", Label: "Signal Period", Type: "int", Default: 9, Min: 1, Max: 500, Step: 1},
	},
	Defaults: map[string]float64{"fast": 12, "slow": 26, "signal": 9},
}

func computeMACD(bars []mdtick.Bar, params map[string]float64) (*Result, error) {
	fast := int(getParam(params, "fast", 12))
	slow := int(getParam(params, "slow", 26))
	sig := int(getParam(params, "signal", 9))
	closes := closeSeries(bars)

	emaFast := ema(closes, fast)
	emaSlow := ema(closes, slow)
	macdLine := make([]decimal.Decimal, len(bars))
	for i := range bars {
		macdLine[i] = emaFast[i].Sub(emaSlow[i])
	}
	signalLine := ema(macdLine, sig)
	histogram := make([]decimal.Decimal, len(bars))
	for i := range bars {
		histogram[i] = macdLine[i].Sub(signalLine[i])
	}
	return &Result{
		DefID: "MACD",
		Lines: map[string][]decimal.Decimal{
			"macd": macdLine, "signal": signalLine, "histogram": histogram,
		},
	}, nil
}
