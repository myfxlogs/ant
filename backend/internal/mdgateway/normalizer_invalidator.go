package mdgateway

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// NormalizerInvalidator listens for PG NOTIFY broker_symbols_changed events
// and invalidates the normalizer cache for affected (broker, symbol_raw) pairs.
// Falls back to 30s ticker polling when the LISTEN connection is lost (ADR-0011 §2.3).
type NormalizerInvalidator struct {
	cancel    context.CancelFunc
	log       *zap.Logger
	onInvalidate func(broker, symbolRaw string)
	mu        sync.Mutex
	running   bool
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
// pgListener is a PG connection with LISTEN capability (pgx.Conn).
func (ni *NormalizerInvalidator) Start(ctx context.Context, pgListener interface{}) {
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

func (ni *NormalizerInvalidator) listenLoop(ctx context.Context, pgListener interface{}) {
	ni.log.Info("normalizer_invalidator: PG LISTEN started")
	// The real implementation uses pgx.Conn.WaitForNotification and
	// parses the JSON payload (broker, symbol_raw).
	// Stub: when runner.go wires the pgx connection, pass it as pgListener.
	<-ctx.Done()
	ni.log.Info("normalizer_invalidator: PG LISTEN stopped")
}

func (ni *NormalizerInvalidator) tickerLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	ni.log.Info("normalizer_invalidator: ticker fallback (30s)")
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// PG: SELECT MAX(updated_at) FROM broker_symbols
			// If changed since last check → invalidate affected rows.
			// Placeholder: no-op until PG is wired.
		}
	}
}
