package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

	mt4adapter "anttrader/internal/mdgateway/adapter/mt4"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/secrets"
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

		gw := mt4adapter.New(cfg, log)
		if err := gw.Connect(ctx); err != nil {
			fmt.Printf("  Connect FAILED: %v\n", err)
			continue
		}
		fmt.Printf("  Connected! Session=%s\n", gw.SessionID())

		mdCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("id", gw.SessionID(), "authorization", "Bearer "+cfg.MtapiToken))

		// Try Quote RPC first (non-streaming, single quote)
		fmt.Println("  Trying Quote(Id) RPC...")
		qResp, err := gw.MT4Client().Quote(mdCtx, &pb4.QuoteRequest{Id: gw.SessionID()})
		if err != nil {
			fmt.Printf("  Quote RPC error: %v\n", err)
		} else if qResp.GetError() != nil {
			fmt.Printf("  Quote app error: %s\n", qResp.GetError().GetMessage())
		} else {
			q := qResp.GetResult()
			if q != nil {
				fmt.Printf("  Quote SUCCESS: %s bid=%.5f ask=%.5f\n", q.GetSymbol(), q.GetBid(), q.GetAsk())
			} else {
				fmt.Println("  Quote returned nil result")
			}
		}

		// Try QuoteMany with BTCUSDm
		symbols := []string{"BTCUSDm", "ETHUSDm", "XAUUSDm"}
		fmt.Printf("  Trying QuoteMany(%v)...\n", symbols)
		qmResp, err := gw.MT4Client().GetQuoteMany(mdCtx, &pb4.GetQuoteManyRequest{Symbols: symbols})
		if err != nil {
			fmt.Printf("  QuoteMany error: %v\n", err)
		} else if qmResp.GetError() != nil {
			fmt.Printf("  QuoteMany app error: %s\n", qmResp.GetError().GetMessage())
		} else {
			results := qmResp.GetResult()
			fmt.Printf("  QuoteMany got %d quotes\n", len(results))
			for _, q := range results {
				if q == nil { continue }
				fmt.Printf("    %s: bid=%.5f ask=%.5f time=%d\n",
					q.GetSymbol(), q.GetBid(), q.GetAsk(), q.GetTime().GetSeconds())
			}
		}

		// Subscribe to OnQuote and check for Error field in reply
		fmt.Println("  Starting OnQuote stream...")
		gotAny := make(chan struct{})
		gw.Subscribe(ctx, nil, func(t *mdtick.Tick) {
			fmt.Printf("  >> TICK: %s bid=%s ask=%s\n", t.SymbolRaw, t.Bid.String(), t.Ask.String())
			close(gotAny)
		})

		select {
		case <-gotAny:
			fmt.Println("  SUCCESS! Tick received")
		case <-time.After(15 * time.Second):
			fmt.Println("  No ticks in 15s")
		}
		gw.Disconnect(ctx)
		return
	}
	_ = strings.Join
}
