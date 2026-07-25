package ai

import (
	"context"
	"fmt"
	"time"

	"alphaforge/internal/repository"
	systemai "alphaforge/internal/service/systemai"
)

// ── read_kline tool ──

type ReadKlineTool struct{ repo repository.MarketDataStore }

func NewReadKlineTool(repo repository.MarketDataStore) *ReadKlineTool {
	return &ReadKlineTool{repo: repo}
}

func (t *ReadKlineTool) Name() string { return "read_kline" }
func (t *ReadKlineTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name:        "read_kline",
Description: "读取K线数据并返回市场分析（bar数/日期/价格/EMA/趋势/波动率/近期OHLC）。当你需要了解当前市场状况、分析行情趋势、查看价格形态时调用此工具。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"symbol":    map[string]any{"type": "string", "description": "交易品种代码，例如 BTCUSDm, XAUUSDm"},
					"timeframe": map[string]any{"type": "string", "enum": []string{"1m", "5m", "15m", "30m", "1h", "4h", "1d", "1w"}},
				},
				"required": []string{"symbol", "timeframe"},
			},
		},
	}
}
func (t *ReadKlineTool) Run(ctx context.Context, in ToolInput) ToolOutput {
	bars, err := t.repo.GetKlines(ctx, in.Symbol, "", in.Timeframe, nil, nil, 2000)
	if err != nil {
		return ToolOutput{Success: false, Error: err.Error()}
	}
	if len(bars) == 0 {
		return ToolOutput{Success: true, Output: map[string]any{
			"bars":    0,
			"message": fmt.Sprintf("数据库中无 %s %s 的数据。请如实告诉用户：该品种暂无K线数据。不要编造任何日期或数量。", in.Symbol, in.Timeframe),
		}}
	}

	first := int64(bars[0].CloseTsUnixMs)
	last := int64(bars[len(bars)-1].CloseTsUnixMs)

	type barSummary struct {
		T int64   `json:"t"`
		O float64 `json:"o"`
		H float64 `json:"h"`
		L float64 `json:"l"`
		C float64 `json:"c"`
	}
	recent := bars
	if len(recent) > 10 {
		recent = recent[len(recent)-10:]
	}
	recentBars := make([]barSummary, len(recent))
	for i, b := range recent {
		recentBars[i] = barSummary{
			T: int64(b.CloseTsUnixMs),
			O: b.Open.InexactFloat64(),
			H: b.High.InexactFloat64(),
			L: b.Low.InexactFloat64(),
			C: b.Close.InexactFloat64(),
		}
	}

	ema20 := ema(bars, 20)
	ema50 := ema(bars, 50)
	currentPrice := recentBars[len(recentBars)-1].C

	trend := "ranging"
	trendStrength := "neutral"
	if ema20 > 0 && ema50 > 0 {
		if ema20 > ema50 && currentPrice > ema20 {
			trend = "上升趋势 (uptrend)"
			trendStrength = "bullish"
		} else if ema20 < ema50 && currentPrice < ema20 {
			trend = "下降趋势 (downtrend)"
			trendStrength = "bearish"
		}
	}

	lookback := bars
	if len(lookback) > 50 {
		lookback = lookback[len(lookback)-50:]
	}
	high, low := lookback[0].High.InexactFloat64(), lookback[0].Low.InexactFloat64()
	for _, b := range lookback {
		if h := b.High.InexactFloat64(); h > high {
			high = h
		}
		if l := b.Low.InexactFloat64(); l < low {
			low = l
		}
	}
	rangePct := (high - low) / low * 100

	volLookback := bars
	if len(volLookback) > 20 {
		volLookback = volLookback[len(volLookback)-20:]
	}
	var sumAbsReturn float64
	for i := 1; i < len(volLookback); i++ {
		r := (volLookback[i].Close.InexactFloat64() - volLookback[i-1].Close.InexactFloat64()) / volLookback[i-1].Close.InexactFloat64() * 100
		if r < 0 {
			r = -r
		}
		sumAbsReturn += r
	}
	meanVol := sumAbsReturn / float64(len(volLookback)-1)

	return ToolOutput{
		Success: true,
		Output: map[string]any{
			"symbol":          in.Symbol,
			"timeframe":       in.Timeframe,
			"total_bars":      len(bars),
			"date_from":       time.UnixMilli(first).UTC().Format("2006-01-02"),
			"date_to":         time.UnixMilli(last).UTC().Format("2006-01-02"),
			"current_price":   fmt.Sprintf("%.5f", currentPrice),
			"ema_20":          fmt.Sprintf("%.5f", ema20),
			"ema_50":          fmt.Sprintf("%.5f", ema50),
			"trend":           trend,
			"trend_strength":  trendStrength,
			"recent_high":     fmt.Sprintf("%.5f", high),
			"recent_low":      fmt.Sprintf("%.5f", low),
			"recent_range_pct": fmt.Sprintf("%.2f", rangePct),
			"volatility_pct":  fmt.Sprintf("%.3f", meanVol),
			"recent_bars":     recentBars,
		},
	}
}

func ema(bars []repository.KlineBar, period int) float64 {
	if len(bars) < period {
		return 0
	}
	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close.InexactFloat64()
	}
	k := 2.0 / float64(period+1)
	result := closes[0]
	for i := 1; i < len(closes); i++ {
		result = closes[i]*k + result*(1-k)
	}
	return result
}
