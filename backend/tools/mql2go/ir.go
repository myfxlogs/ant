package mql2go

// ── Strategy Intent IR ──────────────────────────────────────────────
// Language-agnostic intermediate representation of an MQL strategy.
// Produced by the recognizer; consumed by the Go code generator.

// StrategyIntent holds the complete extracted strategy intent.
type StrategyIntent struct {
	Meta       StrategyMeta
	Params     []ParamSpec
	State      []StateVar
	Entry      []EntryRule
	Exit       []ExitRule
	Modifies     []ModifyRule
	OrderLoops   []OrderLoopRule
	PositionLoops []PositionLoopRule
	Indicators   []IndicatorSpec
	Sizing     *SizingRule
	Risk       []RiskCheck
	Execution  ExecutionModel
	Timer      *TimerRule
	BlindSpots []BlindSpot
}

// StrategyMeta holds strategy-level metadata.
type StrategyMeta struct {
	Name        string
	MQLVersion  string // "mql4" | "mql5"
	Description string
}

// ── Parameters ─────────────────────────────────────────────────────

type ParamType string

const (
	ParamInt    ParamType = "int"
	ParamDouble ParamType = "double"
	ParamString ParamType = "string"
	ParamBool   ParamType = "bool"
	ParamEnum   ParamType = "enum"
)

type ParamGroup string

const (
	GroupEntry  ParamGroup = "entry"
	GroupExit   ParamGroup = "exit"
	GroupSizing ParamGroup = "sizing"
	GroupRisk   ParamGroup = "risk"
	GroupSystem ParamGroup = "system"
)

type ParamSpec struct {
	Name    string
	Label   string
	Type    ParamType
	Default string
	Group   ParamGroup
}

// ── Execution ─────────────────────────────────────────────────────

type ExecKind string

const (
	ExecOnTick     ExecKind = "on_tick"
	ExecOnBar      ExecKind = "on_bar"
	ExecOnInitGrid ExecKind = "on_init_grid"
)

type ExecutionModel struct {
	Kind              ExecKind
	TimeframeFilter   string
	RequireAccount    bool
	RequireOnTrade    bool
}

// ── State ──────────────────────────────────────────────────────────

type StateVar struct {
	Name    string
	GoType  string
	Initial string
}

// ── Entry / Exit ───────────────────────────────────────────────────

type OrderAction string

const (
	ActionMarketBuy  OrderAction = "market_buy"
	ActionMarketSell OrderAction = "market_sell"
	ActionBuyLimit   OrderAction = "buy_limit"
	ActionSellLimit  OrderAction = "sell_limit"
	ActionBuyStop    OrderAction = "buy_stop"
	ActionSellStop   OrderAction = "sell_stop"
)

type ExitTrigger string

const (
	TriggerReverse ExitTrigger = "reverse_signal"
	TriggerMagic   ExitTrigger = "magic_close"
	TriggerDelete  ExitTrigger = "magic_delete"
	TriggerAll     ExitTrigger = "close_all"
)

type EntryRule struct {
	Conditions []string  // Python expression strings (transitional)
	Action     OrderAction
	Volume     string
	Price      string
	StopLoss   string
	TakeProfit string
	Deviation  string
	Magic      string
	Comment    string
}

type ExitRule struct {
	Trigger  ExitTrigger
	Action   string // "position_close" | "order_delete"
	MagicVal string
	MagicMin string
	MagicMax string
}

// ModifyRule represents an OrderModify / CTrade.PositionModify call.
type ModifyRule struct {
	StopLoss   string
	TakeProfit string
	MagicVal   string
	Condition  string // enclosing if condition, if any
	Kind       string // "trailing_stop" | "manual_modify"
}

// OrderLoopRule represents the MQL4 OrdersTotal()+OrderSelect() iteration pattern.
type OrderLoopRule struct {
	BodyActions      []string
	HasMagicFilter   bool
	HasSymbolFilter  bool
	PropertyCalls    []string
}

// PositionLoopRule represents the MQL5 PositionsTotal()+PositionGetTicket() iteration pattern.
type PositionLoopRule struct {
	BodyActions       []string
	HasMagicFilter    bool
	HasSymbolFilter   bool
	PropertyCalls     []string // PositionGetDouble/Integer/String property IDs used
}

// ── Sizing / Risk ──────────────────────────────────────────────────

type SizingKind string

const (
	SizingFixed          SizingKind = "fixed"
	SizingMartingale     SizingKind = "martingale"
	SizingPercentBalance SizingKind = "percent_balance"
)

type SizingRule struct {
	Kind       SizingKind
	Expression string
}

type RiskCheck struct {
	Kind      string
	Condition string
	Action    string
	Trigger   string
}

// ── Timer ──────────────────────────────────────────────────────────

type TimerRule struct {
	IntervalSeconds int
}

// ── Indicators ─────────────────────────────────────────────────────

type IndicatorSpec struct {
	SDKMethod string
	ResultVar string
	Params    map[string]string
	Comment   string
}

// ── Blind Spots ────────────────────────────────────────────────────

type BlindSpot struct {
	ID                 string
	Location           string
	Category           string
	Severity           string // "致命" | "警告" | "信息"
	Description        string
	Handling           string
	UserActionRequired bool
}
