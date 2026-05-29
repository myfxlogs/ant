package mdgateway

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PGListener abstracts a pgx connection for LISTEN/NOTIFY.
type PGListener interface {
	WaitForNotification(ctx context.Context) (payload string, err error)
	Close() error
}

// pgListenConn wraps a pgxpool connection for PG NOTIFY listening.
type pgListenConn struct {
	conn *pgxpool.Conn
}

func (p *pgListenConn) WaitForNotification(ctx context.Context) (string, error) {
	n, err := p.conn.Conn().WaitForNotification(ctx)
	if err != nil {
		return "", err
	}
	return n.Payload, nil
}

func (p *pgListenConn) Close() error { p.conn.Release(); return nil }

// newPGListener acquires a PG connection and starts LISTEN on broker_symbols_changed.
// Returns nil if PG is unavailable, causing ticker fallback.
func newPGListener(ctx context.Context, pool *pgxpool.Pool, log *zap.Logger) PGListener {
	if pool == nil {
		return nil
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		log.Warn("normalizer_invalidator: cannot acquire PG conn, using ticker", zap.Error(err))
		return nil
	}
	if _, err := conn.Exec(ctx, "LISTEN broker_symbols_changed"); err != nil {
		log.Warn("normalizer_invalidator: LISTEN failed, using ticker", zap.Error(err))
		conn.Release()
		return nil
	}
	return &pgListenConn{conn: conn}
}

// NormalizerInvalidator listens for PG NOTIFY broker_symbols_changed events
// and invalidates the normalizer cache for affected (broker, symbol_raw) pairs.
// Falls back to 30s ticker polling when the LISTEN connection is lost (ADR-0011 §2.3).
// When ticker fallback is active, actively queries broker_symbols to detect changes.
type NormalizerInvalidator struct {
	cancel       context.CancelFunc
	log          *zap.Logger
	onInvalidate func(broker, symbolRaw string)
	pg           *pgxpool.Pool // nil if PG not available (ticker will only heartbeat)
	mu           sync.Mutex
	running      bool
}

// NewNormalizerInvalidator creates an invalidator. onInvalidate is called
// when a broker_symbols change is detected, e.g. normalizer.cache.Remove(key).
func NewNormalizerInvalidator(log *zap.Logger, pg *pgxpool.Pool, onInvalidate func(broker, symbolRaw string)) *NormalizerInvalidator {
	return &NormalizerInvalidator{
		log:          log,
		pg:           pg,
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
	ticker := Clk.NewTicker(30 * time.Second)
	defer ticker.Stop()
	ni.log.Info("normalizer_invalidator: ticker fallback started (30s)")

	// Track known symbol set so we only invalidate on actual changes.
	known := make(map[string]bool) // "broker:symbol_raw"

	for {
		select {
		case <-ctx.Done():
			ni.log.Info("normalizer_invalidator: ticker stopped")
			return
		case <-ticker.C():
			if ni.pg == nil {
				ni.log.Debug("normalizer_invalidator: ticker heartbeat (no PG)")
				continue
			}
			ni.refreshFromPG(ctx, known)
		}
	}
}

func (ni *NormalizerInvalidator) refreshFromPG(ctx context.Context, known map[string]bool) {
	rows, err := ni.pg.Query(ctx,
		"SELECT broker_id::text, symbol_raw FROM broker_symbols")
	if err != nil {
		ni.log.Warn("normalizer_invalidator: PG query failed", zap.Error(err))
		return
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var count int
	for rows.Next() {
		var brokerID, symbolRaw string
		if err := rows.Scan(&brokerID, &symbolRaw); err != nil {
			continue
		}
		key := brokerID + ":" + symbolRaw
		seen[key] = true
		if !known[key] {
			ni.onInvalidate(brokerID, symbolRaw)
			count++
		}
	}

	// Replace known set with current state.
	clear(known)
	for k := range seen {
		known[k] = true
	}

	if count > 0 {
		ni.log.Info("normalizer_invalidator: ticker invalidated cache entries",
			zap.Int("count", count))
	}
}
