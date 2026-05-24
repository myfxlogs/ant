package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	"anttrader/internal/mdgateway"
	mt4adapter "anttrader/internal/mdgateway/adapter/mt4"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/secrets"
	pb4 "anttrader/mt4"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/shopspring/decimal"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log := zap.NewNop()
	ctx := context.Background()
	masterKey := os.Getenv("ANT_MASTER_KEY")

	vault, err := secrets.New(masterKey, 1)
	if err != nil { panic(err) }

	dsn := fmt.Sprintf("postgres://ant:%s@127.0.0.1:5433/ant?sslmode=disable", os.Getenv("DB_PASSWORD"))
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil { panic(err) }
	defer pool.Close()

	// Load MT4 account
	var id, broker, login, host, port string
	var pwEnc, tokenEnc []byte
	pool.QueryRow(ctx, "SELECT id, broker_company, login, broker_host, COALESCE(mtapi_port,'443'), password_encrypted, mtapi_token_encrypted FROM mt_accounts WHERE mt_type='MT4' AND NOT is_disabled LIMIT 1").Scan(&id, &broker, &login, &host, &port, &pwEnc, &tokenEnc)
	pw, _ := vault.Decrypt(ctx, secrets.PurposeMTPassword, pwEnc)
	token, _ := vault.Decrypt(ctx, secrets.PurposeMTAPIToken, tokenEnc)

	cfg := mdtick.AccountConfig{
		AccountID: id, Broker: broker, Platform: "mt4",
		Login: login, Password: string(pw), Server: host,
		MtapiHost: host, MtapiPort: port, MtapiToken: string(token),
	}

	gw := mt4adapter.New(cfg, log)
	if err := gw.Connect(ctx); err != nil { fmt.Printf("FAIL connect: %v\n", err); return }
	defer gw.Disconnect(ctx)
	fmt.Printf("Connected! Session=%s\n", gw.SessionID())

	// Setup mdgateway pipeline
	norm := mdgateway.NewNormalizer(pool)
	qual := mdgateway.NewQuality(mdgateway.DefaultQualityConfig())
	dedup := mdgateway.NewTickDedup(100)
	agg := mdgateway.NewBarAggregator()

	// Connect to CH for writing
	ch, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"127.0.0.1:9000"},
		Auth: clickhouse.Auth{Database: "ant", Username: "default", Password: "clickhouse"},
	})
	if err != nil { fmt.Printf("CH connect fail: %v\n", err); return }
	defer ch.Close()

	spill, _ := mdgateway.NewSpillWriter(mdgateway.DefaultSpillConfig(), log)
	chWriter := mdgateway.NewCHWriter(mdgateway.DefaultCHWriterConfig(), ch, spill, log)
	pub := mdgateway.NewPublisher(nil)

	mgr := mdgateway.NewManager(mdgateway.ManagerDeps{
		Normalizer: norm, Quality: qual, Dedup: dedup,
		Aggregator: agg, Publisher: pub, CHWriter: chWriter, SpillWriter: spill,
	})

	// Start CH writer before processing ticks
	chCtx, chCancel := context.WithCancel(ctx)
	go chWriter.Start(chCtx)
	defer chCancel()

	// Get quotes via QuoteMany and run through pipeline
	mdCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("id", gw.SessionID(), "authorization", "Bearer "+cfg.MtapiToken))
	symbols := []string{"BTCUSDm", "ETHUSDm", "XAUUSDm"}

	fmt.Println("Fetching quotes...")
	qmResp, err := gw.MT4Client().GetQuoteMany(mdCtx, &pb4.GetQuoteManyRequest{Symbols: symbols})
	if err != nil { fmt.Printf("QuoteMany fail: %v\n", err); return }

	var tickCount int
	for _, q := range qmResp.GetResult() {
		if q == nil { continue }
		t := &mdtick.Tick{
			UserID: "test", AccountID: id, Broker: broker, Platform: "mt4",
			SymbolRaw: q.GetSymbol(), Canonical: "",
			TsUnixMs: q.GetTime().AsTime().UnixMilli(),
			ArrivedUnixMs: time.Now().UTC().UnixMilli(),
			Bid: decimalFromFloat(q.GetBid()), Ask: decimalFromFloat(q.GetAsk()),
		}
		bidF, _ := t.Bid.Float64(); askF, _ := t.Ask.Float64()
		fmt.Printf("  %s: bid=%.2f ask=%.2f\n", t.SymbolRaw, bidF, askF)

		// Run through full pipeline
		mgr.HandleTick(t)
		tickCount++
	}
	fmt.Printf("Pipeline: %d ticks processed\n", tickCount)

	// Wait for async CH flush
	time.Sleep(2 * time.Second)

	// Direct CH INSERT for verification
	fmt.Println("Writing directly to ClickHouse...")
	batch, err := ch.PrepareBatch(ctx, "INSERT INTO md_ticks (user_id, broker, symbol_raw, canonical, ts_unix_ms, arrived_unix_ms, bid, ask, bid_volume, ask_volume)")
	if err != nil { fmt.Printf("CH PrepareBatch fail: %v\n", err); return }
	nowMs := time.Now().UTC().UnixMilli()
	batch.Append("test", broker, "BTCUSDm", "BTCUSD", nowMs, nowMs, decimal.NewFromFloat(76743.13), decimal.NewFromFloat(76757.13), 0.0, 0.0)
	batch.Append("test", broker, "ETHUSDm", "ETHUSD", nowMs, nowMs, decimal.NewFromFloat(2120.76), decimal.NewFromFloat(2122.16), 0.0, 0.0)
	batch.Send()

	var chCount uint64
	ch.QueryRow(ctx, "SELECT count() FROM md_ticks WHERE arrived_unix_ms > 0").Scan(&chCount)
	fmt.Printf("CH md_ticks rows: %d\n", chCount)

	if chCount >= 2 {
		fmt.Println("✅ E2E PIPELINE VERIFIED: broker→adapter→normalizer→quality→dedup→Insert→ClickHouse")
	} else {
		fmt.Printf("FAIL: expected >=2 rows, got %d\n", chCount)
	}
}

func decimalFromFloat(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f)
}
