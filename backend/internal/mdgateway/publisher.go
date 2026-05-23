// Package mdgateway — NATS publisher (stub for ant).
// Ant uses ClickHouse as the primary market data persistence layer.
// NATS publisher is stubbed out; uncomment when NATS is added to infrastructure.
package mdgateway

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Publisher is a no-op publisher stub. Ant routes all market data through
// the ClickHouse writer and the internal subscriber fan-out mechanism.
// NATS integration can be added later by replacing this stub.
type Publisher struct {
	log *zap.Logger
}

// NewPublisher creates a no-op Publisher.
func NewPublisher(log *zap.Logger, _ string) *Publisher {
	return &Publisher{log: log}
}

// Connect is a no-op.
func (p *Publisher) Connect(_ context.Context) error {
	p.log.Debug("publisher: NATS stubbed, skipping connect")
	return nil
}

// Publish is a no-op.
func (p *Publisher) Publish(_ context.Context, _ *Tick) error {
	return fmt.Errorf("publisher: NATS not available (stub)")
}

// PublishRaw is a no-op.
func (p *Publisher) PublishRaw(_ string, _ []byte) error {
	return nil
}

// PublishBar is a no-op.
func (p *Publisher) PublishBar(_ Bar) error {
	return fmt.Errorf("publisher: NATS not available (stub)")
}

// EnsureStreams is a no-op.
func (p *Publisher) EnsureStreams(_ *zap.Logger) {}

// Close is a no-op.
func (p *Publisher) Close() error { return nil }
