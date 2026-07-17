package mdgateway

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mdgateway/backfiller"
	"alphaforge/internal/repository"
	"alphaforge/internal/secrets"
)

// loadFinalizedBars queries the MarketDataStore for existing close_ts values per key.
// Returns a map of key→[]close_ts for exact-match dedup (M10.5-3d fix).
// If the store is unreachable, logs fatal and returns error — bar finality must never be silently disabled.
func loadFinalizedBars(ctx context.Context, store repository.MarketDataStore, log *zap.Logger) (map[repository.FinalizedKey][]int64, error) {
	if store == nil {
		return nil, fmt.Errorf("mdgateway: market data store is nil")
	}
	result, err := store.LoadFinalizedBars(ctx, time.Now().Add(-30*24*time.Hour))
	if err != nil {
		log.Error("mdgateway: load finalized bars FAILED — store unreachable, refusing to start", zap.Error(err))
		return nil, err
	}
	log.Info("mdgateway: loaded finalized bars", zap.Int("keys", len(result)))
	return result, nil
}

// loadAccountConfigs queries PG for active accounts.
// Passwords are encrypted in DB; decrypted here via secrets client.
func loadAccountConfigs(ctx context.Context, deps RunnerDeps) ([]mdtick.AccountConfig, error) {
	rows, err := deps.PG.Query(ctx, `
		SELECT id, user_id, platform, broker, mtapi_host, mtapi_port,
		       login, password_encrypted, COALESCE(mtapi_token_encrypted, '\x'::bytea), broker_host, server,
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
			passwordEnc, mtTokenEnc []byte
			symbols                     []string
		)
		if err := rows.Scan(&id, &userID, &platform, &broker, &mtapiHost, &mtapiPort,
			&login, &passwordEnc, &mtTokenEnc, &brokerHost, &server, &symbols); err != nil {
			deps.Log.Error("mdgateway: scan account config failed, skipping row", zap.Error(err))
			continue
		}

		var password, mtapiToken string
		if len(passwordEnc) > 0 && deps.Secrets != nil {
			plain, err := deps.Secrets.Decrypt(ctx, secrets.PurposeMTPassword, passwordEnc)
			if err != nil {
				deps.Log.Error("mdgateway: decrypt password failed, skipping account", zap.String("account", id), zap.Error(err))
				continue
			}
			password = string(plain)
		}
		if len(mtTokenEnc) > 0 && deps.Secrets != nil {
			plain, err := deps.Secrets.Decrypt(ctx, secrets.PurposeMTAPIToken, mtTokenEnc)
			if err != nil {
				deps.Log.Warn("mdgateway: decrypt mtapi token failed", zap.String("account", id), zap.Error(err))
			} else {
				mtapiToken = string(plain)
			}
		}

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

// startBackfiller creates and starts the backfiller with real Source + Store + PG wiring.
func startBackfiller(ctx context.Context, deps RunnerDeps, agg *BarAggregator, pub *Publisher, pgw *PgWriter, log *zap.Logger) (*backfiller.Backfiller, *gatewaySourceMap) {
	// Source: routes GetPriceHistory to the correct gateway by accountID.
	srcMap := &gatewaySourceMap{gws: make(map[string]backfiller.MTAPIBarSource)}
	src := backfiller.NewSourceMTAPI(srcMap)

	// Target: aggregator finality check → NATS publish → PG enqueue.
	tgt := backfiller.NewTarget(agg, pub, pgw, nil)

	// CHMaxCloseTs: uses MarketDataStore (PG-primary, CH during transition).
	chMax := &storeMaxCloseTs{store: deps.Store}

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
		for {
			if ctx.Err() != nil {
				return
			}
			conn, err := deps.PG.Acquire(ctx)
			if err != nil {
				log.Warn("backfiller: cannot acquire PG conn for NOTIFY, retrying", zap.Error(err))
				select {
				case <-ctx.Done():
					return
				case <-time.After(10 * time.Second):
				}
				continue
			}
			notifier := &pgxNotifier{conn: conn}
			trig := backfiller.NewPGTrigger(log, bf.BackfillAccount)
			if err := trig.Run(ctx, notifier); err != nil {
				log.Warn("backfiller: PG NOTIFY listener stopped, reconnecting", zap.Error(err))
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
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

// storeMaxCloseTs implements backfiller.CHMaxCloseTs via MarketDataStore.
type storeMaxCloseTs struct {
	store repository.MarketDataStore
}

func (c *storeMaxCloseTs) MaxCloseTs(ctx context.Context, broker, canonical, period string) (int64, error) {
	return c.store.MaxCloseTs(ctx, broker, canonical, period)
}

// loadSingleAccountConfig queries PG for a single account by ID.
// Returns nil if not found or inactive.
func loadSingleAccountConfig(ctx context.Context, pg *pgxpool.Pool, sec secrets.Client, accountID string) (*mdtick.AccountConfig, error) {
	var (
		id, userID, platform, broker, mtapiHost, mtapiPort, login, brokerHost, server string
		passwordEnc, mtTokenEnc []byte
		symbols           []string
	)
	err := pg.QueryRow(ctx, `
		SELECT id, user_id, platform, broker, mtapi_host, mtapi_port,
		       login, password_encrypted, COALESCE(mtapi_token_encrypted, '\x'::bytea), broker_host, server,
		       canonical_subscribed_symbols
		FROM mt_accounts_v2 WHERE id = $1 AND is_active = true
	`, accountID).Scan(&id, &userID, &platform, &broker, &mtapiHost, &mtapiPort,
		&login, &passwordEnc, &mtTokenEnc, &brokerHost, &server, &symbols)
	if err != nil {
		return nil, err
	}
	var password, mtapiToken string
	if len(passwordEnc) > 0 && sec != nil {
		plain, err := sec.Decrypt(ctx, secrets.PurposeMTPassword, passwordEnc)
		if err != nil {
			return nil, fmt.Errorf("decrypt password: %w", err)
		}
		password = string(plain)
	}
	if len(mtTokenEnc) > 0 && sec != nil {
		plain, err := sec.Decrypt(ctx, secrets.PurposeMTAPIToken, mtTokenEnc)
		if err == nil {
			mtapiToken = string(plain)
		}
	}
	return &mdtick.AccountConfig{
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
	}, nil
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
