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
	_, err := p.js.Publish(subj, []byte(t.Bid.String()))
	return err
}
func (p *Publisher) PublishBar(b *mdtick.Bar) error {
	if p.js == nil { return nil }
	subj := fmt.Sprintf("md.bar.%s.%s.%s", b.Broker, b.Canonical, b.Period)
	_, err := p.js.Publish(subj, []byte(b.Close.String()))
	return err
}
