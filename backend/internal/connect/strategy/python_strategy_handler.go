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
	"anttrader/internal/risk"
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

	// D6-A: Gate is the single authoritative risk evaluator.  Injected at
	// construction — if nil, live order dispatch panics (fail-stop, non-bypassable).
	gate *risk.Gate

	// D6-A / T3.2b: AccountStateProvider feeds real account data.
	// Before connected, equity-dependent gate rules fail-closed.
	accountProvider AccountStateProvider

	// Go-native strategy executor — replaces Python for generated Go strategies.
	goExecutor *GoExecutor
}

// AccountStateProvider supplies live account state for gate evaluation (T3.2b).
// Implemented by the MT gateway account status subscription.
type AccountStateProvider interface {
	GetAccountState(ctx context.Context, accountID string) (*risk.AccountState, error)
}

func (s *PythonStrategyServer) SetMarketDataRepo(r repository.MarketDataStore) {
	s.marketDataRepo = r
}

// PaperOrderExecutor abstracts paper trading order simulation.
// Implemented by paper.PaperEngine to avoid circular imports.
type PaperOrderExecutor interface {
	PlacePaperOrder(ctx context.Context, accountID, symbol, side string,
		volume decimal.Decimal, bid, ask float64) error
	ClosePaperOrder(ctx context.Context, accountID, symbol string) error
	ModifyPaperOrder(ctx context.Context, accountID, symbol string, sl, tp decimal.Decimal) error
	CancelPaperOrder(ctx context.Context, accountID, symbol string) error
}

func (s *PythonStrategyServer) SetBarSource(bs BarSource)                 { s.barSource = bs }
func (s *PythonStrategyServer) SetMtHub(h *mthub.MtHubService)            { s.mtHub = h }
func (s *PythonStrategyServer) SetPaperEngine(pe PaperOrderExecutor)      { s.paperEngine = pe }

// SetGate injects the risk gate (D6-A: mandatory, non-optional).
// Must be called before RunLiveStrategy.
func (s *PythonStrategyServer) SetGate(g *risk.Gate) { s.gate = g }

// AddGateRule adds a rule to the Gate after initialization.
func (s *PythonStrategyServer) AddGateRule(r risk.Rule) {
	if s.gate != nil {
		s.gate.AddRule(r)
	}
}

// SetAccountProvider injects the live account state provider (T3.2b).
func (s *PythonStrategyServer) SetAccountProvider(p AccountStateProvider) { s.accountProvider = p }

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
	_ = uid

	// Go-native path: execute generated Go strategies directly.
	if s.goExecutor != nil && isGoStrategy(req.Msg.Code) {
		execReq := ExecuteRequest{
			Symbol:    req.Msg.Symbol,
			Timeframe: req.Msg.Timeframe,
		}
		resp, err := s.goExecutor.Run(ctx, req.Msg.Code, execReq)
		if err != nil {
			s.log.Warn("go executor failed, falling back to python", zap.Error(err))
		} else {
			return connect.NewResponse(&antv1.ExecuteStrategyResponse{
				Success: true,
				Signal:  toProtoSignal(resp),
			}), nil
		}
	}

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

func (s *PythonStrategyServer) TranspileCode(ctx context.Context, req *connect.Request[antv1.TranspileCodeRequest]) (*connect.Response[antv1.TranspileCodeResponse], error) {
	// Python strategy service removed — MQL→Go migration is handled by tools/mql2go.
	// TranspileCode is deprecated; use StrategyImportService instead.
	return connect.NewResponse(&antv1.TranspileCodeResponse{
		IsDeterministic: false,
		GapSamples:       []string{"DEPRECATED: use StrategyImport for MQL→Go migration"},
		Confidence:       0,
	}), nil
}

func (s *PythonStrategyServer) AnalyzeImportCode(ctx context.Context, req *connect.Request[antv1.AnalyzeImportCodeRequest]) (*connect.Response[antv1.AnalyzeImportCodeResponse], error) {
	if s.connectClient != nil {
		return s.connectClient.AnalyzeImportCode(ctx, req)
	}
	return connect.NewResponse(&antv1.AnalyzeImportCodeResponse{}), nil
}

func (s *PythonStrategyServer) GenerateImportCode(ctx context.Context, req *connect.Request[antv1.GenerateImportCodeRequest]) (*connect.Response[antv1.GenerateImportCodeResponse], error) {
	if s.connectClient != nil {
		return s.connectClient.GenerateImportCode(ctx, req)
	}
	return connect.NewResponse(&antv1.GenerateImportCodeResponse{}), nil
}

func (s *PythonStrategyServer) ImportStrategy(ctx context.Context, req *connect.Request[antv1.ImportStrategyRequest]) (*connect.Response[antv1.ImportStrategyResponse], error) {
	if s.connectClient != nil {
		return s.connectClient.ImportStrategy(ctx, req)
	}
	return connect.NewResponse(&antv1.ImportStrategyResponse{}), nil
}

func (s *PythonStrategyServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }

func isGoStrategy(code string) bool {
	return len(code) > 0 && (contains(code, "anttrader/strategy/sdk") || contains(code, "package "))
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func toProtoSignal(resp *ExecuteResponse) *antv1.StrategySignal {
	return &antv1.StrategySignal{
		SignalType: resp.Signal,
		Volume:     resp.Volume,
		Price:      resp.Price,
		StopLoss:   resp.StopLoss,
		TakeProfit: resp.TakeProfit,
		Reason:     resp.Comment,
	}
}
