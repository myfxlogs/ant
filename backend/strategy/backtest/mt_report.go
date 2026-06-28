package backtest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// MTReportTrade is a single trade parsed from an MT4/MT5 Strategy Tester report.
type MTReportTrade struct {
	Ticket     int64
	Side       string // "buy" | "sell"
	Volume     decimal.Decimal
	OpenTime   time.Time
	OpenPrice  decimal.Decimal
	CloseTime  time.Time
	ClosePrice decimal.Decimal
	Profit     decimal.Decimal
	Swap       decimal.Decimal
	Commission decimal.Decimal
	Symbol     string
}

// MTReport holds parsed data from an MT4/MT5 Strategy Tester report.
type MTReport struct {
	Symbol    string
	Timeframe string
	Trades    []MTReportTrade
	// Summary metrics from the report (for quick comparison).
	InitialDeposit decimal.Decimal
	TotalNetProfit  decimal.Decimal
	ProfitTrades    int
	LossTrades      int
	TotalTrades     int
}

// ParseMT4Report parses an MT4 Strategy Tester HTML report.
// MT4 report format: HTML table with rows like:
//   <td>12345</td><td>buy</td><td>0.10</td><td>2024.01.15 10:00</td><td>1.1000</td>
//   <td>2024.01.15 12:00</td><td>1.1050</td><td>50.00</td><td>0.00</td><td>EURUSD</td>
func ParseMT4Report(html string) (*MTReport, error) {
	report := &MTReport{}

	// Extract summary fields.
	report.InitialDeposit = extractMT4Decimal(html, "Initial Deposit")
	report.TotalNetProfit = extractMT4Decimal(html, "Total Net Profit")
	report.ProfitTrades = extractMT4Int(html, "Profit Trades")
	report.LossTrades = extractMT4Int(html, "Loss Trades")
	report.TotalTrades = report.ProfitTrades + report.LossTrades

	// Extract symbol and timeframe from title.
	report.Symbol = extractMT4Symbol(html)
	report.Timeframe = extractMT4Timeframe(html)

	// Parse trade rows from the results table.
	tradeRows := extractMT4TradeRows(html)
	for _, row := range tradeRows {
		trade, err := parseMT4TradeRow(row)
		if err != nil {
			continue // skip malformed rows
		}
		report.Trades = append(report.Trades, trade)
	}

	if len(report.Trades) == 0 {
		return nil, fmt.Errorf("no trades found in MT4 report")
	}

	return report, nil
}

// ParseMT5Report parses an MT5 Strategy Tester HTML/XML report.
// MT5 report format differs slightly: uses "Deal" and "Position" terminology.
func ParseMT5Report(html string) (*MTReport, error) {
	report := &MTReport{}

	report.InitialDeposit = extractMT5Decimal(html, "Initial Deposit")
	report.TotalNetProfit = extractMT5Decimal(html, "Total Net Profit")
	report.ProfitTrades = extractMT5Int(html, "Profit Trades")
	report.LossTrades = extractMT5Int(html, "Loss Trades")
	report.TotalTrades = report.ProfitTrades + report.LossTrades

	report.Symbol = extractMT5Symbol(html)
	report.Timeframe = extractMT5Timeframe(html)

	tradeRows := extractMT5TradeRows(html)
	for _, row := range tradeRows {
		trade, err := parseMT5TradeRow(row)
		if err != nil {
			continue
		}
		report.Trades = append(report.Trades, trade)
	}

	if len(report.Trades) == 0 {
		return nil, fmt.Errorf("no trades found in MT5 report")
	}

	return report, nil
}

// ParseMTReport auto-detects MT4 vs MT5 format and dispatches.
func ParseMTReport(html string) (*MTReport, error) {
	if strings.Contains(html, "MetaTrader 5") || strings.Contains(html, "MT5") {
		return ParseMT5Report(html)
	}
	return ParseMT4Report(html)
}

// ── MT4 parsing helpers ────────────────────────────────────────────

var tdRegex = regexp.MustCompile(`<td[^>]*>(.*?)</td>`)

func extractMT4TradeRows(html string) []string {
	// Find the results table — MT4 reports have a table with trade rows.
	// Each row has 10+ <td> cells: ticket, order time, type, volume, price,
	// s/l, t/p, time, profit, balance, comment.
	tableStart := strings.Index(html, "Results")
	if tableStart < 0 {
		tableStart = strings.Index(html, "Orders")
	}
	if tableStart < 0 {
		return nil
	}
	rest := html[tableStart:]

	rowRegex := regexp.MustCompile(`(?s)<tr[^>]*>(.*?)</tr>`)
	rows := rowRegex.FindAllString(rest, -1)

	var tradeRows []string
	for _, row := range rows {
		cells := tdRegex.FindAllStringSubmatch(row, -1)
		if len(cells) >= 8 {
			tradeRows = append(tradeRows, row)
		}
	}
	return tradeRows
}

func parseMT4TradeRow(row string) (MTReportTrade, error) {
	cells := tdRegex.FindAllStringSubmatch(row, -1)
	if len(cells) < 8 {
		return MTReportTrade{}, fmt.Errorf("not enough cells: %d", len(cells))
	}

	strip := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, "&nbsp;", "")
		s = strings.ReplaceAll(s, "&amp;", "&")
		return s
	}

	get := func(i int) string {
		if i < len(cells) {
			return strip(cells[i][1])
		}
		return ""
	}

	ticket, _ := strconv.ParseInt(get(0), 10, 64)
	side := strings.ToLower(get(2))

	// Skip non-trade rows (balance, credit).
	if side != "buy" && side != "sell" {
		// MT4 may use "buy" / "sell" in a different column.
		// Try column 3 (some report variants).
		side = strings.ToLower(get(3))
		if side != "buy" && side != "sell" {
			return MTReportTrade{}, fmt.Errorf("not a trade row: side=%q", get(2))
		}
	}

	// MT4 standard columns: 0=ticket, 1=open_time, 2=type, 3=volume,
	// 4=price, 5=S/L, 6=T/P, 7=close_time, 8=close_price, 9=profit.
	// Some variants skip S/L and T/P, so we detect by column count.
	vol, _ := decimal.NewFromString(get(3))
	openTime := parseMTTime(get(1))
	openPrice, _ := decimal.NewFromString(get(4))

	var closeTime time.Time
	var closePrice decimal.Decimal
	var profitStr string

	if len(cells) >= 10 {
		// Standard: 5=S/L, 6=T/P, 7=close_time, 8=close_price, 9=profit
		closeTime = parseMTTime(get(7))
		closePrice, _ = decimal.NewFromString(get(8))
		profitStr = get(9)
	} else if len(cells) >= 9 {
		// With S/L+T/P but no close_price: 5=S/L, 6=T/P, 7=close_time, 8=profit
		closeTime = parseMTTime(get(7))
		profitStr = get(8)
	} else if len(cells) >= 8 {
		// Compact: 5=close_time, 6=close_price, 7=profit
		closeTime = parseMTTime(get(5))
		closePrice, _ = decimal.NewFromString(get(6))
		profitStr = get(7)
	}
	profit, _ := decimal.NewFromString(profitStr)

	return MTReportTrade{
		Ticket:     ticket,
		Side:       side,
		Volume:     vol,
		OpenTime:   openTime,
		OpenPrice:  openPrice,
		CloseTime:  closeTime,
		ClosePrice: closePrice,
		Profit:     profit,
		Symbol:     get(10),
	}, nil
}

func extractMT4Decimal(html, label string) decimal.Decimal {
	// Look for <td>Label</td><td>value</td> pattern.
	re := regexp.MustCompile(label + `</td>\s*<td[^>]*>([^<]+)</td>`)
	m := re.FindStringSubmatch(html)
	if m == nil {
		return decimal.Zero
	}
	val, _ := decimal.NewFromString(strings.TrimSpace(strings.ReplaceAll(m[1], "&nbsp;", "")))
	return val
}

func extractMT4Int(html, label string) int {
	re := regexp.MustCompile(label + `</td>\s*<td[^>]*>([^<]+)</td>`)
	m := re.FindStringSubmatch(html)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(strings.ReplaceAll(m[1], "&nbsp;", "")))
	return n
}

func extractMT4Symbol(html string) string {
	re := regexp.MustCompile(`Symbol:\s*(\w+)`)
	m := re.FindStringSubmatch(html)
	if m != nil {
		return m[1]
	}
	return ""
}

func extractMT4Timeframe(html string) string {
	re := regexp.MustCompile(`Period:\s*(\w+)`)
	m := re.FindStringSubmatch(html)
	if m != nil {
		return m[1]
	}
	return ""
}

// ── MT5 parsing helpers ────────────────────────────────────────────

func extractMT5TradeRows(html string) []string {
	// MT5 reports use "Deals" or "Positions" table.
	tableStart := strings.Index(html, "Deals")
	if tableStart < 0 {
		tableStart = strings.Index(html, "Positions")
	}
	if tableStart < 0 {
		return extractMT4TradeRows(html) // fallback
	}
	rest := html[tableStart:]

	rowRegex := regexp.MustCompile(`(?s)<tr[^>]*>(.*?)</tr>`)
	rows := rowRegex.FindAllString(rest, -1)

	var tradeRows []string
	for _, row := range rows {
		cells := tdRegex.FindAllStringSubmatch(row, -1)
		if len(cells) >= 6 {
			tradeRows = append(tradeRows, row)
		}
	}
	return tradeRows
}

func parseMT5TradeRow(row string) (MTReportTrade, error) {
	// MT5 format: Time, Deal, Symbol, Type, Direction, Volume, Price, Profit, Balance, Comment
	cells := tdRegex.FindAllStringSubmatch(row, -1)
	if len(cells) < 6 {
		return MTReportTrade{}, fmt.Errorf("not enough cells: %d", len(cells))
	}

	strip := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, "&nbsp;", "")
		return s
	}
	get := func(i int) string {
		if i < len(cells) {
			return strip(cells[i][1])
		}
		return ""
	}

	side := strings.ToLower(get(3))
	if side != "buy" && side != "sell" {
		// Try "Type" column variations.
		side = strings.ToLower(get(4))
		if side != "buy" && side != "sell" {
			return MTReportTrade{}, fmt.Errorf("not a trade row")
		}
	}

	vol, _ := decimal.NewFromString(get(5))
	price, _ := decimal.NewFromString(get(6))
	openTime := parseMTTime(get(0))
	profit, _ := decimal.NewFromString(get(7))

	return MTReportTrade{
		Side:      side,
		Volume:    vol,
		OpenTime:  openTime,
		OpenPrice: price,
		Profit:    profit,
		Symbol:    get(2),
	}, nil
}

func extractMT5Decimal(html, label string) decimal.Decimal {
	return extractMT4Decimal(html, label) // same pattern
}

func extractMT5Int(html, label string) int {
	return extractMT4Int(html, label) // same pattern
}

func extractMT5Symbol(html string) string {
	re := regexp.MustCompile(`Symbol:\s*<[^>]+>\s*(\w+)`)
	m := re.FindStringSubmatch(html)
	if m != nil {
		return m[1]
	}
	return extractMT4Symbol(html)
}

func extractMT5Timeframe(html string) string {
	re := regexp.MustCompile(`Timeframe:\s*(\w+)`)
	m := re.FindStringSubmatch(html)
	if m != nil {
		return m[1]
	}
	return extractMT4Timeframe(html)
}

// ── Shared helpers ─────────────────────────────────────────────────

// parseMTTime parses MT date formats: "2024.01.15 10:00:00" or "2024.01.15 10:00".
func parseMTTime(s string) time.Time {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, ".", "-")
	// Try with seconds first.
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
