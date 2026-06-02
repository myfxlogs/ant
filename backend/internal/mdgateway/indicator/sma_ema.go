package indicator

import (
	"anttrader/internal/mdgateway/adapter/mdtick"
	"github.com/shopspring/decimal"
)

var DefSMA = Def{
	ID:   "SMA",
	Name: "Simple Moving Average",
	Kind: KindOverlay,
	Params: []Param{
		{Key: "length", Label: "Period", Type: "int", Default: 20, Min: 1, Max: 500, Step: 1},
	},
	Defaults: map[string]float64{"length": 20},
}

func computeSMA(bars []mdtick.Bar, params map[string]float64) (*Result, error) {
	n := int(getParam(params, "length", 20))
	closes := closeSeries(bars)
	vals := sma(closes, n)
	return &Result{
		DefID: "SMA",
		Lines: map[string][]decimal.Decimal{"sma": vals},
	}, nil
}

var DefEMA = Def{
	ID:   "EMA",
	Name: "Exponential Moving Average",
	Kind: KindOverlay,
	Params: []Param{
		{Key: "length", Label: "Period", Type: "int", Default: 20, Min: 1, Max: 500, Step: 1},
	},
	Defaults: map[string]float64{"length": 20},
}

func computeEMA(bars []mdtick.Bar, params map[string]float64) (*Result, error) {
	n := int(getParam(params, "length", 20))
	closes := closeSeries(bars)
	vals := ema(closes, n)
	return &Result{
		DefID: "EMA",
		Lines: map[string][]decimal.Decimal{"ema": vals},
	}, nil
}

// helpers

func closeSeries(bars []mdtick.Bar) []decimal.Decimal {
	out := make([]decimal.Decimal, len(bars))
	for i, b := range bars {
		out[i] = b.Close
	}
	return out
}

func getParam(params map[string]float64, key string, def float64) float64 {
	if v, ok := params[key]; ok {
		return v
	}
	return def
}

func toOHLCV(bars []mdtick.Bar) []BarOHLCV {
	out := make([]BarOHLCV, len(bars))
	for i, b := range bars {
		out[i] = BarOHLCV{
			Open: b.Open, High: b.High, Low: b.Low, Close: b.Close,
			Volume: b.Volume,
		}
	}
	return out
}
