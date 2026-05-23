package mthub

import (
	"testing"
)

func TestSession(t *testing.T) {
	gw := &mockGateway{platform: "mt5", sessionID: "sess-1", brokerID: "broker-1"}
	s := &Session{AccountID: "acc-1", Gateway: gw}

	if s.Platform() != "mt5" {
		t.Errorf("Platform = %s, want mt5", s.Platform())
	}
	if s.SessionID() != "sess-1" {
		t.Errorf("SessionID = %s, want sess-1", s.SessionID())
	}
	if s.BrokerID() != "broker-1" {
		t.Errorf("BrokerID = %s, want broker-1", s.BrokerID())
	}
}

func TestSession_Conn(t *testing.T) {
	gw := &mockGateway{platform: "mt5"}
	s := &Session{AccountID: "acc-1", Gateway: gw}

	if c := s.Conn(); c != nil {
		t.Errorf("Conn = %v, want nil (grpc conn not exposed by mock)", c)
	}
}

func TestSession_NilGateway(t *testing.T) {
	s := &Session{AccountID: "test-account", Gateway: nil}
	if s.AccountID != "test-account" {
		t.Fatalf("expected test-account, got %s", s.AccountID)
	}
}
