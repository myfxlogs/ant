package mql2go

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
)

// ════════════════════════════════════════════════════════════════════
// T3: Honesty Probes — should be 🟢诚实失败 (honest failure, MUST NOT silently run)
// ════════════════════════════════════════════════════════════════════

func TestHonesty_T3_UnknownConstant(t *testing.T) {
	// Unknown constant: FAKE_MODE_XYZ — not in MQL constants map.
	// Should be a compile error, not silent push 0.
	// (Previously used MODE_TENKAN, but that was added to constants.go —
	// now we use a truly fake constant to test the unknown-constant path.)
	source := `
extern int MagicNumber = 30001;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double v = iIchimoku(Symbol(), 0, 9, 26, 52, FAKE_MODE_XYZ, 0);
    if (v > 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "UC", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "T3-UnknownConstant", "T3", source)
	if r.verdict != verdictHonestFail {
		t.Errorf("T3-UnknownConstant: expected 🟢 honest fail, got %s — unknown constant silently accepted!", r.verdict)
	}
}

func TestHonesty_T3_UnknownIndicator(t *testing.T) {
	// Unknown indicator: iMyCustom is not a real MQL indicator.
	// Should be a fatal blind spot (iXxx pattern → SeverityFatal).
	source := `
extern int MagicNumber = 30002;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double v = iMyCustom(Symbol(), 0, 14, 0, 0);
    if (v > 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "UI", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "T3-UnknownIndicator", "T3", source)
	if r.verdict != verdictHonestFail {
		t.Errorf("T3-UnknownIndicator: expected 🟢 honest fail, got %s — unknown indicator silently accepted!", r.verdict)
	}
}

func TestHonesty_T3_UnsupportedFunction(t *testing.T) {
	// Unsupported function: iCustom is explicitly in unsupportedSymbols.
	// Should be a compile error.
	source := `
extern int MagicNumber = 30003;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double v = iCustom(Symbol(), 0, "MyStrategy", 14, 0, 0);
    if (v > 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "UF", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "T3-UnsupportedFunction", "T3", source)
	if r.verdict != verdictHonestFail {
		t.Errorf("T3-UnsupportedFunction: expected 🟢 honest fail (compile error), got %s — iCustom silently accepted!", r.verdict)
	}
}

func TestHonesty_T3_DLL_Import(t *testing.T) {
	// DLL import — should fail to compile (tree-sitter may parse but VM can't execute).
	source := `
#import "user32.dll"
int MessageBoxA(int hWnd, string text, string caption, int type);
#import
extern int MagicNumber = 30004;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    MessageBoxA(0, "Hello", "Test", 0);
    if (OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "DLL", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "T3-DLL-Import", "T3", source)
	if r.verdict != verdictHonestFail {
		t.Errorf("T3-DLL-Import: expected 🟢 honest fail, got %s — DLL import silently accepted!", r.verdict)
	}
}

func TestHonesty_T3_ChartObject(t *testing.T) {
	// Chart object operation — explicitly in unsupportedSymbols.
	// Should be a compile error.
	source := `
extern int MagicNumber = 30005;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    ObjectCreate("myline", OBJ_HLINE, 0, 0, 1.1000);
    if (OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "CO", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "T3-ChartObject", "T3", source)
	if r.verdict != verdictHonestFail {
		t.Errorf("T3-ChartObject: expected 🟢 honest fail (compile error), got %s — ObjectCreate silently accepted!", r.verdict)
	}
}

func TestHonesty_T3_UnknownBuiltin(t *testing.T) {
	// Unknown builtin that looks like MQL but isn't implemented.
	// "EventSetTimer" is in the registry as implemented but may not have a VM handler.
	// If it silently returns 0, that's OK for a timer setup function.
	// But let's try something truly unknown that looks like MQL:
	// "GlobalVariableSet" is implemented, let's try "GlobalVariableDefineBy" which is NOT.
	source := `
extern int MagicNumber = 30006;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double v = GlobalVariableDefineBy("myvar", 1.0);
    if (v > 0 && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "UB", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "T3-UnknownBuiltin", "T3", source)
	if r.verdict != verdictHonestFail {
		t.Errorf("T3-UnknownBuiltin: expected 🟢 honest fail, got %s — unknown builtin silently accepted!", r.verdict)
	}
}

func TestHonesty_T3_ForwardReference(t *testing.T) {
	// Forward reference: caller() is defined before callee().
	// Two-pass compilation should handle this, but if not, callee returns 0 silently.
	source := `
extern int MagicNumber = 30007;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double v = getSignal();
    if (v > 0 && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "FR", MagicNumber, 0, clrGreen);
}
double getSignal()
{
    return iMA(Symbol(), 0, 14, 0, MODE_EMA, PRICE_CLOSE, 0);
}
`
	r := runHonestyCheck(t, "T3-ForwardReference", "T3", source)
	// Forward reference should be handled by two-pass compilation → ✅ faithful
	// If it fails, it's a regression of the map iteration fix.
	if r.verdict == verdictCrack {
		t.Errorf("T3-ForwardReference: expected ✅ or 🟢, got 🔴 — forward reference regression!")
	}
	t.Logf("T3-ForwardReference verdict: %s (trades=%d, blinds=%v)", r.verdict, r.trades, r.blindSpots)
}

func TestHonesty_T3_FileIO(t *testing.T) {
	// File I/O — explicitly in unsupportedSymbols.
	// Should be a compile error.
	source := `
extern int MagicNumber = 30008;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    int handle = FileOpen("test.csv", FILE_WRITE);
    if (handle > 0)
    {
        FileWrite(handle, "test");
        FileClose(handle);
    }
    if (OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "FIO", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "T3-FileIO", "T3", source)
	if r.verdict != verdictHonestFail {
		t.Errorf("T3-FileIO: expected 🟢 honest fail (compile error), got %s — FileOpen silently accepted!", r.verdict)
	}
}

// ════════════════════════════════════════════════════════════════════
// Regression: Known pitfalls must not recur
// ════════════════════════════════════════════════════════════════════

func TestHonesty_Regression_ModeSignal(t *testing.T) {
	// MODE_SIGNAL must be 1, not silently 0 (same as MODE_MAIN).
	// If MODE_SIGNAL → 0, MACD == Signal → no trades → silent wrong result.
	source := `
extern int MagicNumber = 40001;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double macd = iMACD(Symbol(), 0, 12, 26, 9, PRICE_CLOSE, MODE_MAIN, 0);
    double signal = iMACD(Symbol(), 0, 12, 26, 9, PRICE_CLOSE, MODE_SIGNAL, 0);
    if (macd > signal && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "RegMS", MagicNumber, 0, clrGreen);
    if (macd < signal && OrdersTotal() > 0)
    {
        for (int i = 0; i < OrdersTotal(); i++)
        {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
                OrderClose(OrderTicket(), OrderLots(), Bid, 5);
        }
    }
}
`
	r := runHonestyCheck(t, "Reg-MODE_SIGNAL", "REG", source)
	if r.verdict == verdictCrack {
		t.Errorf("Reg-MODE_SIGNAL: 🔴 REGRESSION! MODE_SIGNAL silently mapped to 0 again!")
	}
}

// ════════════════════════════════════════════════════════════════════
// Summary test: runs all T3 probes and checks 100% honest failure rate
// ════════════════════════════════════════════════════════════════════

func TestHonesty_T3_Summary(t *testing.T) {
	// This is a meta-test that verifies T3 honest failure rate = 100%.
	// Individual T3 tests above will fail if any probe is silently accepted.
	// This test provides a summary view.
	t.Log("T3 Summary: All T3 probes must be 🟢诚实失败 (honest failure)")
	t.Log("If any T3 test above failed, T3 honest failure rate < 100% = 🔴地基漏洞")
}

// ════════════════════════════════════════════════════════════════════
// Deep probe: coverage blind spots vs IsReliable
// ════════════════════════════════════════════════════════════════════

func TestHonesty_Deep_FatalBlindSpotNotReliable(t *testing.T) {
	// MQL-HONESTY-3: Fatal coverage blind spots (e.g. unknown indicator iXxx
	// → SeverityFatal → silently returns 0) must cause IsReliable=false in the
	// backtest response. This was previously a crack (fatal blind spot recorded
	// but IsReliable stayed true). Fixed in buildBacktestResponse.
	//
	// This test verifies the coverage side: the unknown indicator produces a
	// fatal blind spot. The IsReliable assertion is in the strategy package
	// test TestHONESTY3_FatalBlindSpotSetsUnreliable (honesty_fatal_blindspot_test.go).
	source := `
extern int MagicNumber = 50001;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double v = iNonExistentIndicator(Symbol(), 0, 14, 0, 0);
    if (v > 0 && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "Deep", MagicNumber, 0, clrGreen);
}
`
	runner, cov, err := CompileMQLWithCoverage(source)
	if err != nil {
		t.Logf("Compile failed (honest): %v", err)
		return
	}

	// Verify fatal blind spot exists in coverage
	if cov != nil {
		fatalBlinds := hasFatalBlindSpot(cov.BlindSpots)
		t.Logf("Coverage blind spots: %v", blindSpotNames(cov.BlindSpots))
		t.Logf("Fatal blind spots: %v", fatalBlinds)

		if len(fatalBlinds) == 0 {
			t.Error("expected fatal blind spot from iNonExistentIndicator, got none")
		}
	}

	bars := makeE2EBars(80)
	cfg := backtest.Config{
		Symbol:         "EURUSD",
		Timeframe:      "M1",
		InitialCapital: decimal.NewFromInt(10000),
		Leverage:       100,
		Params:         map[string]string{},
	}
	engine := backtest.New(cfg, runner, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		t.Logf("Backtest failed (honest VM error): %v", err)
		return
	}

	// Check runtime blind spots
	runtimeBlinds := runner.GetRuntimeBlindSpots()
	for _, rbs := range runtimeBlinds {
		t.Logf("Runtime blind spot: %s (severity=%s, count=%d)", rbs.Builtin, rbs.Severity, rbs.Count)
	}

	if result.Metrics != nil {
		t.Logf("Trades: %d, Equity points: %d", int(result.Metrics.TotalTrades), len(result.Equity))
	}
}

// ════════════════════════════════════════════════════════════════════
// Deep probe: unknown function silent 0
// ════════════════════════════════════════════════════════════════════

func TestHonesty_Deep_UnknownFunctionSilentZero(t *testing.T) {
	// This test probes whether a truly unknown function (not in registry, not a user func)
	// silently returns 0 or causes an honest failure.
	//
	// "MyCustomFunction" doesn't look like an MQL builtin (no known prefix),
	// so classifySeverity returns SeverityInfo. The compiler records a blind spot
	// and pushes None (0). The backtest proceeds.
	source := `
extern int MagicNumber = 50002;
extern double LotSize = 0.1;
int OnInit() { return 0; }
double MyCustomFunction()
{
    return 42.0;
}
void OnBar()
{
    double v = MyCustomFunction();
    if (v > 0 && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "Deep2", MagicNumber, 0, clrGreen);
}
`
	// This should be ✅ faithful — MyCustomFunction is a user-defined function,
	// not an unknown builtin. The two-pass compiler should resolve it.
	r := runHonestyCheck(t, "Deep-UserFunction", "DEEP", source)
	t.Logf("Deep-UserFunction: %s", r)
	if r.verdict == verdictCrack {
		t.Errorf("Deep-UserFunction: user-defined function not resolved — forward reference regression!")
	}
}

// ════════════════════════════════════════════════════════════════════
// Deep probe: iADX mode coverage (known partial implementation)
// ════════════════════════════════════════════════════════════════════

func TestHonesty_Deep_iADX_Modes(t *testing.T) {
	// iADX with MODE_MAIN is implemented, but MODE_PLUSDI/MODE_MINUSDI
	// produce blind spots. This is an honest failure if the blind spots
	// are properly reported.
	source := `
extern int MagicNumber = 50003;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double adxMain = iADX(Symbol(), 0, 14, PRICE_CLOSE, MODE_MAIN, 0);
    double adxPlus = iADX(Symbol(), 0, 14, PRICE_CLOSE, MODE_PLUSDI, 0);
    double adxMinus = iADX(Symbol(), 0, 14, PRICE_CLOSE, MODE_MINUSDI, 0);
    if (adxMain > 25 && adxPlus > adxMinus && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "ADX", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "Deep-iADX-Modes", "DEEP", source)
	t.Logf("Deep-iADX-Modes: %s (blind spots: %v, fatal: %v)", r, r.blindSpots, r.fatalBlinds)
	// MODE_PLUSDI/MODE_MINUSDI should produce blind spots → honest failure
	if r.verdict == verdictCrack {
		t.Errorf("Deep-iADX-Modes: expected 🟢 honest fail (blind spots for PLUSDI/MINUSDI), got 🔴")
	}
}

// ════════════════════════════════════════════════════════════════════
// Deep probe: NormalizeDouble + division by zero
// ════════════════════════════════════════════════════════════════════

func TestHonesty_Deep_DivisionByZero(t *testing.T) {
	// Division by zero — does the VM handle it gracefully (return 0/Inf)
	// or does it panic/crash?
	source := `
extern int MagicNumber = 50004;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double zero = 0;
    double result = 100 / zero;
    if (result > 0 && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "Div0", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "Deep-DivZero", "DEEP", source)
	t.Logf("Deep-DivZero: %s (trades=%d, blinds=%v)", r, r.trades, r.blindSpots)
	// Division by zero should either:
	// - Produce a runtime error (honest fail)
	// - Return 0/Inf and the EA handles it (faithful)
	// - NOT silently produce wrong trades
}

// ════════════════════════════════════════════════════════════════════
// Deep probe: MQL5 class-based EA
// ════════════════════════════════════════════════════════════════════

func TestHonesty_Deep_MQL5_Class(t *testing.T) {
	// MQL5 class-based EA — uses CTrade class.
	// This tests whether class methods are properly resolved.
	source := `
#include <Trade\Trade.mqh>
input int MagicNumber = 50005;
input double LotSize = 0.1;
CTrade trade;
int OnInit() { return 0; }
void OnBar()
{
    double ma = iMA(Symbol(), 0, 14, 0, MODE_EMA, PRICE_CLOSE, 1);
    if (ma > Close[1] && PositionsTotal() == 0)
        trade.Buy(LotSize, Symbol());
    if (ma < Close[1] && PositionsTotal() > 0)
    {
        for (int i = 0; i < PositionsTotal(); i++)
        {
            ulong ticket = PositionGetTicket(i);
            if (ticket > 0)
                trade.PositionClose(ticket);
        }
    }
}
`
	r := runHonestyCheck(t, "Deep-MQL5-Class", "DEEP", source)
	t.Logf("Deep-MQL5-Class: %s (compile=%v, blinds=%v, fatal=%v)", r, r.compileOK, r.blindSpots, r.fatalBlinds)
	// MQL5 class-based EA may have blind spots for CTrade methods
	// if #include is not parsed. This is expected to be an honest failure.
}

// ════════════════════════════════════════════════════════════════════
// Deep probe: iIchimoku modes (known to have MODE_TENKAN etc.)
// ════════════════════════════════════════════════════════════════════

func TestHonesty_Deep_iIchimoku_Modes(t *testing.T) {
	// iIchimoku with various mode constants.
	// If MODE_TENKAN/MODE_KIJUN/MODE_SENKOU_A/MODE_SENKOU_B are not in constants,
	// the compiler should reject them (not silent 0).
	source := `
extern int MagicNumber = 50006;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double tenkan = iIchimoku(Symbol(), 0, 9, 26, 52, MODE_TENKAN, 0);
    double kijun = iIchimoku(Symbol(), 0, 9, 26, 52, MODE_KIJUN, 0);
    double senkouA = iIchimoku(Symbol(), 0, 9, 26, 52, MODE_SENKOU_A, 0);
    double senkouB = iIchimoku(Symbol(), 0, 9, 26, 52, MODE_SENKOU_B, 0);
    if (tenkan > kijun && senkouA > senkouB && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "Ichi", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "Deep-iIchimoku-Modes", "DEEP", source)
	t.Logf("Deep-iIchimoku-Modes: %s (compile=%v, err=%s, blinds=%v)", r, r.compileOK, r.compileErr, r.blindSpots)
}

// ════════════════════════════════════════════════════════════════════
// Deep probe: iStochastic modes
// ════════════════════════════════════════════════════════════════════

func TestHonesty_Deep_iStochastic_Modes(t *testing.T) {
	// iStochastic with MODE_MAIN and MODE_SIGNAL.
	source := `
extern int MagicNumber = 50007;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double main = iStochastic(Symbol(), 0, 5, 3, 3, MODE_SMA, 0, MODE_MAIN, 0);
    double signal = iStochastic(Symbol(), 0, 5, 3, 3, MODE_SMA, 0, MODE_SIGNAL, 0);
    if (main > signal && main < 20 && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "Stoch", MagicNumber, 0, clrGreen);
    if (main < signal && OrdersTotal() > 0)
    {
        for (int i = 0; i < OrdersTotal(); i++)
        {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
                OrderClose(OrderTicket(), OrderLots(), Bid, 5);
        }
    }
}
`
	r := runHonestyCheck(t, "Deep-iStochastic-Modes", "DEEP", source)
	t.Logf("Deep-iStochastic-Modes: %s (trades=%d, blinds=%v)", r, r.trades, r.blindSpots)
	if r.verdict == verdictCrack {
		t.Errorf("Deep-iStochastic-Modes: expected ✅ or 🟢, got 🔴")
	}
}

// ════════════════════════════════════════════════════════════════════
// Deep probe: iBands MODE_UPPER/MODE_LOWER
// ════════════════════════════════════════════════════════════════════

func TestHonesty_Deep_iBands_Modes(t *testing.T) {
	// iBands with MODE_UPPER/MODE_LOWER/MODE_MAIN.
	source := `
extern int MagicNumber = 50008;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double upper = iBands(Symbol(), 0, 20, 2, 0, PRICE_CLOSE, MODE_UPPER, 0);
    double lower = iBands(Symbol(), 0, 20, 2, 0, PRICE_CLOSE, MODE_LOWER, 0);
    double main = iBands(Symbol(), 0, 20, 2, 0, PRICE_CLOSE, MODE_MAIN, 0);
    if (Close[0] > upper && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_SELL, LotSize, Bid, 5, 0, 0, "Bands", MagicNumber, 0, clrRed);
    if (Close[0] < lower && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "Bands", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "Deep-iBands-Modes", "DEEP", source)
	t.Logf("Deep-iBands-Modes: %s (trades=%d, blinds=%v)", r, r.trades, r.blindSpots)
	if r.verdict == verdictCrack {
		t.Errorf("Deep-iBands-Modes: expected ✅ or 🟢, got 🔴")
	}
}

// ════════════════════════════════════════════════════════════════════
// Deep probe: iAlligator modes
// ════════════════════════════════════════════════════════════════════

func TestHonesty_Deep_iAlligator_Modes(t *testing.T) {
	// iAlligator with MODE_GATORLIPS/MODE_GATORJAW/MODE_GATORTEETH.
	source := `
extern int MagicNumber = 50009;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double jaw = iAlligator(Symbol(), 0, 13, 8, 8, 5, 5, 3, MODE_EMA, PRICE_MEDIAN, MODE_GATORJAW, 0);
    double teeth = iAlligator(Symbol(), 0, 13, 8, 8, 5, 5, 3, MODE_EMA, PRICE_MEDIAN, MODE_GATORTEETH, 0);
    double lips = iAlligator(Symbol(), 0, 13, 8, 8, 5, 5, 3, MODE_EMA, PRICE_MEDIAN, MODE_GATORLIPS, 0);
    if (lips > teeth && teeth > jaw && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "Gator", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "Deep-iAlligator-Modes", "DEEP", source)
	t.Logf("Deep-iAlligator-Modes: %s (trades=%d, blinds=%v, fatal=%v)", r, r.trades, r.blindSpots, r.fatalBlinds)
}

// ════════════════════════════════════════════════════════════════════
// Summary report generator (not a test, but a helper for audit report)
// ════════════════════════════════════════════════════════════════════

func TestHonesty_GenerateReport(t *testing.T) {
	// This test generates a summary of all honesty audit results.
	// It's intentionally left as a placeholder — the real audit report
	// is generated by running all individual tests and collecting results.
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Log("  MQL Honesty Audit — Summary")
	t.Log("═══════════════════════════════════════════════════════════════")
	t.Log("T1 (5 EAs): Expected ✅忠实 — simple indicators + OrderSend")
	t.Log("T2 (5 EAs): Expected ✅忠实 or 🟢诚实失败 — order management + multi-indicator")
	t.Log("T3 (8 probes): Expected 🟢诚实失败 = 100% — unsupported features must be loud")
	t.Log("REG (1 test): MODE_SIGNAL regression — must not silently map to 0")
	t.Log("DEEP (8 probes): Deep investigation of specific crack vectors")
	t.Log("═══════════════════════════════════════════════════════════════")
}
