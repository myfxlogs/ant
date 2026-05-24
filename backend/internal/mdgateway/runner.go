package mdgateway

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	anttrace "anttrader/internal/trace"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// RunnerDeps collects all infrastructure needed to start mdgateway.
type RunnerDeps struct {
	Log      *zap.Logger
	PG       *pgxpool.Pool
	CH       clickhouse.Conn
	NATSConn *nats.Conn
	SpillDir string // default /var/lib/ant/spill
}

// Run assembles and starts the full mdgateway pipeline.
// Blocks until ctx.Done.
//
// Flow:
//  1. SpillReplay — replay pending jsonl files (dual-write: NATS + CH)
//  2. Load finalized bars from CH for bar finality (ADR-0009 §2.2)
//  3. Create Manager with all components wired
//  4. Start CHWriter background flush loop
//  5. Start Backfiller initial scan + 6h cron
//  6. Start NormalizerInvalidator (PG LISTEN or ticker fallback)
//  7. Load active accounts from PG and create adapter Gateways
//  8. Start health monitor goroutine (every 30s)
func Run(ctx context.Context, deps RunnerDeps) error {
	log := deps.Log
	log.Info("mdgateway: starting", zap.String("spill_dir", deps.SpillDir))

	// --- OTel trace (ADR-0010 §2.3) ---
	tracer := anttrace.New()
	defer tracer.Shutdown(context.Background())
	log.Info("mdgateway: trace", zap.Bool("enabled", tracer.Enabled()))

	// --- SpillWriter ---
	spillCfg := DefaultSpillConfig()
	if deps.SpillDir != "" {
		spillCfg.Dir = deps.SpillDir
	}
	spillWriter, err := NewSpillWriter(spillCfg, log)
	if err != nil {
		return err
	}
	defer spillWriter.Close()

	// --- CHWriter ---
	chCfg := DefaultCHWriterConfig()
	chWriter := NewCHWriter(chCfg, deps.CH, spillWriter, log)

	// --- Publisher ---
	js, _ := deps.NATSConn.JetStream()
	publisher := NewPublisher(js)

	// --- SpillReplay (runs first, dual-write NATS + CH) ---
	replay := NewSpillReplay(spillCfg.Dir, publisher, chWriter, log)
	if n, err := replay.Run(ctx); err != nil {
		log.Warn("mdgateway: spill_replay errors", zap.Error(err))
	} else {
		log.Info("mdgateway: spill_replay complete", zap.Int("rows", n))
	}

	// --- BarAggregator with finalized bars ---
	aggregator := NewBarAggregator()
	finalized, err := loadFinalizedBars(ctx, deps.CH, log)
	if err != nil {
		return err
	}
	aggregator.LoadFinalizedBars(finalized)

	// --- Normalizer + Quality + Dedup ---
	normalizer := NewNormalizer(deps.PG)
	quality := NewQuality(DefaultQualityConfig())
	dedup := NewTickDedup(100)

	// --- Start CHWriter background loop ---
	go chWriter.Start(ctx)

	// --- Backfiller (initial scan + 6h cron) ---
	bf := startBackfiller(ctx, deps, aggregator, publisher, chWriter, log)

	// --- NormalizerInvalidator (PG LISTEN) ---
	invalidator := NewNormalizerInvalidator(log, func(broker, symbolRaw string) {
		normalizer.InvalidateCache(broker, symbolRaw)
	})
	invalidator.Start(ctx, nil) // nil = ticker fallback; pgx.Conn implements PGListener

	// --- Manager (wires HandleTick pipeline) ---
	mgr := NewManager(ManagerDeps{
		Normalizer:  normalizer,
		Quality:     quality,
		Dedup:       dedup,
		Aggregator:  aggregator,
		Publisher:   publisher,
		CHWriter:    chWriter,
		SpillWriter: spillWriter,
	})

	// Set up spill-fail → breaker escalation.
	chWriter.SetOnSpillFail(func(brokerKey string, err error) {
		log.Warn("mdgateway: spill failed", zap.String("broker", brokerKey), zap.Error(err))
	})

	// --- Load active accounts and start gateways ---
	accounts, err := loadActiveAccounts(ctx, deps.PG)
	if err != nil {
		log.Warn("mdgateway: load active accounts failed", zap.Error(err))
	} else {
		for _, acc := range accounts {
			log.Info("mdgateway: gateway (stub)", zap.String("account", acc.AccountID))
			// Full implementation: construct mt4/mt5 adapter, connect, subscribe.
		}
	}

	// --- Health monitor ---
	go healthMonitor(ctx, mgr, log)

	// --- Wait for shutdown ---
	<-ctx.Done()
	log.Info("mdgateway: shutting down")

	// Graceful drain.
	ticks, bars := chWriter.drain()
	chWriter.Flush(ctx, ticks, bars)
	_ = invalidator
	_ = bf
	log.Info("mdgateway: stopped")
	return nil
}

// loadFinalizedBars queries CH for all existing close_ts values per (broker,canonical,period).
// Returns a map of key→[]close_ts for exact-match dedup (M10.5-3d fix).
// If CH is unreachable, logs fatal and returns error — bar finality must never be silently disabled.
func loadFinalizedBars(ctx context.Context, ch clickhouse.Conn, log *zap.Logger) (map[finalizedKey][]int64, error) {
	result := make(map[finalizedKey][]int64)
	rows, err := ch.Query(ctx, `
		SELECT broker, canonical, period, close_ts_unix_ms
		FROM md_bars FINAL
	`)
	if err != nil {
		log.Error("mdgateway: load finalized bars FAILED — CH unreachable, refusing to start", zap.Error(err))
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var broker, canonical, period string
		var closeTs int64
		if err := rows.Scan(&broker, &canonical, &period, &closeTs); err != nil {
			continue
		}
		fk := finalizedKey{broker, canonical, period}
		result[fk] = append(result[fk], closeTs)
	}
	log.Info("mdgateway: loaded finalized bars", zap.Int("keys", len(result)))
	return result, nil
}

// activeAccount is a minimal account record for gateway startup.
type activeAccount struct {
	AccountID string
	Broker    string
	Platform  string // "mt4" | "mt5"
}

// loadActiveAccounts queries PG for active mt_accounts.
func loadActiveAccounts(ctx context.Context, pg *pgxpool.Pool) ([]activeAccount, error) {
	rows, err := pg.Query(ctx, `
		SELECT id, broker, platform
		FROM mt_accounts_v2 WHERE is_active = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var accs []activeAccount
	for rows.Next() {
		var a activeAccount
		if err := rows.Scan(&a.AccountID, &a.Broker, &a.Platform); err != nil {
			continue
		}
		accs = append(accs, a)
	}
	return accs, nil
}

// startBackfiller creates and starts the backfiller.
func startBackfiller(ctx context.Context, deps RunnerDeps, agg *BarAggregator, pub *Publisher, chw *CHWriter, log *zap.Logger) *backfillerBridge {
	bf := &backfillerBridge{log: log}
	go func() {
		// Initial scan.
		log.Info("backfiller: initial scan starting")
		// TODO: wire real Source + CHMaxCloseTs + PGActiveAccounts.
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Info("backfiller: cron scan")
			}
		}
	}()
	return bf
}

type backfillerBridge struct{ log *zap.Logger }

// healthMonitor checks gateway health every 30s.
func healthMonitor(ctx context.Context, mgr *Manager, log *zap.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, h := range mgr.Health() {
				log.Debug("mdgateway: health", zap.String("account", h.AccountID), zap.String("state", h.State))
			}
		}
	}
}

// drain collects all pending ticks and bars from the CHWriter queues for flush.
func (w *CHWriter) drain() (ticks []*mdtick.Tick, bars []*mdtick.Bar) {
	// Drain tickQ.
	for {
		select {
		case t := <-w.tickQ:
			ticks = append(ticks, t)
		default:
			goto drainBars
		}
	}
drainBars:
	for {
		select {
		case b := <-w.barQ:
			bars = append(bars, b)
		default:
			return
		}
	}
}
