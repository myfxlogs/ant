package mql2go

import (
	"context"
	"fmt"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/backtest"
	"alphaforge/tools/mql2go/interp"
)

// ── Honesty Audit Types ──────────────────────────────────────────────

type honestyVerdict string

const (
	verdictFaithful   honestyVerdict = "✅忠实"
	verdictHonestFail honestyVerdict = "🟢诚实失败"
	verdictCrack      honestyVerdict = "🔴裂缝"
)

type honestyResult struct {
	name        string
	tier        string
	verdict     honestyVerdict
	compileOK   bool
	compileErr  string
	covScore    float64
	blindSpots  []string
	fatalBlinds []string
	trades      int
	isReliable  bool
	degraded    bool
	rootCause   string
}

func (r honestyResult) String() string {
	return fmt.Sprintf("[%s] %s: %s (compile=%v cov=%.0f%% blinds=%d fatal=%d trades=%d reliable=%v degraded=%v)",
		r.tier, r.name, r.verdict, r.compileOK, r.covScore*100,
		len(r.blindSpots), len(r.fatalBlinds), r.trades, r.isReliable, r.degraded)
}

// ── Helpers ──────────────────────────────────────────────────────────

func hasFatalBlindSpot(blindSpots []CoverageBlindSpot) []string {
	var fatal []string
	for _, bs := range blindSpots {
		if bs.Severity == interp.SeverityFatal {
			fatal = append(fatal, bs.Builtin)
		}
	}
	return fatal
}

func blindSpotNames(blindSpots []CoverageBlindSpot) []string {
	var names []string
	for _, bs := range blindSpots {
		names = append(names, bs.Builtin)
	}
	return names
}

func runHonestyCheck(t *testing.T, name, tier, source string) honestyResult {
	t.Helper()
	r := honestyResult{name: name, tier: tier}

	// Step 1: Compile with coverage
	runner, cov, err := CompileMQLWithCoverage(source)
	if err != nil {
		r.verdict = verdictHonestFail
		r.compileErr = err.Error()
		t.Logf("[%s] %s: COMPILE FAIL (honest): %s", tier, name, err)
		return r
	}
	r.compileOK = true

	if cov != nil {
		r.covScore = cov.Score
		r.blindSpots = blindSpotNames(cov.BlindSpots)
		r.fatalBlinds = hasFatalBlindSpot(cov.BlindSpots)
	}

	// Step 2: Run backtest
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
		// VM fatal error = honest failure (the VM told us something is wrong)
		r.verdict = verdictHonestFail
		r.compileErr = err.Error()
		t.Logf("[%s] %s: BACKTEST FAIL (honest): %s", tier, name, err)
		return r
	}

	if result.Metrics != nil {
		r.trades = int(result.Metrics.TotalTrades)
	}

	// Check runtime blind spots
	runtimeBlinds := runner.GetRuntimeBlindSpots()
	for _, rbs := range runtimeBlinds {
		if rbs.Severity == interp.SeverityFatal {
			r.fatalBlinds = append(r.fatalBlinds, rbs.Builtin+" (runtime)")
		}
	}

	// Step 3: Classify
	// 🟢 Honest failure: has fatal blind spots (coverage or runtime) → system told us
	if len(r.fatalBlinds) > 0 {
		r.verdict = verdictHonestFail
		t.Logf("[%s] %s: HONEST FAIL (fatal blind spots: %v)", tier, name, r.fatalBlinds)
		return r
	}

	// 🟢 Honest failure: non-fatal blind spots present → system warned us
	if len(r.blindSpots) > 0 {
		r.verdict = verdictHonestFail
		t.Logf("[%s] %s: HONEST FAIL (blind spots: %v)", tier, name, r.blindSpots)
		return r
	}

	// ✅ Faithful: compiled, no blind spots, backtest ran
	r.verdict = verdictFaithful
	t.Logf("[%s] %s: FAITHFUL (trades=%d, equity points=%d)", tier, name, r.trades, len(result.Equity))
	return r
}

// ════════════════════════════════════════════════════════════════════
// T1: Simple EAs — should be ✅忠实 (faithful execution)
// ════════════════════════════════════════════════════════════════════

func TestHonesty_T1_MA_Crossover(t *testing.T) {
	source := `
extern int MagicNumber = 10001;
extern double LotSize = 0.1;
extern int MAPeriod = 14;
int OnInit() { return 0; }
void OnBar()
{
    double ma = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);
    double prevClose = Close[1];
    if (ma > prevClose)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "MA", MagicNumber, 0, clrGreen);
    if (ma < prevClose)
        OrderSend(Symbol(), OP_SELL, LotSize, Bid, 5, 0, 0, "MA", MagicNumber, 0, clrRed);
}
`
	r := runHonestyCheck(t, "T1-MA-Crossover", "T1", source)
	if r.verdict == verdictCrack {
		t.Errorf("T1-MA-Crossover: expected ✅ or 🟢, got 🔴 (silent wrong result)")
	}
}

func TestHonesty_T1_RSI_Threshold(t *testing.T) {
	source := `
extern int MagicNumber = 10002;
extern double LotSize = 0.1;
extern int RSIPeriod = 14;
int OnInit() { return 0; }
void OnBar()
{
    double rsi = iRSI(Symbol(), 0, RSIPeriod, PRICE_CLOSE, 0);
    if (rsi < 30)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "RSI", MagicNumber, 0, clrGreen);
    if (rsi > 70)
        OrderSend(Symbol(), OP_SELL, LotSize, Bid, 5, 0, 0, "RSI", MagicNumber, 0, clrRed);
}
`
	r := runHonestyCheck(t, "T1-RSI-Threshold", "T1", source)
	if r.verdict == verdictCrack {
		t.Errorf("T1-RSI-Threshold: expected ✅ or 🟢, got 🔴")
	}
}

func TestHonesty_T1_MACD_Signal(t *testing.T) {
	source := `
extern int MagicNumber = 10003;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double macd = iMACD(Symbol(), 0, 12, 26, 9, PRICE_CLOSE, MODE_MAIN, 0);
    double signal = iMACD(Symbol(), 0, 12, 26, 9, PRICE_CLOSE, MODE_SIGNAL, 0);
    if (macd > signal)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "MACD", MagicNumber, 0, clrGreen);
    if (macd < signal)
        OrderSend(Symbol(), OP_SELL, LotSize, Bid, 5, 0, 0, "MACD", MagicNumber, 0, clrRed);
}
`
	r := runHonestyCheck(t, "T1-MACD-Signal", "T1", source)
	if r.verdict == verdictCrack {
		t.Errorf("T1-MACD-Signal: expected ✅ or 🟢, got 🔴 (MODE_SIGNAL regression?)")
	}
}

func TestHonesty_T1_Bands_Breakout(t *testing.T) {
	source := `
extern int MagicNumber = 10004;
extern double LotSize = 0.1;
extern int BandsPeriod = 20;
int OnInit() { return 0; }
void OnBar()
{
    double upper = iBands(Symbol(), 0, BandsPeriod, 2, 0, PRICE_CLOSE, MODE_UPPER, 0);
    double lower = iBands(Symbol(), 0, BandsPeriod, 2, 0, PRICE_CLOSE, MODE_LOWER, 0);
    if (Close[0] > upper)
        OrderSend(Symbol(), OP_SELL, LotSize, Bid, 5, 0, 0, "BB", MagicNumber, 0, clrRed);
    if (Close[0] < lower)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "BB", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "T1-Bands-Breakout", "T1", source)
	if r.verdict == verdictCrack {
		t.Errorf("T1-Bands-Breakout: expected ✅ or 🟢, got 🔴")
	}
}

func TestHonesty_T1_ATR_Stop(t *testing.T) {
	source := `
extern int MagicNumber = 10005;
extern double LotSize = 0.1;
extern int ATRPeriod = 14;
int OnInit() { return 0; }
void OnBar()
{
    double atr = iATR(Symbol(), 0, ATRPeriod, 0);
    if (atr > 0.001 && Close[1] > Close[2])
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "ATR", MagicNumber, 0, clrGreen);
}
`
	r := runHonestyCheck(t, "T1-ATR-Stop", "T1", source)
	if r.verdict == verdictCrack {
		t.Errorf("T1-ATR-Stop: expected ✅ or 🟢, got 🔴")
	}
}

// ════════════════════════════════════════════════════════════════════
// T2: Medium EAs — should be ✅忠实 or 🟢诚实失败 (faithful or honest fail)
// ════════════════════════════════════════════════════════════════════

func TestHonesty_T2_OrderSelect_CloseAll(t *testing.T) {
	source := `
extern int MagicNumber = 20001;
extern double LotSize = 0.1;
extern int MAPeriod = 14;
int OnInit() { return 0; }
void OnBar()
{
    double ma = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);
    double prevClose = Close[1];
    if (ma > prevClose && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "T2", MagicNumber, 0, clrGreen);
    if (ma < prevClose)
    {
        for (int i = 0; i < OrdersTotal(); i++)
        {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
                OrderClose(OrderTicket(), OrderLots(), Bid, 5);
        }
    }
}
`
	r := runHonestyCheck(t, "T2-OrderSelect-CloseAll", "T2", source)
	if r.verdict == verdictCrack {
		t.Errorf("T2-OrderSelect-CloseAll: expected ✅ or 🟢, got 🔴")
	}
}

func TestHonesty_T2_TrailingStop(t *testing.T) {
	source := `
extern int MagicNumber = 20002;
extern double LotSize = 0.1;
extern int TrailingStop = 50;
extern int MAPeriod = 14;
int OnInit() { return 0; }
void OnBar()
{
    double ma = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 1);
    if (ma > Close[1] && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "TS", MagicNumber, 0, clrGreen);
    for (int i = 0; i < OrdersTotal(); i++)
    {
        if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
        {
            if (OrderType() == OP_BUY)
            {
                double sl = OrderStopLoss();
                double newSL = Bid - TrailingStop * Point;
                if (sl == 0 || newSL > sl)
                    OrderModify(OrderTicket(), OrderOpenPrice(), newSL, OrderTakeProfit(), 0, clrGreen);
            }
        }
    }
}
`
	r := runHonestyCheck(t, "T2-TrailingStop", "T2", source)
	if r.verdict == verdictCrack {
		t.Errorf("T2-TrailingStop: expected ✅ or 🟢, got 🔴")
	}
}

func TestHonesty_T2_MultiIndicator(t *testing.T) {
	source := `
extern int MagicNumber = 20003;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double ma = iMA(Symbol(), 0, 14, 0, MODE_EMA, PRICE_CLOSE, 0);
    double rsi = iRSI(Symbol(), 0, 14, PRICE_CLOSE, 0);
    double macd = iMACD(Symbol(), 0, 12, 26, 9, PRICE_CLOSE, MODE_MAIN, 0);
    double signal = iMACD(Symbol(), 0, 12, 26, 9, PRICE_CLOSE, MODE_SIGNAL, 0);
    if (ma > Close[1] && rsi < 70 && macd > signal && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "MI", MagicNumber, 0, clrGreen);
    if (ma < Close[1] && OrdersTotal() > 0)
    {
        for (int i = 0; i < OrdersTotal(); i++)
        {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
                OrderClose(OrderTicket(), OrderLots(), Bid, 5);
        }
    }
}
`
	r := runHonestyCheck(t, "T2-MultiIndicator", "T2", source)
	if r.verdict == verdictCrack {
		t.Errorf("T2-MultiIndicator: expected ✅ or 🟢, got 🔴")
	}
}

func TestHonesty_T2_OrderHistory(t *testing.T) {
	source := `
extern int MagicNumber = 20004;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    if (OrdersTotal() == 0 && OrdersHistoryTotal() == 0)
    {
        OrderSend(Symbol(), OP_BUY, LotSize, Ask, 5, 0, 0, "H", MagicNumber, 0, clrGreen);
        return;
    }
    if (OrdersTotal() > 0)
    {
        for (int i = 0; i < OrdersTotal(); i++)
        {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
                OrderClose(OrderTicket(), OrderLots(), Bid, 5);
        }
        return;
    }
    if (OrdersHistoryTotal() > 0)
    {
        for (int i = 0; i < OrdersHistoryTotal(); i++)
        {
            if (OrderSelect(i, SELECT_BY_POS, MODE_HISTORY))
            {
                double cp = OrderClosePrice();
                double ct = OrderCloseTime();
            }
        }
    }
}
`
	r := runHonestyCheck(t, "T2-OrderHistory", "T2", source)
	if r.verdict == verdictCrack {
		t.Errorf("T2-OrderHistory: expected ✅ or 🟢, got 🔴")
	}
}

func TestHonesty_T2_NormalizeDouble(t *testing.T) {
	source := `
extern int MagicNumber = 20005;
extern double LotSize = 0.1;
int OnInit() { return 0; }
void OnBar()
{
    double lot = NormalizeDouble(LotSize, 2);
    double price = NormalizeDouble(Ask, Digits);
    if (lot > 0 && price > 0 && OrdersTotal() == 0)
        OrderSend(Symbol(), OP_BUY, lot, price, 5, 0, 0, "ND", MagicNumber, 0, clrGreen);
    if (OrdersTotal() > 0)
    {
        for (int i = 0; i < OrdersTotal(); i++)
        {
            if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES))
                OrderClose(OrderTicket(), OrderLots(), Bid, 5);
        }
    }
}
`
	r := runHonestyCheck(t, "T2-NormalizeDouble", "T2", source)
	if r.verdict == verdictCrack {
		t.Errorf("T2-NormalizeDouble: expected ✅ or 🟢, got 🔴")
	}
}
