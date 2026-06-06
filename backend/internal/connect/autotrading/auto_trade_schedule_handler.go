package autotrading

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
)

// Schedule RPCs are implemented by StrategyService (strategy.proto).
// These stubs return Unimplemented to avoid duplicate write paths to the
// same strategy_schedules table.

func (s *AutoTradingServer) CreateSchedule(
	_ context.Context, _ *connect.Request[antv1.CreateScheduleRequest],
) (*connect.Response[antv1.StrategySchedule], error) {
	return nil, connect.NewError(connect.CodeUnimplemented,
		fmt.Errorf("use StrategyService.CreateSchedule instead"))
}

func (s *AutoTradingServer) GetSchedule(
	_ context.Context, _ *connect.Request[antv1.GetScheduleRequest],
) (*connect.Response[antv1.StrategySchedule], error) {
	return nil, connect.NewError(connect.CodeUnimplemented,
		fmt.Errorf("use StrategyService.GetSchedule instead"))
}

func (s *AutoTradingServer) UpdateSchedule(
	_ context.Context, _ *connect.Request[antv1.UpdateScheduleRequest],
) (*connect.Response[antv1.StrategySchedule], error) {
	return nil, connect.NewError(connect.CodeUnimplemented,
		fmt.Errorf("use StrategyService.UpdateSchedule instead"))
}

func (s *AutoTradingServer) DeleteSchedule(
	_ context.Context, _ *connect.Request[antv1.DeleteScheduleRequest],
) (*connect.Response[emptypb.Empty], error) {
	return nil, connect.NewError(connect.CodeUnimplemented,
		fmt.Errorf("use StrategyService.DeleteSchedule instead"))
}

func (s *AutoTradingServer) ToggleSchedule(
	_ context.Context, _ *connect.Request[antv1.ToggleScheduleRequest],
) (*connect.Response[antv1.StrategySchedule], error) {
	return nil, connect.NewError(connect.CodeUnimplemented,
		fmt.Errorf("use StrategyService.ToggleSchedule instead"))
}
