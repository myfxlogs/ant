// Package mdgateway — gateway runner.
// Loads mt_accounts from PG, creates gateways, connects, subscribes,
// and assembles the full publisher+CHWriter+SpillWriter pipeline.
package mdgateway

import (
	"context"
	"fmt"
	"strings"
	"sync"

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
		spill:     spill,
		log:       log,
		metrics:   metrics,
	}, nil
}

// Start connects to NATS, loads accounts from PG, connects gateways, and starts the CH writer.
func (r *Runner) Start(ctx context.Context) error {
	r.mu.Lock()
	r.ctx, r.cancel = context.WithCancel(ctx)
	r.mu.Unlock()

	// 1. Connect NATS publisher
	if err := r.publisher.Connect(r.ctx); err != nil {
		r.log.Warn("runner: NATS publisher connect failed (non-fatal)", zap.Error(err))
	} else {
		r.log.Info("runner: NATS publisher connected")
	}

	// 2. Start CH writer
	r.chWriter.Start(r.ctx)

	// 3. Load mt_accounts from PG
	accounts, err := r.loadAccounts(r.ctx)
	if err != nil {
		return fmt.Errorf("runner: load accounts: %w", err)
	}

	r.log.Info("runner: accounts loaded from PG", zap.Int("count", len(accounts)))

	// 4. Create gateways with real MT client, connect
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
	}

	return nil
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
