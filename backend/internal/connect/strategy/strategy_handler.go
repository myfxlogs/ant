package strategy

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/pglisten"
	"alphaforge/internal/service"
)

// CodeAccessChecker checks whether a user can view full strategy code.
type CodeAccessChecker interface {
	CanAccessCode(ctx context.Context, userID, strategyID string) (bool, error)
}

// BoundAccountChecker ensures the MT account is bound to the user's subscription tier.
type BoundAccountChecker interface {
	EnsureBoundAccount(ctx context.Context, userID, accountID uuid.UUID) error
}

type StrategyServer struct {
	svc             *service.StrategySvc
	log             *zap.Logger
	pgListen        *pglisten.Listener
	engine          *ScheduleEngine
	codeAccess      CodeAccessChecker // marketplace code-access checks
	boundSvc        BoundAccountChecker
	sessionRegistry *SessionRegistry
}

// SetCodeAccessChecker injects the marketplace service for code protection.
func (s *StrategyServer) SetCodeAccessChecker(c CodeAccessChecker) { s.codeAccess = c }

var _ antv1c.StrategyServiceHandler = (*StrategyServer)(nil)

func NewStrategyServer(svc *service.StrategySvc, log *zap.Logger) *StrategyServer {
	return &StrategyServer{svc: svc, log: log}
}

// CancelTemplateDraft unpublishes a user's template (sets is_public=false).
func (s *StrategyServer) CancelTemplateDraft(ctx context.Context, req *connect.Request[antv1.CancelTemplateDraftRequest]) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.UnpublishUserTemplate(ctx, id, s.userID(ctx)); err != nil {
		if errors.Is(err, service.ErrTemplateNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *StrategyServer) SetEngine(e *ScheduleEngine) { s.engine = e }

func (s *StrategyServer) SetBoundSvc(b BoundAccountChecker) { s.boundSvc = b }

func (s *StrategyServer) SetSessionRegistry(r *SessionRegistry) { s.sessionRegistry = r }

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
