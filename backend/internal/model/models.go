package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Nickname     *string    `json:"nickname" db:"nickname"`
	Avatar       *string    `json:"avatar" db:"avatar"`
	Role         string     `json:"role" db:"role"`
	Status       string     `json:"status" db:"status"`
	LastLoginAt  *time.Time `json:"last_login_at" db:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type MTAccount struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	MTType          string     `json:"mt_type" db:"mt_type"`
	BrokerCompany   string     `json:"broker_company" db:"broker_company"`
	BrokerServer    string     `json:"broker_server" db:"broker_server"`
	BrokerHost      string     `json:"broker_host" db:"broker_host"`
	Login           string     `json:"login" db:"login"`
	Password        string     `json:"-" db:"password"`
	Alias           string     `json:"alias" db:"alias"`
	IsDisabled      bool       `json:"is_disabled" db:"is_disabled"`
	Balance         float64    `json:"balance" db:"balance"`
	Credit          float64    `json:"credit" db:"credit"`
	Equity          float64    `json:"equity" db:"equity"`
	Margin          float64    `json:"margin" db:"margin"`
	FreeMargin      float64    `json:"free_margin" db:"free_margin"`
	MarginLevel     float64    `json:"margin_level" db:"margin_level"`
	Leverage        int        `json:"leverage" db:"leverage"`
	Currency        string     `json:"currency" db:"currency"`
	AccountMethod   string     `json:"account_method" db:"account_method"`
	AccountType     string     `json:"account_type" db:"account_type"`
	IsInvestor      bool       `json:"is_investor" db:"is_investor"`
	AccountStatus   string     `json:"account_status" db:"account_status"`
	StreamStatus    string     `json:"stream_status" db:"stream_status"`
	MTToken         string     `json:"-" db:"mt_token"`
	LastError       string     `json:"last_error" db:"last_error"`
	LastConnectedAt *time.Time `json:"last_connected_at" db:"last_connected_at"`
	LastCheckedAt   *time.Time `json:"last_checked_at" db:"last_checked_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

type Position struct {
	ID           uuid.UUID `json:"id" db:"id"`
	MTAccountID  uuid.UUID `json:"mt_account_id" db:"mt_account_id"`
	Platform     string    `json:"platform" db:"platform"`
	Ticket       int64     `json:"ticket" db:"ticket"`
	Symbol       string    `json:"symbol" db:"symbol"`
	OrderType    int16     `json:"order_type" db:"order_type"`
	Volume       float64   `json:"volume" db:"volume"`
	OpenPrice    float64   `json:"open_price" db:"open_price"`
	CurrentPrice float64   `json:"current_price" db:"current_price"`
	StopLoss     float64   `json:"stop_loss" db:"stop_loss"`
	TakeProfit   float64   `json:"take_profit" db:"take_profit"`
	OpenTime     time.Time `json:"open_time" db:"open_time"`
	Profit       float64   `json:"profit" db:"profit"`
	Swap         float64   `json:"swap" db:"swap"`
	Commission   float64   `json:"commission" db:"commission"`
	Fee          float64   `json:"fee" db:"fee"`
	OrderComment string    `json:"order_comment" db:"order_comment"`
	MagicNumber  int       `json:"magic_number" db:"magic_number"`
	CloseReason  string    `json:"close_reason" db:"close_reason"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Order struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	MTAccountID    uuid.UUID  `json:"mt_account_id" db:"mt_account_id"`
	Platform       string     `json:"platform" db:"platform"`
	Ticket         int64      `json:"ticket" db:"ticket"`
	Symbol         string     `json:"symbol" db:"symbol"`
	OrderType      int16      `json:"order_type" db:"order_type"`
	Volume         float64    `json:"volume" db:"volume"`
	Price          float64    `json:"price" db:"price"`
	StopLimitPrice float64    `json:"stop_limit_price" db:"stop_limit_price"`
	StopLoss       float64    `json:"stop_loss" db:"stop_loss"`
	TakeProfit     float64    `json:"take_profit" db:"take_profit"`
	Expiration     *time.Time `json:"expiration" db:"expiration"`
	ExpirationType string     `json:"expiration_type" db:"expiration_type"`
	PlacedType     string     `json:"placed_type" db:"placed_type"`
	OrderComment   string     `json:"order_comment" db:"order_comment"`
	MagicNumber    int        `json:"magic_number" db:"magic_number"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

type TradeLog struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	AccountID uuid.UUID `json:"account_id" db:"account_id"`
	Action    string    `json:"action" db:"action"`
	Symbol    string    `json:"symbol" db:"symbol"`
	OrderType string    `json:"order_type" db:"order_type"`
	Volume    float64   `json:"volume" db:"volume"`
	Price     float64   `json:"price" db:"price"`
	Ticket    int64     `json:"ticket" db:"ticket"`
	Profit    float64   `json:"profit" db:"profit"`
	Message   string    `json:"message" db:"message"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type KlineData struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Symbol     string    `json:"symbol" db:"symbol"`
	Timeframe  string    `json:"timeframe" db:"timeframe"`
	OpenTime   time.Time `json:"open_time" db:"open_time"`
	CloseTime  time.Time `json:"close_time" db:"close_time"`
	KlineDate  time.Time `json:"kline_date" db:"kline_date"`
	OpenPrice  float64   `json:"open_price" db:"open_price"`
	HighPrice  float64   `json:"high_price" db:"high_price"`
	LowPrice   float64   `json:"low_price" db:"low_price"`
	ClosePrice float64   `json:"close_price" db:"close_price"`
	TickVolume int64     `json:"tick_volume" db:"tick_volume"`
	RealVolume float64   `json:"real_volume" db:"real_volume"`
	Spread     int       `json:"spread" db:"spread"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type TradeStats struct {
	TotalTrades          int     `json:"total_trades"`
	WinningTrades        int     `json:"winning_trades"`
	LosingTrades         int     `json:"losing_trades"`
	BuyTrades            int     `json:"buy_trades"`
	SellTrades           int     `json:"sell_trades"`
	WinRate              float64 `json:"win_rate"`
	TotalProfit          float64 `json:"total_profit"`
	TotalLoss            float64 `json:"total_loss"`
	NetProfit            float64 `json:"net_profit"`
	ProfitFactor         float64 `json:"profit_factor"`
	AverageProfit        float64 `json:"average_profit"`
	AverageLoss          float64 `json:"average_loss"`
	AverageTrade         float64 `json:"average_trade"`
	AverageVolume        float64 `json:"average_volume"`
	LargestWin           float64 `json:"largest_win"`
	LargestLoss          float64 `json:"largest_loss"`
	TotalVolume          float64 `json:"total_volume"`
	MaxConsecutiveWins   int     `json:"max_consecutive_wins"`
	MaxConsecutiveLosses int     `json:"max_consecutive_losses"`
	AverageHoldingTime   string  `json:"average_holding_time"`
	TotalDeposit         float64 `json:"total_deposit"`
	TotalWithdrawal      float64 `json:"total_withdrawal"`
	NetDeposit           float64 `json:"net_deposit"`
}

type RiskMetrics struct {
	MaxDrawdown        float64 `json:"max_drawdown"`
	MaxDrawdownPercent float64 `json:"max_drawdown_percent"`
	SharpeRatio        float64 `json:"sharpe_ratio"`
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
	WinRate            float64 `json:"win_rate" db:"win_rate"`
	TotalProfit        float64 `json:"total_profit" db:"total_profit"`
	TotalLoss          float64 `json:"total_loss" db:"total_loss"`
	NetProfit          float64 `json:"net_profit" db:"net_profit"`
	ProfitFactor       float64 `json:"profit_factor" db:"profit_factor"`
	AverageProfit      float64 `json:"average_profit" db:"average_profit"`
	TotalVolume        float64 `json:"total_volume" db:"total_volume"`
	AverageVolume      float64 `json:"average_volume" db:"average_volume"`
	LargestWin         float64 `json:"largest_win" db:"largest_win"`
	LargestLoss        float64 `json:"largest_loss" db:"largest_loss"`
	AverageHoldingTime string  `json:"average_holding_time" db:"average_holding_time"`
}

type DailyEquity struct {
	Date     string  `json:"date"`
	Equity   float64 `json:"equity"`
	Balance  float64 `json:"balance"`
	Profit   float64 `json:"profit"`
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

type TradeRecord struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	ScheduleID   *uuid.UUID `json:"schedule_id" db:"schedule_id"`
	AccountID    uuid.UUID  `json:"account_id" db:"account_id"`
	Ticket       int64      `json:"ticket" db:"ticket"`
	Symbol       string     `json:"symbol" db:"symbol"`
	OrderType    string     `json:"order_type" db:"order_type"`
	Volume       float64    `json:"volume" db:"volume"`
	OpenPrice    float64    `json:"open_price" db:"open_price"`
	ClosePrice   float64    `json:"close_price" db:"close_price"`
	Profit       float64    `json:"profit" db:"profit"`
	Swap         float64    `json:"swap" db:"swap"`
	Commission   float64    `json:"commission" db:"commission"`
	OpenTime     time.Time  `json:"open_time" db:"open_time"`
	CloseTime    time.Time  `json:"close_time" db:"close_time"`
	StopLoss     float64    `json:"stop_loss" db:"stop_loss"`
	TakeProfit   float64    `json:"take_profit" db:"take_profit"`
	OrderComment string     `json:"order_comment" db:"order_comment"`
	MagicNumber  int        `json:"magic_number" db:"magic_number"`
	Platform     string     `json:"platform" db:"platform"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type MonthlyPnL struct {
	Month      string  `json:"month"`
	MonthNum   int     `json:"month_num"`
	Profit     float64 `json:"profit"`
	Trades     int     `json:"trades"`
	WinTrades  int     `json:"win_trades"`
	LossTrades int     `json:"loss_trades"`
}

type MonthlyAnalysisPoint struct {
	Year   int     `json:"year"`
	Month  int     `json:"month"`
	Change float64 `json:"change"`
	Profit float64 `json:"profit"`
	Lots   float64 `json:"lots"`
	Pips   float64 `json:"pips"`
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

type DailyPnL struct {
	Day                    string  `json:"day"`
	DayNum                 int     `json:"day_num"`
	Date                   string  `json:"date"`
	PnL                    float64 `json:"pnl"`
	Trades                 int     `json:"trades"`
	Lots                   float64 `json:"lots"`
	Balance                float64 `json:"balance"`
	ProfitFactor           float64 `json:"profit_factor"`
	MaxFloatingLossAmount  float64 `json:"max_floating_loss_amount"`
	MaxFloatingLossRatio   float64 `json:"max_floating_loss_ratio"`
	MaxFloatingProfitAmount float64 `json:"max_floating_profit_amount"`
	MaxFloatingProfitRatio float64 `json:"max_floating_profit_ratio"`
}

type HourlyStats struct {
	Hour      string  `json:"hour"`
	HourStart int     `json:"hour_start"`
	Trades    int     `json:"trades"`
	Profit    float64 `json:"profit"`
	WinRate   float64 `json:"win_rate"`
	AvgPnL    float64 `json:"avg_pnl"`
	Lots                   float64 `json:"lots"`
	Balance                float64 `json:"balance"`
	ProfitFactor           float64 `json:"profit_factor"`
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
	Equity  float64 `json:"equity"`
	Balance float64 `json:"balance"`
	Profit  float64 `json:"profit"`
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

// AIConfig AI模型配置
type AIConfig struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Provider    string    `json:"provider" db:"provider"`
	APIKey      string    `json:"-" db:"api_key"` // 不在JSON中暴露
	ModelName   string    `json:"model_name" db:"model_name"`
	MaxTokens   int       `json:"max_tokens" db:"max_tokens"`
	Temperature float64   `json:"temperature" db:"temperature"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type AdminLog struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	AdminID       *uuid.UUID             `json:"admin_id" db:"admin_id"`
	Module        string                 `json:"module" db:"module"`
	ActionType    string                 `json:"action_type" db:"action_type"`
	TargetType    string                 `json:"target_type" db:"target_type"`
	TargetID      string                 `json:"target_id" db:"target_id"`
	IPAddress     string                 `json:"ip_address" db:"ip_address"`
	UserAgent     string                 `json:"user_agent" db:"user_agent"`
	RequestMethod string                 `json:"request_method" db:"request_method"`
	RequestPath   string                 `json:"request_path" db:"request_path"`
	Details       map[string]interface{} `json:"details" db:"details"`
	Success       bool                   `json:"success" db:"success"`
	ErrorMessage  string                 `json:"error_message" db:"error_message"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

type Permission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Code        string    `json:"code" db:"code"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type RolePermission struct {
	Role         string    `json:"role" db:"role"`
	PermissionID uuid.UUID `json:"permission_id" db:"permission_id"`
	GrantedAt    time.Time `json:"granted_at" db:"granted_at"`
}

type SystemConfig struct {
	Key         string    `json:"key" db:"key"`
	Value       string    `json:"value" db:"value"`
	Description string    `json:"description" db:"description"`
	Enabled     *bool     `json:"enabled" db:"enabled"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type DashboardStats struct {
	TotalUsers     int64   `json:"total_users"`
	ActiveUsers    int64   `json:"active_users"`
	TotalAccounts  int64   `json:"total_accounts"`
	OnlineAccounts int64   `json:"online_accounts"`
	TodayTrades    int64   `json:"today_trades"`
	TodayVolume    float64 `json:"today_volume"`
	TodayProfit    float64 `json:"today_profit"`
	SystemLoad     float64 `json:"system_load"`
}

type UserListParams struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Search   string `form:"search"`
	Status   string `form:"status"`
	Role     string `form:"role"`
}

type AccountListParams struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Search   string `form:"search"`
	Status   string `form:"status"`
	MTType   string `form:"mt_type"`
	UserID   string `form:"user_id"`
}

type LogListParams struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	Module     string `form:"module"`
	ActionType string `form:"action_type"`
	StartDate  string `form:"start_date"`
	EndDate    string `form:"end_date"`
	AdminID    string `form:"admin_id"`
}

type TradingSummary struct {
	Period struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	} `json:"period"`
	Overview struct {
		TotalUsers        int64 `json:"total_users"`
		ActiveUsers       int64 `json:"active_users"`
		TotalAccounts     int64 `json:"total_accounts"`
		ConnectedAccounts int64 `json:"connected_accounts"`
	} `json:"overview"`
	Trading struct {
		TotalOrders   int64   `json:"total_orders"`
		ClosedOrders  int64   `json:"closed_orders"`
		PendingOrders int64   `json:"pending_orders"`
		TotalVolume   float64 `json:"total_volume"`
		TotalProfit   float64 `json:"total_profit"`
		TotalLoss     float64 `json:"total_loss"`
		NetProfit     float64 `json:"net_profit"`
	} `json:"trading"`
	ByPlatform map[string]struct {
		Accounts int64   `json:"accounts"`
		Orders   int64   `json:"orders"`
		Volume   float64 `json:"volume"`
	} `json:"by_platform"`
}
