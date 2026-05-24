// Package mt4 — MT4 gateway adapter (mtapi gRPC direct, ADR-0003).
// Connects via pb.MT4Client and pb.ConnectionClient.
package mt4

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "anttrader/mt4"
	"anttrader/internal/mdgateway"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Gateway implements mdgateway.Gateway for MT4 via mtapi.io gRPC.
type Gateway struct {
	cfg mdtick.AccountConfig
	log *zap.Logger

	mu        sync.RWMutex
	conn      *grpc.ClientConn
	client    pb.MT4Client     // main trading client
	connCli   pb.ConnectionClient
	streamCli pb.StreamsClient    // quote streaming
	sessionID string
	cancelSub context.CancelFunc
}

func New(cfg mdtick.AccountConfig, log *zap.Logger) *Gateway {
	return &Gateway{cfg: cfg, log: log}
}

func (g *Gateway) Platform() string  { return "mt4" }
func (g *Gateway) AccountID() string { return g.cfg.AccountID }

func (g *Gateway) Connect(ctx context.Context) error {
	addr := g.cfg.MtapiHost + ":" + g.cfg.MtapiPort
	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16*1024*1024)),
	)
	if err != nil { return fmt.Errorf("mt4: dial %s: %w", addr, err) }

	g.mu.Lock()
	g.conn = conn
	g.client = pb.NewMT4Client(conn)
	g.connCli = pb.NewConnectionClient(conn)
	g.streamCli = pb.NewStreamsClient(conn)
	g.mu.Unlock()

	g.log.Info("mt4: connected", zap.String("account", g.cfg.AccountID))
	return nil
}

func (g *Gateway) Disconnect(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.cancelSub != nil { g.cancelSub(); g.cancelSub = nil }
	if g.conn != nil { g.conn.Close(); g.conn = nil }
	g.client = nil
	g.connCli = nil; g.streamCli = nil
	g.sessionID = ""
	return nil
}

// Subscribe starts streaming quotes via OnQuote server streaming.
func (g *Gateway) Subscribe(ctx context.Context, symbols []string, handler mdgateway.TickHandler) error {
	g.mu.RLock()
	streamCli := g.streamCli
	g.mu.RUnlock()
	if streamCli == nil { return fmt.Errorf("mt4: not connected") }

	subCtx, cancel := context.WithCancel(ctx)
	g.mu.Lock(); g.cancelSub = cancel; g.mu.Unlock()

	// Use Bearer token metadata (Q-002)
	md := metadata.New(map[string]string{"authorization": "Bearer " + g.cfg.MtapiToken})
	subCtx = metadata.NewOutgoingContext(subCtx, md)

	stream, err := streamCli.OnQuote(subCtx, &pb.OnQuoteRequest{Id: g.sessionID})
	if err != nil { return fmt.Errorf("mt4: subscribe: %w", err) }

	go func() {
		for {
			quote, err := stream.Recv()
			if err != nil { g.log.Warn("mt4: stream recv", zap.Error(err)); return }

			arrivedMs := time.Now().UTC().UnixMilli()              // Q-001
			q := quote.GetResult()
			if q == nil { continue }

			handler(&mdtick.Tick{
				UserID: g.cfg.UserID, AccountID: g.cfg.AccountID,
				Broker: g.cfg.Broker, Platform: "mt4",
				SymbolRaw: q.GetSymbol(), Canonical: "",
				TsUnixMs: q.GetTime().AsTime().UnixMilli(), ArrivedUnixMs: arrivedMs,
				Bid: decimal.NewFromFloat(q.GetBid()), Ask: decimal.NewFromFloat(q.GetAsk()),
			})
		}
	}()
	return nil
}

func (g *Gateway) HealthCheck(ctx context.Context) error {
	g.mu.RLock(); defer g.mu.RUnlock()
	if g.conn == nil { return fmt.Errorf("mt4: not connected") }
	return nil
}

func (g *Gateway) SessionID() string {
	g.mu.RLock(); defer g.mu.RUnlock()
	return g.sessionID
}
