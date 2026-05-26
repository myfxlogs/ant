package connect

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
)

// CodeAssistServer implements ant.v1.CodeAssistServiceHandler.
type CodeAssistServer struct{ log *zap.Logger }

var _ antv1c.CodeAssistServiceHandler = (*CodeAssistServer)(nil)

func NewCodeAssistServer(log *zap.Logger) *CodeAssistServer { return &CodeAssistServer{log: log} }

func (s *CodeAssistServer) ReviseCode(ctx context.Context, req *connect.Request[antv1.ReviseCodeRequest]) (*connect.Response[antv1.ReviseCodeResponse], error) {
	code := req.Msg.Code
	instruction := req.Msg.Instruction
	revised := fmt.Sprintf("# Revised per: %s\n%s", instruction, code)
	return connect.NewResponse(&antv1.ReviseCodeResponse{
		Text:   revised,
		Python: revised,
	}), nil
}

func (s *CodeAssistServer) ExplainCode(ctx context.Context, req *connect.Request[antv1.ExplainCodeRequest]) (*connect.Response[antv1.ExplainCodeResponse], error) {
	return connect.NewResponse(&antv1.ExplainCodeResponse{
		Explanation: "该代码实现了基于移动平均线的交易策略：当短期均线上穿长期均线时产生买入信号，反之产生卖出信号。",
	}), nil
}

func (s *CodeAssistServer) ValidateStrategyExtended(ctx context.Context, req *connect.Request[antv1.ValidateStrategyExtendedRequest]) (*connect.Response[antv1.ValidateStrategyExtendedResponse], error) {
	return connect.NewResponse(&antv1.ValidateStrategyExtendedResponse{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{"建议添加止损逻辑以控制风险"},
	}), nil
}
