package connection

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func (m *ConnectionManager) checkExistingConnection(accountID uuid.UUID) bool {
	if err := m.connectionsMu.RLock(); err != nil {
		logger.Error("Failed to acquire read lock for connection check", zap.Error(err))
		return false
	}
	defer m.connectionsMu.RUnlock()

	if conn, exists := m.connections[accountID]; exists && conn.IsConnected() {
		return true
	}
	return false
}

func (m *ConnectionManager) registerConnection(accountID uuid.UUID, conn *AccountConnection) error {
	if err := m.connectionsMu.Lock(); err != nil {
		logger.Error("Failed to acquire write lock for connection registration", zap.Error(err))
		return err
	}
	defer m.connectionsMu.Unlock()

	m.connections[accountID] = conn
	return nil
}

func (m *ConnectionManager) restoreConnections() error {
	accounts, err := m.accountRepo.GetAll(m.ctx)
	if err != nil {
		return fmt.Errorf("failed to get all accounts: %w", err)
	}

	if len(accounts) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	// Limit concurrent MT connections to avoid startup storms.
	sem := make(chan struct{}, 5)
	for _, account := range accounts {
		if account.IsDisabled {
			// 检查是否有此禁用账户的残留连接，如果有则清理
			m.connectionsMu.RLock()
			_, exists := m.connections[account.ID]
			m.connectionsMu.RUnlock()
			
			if exists {
				go func(id uuid.UUID) {
					m.Disconnect(m.ctx, id)
				}(account.ID)
			}
			continue
		}

		wg.Add(1)
		go func(acc *model.MTAccount) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := m.Connect(m.ctx, acc); err != nil {
				logger.Warn("Failed to restore connection",
					zap.String("account_id", acc.ID.String()),
					zap.Error(err))
			}
		}(account)
	}
	wg.Wait()

	return nil
}

func (m *ConnectionManager) Connect(ctx context.Context, account *model.MTAccount) error {
	// Fast-path: already connected.
	if m.checkExistingConnection(account.ID) {
		return nil
	}

	// Guard against duplicate concurrent connects.
	// A single account might be triggered by restoreConnections() and stream subscriptions at the same time.
	if err := m.connectionsMu.RLock(); err == nil {
		existing, exists := m.connections[account.ID]
		m.connectionsMu.RUnlock()

		if exists && existing != nil {
			existing.mu.RLock()
			state := existing.State
			lastActive := existing.LastActive
			existing.mu.RUnlock()

			if (state == StateConnecting || state == StateReconnecting) && time.Since(lastActive) < 2*time.Minute {
				return nil
			}
		}
	}

	conn := &AccountConnection{
		AccountID:     account.ID,
		MTType:        account.MTType,
		State:         StateConnecting,
		LastActive:    time.Now(),
		subscriptions: make(map[string]int),
	}

	if err := m.registerConnection(account.ID, conn); err != nil {
		return err
	}

	m.accountRepo.UpdateStatus(m.ctx, account.ID, "connecting", "")

	var err error
	if account.MTType == "MT4" {
		err = m.connectMT4(ctx, conn, account)
	} else {
		err = m.connectMT5(ctx, conn, account)
	}

	if err != nil {
		conn.mu.Lock()
		conn.State = StateError
		conn.LastError = err.Error()
		conn.mu.Unlock()
		m.accountRepo.UpdateStatus(m.ctx, account.ID, "error", err.Error())
		logger.Error("Failed to connect to MT server",
			zap.String("account_id", account.ID.String()),
			zap.String("login", account.Login),
			zap.String("mt_type", account.MTType),
			zap.Error(err))

		userMsg, st, detail := FormatConnectionError(err)
		if userMsg == "" {
			userMsg = "Connection failed"
		}
		m.logConnection(account.UserID, account.ID, model.EventTypeError, st, userMsg, detail, account.BrokerHost)
		return err
	}

	conn.mu.Lock()
	conn.State = StateConnected
	conn.LastError = ""
	conn.LastConnectedAt = time.Now()
	conn.mu.Unlock()
	m.accountRepo.UpdateStatus(m.ctx, account.ID, "connected", "")
	m.accountRepo.UpdateConnectedAt(m.ctx, account.ID, time.Now())

	m.logConnection(account.UserID, account.ID, model.EventTypeConnect, model.ConnectionStatusSuccess, "Connected successfully", "", account.BrokerHost)
	m.AutoSyncOrderHistoryOnConnect(account.ID, account.MTType)

	if account.MTType == "MT4" && conn.mt4Conn != nil {
		m.accountRepo.UpdateToken(m.ctx, account.ID, conn.mt4Conn.GetToken())
	} else if account.MTType == "MT5" && conn.mt5Conn != nil {
		m.accountRepo.UpdateToken(m.ctx, account.ID, conn.mt5Conn.GetID())
	}

	return nil
}

func (m *ConnectionManager) Disconnect(ctx context.Context, accountID uuid.UUID) error {
	if err := m.connectionsMu.Lock(); err != nil {
		logger.Error("Failed to acquire lock for Disconnect", zap.Error(err))
		return err
	}
	conn, exists := m.connections[accountID]
	delete(m.connections, accountID)
	m.connectionsMu.Unlock()

	if !exists {
		return nil
	}

	if conn.mt4Conn != nil {
		m.mt4Client.Disconnect(ctx, conn.mt4Conn.GetAccountID())
	}
	if conn.mt5Conn != nil {
		m.mt5Client.Disconnect(ctx, conn.mt5Conn.GetAccountID())
	}

	m.accountRepo.UpdateStatus(m.ctx, accountID, "disconnected", "")
	return nil
}

func (m *ConnectionManager) MarkDisconnected(accountID uuid.UUID) {
	if err := m.connectionsMu.RLock(); err != nil {
		logger.Error("Failed to acquire read lock for MarkDisconnected", zap.Error(err))
		return
	}
	conn, exists := m.connections[accountID]
	m.connectionsMu.RUnlock()

	if !exists {
		return
	}

	conn.mu.Lock()
	conn.State = StateDisconnected
	conn.mu.Unlock()

	m.accountRepo.UpdateStatus(m.ctx, accountID, "disconnected", "connection lost")
	logger.Warn("Connection marked as disconnected",
		zap.String("account_id", accountID.String()))
}

func (m *ConnectionManager) parseHostPort(hostPort string) (string, int32) {
	parts := strings.Split(hostPort, ":")
	if len(parts) == 2 {
		host := parts[0]
		port, _ := strconv.ParseInt(parts[1], 10, 32)
		return host, int32(port)
	}
	return hostPort, 443
}
