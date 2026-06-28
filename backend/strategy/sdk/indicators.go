package sdk

import "github.com/shopspring/decimal"

// IndicatorSet provides technical indicator computations to strategies.
// Every method returns the indicator value at the given shift.
// shift=0 is the current (latest) bar, shift=1 is the previous bar, etc.
type IndicatorSet interface {
	// MA returns Moving Average value.
	// method: "sma", "ema", "smma", "lwma"
	MA(period, shift int, method string) decimal.Decimal

	// EMA is a convenience method for exponential moving average.
	EMA(period, shift int) decimal.Decimal

	// RSI returns Relative Strength Index.
	RSI(period, shift int) decimal.Decimal

	// MACD returns the MACD line.
	MACD(fastPeriod, slowPeriod, signalPeriod, shift int) decimal.Decimal

	// MACDSignal returns the MACD signal line.
	MACDSignal(fastPeriod, slowPeriod, signalPeriod, shift int) decimal.Decimal

	// ATR returns Average True Range.
	ATR(period, shift int) decimal.Decimal

	// Bollinger returns Bollinger Band values.
	// upper, middle, lower at the given shift.
	// deviation is the number of standard deviations (typically 2.0).
	Bollinger(period int, deviation decimal.Decimal, shift int) (upper, middle, lower decimal.Decimal)

	// Stochastic returns Stochastic oscillator values.
	// k, d values at the given shift.
	Stochastic(kPeriod, dPeriod, slowing, shift int) (k, d decimal.Decimal)

	// CCI returns Commodity Channel Index.
	CCI(period, shift int) decimal.Decimal

	// ADX returns Average Directional Index.
	ADX(period, shift int) decimal.Decimal

	// MFI returns Money Flow Index.
	MFI(period, shift int) decimal.Decimal

	// OBV returns On-Balance Volume.
	OBV(shift int) decimal.Decimal

	// SAR returns Parabolic SAR.
	SAR(step, maximum decimal.Decimal, shift int) decimal.Decimal

	// StdDev returns Standard Deviation.
	StdDev(period, shift int) decimal.Decimal

	// WPR returns Williams %R.
	WPR(period, shift int) decimal.Decimal

	// Momentum returns Momentum indicator.
	Momentum(period, shift int) decimal.Decimal

	// ICustom returns a custom indicator value.
	// name is the indicator name; params are the custom parameters.
	ICustom(name string, params []decimal.Decimal, buffer, shift int) decimal.Decimal

	// ── Shared MQL4/MQL5 indicators (SDK stubs) ──

	// Alligator returns Alligator jaw/teeth/lips values.
	Alligator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, appliedPrice int, shift int) (jaw, teeth, lips decimal.Decimal)

	// Ichimoku returns Ichimoku tenkan/kijun/senkouA/senkouB values.
	Ichimoku(tenkan, kijun, senkou int, shift int) (tenkanVal, kijunVal, senkouA, senkouB decimal.Decimal)

	// Envelopes returns Envelopes upper/lower values.
	Envelopes(period int, deviation decimal.Decimal, method string, appliedPrice int, shift int) (upper, lower decimal.Decimal)

	// DeMarker returns DeMarker oscillator value.
	DeMarker(period, shift int) decimal.Decimal

	// OsMA returns OsMA (MACD histogram) value.
	OsMA(fastPeriod, slowPeriod, signalPeriod, appliedPrice, shift int) decimal.Decimal

	// RVI returns Relative Vigor Index value.
	RVI(period, shift int) decimal.Decimal

	// Force returns Force Index value.
	Force(period int, method string, appliedPrice int, shift int) decimal.Decimal

	// Fractals returns Fractals upper/lower values.
	Fractals(shift int) (upper, lower decimal.Decimal)

	// Gator returns Gator oscillator upper/lower values.
	Gator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, appliedPrice int, shift int) (upper, lower decimal.Decimal)

	// AC returns Accelerator oscillator value.
	AC(shift int) decimal.Decimal

	// AD returns Accumulation/Distribution value.
	AD(shift int) decimal.Decimal

	// AO returns Awesome oscillator value.
	AO(shift int) decimal.Decimal

	// BearsPower returns Bears Power value.
	BearsPower(period int, appliedPrice int, shift int) decimal.Decimal

	// BullsPower returns Bulls Power value.
	BullsPower(period int, appliedPrice int, shift int) decimal.Decimal

	// BWMFI returns Bill Williams Market Facilitation Index value.
	BWMFI(shift int) decimal.Decimal

	// ── MQL5-only indicators ──

	// AMA returns Adaptive Moving Average value.
	AMA(period, fastPeriod, slowPeriod, shift int) decimal.Decimal

	// DEMA returns Double Exponential Moving Average value.
	DEMA(period, shift int) decimal.Decimal

	// TEMA returns Triple Exponential Moving Average value.
	TEMA(period, shift int) decimal.Decimal

	// FrAMA returns Fractal Adaptive Moving Average value.
	FrAMA(period, shift int) decimal.Decimal

	// VIDyA returns Variable Index Dynamic Average value.
	VIDyA(cmoPeriod, cmoShift, maPeriod, maShift, shift int) decimal.Decimal

	// TriX returns Triple Exponential Average value.
	TriX(period, shift int) decimal.Decimal

	// ADXWilder returns ADX Wilder value.
	ADXWilder(period, shift int) decimal.Decimal

	// Chaikin returns Chaikin Oscillator value.
	Chaikin(fastPeriod, slowPeriod, shift int) decimal.Decimal

	// Volumes returns tick/real volume value.
	Volumes(shift int) decimal.Decimal
}

// IndicatorParams holds common indicator parameters.
type IndicatorParams struct {
	Period    int
	Shift     int
	Method    string            // "sma", "ema", etc.
	Deviation decimal.Decimal   // for Bollinger, Envelopes
	Fast      int               // for MACD
	Slow      int               // for MACD
	Signal    int               // for MACD
	Step      decimal.Decimal   // for SAR
	Maximum   decimal.Decimal   // for SAR
}
