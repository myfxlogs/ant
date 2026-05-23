// Package mdgateway — NATS JetStream publisher.
// Publishes normalized Tick and Bar events to NATS subjects for
// cross-service consumption (strategy engine, factor service, analytics).
package mdgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// PublisherConfig holds NATS connection settings.
type PublisherConfig struct {
	URL      string        // NATS server URL (default "nats://localhost:4222")
	Stream   string        // JetStream stream name (default "md_events")
	MaxRetry int           // max publish retries (default 3)
	Timeout  time.Duration // publish timeout (default 5s)
}

// DefaultPublisherConfig returns sensible defaults.
func DefaultPublisherConfig() PublisherConfig {
	return PublisherConfig{
		URL:      nats.DefaultURL,
		Stream:   "md_events",
		MaxRetry: 3,
		Timeout:  5 * time.Second,
	}
}

// Publisher publishes market data events to NATS JetStream.
type Publisher struct {
	cfg    PublisherConfig
	log    *zap.Logger
	nc     *nats.Conn
	js     nats.JetStreamContext
	closed bool
}

// NewPublisher creates a NATS publisher. Call Connect before use.
func NewPublisher(cfg PublisherConfig, log *zap.Logger) *Publisher {
	if cfg.URL == "" {
		cfg.URL = nats.DefaultURL
	}
	if cfg.Stream == "" {
		cfg.Stream = "md_events"
	}
	if cfg.MaxRetry <= 0 {
		cfg.MaxRetry = 3
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &Publisher{cfg: cfg, log: log}
}

// Connect establishes the NATS connection and ensures the JetStream stream exists.
func (p *Publisher) Connect(ctx context.Context) error {
	nc, err := nats.Connect(p.cfg.URL,
		nats.Timeout(p.cfg.Timeout),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if p.log != nil {
				p.log.Warn("nats: disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			if p.log != nil {
				p.log.Info("nats: reconnected")
			}
		}),
	)
	if err != nil {
		return fmt.Errorf("publisher: connect: %w", err)
	}
	p.nc = nc

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return fmt.Errorf("publisher: jetstream: %w", err)
	}
	p.js = js

	if err := p.EnsureStreams(); err != nil {
		nc.Close()
		return fmt.Errorf("publisher: ensure streams: %w", err)
	}

	if p.log != nil {
		p.log.Info("publisher: connected", zap.String("url", p.cfg.URL))
	}
	return nil
}

// EnsureStreams creates the JetStream stream if it doesn't exist.
func (p *Publisher) EnsureStreams() error {
	if p.js == nil {
		return fmt.Errorf("publisher: not connected")
	}
	_, err := p.js.AddStream(&nats.StreamConfig{
		Name:     p.cfg.Stream,
		Subjects: []string{p.cfg.Stream + ".>"},
		MaxAge:   24 * time.Hour,
		Storage:  nats.FileStorage,
	})
	if err != nil {
		// Stream already exists is not an error
		if err.Error() != "stream name already in use" {
			return fmt.Errorf("publisher: add stream: %w", err)
		}
	}
	return nil
}

// Publish sends a Tick to the "md_events.tick" subject.
func (p *Publisher) Publish(ctx context.Context, tick *Tick) error {
	if tick == nil {
		return nil
	}
	return p.publish("tick", tick)
}

// PublishBar sends a Bar to the "md_events.bar" subject.
func (p *Publisher) PublishBar(bar Bar) error {
	return p.publish("bar", bar)
}

// PublishRaw sends raw bytes to a sub-topic under the stream.
func (p *Publisher) PublishRaw(subTopic string, data []byte) error {
	if p.js == nil {
		return fmt.Errorf("publisher: not connected")
	}
	subject := p.cfg.Stream + "." + subTopic
	_, err := p.js.Publish(subject, data)
	return err
}

func (p *Publisher) publish(kind string, v interface{}) error {
	if p.js == nil {
		return fmt.Errorf("publisher: not connected")
	}

	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("publisher: marshal: %w", err)
	}

	subject := p.cfg.Stream + "." + kind
	var lastErr error
	for i := 0; i < p.cfg.MaxRetry; i++ {
		_, err = p.js.Publish(subject, data)
		if err == nil {
			return nil
		}
		lastErr = err
		if p.log != nil {
			p.log.Warn("publisher: retry",
				zap.String("subject", subject),
				zap.Int("attempt", i+1),
				zap.Error(err),
			)
		}
		time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
	}
	return fmt.Errorf("publisher: publish after %d retries: %w", p.cfg.MaxRetry, lastErr)
}

// Close drains and closes the NATS connection.
func (p *Publisher) Close() error {
	if p.nc != nil {
		p.nc.Close()
	}
	p.closed = true
	return nil
}
