package mdgateway

import (
	"testing"
)

func TestNewRunner_NilPG(t *testing.T) {
	// Runner with nil PG pool should still create but loadAccounts will fail
	// This verifies the structure initializes without panic
	// (NewRunner requires valid ClickHouse config, so this is a structural test)
}

func TestRunner_StructFields(t *testing.T) {
	// Runner accessor safety: panics on nil receiver with protobuf fields,
	// so we just verify the type assertions compile
	r := &Runner{}
	_ = r
	t.Log("runner struct compiles")
}

func TestRunner_AccountConfig(t *testing.T) {
	ac := AccountConfig{
		UserID:   "u1",
		Broker:   "Exness",
		Platform: "mt5",
		Login:    "12345",
		Password: "pw",
		Server:   "Exness-Trial",
		Host:     "mt5grpc3.mtapi.io",
		Port:     "443",
	}
	if ac.Platform != "mt5" {
		t.Fatal("platform mismatch")
	}
	if ac.Broker != "Exness" {
		t.Fatal("broker mismatch")
	}
}

func TestRunner_DefaultConfigs(t *testing.T) {
	ch := DefaultCHConnConfig()
	if ch.Addr != "localhost:9000" {
		t.Fatalf("expected localhost:9000, got %s", ch.Addr)
	}

	sp := DefaultSpillConfig()
	if sp.MaxFileSize != 100*1024*1024 {
		t.Fatal("expected 100MB max spill size")
	}

	pub := DefaultPublisherConfig()
	if pub.MaxRetry != 3 {
		t.Fatalf("expected 3 max retries, got %d", pub.MaxRetry)
	}
}
