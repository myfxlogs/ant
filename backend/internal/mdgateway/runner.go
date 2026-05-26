package mdgateway

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	anttrace "anttrader/internal/trace"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/mdgateway/adapter/mt4"
	"anttrader/internal/mdgateway/adapter/mt5"
	"anttrader/internal/mdgateway/backfiller"
	"anttrader/internal/mthub"
	"anttrader/internal/secrets"
)

// RunnerDeps collects all infrastructure needed to start mdgateway.
type RunnerDeps struct {
	Log            *zap.Logger
	PG             *pgxpool.Pool
	CH             clickhouse.Conn
	NATSConn       *nats.Conn
	SpillDir       string          // default /var/lib/ant/spill
	Secrets        secrets.Client  // decrypts account passwords and mtapi tokens
	OnAccountProfit  func(accountID, userID string, p *mdtick.ProfitUpdate)    // receives real-time balance/equity from mtapi OnOrderProfit
	OnOrderUpdate    func(accountID, userID string, o *mdtick.OrderUpdate)     // receives real-time order/position changes from mtapi OnOrderUpdate
	Hub             *mthub.Hub
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
		return fmt.Errorf("create spill writer: %w", err)
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
		return fmt.Errorf("load finalized bars from ClickHouse: %w", err)
	}
	aggregator.LoadFinalizedBars(finalized)

	// --- Normalizer + Quality + Dedup ---
	normalizer := NewNormalizer(deps.PG)
	quality := NewQuality(DefaultQualityConfig())
	dedup := NewTickDedup(0) // default size (1000)

	// --- Start CHWriter background loop ---
	go chWriter.Start(ctx)

	// --- Backfiller (initial scan + 6h cron) ---
	bf, srcMap := startBackfiller(ctx, deps, aggregator, publisher, chWriter, log)

	// --- NormalizerInvalidator (PG LISTEN) ---
	invalidator := NewNormalizerInvalidator(log, deps.PG, func(broker, symbolRaw string) {
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
		Log:         log,
	})
	// L-2: wire real OTel tracer into pipeline spans.
	mgr.SetOTelTracer(tracer)
	mgr.SetBaseContext(ctx)

	// Set up spill-fail → breaker escalation.
	chWriter.SetOnSpillFail(func(brokerKey string, err error) {
		log.Warn("mdgateway: spill failed", zap.String("broker", brokerKey), zap.Error(err))
	})

	// --- Load active accounts and start gateways ---
	cfgs, err := loadAccountConfigs(ctx, deps)
	if err != nil {
		log.Warn("mdgateway: load account configs failed", zap.Error(err))
	} else {
		for _, cfg := range cfgs {
			accID := cfg.AccountID
			log.Info("mdgateway: starting gateway",
				zap.String("account", accID),
				zap.String("platform", cfg.Platform),
				zap.String("broker", cfg.Broker))

			// Build gateway for this account.
			var gw Gateway
			switch strings.ToLower(cfg.Platform) {
			case "mt4":
				gw = mt4.New(cfg, log)
			case "mt5":
				gw = mt5.New(cfg, log)
			default:
				log.Warn("mdgateway: unknown platform", zap.String("platform", cfg.Platform), zap.String("account", accID))
				continue
			}

			// Connect to mtapi.
			if err := gw.Connect(ctx); err != nil {
				log.Error("mdgateway: gateway connect failed",
					zap.String("account", accID), zap.Error(err))
				continue
			}

			// Register gateway for health tracking + backfiller source routing.
			_ = mgr.AddGateway(ctx, gw, nil)

			// Register as OrderExecutor with mthub so trading operations
			// (FetchOpenedOrders, etc.) can use this broker connection.
			if deps.Hub != nil {
				if exec, ok := gw.(mthub.OrderExecutor); ok {
					deps.Hub.Register(accID,
						&mthub.Session{AccountID: accID, CreatedAt: Clk.Now()},
						exec,
					)
				}
			}
			if bfSrc, ok := gw.(backfiller.MTAPIBarSource); ok {
				srcMap.gws[accID] = bfSrc
			}

			// Default symbols for real-time quotes when account has none configured.
			syms := cfg.Symbols
			if len(syms) == 0 {
				syms = defaultQuoteSymbols()
			}

			// Subscribe to tick stream; HandleTick is the pipeline entry point.
			if err := gw.Subscribe(ctx, syms, mgr.HandleTick); err != nil {
				log.Error("mdgateway: gateway subscribe failed",
					zap.String("account", accID), zap.Error(err))
				continue
			}

			// Subscribe to profit stream (real-time balance/equity/P&L from mtapi OnOrderProfit).
			if deps.OnAccountProfit != nil {
				uid := cfg.UserID
				aid := accID
				if err := gw.SubscribeProfit(ctx, func(p *mdtick.ProfitUpdate) {
					deps.OnAccountProfit(aid, uid, p)
				}); err != nil {
					log.Warn("mdgateway: profit subscribe failed",
						zap.String("account", accID), zap.Error(err))
				}
			}

			// Subscribe to order update stream (real-time position changes from mtapi OnOrderUpdate).
			if deps.OnOrderUpdate != nil {
				uid := cfg.UserID
				aid := accID
				if err := gw.SubscribeOrderUpdate(ctx, func(o *mdtick.OrderUpdate) {
					deps.OnOrderUpdate(aid, uid, o)
				}); err != nil {
					log.Warn("mdgateway: order update subscribe failed",
						zap.String("account", accID), zap.Error(err))
				}
			}

			log.Info("mdgateway: gateway active",
				zap.String("account", accID),
				zap.String("platform", cfg.Platform))
		}
	}

	// --- Health monitor ---
	go healthMonitor(ctx, mgr, chWriter, log)

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

// loadAccountConfigs queries PG for active accounts.
// Passwords are stored as plaintext per user requirement.
func loadAccountConfigs(ctx context.Context, deps RunnerDeps) ([]mdtick.AccountConfig, error) {
	rows, err := deps.PG.Query(ctx, `
		SELECT id, user_id, platform, broker, mtapi_host, mtapi_port,
		       login, password, mt_token, broker_host, server,
		       canonical_subscribed_symbols
		FROM mt_accounts_v2 WHERE is_active = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cfgs []mdtick.AccountConfig
	for rows.Next() {
		var (
			id, userID, platform, broker, mtapiHost, mtapiPort, login, brokerHost, server string
			password, mtToken string
			symbols                     []string
		)
		if err := rows.Scan(&id, &userID, &platform, &broker, &mtapiHost, &mtapiPort,
			&login, &password, &mtToken, &brokerHost, &server, &symbols); err != nil {
			continue
		}

		// password and mt_token are plaintext from DB.
		mtapiToken := mtToken

		cfgs = append(cfgs, mdtick.AccountConfig{
			AccountID:  id,
			UserID:     userID,
			Broker:     broker,
			Platform:   platform,
			Login:      login,
			Password:   password,
			Server:     server,
				BrokerHost: brokerHost,
			MtapiHost:  mtapiHost,
			MtapiPort:  mtapiPort,
			MtapiToken: mtapiToken,
				Symbols:    symbols,
		})
	}
	return cfgs, nil
}

// startBackfiller creates and starts the backfiller with real Source + CH + PG wiring.
func startBackfiller(ctx context.Context, deps RunnerDeps, agg *BarAggregator, pub *Publisher, chw *CHWriter, log *zap.Logger) (*backfiller.Backfiller, *gatewaySourceMap) {
	// Source: routes GetPriceHistory to the correct gateway by accountID.
	srcMap := &gatewaySourceMap{gws: make(map[string]backfiller.MTAPIBarSource)}
	src := backfiller.NewSourceMTAPI(srcMap)

	// Target: aggregator finality check → NATS publish → CH enqueue.
	tgt := backfiller.NewTarget(agg, pub, chw)

	// CHMaxCloseTs: queries ClickHouse for max close_ts per (broker, canonical, period).
	chMax := &chMaxCloseTs{conn: deps.CH}

	// PGActiveAccounts: queries pg for active accounts + subscribed symbols.
	pgAcc := &pgActiveAccounts{pool: deps.PG}

	bf := backfiller.New(src, tgt, chMax, pgAcc)

	// Initial scan (async — don't block mdgateway startup).
	go func() {
		log.Info("backfiller: initial scan starting")
		if err := bf.Run(ctx); err != nil {
			log.Warn("backfiller: initial scan error", zap.Error(err))
		} else {
			log.Info("backfiller: initial scan complete")
		}

		// 6h cron for periodic gap checks.
		ticker := Clk.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C():
				log.Info("backfiller: cron scan starting")
				if err := bf.Run(ctx); err != nil {
					log.Warn("backfiller: cron scan error", zap.Error(err))
				}
			}
		}
	}()

	// PG NOTIFY trigger for instant backfill on new subscription.
	go func() {
		conn, err := deps.PG.Acquire(ctx)
		if err != nil {
			log.Warn("backfiller: cannot acquire PG conn for NOTIFY", zap.Error(err))
			return
		}
		notifier := &pgxNotifier{conn: conn}
		trig := backfiller.NewPGTrigger(log, bf.BackfillAccount)
		if err := trig.Run(ctx, notifier); err != nil {
			log.Warn("backfiller: PG NOTIFY listener stopped", zap.Error(err))
		}
	}()

	return bf, srcMap
}

// gatewaySourceMap routes GetPriceHistory calls to the correct gateway by accountID.
type gatewaySourceMap struct {
	gws map[string]backfiller.MTAPIBarSource
}

func (m *gatewaySourceMap) GetPriceHistory(ctx context.Context, accountID, symbolRaw, period string, from, to int64) ([]*mdtick.Bar, error) {
	gw, ok := m.gws[accountID]
	if !ok {
		return nil, nil // no gateway for this account
	}
	return gw.GetPriceHistory(ctx, accountID, symbolRaw, period, from, to)
}

// chMaxCloseTs implements backfiller.CHMaxCloseTs via ClickHouse.
type chMaxCloseTs struct {
	conn clickhouse.Conn
}

func (c *chMaxCloseTs) MaxCloseTs(ctx context.Context, broker, canonical, period string) (int64, error) {
	var ts int64
	err := c.conn.QueryRow(ctx, `
		SELECT max(close_ts_unix_ms) FROM md_bars FINAL
		WHERE broker = ? AND canonical = ? AND period = ?
	`, broker, canonical, period).Scan(&ts)
	if err != nil {
		return 0, nil
	}
	return ts, nil
}

// pgActiveAccounts implements backfiller.PGActiveAccounts via pgxpool.
type pgActiveAccounts struct {
	pool *pgxpool.Pool
}

func (p *pgActiveAccounts) ActiveAccounts(ctx context.Context) ([]backfiller.ActiveAccount, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT a.id, a.broker, a.canonical_subscribed_symbols
		FROM mt_accounts_v2 a
		WHERE a.is_active = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var accs []backfiller.ActiveAccount
	for rows.Next() {
		var a backfiller.ActiveAccount
		var syms []string
		if err := rows.Scan(&a.AccountID, &a.Broker, &syms); err != nil {
			continue
		}
		a.Symbols = syms
		accs = append(accs, a)
	}
	return accs, nil
}

// pgxNotifier adapts an acquired pgxpool.Conn into backfiller.PGNotifier.
type pgxNotifier struct {
	conn *pgxpool.Conn
}

func (n *pgxNotifier) WaitForNotification(ctx context.Context) (string, string, error) {
	notif, err := n.conn.Conn().WaitForNotification(ctx)
	if err != nil {
		return "", "", err
	}
	return notif.Channel, notif.Payload, nil
}

func (n *pgxNotifier) Close() error {
	n.conn.Release()
	return nil
}

// healthMonitor checks gateway health every 30s, monitors memory pressure
// for S-2 auto-degradation, and emits stale/dead account metrics.
func healthMonitor(ctx context.Context, mgr *Manager, chw *CHWriter, log *zap.Logger) {
	ticker := Clk.NewTicker(30 * time.Second)
	defer ticker.Stop()

	const (
		highThreshold = 0.80
		lowThreshold  = 0.60
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			var stale, dead int64
			for _, h := range mgr.Health() {
				switch h.State {
				case "stale":
					stale++
					log.Warn("mdgateway: stale account — no ticks for >5 min",
						zap.String("account", h.AccountID),
						zap.String("platform", h.Platform))
				case "dead":
					dead++
					log.Error("mdgateway: dead account — no ticks for >15 min",
						zap.String("account", h.AccountID),
						zap.String("platform", h.Platform))
				case "no_data":
					log.Debug("mdgateway: no data yet",
						zap.String("account", h.AccountID))
				default:
					log.Debug("mdgateway: health",
						zap.String("account", h.AccountID),
						zap.String("state", h.State))
				}
			}
			SetStaleAccountCount(stale, dead)

			// S-2: memory pressure → auto buffer bypass.
			memRatio := currentMemoryRatio()
			bufEnabled := chw.BufferEnabled()

			if memRatio > highThreshold && bufEnabled {
				log.Warn("mdgateway: memory pressure — disabling CH Buffer engine",
					zap.Float64("mem_ratio", memRatio),
					zap.Float64("threshold", highThreshold))
				chw.SetBufferEnabled(false)
			} else if memRatio < lowThreshold && !bufEnabled {
				log.Info("mdgateway: memory pressure resolved — re-enabling CH Buffer engine",
					zap.Float64("mem_ratio", memRatio),
					zap.Float64("threshold", lowThreshold))
				chw.SetBufferEnabled(true)
			}
		}
	}
}

// currentMemoryRatio returns the current process RSS as a fraction of the memory limit.
func currentMemoryRatio() float64 {
	limitBytes := cgroupMemoryLimit()
	if limitBytes <= 0 {
		return 0
	}
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					return float64(kb*1024) / float64(limitBytes)
				}
			}
		}
	}
	return 0
}

// cgroupMemoryLimit returns the cgroup memory limit in bytes (v1 or v2).
func cgroupMemoryLimit() int64 {
	// cgroup v2
	if data, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		s := strings.TrimSpace(string(data))
		if s != "max" {
			if v, err := strconv.ParseInt(s, 10, 64); err == nil {
				return v
			}
		}
	}
	// cgroup v1
	if data, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes"); err == nil {
		if v, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
			if v < (1 << 62) {
				return v
			}
		}
	}
	// Fallback: system total memory.
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return int64(ms.Sys)
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

// defaultQuoteSymbols returns a default set of major forex + crypto pairs
// for mtapi SymbolSubscribe when an account has no configured symbols.
func defaultQuoteSymbols() []string {
	return []string{
		"BTCUSDm", "ETHUSDm", "XRPUSDm", "SOLUSDm", "BNBUSDm",
		"EURUSDm", "GBPUSDm", "USDJPYm", "XAUUSDm", "US30m",
	}
}
