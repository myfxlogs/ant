// Package analysis provides AI-assisted asset analysis including
// multi-timeframe outlook, S/R level detection, volatility classification,
// and AI-generated strategy recommendations.
//
// Computation reuses patterns from internal/ai/regime.go (EMA, ATR, directional
// efficiency) but produces a richer, multi-dimensional report suitable for
// display in the strategy workspace.
package analysis

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"

	"go.uber.org/zap"

	"anttrader/internal/repository"
)

// Analyzer performs comprehensive asset analysis using OHLC k-line data.
type Analyzer struct {
	marketDataRepo repository.MarketDataStore
	log            *zap.Logger
}

// NewAnalyzer creates an asset analyzer.
func NewAnalyzer(marketDataRepo repository.MarketDataStore, log *zap.Logger) *Analyzer {
	return &Analyzer{marketDataRepo: marketDataRepo, log: log}
}

// AnalysisResult contains the complete asset analysis output.
type AnalysisResult struct {
	MTF              MultiTfOutlook
	KeyLevels        []SRLevel
	VolatilityState  string
	VolatilityValue  float64
	AIRecommendation string
	Error            string
}

// MultiTfOutlook holds trend outlooks for each timeframe.
type MultiTfOutlook struct {
	H1 TfOutlook
	H4 TfOutlook
	D1 TfOutlook
	W1 TfOutlook
}

// TfOutlook describes trend direction and strength for one timeframe.
type TfOutlook struct {
	Trend          string  // BULLISH | BEARISH | NEUTRAL
	Strength       float64 // 0-1
	EMAGapPct      float64
	PriceChangePct float64
}

// SRLevel is a detected support or resistance price level.
type SRLevel struct {
	Price    float64
	Type     string // SUPPORT | RESISTANCE
	Strength string // MAJOR | MINOR
	Touches  int32
}

// Analyze runs the full analysis pipeline and calls the callback for each phase.
// This enables the SSE handler to stream progressive results.
func (a *Analyzer) Analyze(
	ctx context.Context,
	symbol, accountID, primaryTF string,
	klineCount int32,
	phaseFn func(phase string, result *AnalysisResult) error,
) (*AnalysisResult, error) {
	if klineCount <= 0 {
		klineCount = 200
	}
	if klineCount > 500 {
		klineCount = 500
	}

	result := &AnalysisResult{}

	// Fetch K-lines for all 4 timeframes in parallel.
	tfs := []string{"1H", "4H", "D1", "W1"}
	barsByTF := make(map[string][]repository.KlineBar)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, tf := range tfs {
		wg.Add(1)
		go func(tf string) {
			defer wg.Done()
			bars, err := a.marketDataRepo.GetKlines(ctx, symbol, "", tf, nil, nil, klineCount)
			if err != nil {
				a.log.Warn("analysis: failed to fetch klines",
					zap.String("symbol", symbol),
					zap.String("timeframe", tf),
					zap.Error(err),
				)
				return
			}
			mu.Lock()
			barsByTF[tf] = bars
			mu.Unlock()
		}(tf)
	}
	wg.Wait()

	// Phase 1: MTF Outlook.
	result.MTF = a.computeMTF(barsByTF)
	if err := phaseFn("mtf_outlook", result); err != nil {
		return result, err
	}

	// Phase 2: S/R levels (from D1 bars as the most reliable).
	if d1Bars, ok := barsByTF["D1"]; ok && len(d1Bars) >= 30 {
		result.KeyLevels = detectSRLevels(d1Bars)
	}
	if err := phaseFn("sr_levels", result); err != nil {
		return result, err
	}

	// Phase 3: Volatility (from primary timeframe, fallback D1).
	volTF := primaryTF
	if _, ok := barsByTF[volTF]; !ok {
		volTF = "D1"
	}
	if bars, ok := barsByTF[volTF]; ok && len(bars) >= 14 {
		result.VolatilityValue, result.VolatilityState = classifyVolatility(bars)
	}
	if err := phaseFn("volatility", result); err != nil {
		return result, err
	}

	// Phase 4: Analysis complete — handler takes over for AI recommendation + final completion.
	// We do NOT send "ai_recommendation" or "complete" here because the handler
	// calls the LLM after Analyze() returns and then sends those phases itself.
	// If we sent "complete" now, the frontend would break out of the SSE loop
	// before receiving the real AI content.
	return result, nil
}

// BuildAIPrompt constructs an LLM prompt from the computed analysis.
func BuildAIPrompt(symbol string, result *AnalysisResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Analyze %s and recommend a trading strategy.\n\n", symbol))

	b.WriteString("## Multi-Timeframe Outlook\n")
	for _, tf := range []struct {
		name string
		o    TfOutlook
	}{
		{"1H", result.MTF.H1},
		{"4H", result.MTF.H4},
		{"D1", result.MTF.D1},
		{"W1", result.MTF.W1},
	} {
		b.WriteString(fmt.Sprintf("- **%s**: %s (strength: %.0f%%, EMA gap: %.2f%%, change: %.2f%%)\n",
			tf.name, tf.o.Trend, tf.o.Strength*100, tf.o.EMAGapPct, tf.o.PriceChangePct))
	}

	b.WriteString("\n## Key Support / Resistance Levels\n")
	for _, lvl := range result.KeyLevels {
		b.WriteString(fmt.Sprintf("- **%.5f** (%s %s, %d touches)\n",
			lvl.Price, lvl.Strength, lvl.Type, lvl.Touches))
	}
	if len(result.KeyLevels) == 0 {
		b.WriteString("- No significant levels detected\n")
	}

	b.WriteString(fmt.Sprintf("\n## Volatility: %s (ATR: %.2f%%)\n", result.VolatilityState, result.VolatilityValue))

	b.WriteString("\n## Task\n")
	b.WriteString("Based on the above data, recommend:\n")
	b.WriteString("1. The most suitable strategy style (trend-following / mean-reversion / breakout / scalping)\n")
	b.WriteString("2. Optimal trade direction bias (long / short / both)\n")
	b.WriteString("3. Suggested stop-loss and take-profit placement relative to the S/R levels\n")
	b.WriteString("4. Key risk factors to watch\n")
	b.WriteString("\nKeep the recommendation concise (4-6 bullet points). Use markdown.")

	return b.String()
}

// computeMTF builds the multi-timeframe outlook from fetched bars.
func (a *Analyzer) computeMTF(barsByTF map[string][]repository.KlineBar) MultiTfOutlook {
	compute := func(bars []repository.KlineBar) TfOutlook {
		if len(bars) < 30 {
			return TfOutlook{Trend: "NEUTRAL", Strength: 0.3}
		}
		n := len(bars)
		closes := make([]float64, n)
		for i, b := range bars {
			closes[i] = b.Close.InexactFloat64()
		}
		// Reverse to chronological order (oldest first).
		for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
			closes[i], closes[j] = closes[j], closes[i]
		}

		ema10 := ema(closes, 10)
		ema30 := ema(closes, 30)
		emaGap := (ema10/ema30 - 1) * 100
		priceChg := (closes[n-1]/closes[0] - 1) * 100
		dirEff := directionalEfficiency(closes)

		strength := clamp01(0.3 + math.Abs(emaGap)*0.03 + dirEff*0.3)

		trend := "NEUTRAL"
		switch {
		case emaGap >= 0.8 && dirEff >= 0.45:
			trend = "BULLISH"
		case emaGap <= -0.8 && dirEff >= 0.45:
			trend = "BEARISH"
		}

		return TfOutlook{
			Trend:          trend,
			Strength:       math.Round(strength*100) / 100,
			EMAGapPct:      math.Round(emaGap*100) / 100,
			PriceChangePct: math.Round(priceChg*100) / 100,
		}
	}

	return MultiTfOutlook{
		H1: compute(barsByTF["1H"]),
		H4: compute(barsByTF["4H"]),
		D1: compute(barsByTF["D1"]),
		W1: compute(barsByTF["W1"]),
	}
}
