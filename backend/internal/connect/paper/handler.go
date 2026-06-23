// Package paper provides the ConnectRPC handler for PaperTradingService.
package paper

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/connect/strategy"
	"anttrader/internal/interceptor"
	papereng "anttrader/internal/paper"
	"anttrader/internal/repository"
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
	repo             paperRepo
	engine           *papereng.PaperEngine
	strategyRunner   StrategyRunner
	activeStrategies map[string]context.CancelFunc // paperAccountID → cancel
	mu               sync.Mutex
	log              *zap.Logger
	accountLookup    AccountLookup // provides MT4 account ID for bar data in paper mode
}

var _ antv1c.PaperTradingServiceHandler = (*Handler)(nil)

// NewHandler creates a paper trading ConnectRPC handler.
// accountLookup provides the MT4 account ID for bar data subscription in paper mode.
func NewHandler(repo paperRepo, engine *papereng.PaperEngine, runner StrategyRunner, log *zap.Logger, accountLookup AccountLookup) *Handler {
	if accountLookup == nil {
		accountLookup = func(ctx context.Context, userID string) string { return "" }
	}
	return &Handler{
		repo:             repo,
		engine:           engine,
		strategyRunner:   runner,
		activeStrategies: make(map[string]context.CancelFunc),
		log:              log,
		accountLookup:    accountLookup,
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

// StartPaperStrategy launches a strategy in paper trading mode.
// Spawns a LiveStrategyRunner goroutine that subscribes to bar updates and executes
// the strategy code against a virtual paper account (mode="paper").
func (h *Handler) StartPaperStrategy(ctx context.Context, req *connect.Request[antv1.StartPaperStrategyRequest]) (*connect.Response[antv1.StartPaperStrategyResponse], error) {
	uid := h.userID(ctx)
	if uid == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	accountID := req.Msg.PaperAccountId
	if accountID == "" {
		return connect.NewResponse(&antv1.StartPaperStrategyResponse{
			Success: false,
			Error:   "paper_account_id is required",
		}), nil
	}

	if h.strategyRunner == nil {
		return connect.NewResponse(&antv1.StartPaperStrategyResponse{
			Success: false,
			Error:   "strategy server not configured",
		}), nil
	}

	// Prevent duplicate strategies on the same paper account.
	h.mu.Lock()
	if _, running := h.activeStrategies[accountID]; running {
		h.mu.Unlock()
		return connect.NewResponse(&antv1.StartPaperStrategyResponse{
			Success: false,
			Error:   fmt.Sprintf("strategy already running for paper account %s", accountID),
		}), nil
	}
	h.mu.Unlock()

	runCtx, cancel := context.WithCancel(context.Background())

	cfg := strategy.LiveStrategyConfig{
		AccountID: accountID,
		Symbol:    req.Msg.Symbol,
		Timeframe: req.Msg.Timeframe,
		Code:      req.Msg.StrategyCode,
		Mode:      modeOverride(req.Msg.Params),
		Params:    req.Msg.Params,
	}
		cfg.UserID = uid
		// Live mode: use MT4 account ID for order routing.
		if cfg.Mode == "live" {
			if mt4ID := h.accountLookup(ctx, uid); mt4ID != "" {
				cfg.DataSourceAccountID = mt4ID
				cfg.AccountID = mt4ID // route orders to real MT4 account
			}
		} else {
			// Paper mode: use linked MT4 account for bar data subscription only.
			if mt4ID := h.accountLookup(ctx, uid); mt4ID != "" {
				cfg.DataSourceAccountID = mt4ID
			}
		}

	h.mu.Lock()
	h.activeStrategies[accountID] = cancel
	h.mu.Unlock()

	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.activeStrategies, accountID)
			h.mu.Unlock()
			cancel()
		}()

		h.log.Info("StartPaperStrategy: launching LiveStrategyRunner",
			zap.String("user", uid),
			zap.String("paper_account", accountID),
			zap.String("symbol", req.Msg.Symbol),
			zap.String("timeframe", req.Msg.Timeframe),
		)

		if err := h.strategyRunner.RunLiveStrategy(runCtx, cfg); err != nil {
			h.log.Warn("StartPaperStrategy: LiveStrategyRunner exited with error",
				zap.String("paper_account", accountID),
				zap.Error(err),
			)
		} else {
			h.log.Info("StartPaperStrategy: LiveStrategyRunner exited cleanly",
				zap.String("paper_account", accountID),
			)
		}
	}()

	return connect.NewResponse(&antv1.StartPaperStrategyResponse{Success: true}), nil
}

// StopPaperStrategy stops a running paper strategy by cancelling its context.
func (h *Handler) StopPaperStrategy(ctx context.Context, req *connect.Request[antv1.StopPaperStrategyRequest]) (*connect.Response[antv1.StopPaperStrategyResponse], error) {
	uid := h.userID(ctx)
	if uid == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	_ = uid

	accountID := req.Msg.PaperAccountId

	h.mu.Lock()
	cancel, ok := h.activeStrategies[accountID]
	if ok {
		delete(h.activeStrategies, accountID)
	}
	h.mu.Unlock()

	if !ok {
		return connect.NewResponse(&antv1.StopPaperStrategyResponse{
			Success: false,
			Error:   fmt.Sprintf("no running strategy found for paper account %s", accountID),
		}), nil
	}

	cancel()
	h.log.Info("StopPaperStrategy: cancelled",
		zap.String("paper_account", accountID),
	)

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

// modeOverride extracts the mode from params, defaulting to "paper".
// Set params["mode"] = "live" to run against a real MT account.
func modeOverride(params map[string]string) string {
	if params != nil && params["mode"] == "live" {
		return "live"
	}
	return "paper"
}
