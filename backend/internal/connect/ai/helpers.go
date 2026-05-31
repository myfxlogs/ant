package ai

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"anttrader/internal/interceptor"
)

// userIDFromCtx extracts the authenticated user ID from the ConnectRPC context.
func userIDFromCtx(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id"))
	}
	return id, nil
}

// debateErrorCode maps internal service errors to appropriate ConnectRPC codes
// so the frontend can distinguish not-found vs permission-denied vs retryable.
func debateErrorCode(err error) connect.Code {
	if err == nil {
		return connect.CodeInternal
	}
	msg := strings.ToLower(err.Error())

	// Check most specific patterns first.
	switch {
	// — not found —
	case strings.Contains(msg, "session not found"):
		return connect.CodeNotFound
	case strings.Contains(msg, "job") && strings.Contains(msg, "not found"):
		return connect.CodeNotFound

	// — invalid argument —
	case strings.Contains(msg, "invalid session id"),
		strings.Contains(msg, "invalid job id"),
		strings.Contains(msg, "invalid conversation_id"):
		return connect.CodeInvalidArgument
	case strings.Contains(msg, "agents must not be empty"):
		return connect.CodeInvalidArgument

	// — permission —
	case strings.Contains(msg, "unauthorized"):
		return connect.CodePermissionDenied

	// — precondition —
	case strings.Contains(msg, "already at first step"),
		strings.Contains(msg, "no next step available"):
		return connect.CodeFailedPrecondition

	// — retryable —
	case strings.Contains(msg, "concurrent modification"):
		return connect.CodeAborted

	// — conflict —
	case strings.Contains(msg, "already running"):
		return connect.CodeAlreadyExists

	default:
		return connect.CodeInternal
	}
}
