package strategy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
)

// TestWatchActiveStrategies_Heartbeat verifies LIVE-SSE-HEARTBEAT:
// When no session/tick changes occur, the SSE stream sends an empty
// heartbeat event periodically to keep the connection alive.
//
// Adversarial proof: Delete the `case <-heartbeat.C` branch →
// no heartbeat Send within 25s → test times out (RED).
func TestWatchActiveStrategies_Heartbeat(t *testing.T) {
	t.Parallel()

	userID := uuid.New().String()
	srv := NewStrategyExecutionServer(nil, zap.NewNop())
	srv.sessionRegistry = NewSessionRegistry()
	srv.heartbeatInterval = 100 * time.Millisecond // fast heartbeat for test

	mux := http.NewServeMux()
	mux.Handle(antv1c.NewStrategyRuntimeServiceHandler(srv,
		connect.WithInterceptors(&fakeAuthInterceptor{userID: userID})))

	server := httptest.NewServer(mux)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := antv1c.NewStrategyRuntimeServiceClient(http.DefaultClient, server.URL)
	stream, err := client.WatchActiveStrategies(ctx, connect.NewRequest(&antv1.WatchActiveStrategiesRequest{}))
	if err != nil {
		t.Fatalf("WatchActiveStrategies failed: %v", err)
	}

	// First event: the initial list (empty, but non-nil strategies slice)
	stream.Receive()
	if err := stream.Err(); err != nil {
		t.Fatalf("initial Receive failed: %v", err)
	}

	// Second event: heartbeat (empty event with nil/empty strategies)
	heartbeatReceived := false
	deadline := time.After(2 * time.Second)
	for !heartbeatReceived {
		select {
		case <-deadline:
			t.Fatal("no heartbeat received within 2s — RED: case <-heartbeat.C missing or broken")
		default:
		}
		ok := stream.Receive()
		if err := stream.Err(); err != nil {
			t.Fatalf("Receive failed waiting for heartbeat: %v", err)
		}
		if !ok {
			t.Fatal("stream closed before heartbeat")
		}
		event := stream.Msg()
		// Heartbeat = empty event (no strategies field set)
		if len(event.GetStrategies()) == 0 {
			heartbeatReceived = true
		}
	}

	if !heartbeatReceived {
		t.Fatal("heartbeat event was not received")
	}
}

// fakeAuthInterceptor injects a user ID into context for testing.
type fakeAuthInterceptor struct {
	userID string
}

func (f *fakeAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		ctx = context.WithValue(ctx, interceptor.UserIDKey, f.userID)
		return next(ctx, req)
	}
}

func (f *fakeAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (f *fakeAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		ctx = context.WithValue(ctx, interceptor.UserIDKey, f.userID)
		return next(ctx, conn)
	}
}
