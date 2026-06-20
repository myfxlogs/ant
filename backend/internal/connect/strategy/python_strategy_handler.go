package strategy

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	"connectrpc.com/connect"
	"github.com/shopspring/decimal"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/mthub"
	"anttrader/internal/notification"
	"anttrader/internal/repository"
	"anttrader/internal/pglisten"
)

// PythonStrategyServer implements ant.v1.PythonStrategyServiceHandler.
type PythonStrategyServer struct {
	backtestRepo        *repository.BacktestRunRepository
	log                 *zap.Logger
	connectClient       antv1c.PythonStrategyServiceClient // ConnectRPC to Python service
	backtestClient      antv1c.BacktestServiceClient
	marketDataRepo      repository.MarketDataStore
	barSource           BarSource // unified bar data source (backtest or live)
	mtHub               *mthub.MtHubService // for live order submission
	paperEngine         PaperOrderExecutor  // for paper trading simulated fills
	pgListen            *pglisten.Listener
	notifSender         *notification.Sender
	onBacktestComplete  func(ctx context.Context, run *repository.BacktestRun) // auto-gate hook
}

func (s *PythonStrategyServer) SetMarketDataRepo(r repository.MarketDataStore) {
	s.marketDataRepo = r
}

// PaperOrderExecutor abstracts paper trading order simulation.
// Implemented by paper.PaperEngine to avoid circular imports.
type PaperOrderExecutor interface {
	PlacePaperOrder(ctx context.Context, accountID, symbol, side string,
		volume decimal.Decimal, bid, ask float64) error
}

func (s *PythonStrategyServer) SetBarSource(bs BarSource)                 { s.barSource = bs }
func (s *PythonStrategyServer) SetMtHub(h *mthub.MtHubService)            { s.mtHub = h }
func (s *PythonStrategyServer) SetPaperEngine(pe PaperOrderExecutor)      { s.paperEngine = pe }

var _ antv1c.PythonStrategyServiceHandler = (*PythonStrategyServer)(nil)

func NewPythonStrategyServer(backtestRepo *repository.BacktestRunRepository, log *zap.Logger) *PythonStrategyServer {
	return &PythonStrategyServer{backtestRepo: backtestRepo, log: log}
}

func (s *PythonStrategyServer) SetConnectClient(c antv1c.PythonStrategyServiceClient) { s.connectClient = c }
func (s *PythonStrategyServer) SetBacktestClient(c antv1c.BacktestServiceClient)     { s.backtestClient = c }
func (s *PythonStrategyServer) SetNotificationSender(ns *notification.Sender)         { s.notifSender = ns }
func (s *PythonStrategyServer) SetOnBacktestComplete(fn func(context.Context, *repository.BacktestRun)) { s.onBacktestComplete = fn }

// userIDRequire extracts and validates the authenticated user ID from context.
func userIDRequire(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	return id, nil
}

func (s *PythonStrategyServer) Execute(ctx context.Context, req *connect.Request[antv1.ExecuteStrategyRequest]) (*connect.Response[antv1.ExecuteStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	_ = uid // authorization verified; downstream uses AccountId directly

	if s.connectClient != nil {
		resp, err := s.connectClient.Execute(ctx, connect.NewRequest(&antv1.ExecuteStrategyRequest{
			Code:      req.Msg.Code,
			AccountId: req.Msg.AccountId,
			Symbol:    req.Msg.Symbol,
			Timeframe: req.Msg.Timeframe,
		}))
		if err != nil {
			s.log.Warn("python execute failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(resp.Msg), nil
	}
	return connect.NewResponse(&antv1.ExecuteStrategyResponse{
		Success: false,
	}), nil
}

func (s *PythonStrategyServer) Validate(ctx context.Context, req *connect.Request[antv1.ValidateStrategyRequest]) (*connect.Response[antv1.ValidateStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	_ = uid

	if s.connectClient != nil {
		resp, err := s.connectClient.Validate(ctx, connect.NewRequest(&antv1.ValidateStrategyRequest{
			Code: req.Msg.Code,
		}))
		if err != nil {
			s.log.Warn("python validate failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(resp.Msg), nil
	}
	return connect.NewResponse(&antv1.ValidateStrategyResponse{
		Valid: true,
	}), nil
}

func (s *PythonStrategyServer) Backtest(ctx context.Context, req *connect.Request[antv1.BacktestStrategyRequest]) (*connect.Response[antv1.BacktestStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	_ = uid
	if s.backtestClient == nil {
		return connect.NewResponse(&antv1.BacktestStrategyResponse{Success: false}), nil
	}
	klines, _ := fetchKlines(ctx, s.marketDataRepo, req.Msg.Symbol, req.Msg.Timeframe)
	resp, err := s.backtestClient.RunBacktest(ctx, connect.NewRequest(
		&antv1.ExecuteBacktestRequest{
			StrategyCode:   req.Msg.Code,
			Symbol:         req.Msg.Symbol,
			Timeframe:      req.Msg.Timeframe,
			InitialCapital: 10000,
			Klines:         klines,
		}))
	if err != nil {
		s.log.Warn("python backtest failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !resp.Msg.GetSuccess() {
		errMsg := resp.Msg.GetError()
		if errMsg == "" {
			errMsg = "backtest failed"
		}
		return connect.NewResponse(&antv1.BacktestStrategyResponse{Success: false, Error: errMsg}), nil
	}
	met := resp.Msg.GetMetrics()
	return connect.NewResponse(&antv1.BacktestStrategyResponse{
		Success:     true,
		EquityCurve: resp.Msg.GetEquityCurve(),
		Metrics:     mapMetrics(met),
	}), nil
}

// mapMetrics converts ExecuteBacktestMetrics → BacktestMetrics.
func mapMetrics(met *antv1.ExecuteBacktestMetrics) *antv1.BacktestMetrics {
	if met == nil {
		return &antv1.BacktestMetrics{}
	}
	return &antv1.BacktestMetrics{
		TotalReturn: met.GetTotalReturn(), AnnualReturn: met.GetAnnualReturn(),
		MaxDrawdown: met.GetMaxDrawdown(), SharpeRatio: met.GetSharpeRatio(),
		WinRate: met.GetWinRate(), ProfitFactor: met.GetProfitFactor(),
		TotalTrades: met.GetTotalTrades(), WinningTrades: met.GetWinningTrades(),
		LosingTrades: met.GetLosingTrades(), AverageProfit: met.GetAverageProfit(),
		AverageLoss: met.GetAverageLoss(),
	}
}


func (s *PythonStrategyServer) GetTemplates(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetPythonTemplatesResponse], error) {
	return connect.NewResponse(&antv1.GetPythonTemplatesResponse{Templates: builtinTemplates()}), nil
}

// ExecuteLive runs strategy code against a proto-native LiveStrategyContext (live/paper mode).
// Delegates to the Python strategy service which maintains a LiveWorker pool for sub-100ms per-bar calls.
func (s *PythonStrategyServer) ExecuteLive(ctx context.Context, req *connect.Request[antv1.ExecuteLiveRequest]) (*connect.Response[antv1.ExecuteLiveResponse], error) {
	if _, err := userIDRequire(ctx); err != nil {
		return nil, err
	}
	if s.connectClient != nil {
		resp, err := s.connectClient.ExecuteLive(ctx, connect.NewRequest(req.Msg))
		if err != nil {
			s.log.Warn("python ExecuteLive failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(resp.Msg), nil
	}
	return connect.NewResponse(&antv1.ExecuteLiveResponse{Success: false, Error: "Python service not configured"}), nil
}

func (s *PythonStrategyServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
