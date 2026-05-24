package mdgateway

import (
	"fmt"
	natsgo "github.com/nats-io/nats.go"
	"anttrader/internal/mdgateway/adapter/mdtick"
)

type Publisher struct{ js natsgo.JetStreamContext }

func NewPublisher(js natsgo.JetStreamContext) *Publisher { return &Publisher{js: js} }
func (p *Publisher) PublishTick(t *mdtick.Tick) error {
	subj := fmt.Sprintf("md.tick.%s.%s", t.Broker, t.Canonical)
	if p.js == nil { return nil }
	msg := natsgo.NewMsg(subj)
	msg.Data = []byte(t.Bid.String())
	if t.IsReplay { // ADR-0009 §2.1
		msg.Header.Set("X-Ant-Replay", "1")
	}
	_, err := p.js.PublishMsg(msg)
	return err
}
func (p *Publisher) PublishBar(b *mdtick.Bar) error {
	if p.js == nil { return nil }
	subj := fmt.Sprintf("md.bar.%s.%s.%s", b.Broker, b.Canonical, b.Period)
	msg := natsgo.NewMsg(subj)
	msg.Data = []byte(b.Close.String())
	if b.IsReplay { // ADR-0009 §2.1
		msg.Header.Set("X-Ant-Replay", "1")
	}
	_, err := p.js.PublishMsg(msg)
	return err
}
