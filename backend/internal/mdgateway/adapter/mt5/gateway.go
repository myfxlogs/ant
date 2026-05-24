package mt5
import (
	"context"; "fmt"; "strings"; "sync"; "time"
	"github.com/google/uuid"
	pb "anttrader/mt5"
	"anttrader/internal/mdgateway"; "anttrader/internal/mdgateway/adapter/mdtick"
	"github.com/shopspring/decimal"; "go.uber.org/zap"
	"crypto/tls"
	"google.golang.org/grpc"; "google.golang.org/grpc/credentials"; "google.golang.org/grpc/metadata"
)
type Gateway struct {
	cfg mdtick.AccountConfig; log *zap.Logger
	mu sync.RWMutex; conn *grpc.ClientConn
	client pb.MT5Client; connCli pb.ConnectionClient; streamCli pb.StreamsClient
	sessionID string; cancelSub context.CancelFunc
}
func New(cfg mdtick.AccountConfig, log *zap.Logger) *Gateway { return &Gateway{cfg: cfg, log: log} }
func (g *Gateway) Platform() string { return "mt5" }
func (g *Gateway) AccountID() string { return g.cfg.AccountID }
func (g *Gateway) Connect(ctx context.Context) error {
	// Dial mtapi.io gateway (not broker directly)
	gateway := g.cfg.MtapiHost
	if gateway == "" || gateway == g.cfg.Server { gateway = "mt5grpc3.mtapi.io:443" }
	if !strings.Contains(gateway, ":") { gateway += ":443" }
	conn, err := grpc.DialContext(ctx, gateway, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16*1024*1024)))
	if err != nil { return fmt.Errorf("mt5 dial: %w", err) }
	g.mu.Lock(); g.conn=conn; g.client=pb.NewMT5Client(conn); g.connCli=pb.NewConnectionClient(conn); g.streamCli=pb.NewStreamsClient(conn); g.mu.Unlock()
	// Login to broker with unique connection ID
	sessionUUID := uuid.NewString()
	md := metadata.New(map[string]string{"authorization":"Bearer "+g.cfg.MtapiToken, "id": sessionUUID})
	loginCtx := metadata.NewOutgoingContext(ctx, md)
	brokerHost := g.cfg.Server
	if idx := strings.LastIndex(brokerHost, ":"); idx > 0 { brokerHost = brokerHost[:idx] }
	loginResp, err := g.connCli.Connect(loginCtx, &pb.ConnectRequest{Host: brokerHost, Port: 443, User: strToUint64(g.cfg.Login), Password: g.cfg.Password, Id: &sessionUUID})
	if err != nil { g.conn.Close(); g.conn=nil; return fmt.Errorf("mt5 login: %w", err) }
	token := loginResp.GetResult()
	if token == "" { g.conn.Close(); g.conn=nil; return fmt.Errorf("mt5 login: empty token") }
	g.mu.Lock(); g.sessionID=token; g.mu.Unlock()
	return nil
}
func (g *Gateway) Disconnect(ctx context.Context) error {
	g.mu.Lock(); defer g.mu.Unlock()
	if g.cancelSub != nil { g.cancelSub(); g.cancelSub = nil }
	if g.conn != nil { g.conn.Close(); g.conn = nil }
	g.client=nil; g.connCli=nil; g.streamCli=nil; g.sessionID=""; return nil
}
func (g *Gateway) Subscribe(ctx context.Context, syms []string, handler mdgateway.TickHandler) error {
	g.mu.RLock(); sc := g.streamCli; g.mu.RUnlock()
	if sc == nil { return fmt.Errorf("mt5: not connected") }
	subCtx, cancel := context.WithCancel(ctx); g.mu.Lock(); g.cancelSub=cancel; g.mu.Unlock()
	md := metadata.New(map[string]string{"authorization":"Bearer "+g.cfg.MtapiToken}); subCtx=metadata.NewOutgoingContext(subCtx, md)
	stream, err := sc.OnQuote(subCtx, &pb.OnQuoteRequest{Id: g.sessionID})
	if err != nil { return fmt.Errorf("mt5 subscribe: %w", err) }
	go func() {
		for {
			tick, err := stream.Recv()
			if err != nil { g.log.Warn("mt5 recv", zap.Error(err)); return }
			q := tick.GetResult(); if q == nil { continue }
			handler(&mdtick.Tick{
				UserID:g.cfg.UserID, AccountID:g.cfg.AccountID, Broker:g.cfg.Broker, Platform:"mt5",
				SymbolRaw:q.GetSymbol(), Canonical:"", TsUnixMs:q.GetTime().AsTime().UnixMilli(),
				ArrivedUnixMs:time.Now().UTC().UnixMilli(),
				Bid:decimal.NewFromFloat(q.GetBid()), Ask:decimal.NewFromFloat(q.GetAsk()),
				BidVolume:float64(q.GetVolume()),
			})
		}
	}()
	return nil
}
func (g *Gateway) HealthCheck(ctx context.Context) error {
	g.mu.RLock(); defer g.mu.RUnlock()
	if g.conn == nil { return fmt.Errorf("mt5: not connected") }; return nil
}
func (g *Gateway) SessionID() string { g.mu.RLock(); defer g.mu.RUnlock(); return g.sessionID }
func strToUint64(s string) uint64 {
	var v uint64
	for _, c := range s { if c >= '0' && c <= '9' { v = v*10 + uint64(c-'0') } }
	return v
}
