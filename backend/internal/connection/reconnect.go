package connection

import (
	"os"
	"sync"
	"strings"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
	"anttrader/internal/mt4client"
	"anttrader/internal/mt5client"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

var (
	connectionRuntimeLogThrottleMu   sync.Mutex
	connectionRuntimeLogThrottleLast = make(map[string]time.Time)
	connectionRuntimeLogSuppressed   = make(map[string]int)
)

const defaultConnectionLogThrottleWindow = 20 * time.Second

func connectionRuntimeLogWindow() time.Duration {
	v := strings.TrimSpace(os.Getenv("ANTRADER_CONNECTION_LOG_THROTTLE_WINDOW"))
	if v == "" {
		return defaultConnectionLogThrottleWindow
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return defaultConnectionLogThrottleWindow
	}
	return d
}

func (m *ConnectionManager) syncAccountConnectionState(conn *AccountConnection) {
	if m == nil || conn == nil {
		return
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if conn.mt4Conn != nil {
		switch conn.mt4Conn.GetState() {
		case mt4client.StateConnecting:
			conn.State = StateConnecting
		case mt4client.StateDisconnected, mt4client.StateClosed:
			conn.State = StateDisconnected
		case mt4client.StateReady, mt4client.StateSubscribed:
			conn.State = StateConnected
		case mt4client.StateDegraded:
			conn.State = StateReconnecting
		default:
			conn.State = StateDisconnected
		}
		return
	}
	if conn.mt5Conn != nil {
		switch conn.mt5Conn.GetState() {
		case mt5client.StateConnecting:
			conn.State = StateConnecting
		case mt5client.StateDisconnected, mt5client.StateClosed:
			conn.State = StateDisconnected
		case mt5client.StateReady, mt5client.StateSubscribed:
			conn.State = StateConnected
		case mt5client.StateDegraded:
			conn.State = StateReconnecting
		default:
			conn.State = StateDisconnected
		}
	}
}

func (m *ConnectionManager) healthCheckLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkAllConnections()
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *ConnectionManager) checkAllConnections() {
	m.connectionsMu.RLock()
	connections := make([]*AccountConnection, 0, len(m.connections))
	for _, conn := range m.connections {
		connections = append(connections, conn)
	}
	m.connectionsMu.RUnlock()

	for _, conn := range connections {
		logWindow := connectionRuntimeLogWindow()
		m.syncAccountConnectionState(conn)
		if !conn.IsConnected() {
			if !m.ShouldKeepAlive(conn.AccountID) {
				continue
			}
			throttledConnectionRuntimeWarn("health.connection_lost."+conn.AccountID.String(), logWindow,
				"Connection lost, attempting reconnect",
				zap.String("account_id", conn.AccountID.String()))

			go m.Reconnect(conn.AccountID)
		}
	}

	accounts, err := m.accountRepo.GetAllActive(m.ctx)
	if err != nil {
		logger.Error("Failed to get active accounts for health check", zap.Error(err))
		return
	}

	// Subscription-driven lifecycle: keep accounts connected only when there are active subscriptions.
	activeAccountIDs := make(map[uuid.UUID]bool)
	for _, account := range accounts {
		if account == nil {
			continue
		}
		activeAccountIDs[account.ID] = true
	}

	// 检查是否有已禁用账户的连接需要清理
	m.connectionsMu.RLock()
	for accountID, conn := range m.connections {
		if !activeAccountIDs[accountID] {
			// 账户已禁用但连接仍存在，需要清理
			go func(id uuid.UUID, c *AccountConnection) {
				m.Disconnect(m.ctx, id)
			}(accountID, conn)
		}
	}
	m.connectionsMu.RUnlock()

	// Ensure all active accounts have a connection.
	// Limit concurrent connects to avoid storms.
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	for _, account := range accounts {
		if account == nil {
			continue
		}
		if !m.ShouldKeepAlive(account.ID) {
			continue
		}
		if m.checkExistingConnection(account.ID) {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(accID uuid.UUID, accLogin string, acc *model.MTAccount) {
			defer func() {
				<-sem
				wg.Done()
			}()
			if err := m.Connect(m.ctx, acc); err != nil {
				throttledConnectionRuntimeWarn("health.connect_failed."+accID.String(), connectionRuntimeLogWindow(),
					"HealthCheck: failed to connect active account",
					zap.String("account_id", accID.String()),
					zap.String("login", accLogin),
					zap.Error(err))
			}
		}(account.ID, account.Login, account)
	}
	// Block briefly until scheduled connects have finished to avoid piling up across ticks.
	wg.Wait()
}

func (m *ConnectionManager) Reconnect(accountID uuid.UUID) {
	account, err := m.accountRepo.GetByID(m.ctx, accountID)
	if err != nil {
		throttledConnectionRuntimeError("reconnect.get_account_failed."+accountID.String(), connectionRuntimeLogWindow(),
			"Failed to get account for reconnect",
			zap.String("account_id", accountID.String()),
			zap.Error(err))
		return
	}

	if account.IsDisabled {
		return
	}

	if account.LastError != "" && isAuthError(account.LastError) {
		throttledConnectionRuntimeWarn("reconnect.skip_auth_error."+accountID.String(), connectionRuntimeLogWindow(),
			"Account has auth error, skipping reconnect",
			zap.String("account_id", accountID.String()),
			zap.String("login", account.Login),
			zap.String("last_error", account.LastError))
		return
	}

	m.connectionsMu.RLock()
	conn, exists := m.connections[accountID]
	m.connectionsMu.RUnlock()

	if exists {
		conn.mu.Lock()
		conn.State = StateReconnecting
		conn.ReconnectCnt++
		conn.mu.Unlock()
	}

	if err := m.Connect(m.ctx, account); err != nil {
		if isAuthError(err.Error()) {
			throttledConnectionRuntimeWarn("reconnect.auth_error."+accountID.String(), connectionRuntimeLogWindow(),
				"Auth error detected, will not retry reconnect",
				zap.String("account_id", accountID.String()),
				zap.String("login", account.Login),
				zap.Error(err))
			m.accountRepo.UpdateStatus(m.ctx, accountID, "error", err.Error())
			return
		}
		throttledConnectionRuntimeError("reconnect.failed."+accountID.String(), connectionRuntimeLogWindow(),
			"Reconnect failed",
			zap.String("account_id", accountID.String()),
			zap.String("login", account.Login),
			zap.Int("attempt", conn.ReconnectCnt),
			zap.Error(err))
	} else {
		if exists {
			conn.mu.Lock()
			conn.ReconnectCnt = 0
			conn.mu.Unlock()
		}
	}
}

func isAuthError(errMsg string) bool {
	authErrors := []string{
		"Invalid account",
		"Invalid password",
		"Invalid user",
		"Authentication failed",
		"Access denied",
		"Login failed",
	}
	errLower := strings.ToLower(errMsg)
	for _, authErr := range authErrors {
		if strings.Contains(errLower, strings.ToLower(authErr)) {
			return true
		}
	}
	return false
}

func shouldLogConnectionRuntime(key string, window time.Duration) (bool, int) {
	now := time.Now()
	connectionRuntimeLogThrottleMu.Lock()
	defer connectionRuntimeLogThrottleMu.Unlock()
	if last, ok := connectionRuntimeLogThrottleLast[key]; ok && now.Sub(last) < window {
		connectionRuntimeLogSuppressed[key]++
		return false, 0
	}
	suppressed := connectionRuntimeLogSuppressed[key]
	delete(connectionRuntimeLogSuppressed, key)
	connectionRuntimeLogThrottleLast[key] = now
	cutoff := now.Add(-2 * window)
	for k, ts := range connectionRuntimeLogThrottleLast {
		if ts.Before(cutoff) {
			delete(connectionRuntimeLogThrottleLast, k)
			delete(connectionRuntimeLogSuppressed, k)
		}
	}
	return true, suppressed
}

func throttledConnectionRuntimeWarn(key string, window time.Duration, msg string, fields ...zap.Field) {
	if ok, suppressed := shouldLogConnectionRuntime(key, window); ok {
		if suppressed > 0 {
			fields = append(fields, zap.Int("suppressed_count", suppressed))
		}
		logger.Warn(msg, fields...)
	}
}

func throttledConnectionRuntimeError(key string, window time.Duration, msg string, fields ...zap.Field) {
	if ok, suppressed := shouldLogConnectionRuntime(key, window); ok {
		if suppressed > 0 {
			fields = append(fields, zap.Int("suppressed_count", suppressed))
		}
		logger.Error(msg, fields...)
	}
}
