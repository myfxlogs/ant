package strategy

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go"
)

// ShadowVerifier runs a shadow backtest alongside a live strategy session
// to verify that the VM produces consistent signals in both modes.
//
// Bars arriving from the live stream are accumulated and periodically fed
// through the backtest engine. The last signal from the shadow backtest is
// compared with the last live signal. Discrepancies are logged as warnings.
//
// This is a read-only verification layer — it never interferes with live
// order dispatch. It runs on a background goroutine with a configurable
// interval (default: every 50 bars or 5 minutes, whichever comes first).
type ShadowVerifier struct {
	code     string
	cfg      backtest.Config
	log      *zap.Logger
	mu       sync.Mutex
	bars     []sdk.Bar
	liveSigs []shadowSignal
	runner   *mql2go.VMRunner
	stopCh   chan struct{}
}

type shadowSignal struct {
	barTime int64
	action  string
	volume  string
	price   string
}

// NewShadowVerifier creates a verifier for the given strategy code and backtest config.
func NewShadowVerifier(code string, cfg backtest.Config, log *zap.Logger) *ShadowVerifier {
	return &ShadowVerifier{
		code:   code,
		cfg:    cfg,
		log:    log,
		stopCh: make(chan struct{}),
	}
}

// Start launches the background verification goroutine.
func (sv *ShadowVerifier) Start(ctx context.Context) {
	go sv.loop(ctx)
}

// Stop signals the background goroutine to exit.
func (sv *ShadowVerifier) Stop() {
	select {
	case <-sv.stopCh:
	default:
		close(sv.stopCh)
	}
}

// RecordBar adds a live bar to the shadow window.
func (sv *ShadowVerifier) RecordBar(bar sdk.Bar) {
	sv.mu.Lock()
	sv.bars = append(sv.bars, bar)
	if len(sv.bars) > maxContextBars {
		sv.bars = sv.bars[len(sv.bars)-maxContextBars:]
	}
	sv.mu.Unlock()
}

// RecordLiveSignal records a signal dispatched by the live runner.
func (sv *ShadowVerifier) RecordLiveSignal(barTime int64, action, volume, price string) {
	sv.mu.Lock()
	sv.liveSigs = append(sv.liveSigs, shadowSignal{
		barTime: barTime,
		action:  action,
		volume:  volume,
		price:   price,
	})
	if len(sv.liveSigs) > 200 {
		sv.liveSigs = sv.liveSigs[len(sv.liveSigs)-200:]
	}
	sv.mu.Unlock()
}

func (sv *ShadowVerifier) loop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sv.stopCh:
			return
		case <-ticker.C:
			sv.verify(ctx)
		}
	}
}

// verify runs a shadow backtest on accumulated bars and compares with live signals.
func (sv *ShadowVerifier) verify(ctx context.Context) {
	sv.mu.Lock()
	bars := make([]sdk.Bar, len(sv.bars))
	copy(bars, sv.bars)
	liveSigs := make([]shadowSignal, len(sv.liveSigs))
	copy(liveSigs, sv.liveSigs)
	sv.mu.Unlock()

	if len(bars) < 10 {
		return
	}

	if sv.runner == nil {
		r, err := mql2go.CompileMQL(sv.code)
		if err != nil {
			sv.log.Warn("ShadowVerifier: compile failed", zap.Error(err))
			return
		}
		sv.runner = r
	}

	cfg := sv.cfg
	cfg.StartDate = time.UnixMilli(bars[0].Timestamp)
	cfg.EndDate = time.UnixMilli(bars[len(bars)-1].Timestamp)

	engine := backtest.New(cfg, sv.runner, bars)
	result, err := engine.Run(ctx)
	if err != nil {
		sv.log.Warn("ShadowVerifier: backtest failed", zap.Error(err))
		return
	}

	btTrades := len(result.Trades)
	btSignals := extractBacktestSignals(result.Trades)

	mismatches := compareSignals(liveSigs, btSignals)
	if len(mismatches) > 0 {
		sv.log.Warn("ShadowVerifier: signal mismatch detected",
			zap.Int("live_signals", len(liveSigs)),
			zap.Int("backtest_trades", btTrades),
			zap.Int("backtest_signals", len(btSignals)),
			zap.Int("mismatches", len(mismatches)),
			zap.Strings("details", mismatches),
		)
	} else {
		sv.log.Info("ShadowVerifier: consistency check passed",
			zap.Int("bars", len(bars)),
			zap.Int("live_signals", len(liveSigs)),
			zap.Int("backtest_trades", btTrades),
		)
	}
}

func extractBacktestSignals(trades []backtest.Trade) []shadowSignal {
	out := make([]shadowSignal, 0, len(trades))
	for _, t := range trades {
		action := "buy"
		if t.Side == sdk.SideSell {
			action = "sell"
		}
		out = append(out, shadowSignal{
			barTime: t.EntryTime.UnixMilli(),
			action:  action,
			volume:  t.Volume.String(),
			price:   t.EntryPrice.String(),
		})
	}
	return out
}

func compareSignals(live, backtest []shadowSignal) []string {
	var mismatches []string
	maxLen := len(live)
	if len(backtest) < maxLen {
		maxLen = len(backtest)
	}

	for i := 0; i < maxLen; i++ {
		l := live[i]
		b := backtest[i]
		if l.action != b.action {
			mismatches = append(mismatches,
				"signal["+strconv.Itoa(i)+"]: action live="+l.action+" vs backtest="+b.action)
			continue
		}
		lv, _ := decimal.NewFromString(l.volume)
		bv, _ := decimal.NewFromString(b.volume)
		if !lv.Equal(bv) {
			mismatches = append(mismatches,
				"signal["+strconv.Itoa(i)+"]: volume live="+l.volume+" vs backtest="+b.volume)
		}
	}

	if len(live) != len(backtest) {
		mismatches = append(mismatches,
			"count: live="+strconv.Itoa(len(live))+" vs backtest="+strconv.Itoa(len(backtest)))
	}

	return mismatches
}
