package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
	"anttrader/internal/service/systemai"
	"anttrader/strategy/backtest"
	"anttrader/strategy/sdk"
	"anttrader/tools/mql2go"
)

// GatewayServer implements ant.v1.AgentGatewayServiceHandler.
// Synchronous strategy submission → compile → backtest → LLM analysis.
type GatewayServer struct {
	marketDataRepo repository.MarketDataStore
	aiSvc         *systemai.Service
	log           *zap.Logger
	bridge         *Bridge    // reuse bridge instance
	profiler       *Profiler  // reuse profiler instance
	interpreter    *Interpreter // reuse interpreter instance
}

var _ antv1c.AgentGatewayServiceHandler = (*GatewayServer)(nil)

// NewGatewayServer creates the Agent Gateway ConnectRPC handler.
func NewGatewayServer(_ *pgxpool.Pool, mdr repository.MarketDataStore, aiSvc *systemai.Service, log *zap.Logger) *GatewayServer {
	cache := NewLLCache(30 * time.Minute)
	return &GatewayServer{
		marketDataRepo: mdr,
		aiSvc:          aiSvc,
		log:            log,
		bridge:         NewBridge(aiSvc, log, cache),
		profiler:       NewProfiler(aiSvc, log, cache),
		interpreter:    NewInterpreter(aiSvc, log, cache),
	}
}

// SubmitStrategy handles Agent strategy submission: compile MQL → VM backtest → profile + analysis.
// Synchronous mode only — blocks until backtest completes (<30s).
func (s *GatewayServer) SubmitStrategy(
	ctx context.Context,
	req *connect.Request[antv1.SubmitStrategyRequest],
) (*connect.Response[antv1.SubmitStrategyResponse], error) {
	msg := req.Msg
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}

	// SYNC only — async mode is Phase 3.
	if msg.Mode == antv1.SubmitMode_SUBMIT_ASYNC {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("async mode not yet supported"))
	}

	sourceCode := msg.SourceCode
	if len(sourceCode) > mql2go.MaxSourceSize {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("source code exceeds max size %d bytes", mql2go.MaxSourceSize))
	}
	if sourceCode == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("source_code is required"))
	}

	strategyID := uuid.New().String()

	// ── Step 1: Compile with coverage (language dispatch) ──
	language := msg.Language
	if language == "" {
		language = "mql4" // default
	}

	var runner *mql2go.VMRunner
	var coverage *mql2go.CoverageResult
	var err error

	switch language {
	case "python":
		runner, coverage, err = mql2go.CompilePythonWithCoverage(sourceCode)
	default:
		runner, coverage, err = mql2go.CompileMQLWithCoverage(sourceCode)
	}

	if err != nil {
		s.log.Warn("AgentGateway: compile failed", zap.String("strategyID", strategyID), zap.String("language", language), zap.Error(err))
		return connect.NewResponse(&antv1.SubmitStrategyResponse{
			StrategyId:    strategyID,
			CompileSuccess: false,
			CompileError:  err.Error(),
			Mode:          antv1.SubmitMode_SUBMIT_SYNC,
		}), nil
	}

	// ── Step 2: Fetch market data bars ──
	btCfg := msg.BacktestConfig
	if btCfg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("backtest_config is required"))
	}
	bars, err := s.fetchBars(ctx, btCfg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch market data: %w", err))
	}
	if len(bars) < 2 {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("insufficient market data: %d bars (need ≥2)", len(bars)))
	}

	// ── Step 3: Run backtest via VM + SimBroker ──
	btResult, btErr := s.runBacktest(ctx, runner, btCfg, bars, msg.Params)

	// Convert to proto for response + LLM analysis.
	var btProto *antv1.AgentBacktestResult
	if btErr != nil {
		btProto = &antv1.AgentBacktestResult{Success: false, Error: btErr.Error()}
	} else {
		btProto = buildBacktestResultProto(btResult)
	}

	// ── Step 4: Generate strategy profile (LLM injection point [1]) ──
	profile, profErr := s.profiler.GenerateProfile(ctx, userID, sourceCode, coverage)
	if profErr != nil {
		s.log.Warn("AgentGateway: profile generation failed", zap.Error(profErr))
	}

	// ── Step 5: Generate backtest analysis (LLM injection point [4]) ──
	var analysis *antv1.BacktestAnalysis
	if btErr == nil {
		analysis, err = s.interpreter.AnalyzeBacktest(ctx, userID, btProto, profile)
		if err != nil {
			s.log.Warn("AgentGateway: analysis generation failed", zap.Error(err))
		}
	}

	// ── Step 6: Blind-spot bridge (ADR-0024) ──
	// When MQL has blind spots (coverage < 1.0), trigger LLM translation to Python subset.
	// Retry loop: LLM translate → compile → coverage check → backtest → error feedback → retry (max 3).
	var semanticDiff *antv1.SemanticDiff
	bridgeStatus := "not_attempted"
	var bridgedPython string
	var bridgeCompileErr string

	if language != "python" && coverage.Score < 1.0 && len(coverage.BlindSpots) > 0 {
		validateBacktest := func(pyRunner *mql2go.VMRunner) error {
			_, btErr := s.runBacktest(ctx, pyRunner, btCfg, bars, msg.Params)
			return btErr
		}

		bridgeResult, bridgeErr := s.bridge.TranslateWithRetry(ctx, userID, sourceCode, coverage, profile, validateBacktest)
		if bridgeErr != nil {
			s.log.Warn("AgentGateway: bridge translation failed", zap.Error(bridgeErr))
			bridgeStatus = "bridge_failed"
			bridgeCompileErr = bridgeErr.Error()
			semanticDiff = buildBridgeFailureReport(coverage)
		} else {
			bridgeStatus = bridgeResult.Status
			bridgeCompileErr = bridgeResult.CompileError
			if bridgeResult.BacktestError != "" {
				bridgeCompileErr = bridgeResult.BacktestError
			}
			if bridgeResult.Status == "success" {
				bridgedPython = bridgeResult.PythonSource
				pyCov := bridgeResult.BridgedCov
				s.log.Info("AgentGateway: bridge translation successful",
					zap.Int("attempts", bridgeResult.Attempts),
					zap.Float64("original_coverage", coverage.Score),
					zap.Float64("bridged_coverage", pyCov.Score))
				if pyCov != nil {
					semanticDiff = &antv1.SemanticDiff{
						Changes:      buildBridgeChanges(coverage, pyCov),
						EffectSummary: fmt.Sprintf("覆盖率 %.0f%% → %.0f%%", coverage.Score*100, pyCov.Score*100),
					}
				}
			} else {
				// Degradation path — generate blind spot report + MT hosted suggestion
				s.log.Warn("AgentGateway: bridge failed after retries",
					zap.Int("attempts", bridgeResult.Attempts),
					zap.String("last_error", truncate(bridgeCompileErr, 200)))
				semanticDiff = buildBridgeFailureReport(coverage)
			}
		}
	}

	// ── Build response ──
	resp := &antv1.SubmitStrategyResponse{
		StrategyId:          strategyID,
		CompileSuccess:      true,
		Mode:                antv1.SubmitMode_SUBMIT_SYNC,
		Profile:             profile,
		Analysis:            analysis,
		CoverageScore:       coverage.Score,
		Result:              btProto,
		SemanticDiff:        semanticDiff,
		BridgeStatus:        bridgeStatus,
		BridgedPythonSource: bridgedPython,
		BridgeCompileError:  bridgeCompileErr,
	}
	for _, bs := range coverage.BlindSpots {
		resp.BlindSpots = append(resp.BlindSpots, &antv1.AgentBlindSpot{
			Builtin:  bs.Builtin,
			Severity: bs.Severity,
			Count:    int32(bs.Count),
		})
	}

	return connect.NewResponse(resp), nil
}

// SearchExperience searches the knowledge base for similar experiences.
// Stub — pgvector schema not yet created. Returns empty.
func (s *GatewayServer) SearchExperience(
	ctx context.Context,
	req *connect.Request[antv1.SearchExperienceRequest],
) (*connect.Response[antv1.SearchExperienceResponse], error) {
	// knowledge base not yet implemented.
	// Phase 4 will implement pgvector search with tenant_id isolation.
	return connect.NewResponse(&antv1.SearchExperienceResponse{}), nil
}

// StoreExperience stores an experience to the knowledge base.
// Stub — pgvector schema not yet created.
func (s *GatewayServer) StoreExperience(
	ctx context.Context,
	req *connect.Request[antv1.StoreExperienceRequest],
) (*connect.Response[antv1.StoreExperienceResponse], error) {
	// knowledge base not yet implemented.
	return connect.NewResponse(&antv1.StoreExperienceResponse{
		Id:     uuid.New().String(),
		Success: false,
	}), nil
}

// ── Internal helpers ──────────────────────────────────────────────────

func (s *GatewayServer) fetchBars(ctx context.Context, cfg *antv1.AgentBacktestConfig) ([]sdk.Bar, error) {
	if s.marketDataRepo == nil {
		return nil, fmt.Errorf("market data store not configured")
	}
	var from, to *time.Time
	if cfg.StartDateMs > 0 {
		t := time.UnixMilli(cfg.StartDateMs)
		from = &t
	}
	if cfg.EndDateMs > 0 {
		t := time.UnixMilli(cfg.EndDateMs)
		to = &t
	}
	chBars, err := s.marketDataRepo.GetKlines(ctx, cfg.Symbol, "", cfg.Timeframe, from, to, 100000)
	if err != nil {
		return nil, err
	}
	// Convert to sdk.Bar (reverse to chronological order)
	bars := make([]sdk.Bar, 0, len(chBars))
	for i := len(chBars) - 1; i >= 0; i-- {
		b := chBars[i]
		bars = append(bars, sdk.Bar{
			Open:      b.Open,
			High:      b.High,
			Low:       b.Low,
			Close:     b.Close,
			Volume:    int64(b.Volume),
			Timestamp: int64(b.OpenTsUnixMs),
		})
	}
	return bars, nil
}

func (s *GatewayServer) runBacktest(
	ctx context.Context,
	runner *mql2go.VMRunner,
	cfg *antv1.AgentBacktestConfig,
	bars []sdk.Bar,
	params map[string]string,
) (*backtest.Result, error) {
	initialCapital := parseDecimalDefault(cfg.InitialCapital, "10000")
	leverage := int32(100)
	if cfg.Leverage != "" {
		if d, err := decimal.NewFromString(cfg.Leverage); err == nil {
			leverage = int32(d.IntPart())
		}
	}
	commission := parseDecimalDefault(cfg.Commission, "0.0003")
	slippage := parseDecimalDefault(cfg.Slippage, "0.00001")

	btConfig := backtest.Config{
		Symbol:         cfg.Symbol,
		Timeframe:      cfg.Timeframe,
		InitialCapital: initialCapital,
		Leverage:       leverage,
		Commission:     commission,
		Slippage:       slippage,
		SwapRate:       decimal.NewFromFloat(0.00001),
		StrictMode:     cfg.StrictMode,
		Params:         params,
	}

	// Derive symbol info from bars when broker SymbolInfo is unavailable.
	// REUSE: backtest.DeriveSymbolInfoFromBars (shared with executeVMBacktest).
	backtest.DeriveSymbolInfoFromBars(&btConfig, bars)

	// ADR-0024 §5.3: 30s VM timeout — prevent runaway backtests in Agent loops.
	vmCtx, vmCancel := context.WithTimeout(ctx, 30*time.Second)
	defer vmCancel()

	engine := backtest.New(btConfig, runner, bars)
	result, err := engine.Run(vmCtx)
	if err != nil {
		return nil, fmt.Errorf("backtest engine: %w", err)
	}
	return result, nil
}

func buildBacktestResultProto(r *backtest.Result) *antv1.AgentBacktestResult {
	resp := &antv1.AgentBacktestResult{
		Success: true,
	}
	if r.Metrics != nil {
		resp.TotalReturn = r.Metrics.TotalReturn
		resp.AnnualReturn = r.Metrics.AnnualReturn
		resp.MaxDrawdown = r.Metrics.MaxDrawdown
		resp.SharpeRatio = r.Metrics.SharpeRatio
		resp.WinRate = r.Metrics.WinRate
		resp.ProfitFactor = r.Metrics.ProfitFactor
		resp.TotalTrades = r.Metrics.TotalTrades
		resp.WinningTrades = r.Metrics.WinningTrades
		resp.LosingTrades = r.Metrics.LosingTrades
		totalPnl := r.Config.InitialCapital.Mul(decimal.NewFromFloat(r.Metrics.TotalReturn))
		resp.TotalPnlAbsolute = totalPnl.String()
	}
	for _, ep := range r.Equity {
		resp.EquityCurve = append(resp.EquityCurve, ep.Equity.String())
		resp.EquityTimesMs = append(resp.EquityTimesMs, ep.Time.UnixMilli())
	}
	for i, t := range r.Trades {
		side := "BUY"
		if t.Side == sdk.SideSell {
			side = "SELL"
		}
		resp.Trades = append(resp.Trades, &antv1.AgentTrade{
			Ticket:     int64(i + 1),
			Side:       side,
			Volume:     t.Volume.String(),
			OpenTsMs:   t.EntryTime.UnixMilli(),
			OpenPrice:  t.EntryPrice.String(),
			CloseTsMs:  t.ExitTime.UnixMilli(),
			ClosePrice: t.ExitPrice.String(),
			Pnl:        t.Profit.String(),
			Commission: t.Commission.String(),
			Reason:     t.Comment,
		})
	}
	return resp
}
