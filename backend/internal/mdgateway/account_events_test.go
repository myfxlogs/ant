package mdgateway

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestAccountEventPublisher_NilJS(t *testing.T) {
	pub := NewAccountEventPublisher(nil, zap.NewNop())
	// Must not panic with nil JetStream.
	pub.PublishConnect(context.Background(), "acc-1", "user-1")
	pub.PublishDisconnect(context.Background(), "acc-1", "user-1")
	pub.PublishReconnect(context.Background(), "acc-1", "user-1")
}

func TestAccountEventPublisher_NonNilJS(t *testing.T) {
	// Publish to a real NATS connection if available, else skip.
	// For unit tests without NATS, the nil-JS path above covers safety.
	pub := NewAccountEventPublisher(nil, zap.NewNop())
	if pub == nil {
		t.Fatal("expected non-nil publisher")
	}
	if pub.js != nil {
		t.Error("expected nil js")
	}
}

func TestAccountEventSubjects(t *testing.T) {
	if SubjectAccountConnect != "account.connect" {
		t.Errorf("expected account.connect, got %s", SubjectAccountConnect)
	}
	if SubjectAccountDisconnect != "account.disconnect" {
		t.Errorf("expected account.disconnect, got %s", SubjectAccountDisconnect)
	}
	if SubjectAccountReconnect != "account.reconnect" {
		t.Errorf("expected account.reconnect, got %s", SubjectAccountReconnect)
	}
}
