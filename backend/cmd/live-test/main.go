package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

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

	// Load MT5 account
	var id, broker, login, host, port string
	var pwEnc, tokenEnc []byte
	pool.QueryRow(ctx, "SELECT id, broker_company, login, broker_host, COALESCE(mtapi_port,'443'), password_encrypted, mtapi_token_encrypted FROM mt_accounts WHERE mt_type='MT5' AND NOT is_disabled LIMIT 1").Scan(&id, &broker, &login, &host, &port, &pwEnc, &tokenEnc)
	pw, _ := vault.Decrypt(ctx, secrets.PurposeMTPassword, pwEnc)
	token, _ := vault.Decrypt(ctx, secrets.PurposeMTAPIToken, tokenEnc)
	fmt.Printf("=== MT5 login=%s broker=%s ===\n", login, broker)

	cfg := mdtick.AccountConfig{
		AccountID: id, Broker: broker, Platform: "mt5",
		Login: login, Password: string(pw), Server: host,
		MtapiHost: host, MtapiPort: port, MtapiToken: string(token),
	}

	gw := mt5adapter.New(cfg, log)
	if err := gw.Connect(ctx); err != nil {
		fmt.Printf("Connect FAILED: %v\n", err)
		return
	}
	defer gw.Disconnect(ctx)
	fmt.Printf("Connected! Session=%s\n", gw.SessionID())

	mdCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("id", gw.SessionID(), "authorization", "Bearer "+cfg.MtapiToken))

	// Step 1: Get symbols
	fmt.Println("Fetching symbols...")
	symResp, err := gw.MT5Client().Symbols(mdCtx, &pb5.SymbolsRequest{Id: gw.SessionID()})
	if err != nil {
		fmt.Printf("Symbols RPC error: %v\n", err)
		return
	}
	if symResp.GetError() != nil {
		fmt.Printf("Symbols app error: %s\n", symResp.GetError().GetMessage())
		return
	}

	results := symResp.GetResult()
	fmt.Printf("Total symbols: %d\n", len(results))

	// MT5 SymbolInfo: Currency field IS the symbol code (e.g. "BTCUSDm")
	// Description is the human-readable label (e.g. "Bitcoin vs US Dollar")
	var symbolNames []string
	for _, si := range results {
		if si == nil { continue }
		name := si.GetCurrency()
		if name != "" {
			symbolNames = append(symbolNames, name)
			if len(symbolNames) <= 5 {
				fmt.Printf("  %s = %s\n", name, si.GetDescription())
			}
		}
	}
	fmt.Printf("Extracted names: %d\n", len(symbolNames))

	// Filter crypto
	var crypto []string
	for _, s := range symbolNames {
		upper := strings.ToUpper(s)
		if strings.Contains(upper, "BTC") || strings.Contains(upper, "ETH") ||
		   strings.Contains(upper, "XAU") || strings.Contains(upper, "XRP") {
			crypto = append(crypto, s)
		}
	}
	fmt.Printf("Crypto: %v\n", crypto)

	// Step 2: Try QuoteMany with found crypto symbols (or try common MT5 names)
	quoteSymbols := crypto
	if len(quoteSymbols) == 0 {
		// MT5 may use different naming - try common ones
		quoteSymbols = []string{"BTCUSD", "ETHUSD", "XAUUSD", "BTCUSDm", "ETHUSDm"}
	}
	if len(quoteSymbols) > 10 {
		quoteSymbols = quoteSymbols[:10]
	}

	fmt.Printf("Trying QuoteMany: %v\n", quoteSymbols)
	qmResp, err := gw.MT5Client().GetQuoteMany(mdCtx, &pb5.GetQuoteManyRequest{Symbols: quoteSymbols})
	if err != nil {
		fmt.Printf("QuoteMany RPC error: %v\n", err)
	} else if qmResp.GetError() != nil {
		fmt.Printf("QuoteMany app error: %s\n", qmResp.GetError().GetMessage())
	} else {
		quotes := qmResp.GetResult()
		fmt.Printf("Got %d quotes:\n", len(quotes))
		for _, q := range quotes {
			if q == nil { continue }
			fmt.Printf("  %s: bid=%.5f ask=%.5f vol=%d\n",
				q.GetSymbol(), q.GetBid(), q.GetAsk(), q.GetVolume())
		}
	}

	// Step 3: Try OnQuote stream
	fmt.Println("Starting OnQuote stream...")
	gotAny := make(chan struct{})
	count := 0
	gw.Subscribe(ctx, nil, func(t *mdtick.Tick) {
		count++
		fmt.Printf("  >> TICK #%d: %s bid=%s ask=%s\n", count, t.SymbolRaw, t.Bid.String(), t.Ask.String())
		if count >= 3 { close(gotAny) }
	})

	select {
	case <-gotAny:
		fmt.Printf("SUCCESS: %d ticks received\n", count)
	case <-time.After(15 * time.Second):
		fmt.Printf("No ticks in 15s (got %d)\n", count)
	}
}
