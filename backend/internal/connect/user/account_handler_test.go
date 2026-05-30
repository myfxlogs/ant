package user

import (
	"testing"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/mdgateway"
)

func TestSearchBroker_NilSearcher(t *testing.T) {
	srv := &AccountServer{log: zap.NewNop()} // searcher is nil
	req := connect.NewRequest(&antv1.SearchBrokerRequest{Company: "Test"})
	_, err := srv.SearchBroker(t.Context(), req)
	if err == nil {
		t.Fatal("expected error when searcher is nil, got nil")
	}
}

func TestSetPublisher(t *testing.T) {
	srv := &AccountServer{log: zap.NewNop()}
	pub := mdgateway.NewAccountEventPublisher(nil, zap.NewNop())
	srv.SetPublisher(pub)
	if srv.publisher != pub {
		t.Error("SetPublisher did not set publisher")
	}
	// SetPublisher with a nil-safe publisher: no panic.
	srv.SetPublisher(nil)
	if srv.publisher != nil {
		t.Error("expected nil publisher after setting nil")
	}
}

