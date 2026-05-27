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
	OnAccountProfit     func(accountID, userID string, p *mdtick.ProfitUpdate)    // receives real-time balance/equity from mtapi OnOrderProfit
	OnOrderUpdate       func(accountID, userID string, o *mdtick.OrderUpdate)     // receives real-time order/position changes from mtapi OnOrderUpdate
	OnAccountDisconnect func(accountID string)                                     // B-1.3: called when gateway stops/fails for an account
	OnBrokerInfo        func(accountID, platform, broker string, info *mdtick.BrokerInfo) // B-2.2: called once after successful Connect
	Hub                 *mthub.Hub
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

			// B-2.2: fetch broker-level margin thresholds after connect.
			if deps.OnBrokerInfo != nil {
				if fetcher, ok := gw.(mdtick.BrokerInfoFetcher); ok {
					info, err := fetcher.FetchBrokerInfo(ctx)
					if err != nil {
						log.Warn("mdgateway: FetchBrokerInfo failed",
							zap.String("account", accID), zap.Error(err))
					}
					if info != nil {
						deps.OnBrokerInfo(accID, cfg.Platform, cfg.Broker, info)
					}
				}
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
	go healthMonitor(ctx, mgr, chWriter, log, deps.OnAccountDisconnect)

	// --- Account event subscriber (NATS: account.connect/disconnect/reconnect) ---
	startAccountEventSubscriber(ctx, deps.NATSConn, log)

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

// healthMonitor checks gateway health every 30s, monitors memory pressure
// for S-2 auto-degradation, and emits stale/dead account metrics.
func healthMonitor(ctx context.Context, mgr *Manager, chw *CHWriter, log *zap.Logger, onDisconnect func(accountID string)) {
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
					if onDisconnect != nil {
							onDisconnect(h.AccountID)
						}
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

// startAccountEventSubscriber listens for NATS account lifecycle events
// and logs them. Full dynamic gateway reload is phase 2 (S2.4-m2).
func startAccountEventSubscriber(ctx context.Context, nc *nats.Conn, log *zap.Logger) {
	if nc == nil {
		return
	}
	sub, err := nc.Subscribe("account.>", func(m *nats.Msg) {
		log.Info("mdgateway: account event received",
			zap.String("subject", m.Subject),
			zap.String("data", string(m.Data)))
	})
	if err != nil {
		log.Warn("mdgateway: account event subscribe failed", zap.Error(err))
		return
	}
	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
	}()
	log.Info("mdgateway: account event subscriber started",
		zap.String("subject", "account.>"))
}
