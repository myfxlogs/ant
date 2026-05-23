package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Strategy 策略主表
type Strategy struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	AccountID   uuid.UUID `json:"account_id" db:"account_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Symbol      string    `json:"symbol" db:"symbol"`
	Conditions  JSONB     `json:"conditions" db:"conditions"`     // 策略条件数组
	Actions     JSONB     `json:"actions" db:"actions"`           // 交易动作数组
	RiskControl JSONB     `json:"risk_control" db:"risk_control"` // 风控参数
	Status      string    `json:"status" db:"status"`             // active, paused, stopped
	AutoExecute bool      `json:"auto_execute" db:"auto_execute"` // 是否自动执行
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// StrategyCondition 策略条件
type StrategyCondition struct {
	Type   string                 `json:"type"`   // price_above_ma, price_below_ma, rsi_overbought, etc.
	Period int                    `json:"period"` // 周期
	Value  float64                `json:"value"`  // 阈值
	Params map[string]interface{} `json:"params"` // 额外参数
}

// StrategyAction 策略动作
type StrategyAction struct {
	Type       string  `json:"type"`        // buy, sell, close
	Volume     float64 `json:"volume"`      // 交易量
	StopLoss   float64 `json:"stop_loss"`   // 止损点数
	TakeProfit float64 `json:"take_profit"` // 止盈点数
	Comment    string  `json:"comment"`     // 备注
}

// RiskControl 风控参数
type RiskControl struct {
	MaxPositions  int     `json:"max_positions"`    // 最大持仓数
	MaxLossPerDay float64 `json:"max_loss_per_day"` // 每日最大亏损
	MaxDrawdown   float64 `json:"max_drawdown"`     // 最大回撤
}

// StrategySignal 策略信号
type StrategySignal struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	TemplateID uuid.UUID  `json:"template_id" db:"template_id"`
	AccountID  uuid.UUID  `json:"account_id" db:"account_id"`
	Symbol     string     `json:"symbol" db:"symbol"`
	SignalType string     `json:"signal_type" db:"signal_type"` // buy, sell, close
	Volume     float64    `json:"volume" db:"volume"`
	Price      float64    `json:"price" db:"price"`
	StopLoss   float64    `json:"stop_loss" db:"stop_loss"`
	TakeProfit float64    `json:"take_profit" db:"take_profit"`
	Reason     string     `json:"reason" db:"reason"`
	Status     string     `json:"status" db:"status"` // pending, confirmed, executed, cancelled
	ExecutedAt *time.Time `json:"executed_at" db:"executed_at"`
	Ticket     int64      `json:"ticket" db:"ticket"`
	Profit     float64    `json:"profit" db:"profit"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// JSONB 自定义类型用于处理PostgreSQL的JSONB类型
type JSONB []byte

// Value 实现 driver.Valuer 接口
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

// Scan 实现 sql.Scanner 接口
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	*j = bytes
	return nil
}

// MarshalJSON 实现 json.Marshaler 接口
func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (j *JSONB) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*j = nil
		return nil
	}
	*j = data
	return nil
}

// GetConditions 解析策略条件
func (s *Strategy) GetConditions() ([]StrategyCondition, error) {
	if len(s.Conditions) == 0 {
		return nil, nil
	}
	var conditions []StrategyCondition
	err := json.Unmarshal(s.Conditions, &conditions)
	return conditions, err
}

// SetConditions 设置策略条件
func (s *Strategy) SetConditions(conditions []StrategyCondition) error {
	data, err := json.Marshal(conditions)
	if err != nil {
		return err
	}
	s.Conditions = data
	return nil
}

// GetActions 解析策略动作
func (s *Strategy) GetActions() ([]StrategyAction, error) {
	if len(s.Actions) == 0 {
		return nil, nil
	}
	var actions []StrategyAction
	err := json.Unmarshal(s.Actions, &actions)
	return actions, err
}

// SetActions 设置策略动作
func (s *Strategy) SetActions(actions []StrategyAction) error {
	data, err := json.Marshal(actions)
	if err != nil {
		return err
	}
	s.Actions = data
	return nil
}

// GetRiskControl 解析风控参数
func (s *Strategy) GetRiskControl() (*RiskControl, error) {
	if len(s.RiskControl) == 0 {
		return nil, nil
	}
	var rc RiskControl
	err := json.Unmarshal(s.RiskControl, &rc)
	return &rc, err
}

// SetRiskControl 设置风控参数
func (s *Strategy) SetRiskControl(rc *RiskControl) error {
	if rc == nil {
		s.RiskControl = nil
		return nil
	}
	data, err := json.Marshal(rc)
	if err != nil {
		return err
	}
	s.RiskControl = data
	return nil
}

// 策略状态常量
const (
	StrategyStatusActive  = "active"
	StrategyStatusPaused  = "paused"
	StrategyStatusStopped = "stopped"
)

// 信号状态常量
const (
	SignalStatusPending   = "pending"
	SignalStatusConfirmed = "confirmed"
	SignalStatusExecuted  = "executed"
	SignalStatusCancelled = "cancelled"
)

// 信号类型常量
const (
	SignalTypeBuy   = "buy"
	SignalTypeSell  = "sell"
	SignalTypeClose = "close"
)

// 条件类型常量
const (
	ConditionTypePriceAboveMA    = "price_above_ma"
	ConditionTypePriceBelowMA    = "price_below_ma"
	ConditionTypeRSIOverbought   = "rsi_overbought"
	ConditionTypeRSIOversold     = "rsi_oversold"
	ConditionTypeMACDCrossUp     = "macd_cross_up"
	ConditionTypeMACDCrossDown   = "macd_cross_down"
	ConditionTypeBollingerUpper  = "bollinger_upper"
	ConditionTypeBollingerLower  = "bollinger_lower"
	ConditionTypePriceAboveLevel = "price_above_level"
	ConditionTypePriceBelowLevel = "price_below_level"
)

// 动作类型常量
const (
	ActionTypeBuy   = "buy"
	ActionTypeSell  = "sell"
	ActionTypeClose = "close"
)

// NewStrategy 创建新策略
func NewStrategy(userID, accountID uuid.UUID, name, symbol string) *Strategy {
	now := time.Now()
	return &Strategy{
		ID:          uuid.New(),
		UserID:      userID,
		AccountID:   accountID,
		Name:        name,
		Symbol:      symbol,
		Conditions:  []byte("[]"),
		Actions:     []byte("[]"),
		RiskControl: []byte("{}"),
		Status:      StrategyStatusActive,
		AutoExecute: false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewStrategySignal 创建新策略信号
func NewStrategySignal(userID, templateID, accountID uuid.UUID, symbol, signalType string, volume, price, stopLoss, takeProfit float64, reason string) *StrategySignal {
	return &StrategySignal{
		ID:         uuid.New(),
		UserID:     userID,
		TemplateID: templateID,
		AccountID:  accountID,
		Symbol:     symbol,
		SignalType: signalType,
		Volume:     volume,
		Price:      price,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
		Reason:     reason,
		Status:     SignalStatusPending,
		CreatedAt:  time.Now(),
	}
}

// IsActive 检查策略是否激活
func (s *Strategy) IsActive() bool {
	return s.Status == StrategyStatusActive
}

// IsAutoExecuteEnabled 检查是否启用自动执行
func (s *Strategy) IsAutoExecuteEnabled() bool {
	return s.AutoExecute && s.IsActive()
}

// CanGenerateSignal 检查是否可以生成信号
func (s *Strategy) CanGenerateSignal() bool {
	return s.IsActive()
}
