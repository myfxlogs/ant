package autotrading

import (
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
)

// --- GlobalSettings ---

func globalSettingsToProto(gs *model.GlobalSettings) *antv1.GlobalSettings {
	if gs == nil {
		return nil
	}
	return &antv1.GlobalSettings{
		Id:                 gs.ID.String(),
		UserId:             gs.UserID.String(),
		AutoTradeEnabled:   gs.AutoTradeEnabled,
		MaxRiskPercent:     gs.MaxRiskPercent.String(),
		MaxPositions:       int32(gs.MaxPositions),
		MaxLotSize:         gs.MaxLotSize.String(),
		MaxDailyLoss:       gs.MaxDailyLoss.String(),
		MaxDrawdownPercent: gs.MaxDrawdownPercent.String(),
		CreatedAt:          timestamppb.New(gs.CreatedAt),
		UpdatedAt:          timestamppb.New(gs.UpdatedAt),
	}
}

func applyGlobalSettings(existing *model.GlobalSettings, req *antv1.UpdateGlobalSettingsRequest) {
	existing.AutoTradeEnabled = req.GetAutoTradeEnabled()
	existing.MaxRiskPercent = parseDecimalSafe(req.GetMaxRiskPercent())
	existing.MaxPositions = int(req.GetMaxPositions())
	existing.MaxLotSize = parseDecimalSafe(req.GetMaxLotSize())
	existing.MaxDailyLoss = parseDecimalSafe(req.GetMaxDailyLoss())
	existing.MaxDrawdownPercent = parseDecimalSafe(req.GetMaxDrawdownPercent())
}

// --- RiskConfig ---

func riskConfigToProto(rc *model.RiskConfig) *antv1.RiskConfig {
	if rc == nil {
		return nil
	}
	return &antv1.RiskConfig{
		Id:                 rc.ID.String(),
		UserId:             rc.UserID.String(),
		AccountId:          rc.AccountID.String(),
		MaxRiskPercent:     rc.MaxRiskPercent.String(),
		MaxLotSize:         rc.MaxLotSize.String(),
		MaxDailyLoss:       rc.MaxDailyLoss.String(),
		DailyLossUsed:      rc.DailyLossUsed.String(),
		MaxDrawdownPercent: rc.MaxDrawdownPercent.String(),
		MaxPositions:       int32(rc.MaxPositions),
		CreatedAt:          timestamppb.New(rc.CreatedAt),
		UpdatedAt:          timestamppb.New(rc.UpdatedAt),
	}
}

func applyRiskConfig(existing *model.RiskConfig, req *antv1.UpdateRiskConfigRequest) {
	existing.MaxRiskPercent = parseDecimalSafe(req.GetMaxRiskPercent())
	existing.MaxLotSize = parseDecimalSafe(req.GetMaxLotSize())
	existing.MaxDailyLoss = parseDecimalSafe(req.GetMaxDailyLoss())
	existing.MaxDrawdownPercent = parseDecimalSafe(req.GetMaxDrawdownPercent())
	existing.MaxPositions = int(req.GetMaxPositions())
}

// --- TradingLog ---

func tradingLogToProto(l *model.TradingLog) *antv1.TradingLog {
	if l == nil {
		return nil
	}
	return &antv1.TradingLog{
		Id:         l.ID.String(),
		UserId:     l.UserID.String(),
		AccountId:  l.AccountID.String(),
		StrategyId: l.StrategyID.String(),
		LogType:    l.LogType,
		Action:     l.Action,
		Symbol:     l.Symbol,
		Details:    l.Message,
		Volume:     l.Volume.String(),
		Price:      l.Price.String(),
		Ticket:     l.Ticket,
		Profit:     l.Profit.String(),
		CreatedAt:  timestamppb.New(l.CreatedAt),
	}
}

func parseDecimalSafe(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}
