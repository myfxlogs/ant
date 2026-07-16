package model

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/google/uuid"
)

type StrategyExecution struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	TemplateID   uuid.UUID  `json:"template_id" db:"template_id"`
	ScheduleID   uuid.UUID  `json:"schedule_id" db:"schedule_id"`
	AccountID    uuid.UUID  `json:"account_id" db:"account_id"`
	Status       string     `json:"status" db:"status"`
	Signals      JSONB      `json:"signals" db:"signals"`
	Orders       JSONB      `json:"orders" db:"orders"`
	ErrorMessage string     `json:"error_message" db:"error_message"`
	StartedAt    time.Time  `json:"started_at" db:"started_at"`
	CompletedAt  *time.Time `json:"completed_at" db:"completed_at"`
}

type RiskConfig struct {
	ID                  uuid.UUID `json:"id" db:"id"`
	UserID              uuid.UUID `json:"user_id" db:"user_id"`
	AccountID           uuid.UUID `json:"account_id" db:"account_id"`
	MaxRiskPercent      decimal.Decimal `json:"max_risk_percent" db:"max_risk_percent"`
	MaxDailyLoss        decimal.Decimal   `json:"max_daily_loss" db:"max_daily_loss"`
	MaxDrawdownPercent  decimal.Decimal `json:"max_drawdown_percent" db:"max_drawdown_percent"`
	MaxPositions        int       `json:"max_positions" db:"max_positions"`
	MaxLotSize          decimal.Decimal `json:"max_lot_size" db:"max_lot_size"`
	DailyLossUsed       decimal.Decimal   `json:"daily_loss_used" db:"daily_loss_used"`
	TrailingStopEnabled bool            `json:"trailing_stop_enabled" db:"trailing_stop_enabled"`
	TrailingStopPips    decimal.Decimal `json:"trailing_stop_pips" db:"trailing_stop_pips"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

type GlobalSettings struct {
	ID                  uuid.UUID `json:"id" db:"id"`
	UserID              uuid.UUID `json:"user_id" db:"user_id"`
	AutoTradeEnabled    bool      `json:"auto_trade_enabled" db:"auto_trade_enabled"`
	NotificationEnabled bool      `json:"notification_enabled" db:"notification_enabled"`
	EmailNotification   bool      `json:"email_notification" db:"email_notification"`
	SmsNotification     bool      `json:"sms_notification" db:"sms_notification"`
	MaxRiskPercent      decimal.Decimal `json:"max_risk_percent" db:"max_risk_percent"`
	MaxPositions        int       `json:"max_positions" db:"max_positions"`
	MaxLotSize          decimal.Decimal `json:"max_lot_size" db:"max_lot_size"`
	MaxDailyLoss        decimal.Decimal   `json:"max_daily_loss" db:"max_daily_loss"`
	MaxDrawdownPercent  decimal.Decimal `json:"max_drawdown_percent" db:"max_drawdown_percent"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

type TradingLog struct {
	ID         uuid.UUID `json:"id" db:"id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	AccountID  uuid.UUID `json:"account_id" db:"account_id"`
	StrategyID uuid.UUID `json:"strategy_id" db:"strategy_id"`
	LogType    string    `json:"log_type" db:"log_type"`
	Action     string    `json:"action" db:"action"`
	Symbol     string    `json:"symbol" db:"symbol"`
	Details    string    `json:"details" db:"details"`
	Volume     decimal.Decimal   `json:"volume" db:"volume"`
	Price      decimal.Decimal   `json:"price" db:"price"`
	Ticket     int64     `json:"ticket" db:"ticket"`
	Profit     decimal.Decimal   `json:"profit" db:"profit"`
	Message    string    `json:"message" db:"message"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

func NewTradingLog(userID uuid.UUID, logType, action, symbol, message string) *TradingLog {
	return &TradingLog{
		ID:        uuid.New(),
		UserID:    userID,
		LogType:   logType,
		Action:    action,
		Symbol:    symbol,
		Message:   message,
		CreatedAt: time.Now(),
	}
}

type PositionSizingRequest struct {
	AccountID      uuid.UUID
	Symbol         string
	StopLossPips   decimal.Decimal
	RiskPercent    decimal.Decimal
	AccountBalance decimal.Decimal
}

type PositionSizingResult struct {
	Volume     decimal.Decimal
	RiskAmount decimal.Decimal
	PipValue   decimal.Decimal
	LotSize    decimal.Decimal
	MaxVolume  decimal.Decimal
	MinVolume  decimal.Decimal
}

type RiskCheckRequest struct {
	AccountID      uuid.UUID
	Symbol         string
	Volume         decimal.Decimal
	CurrentBalance decimal.Decimal
	CurrentEquity  decimal.Decimal
	OpenPositions  int
}

type RiskCheckResult struct {
	Allowed            bool
	Reason             string
	CurrentRisk        decimal.Decimal
	MaxAllowedRisk     decimal.Decimal
	DailyLossUsed      decimal.Decimal
	DailyLossLimit     decimal.Decimal
	PositionCount      int
	MaxPositions       int
	DrawdownPercent    decimal.Decimal
	MaxDrawdownPercent decimal.Decimal
	IsWithinLimits     bool
	Decision           *RiskDecision
}

type AutoTradingStatus struct {
	GlobalEnabled    bool
	ActiveStrategies int
	PendingSignals   int
	TodayExecutions  int
	TodayProfit      decimal.Decimal
	RiskStatus       *RiskStatusSummary
}

type RiskStatusSummary struct {
	DailyLossUsed      decimal.Decimal
	DailyLossLimit     decimal.Decimal
	DrawdownPercent    decimal.Decimal
	MaxDrawdownPercent decimal.Decimal
	PositionCount      int
	MaxPositions       int
	IsWithinLimits     bool
}

const (
	ScheduleTypeCron     = "cron"
	ScheduleTypeInterval = "interval"
	ScheduleTypeEvent    = "event"

	ExecutionStatusRunning = "running"

	LogTypeTrade  = "trade"
	LogTypeSignal = "signal"
	LogTypeError  = "error"
	LogTypeSystem = "system"
)

func NewStrategyExecution(userID, templateID, accountID uuid.UUID) *StrategyExecution {
	return &StrategyExecution{
		ID:         uuid.New(),
		UserID:     userID,
		TemplateID: templateID,
		AccountID:  accountID,
		Status:     ExecutionStatusRunning,
		StartedAt:  time.Now(),
	}
}

func NewRiskConfig(userID uuid.UUID, accountID uuid.UUID) *RiskConfig {
	return &RiskConfig{
		ID:                  uuid.New(),
		UserID:              userID,
		AccountID:           accountID,
		MaxRiskPercent:      decimal.NewFromFloat(2.0),
		MaxPositions:        5,
		TrailingStopEnabled: false,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
}

func NewGlobalSettings(userID uuid.UUID) *GlobalSettings {
	return &GlobalSettings{
		ID:                  uuid.New(),
		UserID:              userID,
		AutoTradeEnabled:    false,
		NotificationEnabled: true,
		EmailNotification:   false,
		SmsNotification:     false,
		MaxRiskPercent:      decimal.NewFromFloat(2.0),
		MaxPositions:        10,
		MaxLotSize:          decimal.NewFromFloat(100.0),
		MaxDailyLoss:        decimal.NewFromFloat(5000),
		MaxDrawdownPercent:  decimal.NewFromFloat(10.0),
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
}
