// Package paper provides the ConnectRPC handler for PaperTradingService.
package paper

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	papereng "anttrader/internal/paper"
	"anttrader/internal/repository"
)

// Handler implements ant.v1.PaperTradingServiceHandler.
type Handler struct {
	repo   *repository.PaperRepo
	engine *papereng.PaperEngine
	log    *zap.Logger
}

var _ antv1c.PaperTradingServiceHandler = (*Handler)(nil)

// NewHandler creates a paper trading ConnectRPC handler.
func NewHandler(repo *repository.PaperRepo, engine *papereng.PaperEngine, log *zap.Logger) *Handler {
	return &Handler{repo: repo, engine: engine, log: log}
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

// StartPaperStrategy launches a strategy in paper trading mode.
func (h *Handler) StartPaperStrategy(ctx context.Context, req *connect.Request[antv1.StartPaperStrategyRequest]) (*connect.Response[antv1.StartPaperStrategyResponse], error) {
	uid := h.userID(ctx)
	if uid == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	_ = uid

	// TODO: spawn LiveStrategyRunner goroutine with mode="paper"
	// cfg := LiveStrategyConfig{
	//     AccountID: req.Msg.PaperAccountId,
	//     Symbol:    req.Msg.Symbol,
	//     Timeframe: req.Msg.Timeframe,
	//     Code:      req.Msg.StrategyCode,
	//     Mode:      "paper",
	//     Params:    req.Msg.Params,
	// }
	// go strategyServer.RunLiveStrategy(ctx, cfg)

	h.log.Info("StartPaperStrategy requested (LiveStrategyRunner integration pending)",
		zap.String("paper_account", req.Msg.PaperAccountId),
		zap.String("symbol", req.Msg.Symbol),
	)

	return connect.NewResponse(&antv1.StartPaperStrategyResponse{Success: true}), nil
}

// StopPaperStrategy stops a running paper strategy.
func (h *Handler) StopPaperStrategy(ctx context.Context, req *connect.Request[antv1.StopPaperStrategyRequest]) (*connect.Response[antv1.StopPaperStrategyResponse], error) {
	uid := h.userID(ctx)
	if uid == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	_ = uid

	return connect.NewResponse(&antv1.StopPaperStrategyResponse{Success: true}), nil
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
