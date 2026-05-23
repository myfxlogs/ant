package connect

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/connection"
	"anttrader/internal/interceptor"
	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/internal/service"
	"anttrader/pkg/logger"
)

type AccountService struct {
	accountRepo   *repository.AccountRepository
	connManager   *connection.ConnectionManager
	brokerService *service.BrokerService
	streamService *StreamService
	// accountSvc 提供 VerifyTradePermission / UpdateTradingPassword 等涉及 MT 协议的操作；
	// 其它只读/状态类 RPC 仍直接走 accountRepo，以保持最小依赖。
	accountSvc *service.AccountService
}

func NewAccountService(
	accountRepo *repository.AccountRepository,
	connManager *connection.ConnectionManager,
	brokerService *service.BrokerService,
	streamService *StreamService,
	accountSvc *service.AccountService,
) *AccountService {
	return &AccountService{
		accountRepo:   accountRepo,
		connManager:   connManager,
		brokerService: brokerService,
		streamService: streamService,
		accountSvc:    accountSvc,
	}
}

func (s *AccountService) ListAccounts(ctx context.Context, req *connect.Request[v1.ListAccountsRequest]) (*connect.Response[v1.ListAccountsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	accounts, err := s.accountRepo.GetByUserID(ctx, uid)
	if err != nil {
		logger.Error("ListAccounts query failed", zap.String("user_id", uid.String()), zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.ListAccountsResponse{
		Accounts: make([]*v1.Account, len(accounts)),
	}

	for i, acc := range accounts {
		response.Accounts[i] = convertMTAccount(acc)
	}

	return connect.NewResponse(response), nil
}

func (s *AccountService) GetAccount(ctx context.Context, req *connect.Request[v1.GetAccountRequest]) (*connect.Response[v1.Account], error) {
	accountID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("errors.account_not_found"))
	}

	userID := interceptor.GetUserID(ctx)
	if account.UserID.String() != userID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("errors.access_denied"))
	}

	return connect.NewResponse(convertMTAccount(account)), nil
}

func (s *AccountService) CreateAccount(ctx context.Context, req *connect.Request[v1.CreateAccountRequest]) (*connect.Response[v1.Account], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	account := &model.MTAccount{
		ID:            uuid.New(),
		UserID:        uid,
		Login:         req.Msg.Login,
		Password:      req.Msg.Password,
		MTType:        req.Msg.MtType,
		BrokerCompany: req.Msg.BrokerCompany,
		BrokerServer:  req.Msg.BrokerServer,
		BrokerHost:    req.Msg.BrokerHost,
		AccountStatus: "connecting",
	}

	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.connManager.Connect(ctx, account); err != nil {
		logger.Error("Failed to connect to MT server during account creation",
			zap.String("login", req.Msg.Login),
			zap.String("mt_type", req.Msg.MtType),
			zap.Error(err))

		s.accountRepo.Delete(ctx, account.ID)

		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("errors.account_connection_failed"))
	}

	// 模式 A：创建时仅做一次连接校验，不保留后台连接/任务
	_ = s.connManager.Disconnect(ctx, account.ID)

	updatedAccount, _ := s.accountRepo.GetByID(ctx, account.ID)
	if updatedAccount != nil {
		account = updatedAccount
	}

	return connect.NewResponse(convertMTAccount(account)), nil
}

func (s *AccountService) UpdateAccount(ctx context.Context, req *connect.Request[v1.UpdateAccountRequest]) (*connect.Response[v1.Account], error) {
	accountID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("errors.account_not_found"))
	}

	userID := interceptor.GetUserID(ctx)
	if account.UserID.String() != userID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("errors.access_denied"))
	}

	wasDisabled := account.IsDisabled

	if req.Msg.BrokerCompany != nil {
		account.BrokerCompany = *req.Msg.BrokerCompany
	}
	if req.Msg.BrokerServer != nil {
		account.BrokerServer = *req.Msg.BrokerServer
	}
	if req.Msg.BrokerHost != nil {
		account.BrokerHost = *req.Msg.BrokerHost
	}

	// 先更新账户基本信息
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// 如果 is_disabled 状态发生变化，单独更新
	if req.Msg.IsDisabled != nil && *req.Msg.IsDisabled != wasDisabled {
		// 更新数据库中的 is_disabled 字段
		if err := s.accountRepo.UpdateDisabled(ctx, accountID, *req.Msg.IsDisabled); err != nil {
			logger.Error("Failed to update is_disabled",
				zap.String("account_id", accountID.String()),
				zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if *req.Msg.IsDisabled {
			// Notify managed SubscribeEvents(all-enabled) streams immediately.
			if s.streamService != nil {
				s.streamService.NotifyAccountEnabledState(userID, accountID.String(), false)
			}
			// Broadcast before tearing down the stream so currently-connected frontends can observe the change.
			if s.streamService != nil {
				s.streamService.BroadcastAccountStatus(accountID.String(), "disconnected", "disabled")
			}
			if s.connManager != nil {
				s.connManager.Disconnect(ctx, accountID)
			}
			// 关闭账户流，停止所有数据订阅
			if s.streamService != nil {
				s.streamService.CloseDisabledAccountStream(accountID)
			}
		} else {
			// 通知管理的 SubscribeEvents(all-enabled) streams
			if s.streamService != nil {
				s.streamService.NotifyAccountEnabledState(userID, accountID.String(), true)
			}
			// Broadcast early so existing subscribers (before reconnect) can see the transition immediately.
			if s.streamService != nil {
				s.streamService.BroadcastAccountStatus(accountID.String(), "connecting", "enabled")
			}
			if s.connManager != nil {
				// 同步连接 MT 服务器（带超时），确保返回时账户已连接（与 ConnectAccount 一致，避免 MT5 过短超时）
				connectCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()

				if err := s.connManager.Connect(connectCtx, account); err != nil {
					logger.Error("Failed to reconnect account",
						zap.String("account_id", accountID.String()),
						zap.Error(err))
					s.accountRepo.UpdateStatus(connectCtx, accountID, "error", err.Error())
					return nil, connect.NewError(connect.CodeInternal, errors.New("errors.account_connection_failed"))
				}

			}
		}
	}

	updatedAccount, _ := s.accountRepo.GetByID(ctx, accountID)
	if updatedAccount != nil {
		account = updatedAccount
	}

	return connect.NewResponse(convertMTAccount(account)), nil
}

func (s *AccountService) DeleteAccount(ctx context.Context, req *connect.Request[v1.DeleteAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	accountID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("errors.account_not_found"))
	}

	userID := interceptor.GetUserID(ctx)
	if account.UserID.String() != userID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("errors.access_denied"))
	}

	// 硬删除：先强制关闭所有流，再断开连接，最后删除 DB
	if s.streamService != nil {
		s.streamService.closeAccountStream(accountID.String(), "deleted")
	}

	if err := s.connManager.Disconnect(ctx, accountID); err != nil {
		logger.Warn("Failed to disconnect MT connection", zap.Error(err))
	}

	if err := s.accountRepo.Delete(ctx, accountID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *AccountService) ConnectAccount(ctx context.Context, req *connect.Request[v1.ConnectAccountRequest]) (*connect.Response[v1.ConnectAccountResponse], error) {
	accountID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("errors.account_not_found"))
	}

	userID := interceptor.GetUserID(ctx)
	if account.UserID.String() != userID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("errors.access_denied"))
	}

	if s.connManager == nil {
		return connect.NewResponse(&v1.ConnectAccountResponse{
			Success: false,
			Message: "errors.account_connection_failed",
		}), nil
	}

	// MT client parity: return success only after MT session is actually up (DB status + streams),
	// otherwise the UI shows "connected" while GetPositions/stream stay empty.
	connectCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := s.connManager.Connect(connectCtx, account); err != nil {
		logger.Warn("ConnectAccount: MT connect failed",
			zap.String("account_id", account.ID.String()),
			zap.String("user_id", userID),
			zap.Error(err))
		return connect.NewResponse(&v1.ConnectAccountResponse{
			Success: false,
			Message: "errors.account_connection_failed",
		}), nil
	}

	if s.streamService != nil {
		if err := s.streamService.ensureAccountStream(ctx, account); err != nil {
			logger.Warn("ConnectAccount: ensureAccountStream",
				zap.String("account_id", account.ID.String()),
				zap.Error(err))
		}
		s.streamService.NotifyAccountEnabledState(userID, account.ID.String(), !account.IsDisabled)
		s.streamService.BroadcastAccountStatus(account.ID.String(), "connected", "manual_connect")
	}

	return connect.NewResponse(&v1.ConnectAccountResponse{
		Success: true,
		Message: "errors.account_connected",
	}), nil
}

func (s *AccountService) DisconnectAccount(ctx context.Context, req *connect.Request[v1.DisconnectAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	accountID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("errors.account_not_found"))
	}

	userID := interceptor.GetUserID(ctx)
	if account.UserID.String() != userID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("errors.access_denied"))
	}

	s.connManager.Disconnect(ctx, account.ID)

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *AccountService) ReconnectAccount(ctx context.Context, req *connect.Request[v1.ReconnectAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	accountID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("errors.account_not_found"))
	}

	userID := interceptor.GetUserID(ctx)
	if account.UserID.String() != userID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("errors.access_denied"))
	}

	s.connManager.Disconnect(ctx, account.ID)

	if s.connManager == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("errors.account_connection_failed"))
	}
	reconnectCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := s.connManager.Connect(reconnectCtx, account); err != nil {
		logger.Warn("ReconnectAccount: MT connect failed",
			zap.String("account_id", account.ID.String()),
			zap.String("user_id", userID),
			zap.Error(err))
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("errors.account_connection_failed"))
	}

	if s.streamService != nil {
		if err := s.streamService.ensureAccountStream(ctx, account); err != nil {
			logger.Warn("ReconnectAccount: ensureAccountStream",
				zap.String("account_id", account.ID.String()),
				zap.Error(err))
		}
		s.streamService.NotifyAccountEnabledState(userID, account.ID.String(), !account.IsDisabled)
		s.streamService.BroadcastAccountStatus(account.ID.String(), "connected", "manual_reconnect")
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *AccountService) SearchBroker(ctx context.Context, req *connect.Request[v1.SearchBrokerRequest]) (*connect.Response[v1.SearchBrokerResponse], error) {
	mtType := req.Msg.MtType
	if mtType == "" {
		mtType = "MT5"
	}
	companies, err := s.brokerService.Search(ctx, mtType, req.Msg.Company)
	if err != nil {
		logger.Error("Failed to search broker", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.SearchBrokerResponse{
		Companies: make([]*v1.BrokerCompany, 0, len(companies)),
	}

	for _, company := range companies {
		bc := &v1.BrokerCompany{
			CompanyName: company.CompanyName,
			Servers:     make([]*v1.BrokerServer, 0, len(company.Results)),
		}
		for _, result := range company.Results {
			bc.Servers = append(bc.Servers, &v1.BrokerServer{
				Name:   result.Name,
				Access: result.Access,
			})
		}
		response.Companies = append(response.Companies, bc)
	}

	return connect.NewResponse(response), nil
}

func convertMTAccount(acc *model.MTAccount) *v1.Account {
	profit := acc.Equity - acc.Balance - acc.Credit
	profitPercent := 0.0
	if acc.Balance > 0 {
		profitPercent = profit / acc.Balance * 100
	}
	account := &v1.Account{
		Id:            acc.ID.String(),
		UserId:        acc.UserID.String(),
		Login:         acc.Login,
		MtType:        acc.MTType,
		BrokerCompany: acc.BrokerCompany,
		BrokerServer:  acc.BrokerServer,
		BrokerHost:    acc.BrokerHost,
		Status:        acc.AccountStatus,
		Token:         acc.MTToken,
		IsDisabled:    acc.IsDisabled,
		LastError:     acc.LastError,
		Balance:       acc.Balance,
		Credit:        acc.Credit,
		Equity:        acc.Equity,
		Margin:        acc.Margin,
		FreeMargin:    acc.FreeMargin,
		MarginLevel:   acc.MarginLevel,
		Leverage:      int32(acc.Leverage),
		Currency:      acc.Currency,
		AccountType:   acc.AccountType,
		AccountMethod: acc.AccountMethod,
		IsInvestor:    acc.IsInvestor,
		Alias:         acc.Alias,
		Profit:        profit,
		ProfitPercent: profitPercent,
		CreatedAt:     timestamppb.New(acc.CreatedAt),
		UpdatedAt:     timestamppb.New(acc.UpdatedAt),
	}
	if acc.LastConnectedAt != nil {
		account.ConnectedAt = timestamppb.New(*acc.LastConnectedAt)
	}
	return account
}
