package source

import (
	"context"
	"fmt"
	"strconv"

	natsgo "github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// LiveSource implements BarSource via NATS JetStream subscriptions.
// NATS bar messages carry Close price + metadata headers; full OHLC is in ClickHouse.
type LiveSource struct {
	js  natsgo.JetStreamContext
	log *zap.Logger
}

// NewLiveSource creates a live bar source backed by NATS JetStream.
func NewLiveSource(js natsgo.JetStreamContext, log *zap.Logger) *LiveSource {
	return &LiveSource{js: js, log: log}
}

// Subscribe returns a channel that receives bar close-price updates.
// Subject: md.bar.<broker>.<canonical>.<period> — use "*" for broker to match all.
func (s *LiveSource) Subscribe(ctx context.Context, canonical, period string) (<-chan *mdtick.Bar, error) {
	if s.js == nil {
		return nil, fmt.Errorf("source: NATS JetStream not available")
	}

	subject := fmt.Sprintf("md.bar.*.%s.%s", canonical, period)
	ch := make(chan *mdtick.Bar, 128)

	sub, err := s.js.Subscribe(subject, func(msg *natsgo.Msg) {
		bar := &mdtick.Bar{
			Canonical: canonical,
			Period:    period,
		}

		// Parse Close price from body.
		if closePrice, parseErr := decimal.NewFromString(string(msg.Data)); parseErr == nil {
			bar.Close = closePrice
			bar.Open = closePrice
			bar.High = closePrice
			bar.Low = closePrice
		}

		// Extract broker from subject: md.bar.<broker>.<canonical>.<period>
		bar.Broker = extractBroker(msg.Subject)

		// Parse timestamp from Nats-Msg-Id header.
		if id := msg.Header.Get("Nats-Msg-Id"); id != "" {
			// Format: <broker>:<canonical>:<period>:<ts_ms>
			if ts, err := parseLastInt64(id); err == nil {
				bar.OpenTsUnixMs = ts
				bar.CloseTsUnixMs = ts
			}
		}

		bar.IsReplay = msg.Header.Get("X-Ant-Replay") == "true"

		select {
		case ch <- bar:
		case <-ctx.Done():
		default:
		}
	})
	if err != nil {
		return nil, fmt.Errorf("source: subscribe %s: %w", subject, err)
	}

	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
		close(ch)
	}()

	return ch, nil
}

// extractBroker parses the broker segment from subject "md.bar.<broker>.<canonical>.<period>".
func extractBroker(subject string) string {
	parts := splitSubject(subject)
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// splitSubject splits a NATS subject by '.'.
func splitSubject(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// parseLastInt64 extracts the last colon-separated segment as int64.
func parseLastInt64(s string) (int64, error) {
	idx := 0
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			idx = i + 1
			break
		}
	}
	return strconv.ParseInt(s[idx:], 10, 64)
}
