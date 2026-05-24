package mt5client

import (
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"

	"anttrader/internal/config"
	pb "anttrader/mt5"
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

	MaxRetries     = 3
	RetryDelay     = 5 * time.Second
	ConnectTimeout = 60 * time.Second
)

type MT5Connection struct {
	mu             sync.RWMutex
	accountID      string
	conn           *grpc.ClientConn
	id             string
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
	mt5Client          pb.MT5Client
	tradingClient      pb.TradingClient
	subscriptionClient pb.SubscriptionsClient
	streamClient       pb.StreamsClient

	// 实时流通道
	quoteCh   chan *pb.Quote
	profitCh  chan *pb.ProfitUpdate
	orderCh   chan *pb.OrderUpdateSummary
}

type MT5Client struct {
	cfg         *config.MT5Config
	connections map[string]*MT5Connection
	mu          sync.RWMutex

	searchConn   *grpc.ClientConn
	searchConnMu sync.Once
}

type BrokerCompany struct {
	CompanyName string         `json:"company_name"`
	Results     []BrokerResult `json:"results"`
}

type BrokerResult struct {
	Name   string   `json:"name"`
	Access []string `json:"access"`
}
