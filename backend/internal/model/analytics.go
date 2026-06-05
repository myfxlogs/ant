package model

import "github.com/shopspring/decimal"


type TradeStats struct {
	TotalTrades          int     `json:"total_trades"`
	WinningTrades        int     `json:"winning_trades"`
	LosingTrades         int     `json:"losing_trades"`
	BuyTrades            int     `json:"buy_trades"`
	SellTrades           int     `json:"sell_trades"`
	WinRate              decimal.Decimal `json:"win_rate"`
	TotalProfit          decimal.Decimal `json:"total_profit"`
	TotalLoss            decimal.Decimal `json:"total_loss"`
	NetProfit            decimal.Decimal `json:"net_profit"`
	ProfitFactor         decimal.Decimal `json:"profit_factor"`
	AverageProfit        decimal.Decimal `json:"average_profit"`
	AverageLoss          decimal.Decimal `json:"average_loss"`
	AverageTrade         float64 `json:"average_trade"`
	AverageVolume        float64 `json:"average_volume"`
	LargestWin           float64 `json:"largest_win"`
	LargestLoss          float64 `json:"largest_loss"`
	TotalVolume          decimal.Decimal `json:"total_volume"`
	MaxConsecutiveWins   int     `json:"max_consecutive_wins"`
	MaxConsecutiveLosses int     `json:"max_consecutive_losses"`
	AverageHoldingTime   string  `json:"average_holding_time"`
	TotalDeposit         float64 `json:"total_deposit"`
	TotalWithdrawal      float64 `json:"total_withdrawal"`
	NetDeposit           float64 `json:"net_deposit"`
}

type RiskMetrics struct {
	MaxDrawdown        decimal.Decimal `json:"max_drawdown"`
	MaxDrawdownPercent float64 `json:"max_drawdown_percent"`
	SharpeRatio        decimal.Decimal `json:"sharpe_ratio"`
	SortinoRatio       float64 `json:"sortino_ratio"`
	CalmarRatio        float64 `json:"calmar_ratio"`
	Volatility         float64 `json:"volatility"`
	ValueAtRisk95      float64 `json:"value_at_risk_95"`
	ExpectedShortfall  float64 `json:"expected_shortfall"`
	MaxDailyLoss       float64 `json:"max_daily_loss"`
	MaxWeeklyLoss      float64 `json:"max_weekly_loss"`
	AverageDailyReturn float64 `json:"average_daily_return"`
	ReturnStdDev       float64 `json:"return_std_dev"`
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
	AverageVolume      float64 `json:"average_volume" db:"average_volume"`
	LargestWin         float64 `json:"largest_win" db:"largest_win"`
	LargestLoss        float64 `json:"largest_loss" db:"largest_loss"`
	AverageHoldingTime string  `json:"average_holding_time" db:"average_holding_time"`
}

type DailyEquity struct {
	Date     string  `json:"date"`
	Equity   decimal.Decimal `json:"equity"`
	Balance  decimal.Decimal `json:"balance"`
	Profit   decimal.Decimal `json:"profit"`
	Drawdown float64 `json:"drawdown"`
}

type TradeReport struct {
	AccountID     string         `json:"account_id"`
	StartDate     string         `json:"start_date"`
	EndDate       string         `json:"end_date"`
	TradeStats    TradeStats     `json:"trade_stats"`
	RiskMetrics   RiskMetrics    `json:"risk_metrics"`
	SymbolStats   []*SymbolStats `json:"symbol_stats"`
	DailyEquity   []*DailyEquity `json:"daily_equity"`
	EquityCurve   []float64      `json:"equity_curve"`
	DrawdownCurve []float64      `json:"drawdown_curve"`
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
	MaxFloatingLossAmount  float64 `json:"max_floating_loss_amount"`
	MaxFloatingLossRatio   float64 `json:"max_floating_loss_ratio"`
	MaxFloatingProfitAmount float64 `json:"max_floating_profit_amount"`
	MaxFloatingProfitRatio float64 `json:"max_floating_profit_ratio"`
}

type HourlyStats struct {
	Hour      string  `json:"hour"`
	HourStart int     `json:"hour_start"`
	Trades    int     `json:"trades"`
	Profit    decimal.Decimal `json:"profit"`
	WinRate   decimal.Decimal `json:"win_rate"`
	AvgPnL    float64 `json:"avg_pnl"`
	Lots                   decimal.Decimal `json:"lots"`
	Balance                decimal.Decimal `json:"balance"`
	ProfitFactor           decimal.Decimal `json:"profit_factor"`
	MaxFloatingLossAmount  float64 `json:"max_floating_loss_amount"`
	MaxFloatingLossRatio   float64 `json:"max_floating_loss_ratio"`
	MaxFloatingProfitAmount float64 `json:"max_floating_profit_amount"`
	MaxFloatingProfitRatio float64 `json:"max_floating_profit_ratio"`
}

// WeekdayPnL aggregates closed trades by ISO weekday (1=Monday … 7=Sunday).
type WeekdayPnL struct {
	Weekday int     `json:"weekday"`
	PnL     float64 `json:"pnl"`
	Trades  int     `json:"trades"`
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
	SharePercent float64 `json:"share_percent"` // lot-volume share for pie, 0–100
}

type MonthlyBonusRiskRow struct {
	Symbol    string  `json:"symbol"`
	RiskRatio float64 `json:"risk_ratio"`
}

type MonthlyBonusHoldingRow struct {
	Symbol           string  `json:"symbol"`
	BullsSeconds     float64 `json:"bulls_seconds"`
	ShortTermSeconds float64 `json:"short_term_seconds"`
}

type MonthlyAnalysisBonus struct {
	RiskRatio         float64                   `json:"risk_ratio"`
	Symbols           []*MonthlyBonusSymbol     `json:"symbols"`
	SymbolRisks       []*MonthlyBonusRiskRow    `json:"symbol_risks"`
	SymbolHoldings    []*MonthlyBonusHoldingRow `json:"symbol_holdings"`
	AvgHoldingSeconds float64                   `json:"average_holding_seconds"`
	TotalTrades       int                       `json:"total_trades"`
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
