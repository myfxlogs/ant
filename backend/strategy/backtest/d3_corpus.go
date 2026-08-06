package backtest

import (
	"fmt"
	"strings"

	"alphaforge/strategy/sdk"
)

// D3 EA Corpus Framework
//
// Defines a corpus of minimal MQL4 EAs that each exercise a specific indicator.
// These EAs are compiled with the MQL2Go transpiler and run through the backtest
// engine. The extracted indicator values are compared against the reference
// implementations.
//
// Each corpus entry is a self-contained MQL4 EA that stores its indicator
// value in a global variable for post-execution extraction via VMRunner.GetGlobal.

// D3CorpusEntry defines a single EA in the test corpus.
type D3CorpusEntry struct {
	Name        string
	Description string
	Source      string
	// GlobalsToExtract lists global variable names to extract after execution.
	GlobalsToExtract []string
}

// D3Corpus is the full set of corpus entries.
var D3Corpus = []D3CorpusEntry{
	{
		Name:        "iMA_SMA",
		Description: "Simple Moving Average with period 20",
		Source: `#property strict
extern int MAPeriod = 20;
double g_ma;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    g_ma = iMA(Symbol(), 0, MAPeriod, 0, MODE_SMA, PRICE_CLOSE, 0);
}
`,
		GlobalsToExtract: []string{"g_ma"},
	},
	{
		Name:        "iMA_EMA",
		Description: "Exponential Moving Average with period 12",
		Source: `#property strict
extern int MAPeriod = 12;
double g_ema;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    g_ema = iMA(Symbol(), 0, MAPeriod, 0, MODE_EMA, PRICE_CLOSE, 0);
}
`,
		GlobalsToExtract: []string{"g_ema"},
	},
	{
		Name:        "iRSI",
		Description: "RSI with period 14",
		Source: `#property strict
extern int RSIPeriod = 14;
double g_rsi;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    g_rsi = iRSI(Symbol(), 0, RSIPeriod, PRICE_CLOSE, 0);
}
`,
		GlobalsToExtract: []string{"g_rsi"},
	},
	{
		Name:        "iATR",
		Description: "ATR with period 14",
		Source: `#property strict
extern int ATRPeriod = 14;
double g_atr;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    g_atr = iATR(Symbol(), 0, ATRPeriod, 0);
}
`,
		GlobalsToExtract: []string{"g_atr"},
	},
	{
		Name:        "iMACD",
		Description: "MACD line (fast=12, slow=26, signal=9)",
		Source: `#property strict
extern int FastEMA = 12;
extern int SlowEMA = 26;
extern int SignalEMA = 9;
double g_macd;
double g_macd_signal;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    g_macd = iMACD(Symbol(), 0, FastEMA, SlowEMA, SignalEMA, PRICE_CLOSE, MODE_MAIN, 0);
    g_macd_signal = iMACD(Symbol(), 0, FastEMA, SlowEMA, SignalEMA, PRICE_CLOSE, MODE_SIGNAL, 0);
}
`,
		GlobalsToExtract: []string{"g_macd", "g_macd_signal"},
	},
	{
		Name:        "iBands",
		Description: "Bollinger Bands (period=20, deviation=2.0)",
		Source: `#property strict
extern int BandsPeriod = 20;
extern double BandsDeviation = 2.0;
double g_bands_upper;
double g_bands_middle;
double g_bands_lower;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    g_bands_upper = iBands(Symbol(), 0, BandsPeriod, BandsDeviation, 0, PRICE_CLOSE, MODE_UPPER, 0);
    g_bands_middle = iBands(Symbol(), 0, BandsPeriod, BandsDeviation, 0, PRICE_CLOSE, MODE_MAIN, 0);
    g_bands_lower = iBands(Symbol(), 0, BandsPeriod, BandsDeviation, 0, PRICE_CLOSE, MODE_LOWER, 0);
}
`,
		GlobalsToExtract: []string{"g_bands_upper", "g_bands_middle", "g_bands_lower"},
	},
	{
		Name:        "iStochastic",
		Description: "Stochastic oscillator (K=5, D=3, slowing=3)",
		Source: `#property strict
extern int KPeriod = 5;
extern int DPeriod = 3;
extern int Slowing = 3;
double g_stoch_k;
double g_stoch_d;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    g_stoch_k = iStochastic(Symbol(), 0, KPeriod, DPeriod, Slowing, MODE_SMA, 0, MODE_MAIN, 0);
    g_stoch_d = iStochastic(Symbol(), 0, KPeriod, DPeriod, Slowing, MODE_SMA, 0, MODE_SIGNAL, 0);
}
`,
		GlobalsToExtract: []string{"g_stoch_k", "g_stoch_d"},
	},
	{
		Name:        "iADX",
		Description: "ADX main line with period 14",
		Source: `#property strict
extern int ADXPeriod = 14;
double g_adx;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    g_adx = iADX(Symbol(), 0, ADXPeriod, PRICE_CLOSE, MODE_MAIN, 0);
}
`,
		GlobalsToExtract: []string{"g_adx"},
	},
	{
		Name:        "iCCI",
		Description: "CCI with period 14",
		Source: `#property strict
extern int CCIPeriod = 14;
double g_cci;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    g_cci = iCCI(Symbol(), 0, CCIPeriod, PRICE_TYPICAL, 0);
}
`,
		GlobalsToExtract: []string{"g_cci"},
	},
	{
		Name:        "iSAR",
		Description: "Parabolic SAR (step=0.02, max=0.2)",
		Source: `#property strict
extern double SARStep = 0.02;
extern double SARMax = 0.2;
double g_sar;

int OnInit() { return(INIT_SUCCEEDED); }

void OnTick() {
    g_sar = iSAR(Symbol(), 0, SARStep, SARMax, 0);
}
`,
		GlobalsToExtract: []string{"g_sar"},
	},
}

// D3CorpusResult holds the extracted values for a single corpus entry execution.
type D3CorpusResult struct {
	EntryName string
	Globals   map[string]float64
	Error     error
}

// D3CorpusSummary aggregates results across all corpus entries.
type D3CorpusSummary struct {
	Total    int
	Passed   int
	Failed   int
	Results  []D3CorpusResult
}

// D3RunCorpusEntry compiles and runs a single corpus EA against the given bars,
// extracting the specified global variables after execution.
//
// Returns a map of global name → extracted value (as float64).
func D3RunCorpusEntry(entry D3CorpusEntry, bars []sdk.Bar, config Config) (map[string]float64, error) {
	// This function is implemented in the test file to avoid importing
	// mql2go in the main backtest package (which would create a dependency cycle).
	// The test file uses the mql2go package directly.
	return nil, fmt.Errorf("D3RunCorpusEntry must be called from test context")
}

// D3FormatCorpusSummary renders a human-readable summary of corpus results.
func D3FormatCorpusSummary(s D3CorpusSummary) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("D3 Corpus Summary: %d/%d passed\n", s.Passed, s.Total))
	for _, r := range s.Results {
		status := "PASS"
		if r.Error != nil {
			status = "FAIL"
		}
		sb.WriteString(fmt.Sprintf("  [%s] %s", status, r.EntryName))
		if r.Error != nil {
			sb.WriteString(fmt.Sprintf(": %v", r.Error))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
