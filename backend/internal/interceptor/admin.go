package interceptor

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AdminChecker is the interface for checking admin privileges.
type AdminChecker interface {
	IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
}

// AdminInterceptor rejects requests from non-admin users.
type AdminInterceptor struct {
	checker AdminChecker
	log     *zap.Logger
}

func NewAdminInterceptor(c AdminChecker, log *zap.Logger) *AdminInterceptor {
	return &AdminInterceptor{checker: c, log: log}
}

func (i *AdminInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		uid, err := uuid.Parse(GetUserID(ctx))
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		ok, err := i.checker.IsAdmin(ctx, uid)
		if err != nil || !ok {
			i.log.Warn("admin denied",
				zap.String("user", uid.String()),
				zap.String("rpc", req.Spec().Procedure),
			)
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin required"))
		}
		return next(ctx, req)
	}
}

func (i *AdminInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *AdminInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		uid, err := uuid.Parse(GetUserID(ctx))
		if err != nil {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}
		ok, err := i.checker.IsAdmin(ctx, uid)
		if err != nil || !ok {
			return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin required"))
		}
		return next(ctx, conn)
	}
}
