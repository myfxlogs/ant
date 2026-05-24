package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	mt5adapter "anttrader/internal/mdgateway/adapter/mt5"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/secrets"
	pb5 "anttrader/mt5"
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

	var id, broker, login, host, port string
	var pwEnc, tokenEnc []byte
	pool.QueryRow(ctx, "SELECT id, broker_company, login, broker_host, COALESCE(mtapi_port,'443'), password_encrypted, mtapi_token_encrypted FROM mt_accounts WHERE mt_type='MT5' AND NOT is_disabled LIMIT 1").Scan(&id, &broker, &login, &host, &port, &pwEnc, &tokenEnc)
	pw, _ := vault.Decrypt(ctx, secrets.PurposeMTPassword, pwEnc)
	token, _ := vault.Decrypt(ctx, secrets.PurposeMTAPIToken, tokenEnc)
	fmt.Printf("MT5 login=%s broker=%s\n", login, broker)

	cfg := mdtick.AccountConfig{
		AccountID: id, Broker: broker, Platform: "mt5",
		Login: login, Password: string(pw), Server: host,
		MtapiHost: host, MtapiPort: port, MtapiToken: string(token),
	}

	gw := mt5adapter.New(cfg, log)
	if err := gw.Connect(ctx); err != nil { fmt.Printf("Connect FAILED: %v\n", err); return }
	defer gw.Disconnect(ctx)
	fmt.Printf("Connected! Session=%s\n", gw.SessionID())

	mdCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("id", gw.SessionID(), "authorization", "Bearer "+cfg.MtapiToken))

	// Step 1: Get ALL symbols
	symResp, err := gw.MT5Client().Symbols(mdCtx, &pb5.SymbolsRequest{Id: gw.SessionID()})
	if err != nil { fmt.Printf("Symbols RPC error: %v\n", err); return }
	if symResp.GetError() != nil { fmt.Printf("Symbols app error: %s\n", symResp.GetError().GetMessage()); return }

	allSymbols := symResp.GetResult()
	fmt.Printf("Total symbols: %d\n", len(allSymbols))

	// Build symbol list from SymbolInfo.Currency field
	var symNames []string
	for _, si := range allSymbols {
		if si == nil { continue }
		name := si.GetCurrency()
		if name != "" {
			symNames = append(symNames, name)
		}
	}
	fmt.Printf("Symbol names: %d\n", len(symNames))

	// Show sample symbol names
	fmt.Println("First 10 symbols:", symNames[:min(10, len(symNames))])

	// Find crypto
	var crypto []string
	for _, s := range symNames {
		upper := strings.ToUpper(s)
		if strings.Contains(upper, "BTC") || strings.Contains(upper, "ETH") ||
		   strings.Contains(upper, "XAU") || strings.Contains(upper, "XRP") {
			crypto = append(crypto, s)
		}
	}
	fmt.Printf("Crypto/gold symbols: %v\n", crypto)

	if len(crypto) == 0 {
		fmt.Println("No crypto found. Showing all:")
		for _, s := range symNames {
			fmt.Printf("  %s\n", s)
		}
		return
	}

	// Step 2: QuoteMany with crypto symbols
	fmt.Printf("QuoteMany(%v)...\n", crypto[:min(10, len(crypto))])
	qmResp, err := gw.MT5Client().GetQuoteMany(mdCtx, &pb5.GetQuoteManyRequest{Symbols: crypto[:min(10, len(crypto))]})
	if err != nil { fmt.Printf("QuoteMany RPC error: %v\n", err) }
	if qmResp.GetError() != nil { fmt.Printf("QuoteMany app error: %s\n", qmResp.GetError().GetMessage()) }
	quotes := qmResp.GetResult()
	fmt.Printf("Got %d quotes\n", len(quotes))
	for _, q := range quotes {
		if q == nil { continue }
		fmt.Printf("  %s: bid=%.5f ask=%.5f\n", q.GetSymbol(), q.GetBid(), q.GetAsk())
	}
}

func min(a, b int) int { if a < b { return a }; return b }
