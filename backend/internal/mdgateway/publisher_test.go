package mdgateway

import (
	"testing"

	"go.uber.org/zap"
)

func TestPublisher_Config(t *testing.T) {
	cfg := DefaultPublisherConfig()
	if cfg.URL != "nats://localhost:4222" {
		t.Fatalf("expected nats://localhost:4222, got %s", cfg.URL)
	}
	if cfg.Stream != "md_events" {
		t.Fatalf("expected md_events, got %s", cfg.Stream)
	}
}

func TestPublisher_New(t *testing.T) {
	p := NewPublisher(DefaultPublisherConfig(), zap.NewNop())
	if p == nil {
		t.Fatal("NewPublisher returned nil")
	}
	if p.cfg.MaxRetry != 3 {
		t.Fatal("expected MaxRetry=3")
	}
}

func TestPublisher_CloseWithoutConnect(t *testing.T) {
	p := NewPublisher(DefaultPublisherConfig(), zap.NewNop())
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPublisher_PublishWithoutConnect(t *testing.T) {
	p := NewPublisher(DefaultPublisherConfig(), zap.NewNop())
	err := p.Publish(nil, &Tick{UserID: "u1", Symbol: "EURUSD"})
	if err == nil {
		t.Fatal("expected error publishing without connect")
	}
}

func TestPublisher_PublishBarWithoutConnect(t *testing.T) {
	p := NewPublisher(DefaultPublisherConfig(), zap.NewNop())
	err := p.PublishBar(Bar{UserID: "u1"})
	if err == nil {
		t.Fatal("expected error without connect")
	}
}

func TestPublisher_NilTick(t *testing.T) {
	p := NewPublisher(DefaultPublisherConfig(), zap.NewNop())
	// nil tick guard should return nil
	if err := p.Publish(nil, nil); err != nil {
		t.Fatal("nil tick should not error")
	}
}
