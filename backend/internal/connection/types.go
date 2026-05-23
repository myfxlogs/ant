package connection

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/config"
	"anttrader/internal/mt4client"
	"anttrader/internal/mt5client"
	"anttrader/internal/pkg/lockutil"
	"anttrader/internal/repository"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateReconnecting
	StateError
)

func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

type AccountConnection struct {
	AccountID       uuid.UUID
	MTType          string
	State           ConnectionState
	LastActive      time.Time
	LastConnectedAt time.Time
	LastError       string
	ReconnectCnt    int

	mt4Conn *mt4client.MT4Connection
	mt5Conn *mt5client.MT5Connection

	subscriptions map[string]int
	mu            sync.RWMutex
}

func (c *AccountConnection) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.mt4Conn != nil {
		return c.mt4Conn.IsConnected()
	}
	if c.mt5Conn != nil {
		return c.mt5Conn.IsConnected()
	}
	return false
}

func (c *AccountConnection) GetMT4Connection() *mt4client.MT4Connection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mt4Conn
}

func (c *AccountConnection) GetMT5Connection() *mt5client.MT5Connection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mt5Conn
}

func (c *AccountConnection) AddSubscription(streamType string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscriptions[streamType]++
	c.LastActive = time.Now()
	return c.subscriptions[streamType]
}

func (c *AccountConnection) RemoveSubscription(streamType string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.subscriptions[streamType] > 0 {
		c.subscriptions[streamType]--
	}
	c.LastActive = time.Now()
	return c.subscriptions[streamType]
}

func (c *AccountConnection) GetSubscriptionCount(streamType string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.subscriptions[streamType]
}

func (c *AccountConnection) TotalSubscriptions() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	total := 0
	for _, count := range c.subscriptions {
		total += count
	}
	return total
}

type ConnectionManager struct {
	accountRepo     *repository.AccountRepository
	tradeRecordRepo *repository.TradeRecordRepository
	logRepo         *repository.LogRepository
	mt4Config       *config.MT4Config
	mt5Config       *config.MT5Config

	historySyncMu   sync.Mutex
	lastAutoSyncAt map[uuid.UUID]time.Time

	connections map[uuid.UUID]*AccountConnection
	lockChecker *lockutil.LockOrderChecker
	connectionsMu *lockutil.SafeLocker
	stateMu     *lockutil.SafeLocker

	mt4Client *mt4client.MT4Client
	mt5Client *mt5client.MT5Client

	connLogThrottleMu   sync.Mutex
	connLogThrottleLast map[string]time.Time

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func (m *ConnectionManager) AddSubscription(accountID uuid.UUID, streamType string) {
	if err := m.connectionsMu.Lock(); err != nil {
		logger.Error("Failed to acquire lock for AddSubscription", zap.Error(err))
		return
	}
	conn, exists := m.connections[accountID]
	if !exists || conn == nil {
		conn = &AccountConnection{
			AccountID:      accountID,
			State:          StateDisconnected,
			LastActive:     time.Now(),
			subscriptions:  make(map[string]int),
			ReconnectCnt:   0,
			LastError:      "",
			LastConnectedAt: time.Time{},
		}
		m.connections[accountID] = conn
	}
	m.connectionsMu.Unlock()

	prevTotal := conn.TotalSubscriptions()
	conn.AddSubscription(streamType)
	newTotal := conn.TotalSubscriptions()

	// MT-official-like: first subscriber should immediately trigger a connect attempt.
	// Enabled/disabled is handled by accountRepo; enable does not force connect (A semantics).
	if prevTotal == 0 && newTotal > 0 {
		go func() {
			if m == nil {
				return
			}
			account, err := m.accountRepo.GetByID(m.ctx, accountID)
			if err != nil || account == nil {
				return
			}
			if account.IsDisabled {
				return
			}
			cctx, cancel := context.WithTimeout(m.ctx, 15*time.Second)
			defer cancel()
			_ = m.Connect(cctx, account)
		}()
	}
}

func (m *ConnectionManager) RemoveSubscription(accountID uuid.UUID, streamType string) {
	if err := m.connectionsMu.RLock(); err != nil {
		logger.Error("Failed to acquire read lock for RemoveSubscription", zap.Error(err))
		return
	}
	conn, exists := m.connections[accountID]
	m.connectionsMu.RUnlock()
	if !exists || conn == nil {
		return
	}
	conn.RemoveSubscription(streamType)
}

func (m *ConnectionManager) ShouldKeepAlive(accountID uuid.UUID) bool {
	if err := m.connectionsMu.RLock(); err != nil {
		logger.Error("Failed to acquire read lock for ShouldKeepAlive", zap.Error(err))
		return false
	}
	conn, exists := m.connections[accountID]
	m.connectionsMu.RUnlock()
	if !exists || conn == nil {
		return false
	}
	return conn.TotalSubscriptions() > 0
}

func NewConnectionManager(
	accountRepo *repository.AccountRepository,
	tradeRecordRepo *repository.TradeRecordRepository,
	logRepo *repository.LogRepository,
	mt4Config *config.MT4Config,
	mt5Config *config.MT5Config,
) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	lockChecker := lockutil.NewLockOrderChecker()
	
	connectionsMu := lockutil.NewSafeLocker("connections", 1, lockChecker)
	stateMu := lockutil.NewSafeLocker("state", 2, lockChecker)

	return &ConnectionManager{
		accountRepo:     accountRepo,
		tradeRecordRepo: tradeRecordRepo,
		logRepo:         logRepo,
		mt4Config:       mt4Config,
		mt5Config:       mt5Config,
		lastAutoSyncAt:  make(map[uuid.UUID]time.Time),
		connections:     make(map[uuid.UUID]*AccountConnection),
		lockChecker:     lockChecker,
		connectionsMu:   connectionsMu,
		stateMu:         stateMu,
		mt4Client:       mt4client.NewMT4Client(mt4Config),
		mt5Client:       mt5client.NewMT5Client(mt5Config),
		connLogThrottleLast: make(map[string]time.Time),
		ctx:             ctx,
		cancel:          cancel,
	}
}

func (m *ConnectionManager) Start() error {
	m.wg.Add(1)
	go m.healthCheckLoop()

	m.wg.Add(1)
	go m.cleanupLoop()

	// Rehydrate MT connections for persisted accounts on process startup.
	// Without this step, account_status may remain "connected" in DB while
	// in-memory client connections are empty after restart.
	if err := m.restoreConnections(); err != nil {
		logger.Warn("Failed to restore persisted MT connections on startup", zap.Error(err))
	}

	return nil
}

func (m *ConnectionManager) Stop() {
	m.cancel()
	m.wg.Wait()

	if err := m.connectionsMu.Lock(); err != nil {
		logger.Error("Failed to acquire lock for stopping connection manager", zap.Error(err))
		return
	}
	defer m.connectionsMu.Unlock()

	for _, conn := range m.connections {
		if conn.mt4Conn != nil {
			m.mt4Client.Disconnect(m.ctx, conn.mt4Conn.GetAccountID())
		}
		if conn.mt5Conn != nil {
			m.mt5Client.Disconnect(m.ctx, conn.mt5Conn.GetAccountID())
		}
	}
	m.connections = make(map[uuid.UUID]*AccountConnection)

}
