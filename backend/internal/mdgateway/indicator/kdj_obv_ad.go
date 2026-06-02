package indicator

import (
	"anttrader/internal/mdgateway/adapter/mdtick"
	"github.com/shopspring/decimal"
)

var DefKDJ = Def{
	ID:   "KDJ",
	Name: "Stochastic (KDJ)",
	Kind: KindSubPane,
	Params: []Param{
		{Key: "period", Label: "Period", Type: "int", Default: 9, Min: 1, Max: 500, Step: 1},
		{Key: "k", Label: "K Smooth", Type: "int", Default: 3, Min: 1, Max: 100, Step: 1},
		{Key: "d", Label: "D Smooth", Type: "int", Default: 3, Min: 1, Max: 100, Step: 1},
	},
	Defaults: map[string]float64{"period": 9, "k": 3, "d": 3},
}

func computeKDJ(bars []mdtick.Bar, params map[string]float64) (*Result, error) {
	period := int(getParam(params, "period", 9))
	ks := int(getParam(params, "k", 3))
	ds := int(getParam(params, "d", 3))
	ohlcv := toOHLCV(bars)
	k := make([]decimal.Decimal, len(bars))
	d := make([]decimal.Decimal, len(bars))
	j := make([]decimal.Decimal, len(bars))
	if len(bars) <= period {
		return &Result{DefID: "KDJ", Lines: map[string][]decimal.Decimal{"k": k, "d": d, "j": j}}, nil
	}
	highs := make([]decimal.Decimal, len(bars))
	lows := make([]decimal.Decimal, len(bars))
	for i := range ohlcv {
		highs[i] = ohlcv[i].High
		lows[i] = ohlcv[i].Low
	}
	hh := highest(highs, period)
	ll := lowest(lows, period)
	rsv := make([]decimal.Decimal, len(bars))
	for i := period - 1; i < len(bars); i++ {
		diff := hh[i].Sub(ll[i])
		if !diff.IsZero() {
			rsv[i] = ohlcv[i].Close.Sub(ll[i]).Div(diff).Mul(decimal.NewFromInt(100))
		}
	}
	// Smooth RSV → K → D → J
	for i := range bars {
		if i == 0 {
			k[i] = decimal.NewFromInt(50)
			d[i] = decimal.NewFromInt(50)
		} else {
			k[i] = k[i-1].Mul(decimal.NewFromFloat(float64(ks-1))).Add(rsv[i]).Div(decimal.NewFromInt(int64(ks)))
			d[i] = d[i-1].Mul(decimal.NewFromFloat(float64(ds-1))).Add(k[i]).Div(decimal.NewFromInt(int64(ds)))
		}
		j[i] = k[i].Mul(decimal.NewFromInt(3)).Sub(d[i].Mul(decimal.NewFromInt(2)))
	}
	return &Result{DefID: "KDJ", Lines: map[string][]decimal.Decimal{"k": k, "d": d, "j": j}}, nil
}

var DefOBV = Def{
	ID:   "OBV",
	Name: "On-Balance Volume",
	Kind: KindSubPane,
	Params: []Param{},
	Defaults: map[string]float64{},
}

func computeOBV(bars []mdtick.Bar, _ map[string]float64) (*Result, error) {
	obv := make([]decimal.Decimal, len(bars))
	if len(bars) == 0 {
		return &Result{DefID: "OBV", Lines: map[string][]decimal.Decimal{"obv": obv}}, nil
	}
	obv[0] = decimal.NewFromFloat(bars[0].Volume)
	for i := 1; i < len(bars); i++ {
		vol := decimal.NewFromFloat(bars[i].Volume)
		switch {
		case bars[i].Close.GreaterThan(bars[i-1].Close):
			obv[i] = obv[i-1].Add(vol)
		case bars[i].Close.LessThan(bars[i-1].Close):
			obv[i] = obv[i-1].Sub(vol)
		default:
			obv[i] = obv[i-1]
		}
	}
	return &Result{DefID: "OBV", Lines: map[string][]decimal.Decimal{"obv": obv}}, nil
}

var DefAD = Def{
	ID:   "AD",
	Name: "Accumulation/Distribution Line",
	Kind: KindSubPane,
	Params: []Param{},
	Defaults: map[string]float64{},
}

func computeAD(bars []mdtick.Bar, _ map[string]float64) (*Result, error) {
	ad := make([]decimal.Decimal, len(bars))
	if len(bars) == 0 {
		return &Result{DefID: "AD", Lines: map[string][]decimal.Decimal{"ad": ad}}, nil
	}
	for i := range bars {
		highLowDiff := bars[i].High.Sub(bars[i].Low)
		var mfm decimal.Decimal
		if !highLowDiff.IsZero() {
			clv := bars[i].Close.Mul(decimal.NewFromInt(2)).Sub(bars[i].Low).Sub(bars[i].High)
			mfm = clv.Div(highLowDiff)
		}
		mfv := mfm.Mul(decimal.NewFromFloat(bars[i].Volume))
		if i == 0 {
			ad[i] = mfv
		} else {
			ad[i] = ad[i-1].Add(mfv)
		}
	}
	return &Result{DefID: "AD", Lines: map[string][]decimal.Decimal{"ad": ad}}, nil
}
