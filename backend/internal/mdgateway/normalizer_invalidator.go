package mdgateway

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"go.uber.org/zap"
)

// PGListener abstracts a pgx connection for LISTEN/NOTIFY.
type PGListener interface {
	WaitForNotification(ctx context.Context) (payload string, err error)
	Close() error
}

// NormalizerInvalidator listens for PG NOTIFY broker_symbols_changed events
// and invalidates the normalizer cache for affected (broker, symbol_raw) pairs.
// Falls back to 30s ticker polling when the LISTEN connection is lost (ADR-0011 §2.3).
type NormalizerInvalidator struct {
	cancel       context.CancelFunc
	log          *zap.Logger
	onInvalidate func(broker, symbolRaw string)
	mu           sync.Mutex
	running      bool
}

// NewNormalizerInvalidator creates an invalidator. onInvalidate is called
// when a broker_symbols change is detected, e.g. normalizer.cache.Remove(key).
func NewNormalizerInvalidator(log *zap.Logger, onInvalidate func(broker, symbolRaw string)) *NormalizerInvalidator {
	return &NormalizerInvalidator{
		log:          log,
		onInvalidate: onInvalidate,
	}
}

// Start begins listening. If pgListener is nil, falls back to 30s ticker.
func (ni *NormalizerInvalidator) Start(ctx context.Context, pgListener PGListener) {
	ni.mu.Lock()
	defer ni.mu.Unlock()
	if ni.running {
		return
	}
	ni.running = true

	ctx, ni.cancel = context.WithCancel(ctx)

	if pgListener != nil {
		go ni.listenLoop(ctx, pgListener)
	} else {
		go ni.tickerLoop(ctx)
	}
}

// Stop shuts down the invalidator.
func (ni *NormalizerInvalidator) Stop() {
	ni.mu.Lock()
	defer ni.mu.Unlock()
	if ni.cancel != nil {
		ni.cancel()
		ni.cancel = nil
	}
	ni.running = false
}

type notifyPayload struct {
	Broker    string `json:"broker"`
	SymbolRaw string `json:"symbol_raw"`
}

func (ni *NormalizerInvalidator) listenLoop(ctx context.Context, listener PGListener) {
	ni.log.Info("normalizer_invalidator: PG LISTEN active")
	defer listener.Close()

	for {
		payload, err := listener.WaitForNotification(ctx)
		if err != nil {
			ni.log.Warn("normalizer_invalidator: LISTEN lost, falling back to ticker", zap.Error(err))
			go ni.tickerLoop(ctx)
			return
		}

		var np notifyPayload
		if err := json.Unmarshal([]byte(payload), &np); err != nil {
			ni.log.Debug("normalizer_invalidator: bad payload", zap.Error(err))
			continue
		}
		if np.Broker != "" && np.SymbolRaw != "" {
			ni.onInvalidate(np.Broker, np.SymbolRaw)
		}
	}
}

func (ni *NormalizerInvalidator) tickerLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	ni.log.Info("normalizer_invalidator: ticker fallback started (30s)")

	for {
		select {
		case <-ctx.Done():
			ni.log.Info("normalizer_invalidator: ticker stopped")
			return
		case <-ticker.C:
			ni.log.Debug("normalizer_invalidator: ticker heartbeat")
		}
	}
}
