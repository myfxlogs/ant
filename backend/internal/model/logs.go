package model

import (
	"time"

	"github.com/google/uuid"
)

type ConnectionEventType string

const (
	EventTypeConnect    ConnectionEventType = "connect"
	EventTypeDisconnect ConnectionEventType = "disconnect"
	EventTypeReconnect  ConnectionEventType = "reconnect"
	EventTypeError      ConnectionEventType = "error"
	EventTypeHeartbeat  ConnectionEventType = "heartbeat"
)

type ConnectionStatus string

const (
	ConnectionStatusSuccess ConnectionStatus = "success"
	ConnectionStatusFailed  ConnectionStatus = "failed"
	ConnectionStatusTimeout ConnectionStatus = "timeout"
)

type AccountConnectionLog struct {
	ID                     uuid.UUID           `db:"id" json:"id"`
	UserID                 uuid.UUID           `db:"user_id" json:"user_id"`
	AccountID              uuid.UUID           `db:"account_id" json:"account_id"`
	EventType              ConnectionEventType `db:"event_type" json:"event_type"`
	Status                 ConnectionStatus    `db:"status" json:"status"`
	Message                string              `db:"message" json:"message"`
	ErrorDetail            string              `db:"error_detail" json:"error_detail,omitempty"`
	ServerHost             string              `db:"server_host" json:"server_host"`
	ServerPort             int                 `db:"server_port" json:"server_port"`
	LoginID                int64               `db:"login_id" json:"login_id"`
	ConnectionDurationSecs int64               `db:"connection_duration_seconds" json:"connection_duration_seconds"`
	CreatedAt              time.Time           `db:"created_at" json:"created_at"`
}

func NewAccountConnectionLog(userID, accountID uuid.UUID, eventType ConnectionEventType, status ConnectionStatus) *AccountConnectionLog {
	return &AccountConnectionLog{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: accountID,
		EventType: eventType,
		Status:    status,
		CreatedAt: time.Now(),
	}
}

type StrategyExecutionLogStatus string

const (
	StrategyExecutionStatusPending   StrategyExecutionLogStatus = "pending"
	StrategyExecutionStatusRunning   StrategyExecutionLogStatus = "running"
	StrategyExecutionStatusCompleted StrategyExecutionLogStatus = "completed"
	StrategyExecutionStatusFailed    StrategyExecutionLogStatus = "failed"
	StrategyExecutionStatusSkipped   StrategyExecutionLogStatus = "skipped"
)

type StrategyExecutionLogSignalType string

const (
	StrategySignalTypeBuy    StrategyExecutionLogSignalType = "buy"
	StrategySignalTypeSell   StrategyExecutionLogSignalType = "sell"
	StrategySignalTypeClose  StrategyExecutionLogSignalType = "close"
	StrategySignalTypeHold   StrategyExecutionLogSignalType = "hold"
	StrategySignalTypeModify StrategyExecutionLogSignalType = "modify"
)

type StrategyExecutionLog struct {
	ID               uuid.UUID                      `db:"id" json:"id"`
	UserID           uuid.UUID                      `db:"user_id" json:"user_id"`
	ScheduleID       *uuid.UUID                     `db:"schedule_id" json:"schedule_id,omitempty"`
	TemplateID       *uuid.UUID                     `db:"template_id" json:"template_id,omitempty"`
	AccountID        *uuid.UUID                     `db:"account_id" json:"account_id,omitempty"`
	Symbol           string                         `db:"symbol" json:"symbol"`
	Timeframe        string                         `db:"timeframe" json:"timeframe"`
	Status           StrategyExecutionLogStatus     `db:"status" json:"status"`
	SignalType       StrategyExecutionLogSignalType `db:"signal_type" json:"signal_type,omitempty"`
	SignalPrice      float64                        `db:"signal_price" json:"signal_price,omitempty"`
	SignalVolume     float64                        `db:"signal_volume" json:"signal_volume,omitempty"`
	SignalStopLoss   float64                        `db:"signal_stop_loss" json:"signal_stop_loss,omitempty"`
	SignalTakeProfit float64                        `db:"signal_take_profit" json:"signal_take_profit,omitempty"`
	ExecutedOrderID  string                         `db:"executed_order_id" json:"executed_order_id,omitempty"`
	ExecutedPrice    float64                        `db:"executed_price" json:"executed_price,omitempty"`
	ExecutedVolume   float64                        `db:"executed_volume" json:"executed_volume,omitempty"`
	Profit           float64                        `db:"profit" json:"profit,omitempty"`
	ErrorMessage     string                         `db:"error_message" json:"error_message,omitempty"`
	ExecutionTimeMs  int64                          `db:"execution_time_ms" json:"execution_time_ms"`
	KlineData        interface{}                    `db:"kline_data" json:"kline_data,omitempty"`
	StrategyParams   interface{}                    `db:"strategy_params" json:"strategy_params,omitempty"`
	CreatedAt        time.Time                      `db:"created_at" json:"created_at"`
}

func NewStrategyExecutionLog(userID uuid.UUID, symbol, timeframe string) *StrategyExecutionLog {
	return &StrategyExecutionLog{
		ID:         uuid.New(),
		UserID:     userID,
		ScheduleID: nil,
		TemplateID: nil,
		AccountID:  nil,
		Symbol:     symbol,
		Timeframe:  timeframe,
		Status:     StrategyExecutionStatusPending,
		CreatedAt:  time.Now(),
	}
}

type OrderHistoryType string

const (
	OrderHistoryTypeBuy       OrderHistoryType = "buy"
	OrderHistoryTypeSell      OrderHistoryType = "sell"
	OrderHistoryTypeBuyLimit  OrderHistoryType = "buy_limit"
	OrderHistoryTypeSellLimit OrderHistoryType = "sell_limit"
	OrderHistoryTypeBuyStop   OrderHistoryType = "buy_stop"
	OrderHistoryTypeSellStop  OrderHistoryType = "sell_stop"
)

type OrderHistory struct {
	ID          uuid.UUID        `db:"id" json:"id"`
	UserID      uuid.UUID        `db:"user_id" json:"user_id"`
	AccountID   uuid.UUID        `db:"account_id" json:"account_id"`
	Ticket      int64            `db:"ticket" json:"ticket"`
	OrderType   OrderHistoryType `db:"order_type" json:"order_type"`
	Symbol      string           `db:"symbol" json:"symbol"`
	Volume      float64          `db:"volume" json:"volume"`
	OpenPrice   float64          `db:"open_price" json:"open_price"`
	ClosePrice  float64          `db:"close_price" json:"close_price,omitempty"`
	OpenTime    time.Time        `db:"open_time" json:"open_time"`
	CloseTime   *time.Time       `db:"close_time" json:"close_time,omitempty"`
	StopLoss    float64          `db:"stop_loss" json:"stop_loss,omitempty"`
	TakeProfit  float64          `db:"take_profit" json:"take_profit,omitempty"`
	Profit      float64          `db:"profit" json:"profit"`
	Commission  float64          `db:"commission" json:"commission"`
	Swap        float64          `db:"swap" json:"swap"`
	Comment     string           `db:"comment" json:"comment,omitempty"`
	MagicNumber int64            `db:"magic_number" json:"magic_number,omitempty"`
	IsAutoTrade bool             `db:"is_auto_trade" json:"is_auto_trade"`
	ScheduleID  uuid.UUID        `db:"schedule_id" json:"schedule_id,omitempty"`
	CreatedAt   time.Time        `db:"created_at" json:"created_at"`
}

type OperationType string

const (
	OperationTypeCreate OperationType = "create"
	OperationTypeUpdate OperationType = "update"
	OperationTypeDelete OperationType = "delete"
	OperationTypeLogin  OperationType = "login"
	OperationTypeLogout OperationType = "logout"
	OperationTypeExport OperationType = "export"
	OperationTypeImport OperationType = "import"
)

type OperationStatus string

const (
	OperationStatusSuccess   OperationStatus = "success"
	OperationStatusFailed    OperationStatus = "failed"
	OperationStatusRunning   OperationStatus = "running"
	OperationStatusCompleted OperationStatus = "completed"
)

type SystemOperationLog struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	UserID        uuid.UUID       `db:"user_id" json:"user_id"`
	OperationType OperationType   `db:"operation_type" json:"operation_type"`
	Module        string          `db:"module" json:"module"`
	ResourceType  string          `db:"resource_type" json:"resource_type,omitempty"`
	ResourceID    uuid.UUID       `db:"resource_id" json:"resource_id,omitempty"`
	Action        string          `db:"action" json:"action"`
	OldValue      interface{}     `db:"old_value" json:"old_value,omitempty"`
	NewValue      interface{}     `db:"new_value" json:"new_value,omitempty"`
	IPAddress     string          `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent     string          `db:"user_agent" json:"user_agent,omitempty"`
	Status        OperationStatus `db:"status" json:"status"`
	ErrorMessage  string          `db:"error_message" json:"error_message,omitempty"`
	DurationMs    int64           `db:"duration_ms" json:"duration_ms"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
}

func NewSystemOperationLog(userID uuid.UUID, opType OperationType, module, action string) *SystemOperationLog {
	return &SystemOperationLog{
		ID:            uuid.New(),
		UserID:        userID,
		OperationType: opType,
		Module:        module,
		Action:        action,
		CreatedAt:     time.Now(),
	}
}

type LogQueryParams struct {
	Page      int    `form:"page" json:"page"`
	PageSize  int    `form:"page_size" json:"page_size"`
	AccountID string `form:"account_id" json:"account_id"`
	ScheduleID string `form:"schedule_id" json:"schedule_id"`
	Symbol    string `form:"symbol" json:"symbol"`
	StartDate string `form:"start_date" json:"start_date"`
	EndDate   string `form:"end_date" json:"end_date"`
	Status    string `form:"status" json:"status"`
	Type      string `form:"type" json:"type"`
	Module    string `form:"module" json:"module"`
	Action    string `form:"action" json:"action"`
	ResourceType string `form:"resource_type" json:"resource_type"`
	ResourceID string `form:"resource_id" json:"resource_id"`
}
