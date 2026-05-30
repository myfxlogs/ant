package user

import (
	"testing"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/mdgateway"
)

func TestSearchBroker_NilSearcher(t *testing.T) {
	srv := NewAccountServer(nil, nil, nil, nil, zap.NewNop()) // searcher is nil
	req := connect.NewRequest(&antv1.SearchBrokerRequest{Company: "Test"})
	_, err := srv.SearchBroker(t.Context(), req)
	if err == nil {
		t.Fatal("expected error when searcher is nil, got nil")
	}
}

func TestPublisher(t *testing.T) {
	srv := NewAccountServer(nil, nil, nil, nil, zap.NewNop())
	if srv.publisher != nil {
		t.Error("expected nil publisher from constructor")
	}
	pub := mdgateway.NewAccountEventPublisher(nil, zap.NewNop())
	srv.publisher = pub
	if srv.publisher != pub {
		t.Error("publisher not set")
	}
}

