package backtest

import (
	"testing"
	"time"

	"alphaforge/strategy/sdk"
)

func TestDeriveSymbolInfoFromBars_Empty(t *testing.T) {
	cfg := &Config{}
	DeriveSymbolInfoFromBars(cfg, nil)
	if cfg.SymbolDigits != 0 {
		t.Errorf("SymbolDigits = %d, want 0 for empty bars", cfg.SymbolDigits)
	}
}

func TestDeriveSymbolInfoFromBars_FiveDigit(t *testing.T) {
	cfg := &Config{}
	bars := []sdk.Bar{{Close: d("1.12345")}}
	DeriveSymbolInfoFromBars(cfg, bars)
	if cfg.SymbolDigits != 5 {
		t.Errorf("SymbolDigits = %d, want 5", cfg.SymbolDigits)
	}
	if !cfg.SymbolPoint.Equal(d("0.00001")) {
		t.Errorf("SymbolPoint = %s, want 0.00001", cfg.SymbolPoint)
	}
	if !cfg.Spread.Equal(d("0.0001")) {
		t.Errorf("Spread = %s, want 0.0001", cfg.Spread)
	}
}

func TestDeriveSymbolInfoFromBars_TwoDigit(t *testing.T) {
	cfg := &Config{}
	bars := []sdk.Bar{{Close: d("123.45")}}
	DeriveSymbolInfoFromBars(cfg, bars)
	if cfg.SymbolDigits != 2 {
		t.Errorf("SymbolDigits = %d, want 2", cfg.SymbolDigits)
	}
}

func TestDeriveSymbolInfoFromBars_Integer(t *testing.T) {
	cfg := &Config{}
	bars := []sdk.Bar{{Close: d("100")}}
	DeriveSymbolInfoFromBars(cfg, bars)
	if cfg.SymbolDigits != 0 {
		t.Errorf("SymbolDigits = %d, want 0 for integer price", cfg.SymbolDigits)
	}
}

func TestDeriveSymbolInfoFromBars_CapAt8(t *testing.T) {
	cfg := &Config{}
	bars := []sdk.Bar{{Close: d("1.123456789012345")}}
	DeriveSymbolInfoFromBars(cfg, bars)
	if cfg.SymbolDigits != 8 {
		t.Errorf("SymbolDigits = %d, want 8 (capped)", cfg.SymbolDigits)
	}
}

func TestParityReport_Summarize(t *testing.T) {
	report := &ParityReport{
		Passed:        true,
		GoTradeCount:  10,
		MTTradeCount:  10,
		MatchedCount:  9,
		GoTotalProfit: d("1000"),
		MTTotalProfit: d("990"),
		ProfitDiff:    d("10"),
		Mismatches: []ParityMismatch{
			{Severity: "fatal"},
			{Severity: "warning"},
			{Severity: "warning"},
		},
	}
	summary := report.Summarize()
	if !summary.Passed {
		t.Error("Passed should be true")
	}
	if summary.FatalCount != 1 {
		t.Errorf("FatalCount = %d, want 1", summary.FatalCount)
	}
	if summary.WarningCount != 2 {
		t.Errorf("WarningCount = %d, want 2", summary.WarningCount)
	}
	if summary.GoProfit != "1000" {
		t.Errorf("GoProfit = %q, want 1000", summary.GoProfit)
	}
}

func TestMakeSyntheticMTReport(t *testing.T) {
	trades := []MTReportTrade{
		{Side: "buy", Profit: d("100")},
		{Side: "sell", Profit: d("-50")},
	}
	report := MakeSyntheticMTReport(trades, "EURUSD", "M5")
	if report.Symbol != "EURUSD" {
		t.Errorf("Symbol = %q, want EURUSD", report.Symbol)
	}
	if report.TotalTrades != 2 {
		t.Errorf("TotalTrades = %d, want 2", report.TotalTrades)
	}
	if report.ProfitTrades != 1 {
		t.Errorf("ProfitTrades = %d, want 1", report.ProfitTrades)
	}
	if report.LossTrades != 1 {
		t.Errorf("LossTrades = %d, want 1", report.LossTrades)
	}
	if !report.TotalNetProfit.Equal(d("50")) {
		t.Errorf("TotalNetProfit = %s, want 50", report.TotalNetProfit)
	}
}

func TestMakeMTReportTrade(t *testing.T) {
	now := time.Now()
	trade := MakeMTReportTrade("buy", d("0.1"), d("1.1"), d("1.2"), d("100"), now, now.Add(time.Hour))
	if trade.Side != "buy" {
		t.Errorf("Side = %q, want buy", trade.Side)
	}
	if !trade.Volume.Equal(d("0.1")) {
		t.Errorf("Volume = %s, want 0.1", trade.Volume)
	}
	if !trade.Profit.Equal(d("100")) {
		t.Errorf("Profit = %s, want 100", trade.Profit)
	}
}
