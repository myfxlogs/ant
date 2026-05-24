// V1-LEGACY: will be replaced by M7.1-7.4 cards. Do not extend; new code goes alongside.
// Package mdgateway — gateway runner.
// Loads mt_accounts from PG, creates gateways, connects, subscribes,
// and assembles the full publisher+CHWriter+SpillWriter pipeline.
package mdgateway

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"anttrader/internal/config"
	"anttrader/internal/mt4client"
	"anttrader/internal/mt5client"
)

// Runner orchestrates the mdgateway lifecycle.
type Runner struct {
	cfg        *config.Config
	pg         *pgxpool.Pool
	mt4Client  *mt4client.MT4Client
	mt5Client  *mt5client.MT5Client
	manager    *Manager
	publisher  *Publisher
	chWriter   *CHWriter
	chConn     *CHConn
	spill      *SpillWriter
	log        *zap.Logger
	metrics    *MDMetrics

	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewRunner creates a gateway runner from application config.
func NewRunner(cfg *config.Config, pg *pgxpool.Pool, log *zap.Logger) (*Runner, error) {
	chCfg := CHConnConfig{
		Addr:     fmt.Sprintf("%s:%d", cfg.ClickHouse.Host, cfg.ClickHouse.Port),
		Database: cfg.ClickHouse.Database,
		User:     cfg.ClickHouse.User,
		Password: cfg.ClickHouse.Password,
	}

	chConn, err := NewCHConn(chCfg, log)
	if err != nil {
		return nil, fmt.Errorf("runner: clickhouse connect: %w", err)
	}

	spillCfg := DefaultSpillConfig()
	spill, err := NewSpillWriter(spillCfg, log)
	if err != nil {
		return nil, fmt.Errorf("runner: spill writer: %w", err)
	}

	pubCfg := DefaultPublisherConfig()
	pub := NewPublisher(pubCfg, log)

	chWriterCfg := DefaultCHWriterConfig()
	chWriterCfg.SpillWriter = spill
	chWriter := NewCHWriter(chWriterCfg, chConn, log)

	metrics := NewMDMetrics(nil)
	chWriter.SetMetrics(metrics)

	mt4c := mt4client.NewMT4Client(&cfg.MT4)
	mt5c := mt5client.NewMT5Client(&cfg.MT5)

	mgr := NewEmptyManager()
	mgr.SetMetrics(metrics)

	return &Runner{
		cfg:       cfg,
		pg:        pg,
		mt4Client: mt4c,
		mt5Client: mt5c,
		manager:   mgr,
		publisher: pub,
		chWriter:  chWriter,
		chConn:    chConn,
		spill:     spill,
		log:       log,
		metrics:   metrics,
	}, nil
}

// Start connects to NATS, replays spills, loads accounts, connects and subscribes gateways,
// starts the CH writer, and begins the health check loop.
func (r *Runner) Start(ctx context.Context) error {
	r.mu.Lock()
	r.ctx, r.cancel = context.WithCancel(ctx)
	r.mu.Unlock()

	// 1. Connect NATS publisher (non-fatal)
	if err := r.publisher.Connect(r.ctx); err != nil {
		r.log.Warn("runner: NATS publisher connect failed (non-fatal)", zap.Error(err))
	} else {
		r.log.Info("runner: NATS publisher connected")
	}

	// 2. Replay any spill files from previous run
	chConn, err := NewCHConn(CHConnConfig{
		Addr:     fmt.Sprintf("%s:%d", r.cfg.ClickHouse.Host, r.cfg.ClickHouse.Port),
		Database: r.cfg.ClickHouse.Database,
		User:     r.cfg.ClickHouse.User,
		Password: r.cfg.ClickHouse.Password,
	}, r.log)
	if err == nil {
		sr := NewSpillReplay(DefaultSpillConfig().Dir, chConn, r.log)
		if replayed, err := sr.Replay(r.ctx); err != nil {
			r.log.Warn("runner: spill replay failed", zap.Error(err))
		} else if replayed > 0 {
			r.log.Info("runner: spill replay complete", zap.Int("rows", replayed))
		}
	}

	// 3. Start CH writer
	r.chWriter.Start(r.ctx)

	// 4. Load mt_accounts from PG
	accounts, err := r.loadAccounts(r.ctx)
	if err != nil {
		return fmt.Errorf("runner: load accounts: %w", err)
	}

	r.log.Info("runner: accounts loaded from PG", zap.Int("count", len(accounts)))

	// 5. Create gateways with real MT client, connect + subscribe
	for _, ac := range accounts {
		mgrKey := ac.Broker + "-" + ac.Login
		r.manager.AddGatewayWithClient(ac, r.mt4Client, r.mt5Client)

		if err := r.manager.ConnectGateway(r.ctx, mgrKey); err != nil {
			r.log.Warn("runner: gateway connect failed",
				zap.String("key", mgrKey),
				zap.String("platform", ac.Platform),
				zap.Error(err),
			)
			continue
		}

		r.log.Info("runner: gateway connected",
			zap.String("platform", ac.Platform),
			zap.String("broker", ac.Broker),
		)

		// Get the gateway and subscribe with tick handler
		conns := r.manager.Connections()
		if gw, ok := conns[mgrKey]; ok {
			r.startGatewaySubscription(r.ctx, gw, mgrKey)
		}
	}

	// 6. Start health check loop
	go r.healthLoop(r.ctx)

	return nil
}

// startGatewaySubscription subscribes to quote ticks and feeds them through
// the quality → bar → CHWriter pipeline.
func (r *Runner) startGatewaySubscription(ctx context.Context, gw Gateway, key string) {
	// Default symbols — in alfq these came from broker_symbols.
	// TODO: load per-account symbols from PG broker_symbols table.
	defaultSymbols := []string{"EURUSD", "GBPUSD", "USDJPY", "XAUUSD", "BTCUSD"}

	agg := NewAggregator()

	if err := gw.Subscribe(ctx, defaultSymbols, func(tick *Tick) {
		// Quality check
		// TODO: wire Quality instance
		// Write to CH
		r.chWriter.Write(tick)
		// Publish to NATS
		_ = r.publisher.Publish(ctx, tick)
	}); err != nil {
		r.log.Warn("runner: subscribe failed",
			zap.String("key", key),
			zap.String("platform", gw.Platform()),
			zap.Error(err),
		)
		return
	}

	// Periodic bar flush
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				agg.FlushAll(func(b Bar) {
					r.chWriter.Write(&Tick{UserID: b.UserID, Broker: b.Broker, Symbol: b.SymbolRaw, Canonical: b.Canonical})
					_ = r.publisher.PublishBar(b)
					if r.metrics != nil {
						r.metrics.BarFlushed.WithLabelValues(b.Period).Inc()
					}
				})
				return
			case <-ticker.C:
				agg.FlushAll(func(b Bar) {
					_ = r.publisher.PublishBar(b)
					if r.metrics != nil {
						r.metrics.BarFlushed.WithLabelValues(b.Period).Inc()
					}
				})
			}
		}
	}()

	r.log.Info("runner: subscribed",
		zap.String("key", key),
		zap.String("platform", gw.Platform()),
		zap.Strings("symbols", defaultSymbols),
	)
}

// healthLoop periodically checks and reconnects gateways.
func (r *Runner) healthLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for key := range r.manager.Connections() {
				if err := r.manager.HealthCheckGateway(ctx, key); err != nil {
					r.log.Warn("runner: health check failed, reconnecting",
						zap.String("key", key),
						zap.Error(err),
					)
					// Attempt reconnect
					if err := r.manager.ConnectGateway(ctx, key); err != nil {
						r.log.Warn("runner: reconnect failed",
							zap.String("key", key),
							zap.Error(err),
						)
					}
				}
			}
		}
	}
}

// Stop shuts down the gateway runner gracefully.
func (r *Runner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		r.cancel()
	}

	var errs []error
	if err := r.chWriter.Close(); err != nil {
		errs = append(errs, fmt.Errorf("chwriter close: %w", err))
	}
	if err := r.spill.Close(); err != nil {
		errs = append(errs, fmt.Errorf("spill close: %w", err))
	}
	if err := r.publisher.Close(); err != nil {
		errs = append(errs, fmt.Errorf("publisher close: %w", err))
	}

	// Disconnect all gateways
	for key, gw := range r.manager.Connections() {
		if err := gw.Disconnect(context.Background()); err != nil {
			r.log.Warn("runner: gateway disconnect failed", zap.String("key", key), zap.Error(err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("runner: stop errors: %v", errs)
	}
	return nil
}

// Manager returns the underlying gateway manager.
func (r *Runner) Manager() *Manager { return r.manager }

// Publisher returns the NATS publisher.
func (r *Runner) Publisher() *Publisher { return r.publisher }

// CHWriter returns the ClickHouse writer.
func (r *Runner) CHWriter() *CHWriter { return r.chWriter }

// Metrics returns the Prometheus metrics registry.
func (r *Runner) Metrics() *MDMetrics { return r.metrics }

// mtAccountRow is a flat PG row for mt_accounts.
type mtAccountRow struct {
	UserID        string
	BrokerCompany string
	MTType        string
	Login         string
	Password      string
	BrokerServer  string
	BrokerHost    string
}

func (r *Runner) loadAccounts(ctx context.Context) ([]AccountConfig, error) {
	if r.pg == nil {
		return nil, fmt.Errorf("runner: no PG pool configured")
	}

	rows, err := r.pg.Query(ctx, `SELECT user_id::text, broker_company, mt_type, login, password,
		COALESCE(broker_server,''), COALESCE(broker_host,'')
		FROM mt_accounts WHERE is_disabled = false`)
	if err != nil {
		return nil, fmt.Errorf("runner: query mt_accounts: %w", err)
	}
	defer rows.Close()

	var accounts []AccountConfig
	for rows.Next() {
		var row mtAccountRow
		if err := rows.Scan(&row.UserID, &row.BrokerCompany, &row.MTType,
			&row.Login, &row.Password, &row.BrokerServer, &row.BrokerHost); err != nil {
			r.log.Warn("runner: scan mt_accounts row", zap.Error(err))
			continue
		}
		ac := AccountConfig{
			UserID:   row.UserID,
			Broker:   row.BrokerCompany,
			Platform: row.MTType,
			Login:    row.Login,
			Password: row.Password,
			Server:   row.BrokerServer,
			Host:     row.BrokerHost,
			Port:     "443",
		}
		// Normalize platform (case-insensitive)
		ac.Platform = strings.ToLower(ac.Platform)
		if ac.Platform != "mt4" && ac.Platform != "mt5" {
			r.log.Warn("runner: unknown platform", zap.String("mt_type", ac.Platform))
			continue
		}
		accounts = append(accounts, ac)
	}
	return accounts, rows.Err()
}

// BarRow is a flat CH row for md_bars queries.
type BarRow struct {
	Broker       string
	SymbolRaw    string
	Canonical    string
	Period       string
	OpenTsUnixMs uint64
	CloseTsUnixMs uint64
	Open         float64
	High         float64
	Low          float64
	Close        float64
	Volume       float64
	TickCount    uint32
}

// QueryBars reads OHLCV bars from ClickHouse md_bars.
// Used by connect services as the primary K-line source (M7.8-7).
func (r *Runner) QueryBars(ctx context.Context, broker, symbol, period string, fromTs, toTs int64, limit int) ([]BarRow, error) {
	if r.chConn == nil {
		return nil, fmt.Errorf("runner: CH not available")
	}

	conn, err := r.chConn.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("runner: CH conn: %w", err)
	}

	query := `SELECT broker, symbol_raw, canonical, period, open_ts_unix_ms, close_ts_unix_ms,
		open, high, low, close, volume, tick_count
		FROM md_bars
		WHERE canonical = ? AND period = ?
		ORDER BY close_ts_unix_ms DESC
		LIMIT ?`

	rows, err := conn.Query(ctx, query, symbol, period, limit)
	if err != nil {
		return nil, fmt.Errorf("runner: CH query: %w", err)
	}
	defer rows.Close()

	var bars []BarRow
	for rows.Next() {
		var b BarRow
		if err := rows.Scan(&b.Broker, &b.SymbolRaw, &b.Canonical, &b.Period,
			&b.OpenTsUnixMs, &b.CloseTsUnixMs, &b.Open, &b.High, &b.Low, &b.Close,
			&b.Volume, &b.TickCount); err != nil {
			r.log.Warn("runner: scan CH bar", zap.Error(err))
			continue
		}
		bars = append(bars, b)
	}
	return bars, rows.Err()
}
