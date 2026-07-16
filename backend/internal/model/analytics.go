package model

import "github.com/shopspring/decimal"


type TradeStats struct {
	TotalTrades          int             `json:"total_trades"`
	WinningTrades        int             `json:"winning_trades"`
	LosingTrades         int             `json:"losing_trades"`
	BuyTrades            int             `json:"buy_trades"`
	SellTrades           int             `json:"sell_trades"`
	WinRate              decimal.Decimal `json:"win_rate"`
	TotalProfit          decimal.Decimal `json:"total_profit"`
	TotalLoss            decimal.Decimal `json:"total_loss"`
	NetProfit            decimal.Decimal `json:"net_profit"`
	ProfitFactor         decimal.Decimal `json:"profit_factor"`
	AverageProfit        decimal.Decimal `json:"average_profit"`
	AverageLoss          decimal.Decimal `json:"average_loss"`
	AverageTrade         decimal.Decimal `json:"average_trade"`
	AverageVolume        decimal.Decimal `json:"average_volume"`
	LargestWin           decimal.Decimal `json:"largest_win"`
	LargestLoss          decimal.Decimal `json:"largest_loss"`
	TotalVolume          decimal.Decimal `json:"total_volume"`
	MaxConsecutiveWins   int             `json:"max_consecutive_wins"`
	MaxConsecutiveLosses int             `json:"max_consecutive_losses"`
	AverageHoldingTime   string          `json:"average_holding_time"`
	TotalDeposit         decimal.Decimal `json:"total_deposit"`
	TotalWithdrawal      decimal.Decimal `json:"total_withdrawal"`
	NetDeposit           decimal.Decimal `json:"net_deposit"`
}

type RiskMetrics struct {
	MaxDrawdown        decimal.Decimal `json:"max_drawdown"`
	MaxDrawdownPercent decimal.Decimal `json:"max_drawdown_percent"`
	SharpeRatio        decimal.Decimal `json:"sharpe_ratio"`
	SortinoRatio       decimal.Decimal `json:"sortino_ratio"`
	CalmarRatio        decimal.Decimal `json:"calmar_ratio"`
	Volatility         decimal.Decimal `json:"volatility"`
	ValueAtRisk95      decimal.Decimal `json:"value_at_risk_95"`
	ExpectedShortfall  decimal.Decimal `json:"expected_shortfall"`
	MaxDailyLoss       decimal.Decimal `json:"max_daily_loss"`
	MaxWeeklyLoss      decimal.Decimal `json:"max_weekly_loss"`
	AverageDailyReturn decimal.Decimal `json:"average_daily_return"`
	ReturnStdDev       decimal.Decimal `json:"return_std_dev"`
}

type SymbolStats struct {
	Symbol             string  `json:"symbol" db:"symbol"`
	TotalTrades        int     `json:"total_trades" db:"total_trades"`
	WinningTrades      int     `json:"winning_trades" db:"winning_trades"`
	LosingTrades       int     `json:"losing_trades" db:"losing_trades"`
	WinRate            decimal.Decimal `json:"win_rate" db:"win_rate"`
	TotalProfit        decimal.Decimal `json:"total_profit" db:"total_profit"`
	TotalLoss          decimal.Decimal `json:"total_loss" db:"total_loss"`
	NetProfit          decimal.Decimal `json:"net_profit" db:"net_profit"`
	ProfitFactor       decimal.Decimal `json:"profit_factor" db:"profit_factor"`
	AverageProfit      decimal.Decimal `json:"average_profit" db:"average_profit"`
	TotalVolume        decimal.Decimal `json:"total_volume" db:"total_volume"`
	AverageVolume      decimal.Decimal `json:"average_volume" db:"average_volume"`
	LargestWin         decimal.Decimal `json:"largest_win" db:"largest_win"`
	LargestLoss        decimal.Decimal `json:"largest_loss" db:"largest_loss"`
	AverageHoldingTime string  `json:"average_holding_time" db:"average_holding_time"`
}

type DailyEquity struct {
	Date     string  `json:"date"`
	Equity   decimal.Decimal `json:"equity"`
	Balance  decimal.Decimal `json:"balance"`
	Profit   decimal.Decimal `json:"profit"`
	Drawdown decimal.Decimal `json:"drawdown"`
}

type TradeReport struct {
	AccountID     string         `json:"account_id"`
	StartDate     string         `json:"start_date"`
	EndDate       string         `json:"end_date"`
	TradeStats    TradeStats     `json:"trade_stats"`
	RiskMetrics   RiskMetrics    `json:"risk_metrics"`
	SymbolStats   []*SymbolStats `json:"symbol_stats"`
	DailyEquity   []*DailyEquity `json:"daily_equity"`
	EquityCurve   []decimal.Decimal `json:"equity_curve"`
	DrawdownCurve []decimal.Decimal `json:"drawdown_curve"`
}

type MonthlyPnL struct {
	Month      string  `json:"month"`
	MonthNum   int     `json:"month_num"`
	Profit     decimal.Decimal `json:"profit"`
	Trades     int     `json:"trades"`
	WinTrades  int     `json:"win_trades"`
	LossTrades int     `json:"loss_trades"`
}

type DailyPnL struct {
	Day                    string  `json:"day"`
	DayNum                 int     `json:"day_num"`
	Date                   string  `json:"date"`
	PnL                    decimal.Decimal `json:"pnl"`
	Trades                 int     `json:"trades"`
	Lots                   decimal.Decimal `json:"lots"`
	Balance                decimal.Decimal `json:"balance"`
	ProfitFactor           decimal.Decimal `json:"profit_factor"`
	MaxFloatingLossAmount   decimal.Decimal `json:"max_floating_loss_amount"`
	MaxFloatingLossRatio    decimal.Decimal `json:"max_floating_loss_ratio"`
	MaxFloatingProfitAmount decimal.Decimal `json:"max_floating_profit_amount"`
	MaxFloatingProfitRatio  decimal.Decimal `json:"max_floating_profit_ratio"`
}

type HourlyStats struct {
	Hour      string  `json:"hour"`
	HourStart int     `json:"hour_start"`
	Trades    int     `json:"trades"`
	Profit    decimal.Decimal `json:"profit"`
	WinRate   decimal.Decimal `json:"win_rate"`
	AvgPnL    decimal.Decimal `json:"avg_pnl"`
	Lots                   decimal.Decimal `json:"lots"`
	Balance                decimal.Decimal `json:"balance"`
	ProfitFactor           decimal.Decimal `json:"profit_factor"`
	MaxFloatingLossAmount   decimal.Decimal `json:"max_floating_loss_amount"`
	MaxFloatingLossRatio    decimal.Decimal `json:"max_floating_loss_ratio"`
	MaxFloatingProfitAmount decimal.Decimal `json:"max_floating_profit_amount"`
	MaxFloatingProfitRatio  decimal.Decimal `json:"max_floating_profit_ratio"`
}

// WeekdayPnL aggregates closed trades by ISO weekday (1=Monday … 7=Sunday).
type WeekdayPnL struct {
	Weekday int             `json:"weekday"`
	PnL     decimal.Decimal `json:"pnl"`
	Trades  int             `json:"trades"`
}

type EquityPoint struct {
	Date    string  `json:"date"`
	Equity  decimal.Decimal `json:"equity"`
	Balance decimal.Decimal `json:"balance"`
	Profit  decimal.Decimal `json:"profit"`
}

type MonthlyAnalysisPoint struct {
	Year   int     `json:"year"`
	Month  int     `json:"month"`
	Change decimal.Decimal `json:"change"`
	Profit decimal.Decimal `json:"profit"`
	Lots   decimal.Decimal `json:"lots"`
	Pips   decimal.Decimal `json:"pips"`
	Trades int     `json:"trades"`
}

type MonthlyBonusSymbol struct {
	Symbol       string  `json:"symbol"`
	Trades       int     `json:"trades"`
	SharePercent decimal.Decimal `json:"share_percent"` // lot-volume share for pie, 0–100
}

type MonthlyBonusRiskRow struct {
	Symbol    string          `json:"symbol"`
	RiskRatio decimal.Decimal `json:"risk_ratio"`
}

type MonthlyBonusHoldingRow struct {
	Symbol           string          `json:"symbol"`
	BullsSeconds     decimal.Decimal `json:"bulls_seconds"`
	ShortTermSeconds decimal.Decimal `json:"short_term_seconds"`
}

type MonthlyAnalysisBonus struct {
	RiskRatio         decimal.Decimal             `json:"risk_ratio"`
	SymbolPopularity  []*MonthlyBonusSymbol     `json:"symbol_popularity"`
	SymbolRisks       []*MonthlyBonusRiskRow    `json:"symbol_risks"`
	SymbolHoldings    []*MonthlyBonusHoldingRow `json:"symbol_holdings"`
}

type AccountAnalytics struct {
	TradeStats   *TradeStats    `json:"trade_stats"`
	RiskMetrics  *RiskMetrics   `json:"risk_metrics"`
	SymbolStats  []*SymbolStats `json:"symbol_stats"`
	MonthlyPnL   []*MonthlyPnL  `json:"monthly_pnl"`
	DailyPnL     []*DailyPnL    `json:"daily_pnl"`
	HourlyStats  []*HourlyStats `json:"hourly_stats"`
	WeekdayPnL   []*WeekdayPnL  `json:"weekday_pnl"`
	EquityCurve  []*EquityPoint `json:"equity_curve"`
	RecentTrades []*TradeRecord `json:"recent_trades"`
}

// ── Monthly Detail (drill-down) ──

// MonthlyDetailMetrics holds aggregated metrics for a single month.
type MonthlyDetailMetrics struct {
	NetReturn     decimal.Decimal `json:"net_return"`
	ReturnPercent decimal.Decimal `json:"return_percent"` // NetReturn / startingBalance * 100, sourced from account_balance_history
	TotalTrades   int             `json:"total_trades"`
	WinRate       decimal.Decimal `json:"win_rate"`
	ProfitFactor  decimal.Decimal `json:"profit_factor"`
	BestTrade     decimal.Decimal `json:"best_trade"`
	WorstTrade    decimal.Decimal `json:"worst_trade"`
}

// MonthlySymbolPnL holds per-symbol P&L for a single month.
type MonthlySymbolPnL struct {
	Symbol    string          `json:"symbol"`
	NetProfit decimal.Decimal `json:"net_profit"`
	Trades    int             `json:"trades"`
	WinRate   decimal.Decimal `json:"win_rate"`
}

// MonthlyHoldingStats holds holding time statistics for a single month.
type MonthlyHoldingStats struct {
	AverageHours decimal.Decimal `json:"average_hours"`
	MedianHours  decimal.Decimal `json:"median_hours"`
	MaxHours     decimal.Decimal `json:"max_hours"`
	MinHours     decimal.Decimal `json:"min_hours"`
}
