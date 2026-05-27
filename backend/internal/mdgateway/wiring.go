package mdgateway

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/mdgateway/backfiller"
)

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
