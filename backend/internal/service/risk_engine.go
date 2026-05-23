package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"anttrader/internal/model"
)

type RiskEngine struct{}

func NewRiskEngine() *RiskEngine {
	return &RiskEngine{}
}

func (e *RiskEngine) CheckManualOrderSend(ctx context.Context, vctx *RiskValidationContext, req *OrderSendRequest) *model.RiskDecision {
	return RiskDecisionFromError(validateOrderSendRisk(ctx, vctx, req), model.RiskDecisionSourceManual)
}

func (e *RiskEngine) CheckManualOrderModify(ctx context.Context, vctx *RiskValidationContext, req *OrderModifyRequest) *model.RiskDecision {
	return RiskDecisionFromError(validateOrderModifyRisk(ctx, vctx, req), model.RiskDecisionSourceManual)
}

func (e *RiskEngine) CheckManualOrderClose(vctx *RiskValidationContext, req *OrderCloseRequest, position *PositionResponse) *model.RiskDecision {
	return RiskDecisionFromError(validateOrderCloseRisk(vctx, req, position), model.RiskDecisionSourceManual)
}

func (e *RiskEngine) CheckAuto(req *model.RiskCheckRequest, cfg *model.RiskConfig) *model.RiskCheckResult {
	result := &model.RiskCheckResult{
		Allowed:        true,
		IsWithinLimits: true,
		Decision:       model.AllowRiskDecision(model.RiskDecisionSourceAuto),
	}
	if req != nil {
		result.PositionCount = req.OpenPositions
	}
	if req == nil {
		return result
	}
	if strings.TrimSpace(req.Symbol) == "" {
		result.SetDecision(model.RejectRiskDecision(model.RiskDecisionSourceAuto, "RISK_SYMBOL_EMPTY", "symbol is required", false))
		return result
	}
	if req.Volume <= 0 || math.IsNaN(req.Volume) || math.IsInf(req.Volume, 0) {
		result.SetDecision(model.RejectRiskDecision(model.RiskDecisionSourceAuto, "RISK_VOLUME_INVALID", "volume must be positive", false))
		return result
	}
	if req.OpenPositions < 0 {
		result.SetDecision(model.RejectRiskDecision(model.RiskDecisionSourceAuto, "RISK_POSITION_COUNT_INVALID", "open positions cannot be negative", false))
		return result
	}
	if req.CurrentBalance < 0 || math.IsNaN(req.CurrentBalance) || math.IsInf(req.CurrentBalance, 0) {
		result.SetDecision(model.RejectRiskDecision(model.RiskDecisionSourceAuto, "RISK_BALANCE_INVALID", "balance cannot be negative or invalid", false))
		return result
	}
	if req.CurrentEquity < 0 || math.IsNaN(req.CurrentEquity) || math.IsInf(req.CurrentEquity, 0) {
		result.SetDecision(model.RejectRiskDecision(model.RiskDecisionSourceAuto, "RISK_EQUITY_INVALID", "equity cannot be negative or invalid", false))
		return result
	}
	if cfg == nil {
		return result
	}
	result.MaxPositions = cfg.MaxPositions
	result.DailyLossLimit = cfg.MaxDailyLoss
	result.DailyLossUsed = cfg.DailyLossUsed
	result.MaxDrawdownPercent = cfg.MaxDrawdownPercent
	result.MaxAllowedRisk = cfg.MaxRiskPercent
	if req.CurrentBalance > 0 {
		result.DrawdownPercent = (req.CurrentBalance - req.CurrentEquity) / req.CurrentBalance * 100
	}
	if cfg.MaxPositions > 0 && req.OpenPositions >= cfg.MaxPositions {
		result.SetDecision(model.RejectRiskDecision(model.RiskDecisionSourceAuto, "RISK_MAX_POSITIONS_REACHED", fmt.Sprintf("已达到最大持仓数量限制 (%d/%d)", req.OpenPositions, cfg.MaxPositions), false))
		return result
	}
	if cfg.MaxLotSize > 0 && req.Volume > cfg.MaxLotSize {
		result.SetDecision(model.RejectRiskDecision(model.RiskDecisionSourceAuto, "RISK_MAX_LOT_SIZE_EXCEEDED", fmt.Sprintf("交易量 %.2f 超过最大限制 %.2f", req.Volume, cfg.MaxLotSize), false))
		return result
	}
	if cfg.MaxDailyLoss > 0 && cfg.DailyLossUsed >= cfg.MaxDailyLoss {
		result.SetDecision(model.RejectRiskDecision(model.RiskDecisionSourceAuto, "RISK_DAILY_LOSS_LIMIT_EXCEEDED", fmt.Sprintf("已达到每日亏损限制 (%.2f/%.2f)", cfg.DailyLossUsed, cfg.MaxDailyLoss), false))
		return result
	}
	if cfg.MaxDrawdownPercent > 0 && result.DrawdownPercent >= cfg.MaxDrawdownPercent {
		result.SetDecision(model.RejectRiskDecision(model.RiskDecisionSourceAuto, "RISK_MAX_DRAWDOWN_EXCEEDED", fmt.Sprintf("当前回撤 %.2f%% 已达到最大限制 %.2f%%", result.DrawdownPercent, cfg.MaxDrawdownPercent), false))
		return result
	}
	return result
}

func (e *RiskEngine) CheckSchedule(ctx context.Context, gate *riskGate) *model.RiskDecision {
	if gate == nil {
		return model.AllowRiskDecision(model.RiskDecisionSourceSchedule)
	}
	return gate.decision(ctx)
}
