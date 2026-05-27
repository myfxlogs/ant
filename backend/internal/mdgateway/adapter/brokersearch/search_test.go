package brokersearch

import (
	"testing"

	mt4pb "anttrader/mt4"
	mt5pb "anttrader/mt5"
)

func TestNew_Defaults(t *testing.T) {
	s := New("", "")
	if s.mt4Gateway != "mt4grpc3.mtapi.io:443" {
		t.Errorf("expected default mt4 gateway, got %s", s.mt4Gateway)
	}
	if s.mt5Gateway != "mt5grpc3.mtapi.io:443" {
		t.Errorf("expected default mt5 gateway, got %s", s.mt5Gateway)
	}
}

func TestNew_Custom(t *testing.T) {
	s := New("mt4.custom:443", "mt5.custom:443")
	if s.mt4Gateway != "mt4.custom:443" {
		t.Errorf("expected custom mt4 gateway, got %s", s.mt4Gateway)
	}
	if s.mt5Gateway != "mt5.custom:443" {
		t.Errorf("expected custom mt5 gateway, got %s", s.mt5Gateway)
	}
}

func TestMapMT4Reply(t *testing.T) {
	reply := &mt4pb.SearchReply{
		Result: []*mt4pb.Company{
			{
				CompanyName: "Exness",
				Results: []*mt4pb.Result{
					{Name: "Exness-Real", Access: []string{"mt4"}},
					{Name: "Exness-Demo", Access: []string{"mt4"}},
				},
			},
		},
	}
	companies := mapMT4Reply(reply)
	if len(companies) != 1 {
		t.Fatalf("expected 1 company, got %d", len(companies))
	}
	if companies[0].CompanyName != "Exness" {
		t.Errorf("expected Exness, got %s", companies[0].CompanyName)
	}
	if len(companies[0].Servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(companies[0].Servers))
	}
	if companies[0].Servers[0].Name != "Exness-Real" {
		t.Errorf("expected Exness-Real, got %s", companies[0].Servers[0].Name)
	}
}

func TestMapMT5Reply(t *testing.T) {
	reply := &mt5pb.SearchReply{
		Result: []*mt5pb.Company{
			{
				CompanyName: "ICMarkets",
				Results: []*mt5pb.Result{
					{Name: "ICMarkets-Demo", Access: []string{"mt5"}},
				},
			},
		},
	}
	companies := mapMT5Reply(reply)
	if len(companies) != 1 {
		t.Fatalf("expected 1 company, got %d", len(companies))
	}
	if companies[0].CompanyName != "ICMarkets" {
		t.Errorf("expected ICMarkets, got %s", companies[0].CompanyName)
	}
	if len(companies[0].Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(companies[0].Servers))
	}
}

func TestMapMT4Reply_Nil(t *testing.T) {
	companies := mapMT4Reply(nil)
	if companies != nil {
		t.Errorf("expected nil for nil reply, got %v", companies)
	}
}

func TestMapMT5Reply_Nil(t *testing.T) {
	companies := mapMT5Reply(nil)
	if companies != nil {
		t.Errorf("expected nil for nil reply, got %v", companies)
	}
}

func TestMapMT4Reply_EmptyResult(t *testing.T) {
	reply := &mt4pb.SearchReply{}
	companies := mapMT4Reply(reply)
	if len(companies) != 0 {
		t.Errorf("expected 0 companies, got %d", len(companies))
	}
}
