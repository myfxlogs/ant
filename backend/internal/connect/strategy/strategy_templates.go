package strategy

import antv1 "alphaforge/gen/proto/ant/v1"

// builtinTemplates returns the built-in Go strategy templates.
func builtinTemplates() []*antv1.StrategyTemplateInfo {
	return []*antv1.StrategyTemplateInfo{
		tplMACrossover(),
		tplRSIMeanReversion(),
		tplBollingerBreakout(),
	}
}

func tplMACrossover() *antv1.StrategyTemplateInfo {
	return &antv1.StrategyTemplateInfo{
		Name:        "MA Crossover",
		Description: "双均线交叉策略 — Go SDK",
	Code: `package main

import (
	"alphaforge/strategy/sdk"
	"github.com/shopspring/decimal"
)

type MACrossStrategy struct {
	fastPeriod int
	slowPeriod int
	entryPct   float64
	slPct      float64
	tpPct      float64
}

func (s *MACrossStrategy) OnInit(ctx sdk.Context) error {
	s.fastPeriod = ctx.ParamInt("fast_period", 10)
	s.slowPeriod = ctx.ParamInt("slow_period", 30)
	s.entryPct = ctx.ParamDecimal("entryPct", decimal.NewFromFloat(0.25)).InexactFloat64()
	s.slPct = ctx.ParamDecimal("stopLossPct", decimal.NewFromFloat(0.03)).InexactFloat64()
	s.tpPct = ctx.ParamDecimal("takeProfitPct", decimal.NewFromFloat(0.06)).InexactFloat64()
	return nil
}

func (s *MACrossStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
	bars := ctx.Bars()
	if bars.Len() < s.slowPeriod+1 {
		return nil, nil
	}

	ind := ctx.Indicators()
	fastNow := ind.EMA(s.fastPeriod, 0)
	slowNow := ind.EMA(s.slowPeriod, 0)
	fastPrev := ind.EMA(s.fastPeriod, 1)
	slowPrev := ind.EMA(s.slowPeriod, 1)

	price := bars.Close(0)
	positions := ctx.Broker().Positions(0)
	hasLong := false
	hasShort := false
	for _, pos := range positions {
		if pos.Side == sdk.SideBuy {
			hasLong = true
		} else if pos.Side == sdk.SideSell {
			hasShort = true
		}
	}

	balance := ctx.Broker().Account().Balance
	volume := balance.Mul(decimal.NewFromFloat(s.entryPct)).Div(price)

	if fastNow > slowNow && fastPrev <= slowPrev {
		if hasShort {
			return &sdk.Signal{Action: sdk.ActionClose, Symbol: ctx.Symbol()}, nil
		}
		if !hasLong {
			return &sdk.Signal{
				Action:     sdk.ActionBuy,
				Symbol:     ctx.Symbol(),
				Volume:     volume,
				Price:      price,
				StopLoss:   price.Mul(decimal.NewFromFloat(1 - s.slPct)),
				TakeProfit: price.Mul(decimal.NewFromFloat(1 + s.tpPct)),
			}, nil
		}
	}

	if fastNow < slowNow && fastPrev >= slowPrev {
		if hasLong {
			return &sdk.Signal{Action: sdk.ActionClose, Symbol: ctx.Symbol()}, nil
		}
		if !hasShort {
			return &sdk.Signal{
				Action:     sdk.ActionSell,
				Symbol:     ctx.Symbol(),
				Volume:     volume,
				Price:      price,
				StopLoss:   price.Mul(decimal.NewFromFloat(1 + s.slPct)),
				TakeProfit: price.Mul(decimal.NewFromFloat(1 - s.tpPct)),
			}, nil
		}
	}

	return nil, nil
}

func (s *MACrossStrategy) OnDeinit(ctx sdk.Context, reason string) error {
	return nil
}
`,
	}
}

func tplRSIMeanReversion() *antv1.StrategyTemplateInfo {
	return &antv1.StrategyTemplateInfo{
		Name:        "RSI Mean Reversion",
		Description: "RSI超买超卖反转策略 — Go SDK",
	Code: `package main

import (
	"alphaforge/strategy/sdk"
	"github.com/shopspring/decimal"
)

type RSIStrategy struct {
	rsiPeriod  int
	oversold   float64
	overbought float64
	entryPct   float64
	slPct      float64
	tpPct      float64
}

func (s *RSIStrategy) OnInit(ctx sdk.Context) error {
	s.rsiPeriod = ctx.ParamInt("rsi_period", 14)
	s.oversold = ctx.ParamDecimal("oversold", decimal.NewFromInt(30)).InexactFloat64()
	s.overbought = ctx.ParamDecimal("overbought", decimal.NewFromInt(70)).InexactFloat64()
	s.entryPct = ctx.ParamDecimal("entryPct", decimal.NewFromFloat(0.25)).InexactFloat64()
	s.slPct = ctx.ParamDecimal("stopLossPct", decimal.NewFromFloat(0.02)).InexactFloat64()
	s.tpPct = ctx.ParamDecimal("takeProfitPct", decimal.NewFromFloat(0.04)).InexactFloat64()
	return nil
}

func (s *RSIStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
	bars := ctx.Bars()
	if bars.Len() < s.rsiPeriod+1 {
		return nil, nil
	}

	rsiVal := ctx.Indicators().RSI(s.rsiPeriod, 0)

	price := bars.Close(0)
	positions := ctx.Broker().Positions(0)
	hasLong := false
	hasShort := false
	for _, pos := range positions {
		if pos.Side == sdk.SideBuy {
			hasLong = true
		} else if pos.Side == sdk.SideSell {
			hasShort = true
		}
	}

	balance := ctx.Broker().Account().Balance
	volume := balance.Mul(decimal.NewFromFloat(s.entryPct)).Div(price)

	if rsiVal < s.oversold {
		if hasShort {
			return &sdk.Signal{Action: sdk.ActionClose, Symbol: ctx.Symbol()}, nil
		}
		if !hasLong {
			return &sdk.Signal{
				Action:     sdk.ActionBuy,
				Symbol:     ctx.Symbol(),
				Volume:     volume,
				Price:      price,
				StopLoss:   price.Mul(decimal.NewFromFloat(1 - s.slPct)),
				TakeProfit: price.Mul(decimal.NewFromFloat(1 + s.tpPct)),
			}, nil
		}
	}

	if rsiVal > s.overbought {
		if hasLong {
			return &sdk.Signal{Action: sdk.ActionClose, Symbol: ctx.Symbol()}, nil
		}
		if !hasShort {
			return &sdk.Signal{
				Action:     sdk.ActionSell,
				Symbol:     ctx.Symbol(),
				Volume:     volume,
				Price:      price,
				StopLoss:   price.Mul(decimal.NewFromFloat(1 + s.slPct)),
				TakeProfit: price.Mul(decimal.NewFromFloat(1 - s.tpPct)),
			}, nil
		}
	}

	return nil, nil
}

func (s *RSIStrategy) OnDeinit(ctx sdk.Context, reason string) error {
	return nil
}
`,
	}
}

func tplBollingerBreakout() *antv1.StrategyTemplateInfo {
	return &antv1.StrategyTemplateInfo{
		Name:        "Bollinger Breakout",
		Description: "布林带突破策略 — Go SDK",
	Code: `package main

import (
	"alphaforge/strategy/sdk"
	"github.com/shopspring/decimal"
)

type BollingerStrategy struct {
	bbPeriod int
	bbStd    float64
	entryPct float64
	slPct    float64
	tpPct    float64
}

func (s *BollingerStrategy) OnInit(ctx sdk.Context) error {
	s.bbPeriod = ctx.ParamInt("bb_period", 20)
	s.bbStd = ctx.ParamDecimal("bb_std", decimal.NewFromFloat(2.0)).InexactFloat64()
	s.entryPct = ctx.ParamDecimal("entryPct", decimal.NewFromFloat(0.25)).InexactFloat64()
	s.slPct = ctx.ParamDecimal("stopLossPct", decimal.NewFromFloat(0.03)).InexactFloat64()
	s.tpPct = ctx.ParamDecimal("takeProfitPct", decimal.NewFromFloat(0.06)).InexactFloat64()
	return nil
}

func (s *BollingerStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
	bars := ctx.Bars()
	if bars.Len() < s.bbPeriod+1 {
		return nil, nil
	}

	upper, _, lower := ctx.Indicators().Bollinger(s.bbPeriod, s.bbStd, 0)

	price := bars.Close(0)
	positions := ctx.Broker().Positions(0)
	hasLong := false
	hasShort := false
	for _, pos := range positions {
		if pos.Side == sdk.SideBuy {
			hasLong = true
		} else if pos.Side == sdk.SideSell {
			hasShort = true
		}
	}

	balance := ctx.Broker().Account().Balance
	volume := balance.Mul(decimal.NewFromFloat(s.entryPct)).Div(price)

	if price.GreaterThan(decimal.NewFromFloat(upper)) {
		if hasShort {
			return &sdk.Signal{Action: sdk.ActionClose, Symbol: ctx.Symbol()}, nil
		}
		if !hasLong {
			return &sdk.Signal{
				Action:     sdk.ActionBuy,
				Symbol:     ctx.Symbol(),
				Volume:     volume,
				Price:      price,
				StopLoss:   price.Mul(decimal.NewFromFloat(1 - s.slPct)),
				TakeProfit: price.Mul(decimal.NewFromFloat(1 + s.tpPct)),
			}, nil
		}
	}

	if price.LessThan(decimal.NewFromFloat(lower)) {
		if hasLong {
			return &sdk.Signal{Action: sdk.ActionClose, Symbol: ctx.Symbol()}, nil
		}
		if !hasShort {
			return &sdk.Signal{
				Action:     sdk.ActionSell,
				Symbol:     ctx.Symbol(),
				Volume:     volume,
				Price:      price,
				StopLoss:   price.Mul(decimal.NewFromFloat(1 + s.slPct)),
				TakeProfit: price.Mul(decimal.NewFromFloat(1 - s.tpPct)),
			}, nil
		}
	}

	return nil, nil
}

func (s *BollingerStrategy) OnDeinit(ctx sdk.Context, reason string) error {
	return nil
}
`,
	}
}
