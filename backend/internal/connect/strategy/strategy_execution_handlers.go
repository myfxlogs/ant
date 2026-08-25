package strategy

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/ai"
	"alphaforge/strategy/sdk"
)

func (s *StrategyExecutionServer) Execute(ctx context.Context, req *connect.Request[antv1.ExecuteStrategyRequest]) (*connect.Response[antv1.ExecuteStrategyResponse], error) {
	_, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}

	// GoExecutor removed (Gap 3). Go strategies must be converted to MQL for Bytecode VM.
	if isGoStrategy(req.Msg.Code) {
		return nil, connect.NewError(connect.CodeUnimplemented,
			fmt.Errorf("go strategy execution has been retired — please convert your strategy to MQL"))
	}

	// MQL source requires bar data to produce signals — use StartBacktestRun or ExecuteLive.
	if isMQLStrategy(req.Msg.Code) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("MQL strategies require bar data — use StartBacktestRun or ExecuteLive"))
	}

	return connect.NewResponse(&antv1.ExecuteStrategyResponse{
		Success: false,
		Error:   "strategy execution not available",
	}), nil
}

func (s *StrategyExecutionServer) Validate(ctx context.Context, req *connect.Request[antv1.ValidateStrategyRequest]) (*connect.Response[antv1.ValidateStrategyResponse], error) {
	_, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}

	code := req.Msg.GetCode()
	if code == "" {
		return connect.NewResponse(&antv1.ValidateStrategyResponse{
			Valid:  false,
			Errors: []string{"code is empty"},
		}), nil
	}

	hasSig, missing := ai.HasRequiredSignature(code)
	var errors []string
	if !hasSig {
		errors = missing
	}
	warnings := ai.StructuralWarnings(code)

	return connect.NewResponse(&antv1.ValidateStrategyResponse{
		Valid:    len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
	}), nil
}

func (s *StrategyExecutionServer) Backtest(ctx context.Context, req *connect.Request[antv1.BacktestStrategyRequest]) (*connect.Response[antv1.BacktestStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}

	// Synchronous backtest is deprecated.
	// Use StartBacktestRun for async execution via the Go-native backtest engine.
	s.log.Debug("Backtest: deprecated sync endpoint called", zap.String("userID", uid.String()))
	return connect.NewResponse(&antv1.BacktestStrategyResponse{
		Success: false,
		Error:   "use StartBacktestRun for async backtesting via the Go-native engine",
	}), nil
}

func (s *StrategyExecutionServer) GetTemplates(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetStrategyTemplatesResponse], error) {
	return connect.NewResponse(&antv1.GetStrategyTemplatesResponse{Templates: builtinTemplates()}), nil
}

// ExecuteLive runs strategy code against a live bar stream.
// Go-native path retired (GoExecutor removed); MQL path uses in-process Bytecode VM.
func (s *StrategyExecutionServer) ExecuteLive(ctx context.Context, req *connect.Request[antv1.ExecuteLiveRequest]) (*connect.Response[antv1.ExecuteLiveResponse], error) {
	if _, err := userIDRequire(ctx); err != nil {
		return nil, err
	}

	// D-VM-LIVE-001-P1: reject live mode before compilation to close
	// client-supplied account truth attack surface.
	if err := validateExecuteLiveRequestMode(req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Go-native compilation path: generated Go strategy via go run.
	// GoExecutor removed (Gap 3). Go strategies must be converted to MQL for Bytecode VM.
	if isGoStrategy(req.Msg.StrategyCode) {
		return nil, connect.NewError(connect.CodeUnimplemented,
			fmt.Errorf("go strategy live execution has been retired — please convert your strategy to MQL"))
	}

	// MQL path: in-process Bytecode VM execution.
	if isMQLStrategy(req.Msg.StrategyCode) {
		resp, err := s.executeVMLive(ctx, req.Msg)
		if err != nil {
			s.log.Warn("vm live execution failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("live execution failed: %w", err))
		}
		return connect.NewResponse(resp), nil
	}

	// Python path: in-process Bytecode VM execution (same VM, different compiler).
	if sdk.IsPython(req.Msg.StrategyCode) {
		resp, err := s.executePythonVMLive(ctx, req.Msg)
		if err != nil {
			s.log.Warn("vm python live execution failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("python live execution failed: %w", err))
		}
		return connect.NewResponse(resp), nil
	}

	return connect.NewResponse(&antv1.ExecuteLiveResponse{Success: false, Error: "live execution not available"}), nil
}

// toCamelCase converts a filename like "my_strategy" to "MyStrategy".
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(p[1:])
		}
	}
	return b.String()
}

func isGoStrategy(code string) bool {
	return sdk.IsGo(code)
}

// isMQLStrategy returns true if the code looks like MQL source (not Go, not Python).
func isMQLStrategy(code string) bool {
	return sdk.IsMQL(code)
}
