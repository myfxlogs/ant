package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/config"
	"anttrader/internal/connection"
	"anttrader/internal/event"
	"anttrader/internal/model"
	"anttrader/internal/mt4client"
	"anttrader/internal/mt5client"
	"anttrader/internal/repository"
	"anttrader/pkg/logger"
)

var (
	ErrInvalidPlatform     = errors.New("invalid platform, must be MT4 or MT5")
	ErrAccountNotFound     = errors.New("account not found")
	ErrAccountNotConnected = errors.New("account not connected")
	ErrConnectionFailed    = errors.New("failed to connect to MT server")
)

type AccountService struct {
	repo             *repository.AccountRepository
	connMgr          *connection.ConnectionManager
	eventBus         *event.Bus
	mt4Client        *mt4client.MT4Client
	mt5Client        *mt5client.MT5Client
	mt4Config        *config.MT4Config
	mt5Config        *config.MT5Config
	dynamicConfigSvc *DynamicConfigService
	logService       *LogService
}

func NewAccountService(
	repo *repository.AccountRepository,
	connMgr *connection.ConnectionManager,
	mt4Config *config.MT4Config,
	mt5Config *config.MT5Config,
	dynamicConfigSvc *DynamicConfigService,
	logService *LogService,
) *AccountService {
	return &AccountService{
		repo:             repo,
		connMgr:          connMgr,
		eventBus:         event.GetBus(),
		mt4Config:        mt4Config,
		mt5Config:        mt5Config,
		dynamicConfigSvc: dynamicConfigSvc,
		logService:       logService,
	}
}

type BindAccountRequest struct {
	MTType        string `json:"mt_type" binding:"required,oneof=MT4 MT5"`
	BrokerCompany string `json:"broker_company" binding:"required"`
	BrokerServer  string `json:"broker_server" binding:"required"`
	BrokerHost    string `json:"broker_host" binding:"required"`
	Login         string `json:"login" binding:"required"`
	Password      string `json:"password" binding:"required"`
	Alias         string `json:"alias"`
}

type BindAccountResponse struct {
	ID            uuid.UUID `json:"id"`
	MTType        string    `json:"mt_type"`
	BrokerCompany string    `json:"broker_company"`
	BrokerServer  string    `json:"broker_server"`
	BrokerHost    string    `json:"broker_host"`
	Login         string    `json:"login"`
	Alias         string    `json:"alias"`
	Balance       float64   `json:"balance"`
	Equity        float64   `json:"equity"`
	Currency      string    `json:"currency"`
	AccountStatus string    `json:"account_status"`
	AccountType   string    `json:"account_type"`
	IsInvestor    bool      `json:"is_investor"`
}

func (s *AccountService) BindAccount(ctx context.Context, userID uuid.UUID, req *BindAccountRequest) (*BindAccountResponse, error) {
	if req.MTType != "MT4" && req.MTType != "MT5" {
		return nil, ErrInvalidPlatform
	}

	maxAccounts, limitEnabled := s.dynamicConfigSvc.MaxAccountsPerUser(ctx)
	if limitEnabled {
		count, err := s.repo.CountByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if count >= maxAccounts {
			return nil, errors.New("已达到最大账户数限制")
		}
	}

	existing, err := s.repo.GetByLoginAndHost(ctx, req.Login, req.BrokerHost)
	if err != nil && !errors.Is(err, repository.ErrAccountNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, repository.ErrAccountAlreadyExists
	}

	host, port := s.parseHostPort(req.BrokerHost)
	loginInt, _ := strconv.ParseInt(req.Login, 10, 64)
	accountID := uuid.New()

	var balance, equity, margin, freeMargin, marginLevel float64
	var leverage int
	var currency, mtToken, accountType string
	var isInvestor bool

	if req.MTType == "MT4" {
		s.mt4Client = mt4client.NewMT4Client(s.mt4Config)
		loginInt32 := int32(loginInt)
		mt4Conn, err := s.mt4Client.Connect(ctx, loginInt32, req.Password, host, port)
		if err != nil {
			userMsg, st, detail := connection.FormatConnectionError(err)
			if userMsg == "" {
				userMsg = "MT4 connection failed"
			}
			s.logConnection(userID, accountID, model.EventTypeConnect, st, userMsg, detail, req.BrokerHost, loginInt)
			return nil, fmt.Errorf("MT4连接失败: %w", err)
		}

		summary, err := mt4Conn.AccountSummary(ctx)
		if err != nil {
			s.mt4Client.Disconnect(ctx, mt4Conn.GetAccountID())
			userMsg, st, detail := connection.FormatConnectionError(err)
			if userMsg == "" {
				userMsg = "Failed to get account summary"
			}
			s.logConnection(userID, accountID, model.EventTypeConnect, st, userMsg, detail, req.BrokerHost, loginInt)
			return nil, fmt.Errorf("获取账户信息失败: %w", err)
		}

		// BindAccount 仅用于绑定/校验，不应遗留后台连接与流。
		// 注意：mt4Conn 属于 s.mt4Client（非 ConnectionManager.mt4Client），注册到 connMgr 会导致后续无法正确断开。
		s.mt4Client.Disconnect(ctx, mt4Conn.GetAccountID())

		balance = summary.Balance
		equity = summary.Equity
		margin = summary.Margin
		freeMargin = summary.FreeMargin
		marginLevel = summary.MarginLevel
		leverage = int(summary.Leverage)
		currency = summary.Currency
		mtToken = mt4Conn.GetToken()
		isInvestor = false

		accountType = "unknown"

	} else {
		s.mt5Client = mt5client.NewMT5Client(s.mt5Config)
		mt5Conn, err := s.mt5Client.Connect(ctx, uint64(loginInt), req.Password, host, port)
		if err != nil {
			userMsg, st, detail := connection.FormatConnectionError(err)
			if userMsg == "" {
				userMsg = "MT5 connection failed"
			}
			s.logConnection(userID, accountID, model.EventTypeConnect, st, userMsg, detail, req.BrokerHost, loginInt)
			return nil, fmt.Errorf("MT5连接失败: %w", err)
		}

		summary, err := mt5Conn.AccountSummary(ctx)
		if err != nil {
			s.mt5Client.Disconnect(ctx, mt5Conn.GetAccountID())
			userMsg, st, detail := connection.FormatConnectionError(err)
			if userMsg == "" {
				userMsg = "Failed to get account summary"
			}
			s.logConnection(userID, accountID, model.EventTypeConnect, st, userMsg, detail, req.BrokerHost, loginInt)
			return nil, fmt.Errorf("获取账户信息失败: %w", err)
		}

		// BindAccount 仅用于绑定/校验，不应遗留后台连接与流。
		s.mt5Client.Disconnect(ctx, mt5Conn.GetAccountID())

		balance = summary.Balance
		equity = summary.Equity
		margin = summary.Margin
		freeMargin = summary.FreeMargin
		marginLevel = summary.MarginLevel
		leverage = int(summary.Leverage)
		currency = summary.Currency
		mtToken = mt5Conn.GetID()
		isInvestor = summary.IsInvestor
		accountType = strings.ToLower(summary.Type)
		if accountType == "" {
			accountType = "unknown"
		}

	}

	account := &model.MTAccount{
		ID:            accountID,
		UserID:        userID,
		MTType:        req.MTType,
		BrokerCompany: req.BrokerCompany,
		BrokerServer:  req.BrokerServer,
		BrokerHost:    req.BrokerHost,
		Login:         req.Login,
		Password:      req.Password,
		Alias:         req.Alias,
		Balance:       balance,
		Equity:        equity,
		Margin:        margin,
		FreeMargin:    freeMargin,
		MarginLevel:   marginLevel,
		Leverage:      leverage,
		Currency:      currency,
		AccountType:   accountType,
		IsInvestor:    isInvestor,
		MTToken:       mtToken,
		AccountStatus: "connected",
	}

	if err := s.repo.Create(ctx, account); err != nil {
		s.logConnection(userID, accountID, model.EventTypeConnect, model.ConnectionStatusFailed, "Failed to create account record", err.Error(), req.BrokerHost, loginInt)
		return nil, err
	}

	s.repo.UpdateToken(ctx, account.ID, account.MTToken)
	s.logConnection(userID, accountID, model.EventTypeConnect, model.ConnectionStatusSuccess, "Account connected successfully", "", req.BrokerHost, loginInt)

	return &BindAccountResponse{
		ID:            account.ID,
		MTType:        account.MTType,
		BrokerCompany: account.BrokerCompany,
		BrokerServer:  account.BrokerServer,
		BrokerHost:    account.BrokerHost,
		Login:         account.Login,
		Alias:         account.Alias,
		Balance:       account.Balance,
		Equity:        account.Equity,
		Currency:      account.Currency,
		AccountStatus: account.AccountStatus,
		AccountType:   account.AccountType,
		IsInvestor:    account.IsInvestor,
	}, nil
}

func (s *AccountService) GetAccounts(ctx context.Context, userID uuid.UUID) ([]*model.MTAccount, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *AccountService) GetAccount(ctx context.Context, userID, accountID uuid.UUID) (*model.MTAccount, error) {
	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.UserID != userID {
		return nil, ErrAccountNotFound
	}
	return account, nil
}

func (s *AccountService) DeleteAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account.UserID != userID {
		return ErrAccountNotFound
	}
	return s.repo.Delete(ctx, accountID)
}

func (s *AccountService) DisableAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account.UserID != userID {
		return ErrAccountNotFound
	}

	if err := s.repo.SetDisabled(ctx, accountID, true); err != nil {
		return err
	}

	if s.connMgr != nil {
		if err := s.connMgr.Disconnect(ctx, accountID); err != nil {
			// Log but don't fail
		}
	}

	return nil
}

func (s *AccountService) EnableAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account.UserID != userID {
		return ErrAccountNotFound
	}

	if err := s.repo.SetDisabled(ctx, accountID, false); err != nil {
		return err
	}

	account.IsDisabled = false

	if s.connMgr != nil {
		go func() {
			bgCtx := context.Background()
			err := s.connMgr.Connect(bgCtx, account)

			var status string
			var statusMsg string
			if err != nil {
				status = "error"
				statusMsg = err.Error()
			} else {
				status = "connected"
				statusMsg = ""
			}

			if s.eventBus != nil {
				s.eventBus.Publish(accountID.String(), &event.Event{
					Type:      event.EventAccountStatus,
					AccountID: accountID.String(),
					Data: map[string]string{
						"status":  status,
						"message": statusMsg,
					},
				})
			}
		}()
	}

	return nil
}

func (s *AccountService) parseHostPort(hostPort string) (string, int32) {
	parts := strings.Split(hostPort, ":")
	if len(parts) == 2 {
		host := parts[0]
		port, _ := strconv.ParseInt(parts[1], 10, 32)
		return host, int32(port)
	}
	return hostPort, 443
}

func (s *AccountService) logConnection(userID, accountID uuid.UUID, eventType model.ConnectionEventType, status model.ConnectionStatus, message, errorDetail, serverHost string, loginID int64) {
	if s.logService == nil {
		return
	}

	host, port := s.parseHostPort(serverHost)

	log := &model.AccountConnectionLog{
		ID:          uuid.New(),
		UserID:      userID,
		AccountID:   accountID,
		EventType:   eventType,
		Status:      status,
		Message:     message,
		ErrorDetail: errorDetail,
		ServerHost:  host,
		ServerPort:  int(port),
		LoginID:     loginID,
		CreatedAt:   time.Now(),
	}

	if err := s.logService.LogConnection(context.Background(), log); err != nil {
		logger.Warn("Failed to log connection event", zap.Error(err))
	}
}
