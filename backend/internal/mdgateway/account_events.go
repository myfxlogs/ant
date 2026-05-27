package mdgateway

import (
	"context"
	"encoding/json"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Account lifecycle NATS subjects.
const (
	SubjectAccountConnect    = "account.connect"
	SubjectAccountDisconnect = "account.disconnect"
	SubjectAccountReconnect  = "account.reconnect"
)

// AccountEvent is published on account lifecycle changes.
type AccountEvent struct {
	AccountID string    `json:"account_id"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}

// AccountEventPublisher publishes account lifecycle events to NATS.
type AccountEventPublisher struct {
	js  natsgo.JetStreamContext
	log *zap.Logger
}

// NewAccountEventPublisher creates a publisher. js may be nil (events silently dropped).
func NewAccountEventPublisher(js natsgo.JetStreamContext, log *zap.Logger) *AccountEventPublisher {
	return &AccountEventPublisher{js: js, log: log}
}

func (p *AccountEventPublisher) publish(ctx context.Context, subject string, ev *AccountEvent) {
	if p.js == nil {
		return
	}
	data, err := json.Marshal(ev)
	if err != nil {
		p.log.Warn("account event marshal failed", zap.Error(err))
		return
	}
	if _, err := p.js.Publish(subject, data); err != nil {
		p.log.Warn("account event publish failed",
			zap.String("subject", subject),
			zap.String("account_id", ev.AccountID),
			zap.Error(err))
	}
}

// PublishConnect publishes an account.connect event.
func (p *AccountEventPublisher) PublishConnect(ctx context.Context, accountID, userID string) {
	p.publish(ctx, SubjectAccountConnect, &AccountEvent{
		AccountID: accountID, UserID: userID, Timestamp: time.Now(),
	})
}

// PublishDisconnect publishes an account.disconnect event.
func (p *AccountEventPublisher) PublishDisconnect(ctx context.Context, accountID, userID string) {
	p.publish(ctx, SubjectAccountDisconnect, &AccountEvent{
		AccountID: accountID, UserID: userID, Timestamp: time.Now(),
	})
}

// PublishReconnect publishes an account.reconnect event.
func (p *AccountEventPublisher) PublishReconnect(ctx context.Context, accountID, userID string) {
	p.publish(ctx, SubjectAccountReconnect, &AccountEvent{
		AccountID: accountID, UserID: userID, Timestamp: time.Now(),
	})
}
