package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/mdgateway"
	"alphaforge/internal/mdgateway/adapter/brokersearch"
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/service"
)

// MTConnectionTester validates MT account credentials by connecting to the broker.
type MTConnectionTester interface {
	Test(ctx context.Context, platform, brokerHost, login, password string) (*mdtick.MTAccountInfo, error)
	// VerifyPassword connects to the broker to verify credentials without fetching account info.
	// brokerCompany and accountID are used for §0 host rediscovery on ErrHost failures.
	VerifyPassword(ctx context.Context, platform, brokerHost, login, password, brokerCompany, accountID string) error
}

// SessionReadyWaiter provides an event-driven (channel-based) way to wait for
// an MT session to be established — no polling. Implemented by mthub.Hub.
type SessionReadyWaiter interface {
	WaitSession(accountID string) <-chan struct{}
}

// AccountQuotaChecker checks subscription plan limits for account operations.
// Implemented by service.QuotaChecker.
type AccountQuotaChecker interface {
	CheckAccountLimit(userID uuid.UUID, currentCount int) bool
}

// AccountServer implements ant.v1.AccountServiceHandler.
type AccountServer struct {
	svc           *service.AccountService
	searcher      *brokersearch.Searcher
	publisher     *mdgateway.AccountEventPublisher
	mtTester      MTConnectionTester
	sessionWaiter SessionReadyWaiter                                // may be nil
	stopGateway   func(ctx context.Context, accountID string) error // may be nil
	quotaChecker  AccountQuotaChecker                               // may be nil
	log           *zap.Logger
}

var _ antv1c.AccountServiceHandler = (*AccountServer)(nil)

func NewAccountServer(svc *service.AccountService, searcher *brokersearch.Searcher, publisher *mdgateway.AccountEventPublisher, tester MTConnectionTester, log *zap.Logger) *AccountServer {
	return &AccountServer{svc: svc, searcher: searcher, publisher: publisher, mtTester: tester, log: log}
}

// WithStopGateway sets an optional function to synchronously stop a running gateway.
func (s *AccountServer) WithStopGateway(fn func(ctx context.Context, accountID string) error) *AccountServer {
	s.stopGateway = fn
	return s
}

// WithSessionWaiter sets an optional readiness waiter used by ConnectAccount.
func (s *AccountServer) WithSessionWaiter(w SessionReadyWaiter) *AccountServer {
	s.sessionWaiter = w
	return s
}

// WithQuotaChecker sets the subscription plan quota checker for account slot limits.
func (s *AccountServer) WithQuotaChecker(qc AccountQuotaChecker) *AccountServer {
	s.quotaChecker = qc
	return s
}

// parseUserID extracts and validates the user ID from the request context.
func parseUserID(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id"))
	}
	return id, nil
}

// accountToProto converts a service.AccountDTO to a protobuf Account message.
func accountToProto(a *service.AccountDTO) *antv1.Account {
	var connectedAt *timestamppb.Timestamp
	if a.LastConnectedAt != "" {
		if t, err := time.Parse(time.RFC3339, a.LastConnectedAt); err == nil {
			connectedAt = timestamppb.New(t)
		}
	}
	profit := a.Equity.Sub(a.Balance)
	var profitPercent float64
	if a.Balance.GreaterThan(decimal.Zero) {
		pp, _ := profit.Div(a.Balance).Mul(decimal.NewFromInt(100)).Float64()
		profitPercent = pp
	}
	return &antv1.Account{
		Id: a.ID, UserId: a.UserID, Login: a.Login,
		MtType: a.Platform, BrokerCompany: a.Broker, BrokerServer: a.Server,
		BrokerHost: a.BrokerHost,
		Status:     a.Status,
		Balance:    a.Balance.String(), Credit: a.Credit.String(), Equity: a.Equity.String(), Margin: a.Margin.String(),
		FreeMargin: a.FreeMargin.String(), MarginLevel: a.MarginLevel.String(),
		Leverage: a.Leverage, Currency: a.Currency,
		IsInvestor: a.IsInvestor, LastError: a.LastError,
		IsDisabled:  a.Status == string(service.StatusDisconnected) || a.Status == string(service.StatusFrozen),
		ConnectedAt: connectedAt,
		Profit:      profit.String(), ProfitPercent: profitPercent,
	}
}

func (s *AccountServer) ListAccounts(ctx context.Context, req *connect.Request[antv1.ListAccountsRequest]) (*connect.Response[antv1.ListAccountsResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	accounts, err := s.svc.ListAccounts(ctx, userID)
	if err != nil {
		s.log.Error("ListAccounts", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &antv1.ListAccountsResponse{}
	for _, a := range accounts {
		out.Accounts = append(out.Accounts, accountToProto(&a))
	}
	return connect.NewResponse(out), nil
}

func (s *AccountServer) GetAccount(ctx context.Context, req *connect.Request[antv1.GetAccountRequest]) (*connect.Response[antv1.Account], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	a, err := s.svc.GetAccount(ctx, userID, req.Msg.Id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("account not found"))
		}
		s.log.Error("GetAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(accountToProto(a)), nil
}
