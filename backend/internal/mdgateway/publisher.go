package mdgateway

import (
	antv1 "anttrader/gen/proto/ant/v1"
	"context"
	"google.golang.org/protobuf/proto"
	"fmt"
	"regexp"
	natsgo "github.com/nats-io/nats.go"
	"anttrader/internal/interceptor"
	"anttrader/internal/mdgateway/adapter/mdtick"
)

// sanitizeNATSSubject replaces characters that are invalid in NATS subjects.
// NATS subjects: only alphanumeric, `.`, `_`, `-` are allowed; spaces/special chars cause errors.
var sanitizeNATSSubject = regexp.MustCompile(`[^a-zA-Z0-9._\-]`).ReplaceAllString

type Publisher struct {
	js natsgo.JetStreamContext
}

func NewPublisher(js natsgo.JetStreamContext) *Publisher { return &Publisher{js: js} }

func (p *Publisher) subjectKey(broker string) string {
	return sanitizeNATSSubject(broker, "_")
}


func (p *Publisher) PublishTick(ctx context.Context, t *mdtick.Tick) error {
	subj := fmt.Sprintf("md.tick.%s.%s", p.subjectKey(t.Broker), t.Canonical)
	if p.js == nil { return nil }
	msg := natsgo.NewMsg(subj)
	payload, _ := proto.Marshal(&antv1.TickPayload{
		Broker: t.Broker, Canonical: t.Canonical,
		TsUnixMs: t.TsUnixMs, Bid: t.Bid.String(), Ask: t.Ask.String(),
	})
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal tick payload: %w", err)
	}
	msg.Data = data
	msg.Header.Set("X-Ant-Replay", boolToStr(t.IsReplay))
	msg.Header.Set("Nats-Msg-Id", fmt.Sprintf("%s:%s:%d:%x", p.subjectKey(t.Broker), t.Canonical, t.TsUnixMs, hashTick(t)))
	interceptor.InjectNATSTraceHeaders(ctx, msg.Header)
	_, err = p.js.PublishMsg(msg)
	if err != nil {
		RecordNATSPublishDropped()
		return fmt.Errorf("publish tick to NATS: %w", err)
	}
	return nil
}

func (p *Publisher) PublishBar(ctx context.Context, b *mdtick.Bar) error {
	if p.js == nil { return nil }
	subj := fmt.Sprintf("md.bar.%s.%s.%s", p.subjectKey(b.Broker), b.Canonical, b.Period)
	msg := natsgo.NewMsg(subj)
	// Publish full OHLCV as JSON for downstream consumers.
	payload, _ := json.Marshal(map[string]interface{}{
		"open":  b.Open.String(),
		"high":  b.High.String(),
		"low":   b.Low.String(),
		"close": b.Close.String(),
		"volume": b.Volume,
		"tick_count": b.TickCount,
		"open_ts": b.OpenTsUnixMs,
		"close_ts": b.CloseTsUnixMs,
		"period": b.Period,
	})
	msg.Data = payload
	msg.Header.Set("X-Ant-Replay", boolToStr(b.IsReplay))
	msg.Header.Set("Nats-Msg-Id", fmt.Sprintf("%s:%s:%s:%d", p.subjectKey(b.Broker), b.Canonical, b.Period, b.CloseTsUnixMs))
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
	subj := fmt.Sprintf("md.bar.revision.%s.%s.%s", p.subjectKey(b.Broker), b.Canonical, b.Period)
	msg := natsgo.NewMsg(subj)
	// Publish full OHLCV as JSON for downstream consumers.
	payload, _ := json.Marshal(map[string]interface{}{
		"open":  b.Open.String(),
		"high":  b.High.String(),
		"low":   b.Low.String(),
		"close": b.Close.String(),
		"volume": b.Volume,
		"tick_count": b.TickCount,
		"open_ts": b.OpenTsUnixMs,
		"close_ts": b.CloseTsUnixMs,
		"period": b.Period,
	})
	msg.Data = payload
	msg.Header.Set("X-Ant-Bar-Version", "2")
	msg.Header.Set("Nats-Msg-Id", fmt.Sprintf("rev:%s:%s:%s:%d", p.subjectKey(b.Broker), b.Canonical, b.Period, b.CloseTsUnixMs))
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
