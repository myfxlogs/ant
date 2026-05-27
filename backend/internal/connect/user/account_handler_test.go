package user

import (
	"testing"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
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
