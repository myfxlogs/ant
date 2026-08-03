package sdk

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestDetectLanguage_Empty(t *testing.T) {
	if got := DetectLanguage(""); got != LangUnknown {
		t.Errorf("DetectLanguage(\"\") = %v, want %v", got, LangUnknown)
	}
}

func TestDetectLanguage_Go(t *testing.T) {
	code := `package mystrategy
import "alphaforge/strategy/sdk"
func init() {}
`
	if got := DetectLanguage(code); got != LangGo {
		t.Errorf("DetectLanguage(go) = %v, want %v", got, LangGo)
	}
}

func TestDetectLanguage_MQL(t *testing.T) {
	tests := []string{
		`void OnTick() { }`,
		`void OnBar() { }`,
		`int OnInit() { return 0; }`,
		`void OnDeinit() { }`,
		`void OnTimer() { }`,
		`#property strict`,
		`extern int MagicNumber = 123;`,
		`input int Period = 14;`,
	}
	for _, code := range tests {
		if got := DetectLanguage(code); got != LangMQL {
			t.Errorf("DetectLanguage(%q) = %v, want %v", code, got, LangMQL)
		}
	}
}

func TestDetectLanguage_Python(t *testing.T) {
	tests := []string{
		`class MyStrategy(StrategyBase): pass`,
		`from alphaforge import Strategy`,
		`def on_init(ctx): pass`,
		`def on_bar(ctx, tf): pass`,
		`def on_tick(ctx, bid, ask): pass`,
		`def on_deinit(ctx): pass`,
	}
	for _, code := range tests {
		if got := DetectLanguage(code); got != LangPython {
			t.Errorf("DetectLanguage(%q) = %v, want %v", code, got, LangPython)
		}
	}
}

func TestDetectLanguage_Unknown(t *testing.T) {
	if got := DetectLanguage("hello world"); got != LangUnknown {
		t.Errorf("DetectLanguage(\"hello world\") = %v, want %v", got, LangUnknown)
	}
}

func TestDetectLanguage_GoTakesPrecedenceOverMQL(t *testing.T) {
	code := `package strat
import "alphaforge/strategy/sdk"
// void OnTick() { }
`
	if got := DetectLanguage(code); got != LangGo {
		t.Errorf("Go should take precedence over MQL markers, got %v", got)
	}
}

func TestIsMQL(t *testing.T) {
	if !IsMQL(`void OnTick() {}`) {
		t.Error("IsMQL should return true for MQL code")
	}
	if IsMQL(`package foo`) {
		t.Error("IsMQL should return false for non-MQL code")
	}
}

func TestIsPython(t *testing.T) {
	if !IsPython(`class S(StrategyBase): pass`) {
		t.Error("IsPython should return true for Python code")
	}
	if IsPython(`void OnTick() {}`) {
		t.Error("IsPython should return false for MQL code")
	}
}

func TestIsGo(t *testing.T) {
	if !IsGo(`package x
import "alphaforge/strategy/sdk"`) {
		t.Error("IsGo should return true for Go code")
	}
	if IsGo(`void OnTick() {}`) {
		t.Error("IsGo should return false for MQL code")
	}
}

func TestBarsToSlice_BasicAccess(t *testing.T) {
	bars := []Bar{
		{Open: dec("1.0"), High: dec("2.0"), Low: dec("0.5"), Close: dec("1.5"), Volume: 100, Timestamp: 1000},
		{Open: dec("1.5"), High: dec("2.5"), Low: dec("1.0"), Close: dec("2.0"), Volume: 200, Timestamp: 2000},
		{Open: dec("2.0"), High: dec("3.0"), Low: dec("1.5"), Close: dec("2.5"), Volume: 300, Timestamp: 3000},
	}
	series := BarsToSlice(bars)

	if series.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", series.Len())
	}

	// shift=0 is the latest bar (last element).
	if !series.Close(0).Equal(dec("2.5")) {
		t.Errorf("Close(0) = %s, want 2.5", series.Close(0))
	}
	if !series.Open(0).Equal(dec("2.0")) {
		t.Errorf("Open(0) = %s, want 2.0", series.Open(0))
	}
	if !series.High(0).Equal(dec("3.0")) {
		t.Errorf("High(0) = %s, want 3.0", series.High(0))
	}
	if !series.Low(0).Equal(dec("1.5")) {
		t.Errorf("Low(0) = %s, want 1.5", series.Low(0))
	}
	if series.Volume(0) != 300 {
		t.Errorf("Volume(0) = %d, want 300", series.Volume(0))
	}
	if series.Time(0) != 3000 {
		t.Errorf("Time(0) = %d, want 3000", series.Time(0))
	}

	// shift=1 is the previous bar.
	if !series.Close(1).Equal(dec("2.0")) {
		t.Errorf("Close(1) = %s, want 2.0", series.Close(1))
	}

	// shift=2 is the oldest bar.
	if !series.Close(2).Equal(dec("1.5")) {
		t.Errorf("Close(2) = %s, want 1.5", series.Close(2))
	}
}

func TestBarsToSlice_OutOfBounds(t *testing.T) {
	bars := []Bar{
		{Close: dec("1.0"), Timestamp: 1000},
	}
	series := BarsToSlice(bars)

	// shift beyond length returns zero Bar.
	got := series.Close(5)
	if !got.Equal(decimal.Zero) {
		t.Errorf("Close(5) out of bounds = %s, want 0", got)
	}
	got = series.Open(-1)
	if !got.Equal(decimal.Zero) {
		t.Errorf("Open(-1) out of bounds = %s, want 0", got)
	}
}

func TestBarsToSlice_Empty(t *testing.T) {
	series := BarsToSlice(nil)
	if series.Len() != 0 {
		t.Errorf("Len() = %d, want 0", series.Len())
	}
	if series.Close(0).Cmp(decimal.Zero) != 0 {
		t.Errorf("Close(0) on empty = %s, want 0", series.Close(0))
	}
}

func TestBarsToSlice_Slice(t *testing.T) {
	bars := []Bar{
		{Close: dec("1.0"), Timestamp: 1000},
		{Close: dec("2.0"), Timestamp: 2000},
		{Close: dec("3.0"), Timestamp: 3000},
		{Close: dec("4.0"), Timestamp: 4000},
		{Close: dec("5.0"), Timestamp: 5000},
	}
	series := BarsToSlice(bars)

	// Slice(2) returns the last 2 bars.
	sliced := series.Slice(2)
	if sliced.Len() != 2 {
		t.Fatalf("Slice(2).Len() = %d, want 2", sliced.Len())
	}
	if !sliced.Close(0).Equal(dec("5.0")) {
		t.Errorf("Slice(2).Close(0) = %s, want 5.0", sliced.Close(0))
	}
	if !sliced.Close(1).Equal(dec("4.0")) {
		t.Errorf("Slice(2).Close(1) = %s, want 4.0", sliced.Close(1))
	}

	// Slice with n >= len returns all bars.
	all := series.Slice(10)
	if all.Len() != 5 {
		t.Errorf("Slice(10).Len() = %d, want 5", all.Len())
	}
}

func TestBarsToSlice_TimeframeAndSymbol(t *testing.T) {
	series := BarsToSlice([]Bar{{Close: dec("1.0")}})
	if series.Timeframe() != "" {
		t.Errorf("Timeframe() = %q, want empty", series.Timeframe())
	}
	if series.Symbol() != "" {
		t.Errorf("Symbol() = %q, want empty", series.Symbol())
	}
}

func TestSpreadDecimal(t *testing.T) {
	si := SymbolInfo{
		Point:  dec("0.00001"),
		Spread: 15,
	}
	got := si.SpreadDecimal()
	want := dec("0.00015")
	if !got.Equal(want) {
		t.Errorf("SpreadDecimal() = %s, want %s", got, want)
	}
}

func TestSpreadDecimal_ZeroSpread(t *testing.T) {
	si := SymbolInfo{
		Point:  dec("0.001"),
		Spread: 0,
	}
	got := si.SpreadDecimal()
	if !got.Equal(decimal.Zero) {
		t.Errorf("SpreadDecimal() with 0 spread = %s, want 0", got)
	}
}

func TestSpreadDecimal_ZeroPoint(t *testing.T) {
	si := SymbolInfo{
		Point:  decimal.Zero,
		Spread: 10,
	}
	got := si.SpreadDecimal()
	if !got.Equal(decimal.Zero) {
		t.Errorf("SpreadDecimal() with 0 point = %s, want 0", got)
	}
}

func dec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}
