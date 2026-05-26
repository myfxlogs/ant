package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	mt4adapter "anttrader/internal/mdgateway/adapter/mt4"
	"anttrader/internal/mdgateway"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/secrets"
	pb4 "anttrader/mt4"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	log := zap.NewNop()
	masterKey := os.Getenv("ANT_MASTER_KEY")
	vault, _ := secrets.New(masterKey, 1)
	dsn := fmt.Sprintf("postgres://ant:%s@127.0.0.1:5433/ant?sslmode=disable", os.Getenv("DB_PASSWORD"))
	pool, _ := pgxpool.New(ctx, dsn); defer pool.Close()

	var id, broker, login, host, port string
	var pwEnc, tokenEnc []byte
	pool.QueryRow(ctx, "SELECT id, broker_company, login, broker_host, COALESCE(mtapi_port,'443'), password_encrypted, mtapi_token_encrypted FROM mt_accounts WHERE mt_type='MT4' AND NOT is_disabled LIMIT 1").Scan(&id, &broker, &login, &host, &port, &pwEnc, &tokenEnc)
	pw, _ := vault.Decrypt(ctx, secrets.PurposeMTPassword, pwEnc)
	token, _ := vault.Decrypt(ctx, secrets.PurposeMTAPIToken, tokenEnc)
	fmt.Printf("MT4 %s @ %s\n", login, broker)

	cfg := mdtick.AccountConfig{AccountID: id, Broker: broker, Platform: "mt4", Login: login, Password: string(pw), Server: host, MtapiHost: host, MtapiPort: port, MtapiToken: string(token)}
	gw := mt4adapter.New(cfg, log)
	gw.Connect(ctx); defer gw.Disconnect(ctx)
	fmt.Printf("Connected session=%s\n", gw.SessionID())

	mdCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("id", gw.SessionID(), "authorization", "Bearer "+cfg.MtapiToken))
	resp, _ := gw.MT4Client().GetQuoteMany(mdCtx, &pb4.GetQuoteManyRequest{Symbols: []string{"BTCUSDm", "ETHUSDm", "XAUUSDm"}})
	quotes := resp.GetResult()
	fmt.Printf("MT4 quotes: %d\n", len(quotes))

	norm := mdgateway.NewNormalizer(pool)
	qual := mdgateway.NewQuality(mdgateway.DefaultQualityConfig())
	dedup := mdgateway.NewTickDedup(100)
	ch, _ := clickhouse.Open(&clickhouse.Options{Addr: []string{"127.0.0.1:9000"}, Auth: clickhouse.Auth{Database: "ant", Username: "default", Password: "clickhouse"}})
	defer ch.Close()

	for _, q := range quotes {
		bid, _ := decimal.NewFromString(fmt.Sprintf("%.5f", q.GetBid()))
		ask, _ := decimal.NewFromString(fmt.Sprintf("%.5f", q.GetAsk()))
		t := &mdtick.Tick{
			UserID: "admin@1.com", AccountID: id, Broker: broker, Platform: "mt4",
			SymbolRaw: q.GetSymbol(), Canonical: "",
			TsUnixMs: q.GetTime().AsTime().UnixMilli(), ArrivedUnixMs: time.Now().UTC().UnixMilli(),
			Bid: bid, Ask: ask,
		}
		t.Canonical = norm.Resolve(ctx, broker, t.SymbolRaw)
		if qual.Check(ctx, t).Dropped { fmt.Printf("  %s DROPPED\n", t.SymbolRaw); continue }
		if dedup.Seen(t) { fmt.Printf("  %s DUP\n", t.SymbolRaw); continue }

		batch, _ := ch.PrepareBatch(ctx, "INSERT INTO ant.md_ticks (user_id, account_id, broker, symbol_raw, canonical, ts_unix_ms, arrived_unix_ms, bid, ask, bid_volume, ask_volume)")
		batch.Append(t.UserID, t.AccountID, t.Broker, t.SymbolRaw, t.Canonical, t.TsUnixMs, t.ArrivedUnixMs, t.Bid, t.Ask, t.BidVolume, t.AskVolume)
		batch.Send()
		fmt.Printf("  %s -> %s bid=%s ask=%s OK\n", t.SymbolRaw, t.Canonical, t.Bid.String(), t.Ask.String())
	}

	var count uint64
	ch.QueryRow(ctx, "SELECT count() FROM ant.md_ticks WHERE arrived_unix_ms > 0").Scan(&count)
	fmt.Printf("\nCH md_ticks total: %d rows\n", count)
}
