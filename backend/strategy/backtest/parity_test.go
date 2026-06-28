package backtest

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/strategy/sdk"
)

func TestParseMTTime(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"2024.01.15 10:00:00", true},
		{"2024.01.15 10:00", true},
		{"2024.01.15", true},
		{"invalid", false},
	}

	for _, tt := range tests {
		result := parseMTTime(tt.input)
		if tt.valid && result.IsZero() {
			t.Errorf("parseMTTime(%q) returned zero time", tt.input)
		}
		if !tt.valid && !result.IsZero() {
			t.Errorf("parseMTTime(%q) should return zero time, got %v", tt.input, result)
		}
	}
}

func TestParseMT4Report(t *testing.T) {
	html := `
<html><body>
<h2>Testing Sample EA</h2>
<p>Symbol: EURUSD</p>
<p>Period: H1 (Hourly)</p>
<table>
<tr><td>Initial Deposit</td><td>10000.00</td></tr>
<tr><td>Total Net Profit</td><td>500.00</td></tr>
<tr><td>Profit Trades</td><td>30</td></tr>
<tr><td>Loss Trades</td><td>20</td></tr>
</table>
<h3>Results</h3>
<table>
<tr><th>Ticket</th><th>Open Time</th><th>Type</th><th>Volume</th><th>Price</th><th>S/L</th><th>T/P</th><th>Close Time</th><th>Profit</th></tr>
<tr><td>1001</td><td>2024.01.15 10:00:00</td><td>buy</td><td>0.10</td><td>1.1000</td><td>1.0950</td><td>1.1100</td><td>2024.01.15 12:00:00</td><td>50.00</td></tr>
<tr><td>1002</td><td>2024.01.15 14:00:00</td><td>sell</td><td>0.10</td><td>1.1050</td><td>1.1100</td><td>1.0950</td><td>2024.01.15 16:00:00</td><td>-30.00</td></tr>
</table>
</body></html>
`
	report, err := ParseMT4Report(html)
	if err != nil {
		t.Fatalf("ParseMT4Report failed: %v", err)
	}

	if report.Symbol != "EURUSD" {
		t.Errorf("expected symbol EURUSD, got %s", report.Symbol)
	}
	if len(report.Trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(report.Trades))
	}

	trade1 := report.Trades[0]
	if trade1.Side != "buy" {
		t.Errorf("trade 1: expected buy, got %s", trade1.Side)
	}
	if !trade1.Volume.Equal(decimal.NewFromFloat(0.10)) {
		t.Errorf("trade 1: expected volume 0.10, got %s", trade1.Volume.String())
	}
	if !trade1.OpenPrice.Equal(decimal.NewFromFloat(1.1000)) {
		t.Errorf("trade 1: expected open price 1.1000, got %s", trade1.OpenPrice.String())
	}
	if !trade1.Profit.Equal(decimal.NewFromFloat(50.00)) {
		t.Errorf("trade 1: expected profit 50.00, got %s", trade1.Profit.String())
	}

	trade2 := report.Trades[1]
	if trade2.Side != "sell" {
		t.Errorf("trade 2: expected sell, got %s", trade2.Side)
	}
}

func TestCompareParity_ExactMatch(t *testing.T) {
	openTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	closeTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	goTrades := []Trade{
		{
			Side:       sdk.SideBuy,
			Volume:     decimal.NewFromFloat(0.10),
			EntryTime:  openTime,
			EntryPrice: decimal.NewFromFloat(1.1000),
			ExitTime:   closeTime,
			ExitPrice:  decimal.NewFromFloat(1.1050),
			Profit:     decimal.NewFromFloat(50.00),
		},
	}

	mtTrades := []MTReportTrade{
		{
			Side:       "buy",
			Volume:     decimal.NewFromFloat(0.10),
			OpenTime:   openTime,
			OpenPrice:  decimal.NewFromFloat(1.1000),
			CloseTime:  closeTime,
			ClosePrice: decimal.NewFromFloat(1.1050),
			Profit:     decimal.NewFromFloat(50.00),
		},
	}

	report := CompareParity(goTrades, mtTrades, DefaultParityConfig())
	if !report.Passed {
		t.Errorf("expected parity to pass, got %d mismatches", len(report.Mismatches))
		for _, m := range report.Mismatches {
			t.Logf("  mismatch: %s (%s) Go=%s MT=%s", m.Type, m.Severity, m.GoValue, m.MTValue)
		}
	}
	if report.MatchedCount != 1 {
		t.Errorf("expected 1 matched trade, got %d", report.MatchedCount)
	}
}

func TestCompareParity_SideMismatch(t *testing.T) {
	openTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	goTrades := []Trade{
		{Side: sdk.SideBuy, Volume: decimal.NewFromFloat(0.10), EntryTime: openTime, EntryPrice: decimal.NewFromFloat(1.1000), Profit: decimal.NewFromFloat(50.00)},
	}

	mtTrades := []MTReportTrade{
		{Side: "sell", Volume: decimal.NewFromFloat(0.10), OpenTime: openTime, OpenPrice: decimal.NewFromFloat(1.1000), Profit: decimal.NewFromFloat(50.00)},
	}

	report := CompareParity(goTrades, mtTrades, DefaultParityConfig())
	if report.Passed {
		t.Error("expected parity to fail due to side mismatch")
	}
	found := false
	for _, m := range report.Mismatches {
		if m.Type == MismatchSide {
			found = true
			if m.Severity != "fatal" {
				t.Errorf("expected side mismatch to be fatal, got %s", m.Severity)
			}
		}
	}
	if !found {
		t.Error("expected MismatchSide in mismatches")
	}
}

func TestCompareParity_MissingInGo(t *testing.T) {
	openTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	goTrades := []Trade{
		{Side: sdk.SideBuy, Volume: decimal.NewFromFloat(0.10), EntryTime: openTime, EntryPrice: decimal.NewFromFloat(1.1000), Profit: decimal.NewFromFloat(50.00)},
	}

	mtTrades := []MTReportTrade{
		{Side: "buy", Volume: decimal.NewFromFloat(0.10), OpenTime: openTime, OpenPrice: decimal.NewFromFloat(1.1000), Profit: decimal.NewFromFloat(50.00)},
		{Side: "sell", Volume: decimal.NewFromFloat(0.10), OpenTime: openTime.Add(2 * time.Hour), OpenPrice: decimal.NewFromFloat(1.1050), Profit: decimal.NewFromFloat(-30.00)},
	}

	report := CompareParity(goTrades, mtTrades, DefaultParityConfig())
	found := false
	for _, m := range report.Mismatches {
		if m.Type == MismatchMissingInGo {
			found = true
		}
	}
	if !found {
		t.Error("expected MismatchMissingInGo for the extra MT trade")
	}
}

func TestCompareParity_PriceTolerance(t *testing.T) {
	openTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	goTrades := []Trade{
		{Side: sdk.SideBuy, Volume: decimal.NewFromFloat(0.10), EntryTime: openTime, EntryPrice: decimal.NewFromFloat(1.1001), Profit: decimal.NewFromFloat(50.00)},
	}

	mtTrades := []MTReportTrade{
		{Side: "buy", Volume: decimal.NewFromFloat(0.10), OpenTime: openTime, OpenPrice: decimal.NewFromFloat(1.1000), Profit: decimal.NewFromFloat(50.00)},
	}

	cfg := DefaultParityConfig()
	report := CompareParity(goTrades, mtTrades, cfg)
	// 1 pip diff < 2 pip tolerance → should pass
	if !report.Passed {
		t.Error("expected parity to pass within price tolerance")
	}
}

func TestFormatReport(t *testing.T) {
	report := &ParityReport{
		Passed:       true,
		GoTradeCount: 10,
		MTTradeCount: 10,
		MatchedCount: 10,
		Mismatches:   nil,
	}

	output := report.FormatReport()
	if output == "" {
		t.Error("expected non-empty report output")
	}
	if !strings.Contains(output, "PASS") {
		t.Errorf("expected PASS in output, got: %s", output)
	}
}

func TestParseMTReport_AutoDetect(t *testing.T) {
	mt4HTML := `<html><head><title>Strategy Tester Report</title></head><body>
<p>Symbol: EURUSD</p><p>Period: M15</p>
<table><tr><td>Initial Deposit</td><td>10000</td></tr></table>
<h3>Results</h3><table>
<tr><td>1</td><td>2024.01.15 10:00</td><td>buy</td><td>0.10</td><td>1.1000</td><td>0</td><td>0</td><td>2024.01.15 12:00</td><td>50.00</td></tr>
</table></body></html>`

	report, err := ParseMTReport(mt4HTML)
	if err != nil {
		t.Fatalf("ParseMTReport failed: %v", err)
	}
	if len(report.Trades) != 1 {
		t.Errorf("expected 1 trade, got %d", len(report.Trades))
	}
}
