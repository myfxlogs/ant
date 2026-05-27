package mdgateway

import (
	"context"
	"fmt"
	natsgo "github.com/nats-io/nats.go"
	"anttrader/internal/interceptor"
	"anttrader/internal/mdgateway/adapter/mdtick"
)

type Publisher struct {
	js natsgo.JetStreamContext
}

func NewPublisher(js natsgo.JetStreamContext) *Publisher { return &Publisher{js: js} }

func (p *Publisher) PublishTick(ctx context.Context, t *mdtick.Tick) error {
	subj := fmt.Sprintf("md.tick.%s.%s", t.Broker, t.Canonical)
	if p.js == nil { return nil }
	msg := natsgo.NewMsg(subj)
	msg.Data = []byte(t.Bid.String())
	msg.Header.Set("X-Ant-Replay", boolToStr(t.IsReplay))
	msg.Header.Set("Nats-Msg-Id", fmt.Sprintf("%s:%s:%d:%x", t.Broker, t.Canonical, t.TsUnixMs, hashTick(t)))
	interceptor.InjectNATSTraceHeaders(ctx, msg.Header)
	_, err := p.js.PublishMsg(msg)
	if err != nil {
		RecordNATSPublishDropped()
		return fmt.Errorf("publish tick to NATS: %w", err)
	}
	return nil
}

func (p *Publisher) PublishBar(ctx context.Context, b *mdtick.Bar) error {
	if p.js == nil { return nil }
	subj := fmt.Sprintf("md.bar.%s.%s.%s", b.Broker, b.Canonical, b.Period)
	msg := natsgo.NewMsg(subj)
	msg.Data = []byte(b.Close.String())
	msg.Header.Set("X-Ant-Replay", boolToStr(b.IsReplay))
	msg.Header.Set("Nats-Msg-Id", fmt.Sprintf("%s:%s:%s:%d", b.Broker, b.Canonical, b.Period, b.CloseTsUnixMs))
	interceptor.InjectNATSTraceHeaders(ctx, msg.Header)
	_, err := p.js.PublishMsg(msg)
	if err != nil {
		RecordNATSPublishDropped()
		return fmt.Errorf("publish bar to NATS: %w", err)
	}
	return nil
}

// PublishBarRevision publishes a bar revision event (ADR-0016).
func (p *Publisher) PublishBarRevision(ctx context.Context, b *mdtick.Bar) error {
	if p.js == nil { return nil }
	subj := fmt.Sprintf("md.bar.revision.%s.%s.%s", b.Broker, b.Canonical, b.Period)
	msg := natsgo.NewMsg(subj)
	msg.Data = []byte(b.Close.String())
	msg.Header.Set("X-Ant-Bar-Version", "2")
	msg.Header.Set("Nats-Msg-Id", fmt.Sprintf("rev:%s:%s:%s:%d", b.Broker, b.Canonical, b.Period, b.CloseTsUnixMs))
	interceptor.InjectNATSTraceHeaders(ctx, msg.Header)
	_, err := p.js.PublishMsg(msg)
	if err != nil {
		return fmt.Errorf("publish bar revision to NATS: %w", err)
	}
	return nil
}

func boolToStr(b bool) string {
	if b { return "1" }
	return "0"
}

func hashTick(t *mdtick.Tick) uint64 {
	var h uint64 = 14695981039346656037
	for _, c := range t.Broker { h = (h ^ uint64(c)) * 1099511628211 }
	h = (h ^ '/') * 1099511628211
	for _, c := range t.Canonical { h = (h ^ uint64(c)) * 1099511628211 }
	return h ^ uint64(t.TsUnixMs)
}
