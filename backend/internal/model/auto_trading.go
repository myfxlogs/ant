package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type StrategyScheduleLegacy struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	StrategyID     uuid.UUID  `json:"strategy_id" db:"strategy_id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	TemplateID     uuid.UUID  `json:"template_id" db:"template_id"`
	AccountID      uuid.UUID  `json:"account_id" db:"account_id"`
	Name           string     `json:"name" db:"name"`
	Symbol         string     `json:"symbol" db:"symbol"`
	Timeframe      string     `json:"timeframe" db:"timeframe"`
	Parameters     JSONB      `json:"parameters" db:"parameters"`
	ScheduleType   string     `json:"schedule_type" db:"schedule_type"`
	ScheduleConfig JSONB      `json:"schedule_config" db:"schedule_config"`
	IsActive       bool       `json:"is_active" db:"is_active"`
	LastRunAt      *time.Time `json:"last_run_at" db:"last_run_at"`
	NextRunAt      *time.Time `json:"next_run_at" db:"next_run_at"`
	LastError      string     `json:"last_error" db:"last_error"`
	RunCount       int        `json:"run_count" db:"run_count"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

type ScheduleConfig struct {
	CronExpression string `json:"cron_expression,omitempty"` // cron表达式
	IntervalMs     int64  `json:"interval_ms,omitempty"`     // 间隔毫秒数
	EventTrigger   string `json:"event_trigger,omitempty"`   // 事件触发类型
}

func (s *StrategyScheduleLegacy) GetScheduleConfig() (*ScheduleConfig, error) {
	if len(s.ScheduleConfig) == 0 {
		return nil, nil
	}
	var config ScheduleConfig
	err := json.Unmarshal(s.ScheduleConfig, &config)
	return &config, err
}

func (s *StrategyScheduleLegacy) SetScheduleConfig(config *ScheduleConfig) error {
	if config == nil {
		s.ScheduleConfig = nil
		return nil
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	s.ScheduleConfig = data
	return nil
}

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

type ExecutionSignal struct {
	SignalID   uuid.UUID        `json:"signal_id"`
	Symbol     string           `json:"symbol"`
	Type       string           `json:"type"`
	Volume     float64          `json:"volume"`
	Price      float64          `json:"price"`
	StopLoss   float64          `json:"stop_loss"`
	TakeProfit float64          `json:"take_profit"`
	Result     *ExecutionResult `json:"result,omitempty"`
}

type ExecutionResult struct {
	Ticket int64   `json:"ticket"`
	Profit float64 `json:"profit"`
	Error  string  `json:"error,omitempty"`
}

func (e *StrategyExecution) GetSignals() ([]ExecutionSignal, error) {
	if len(e.Signals) == 0 {
		return nil, nil
	}
	var signals []ExecutionSignal
	err := json.Unmarshal(e.Signals, &signals)
	return signals, err
}

func (e *StrategyExecution) SetSignals(signals []ExecutionSignal) error {
	data, err := json.Marshal(signals)
	if err != nil {
		return err
	}
	e.Signals = data
	return nil
}

func (e *StrategyExecution) GetOrders() ([]ExecutionResult, error) {
	if len(e.Orders) == 0 {
		return nil, nil
	}
	var orders []ExecutionResult
	err := json.Unmarshal(e.Orders, &orders)
	return orders, err
}

func (e *StrategyExecution) SetOrders(orders []ExecutionResult) error {
	data, err := json.Marshal(orders)
	if err != nil {
		return err
	}
	e.Orders = data
	return nil
}

type RiskConfig struct {
	ID                  uuid.UUID `json:"id" db:"id"`
	UserID              uuid.UUID `json:"user_id" db:"user_id"`
	AccountID           uuid.UUID `json:"account_id" db:"account_id"`
	MaxRiskPercent      float64   `json:"max_risk_percent" db:"max_risk_percent"`
	MaxDailyLoss        float64   `json:"max_daily_loss" db:"max_daily_loss"`
	MaxDrawdownPercent  float64   `json:"max_drawdown_percent" db:"max_drawdown_percent"`
	MaxPositions        int       `json:"max_positions" db:"max_positions"`
	MaxLotSize          float64   `json:"max_lot_size" db:"max_lot_size"`
	DailyLossUsed       float64   `json:"daily_loss_used" db:"daily_loss_used"`
	TrailingStopEnabled bool      `json:"trailing_stop_enabled" db:"trailing_stop_enabled"`
	TrailingStopPips    float64   `json:"trailing_stop_pips" db:"trailing_stop_pips"`
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
	MaxRiskPercent      float64   `json:"max_risk_percent" db:"max_risk_percent"`
	MaxPositions        int       `json:"max_positions" db:"max_positions"`
	MaxLotSize          float64   `json:"max_lot_size" db:"max_lot_size"`
	MaxDailyLoss        float64   `json:"max_daily_loss" db:"max_daily_loss"`
	MaxDrawdownPercent  float64   `json:"max_drawdown_percent" db:"max_drawdown_percent"`
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
	Volume     float64   `json:"volume" db:"volume"`
	Price      float64   `json:"price" db:"price"`
	Ticket     int64     `json:"ticket" db:"ticket"`
	Profit     float64   `json:"profit" db:"profit"`
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
	AccountID      uuid.UUID `json:"account_id"`
	Symbol         string    `json:"symbol"`
	StopLossPips   float64   `json:"stop_loss_pips"`
	RiskPercent    float64   `json:"risk_percent"`
	AccountBalance float64   `json:"account_balance"`
}

type PositionSizingResult struct {
	Volume     float64 `json:"volume"`
	RiskAmount float64 `json:"risk_amount"`
	PipValue   float64 `json:"pip_value"`
	LotSize    float64 `json:"lot_size"`
	MaxVolume  float64 `json:"max_volume"`
	MinVolume  float64 `json:"min_volume"`
}

type RiskCheckRequest struct {
	AccountID      uuid.UUID `json:"account_id"`
	Symbol         string    `json:"symbol"`
	Volume         float64   `json:"volume"`
	CurrentBalance float64   `json:"current_balance"`
	CurrentEquity  float64   `json:"current_equity"`
	OpenPositions  int       `json:"open_positions"`
}

type RiskCheckResult struct {
	Allowed            bool          `json:"allowed"`
	Reason             string        `json:"reason,omitempty"`
	CurrentRisk        float64       `json:"current_risk"`
	MaxAllowedRisk     float64       `json:"max_allowed_risk"`
	DailyLossUsed      float64       `json:"daily_loss_used"`
	DailyLossLimit     float64       `json:"daily_loss_limit"`
	PositionCount      int           `json:"position_count"`
	MaxPositions       int           `json:"max_positions"`
	DrawdownPercent    float64       `json:"drawdown_percent"`
	MaxDrawdownPercent float64       `json:"max_drawdown_percent"`
	IsWithinLimits     bool          `json:"is_within_limits"`
	Decision           *RiskDecision `json:"decision,omitempty"`
}

type AutoTradingStatus struct {
	GlobalEnabled    bool               `json:"global_enabled"`
	ActiveStrategies int                `json:"active_strategies"`
	PendingSignals   int                `json:"pending_signals"`
	TodayExecutions  int                `json:"today_executions"`
	TodayProfit      float64            `json:"today_profit"`
	RiskStatus       *RiskStatusSummary `json:"risk_status,omitempty"`
}

type RiskStatusSummary struct {
	DailyLossUsed      float64 `json:"daily_loss_used"`
	DailyLossLimit     float64 `json:"daily_loss_limit"`
	DrawdownPercent    float64 `json:"drawdown_percent"`
	MaxDrawdownPercent float64 `json:"max_drawdown_percent"`
	PositionCount      int     `json:"position_count"`
	MaxPositions       int     `json:"max_positions"`
	IsWithinLimits     bool    `json:"is_within_limits"`
}

const (
	ScheduleTypeCron     = "cron"
	ScheduleTypeInterval = "interval"
	ScheduleTypeEvent    = "event"

	ExecutionStatusRunning   = "running"
	ExecutionStatusCompleted = "completed"
	ExecutionStatusFailed    = "failed"
	ExecutionStatusCancelled = "cancelled"

	LogTypeTrade  = "trade"
	LogTypeSignal = "signal"
	LogTypeError  = "error"
	LogTypeSystem = "system"
)

func NewStrategyScheduleLegacy(userID, templateID, accountID uuid.UUID, scheduleType string, config *ScheduleConfig) *StrategyScheduleLegacy {
	schedule := &StrategyScheduleLegacy{
		ID:           uuid.New(),
		UserID:       userID,
		TemplateID:   templateID,
		AccountID:    accountID,
		ScheduleType: scheduleType,
		IsActive:     false,
		RunCount:     0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if config != nil {
		schedule.SetScheduleConfig(config)
	}
	return schedule
}

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
		MaxRiskPercent:      2.0,
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
		MaxRiskPercent:      2.0,
		MaxPositions:        10,
		MaxLotSize:          100.0,
		MaxDailyLoss:        5000.0,
		MaxDrawdownPercent:  10.0,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
}
