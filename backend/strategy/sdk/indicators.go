package sdk

// IndicatorSet provides technical indicator computations to strategies.
// Every method returns the indicator value at the given shift.
// shift=0 is the current (latest) bar, shift=1 is the previous bar, etc.
type IndicatorSet interface {
	// MA returns Moving Average value.
	// method: "sma", "ema", "smma", "lwma"
	MA(period, shift int, method string) float64

	// EMA is a convenience method for exponential moving average.
	EMA(period, shift int) float64

	// RSI returns Relative Strength Index.
	RSI(period, shift int) float64

	// MACD returns the MACD line.
	MACD(fastPeriod, slowPeriod, signalPeriod, shift int) float64

	// MACDSignal returns the MACD signal line.
	MACDSignal(fastPeriod, slowPeriod, signalPeriod, shift int) float64

	// ATR returns Average True Range.
	ATR(period, shift int) float64

	// Bollinger returns Bollinger Band values.
	// upper, middle, lower at the given shift.
	Bollinger(period, deviation, shift int) (upper, middle, lower float64)

	// Stochastic returns Stochastic oscillator values.
	// k, d values at the given shift.
	Stochastic(kPeriod, dPeriod, slowing, shift int) (k, d float64)

	// CCI returns Commodity Channel Index.
	CCI(period, shift int) float64

	// ADX returns Average Directional Index.
	ADX(period, shift int) float64

	// MFI returns Money Flow Index.
	MFI(period, shift int) float64

	// OBV returns On-Balance Volume.
	OBV(shift int) float64

	// SAR returns Parabolic SAR.
	SAR(step, maximum float64, shift int) float64

	// StdDev returns Standard Deviation.
	StdDev(period, shift int) float64

	// WPR returns Williams %R.
	WPR(period, shift int) float64

	// Momentum returns Momentum indicator.
	Momentum(period, shift int) float64

	// ICustom returns a custom indicator value.
	// name is the indicator name; params are the custom parameters.
	ICustom(name string, params []float64, buffer, shift int) float64
}

// IndicatorParams holds common indicator parameters.
type IndicatorParams struct {
	Period    int
	Shift     int
	Method    string  // "sma", "ema", etc.
	Deviation float64 // for Bollinger, Envelopes
	Fast      int     // for MACD
	Slow      int     // for MACD
	Signal    int     // for MACD
	Step      float64 // for SAR
	Maximum   float64 // for SAR
}
