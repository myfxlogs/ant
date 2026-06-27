package strategy

import (
	"context"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/service"
	"anttrader/internal/pglisten"
)

// CodeAccessChecker checks whether a user can view full strategy code.
type CodeAccessChecker interface {
	CanAccessCode(ctx context.Context, userID, strategyID string) (bool, error)
}

type StrategyServer struct {
	svc            *service.StrategySvc
	log            *zap.Logger
	pgListen       *pglisten.Listener
	engine         *ScheduleEngine
	codeAccess     CodeAccessChecker // marketplace code-access checks
}

// SetCodeAccessChecker injects the marketplace service for code protection.
func (s *StrategyServer) SetCodeAccessChecker(c CodeAccessChecker) { s.codeAccess = c }

var _ antv1c.StrategyServiceHandler = (*StrategyServer)(nil)

func NewStrategyServer(svc *service.StrategySvc, log *zap.Logger) *StrategyServer {
	return &StrategyServer{svc: svc, log: log}
}

// CancelTemplateDraft reverts a draft template back to published status.
func (s *StrategyServer) CancelTemplateDraft(ctx context.Context, req *connect.Request[antv1.CancelTemplateDraftRequest]) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.SetTemplateStatus(ctx, id, s.userID(ctx), "published"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *StrategyServer) SetEngine(e *ScheduleEngine) { s.engine = e }

func (s *StrategyServer) userID(ctx context.Context) uuid.UUID {
	raw := interceptor.GetUserID(ctx)
	if raw == "" {
		s.log.Warn("strategy: userID called but no user in context")
		return uuid.Nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		s.log.Warn("strategy: userID parse failed", zap.String("raw", raw), zap.Error(err))
		return uuid.Nil
	}
	return id
}

func (s *StrategyServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
