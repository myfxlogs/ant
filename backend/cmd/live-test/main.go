package main

import (
	"context"
	"fmt"
	"os"
	"time"

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

	var id, broker, login, host, port string
	var pwEnc, tokenEnc []byte
	err = pool.QueryRow(ctx, `
		SELECT id, broker_company, login, broker_host,
			COALESCE(mtapi_port, '443'),
			password_encrypted, mtapi_token_encrypted
		FROM mt_accounts WHERE mt_type='MT5' AND NOT is_disabled LIMIT 1`,
	).Scan(&id, &broker, &login, &host, &port, &pwEnc, &tokenEnc)
	if err != nil { panic(fmt.Sprintf("query: %v", err)) }

	fmt.Printf("ID=%s broker=%s login=%s host=%s port=%s\n", id, broker, login, host, port)

	pw, err := vault.Decrypt(ctx, secrets.PurposeMTPassword, pwEnc)
	if err != nil { panic(fmt.Sprintf("decrypt pw: %v", err)) }
	token, err := vault.Decrypt(ctx, secrets.PurposeMTAPIToken, tokenEnc)
	if err != nil { panic(fmt.Sprintf("decrypt token: %v", err)) }

	fmt.Printf("Password decrypted: %d bytes, Token decrypted: %d bytes\n", len(pw), len(token))

	cfg := mdtick.AccountConfig{
		AccountID: id, Broker: broker, Platform: "mt5",
		Login: login, Password: string(pw), Server: host,
		MtapiHost: host, MtapiPort: port, MtapiToken: string(token),
	}

	gw := mt5adapter.New(cfg, log)
	fmt.Println("Connecting...")
	if err := gw.Connect(ctx); err != nil {
		fmt.Printf("Connect FAILED: %v\n", err)
		return
	}
	fmt.Println("Connected!")

	symbols := []string{"BTCUSD"}
	fmt.Printf("Subscribing to: %v\n", symbols)

	tickCount := 0
	done := make(chan struct{})
	err = gw.Subscribe(ctx, symbols, func(t *mdtick.Tick) {
		tickCount++
		fmt.Printf("[%d] %s bid=%s ask=%s\n", tickCount, t.SymbolRaw, t.Bid.String(), t.Ask.String())
		if tickCount >= 3 { close(done) }
	})
	if err != nil { panic(err) }

	select {
	case <-done:
		fmt.Printf("SUCCESS: %d ticks received\n", tickCount)
	case <-time.After(30 * time.Second):
		fmt.Printf("TIMEOUT: no ticks in 30s (market closed or no quotes)\n")
	}
	gw.Disconnect(ctx)
}
