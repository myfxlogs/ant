package runner

import (
	"github.com/shopspring/decimal"

	"alphaforge/strategy/indicators"
	"alphaforge/strategy/sdk"
)

// indicatorSet implements sdk.IndicatorSet backed by the runner's bar data.
type indicatorSet struct {
	runner *Runner
	src    *runnerBarSource
	cache  *indicators.SeriesCache
}

func (is *indicatorSet) bars() sdk.BarSeries {
	is.runner.ctx.mu.RLock()
	defer is.runner.ctx.mu.RUnlock()
	return is.runner.ctx.bars
}

func (is *indicatorSet) ensureCache() *indicators.SeriesCache {
	bars := is.bars()
	if is.src == nil {
		is.src = &runnerBarSource{bars: bars, runner: is.runner}
		is.cache = indicators.NewSeriesCache(is.src)
	}
	is.src.bars = bars
	is.cache.EnsureUpdated()
	return is.cache
}

func (is *indicatorSet) MA(period, shift int, method string, appliedPrice int) decimal.Decimal {
	if appliedPrice <= 1 {
		switch method {
		case "EMA", "ema":
			return is.ensureCache().EMADecimal(period, shift)
		case "SMMA", "smma":
			return is.ensureCache().SMMADecimal(period, shift)
		case "SMA", "sma":
			return is.ensureCache().SMADecimal(period, shift)
		case "LWMA", "lwma":
			return is.ensureCache().LWMADecimal(period, shift)
		}
	}
	return indicators.MA(is.barSource(), period, shift, method, appliedPrice)
}

func (is *indicatorSet) EMA(period, shift int) decimal.Decimal {
	return is.ensureCache().EMADecimal(period, shift)
}

func (is *indicatorSet) RSI(period, shift int, appliedPrice int) decimal.Decimal {
	if appliedPrice <= 1 {
		return is.ensureCache().RSIDecimal(period, shift)
	}
	return indicators.RSI(is.barSource(), period, shift, appliedPrice)
}

func (is *indicatorSet) MACD(fastPeriod, slowPeriod, signalPeriod, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return is.ensureCache().MACDLineDecimal(fastPeriod, slowPeriod, signalPeriod, shift)
	}
	return indicators.MACD(is.barSource(), fastPeriod, slowPeriod, shift, appliedPrice)
}

func (is *indicatorSet) MACDSignal(fastPeriod, slowPeriod, signalPeriod, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return is.ensureCache().MACDSignalDecimal(fastPeriod, slowPeriod, signalPeriod, shift)
	}
	return indicators.MACDSignal(is.barSource(), fastPeriod, slowPeriod, signalPeriod, shift, appliedPrice)
}

func (is *indicatorSet) ATR(period, shift int) decimal.Decimal {
	return is.ensureCache().ATRDecimal(period, shift)
}

func (is *indicatorSet) Bollinger(period int, deviation decimal.Decimal, appliedPrice, shift int) (upper, middle, lower decimal.Decimal) {
	return indicators.BollingerBands(is.barSource(), period, deviation, shift, appliedPrice)
}

func (is *indicatorSet) Stochastic(kPeriod, dPeriod, slowing, shift int) (k, d decimal.Decimal) {
	return indicators.Stochastic(is.barSource(), kPeriod, dPeriod, slowing, shift)
}

func (is *indicatorSet) CCI(period, shift int, appliedPrice int) decimal.Decimal {
	return indicators.CCI(is.barSource(), period, shift, appliedPrice)
}

func (is *indicatorSet) ADX(period, shift int) decimal.Decimal {
	return is.ensureCache().ADXDecimal(period, shift)
}

func (is *indicatorSet) MFI(period, shift int) decimal.Decimal {
	return indicators.MFI(is.barSource(), period, shift)
}

func (is *indicatorSet) OBV(appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return is.ensureCache().OBVDecimal(shift)
	}
	return indicators.OBV(is.barSource(), shift, appliedPrice)
}

func (is *indicatorSet) SAR(step, maximum decimal.Decimal, shift int) decimal.Decimal {
	return is.ensureCache().SARDecimal(step, maximum, shift)
}

func (is *indicatorSet) StdDev(period, shift int, method string, appliedPrice int) decimal.Decimal {
	return indicators.StdDev(is.barSource(), period, shift, method, appliedPrice)
}

func (is *indicatorSet) WPR(period, shift int) decimal.Decimal {
	return indicators.WPR(is.barSource(), period, shift)
}

func (is *indicatorSet) Momentum(period, shift int, appliedPrice int) decimal.Decimal {
	return indicators.Momentum(is.barSource(), period, shift, appliedPrice)
}

func (is *indicatorSet) ICustom(name string, params []decimal.Decimal, buffer, shift int) decimal.Decimal {
	return decimal.Zero
}

// ── Shared MQL4/MQL5 indicators ──

func (is *indicatorSet) Alligator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, appliedPrice int, shift int) (jaw, teeth, lips decimal.Decimal) {
	if appliedPrice <= 1 {
		return is.ensureCache().AlligatorDecimal(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, shift)
	}
	return indicators.Alligator(is.barSource(), jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, appliedPrice, shift)
}

func (is *indicatorSet) Ichimoku(tenkan, kijun, senkou int, shift int) (tenkanVal, kijunVal, senkouA, senkouB decimal.Decimal) {
	return indicators.Ichimoku(is.barSource(), tenkan, kijun, senkou, shift)
}

func (is *indicatorSet) Envelopes(period int, deviation decimal.Decimal, method string, appliedPrice int, shift int) (upper, lower decimal.Decimal) {
	if appliedPrice <= 1 {
		return is.ensureCache().EnvelopesDecimal(period, deviation, method, shift)
	}
	return indicators.Envelopes(is.barSource(), period, deviation, method, appliedPrice, shift)
}

func (is *indicatorSet) DeMarker(period, shift int) decimal.Decimal {
	return indicators.DeMarker(is.barSource(), period, shift)
}

func (is *indicatorSet) OsMA(fastPeriod, slowPeriod, signalPeriod, appliedPrice, shift int) decimal.Decimal {
	return decimal.NewFromFloat(is.ensureCache().OsMAWithPrice(fastPeriod, slowPeriod, signalPeriod, appliedPrice, shift))
}

func (is *indicatorSet) RVI(period, shift int) decimal.Decimal {
	return indicators.RVI(is.barSource(), period, shift)
}

func (is *indicatorSet) Force(period int, method string, appliedPrice int, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return is.ensureCache().ForceDecimal(period, method, shift)
	}
	return indicators.Force(is.barSource(), period, method, appliedPrice, shift)
}

func (is *indicatorSet) Fractals(shift int) (upper, lower decimal.Decimal) {
	return indicators.Fractals(is.barSource(), shift)
}

func (is *indicatorSet) Gator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, appliedPrice int, shift int) (upper, lower decimal.Decimal) {
	if appliedPrice <= 1 {
		return is.ensureCache().GatorDecimal(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, shift)
	}
	u, l := indicators.Gator(is.barSource(), jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, appliedPrice, shift)
	return u, l
}

func (is *indicatorSet) AC(shift int) decimal.Decimal {
	return indicators.AC(is.barSource(), shift)
}

func (is *indicatorSet) AD(shift int) decimal.Decimal {
	return is.ensureCache().ADDecimal(shift)
}

func (is *indicatorSet) AO(shift int) decimal.Decimal {
	return indicators.AO(is.barSource(), shift)
}

func (is *indicatorSet) BearsPower(period int, appliedPrice int, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return is.ensureCache().BearsPowerDecimal(period, shift)
	}
	return indicators.BearsPower(is.barSource(), period, appliedPrice, shift)
}

func (is *indicatorSet) BullsPower(period int, appliedPrice int, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return is.ensureCache().BullsPowerDecimal(period, shift)
	}
	return indicators.BullsPower(is.barSource(), period, appliedPrice, shift)
}

func (is *indicatorSet) BWMFI(shift int) decimal.Decimal {
	return indicators.BWMFI(is.barSource(), shift)
}

// ── MQL5-only indicators ──

func (is *indicatorSet) AMA(period, fastPeriod, slowPeriod, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return is.ensureCache().AMADecimal(period, fastPeriod, slowPeriod, shift)
	}
	return indicators.AMA(is.barSource(), period, fastPeriod, slowPeriod, shift, appliedPrice)
}

func (is *indicatorSet) DEMA(period, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return is.ensureCache().DEMADecimal(period, shift)
	}
	return indicators.DEMA(is.barSource(), period, shift, appliedPrice)
}

func (is *indicatorSet) TEMA(period, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return is.ensureCache().TEMADecimal(period, shift)
	}
	return indicators.TEMA(is.barSource(), period, shift, appliedPrice)
}

func (is *indicatorSet) FrAMA(period, appliedPrice, shift int) decimal.Decimal {
	return indicators.FrAMA(is.barSource(), period, shift, appliedPrice)
}

func (is *indicatorSet) VIDyA(cmoPeriod, cmoShift, maPeriod, maShift, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return is.ensureCache().VIDyADecimal(cmoPeriod, cmoShift, maPeriod, maShift, shift)
	}
	return indicators.VIDyA(is.barSource(), cmoPeriod, cmoShift, maPeriod, maShift, shift, appliedPrice)
}

func (is *indicatorSet) TriX(period, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return is.ensureCache().TriXDecimal(period, shift)
	}
	return indicators.TriX(is.barSource(), period, shift, appliedPrice)
}

func (is *indicatorSet) ADXWilder(period, shift int) decimal.Decimal {
	return is.ensureCache().ADXDecimal(period, shift)
}

func (is *indicatorSet) Chaikin(fastPeriod, slowPeriod, shift int) decimal.Decimal {
	return is.ensureCache().ChaikinDecimal(fastPeriod, slowPeriod, shift)
}

func (is *indicatorSet) Volumes(shift int) decimal.Decimal {
	return indicators.Volumes(is.barSource(), shift)
}
