package indicator

import (
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"github.com/shopspring/decimal"
)

var DefVolume = Def{
	ID:   "VOL",
	Name: "Volume",
	Kind: KindSubPane,
	Params: []Param{},
	Defaults: map[string]float64{},
}

func computeVOL(bars []mdtick.Bar, _ map[string]float64) (*Result, error) {
	vols := make([]decimal.Decimal, len(bars))
	for i, b := range bars {
		vols[i] = decimal.NewFromFloat(b.Volume)
	}
	return &Result{DefID: "VOL", Lines: map[string][]decimal.Decimal{"volume": vols}}, nil
}

var DefATR = Def{
	ID:   "ATR",
	Name: "Average True Range",
	Kind: KindSubPane,
	Params: []Param{
		{Key: keyLength, Label: labelPeriod, Type: typeInt, Default: 14, Min: 1, Max: 500, Step: 1},
	},
	Defaults: map[string]float64{keyLength: 14},
}

func computeATR(bars []mdtick.Bar, params map[string]float64) (*Result, error) {
	n := int(getParam(params, keyLength, 14))
	ohlcv := toOHLCV(bars)
	tr := trueRange(ohlcv)
	atrVals := ema(tr, n) // Wilder's ATR uses EMA-like smoothing
	return &Result{DefID: "ATR", Lines: map[string][]decimal.Decimal{"atr": atrVals}}, nil
}

var DefCCI = Def{
	ID:   "CCI",
	Name: "Commodity Channel Index",
	Kind: KindSubPane,
	Params: []Param{
		{Key: keyLength, Label: labelPeriod, Type: typeInt, Default: 20, Min: 1, Max: 500, Step: 1},
	},
	Defaults: map[string]float64{keyLength: 20},
}

func computeCCI(bars []mdtick.Bar, params map[string]float64) (*Result, error) {
	n := int(getParam(params, keyLength, 20))
	ohlcv := toOHLCV(bars)
	cci := make([]decimal.Decimal, len(bars))
	if len(bars) <= n {
		return &Result{DefID: "CCI", Lines: map[string][]decimal.Decimal{"cci": cci}}, nil
	}
	for i := n - 1; i < len(bars); i++ {
		var sumTP decimal.Decimal
		for j := i - n + 1; j <= i; j++ {
			tp := ohlcv[j].High.Add(ohlcv[j].Low).Add(ohlcv[j].Close).Div(decimal.NewFromInt(3))
			sumTP = sumTP.Add(tp)
		}
		smaTP := sumTP.Div(decimal.NewFromInt(int64(n)))
		var sumDev decimal.Decimal
		for j := i - n + 1; j <= i; j++ {
			tp := ohlcv[j].High.Add(ohlcv[j].Low).Add(ohlcv[j].Close).Div(decimal.NewFromInt(3))
			sumDev = sumDev.Add(tp.Sub(smaTP).Abs())
		}
		meanDev := sumDev.Div(decimal.NewFromInt(int64(n)))
		if !meanDev.IsZero() {
			tp := ohlcv[i].High.Add(ohlcv[i].Low).Add(ohlcv[i].Close).Div(decimal.NewFromInt(3))
			cci[i] = tp.Sub(smaTP).Div(meanDev.Mul(decimal.NewFromFloat(0.015)))
		}
	}
	return &Result{DefID: "CCI", Lines: map[string][]decimal.Decimal{"cci": cci}}, nil
}

var DefWilliamsR = Def{
	ID:   "WILLR",
	Name: "Williams %R",
	Kind: KindSubPane,
	Params: []Param{
		{Key: keyLength, Label: labelPeriod, Type: typeInt, Default: 14, Min: 1, Max: 500, Step: 1},
	},
	Defaults: map[string]float64{keyLength: 14},
}

func computeWilliamsR(bars []mdtick.Bar, params map[string]float64) (*Result, error) {
	n := int(getParam(params, keyLength, 14))
	ohlcv := toOHLCV(bars)
	wr := make([]decimal.Decimal, len(bars))
	if len(bars) <= n {
		return &Result{DefID: "WILLR", Lines: map[string][]decimal.Decimal{"willr": wr}}, nil
	}
	highs := make([]decimal.Decimal, len(bars))
	lows := make([]decimal.Decimal, len(bars))
	for i := range ohlcv {
		highs[i] = ohlcv[i].High
		lows[i] = ohlcv[i].Low
	}
	hh := highest(highs, n)
	ll := lowest(lows, n)
	for i := n - 1; i < len(bars); i++ {
		diff := hh[i].Sub(ll[i])
		if !diff.IsZero() {
			wr[i] = hh[i].Sub(ohlcv[i].Close).Div(diff).Mul(decimal.NewFromInt(-100))
		}
	}
	return &Result{DefID: "WILLR", Lines: map[string][]decimal.Decimal{"willr": wr}}, nil
}
