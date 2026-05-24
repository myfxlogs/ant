package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"anttrader/internal/mdgateway"
	mt4adapter "anttrader/internal/mdgateway/adapter/mt4"
	mt5adapter "anttrader/internal/mdgateway/adapter/mt5"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/secrets"
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

	// Load MT4 account (try Exness MT4 which might have different symbol setup)
	var id, broker, login, host, port string
	var pwEnc, tokenEnc []byte

	// Try both accounts
	rows, _ := pool.Query(ctx, "SELECT id, mt_type, broker_company, login, broker_host, COALESCE(mtapi_port,'443'), password_encrypted, mtapi_token_encrypted FROM mt_accounts WHERE NOT is_disabled")
	defer rows.Close()
	
	for rows.Next() {
		var mtType string
		rows.Scan(&id, &mtType, &broker, &login, &host, &port, &pwEnc, &tokenEnc)
		pw, _ := vault.Decrypt(ctx, secrets.PurposeMTPassword, pwEnc)
		token, _ := vault.Decrypt(ctx, secrets.PurposeMTAPIToken, tokenEnc)
		fmt.Printf("Account: %s type=%s broker=%s host=%s\n", login, mtType, broker, host)
		
		cfg := mdtick.AccountConfig{
			AccountID: id, Broker: broker, Platform: mtType,
			Login: login, Password: string(pw), Server: host,
			MtapiHost: host, MtapiPort: port, MtapiToken: string(token),
		}
		
		var gw mdgateway.Gateway
		if mtType == "MT5" { gw = mt5adapter.New(cfg, log) } else { gw = mt4adapter.New(cfg, log) }
		fmt.Println("  Connecting...")
		if err := gw.Connect(ctx); err != nil {
			fmt.Printf("  Connect FAILED: %v\n", err)
			continue
		}
		fmt.Printf("  Connected! SessionID=%s\n", gw.SessionID())

		// Try different crypto symbols including broker-specific suffixes
		for _, symbols := range [][]string{
			{"BTCUSD", "ETHUSD", "XAUUSD"},
			{"BTCUSDm", "ETHUSDm", "XAUUSDm"},
			{"BTCUSD.pro", "BTCUSD.c"},
		} {
			fmt.Printf("  Trying symbols: %v\n", symbols)
			gotTick := make(chan struct{})
			err = gw.Subscribe(ctx, symbols, func(t *mdtick.Tick) {
				fmt.Printf("  >> TICK: %s bid=%s ask=%s\n", t.SymbolRaw, t.Bid.String(), t.Ask.String())
				close(gotTick)
			})
			if err != nil {
				fmt.Printf("  Subscribe error: %v\n", err)
				continue
			}
			select {
			case <-gotTick:
				fmt.Println("  GOT TICK!")
				gw.Disconnect(ctx)
				return
			case <-time.After(8 * time.Second):
				fmt.Println("  No ticks in 8s")
			}
		}
		gw.Disconnect(ctx)
	}
	fmt.Println("Both accounts tested, no ticks received (weekend).")
}
