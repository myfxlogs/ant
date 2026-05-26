package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"time"

	pb "anttrader/mt4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

func main() {
	gateway := "mt4grpc3.mtapi.io:443"
	login := "95172262"
	password := os.Getenv("MT4_PASSWORD")
	if password == "" {
		password = "QxhrPqrizFg0iTWNOnabaFvv"
	}
	brokerHost := "43.199.125.167"
	symbol := "BTCUSDm"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Dial mtapi
	conn, err := grpc.DialContext(ctx, gateway,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithBlock(),
	)
	if err != nil {
		fmt.Printf("DIAL ERROR: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	connCli := pb.NewConnectionClient(conn)
	tradingCli := pb.NewTradingClient(conn)

	// 2. Connect
	tempID := "mdgw-" + login
	md := metadata.New(map[string]string{"id": tempID})
	loginCtx := metadata.NewOutgoingContext(ctx, md)

	loginResp, err := connCli.Connect(loginCtx, &pb.ConnectRequest{
		Host:     brokerHost,
		Port:     443,
		User:     int32(strToInt(login)),
		Password: password,
		Id:       &tempID,
	})
	if err != nil {
		fmt.Printf("CONNECT ERROR: %v\n", err)
		os.Exit(1)
	}

	sessionID := loginResp.GetResult()
	respErr := loginResp.GetError()
	fmt.Printf("Connect Result (sessionID): %q\n", sessionID)
	fmt.Printf("Connect Error: %+v\n", respErr)

	if sessionID == "" {
		fmt.Println("FATAL: empty session ID")
		os.Exit(1)
	}

	// 3. Place order
	orderCtx, orderCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer orderCancel()
	orderMd := metadata.New(map[string]string{"id": sessionID})
	orderCtx = metadata.NewOutgoingContext(orderCtx, orderMd)

	fmt.Printf("\nPlacing order: %s BUY 0.01 lots, session=%q\n", symbol, sessionID)

	orderResp, err := tradingCli.OrderSend(orderCtx, &pb.OrderSendRequest{
		Id:        sessionID,
		Symbol:    symbol,
		Operation: pb.Op_Op_Buy,
		Volume:    0.01,
		Slippage:  100,
		Comment:   "ant-test",
	})
	if err != nil {
		fmt.Printf("ORDER SEND ERROR: %v\n", err)
		os.Exit(1)
	}

	result := orderResp.GetResult()
	orderErr := orderResp.GetError()
	fmt.Printf("Order Result: %+v\n", result)
	if result != nil {
		fmt.Printf("  Ticket: %d\n", result.GetTicket())
		fmt.Printf("  Symbol: %s\n", result.GetSymbol())
		fmt.Printf("  Lots: %.2f\n", result.GetLots())
		fmt.Printf("  OpenPrice: %.5f\n", result.GetOpenPrice())
		fmt.Printf("  Comment: %s\n", result.GetComment())
	}
	if orderErr != nil && orderErr.GetCode() != 0 {
		fmt.Printf("Order Error: code=%d msg=%s\n", orderErr.GetCode(), orderErr.GetMessage())
	}

	// 4. Check connection state
	checkResp, err := connCli.CheckConnect(ctx, &pb.CheckConnectRequest{Id: sessionID})
	if err != nil {
		fmt.Printf("CHECK CONNECT ERROR: %v\n", err)
	} else {
		fmt.Printf("CheckConnect Result: %q, Error: %+v\n", checkResp.GetResult(), checkResp.GetError())
	}
}

func strToInt(s string) int {
	v := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + int(c-'0')
		}
	}
	return v
}
