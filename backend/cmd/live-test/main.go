package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

	"anttrader/internal/mdgateway"
	mt4adapter "anttrader/internal/mdgateway/adapter/mt4"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/secrets"
	mt4gw "anttrader/internal/mdgateway/adapter/mt4"
	pb4 "anttrader/mt4"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
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

	rows, _ := pool.Query(ctx, "SELECT id, mt_type, broker_company, login, broker_host, COALESCE(mtapi_port,'443'), password_encrypted, mtapi_token_encrypted FROM mt_accounts WHERE NOT is_disabled")
	defer rows.Close()

	for rows.Next() {
		var id, mtType, broker, login, host, port string
		var pwEnc, tokenEnc []byte
		rows.Scan(&id, &mtType, &broker, &login, &host, &port, &pwEnc, &tokenEnc)
		if mtType != "MT4" { continue }
		pw, _ := vault.Decrypt(ctx, secrets.PurposeMTPassword, pwEnc)
		token, _ := vault.Decrypt(ctx, secrets.PurposeMTAPIToken, tokenEnc)
		fmt.Printf("=== %s (%s) login=%s ===\n", mtType, broker, login)

		cfg := mdtick.AccountConfig{
			AccountID: id, Broker: broker, Platform: mtType,
			Login: login, Password: string(pw), Server: host,
			MtapiHost: host, MtapiPort: port, MtapiToken: string(token),
		}

		var gw mdgateway.Gateway
		gw = mt4adapter.New(cfg, log)

		if err := gw.Connect(ctx); err != nil {
			fmt.Printf("  Connect FAILED: %v\n", err)
			continue
		}
		fmt.Printf("  Connected! Session=%s\n", gw.SessionID())

		// Get all symbols
		var allSymbols []string
		switch mtType {
		case "MT4":
			if g, ok := gw.(*mt4gw.Gateway); ok {
				mdCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("id", g.SessionID(), "authorization", "Bearer "+cfg.MtapiToken))
				resp, err := g.MT4Client().Symbols(mdCtx, &pb4.SymbolsRequest{Id: g.SessionID()})
				if err != nil { fmt.Printf("  Symbols error: %v\n", err); continue }
				allSymbols = resp.GetResult()
			}
		default:
			fmt.Println("  Only MT4 supported for Symbol fetch")
			continue
		}

		fmt.Printf("  Total symbols: %d\n", len(allSymbols))

		// Filter crypto/gold symbols
		crypto := []string{}
		for _, s := range allSymbols {
			upper := strings.ToUpper(s)
			if strings.Contains(upper, "BTC") || strings.Contains(upper, "ETH") ||
			   strings.Contains(upper, "XAU") || strings.Contains(upper, "XRP") ||
			   strings.Contains(upper, "LTC") || strings.Contains(upper, "CRYPTO") {
				crypto = append(crypto, s)
			}
		}
		fmt.Printf("  Crypto/gold symbols: %v\n", crypto)

		if len(crypto) == 0 && len(allSymbols) > 0 {
			fmt.Printf("  All symbols (first 20): %v\n", allSymbols[:min(20, len(allSymbols))])
		}

		// Subscribe to OnQuote (sends ALL quotes, not filtered by symbols)
		count := 0
		gotTick := make(chan struct{})
		gw.Subscribe(ctx, nil, func(t *mdtick.Tick) {
			count++
			fmt.Printf("  >> TICK #%d: %s bid=%s ask=%s\n", count, t.SymbolRaw, t.Bid.String(), t.Ask.String())
			if count >= 3 { close(gotTick) }
		})

		select {
		case <-gotTick:
			fmt.Printf("  SUCCESS! %d ticks received\n", count)
			gw.Disconnect(ctx)
			return
		case <-time.After(15 * time.Second):
			fmt.Printf("  No ticks in 15s (total: %d)\n", count)
		}
		gw.Disconnect(ctx)
	}
}

func min(a, b int) int { if a < b { return a }; return b }
