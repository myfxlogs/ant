package indicator

import (
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"github.com/shopspring/decimal"
)

var DefMFI = Def{
	ID:   "MFI",
	Name: "Money Flow Index",
	Kind: KindSubPane,
	Params: []Param{
		{Key: "length", Label: "Period", Type: "int", Default: 14, Min: 1, Max: 500, Step: 1},
	},
	Defaults: map[string]float64{"length": 14},
}

func computeMFI(bars []mdtick.Bar, params map[string]float64) (*Result, error) {
	n := int(getParam(params, "length", 14))
	mfi := make([]decimal.Decimal, len(bars))
	if len(bars) <= n {
		return &Result{DefID: "MFI", Lines: map[string][]decimal.Decimal{"mfi": mfi}}, nil
	}
	tp := make([]decimal.Decimal, len(bars))
	mf := make([]decimal.Decimal, len(bars))
	for i := range bars {
		tp[i] = bars[i].High.Add(bars[i].Low).Add(bars[i].Close).Div(decimal.NewFromInt(3))
		mf[i] = tp[i].Mul(decimal.NewFromFloat(bars[i].Volume))
	}
	for i := n; i < len(bars); i++ {
		var posFlow, negFlow decimal.Decimal
		for j := i - n + 1; j <= i; j++ {
			if j > 0 && tp[j].GreaterThan(tp[j-1]) {
				posFlow = posFlow.Add(mf[j])
			} else if j > 0 && tp[j].LessThan(tp[j-1]) {
				negFlow = negFlow.Add(mf[j])
			}
		}
		if !negFlow.IsZero() {
			mr := posFlow.Div(negFlow)
			mfi[i] = decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(mr)))
		} else if !posFlow.IsZero() {
			mfi[i] = decimal.NewFromInt(100)
		}
	}
	return &Result{DefID: "MFI", Lines: map[string][]decimal.Decimal{"mfi": mfi}}, nil
}

var DefADX = Def{
	ID:   "ADX",
	Name: "Average Directional Index",
	Kind: KindSubPane,
	Params: []Param{
		{Key: "length", Label: "Period", Type: "int", Default: 14, Min: 1, Max: 500, Step: 1},
	},
	Defaults: map[string]float64{"length": 14},
}

func computeADX(bars []mdtick.Bar, params map[string]float64) (*Result, error) {
	n := int(getParam(params, "length", 14))
	ohlcv := toOHLCV(bars)
	adx := make([]decimal.Decimal, len(bars))
	pdi := make([]decimal.Decimal, len(bars))
	ndi := make([]decimal.Decimal, len(bars))
	if len(bars) <= n {
		return &Result{DefID: "ADX", Lines: map[string][]decimal.Decimal{"adx": adx, "pdi": pdi, "ndi": ndi}}, nil
	}
	tr := trueRange(ohlcv)
	pdm := make([]decimal.Decimal, len(bars))
	ndm := make([]decimal.Decimal, len(bars))
	for i := 1; i < len(bars); i++ {
		up := ohlcv[i].High.Sub(ohlcv[i-1].High)
		down := ohlcv[i-1].Low.Sub(ohlcv[i].Low)
		if up.GreaterThan(down) && up.GreaterThan(decimal.Zero) {
			pdm[i] = up
		}
		if down.GreaterThan(up) && down.GreaterThan(decimal.Zero) {
			ndm[i] = down
		}
	}
	trEma := ema(tr, n)
	pdmEma := ema(pdm, n)
	ndmEma := ema(ndm, n)
	for i := n - 1; i < len(bars); i++ {
		if !trEma[i].IsZero() {
			pdi[i] = pdmEma[i].Div(trEma[i]).Mul(decimal.NewFromInt(100))
			ndi[i] = ndmEma[i].Div(trEma[i]).Mul(decimal.NewFromInt(100))
			sumDI := pdi[i].Add(ndi[i])
			if !sumDI.IsZero() {
				dx := pdi[i].Sub(ndi[i]).Abs().Div(sumDI).Mul(decimal.NewFromInt(100))
				if i == n-1 {
					adx[i] = dx
				} else {
					adx[i] = adx[i-1].Mul(decimal.NewFromFloat(float64(n-1))).Add(dx).Div(decimal.NewFromInt(int64(n)))
				}
			}
		}
	}
	return &Result{DefID: "ADX", Lines: map[string][]decimal.Decimal{"adx": adx, "pdi": pdi, "ndi": ndi}}, nil
}

var DefADOSC = Def{
	ID:   "ADOSC",
	Name: "Accumulation/Distribution Oscillator",
	Kind: KindSubPane,
	Params: []Param{
		{Key: "fast", Label: "Fast Period", Type: "int", Default: 3, Min: 1, Max: 100, Step: 1},
		{Key: "slow", Label: "Slow Period", Type: "int", Default: 10, Min: 1, Max: 100, Step: 1},
	},
	Defaults: map[string]float64{"fast": 3, "slow": 10},
}

func computeADOSC(bars []mdtick.Bar, params map[string]float64) (*Result, error) {
	fast := int(getParam(params, "fast", 3))
	slow := int(getParam(params, "slow", 10))
	adResult, _ := computeAD(bars, nil)
	adLine := adResult.Lines["ad"]
	emaFast := ema(adLine, fast)
	emaSlow := ema(adLine, slow)
	adosc := make([]decimal.Decimal, len(bars))
	for i := range bars {
		adosc[i] = emaFast[i].Sub(emaSlow[i])
	}
	return &Result{DefID: "ADOSC", Lines: map[string][]decimal.Decimal{"adosc": adosc}}, nil
}
