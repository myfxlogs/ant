package connect

import (
	"context"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
)

// LogServiceServer implements ant.v1.LogServiceHandler.
type LogServiceServer struct{ log *zap.Logger }

var _ antv1c.LogServiceHandler = (*LogServiceServer)(nil)

func NewLogServiceServer(log *zap.Logger) *LogServiceServer {
	return &LogServiceServer{log: log}
}

func (s *LogServiceServer) GetConnectionLogs(ctx context.Context, req *connect.Request[antv1.GetConnectionLogsRequest]) (*connect.Response[antv1.GetConnectionLogsResponse], error) {
	return connect.NewResponse(&antv1.GetConnectionLogsResponse{
		Logs:  []*antv1.ConnectionLog{},
		Total: 0,
	}), nil
}

func (s *LogServiceServer) GetExecutionLogs(ctx context.Context, req *connect.Request[antv1.GetExecutionLogsRequest]) (*connect.Response[antv1.GetExecutionLogsResponse], error) {
	return connect.NewResponse(&antv1.GetExecutionLogsResponse{
		Logs:  []*antv1.ExecutionLog{},
		Total: 0,
	}), nil
}

func (s *LogServiceServer) GetOrderLogHistory(ctx context.Context, req *connect.Request[antv1.GetOrderLogHistoryRequest]) (*connect.Response[antv1.GetOrderLogHistoryResponse], error) {
	return connect.NewResponse(&antv1.GetOrderLogHistoryResponse{
		Orders: []*antv1.OrderHistoryRecord{},
		Total:  0,
	}), nil
}

func (s *LogServiceServer) GetOperationLogs(ctx context.Context, req *connect.Request[antv1.GetOperationLogsRequest]) (*connect.Response[antv1.GetOperationLogsResponse], error) {
	return connect.NewResponse(&antv1.GetOperationLogsResponse{
		Logs:  []*antv1.OperationLog{},
		Total: 0,
	}), nil
}

func (s *LogServiceServer) GetScheduleRunLogs(ctx context.Context, req *connect.Request[antv1.GetScheduleRunLogsRequest]) (*connect.Response[antv1.GetScheduleRunLogsResponse], error) {
	return connect.NewResponse(&antv1.GetScheduleRunLogsResponse{
		Logs:  []*antv1.ScheduleRunLog{},
		Total: 0,
	}), nil
}
