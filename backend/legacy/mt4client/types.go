package mt4client

import (
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"

	"anttrader/internal/config"
	pb "anttrader/mt4"
	"anttrader/pkg/faultisol"
)

type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateReady
	StateSubscribed
	StateDegraded
	StateClosed

	// Backward-compatible aliases.
	StateIdle      = StateDisconnected
	StateConnected = StateReady
)

type MT4Connection struct {
	mu             sync.RWMutex
	accountID      string
	conn           *grpc.ClientConn
	token          string
	state          ConnectionState
	lastActive     time.Time
	createdAt      time.Time
	done           chan struct{}
	reconnectCnt   int
	circuitBreaker *faultisol.CircuitBreaker
	executor       *faultisol.IsolatedExecutor

	profitRecvAt atomic.Int64
	orderRecvAt  atomic.Int64

	connectionClient   pb.ConnectionClient
	mt4Client          pb.MT4Client
	tradingClient      pb.TradingClient
	subscriptionClient pb.SubscriptionsClient
	streamClient       pb.StreamsClient

	// 实时流通道
	quoteCh   chan *pb.QuoteEventArgs
	profitCh  chan *pb.ProfitUpdate
	orderCh   chan *pb.OrderUpdateSummary
}

type MT4Client struct {
	cfg         *config.MT4Config
	connections map[string]*MT4Connection
	mu          sync.RWMutex

	searchConn   *grpc.ClientConn
	searchConnMu sync.Once
}
