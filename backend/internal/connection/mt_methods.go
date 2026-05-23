package connection

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
	"anttrader/internal/mt4client"
	"anttrader/internal/mt5client"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func (m *ConnectionManager) RegisterMT4Connection(accountID uuid.UUID, conn *mt4client.MT4Connection) {
	if err := m.connectionsMu.Lock(); err != nil {
		logger.Error("Failed to acquire lock for RegisterMT4Connection", zap.Error(err))
		return
	}
	defer m.connectionsMu.Unlock()

	accountConn, exists := m.connections[accountID]
	if !exists {
		accountConn = &AccountConnection{
			AccountID:     accountID,
			MTType:        "MT4",
			State:         StateConnected,
			LastActive:    time.Now(),
			subscriptions: make(map[string]int),
		}
		m.connections[accountID] = accountConn
	}

	accountConn.mu.Lock()
	accountConn.mt4Conn = conn
	accountConn.State = StateConnected
	accountConn.LastError = ""
	accountConn.mu.Unlock()

}

func (m *ConnectionManager) RegisterMT5Connection(accountID uuid.UUID, conn *mt5client.MT5Connection) {
	if err := m.connectionsMu.Lock(); err != nil {
		logger.Error("Failed to acquire lock for RegisterMT5Connection", zap.Error(err))
		return
	}
	defer m.connectionsMu.Unlock()

	accountConn, exists := m.connections[accountID]
	if !exists {
		accountConn = &AccountConnection{
			AccountID:     accountID,
			MTType:        "MT5",
			State:         StateConnected,
			LastActive:    time.Now(),
			subscriptions: make(map[string]int),
		}
		m.connections[accountID] = accountConn
	}

	accountConn.mu.Lock()
	accountConn.mt5Conn = conn
	accountConn.State = StateConnected
	accountConn.LastError = ""
	accountConn.mu.Unlock()

}

func (m *ConnectionManager) GetMT4Connection(accountID uuid.UUID) (*mt4client.MT4Connection, error) {
	if err := m.connectionsMu.RLock(); err != nil {
		logger.Error("Failed to acquire read lock for GetMT4Connection", zap.Error(err))
		return nil, err
	}
	conn, exists := m.connections[accountID]
	m.connectionsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("connection not found for account: %s", accountID)
	}

	mt4Conn := conn.GetMT4Connection()
	if mt4Conn == nil || !mt4Conn.IsConnected() {
		return nil, fmt.Errorf("MT4 connection not active for account: %s", accountID)
	}

	conn.LastActive = time.Now()
	return mt4Conn, nil
}

func (m *ConnectionManager) GetMT5Connection(accountID uuid.UUID) (*mt5client.MT5Connection, error) {
	if err := m.connectionsMu.RLock(); err != nil {
		logger.Error("Failed to acquire read lock for GetMT5Connection", zap.Error(err))
		return nil, err
	}
	conn, exists := m.connections[accountID]
	m.connectionsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("connection not found for account: %s", accountID)
	}

	mt5Conn := conn.GetMT5Connection()
	if mt5Conn == nil || !mt5Conn.IsConnected() {
		return nil, fmt.Errorf("MT5 connection not active for account: %s", accountID)
	}

	conn.LastActive = time.Now()
	return mt5Conn, nil
}

func (m *ConnectionManager) connectMT4(ctx context.Context, conn *AccountConnection, account *model.MTAccount) error {
	loginInt, _ := strconv.ParseInt(account.Login, 10, 64)
	host, port := m.parseHostPort(account.BrokerHost)

	mt4Conn, err := m.mt4Client.Connect(ctx, int32(loginInt), account.Password, host, port)
	if err != nil {
		return err
	}

	conn.mu.Lock()
	conn.mt4Conn = mt4Conn
	conn.mu.Unlock()

	m.accountRepo.UpdateToken(m.ctx, account.ID, mt4Conn.GetToken())

	summary, err := mt4Conn.AccountSummary(ctx)
	if err != nil {
		logger.Warn("Failed to get MT4 account summary", zap.Error(err))
	} else {
		accountType := "demo"
		if summary.Type.String() == "AccountType_Real" {
			accountType = "real"
		} else if summary.Type.String() == "AccountType_Contest" {
			accountType = "contest"
		}

		m.accountRepo.UpdateAccountFullInfo(m.ctx, account.ID,
			summary.Balance, summary.Credit, summary.Equity,
			summary.Margin, summary.FreeMargin, summary.MarginLevel,
			int(summary.Leverage), summary.Currency, accountType, summary.IsInvestor)
	}

	return nil
}

func (m *ConnectionManager) connectMT5(ctx context.Context, conn *AccountConnection, account *model.MTAccount) error {
	loginInt, _ := strconv.ParseUint(account.Login, 10, 64)
	host, port := m.parseHostPort(account.BrokerHost)

	mt5Conn, err := m.mt5Client.ConnectWithRetry(ctx, loginInt, account.Password, host, port)
	if err != nil {
		return err
	}

	conn.mu.Lock()
	conn.mt5Conn = mt5Conn
	conn.mu.Unlock()

	m.accountRepo.UpdateToken(m.ctx, account.ID, mt5Conn.GetID())

	summary, err := mt5Conn.AccountSummary(ctx)
	if err != nil {
		logger.Warn("Failed to get MT5 account summary", zap.Error(err))
	} else {
		accountType := "demo"
		if summary.Type == "real" {
			accountType = "real"
		} else if summary.Type == "contest" {
			accountType = "contest"
		}

		accountMethod := "default"
		if summary.Method.String() == "AccMethod_Netting" {
			accountMethod = "netting"
		} else if summary.Method.String() == "AccMethod_Hedging" {
			accountMethod = "hedging"
		}

		m.accountRepo.UpdateAccountFullInfo(m.ctx, account.ID,
			summary.Balance, summary.Credit, summary.Equity,
			summary.Margin, summary.FreeMargin, summary.MarginLevel,
			int(summary.Leverage), summary.Currency, accountType, summary.IsInvestor)
		m.accountRepo.UpdateAccountMethod(m.ctx, account.ID, accountMethod)
	}

	return nil
}
