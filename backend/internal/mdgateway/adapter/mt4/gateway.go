package mt4
import (
	"context"; "fmt"; "strings"; "sync"; "time"
	pb "anttrader/mt4"
	"anttrader/internal/mdgateway"; "anttrader/internal/mdgateway/adapter/mdtick"
	"github.com/shopspring/decimal"; "go.uber.org/zap"
	"crypto/tls"
	"google.golang.org/grpc"; "google.golang.org/grpc/credentials"; "google.golang.org/grpc/metadata"
)
type Gateway struct {
	cfg mdtick.AccountConfig; log *zap.Logger
	mu sync.RWMutex; conn *grpc.ClientConn
	client pb.MT4Client; connCli pb.ConnectionClient; streamCli pb.StreamsClient
	sessionID string; cancelSub context.CancelFunc
}
func New(cfg mdtick.AccountConfig, log *zap.Logger) *Gateway { return &Gateway{cfg: cfg, log: log} }
func (g *Gateway) Platform() string { return "mt4" }
func (g *Gateway) AccountID() string { return g.cfg.AccountID }
func (g *Gateway) Connect(ctx context.Context) error {
	gateway := g.cfg.MtapiHost
	if gateway == "" || gateway == g.cfg.Server { gateway = "mt4grpc3.mtapi.io:443" }
	if !strings.Contains(gateway, ":") { gateway += ":443" }
	conn, err := grpc.DialContext(ctx, gateway, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16*1024*1024)))
	if err != nil { return fmt.Errorf("mt4 dial: %w", err) }
	g.mu.Lock(); g.conn=conn; g.client=pb.NewMT4Client(conn); g.connCli=pb.NewConnectionClient(conn); g.streamCli=pb.NewStreamsClient(conn); g.mu.Unlock()
	md := metadata.New(map[string]string{"authorization":"Bearer "+g.cfg.MtapiToken})
	loginCtx := metadata.NewOutgoingContext(ctx, md)
	loginResp, err := g.connCli.Connect(loginCtx, &pb.ConnectRequest{Host: g.cfg.Server, Port: 443, User: int32(strToInt(g.cfg.Login)), Password: g.cfg.Password})
	if err != nil { g.conn.Close(); g.conn=nil; return fmt.Errorf("mt4 login: %w", err) }
	g.mu.Lock(); g.sessionID=loginResp.GetResult(); g.mu.Unlock()
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
	if sc == nil { return fmt.Errorf("mt4: not connected") }
	subCtx, cancel := context.WithCancel(ctx); g.mu.Lock(); g.cancelSub=cancel; g.mu.Unlock()
	md := metadata.New(map[string]string{"authorization":"Bearer "+g.cfg.MtapiToken}); subCtx=metadata.NewOutgoingContext(subCtx, md)
	stream, err := sc.OnQuote(subCtx, &pb.OnQuoteRequest{Id: g.sessionID})
	if err != nil { return fmt.Errorf("mt4 subscribe: %w", err) }
	go func() {
		for {
			quote, err := stream.Recv()
			if err != nil { g.log.Warn("mt4 recv", zap.Error(err)); return }
			q := quote.GetResult(); if q == nil { continue }
			handler(&mdtick.Tick{
				UserID:g.cfg.UserID, AccountID:g.cfg.AccountID, Broker:g.cfg.Broker, Platform:"mt4",
				SymbolRaw:q.GetSymbol(), Canonical:"", TsUnixMs:q.GetTime().AsTime().UnixMilli(),
				ArrivedUnixMs:time.Now().UTC().UnixMilli(),
				Bid:decimal.NewFromFloat(q.GetBid()), Ask:decimal.NewFromFloat(q.GetAsk()),
			})
		}
	}()
	return nil
}
func (g *Gateway) HealthCheck(ctx context.Context) error {
	g.mu.RLock(); defer g.mu.RUnlock()
	if g.conn == nil { return fmt.Errorf("mt4: not connected") }; return nil
}
func (g *Gateway) SessionID() string { g.mu.RLock(); defer g.mu.RUnlock(); return g.sessionID }
func strToInt(s string) int {
	v := 0
	for _, c := range s { if c >= '0' && c <= '9' { v = v*10 + int(c-'0') } }
	return v
}
