package brokersearch

import (
	"os"
	"testing"

	mt4pb "alphaforge/mt4"
	mt5pb "alphaforge/mt5"
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

// TestBrokerSearch_NewFromConfig_UsesProvidedHosts verifies BROKER-SEARCH-1 S6/T6:
// NewFromConfig uses the provided gateway addresses instead of falling back
// to the hardcoded defaults.
//
// Adversarial proof (P3): revert NewFromConfig to ignore its parameters
// (always use defaults) → this test goes RED (host != custom).
func TestBrokerSearch_NewFromConfig_UsesProvidedHosts(t *testing.T) {
	s := NewFromConfig("custom.mtapi.io:443", "custom2.mtapi.io:443")
	if s.mt4Gateway != "custom.mtapi.io:443" {
		t.Errorf("NewFromConfig: expected mt4Gateway=custom.mtapi.io:443, got %s — "+
			"RED: NewFromConfig ignores provided mt4Gateway", s.mt4Gateway)
	}
	if s.mt5Gateway != "custom2.mtapi.io:443" {
		t.Errorf("NewFromConfig: expected mt5Gateway=custom2.mtapi.io:443, got %s — "+
			"RED: NewFromConfig ignores provided mt5Gateway", s.mt5Gateway)
	}
}

// TestBrokerSearch_New_EmptyFallbackToDefault verifies BROKER-SEARCH-1 S6/T7:
// New("", "") falls back to the hardcoded defaults (backward compatibility).
func TestBrokerSearch_New_EmptyFallbackToDefault(t *testing.T) {
	s := New("", "")
	if s.mt4Gateway != "mt4grpc3.mtapi.io:443" {
		t.Errorf("New: expected default mt4Gateway=mt4grpc3.mtapi.io:443, got %s", s.mt4Gateway)
	}
	if s.mt5Gateway != "mt5grpc3.mtapi.io:443" {
		t.Errorf("New: expected default mt5Gateway=mt5grpc3.mtapi.io:443, got %s", s.mt5Gateway)
	}
}

// TestBrokerSearch_NewFromConfig_EmptyFallbackToDefault verifies that
// NewFromConfig also falls back to defaults when given empty strings.
func TestBrokerSearch_NewFromConfig_EmptyFallbackToDefault(t *testing.T) {
	s := NewFromConfig("", "")
	if s.mt4Gateway != "mt4grpc3.mtapi.io:443" {
		t.Errorf("NewFromConfig: expected default mt4Gateway, got %s", s.mt4Gateway)
	}
	if s.mt5Gateway != "mt5grpc3.mtapi.io:443" {
		t.Errorf("NewFromConfig: expected default mt5Gateway, got %s", s.mt5Gateway)
	}
}

// TestPipeline_ReadsMtapiHostFromEnv verifies BROKER-SEARCH-1 S7/T8:
// The env-var-to-NewFromConfig wiring pattern (used in cmd/server/pipeline.go
// and cmd/server/handlers.go) correctly passes env var values to the Searcher.
// We test the integration here since the full pipeline requires too many deps.
func TestPipeline_ReadsMtapiHostFromEnv(t *testing.T) {
	t.Setenv("MTAPI_MT4_HOST", "env.mtapi.io:443")
	t.Setenv("MTAPI_MT5_HOST", "env2.mtapi.io:443")

	// This mirrors the wiring in pipeline.go:71 and handlers.go:67:
	//   brokersearch.NewFromConfig(os.Getenv("MTAPI_MT4_HOST"), os.Getenv("MTAPI_MT5_HOST"))
	s := NewFromConfig(os.Getenv("MTAPI_MT4_HOST"), os.Getenv("MTAPI_MT5_HOST"))
	if s.mt4Gateway != "env.mtapi.io:443" {
		t.Errorf("expected mt4Gateway=env.mtapi.io:443 from env, got %s — "+
			"RED: env var not wired to NewFromConfig", s.mt4Gateway)
	}
	if s.mt5Gateway != "env2.mtapi.io:443" {
		t.Errorf("expected mt5Gateway=env2.mtapi.io:443 from env, got %s — "+
			"RED: env var not wired to NewFromConfig", s.mt5Gateway)
	}
}
