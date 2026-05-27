package user

import (
	"testing"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/mdgateway"
)

func TestSearchBroker_FallbackToMock(t *testing.T) {
	srv := &AccountServer{log: zap.NewNop()} // searcher is nil
	req := connect.NewRequest(&antv1.SearchBrokerRequest{Company: "Test"})
	resp, err := srv.SearchBroker(t.Context(), req)
	if err != nil {
		t.Fatalf("SearchBroker: %v", err)
	}
	companies := resp.Msg.Companies
	if len(companies) != 1 {
		t.Fatalf("expected 1 mock company, got %d", len(companies))
	}
	if companies[0].CompanyName != "Exness" {
		t.Errorf("expected Exness fallback, got %s", companies[0].CompanyName)
	}
	if len(companies[0].Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(companies[0].Servers))
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

