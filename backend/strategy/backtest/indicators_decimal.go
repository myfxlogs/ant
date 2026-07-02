package backtest

import (
	"github.com/shopspring/decimal"

	"anttrader/strategy/indicators"
	"anttrader/strategy/sdk"
)

// ── Indicators (backtest implementation) ──────────────────────────

type btIndicators struct {
	bars   []sdk.Bar
	barIdx *int // pointer to backtestContext.barIndex, updated each bar
	src    *btBarSource
	cache  *indicators.SeriesCache
}

// visibleBars returns bars up to and including the current barIndex.
func (i *btIndicators) visibleBars() []sdk.Bar {
	end := len(i.bars)
	if i.barIdx != nil && *i.barIdx+1 < end {
		end = *i.barIdx + 1
	}
	return i.bars[:end]
}

func (i *btIndicators) ensureCache() *indicators.SeriesCache {
	visible := i.visibleBars()
	if i.src == nil {
		i.src = &btBarSource{bars: visible}
		i.cache = indicators.NewSeriesCache(i.src)
	}
	i.src.bars = visible
	i.cache.EnsureUpdated()
	return i.cache
}

func (i *btIndicators) EMA(period, shift int) decimal.Decimal {
	return i.ensureCache().EMADecimal(period, shift)
}

func (i *btIndicators) MA(period, shift int, method string, appliedPrice int) decimal.Decimal {
	if appliedPrice <= 1 {
		switch method {
		case "EMA", "ema":
			return i.ensureCache().EMADecimal(period, shift)
		case "SMMA", "smma":
			return i.ensureCache().SMMADecimal(period, shift)
		case "SMA", "sma":
			return i.ensureCache().SMADecimal(period, shift)
		case "LWMA", "lwma":
			return i.ensureCache().LWMADecimal(period, shift)
		}
	}
	return indicators.MA(i.barSource(), period, shift, method, appliedPrice)
}

func (i *btIndicators) RSI(period, shift int, appliedPrice int) decimal.Decimal {
	if appliedPrice <= 1 {
		return i.ensureCache().RSIDecimal(period, shift)
	}
	return indicators.RSI(i.barSource(), period, shift, appliedPrice)
}

func (i *btIndicators) MACD(fast, slow, signal, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return i.ensureCache().MACDLineDecimal(fast, slow, signal, shift)
	}
	return indicators.MACD(i.barSource(), fast, slow, shift, appliedPrice)
}

func (i *btIndicators) MACDSignal(fast, slow, signal, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return i.ensureCache().MACDSignalDecimal(fast, slow, signal, shift)
	}
	return indicators.MACDSignal(i.barSource(), fast, slow, signal, shift, appliedPrice)
}

func (i *btIndicators) ATR(period, shift int) decimal.Decimal {
	return i.ensureCache().ATRDecimal(period, shift)
}

func (i *btIndicators) Bollinger(period int, deviation decimal.Decimal, appliedPrice, shift int) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	return indicators.BollingerBands(i.barSource(), period, deviation, shift, appliedPrice)
}

func (i *btIndicators) Momentum(period, shift int, appliedPrice int) decimal.Decimal {
	return indicators.Momentum(i.barSource(), period, shift, appliedPrice)
}

func (i *btIndicators) StdDev(period, shift int, method string, appliedPrice int) decimal.Decimal {
	return indicators.StdDev(i.barSource(), period, shift, method, appliedPrice)
}

func (i *btIndicators) Stochastic(kPeriod, dPeriod, slowing, shift int) (decimal.Decimal, decimal.Decimal) {
	return indicators.Stochastic(i.barSource(), kPeriod, dPeriod, slowing, shift)
}

func (i *btIndicators) CCI(period, shift int, appliedPrice int) decimal.Decimal {
	return indicators.CCI(i.barSource(), period, shift, appliedPrice)
}

func (i *btIndicators) ADX(period, shift int) decimal.Decimal {
	return i.ensureCache().ADXDecimal(period, shift)
}

func (i *btIndicators) MFI(period, shift int) decimal.Decimal {
	return indicators.MFI(i.barSource(), period, shift)
}

func (i *btIndicators) OBV(appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return i.ensureCache().OBVDecimal(shift)
	}
	return indicators.OBV(i.barSource(), shift, appliedPrice)
}

func (i *btIndicators) SAR(step, maximum decimal.Decimal, shift int) decimal.Decimal {
	return i.ensureCache().SARDecimal(step, maximum, shift)
}

func (i *btIndicators) WPR(period, shift int) decimal.Decimal {
	return indicators.WPR(i.barSource(), period, shift)
}

func (i *btIndicators) ICustom(name string, params []decimal.Decimal, buffer, shift int) decimal.Decimal {
	return decimal.Zero
}

// ── Shared MQL4/MQL5 indicators ──

func (i *btIndicators) Alligator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, appliedPrice int, shift int) (jaw, teeth, lips decimal.Decimal) {
	if appliedPrice <= 1 {
		return i.ensureCache().AlligatorDecimal(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, shift)
	}
	return indicators.Alligator(i.barSource(), jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, appliedPrice, shift)
}
func (i *btIndicators) Ichimoku(tenkan, kijun, senkou int, shift int) (tenkanVal, kijunVal, senkouA, senkouB decimal.Decimal) {
	return indicators.Ichimoku(i.barSource(), tenkan, kijun, senkou, shift)
}
func (i *btIndicators) Envelopes(period int, deviation decimal.Decimal, method string, appliedPrice int, shift int) (upper, lower decimal.Decimal) {
	if appliedPrice <= 1 {
		return i.ensureCache().EnvelopesDecimal(period, deviation, method, shift)
	}
	return indicators.Envelopes(i.barSource(), period, deviation, method, appliedPrice, shift)
}
func (i *btIndicators) DeMarker(period, shift int) decimal.Decimal {
	return indicators.DeMarker(i.barSource(), period, shift)
}
func (i *btIndicators) OsMA(fastPeriod, slowPeriod, signalPeriod, appliedPrice, shift int) decimal.Decimal {
	return decimal.NewFromFloat(i.ensureCache().OsMAWithPrice(fastPeriod, slowPeriod, signalPeriod, appliedPrice, shift))
}
func (i *btIndicators) RVI(period, shift int) decimal.Decimal {
	return indicators.RVI(i.barSource(), period, shift)
}
func (i *btIndicators) Force(period int, method string, appliedPrice int, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return i.ensureCache().ForceDecimal(period, method, shift)
	}
	return indicators.Force(i.barSource(), period, method, appliedPrice, shift)
}
func (i *btIndicators) Fractals(shift int) (upper, lower decimal.Decimal) {
	return indicators.Fractals(i.barSource(), shift)
}
func (i *btIndicators) Gator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, appliedPrice int, shift int) (upper, lower decimal.Decimal) {
	if appliedPrice <= 1 {
		return i.ensureCache().GatorDecimal(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, shift)
	}
	u, l := indicators.Gator(i.barSource(), jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, appliedPrice, shift)
	return u, l
}
func (i *btIndicators) AC(shift int) decimal.Decimal {
	return indicators.AC(i.barSource(), shift)
}
func (i *btIndicators) AD(shift int) decimal.Decimal {
	return i.ensureCache().ADDecimal(shift)
}
func (i *btIndicators) AO(shift int) decimal.Decimal {
	return indicators.AO(i.barSource(), shift)
}
func (i *btIndicators) BearsPower(period int, appliedPrice int, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return i.ensureCache().BearsPowerDecimal(period, shift)
	}
	return indicators.BearsPower(i.barSource(), period, appliedPrice, shift)
}
func (i *btIndicators) BullsPower(period int, appliedPrice int, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return i.ensureCache().BullsPowerDecimal(period, shift)
	}
	return indicators.BullsPower(i.barSource(), period, appliedPrice, shift)
}
func (i *btIndicators) BWMFI(shift int) decimal.Decimal {
	return indicators.BWMFI(i.barSource(), shift)
}

// ── MQL5-only indicators ──

func (i *btIndicators) AMA(period, fastPeriod, slowPeriod, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return i.ensureCache().AMADecimal(period, fastPeriod, slowPeriod, shift)
	}
	return indicators.AMA(i.barSource(), period, fastPeriod, slowPeriod, shift, appliedPrice)
}
func (i *btIndicators) DEMA(period, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return i.ensureCache().DEMADecimal(period, shift)
	}
	return indicators.DEMA(i.barSource(), period, shift, appliedPrice)
}
func (i *btIndicators) TEMA(period, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return i.ensureCache().TEMADecimal(period, shift)
	}
	return indicators.TEMA(i.barSource(), period, shift, appliedPrice)
}
func (i *btIndicators) FrAMA(period, appliedPrice, shift int) decimal.Decimal {
	return indicators.FrAMA(i.barSource(), period, shift, appliedPrice)
}
func (i *btIndicators) VIDyA(cmoPeriod, cmoShift, maPeriod, maShift, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return i.ensureCache().VIDyADecimal(cmoPeriod, cmoShift, maPeriod, maShift, shift)
	}
	return indicators.VIDyA(i.barSource(), cmoPeriod, cmoShift, maPeriod, maShift, shift, appliedPrice)
}
func (i *btIndicators) TriX(period, appliedPrice, shift int) decimal.Decimal {
	if appliedPrice <= 1 {
		return i.ensureCache().TriXDecimal(period, shift)
	}
	return indicators.TriX(i.barSource(), period, shift, appliedPrice)
}
func (i *btIndicators) ADXWilder(period, shift int) decimal.Decimal {
	return i.ensureCache().ADXDecimal(period, shift)
}
func (i *btIndicators) Chaikin(fastPeriod, slowPeriod, shift int) decimal.Decimal {
	return i.ensureCache().ChaikinDecimal(fastPeriod, slowPeriod, shift)
}
func (i *btIndicators) Volumes(shift int) decimal.Decimal {
	return indicators.Volumes(i.barSource(), shift)
}
