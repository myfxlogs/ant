package connect

import (
	"context"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
)

// --- DebateV2Service ---

type DebateV2Server struct{ log *zap.Logger }

var _ antv1c.DebateV2ServiceHandler = (*DebateV2Server)(nil)

func NewDebateV2Server(log *zap.Logger) *DebateV2Server { return &DebateV2Server{log: log} }

func (s *DebateV2Server) StartDebateV2(ctx context.Context, req *connect.Request[antv1.StartDebateV2Request]) (*connect.Response[antv1.DebateV2Session], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) ChatDebateV2(ctx context.Context, req *connect.Request[antv1.ChatDebateV2Request]) (*connect.Response[antv1.DebateV2Session], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) AdvanceDebateV2(ctx context.Context, req *connect.Request[antv1.DebateV2SessionRequest]) (*connect.Response[antv1.DebateV2Session], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) StartDebateV2AdvanceJob(ctx context.Context, req *connect.Request[antv1.DebateV2SessionRequest]) (*connect.Response[antv1.StartDebateV2AdvanceJobResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) GetDebateV2AdvanceJob(ctx context.Context, req *connect.Request[antv1.GetDebateV2AdvanceJobRequest]) (*connect.Response[antv1.GetDebateV2AdvanceJobResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) PrepareDebateV2ChatJob(ctx context.Context, req *connect.Request[antv1.ChatDebateV2Request]) (*connect.Response[antv1.StartDebateV2ChatJobResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) RunDebateV2ChatJob(ctx context.Context, req *connect.Request[antv1.RunDebateV2ChatJobRequest]) (*connect.Response[emptypb.Empty], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) GetDebateV2ChatJob(ctx context.Context, req *connect.Request[antv1.GetDebateV2ChatJobRequest]) (*connect.Response[antv1.GetDebateV2ChatJobResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) BackDebateV2(ctx context.Context, req *connect.Request[antv1.DebateV2SessionRequest]) (*connect.Response[antv1.DebateV2Session], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) SetDebateV2Params(ctx context.Context, req *connect.Request[antv1.SetDebateV2ParamsRequest]) (*connect.Response[antv1.DebateV2Session], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) RejectDebateV2Code(ctx context.Context, req *connect.Request[antv1.RejectDebateV2CodeRequest]) (*connect.Response[antv1.DebateV2Session], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) StartDebateV2RejectCodeJob(ctx context.Context, req *connect.Request[antv1.RejectDebateV2CodeRequest]) (*connect.Response[antv1.StartDebateV2AdvanceJobResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) ListDebateV2Sessions(ctx context.Context, req *connect.Request[antv1.ListDebateV2SessionsRequest]) (*connect.Response[antv1.ListDebateV2SessionsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) GetDebateV2Session(ctx context.Context, req *connect.Request[antv1.GetDebateV2SessionRequest]) (*connect.Response[antv1.DebateV2Session], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *DebateV2Server) DeleteDebateV2Session(ctx context.Context, req *connect.Request[antv1.DeleteDebateV2SessionRequest]) (*connect.Response[emptypb.Empty], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

// --- DebateV2StreamService ---

type DebateV2StreamServer struct{ log *zap.Logger }

var _ antv1c.DebateV2StreamServiceHandler = (*DebateV2StreamServer)(nil)

func NewDebateV2StreamServer(log *zap.Logger) *DebateV2StreamServer {
	return &DebateV2StreamServer{log: log}
}

func (s *DebateV2StreamServer) SubscribeDebateV2Session(ctx context.Context, req *connect.Request[antv1.SubscribeDebateV2SessionRequest], stream *connect.ServerStream[antv1.DebateV2SessionEvent]) error {
	return connect.NewError(connect.CodeUnimplemented, nil)
}
