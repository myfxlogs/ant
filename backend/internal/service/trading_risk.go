package service

import (
	"context"
	"errors"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"anttrader/internal/model"
)

type RiskErrorCode string

const (
	RiskAccountTradeDisabled     RiskErrorCode = "RISK_ACCOUNT_TRADE_DISABLED"
	RiskSymbolTradeDisabled      RiskErrorCode = "RISK_SYMBOL_TRADE_DISABLED"
	RiskMarketSessionClosed      RiskErrorCode = "RISK_MARKET_SESSION_CLOSED"
	RiskVolumeInvalid            RiskErrorCode = "RISK_VOLUME_INVALID"
	RiskOrderTypeUnsupported     RiskErrorCode = "RISK_ORDER_TYPE_UNSUPPORTED"
	RiskStopDistanceTooClose     RiskErrorCode = "RISK_STOP_DISTANCE_TOO_CLOSE"
	RiskOrderFrozenZone          RiskErrorCode = "RISK_ORDER_FROZEN_ZONE"
	RiskMarginInsufficient       RiskErrorCode = "RISK_MARGIN_INSUFFICIENT"
	RiskMaxOpenPositionsExceeded RiskErrorCode = "RISK_MAX_OPEN_POSITIONS_EXCEEDED"
	RiskMaxPendingOrdersExceeded RiskErrorCode = "RISK_MAX_PENDING_ORDERS_EXCEEDED"
	RiskInternalRuleUnavailable  RiskErrorCode = "RISK_INTERNAL_RULE_UNAVAILABLE"
)

type RiskError struct {
	Code        RiskErrorCode
	Reason      string
	UserMessage string
	Retryable   bool
	ContextJSON string
}

func (e *RiskError) Error() string {
	if e == nil {
		return ""
	}
	return string(e.Code)
}

func NewRiskError(code RiskErrorCode, reason string, retryable bool) error {
	return &RiskError{Code: code, Reason: reason, Retryable: retryable}
}

func AsRiskError(err error) (*RiskError, bool) {
	if err == nil {
		return nil, false
	}
	var re *RiskError
	if errors.As(err, &re) && re != nil {
		return re, true
	}
	return nil, false
}

func RiskDecisionFromError(err error, source string) *model.RiskDecision {
	if err == nil {
		return model.AllowRiskDecision(source)
	}
	if re, ok := AsRiskError(err); ok && re != nil {
		return model.RejectRiskDecision(source, string(re.Code), re.Reason, re.Retryable)
	}
	msg := err.Error()
	return model.RejectRiskDecision(source, msg, msg, false)
}

func RiskErrorFromDecision(decision *model.RiskDecision) error {
	if decision == nil || decision.Allowed {
		return nil
	}
	code := RiskErrorCode(decision.Code)
	if code == "" {
		code = RiskInternalRuleUnavailable
	}
	return NewRiskError(code, decision.Reason, decision.Retryable)
}

type AccountRule struct {
	TradeEnabled     bool
	MaxOpenPositions int
	MaxPendingOrders int
}

type SymbolRule struct {
	Symbol                string
	TradeEnabled          bool
	OrderTypeSupport      map[string]struct{}
	MinLot                float64
	MaxLot                float64
	LotStep               float64
	MinStopDistancePoints int
	FreezeDistancePoints  int
	Point                 float64
}

type RiskSnapshot struct {
	FreeMargin         float64
	OpenPositionsCount int
	PendingOrdersCount int
}

type RiskValidationContext struct {
	Account     *model.MTAccount
	AccountRule AccountRule
	SymbolRule  SymbolRule
	Snapshot    RiskSnapshot
}

type riskRuleResolver struct{}

func newRiskRuleResolver() *riskRuleResolver {
	return &riskRuleResolver{}
}

func (r *riskRuleResolver) resolve(account *model.MTAccount, symbol string, openPositionsCount int) (*RiskValidationContext, error) {
	if account == nil {
		return nil, NewRiskError(RiskInternalRuleUnavailable, "account context unavailable", true)
	}
	sr, err := r.resolveSymbolRule(account.MTType, symbol)
	if err != nil {
		return nil, err
	}
	accountRule := AccountRule{
		TradeEnabled:     !account.IsDisabled && !account.IsInvestor,
		MaxOpenPositions: envInt("ANTRADER_RISK_MAX_OPEN_POSITIONS", 20),
		MaxPendingOrders: envInt("ANTRADER_RISK_MAX_PENDING_ORDERS", 50),
	}
	return &RiskValidationContext{
		Account:     account,
		AccountRule: accountRule,
		SymbolRule:  sr,
		Snapshot: RiskSnapshot{
			FreeMargin:         account.FreeMargin,
			OpenPositionsCount: openPositionsCount,
			PendingOrdersCount: 0,
		},
	}, nil
}

func (r *riskRuleResolver) resolveSymbolRule(mtType string, symbol string) (SymbolRule, error) {
	s := strings.TrimSpace(symbol)
	if s == "" {
		return SymbolRule{}, NewRiskError(RiskSymbolTradeDisabled, "symbol is empty", false)
	}
	point := envFloat("ANTRADER_RISK_DEFAULT_POINT", 0.00001)
	if strings.Contains(strings.ToUpper(s), "JPY") {
		point = envFloat("ANTRADER_RISK_JPY_POINT", 0.001)
	}
	return SymbolRule{
		Symbol:                s,
		TradeEnabled:          true,
		OrderTypeSupport:      supportedOrderTypes(mtType),
		MinLot:                envFloat("ANTRADER_RISK_MIN_LOT", 0.01),
		MaxLot:                envFloat("ANTRADER_RISK_MAX_LOT", 100),
		LotStep:               envFloat("ANTRADER_RISK_LOT_STEP", 0.01),
		MinStopDistancePoints: envInt("ANTRADER_RISK_MIN_STOP_DISTANCE_POINTS", 150),
		FreezeDistancePoints:  envInt("ANTRADER_RISK_FREEZE_DISTANCE_POINTS", 50),
		Point:                 point,
	}, nil
}

func validateOrderSendRisk(ctx context.Context, vctx *RiskValidationContext, req *OrderSendRequest) error {
	if vctx == nil {
		return NewRiskError(RiskInternalRuleUnavailable, "risk context unavailable", true)
	}
	if req == nil {
		return NewRiskError(RiskInternalRuleUnavailable, "request unavailable", false)
	}
	if err := validateTradingEnabled(vctx); err != nil {
		return err
	}
	if err := validateMarketSession(ctx); err != nil {
		return err
	}
	if err := validateVolume(vctx.SymbolRule, req.Volume); err != nil {
		return err
	}
	if err := validateOrderType(vctx.SymbolRule, req.Type); err != nil {
		return err
	}
	if err := validateStopDistance(vctx.SymbolRule, req.Price, req.StopLoss, req.TakeProfit); err != nil {
		return err
	}
	if err := validateMargin(vctx.Snapshot.FreeMargin); err != nil {
		return err
	}
	if err := validateOpenPositions(vctx.AccountRule, vctx.Snapshot.OpenPositionsCount); err != nil {
		return err
	}
	if err := validatePendingOrders(vctx.AccountRule, vctx.Snapshot.PendingOrdersCount); err != nil {
		return err
	}
	return nil
}

func validateOrderModifyRisk(ctx context.Context, vctx *RiskValidationContext, req *OrderModifyRequest) error {
	if vctx == nil {
		return NewRiskError(RiskInternalRuleUnavailable, "risk context unavailable", true)
	}
	if req == nil || req.Ticket <= 0 {
		return NewRiskError(RiskInternalRuleUnavailable, "invalid ticket", false)
	}
	if err := validateTradingEnabled(vctx); err != nil {
		return err
	}
	if err := validateMarketSession(ctx); err != nil {
		return err
	}
	if err := validateStopDistance(vctx.SymbolRule, req.Price, req.StopLoss, req.TakeProfit); err != nil {
		return err
	}
	if err := validateFreezeDistance(vctx.SymbolRule, req.Price, req.StopLoss, req.TakeProfit); err != nil {
		return err
	}
	return nil
}

func validateOrderCloseRisk(vctx *RiskValidationContext, req *OrderCloseRequest, position *PositionResponse) error {
	if vctx == nil {
		return NewRiskError(RiskInternalRuleUnavailable, "risk context unavailable", true)
	}
	if req == nil || req.Ticket <= 0 {
		return NewRiskError(RiskInternalRuleUnavailable, "invalid ticket", false)
	}
	if err := validateTradingEnabled(vctx); err != nil {
		return err
	}
	if req.Volume <= 0 {
		return NewRiskError(RiskVolumeInvalid, "volume must be greater than zero", false)
	}
	if position == nil {
		return NewRiskError(RiskInternalRuleUnavailable, "position not found for close", false)
	}
	if req.Volume-position.Volume > 1e-8 {
		return NewRiskError(RiskVolumeInvalid, "close volume exceeds opened volume", false)
	}
	return nil
}

func validateTradingEnabled(vctx *RiskValidationContext) error {
	if !vctx.AccountRule.TradeEnabled {
		return NewRiskError(RiskAccountTradeDisabled, "account trade is disabled", false)
	}
	if !vctx.SymbolRule.TradeEnabled {
		return NewRiskError(RiskSymbolTradeDisabled, "symbol trade is disabled", false)
	}
	return nil
}

func validateMarketSession(ctx context.Context) error {
	now := time.Now().UTC()
	if deadline, ok := ctx.Deadline(); ok && !deadline.IsZero() {
		now = deadline.Add(-time.Second).UTC()
	}
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return NewRiskError(RiskMarketSessionClosed, "market session closed on weekend", true)
	}
	return nil
}

func validateVolume(sr SymbolRule, volume float64) error {
	if volume < sr.MinLot-1e-8 || volume > sr.MaxLot+1e-8 {
		return NewRiskError(RiskVolumeInvalid, "volume out of min/max range", false)
	}
	step := sr.LotStep
	if step <= 0 {
		return NewRiskError(RiskInternalRuleUnavailable, "invalid lot step", false)
	}
	steps := math.Round((volume - sr.MinLot) / step)
	rebuilt := sr.MinLot + steps*step
	if math.Abs(rebuilt-volume) > 1e-8 {
		return NewRiskError(RiskVolumeInvalid, "volume does not match lot step", false)
	}
	return nil
}

func validateOrderType(sr SymbolRule, orderType string) error {
	if _, ok := sr.OrderTypeSupport[strings.ToLower(strings.TrimSpace(orderType))]; !ok {
		return NewRiskError(RiskOrderTypeUnsupported, "order type is unsupported", false)
	}
	return nil
}

func validateStopDistance(sr SymbolRule, price float64, stopLoss float64, takeProfit float64) error {
	if price <= 0 {
		return nil
	}
	minDistance := float64(sr.MinStopDistancePoints) * sr.Point
	if stopLoss > 0 && math.Abs(price-stopLoss) < minDistance {
		return NewRiskError(RiskStopDistanceTooClose, "stop loss too close", false)
	}
	if takeProfit > 0 && math.Abs(price-takeProfit) < minDistance {
		return NewRiskError(RiskStopDistanceTooClose, "take profit too close", false)
	}
	return nil
}

func validateFreezeDistance(sr SymbolRule, price float64, stopLoss float64, takeProfit float64) error {
	if price <= 0 || sr.FreezeDistancePoints <= 0 {
		return nil
	}
	freezeDistance := float64(sr.FreezeDistancePoints) * sr.Point
	if stopLoss > 0 && math.Abs(price-stopLoss) < freezeDistance {
		return NewRiskError(RiskOrderFrozenZone, "stop loss in freeze zone", true)
	}
	if takeProfit > 0 && math.Abs(price-takeProfit) < freezeDistance {
		return NewRiskError(RiskOrderFrozenZone, "take profit in freeze zone", true)
	}
	return nil
}

func validateMargin(freeMargin float64) error {
	minRequired := envFloat("ANTRADER_RISK_MIN_REQUIRED_FREE_MARGIN", 0)
	if freeMargin < minRequired {
		return NewRiskError(RiskMarginInsufficient, "insufficient free margin", false)
	}
	return nil
}

func validateOpenPositions(ar AccountRule, openCount int) error {
	if ar.MaxOpenPositions > 0 && openCount >= ar.MaxOpenPositions {
		return NewRiskError(RiskMaxOpenPositionsExceeded, "maximum open positions exceeded", false)
	}
	return nil
}

func validatePendingOrders(ar AccountRule, pendingCount int) error {
	if ar.MaxPendingOrders > 0 && pendingCount >= ar.MaxPendingOrders {
		return NewRiskError(RiskMaxPendingOrdersExceeded, "maximum pending orders exceeded", false)
	}
	return nil
}

func isSupportedOrderType(mtType string, orderType string) bool {
	switch strings.ToUpper(strings.TrimSpace(mtType)) {
	case "MT4":
		switch strings.ToLower(orderType) {
		case "buy":
			return true
		case "sell":
			return true
		case "buy_limit":
			return true
		case "sell_limit":
			return true
		case "buy_stop":
			return true
		case "sell_stop":
			return true
		}
	case "MT5":
		switch strings.ToLower(orderType) {
		case "buy":
			return true
		case "sell":
			return true
		case "buy_limit":
			return true
		case "sell_limit":
			return true
		case "buy_stop":
			return true
		case "sell_stop":
			return true
		case "buy_stop_limit":
			return true
		case "sell_stop_limit":
			return true
		}
	}
	return false
}

func supportedOrderTypes(mtType string) map[string]struct{} {
	out := map[string]struct{}{
		"buy":        {},
		"sell":       {},
		"buy_limit":  {},
		"sell_limit": {},
		"buy_stop":   {},
		"sell_stop":  {},
	}
	if strings.EqualFold(strings.TrimSpace(mtType), "MT5") {
		out["buy_stop_limit"] = struct{}{}
		out["sell_stop_limit"] = struct{}{}
	}
	return out
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func envFloat(key string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return v
}
