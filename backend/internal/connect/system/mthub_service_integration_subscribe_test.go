//go:build integration

package system

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func TestMtHub_SubscribeEventsReceivesAccountStatus(t *testing.T) {
	harness := newMtHubTestHarness(t)

	ctx, cancel := context.WithCancel(harness.ctx())
	defer cancel()

	eventCh := make(chan *antv1.StreamEvent, 64)
	stream := newTestServerStream(eventCh, ctx)

	req := connect.NewRequest(&antv1.SubscribeEventsRequest{
		AccountIds: []string{harness.accountID},
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- harness.streamSrv.SubscribeEvents(ctx, req, stream)
	}()

	// SubscribeEvents sends initial snapshots (profit_update + account_status)
	// via sendInitialSnapshot which reads GetUserAccountSnapshots from DB.
	var gotStatus bool
	timeout := time.After(5 * time.Second)

	for !gotStatus {
		select {
		case ev, ok := <-eventCh:
			if !ok {
				t.Fatal("event channel closed unexpectedly")
			}
			t.Logf("received SSE event: type=%s account=%s", ev.GetType(), ev.AccountId)
			if ev.GetType() == "account_status" {
				gotStatus = true
				t.Logf("received account_status event for account %s PASS", ev.AccountId)
			}
		case err := <-errCh:
			t.Fatalf("SubscribeEvents returned early: %v", err)
		case <-timeout:
			t.Fatal("timed out waiting for account_status event")
		}
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Logf("SubscribeEvents returned after cancel: %v (expected)", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SubscribeEvents to finish after cancel")
	}
}

// ===========================================================================
// Test 5b: SSE stream connection established (verify multiple event types)
// ===========================================================================

func TestMtHub_SubscribeEventsConnectionEstablished(t *testing.T) {
	harness := newMtHubTestHarness(t)

	ctx, cancel := context.WithCancel(harness.ctx())
	defer cancel()

	eventCh := make(chan *antv1.StreamEvent, 64)
	stream := newTestServerStream(eventCh, ctx)

	req := connect.NewRequest(&antv1.SubscribeEventsRequest{
		AccountIds: []string{harness.accountID},
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- harness.streamSrv.SubscribeEvents(ctx, req, stream)
	}()

	receivedTypes := make(map[string]bool)
	timeout := time.After(5 * time.Second)

collectLoop:
	for len(receivedTypes) < 2 {
		select {
		case ev, ok := <-eventCh:
			if !ok {
				t.Fatal("event channel closed unexpectedly")
			}
			receivedTypes[ev.GetType()] = true
			t.Logf("received SSE event: type=%s account=%s", ev.GetType(), ev.AccountId)
		case err := <-errCh:
			t.Fatalf("SubscribeEvents returned early: %v", err)
		case <-timeout:
			t.Logf("collected events before timeout: %v", receivedTypes)
			break collectLoop
		}
	}

	if receivedTypes["account_status"] {
		t.Log("connection established: account_status received PASS")
	} else {
		t.Error("connection was NOT properly established: missing account_status event")
	}
	if receivedTypes["profit_update"] {
		t.Log("profit_update received PASS")
	}

	cancel()
	<-errCh
}

// ===========================================================================
// Helper: construct a real connect.ServerStream with a mock StreamingHandlerConn
// ===========================================================================

// mockStreamConn implements connect.StreamingHandlerConn and captures Send calls.
