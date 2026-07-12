// Package paper provides the ConnectRPC handler for PaperTradingService.
package paper

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/connect/strategy"
	"alphaforge/internal/interceptor"
	papereng "alphaforge/internal/paper"
	"alphaforge/internal/repository"
)

// StrategyRunner is the paper package's local interface for launching live/paper strategy goroutines.
// Defined here (consumer side) per Go best practice — the strategy package need not know about this interface.
type StrategyRunner interface {
	RunLiveStrategy(ctx context.Context, cfg strategy.LiveStrategyConfig) error
}

// paperRepo is the local interface for paper account persistence.
type paperRepo interface {
	CreateAccount(ctx context.Context, userID, name string, initialBalance decimal.Decimal) (*repository.PaperAccount, error)
	ListAccounts(ctx context.Context, userID string) ([]*repository.PaperAccount, error)
}

// AccountLookup provides the MT account ID for bar data subscription.
type AccountLookup func(ctx context.Context, userID string) string // returns MT4 account ID or ""

// Handler implements ant.v1.PaperTradingServiceHandler.
type Handler struct {
	repo          paperRepo
	engine        *papereng.PaperEngine
	log           *zap.Logger
	accountLookup AccountLookup // provides MT4 account ID for bar data in paper mode
}

var _ antv1c.PaperTradingServiceHandler = (*Handler)(nil)

// NewHandler creates a paper trading ConnectRPC handler.
// accountLookup provides the MT4 account ID for bar data subscription in paper mode.
func NewHandler(repo paperRepo, engine *papereng.PaperEngine, runner StrategyRunner, log *zap.Logger, accountLookup AccountLookup) *Handler {
	if accountLookup == nil {
		accountLookup = func(ctx context.Context, userID string) string { return "" }
	}
	return &Handler{
		repo:          repo,
		engine:        engine,
		log:           log,
		accountLookup: accountLookup,
	}
}

func (h *Handler) userID(ctx context.Context) string {
	return strings.TrimSpace(interceptor.GetUserID(ctx))
}

// CreatePaperAccount creates a new virtual paper trading account.
func (h *Handler) CreatePaperAccount(ctx context.Context, req *connect.Request[antv1.CreatePaperAccountRequest]) (*connect.Response[antv1.PaperAccount], error) {
	uid := h.userID(ctx)
	if uid == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	initialBalance, err := decimal.NewFromString(req.Msg.InitialBalance)
	if err != nil {
		initialBalance = decimal.NewFromInt(10000) // default $10k
	}

	acct, err := h.repo.CreateAccount(ctx, uid, req.Msg.Name, initialBalance)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create paper account: %w", err))
	}

	return connect.NewResponse(paperAccountToProto(acct)), nil
}

// ListPaperAccounts returns all non-archived paper accounts for the authenticated user.
func (h *Handler) ListPaperAccounts(ctx context.Context, req *connect.Request[antv1.ListPaperAccountsRequest]) (*connect.Response[antv1.ListPaperAccountsResponse], error) {
	uid := h.userID(ctx)
	if uid == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	accts, err := h.repo.ListAccounts(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list paper accounts: %w", err))
	}

	pb := make([]*antv1.PaperAccount, len(accts))
	for i, a := range accts {
		pb[i] = paperAccountToProto(a)
	}
	return connect.NewResponse(&antv1.ListPaperAccountsResponse{Accounts: pb}), nil
}

// StartPaperStrategy is deprecated. Use StrategyRuntimeService.StartStrategy instead.
func (h *Handler) StartPaperStrategy(ctx context.Context, req *connect.Request[antv1.StartPaperStrategyRequest]) (*connect.Response[antv1.StartPaperStrategyResponse], error) {
	return connect.NewResponse(&antv1.StartPaperStrategyResponse{
		Success: false,
		Error:   "deprecated: use StrategyRuntimeService.StartStrategy instead",
	}), nil
}

// StopPaperStrategy is deprecated. Use StrategyRuntimeService.StopStrategy instead.
func (h *Handler) StopPaperStrategy(ctx context.Context, req *connect.Request[antv1.StopPaperStrategyRequest]) (*connect.Response[antv1.StopPaperStrategyResponse], error) {
	return connect.NewResponse(&antv1.StopPaperStrategyResponse{
		Success: false,
		Error:   "deprecated: use StrategyRuntimeService.StopStrategy instead",
	}), nil
}

// WatchPaperAccount streams paper account state updates via SSE.
func (h *Handler) WatchPaperAccount(ctx context.Context, req *connect.Request[antv1.WatchPaperAccountRequest], stream *connect.ServerStream[antv1.PaperAccountUpdate]) error {
	uid := h.userID(ctx)
	if uid == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	_ = uid

	ch, unsubscribe := h.engine.Subscribe(req.Msg.PaperAccountId)
	defer unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case acct, ok := <-ch:
			if !ok {
				return nil
			}
			update := &antv1.PaperAccountUpdate{
				PaperAccountId: acct.ID,
				Account:        paperAccountToProto(acct),
			}
			if err := stream.Send(update); err != nil {
				return err
			}
		}
	}
}

// ── Converters ──

func paperAccountToProto(a *repository.PaperAccount) *antv1.PaperAccount {
	return &antv1.PaperAccount{
		Id:               a.ID,
		UserId:           a.UserID,
		Name:             a.Name,
		InitialBalance:   a.InitialBalance.String(),
		CurrentBalance:   a.CurrentBalance.String(),
		Equity:           a.Equity.String(),
		Currency:         a.Currency,
		CreatedAtUnixMs:  a.CreatedAt.UnixMilli(),
		Archived:         a.Archived,
	}
}
