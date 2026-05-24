// V1-LEGACY: will be replaced by M7.1-7.4 cards. Do not extend; new code goes alongside.
// Package quantengine wires factor evaluation and strategy execution into a single process.
// Per-strategy goroutine isolation + snapshot persistence + hot-reload.
package quantengine

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// SignalHandler receives signals and routes them to OMS.
// strategyID is the DB UUID of the strategy that produced this signal.
// signalID is a unique audit identifier for tracing.
type SignalHandler func(strategyID, symbol, side string, qty float64, reason string)

// RunQuantEngine starts the quant engine with the given specs and optional signal handler.
func RunQuantEngine(mux *http.ServeMux, specs []*StrategySpec, log *zap.Logger) *RuntimeManager {
	return RunQuantEngineWithSignalHandler(mux, specs, nil, log)
}

// RunQuantEngineWithSignalHandler starts quant-engine with an optional signal handler for OMS wiring.
func RunQuantEngineWithSignalHandler(
	mux *http.ServeMux,
	specs []*StrategySpec,
	onSignal SignalHandler,
	log *zap.Logger,
) *RuntimeManager {
	ctx := context.Background()

	// ── RuntimeManager with per-strategy isolation ──
	runtimeMgr := NewRuntimeManager(log)

	for _, spec := range specs {
		if spec == nil {
			continue
		}
		rt, err := NewStrategyRuntime(spec, onSignal, log)
		if err != nil {
			log.Warn("runtime creation failed", zap.String("spec", spec.Name), zap.Error(err))
			continue
		}
		rt.strategyID = spec.Name // use name as id when no DB
		runtimeMgr.Add(ctx, rt)
		log.Info("runtime started", zap.String("strategy", spec.Name))
	}

	if mux != nil {
		mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
		})
	}

	// ── Bar-driven evaluation loop ──
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runtimeMgr.OnBar()
			}
		}
	}()

	log.Info("quant-engine starting",
		zap.Int("runtimes", runtimeMgr.Count()),
	)

	return runtimeMgr
}

// DefaultDemoSpec returns a demo SMA crossover strategy spec for testing.
func DefaultDemoSpec() *StrategySpec {
	return &StrategySpec{
		Name:             "demo_sma_e2e",
		Version:          "1.0.0",
		CanonicalSymbols: []string{"BTCUSD"},
		Period:           "1h",
		Factors: map[string]string{
			"sma20": "sma($close, 20)",
			"sma60": "sma($close, 60)",
		},
		SignalRule: "sma20 > sma60 ? 1 : -1",
		Sizing:     map[string]any{"type": "fixed_lots", "lots": 0.1},
	}
}
