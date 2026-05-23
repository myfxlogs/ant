package mdgateway

import (
	"testing"
)

func TestNewEmptyManager(t *testing.T) {
	m := NewEmptyManager()
	if m == nil {
		t.Fatal("NewEmptyManager returned nil")
	}
	conns := m.Connections()
	if len(conns) != 0 {
		t.Fatalf("expected 0 connections, got %d", len(conns))
	}
}

func TestManager_AddRemoveGateway(t *testing.T) {
	m := NewEmptyManager()
	cfg := AccountConfig{
		Broker:   "demo",
		Platform: "mt4",
		Login:    "12345",
		Password: "secret",
		Server:   "demo",
		Host:     "localhost",
		Port:     "443",
	}
	m.AddGateway(cfg)
	if len(m.Connections()) != 1 {
		t.Fatal("expected 1 connection after AddGateway")
	}
	m.RemoveGateway("demo-12345")
	if len(m.Connections()) != 0 {
		t.Fatal("expected 0 connections after RemoveGateway")
	}
}

func TestManager_SetNormalizer(t *testing.T) {
	m := NewEmptyManager()
	n := NewNormalizer(nil)
	m.SetNormalizer(n)
	if m.normalizer == nil {
		t.Fatal("normalizer should be set")
	}
}

func TestAccountConfig_Fields(t *testing.T) {
	ac := AccountConfig{
		Broker:     "demo",
		Platform:   "mt5",
		Login:      "12345",
		Password:   "pw",
		Server:     "srv",
		Host:       "h",
		Port:       "443",
		MtapiToken: "tok",
		UserID:     "u1",
		Status:     "active",
		IsDisabled: false,
	}
	if ac.Broker != "demo" || ac.Platform != "mt5" {
		t.Fatal("field mismatch")
	}
}

func TestConfig_Accounts(t *testing.T) {
	cfg := Config{
		Accounts: []AccountEntry{
			{
				UserID:   "u1",
				Broker:   "b1",
				Platform: "mt4",
				Login:    "111",
				Password: "pw",
				Server:   "s",
				Host:     "h",
				Port:     "443",
				Symbols:  []string{"EURUSD"},
			},
		},
	}
	if len(cfg.Accounts) != 1 {
		t.Fatal("expected 1 account")
	}
}
