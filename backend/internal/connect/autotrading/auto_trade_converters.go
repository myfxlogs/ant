package autotrading

import (
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/model"
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
		MaxRiskPercent:     gs.MaxRiskPercent,
		MaxPositions:       int32(gs.MaxPositions),
		MaxLotSize:         gs.MaxLotSize,
		MaxDailyLoss:       gs.MaxDailyLoss.InexactFloat64(),
		MaxDrawdownPercent: gs.MaxDrawdownPercent,
		CreatedAt:          timestamppb.New(gs.CreatedAt),
		UpdatedAt:          timestamppb.New(gs.UpdatedAt),
	}
}

func applyGlobalSettings(existing *model.GlobalSettings, req *antv1.UpdateGlobalSettingsRequest) {
	existing.AutoTradeEnabled = req.GetAutoTradeEnabled()
	existing.MaxRiskPercent = req.GetMaxRiskPercent()
	existing.MaxPositions = int(req.GetMaxPositions())
	existing.MaxLotSize = req.GetMaxLotSize()
	existing.MaxDailyLoss = decimal.NewFromFloat(req.GetMaxDailyLoss())
	existing.MaxDrawdownPercent = req.GetMaxDrawdownPercent()
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
		MaxRiskPercent:     rc.MaxRiskPercent,
		MaxLotSize:         rc.MaxLotSize,
		MaxDailyLoss:       rc.MaxDailyLoss.InexactFloat64(),
		DailyLossUsed:      rc.DailyLossUsed.InexactFloat64(),
		MaxDrawdownPercent: rc.MaxDrawdownPercent,
		MaxPositions:       int32(rc.MaxPositions),
		CreatedAt:          timestamppb.New(rc.CreatedAt),
		UpdatedAt:          timestamppb.New(rc.UpdatedAt),
	}
}

func applyRiskConfig(existing *model.RiskConfig, req *antv1.UpdateRiskConfigRequest) {
	existing.MaxRiskPercent = req.GetMaxRiskPercent()
	existing.MaxLotSize = req.GetMaxLotSize()
	existing.MaxDailyLoss = decimal.NewFromFloat(req.GetMaxDailyLoss())
	existing.MaxDrawdownPercent = req.GetMaxDrawdownPercent()
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
		Volume:     l.Volume.InexactFloat64(),
		Price:      l.Price.InexactFloat64(),
		Ticket:     l.Ticket,
		Profit:     l.Profit.InexactFloat64(),
		CreatedAt:  timestamppb.New(l.CreatedAt),
	}
}
