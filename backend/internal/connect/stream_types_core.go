package connect

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"anttrader/internal/config"
	"anttrader/internal/coordination"
	"anttrader/internal/connection"
	"anttrader/internal/model"
	"anttrader/internal/pkg/goroutine"
	"anttrader/internal/repository"

	v1 "anttrader/gen/proto"
)

type AccountRepo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.MTAccount, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.MTAccount, error)
}

type StreamSubscriber struct {
	ID        string
	AccountID string
	UserID    string
	EventCh   chan *v1.StreamEvent
	QuoteCh   chan *v1.Quote
	OrderCh   chan *v1.OrderUpdateEvent
	ProfitCh  chan *v1.ProfitUpdateEvent
	Cancel    context.CancelFunc
	Symbols   []string

	OrderDropCnt  atomic.Uint64
	ProfitDropCnt atomic.Uint64

	NeedsProfit bool
	NeedsOrder  bool
}

type userSummaryStats struct {
	pnlToday           float64
	pnlWeek            float64
	pnlMonth           float64
	tradesToday        int32
	tradesWeek         int32
	tradesMonth        int32
	winRate            float64
	profitFactor       float64
	maxDrawdownPercent float64
	maxConsecWins      int32
	maxConsecLosses    int32
}

type accountStatusRecord struct {
	status  string
	message string
	at      time.Time
}

type AccountStream struct {
	AccountID   string
	Subscribers map[string]*StreamSubscriber
	mu          sync.RWMutex
	Ctx         context.Context
	Cancel      context.CancelFunc

	snapshotMu   sync.RWMutex
	lastProfit   *v1.ProfitUpdateEvent
	lastLedger   *v1.LedgerEntryEvent
	profitNotify chan struct{}
	profitVer    atomic.Uint64
	// openedOrders keeps the latest known opened orders snapshot keyed by ticket.
	// Snapshot is sent to new subscribers; deltas are sent as order update events.
	openedOrders   map[int64]*v1.OrderUpdateEvent
	positions      map[int64]*v1.PositionUpdateEvent
	deals          map[int64]*v1.DealUpdateEvent
	lastOrderDelta *v1.OrderUpdateEvent
	orderNotify    chan struct{}
	orderVer       atomic.Uint64
	EventCh        chan *v1.StreamEvent
	closeOnce      sync.Once

	streamEnabled map[string]bool
	streamMu      sync.RWMutex

	idleSeq atomic.Uint64
}

type StreamService struct {
	accountRepo     AccountRepo
	tradeRecordRepo *repository.TradeRecordRepository
	analyticsRepo   *repository.AnalyticsRepository
	connManager     *connection.ConnectionManager
	mt4Config       *config.MT4Config
	mt5Config       *config.MT5Config

	disableSupervisorForTest bool

	subscribers map[string]*StreamSubscriber
	mu          sync.RWMutex

	accountStreams map[string]*AccountStream
	streamMu       sync.RWMutex

	accountChangeMu   sync.Mutex
	accountChangeSubs map[string]map[string]chan accountEnabledChange

	activeUserStreams map[string]int
	activeMu          sync.Mutex

	zeroBalanceAccounts sync.Map

	supervisors   map[string]*SessionAgent
	supervisorsMu sync.Mutex

	sessionLeader *coordination.SessionLeader
	leaderLeases  map[string]*coordination.Lease
	redisClient   *redis.Client
	instanceID    string

	eventBus *coordination.EventBus

	forwarders   map[string]func()
	forwardersMu sync.Mutex

	wakeupOnce sync.Once

	goroutineMgr *goroutine.Manager

	statusMu          sync.Mutex
	lastAccountStatus map[string]accountStatusRecord
}

type accountEnabledChange struct {
	accountID string
	enabled   bool
}

func NewStreamService(
	accountRepo AccountRepo,
	tradeRecordRepo *repository.TradeRecordRepository,
	analyticsRepo *repository.AnalyticsRepository,
	connManager *connection.ConnectionManager,
	mt4Config *config.MT4Config,
	mt5Config *config.MT5Config,
) *StreamService {
	return &StreamService{
		accountRepo:       accountRepo,
		tradeRecordRepo:   tradeRecordRepo,
		analyticsRepo:     analyticsRepo,
		connManager:       connManager,
		mt4Config:         mt4Config,
		mt5Config:         mt5Config,
		subscribers:       make(map[string]*StreamSubscriber),
		accountStreams:    make(map[string]*AccountStream),
		activeUserStreams: make(map[string]int),
		supervisors:       make(map[string]*SessionAgent),
		leaderLeases:      make(map[string]*coordination.Lease),
		forwarders:        make(map[string]func()),
		accountChangeSubs: make(map[string]map[string]chan accountEnabledChange),
		goroutineMgr:      goroutine.NewManager(goroutine.WithMaxGoroutines(5000)),
		lastAccountStatus: make(map[string]accountStatusRecord),
	}
}

func (s *StreamService) SetRedisClientForTest(client *redis.Client) {
	if s == nil {
		return
	}
	s.redisClient = client
}

func (s *StreamService) DisableSupervisorForTest() {
	if s == nil {
		return
	}
	s.disableSupervisorForTest = true
}
