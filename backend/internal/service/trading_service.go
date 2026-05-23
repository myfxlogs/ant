package service

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/config"
	"anttrader/internal/connection"
	"anttrader/internal/model"
	"anttrader/internal/mt4client"
	"anttrader/internal/mt5client"
	"anttrader/internal/repository"
	"anttrader/pkg/faultisol"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

var (
	ErrInvalidOrderType   = NewRiskError(RiskOrderTypeUnsupported, "invalid order type", false)
	ErrInvalidVolume      = NewRiskError(RiskVolumeInvalid, "invalid volume", false)
	ErrConnectionNotFound = errors.New("connection not found")
	ErrTradeWriteDisabled = errors.New("trade write disabled")
)

func tradingWriteEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ANTRADER_TRADING_WRITE_ENABLED")))
	if v == "" {
		return true
	}
	return v == "1" || v == "true" || v == "yes" || v == "y" || v == "on"
}

type TradingService struct {
	accountRepo     *repository.AccountRepository
	tradeLogRepo    *repository.TradeLogRepository
	tradeRecordRepo *repository.TradeRecordRepository
	connManager     *connection.ConnectionManager
	mt4Config       *config.MT4Config
	mt5Config       *config.MT5Config
	executor        *faultisol.IsolatedExecutor
	riskEngine      *RiskEngine
}

func NewTradingService(
	accountRepo *repository.AccountRepository,
	tradeLogRepo *repository.TradeLogRepository,
	tradeRecordRepo *repository.TradeRecordRepository,
	connManager *connection.ConnectionManager,
	mt4Config *config.MT4Config,
	mt5Config *config.MT5Config,
) *TradingService {
	executor := faultisol.NewIsolatedExecutor("trading-service",
		faultisol.WithTimeoutConfig(faultisol.TimeoutConfig{
			Connect:   60 * time.Second,
			Query:     30 * time.Second,
			Trading:   30 * time.Second,
			Subscribe: 10 * time.Second,
			Default:   30 * time.Second,
		}),
	)

	return &TradingService{
		accountRepo:     accountRepo,
		tradeLogRepo:    tradeLogRepo,
		tradeRecordRepo: tradeRecordRepo,
		connManager:     connManager,
		mt4Config:       mt4Config,
		mt5Config:       mt5Config,
		executor:        executor,
		riskEngine:      NewRiskEngine(),
	}
}

type OrderSendRequest struct {
	AccountID  string  `json:"account_id" binding:"required"`
	Symbol     string  `json:"symbol" binding:"required"`
	Type       string  `json:"type" binding:"required"`
	Volume     float64 `json:"volume" binding:"required"`
	Price      float64 `json:"price"`
	Slippage   int32   `json:"slippage"`
	StopLoss   float64 `json:"stoploss"`
	TakeProfit float64 `json:"takeprofit"`
	Comment    string  `json:"comment"`
	Magic      int64   `json:"magic_number"`
}

type OrderModifyRequest struct {
	AccountID  string  `json:"account_id" binding:"required"`
	Ticket     int64   `json:"ticket" binding:"required"`
	StopLoss   float64 `json:"stoploss"`
	TakeProfit float64 `json:"takeprofit"`
	Price      float64 `json:"price"`
}

type OrderCloseRequest struct {
	AccountID   string  `json:"account_id" binding:"required"`
	Ticket      int64   `json:"ticket" binding:"required"`
	Volume      float64 `json:"volume"`
	CloseReason string  `json:"close_reason"`
}

type OrderResponse struct {
	Ticket     int64   `json:"ticket"`
	Symbol     string  `json:"symbol"`
	Type       string  `json:"type"`
	Volume     float64 `json:"volume"`
	Price      float64 `json:"price"`
	StopLoss   float64 `json:"stoploss"`
	TakeProfit float64 `json:"takeprofit"`
	Comment    string  `json:"comment"`
	Magic      int64   `json:"magic_number"`
	OpenTime   string  `json:"open_time"`
	// CloseTime is set for OrderClose results when the bridge exposes it (RFC3339).
	CloseTime string `json:"close_time,omitempty"`
	Profit    float64 `json:"profit"`
	Swap      float64 `json:"swap,omitempty"`
	Commission float64 `json:"commission,omitempty"`
}

type PositionResponse struct {
	Ticket       int64   `json:"ticket"`
	Symbol       string  `json:"symbol"`
	Type         string  `json:"type"`
	Volume       float64 `json:"volume"`
	OpenPrice    float64 `json:"open_price"`
	CurrentPrice float64 `json:"current_price"`
	StopLoss     float64 `json:"stoploss"`
	TakeProfit   float64 `json:"takeprofit"`
	Profit       float64 `json:"profit"`
	Swap         float64 `json:"swap"`
	Commission   float64 `json:"commission"`
	OpenTime     string  `json:"open_time"`
	Comment      string  `json:"comment"`
	Magic        int64   `json:"magic_number"`
}

func (s *TradingService) Connect(ctx context.Context, userID, accountID uuid.UUID) error {
	return s.executor.Execute(ctx, func(ctx context.Context) error {
		account, err := s.getAccountAndVerify(ctx, userID, accountID)
		if err != nil {
			return err
		}

		if s.connManager != nil {
			return s.connManager.Connect(ctx, account)
		}
		return errors.New("connection manager not available")
	}, faultisol.WithOperation("connect"))
}

func (s *TradingService) Disconnect(ctx context.Context, userID, accountID uuid.UUID) error {
	_, err := s.getAccountAndVerify(ctx, userID, accountID)
	if err != nil {
		return err
	}

	if s.connManager != nil {
		return s.connManager.Disconnect(ctx, accountID)
	}
	return errors.New("connection manager not available")
}

func (s *TradingService) getMT4Connection(accountID uuid.UUID) (*mt4client.MT4Connection, error) {
	if s.connManager == nil {
		return nil, errors.New("connection manager not available")
	}
	return s.connManager.GetMT4Connection(accountID)
}

func (s *TradingService) getMT5Connection(accountID uuid.UUID) (*mt5client.MT5Connection, error) {
	if s.connManager == nil {
		return nil, errors.New("connection manager not available")
	}
	return s.connManager.GetMT5Connection(accountID)
}

func (s *TradingService) getAccountAndVerify(ctx context.Context, userID, accountID uuid.UUID) (*model.MTAccount, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.UserID != userID {
		return nil, ErrAccountNotFound
	}
	if account.IsDisabled {
		return nil, errors.New("account is disabled")
	}
	return account, nil
}

// getAccountForTrade 在 getAccountAndVerify 的基础上额外检查：
// 该账户必须具备交易权限（非投资者只读模式）。
// 仅用于写路径（OrderSend / OrderModify / OrderClose）；读路径如
// Connect / Disconnect / 行情 / 持仓查询不应调用这个方法。
func (s *TradingService) getAccountForTrade(ctx context.Context, userID, accountID uuid.UUID) (*model.MTAccount, error) {
	account, err := s.getAccountAndVerify(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	if account.IsInvestor {
		return nil, ErrNoTradePermission
	}
	return account, nil
}

func (s *TradingService) ParseHostPort(hostPort string) (string, int32) {
	parts := strings.Split(hostPort, ":")
	if len(parts) == 2 {
		host := parts[0]
		port, _ := strconv.ParseInt(parts[1], 10, 32)
		return host, int32(port)
	}
	return hostPort, 443
}

func (s *TradingService) logTrade(ctx context.Context, userID, accountID uuid.UUID, action, symbol, orderType string, volume, price float64, ticket int64, profit float64, errMsg string) {
	if s.tradeLogRepo == nil {
		return
	}

	logEntry := &model.TradeLog{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: accountID,
		Action:    action,
		Symbol:    symbol,
		OrderType: orderType,
		Volume:    volume,
		Price:     price,
		Ticket:    ticket,
		Profit:    profit,
		Message:   errMsg,
		CreatedAt: time.Now(),
	}

	if err := s.tradeLogRepo.Create(ctx, logEntry); err != nil {
		logger.Error("Failed to create trade log",
			zap.Error(err),
			zap.String("action", action),
			zap.String("symbol", symbol))
	}
}
