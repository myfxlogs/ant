package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

// schedule_risk.go —— 调度层面的轻量风控参数。
//
// 这些参数通过 schedule.parameters (map<string,string>) 里以
// "__risk." 为前缀的键传入，使得我们可以在不改 proto schema 的前提下
// 把风控字段从前端传到 runner 这一层。
//
// 支持的键：
//
//   __risk.default_volume         float > 0  策略信号没给 volume 时的默认下单量
//   __risk.stop_loss_price_offset float >= 0 策略信号没给 SL 时的价格距离（绝对价格差）
//   __risk.take_profit_price_offset float >= 0 同上，止盈
//   __risk.max_positions          int >= 1   单个调度在同一 symbol 上允许持有的最大持仓数；
//                                            达到后本次信号跳过下单
//   __risk.max_drawdown_pct       float 0-1  自峰值 equity 的最大回撤比例，超过则自动停用调度
//
// 空值或不合法值 = 该风控项不启用。

const (
	riskKeyDefaultVolume         = "__risk.default_volume"
	riskKeyStopLossPriceOffset   = "__risk.stop_loss_price_offset"
	riskKeyTakeProfitPriceOffset = "__risk.take_profit_price_offset"
	riskKeyMaxPositions          = "__risk.max_positions"
	riskKeyMaxDrawdownPct        = "__risk.max_drawdown_pct"
)

// riskParams 是上面那些键解析后的强类型视图。字段值为 0 / <=0 表示对应风控不启用。
type riskParams struct {
	DefaultVolume         float64
	StopLossPriceOffset   float64
	TakeProfitPriceOffset float64
	MaxPositions          int
	MaxDrawdownPct        float64
}

// parseRiskParams 从 schedule.parameters 中提取所有 __risk.* 键。
// schedule 的 parameters 是 map[string]interface{}（因为是 JSONB 反序列化），
// 所以这里需要对每个值做一次 stringify + parse。
func parseRiskParams(params map[string]interface{}) riskParams {
	if params == nil {
		return riskParams{}
	}
	var rp riskParams
	if f, ok := parseRiskFloat(params[riskKeyDefaultVolume]); ok && f > 0 {
		rp.DefaultVolume = f
	}
	if f, ok := parseRiskFloat(params[riskKeyStopLossPriceOffset]); ok && f > 0 {
		rp.StopLossPriceOffset = f
	}
	if f, ok := parseRiskFloat(params[riskKeyTakeProfitPriceOffset]); ok && f > 0 {
		rp.TakeProfitPriceOffset = f
	}
	if n, ok := parseRiskInt(params[riskKeyMaxPositions]); ok && n >= 1 {
		rp.MaxPositions = n
	}
	if f, ok := parseRiskFloat(params[riskKeyMaxDrawdownPct]); ok && f > 0 && f <= 1 {
		rp.MaxDrawdownPct = f
	}
	return rp
}

// parseRiskFloat 宽松地把 JSON 里可能是 float64 / string / int 的值统一转成 float64。
func parseRiskFloat(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case nil:
		return 0, false
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		f, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

func parseRiskInt(v interface{}) (int, bool) {
	f, ok := parseRiskFloat(v)
	if !ok {
		return 0, false
	}
	return int(f), true
}

// applySignalDefaults 填入策略信号没给但被 risk 参数覆盖了的字段。
// orderType 是 "buy" 或 "sell"，用来决定 SL/TP 的符号方向。
func (rp riskParams) applySignalDefaults(orderType string, req *OrderSendRequest) {
	if req == nil {
		return
	}
	// Normalize orderType so pending orders like buy_limit / sell_stop still
	// inherit SL/TP offset defaults.
	norm := orderType
	if strings.HasPrefix(norm, "buy") {
		norm = "buy"
	} else if strings.HasPrefix(norm, "sell") {
		norm = "sell"
	}
	if req.Volume <= 0 && rp.DefaultVolume > 0 {
		req.Volume = rp.DefaultVolume
	}
	// 仅在信号没提供 SL/TP（= 0）且策略没用 Price（市价单，用实时行情）时才注入。
	// Price 为 0 时 broker 会按当前行情成交，SL/TP 也应基于成交价；这里我们退而求其次，
	// 如果 Price==0 则不注入 SL/TP，等 broker 的 market watch 填回成交价后再由后续 OrderModify 处理。
	// 这是一个保守策略，避免往 broker 发出不合理的 SL/TP。
	if req.Price > 0 {
		if req.StopLoss <= 0 && rp.StopLossPriceOffset > 0 {
			switch norm {
			case "buy":
				req.StopLoss = req.Price - rp.StopLossPriceOffset
			case "sell":
				req.StopLoss = req.Price + rp.StopLossPriceOffset
			}
		}
		if req.TakeProfit <= 0 && rp.TakeProfitPriceOffset > 0 {
			switch norm {
			case "buy":
				req.TakeProfit = req.Price + rp.TakeProfitPriceOffset
			case "sell":
				req.TakeProfit = req.Price - rp.TakeProfitPriceOffset
			}
		}
	}
}

// riskGate 在下单前执行 max_positions 与 max_drawdown_pct 检查。
// 返回 true 表示允许下单；返回 false 表示被风控拦截，调用方应跳过本次下单
// （reason 用于日志 / schedule.last_error）。
//
// 如果触发 max_drawdown_pct，额外会把调度置为 is_active=false（通过 autoDisable 回调）。
type riskGate struct {
	userID    uuid.UUID
	accountID uuid.UUID
	symbol    string
	params    riskParams
	// countPositions 返回当前账户/品种上该调度的活跃持仓数。runner 里注入真实实现。
	countPositions func(ctx context.Context) (int, error)
	// readEquity 返回当前账户权益。用于回撤检查。
	readEquity func(ctx context.Context) (float64, error)
	// peakEquityState 提供回撤跟踪用的 peak equity 读写。
	// 首次调用若 peak<=0，则以当前 equity 初始化为 peak 并返回 peak == 当前 equity。
	peakEquityState func(current float64) (peak float64)
	// autoDisable 在触发 max_drawdown_pct 时被调用，将调度置为停用。
	autoDisable func(ctx context.Context, reason string)
}

func (g *riskGate) allow(ctx context.Context) (bool, string) {
	decision := g.decision(ctx)
	if decision == nil {
		return true, ""
	}
	return decision.Allowed, decision.Reason
}

func (g *riskGate) decision(ctx context.Context) *model.RiskDecision {
	if g == nil {
		return model.AllowRiskDecision(model.RiskDecisionSourceSchedule)
	}
	// 1) max_positions
	if g.params.MaxPositions > 0 && g.countPositions != nil {
		n, err := g.countPositions(ctx)
		if err == nil && n >= g.params.MaxPositions {
			return model.RejectRiskDecision(model.RiskDecisionSourceSchedule, "RISK_MAX_POSITIONS_REACHED", "max_positions reached", false)
		}
	}
	// 2) max_drawdown_pct
	if g.params.MaxDrawdownPct > 0 && g.readEquity != nil && g.peakEquityState != nil {
		cur, err := g.readEquity(ctx)
		if err == nil && cur > 0 {
			peak := g.peakEquityState(cur)
			if peak > 0 {
				dd := (peak - cur) / peak
				if dd >= g.params.MaxDrawdownPct {
					if g.autoDisable != nil {
						g.autoDisable(ctx, "max_drawdown_pct triggered")
					}
					return model.RejectRiskDecision(model.RiskDecisionSourceSchedule, "RISK_MAX_DRAWDOWN_EXCEEDED", "max_drawdown_pct triggered", false)
				}
			}
		}
	}
	return model.AllowRiskDecision(model.RiskDecisionSourceSchedule)
}

// toggleScheduleInactive 用于 max_drawdown_pct 触发后把调度置为 is_active=false。
// 单独抽出是为了方便测试和在 runner 注入。
func (r *StrategyScheduleRunner) toggleScheduleInactive(ctx context.Context, schedule *model.StrategySchedule, reason string) {
	if r == nil || r.scheduleRepo == nil || schedule == nil {
		return
	}
	// 只用 SetActive 的布尔翻转就够了；历史 last_error 由 UpdateLastRun 路径更新。
	_ = r.scheduleRepo.SetActive(ctx, schedule.ID, false)
}
