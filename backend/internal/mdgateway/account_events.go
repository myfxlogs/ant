package mdgateway

import (
	"context"
	"google.golang.org/protobuf/proto"
	antv1 "alphaforge/gen/proto/ant/v1"
	"sync/atomic"
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
	AccountID string
	UserID    string
	Timestamp time.Time
}

// AccountEventPublisher publishes account lifecycle events to NATS JetStream.
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
	// Respect caller's context deadline/cancellation.
	if err := ctx.Err(); err != nil {
		return
	}
	data, err := proto.Marshal(&antv1.AccountEventPayload{AccountId: ev.AccountID, UserId: ev.UserID, TsUnixMs: ev.Timestamp.UnixMilli()})
	if err != nil {
		p.log.Warn("account event marshal failed", zap.Error(err))
		return
	}
	// Ensure the JetStream stream exists (idempotent).
	tryEnsureAccountEventsStream(p.js, p.log)

	// Retry once with backoff on transient publish failures.
	opts := []natsgo.PubOpt{natsgo.Context(ctx)}
	if _, err := p.js.Publish(subject, data, opts...); err != nil {
		p.log.Warn("account event publish failed, retrying",
			zap.String("subject", subject),
			zap.String("account_id", ev.AccountID),
			zap.Error(err))
		time.Sleep(100 * time.Millisecond)
		if _, err := p.js.Publish(subject, data, opts...); err != nil {
			p.log.Warn("account event publish failed after retry",
				zap.String("subject", subject),
				zap.String("account_id", ev.AccountID),
				zap.Error(err))
		}
	}
}

var accountEventsStreamEnsured atomic.Bool
var streamEnsureFailures atomic.Int32
var lastStreamEnsureFailure atomic.Int64 // unix timestamp of last failure; 0 means never failed

func tryEnsureAccountEventsStream(js natsgo.JetStreamContext, log *zap.Logger) {
	// Already ensured — fast path.
	if accountEventsStreamEnsured.Load() {
		return
	}
	// H3: Reset failure counter after 5 minutes of recovery to prevent
	// permanent deadlock where NATS recovers but the breaker never resets.
	failures := streamEnsureFailures.Load()
	if failures >= 5 {
		if last := lastStreamEnsureFailure.Load(); last > 0 {
			if time.Since(time.Unix(last, 0)) > 5*time.Minute {
				streamEnsureFailures.Store(0)
				failures = 0
			}
		}
		if failures >= 5 {
			return
		}
	}
	// CAS ensures only one goroutine attempts creation.
	if !accountEventsStreamEnsured.CompareAndSwap(false, true) {
		return
	}
	_, err := js.StreamInfo("ACCOUNT_EVENTS")
	if err == nil {
		streamEnsureFailures.Store(0)
		return
	}
	_, err = js.AddStream(&natsgo.StreamConfig{
		Name:      "ACCOUNT_EVENTS",
		Subjects:  []string{"account.>"},
		Retention: natsgo.InterestPolicy,
		MaxAge:    24 * time.Hour,
	})
	if err != nil {
		log.Warn("mdgateway: add ACCOUNT_EVENTS stream failed", zap.Error(err))
		streamEnsureFailures.Add(1)
		lastStreamEnsureFailure.Store(time.Now().Unix())
		accountEventsStreamEnsured.Store(false)
		return
	}
	streamEnsureFailures.Store(0)
}

// PublishConnect publishes an account.connect.<accountID> event.
func (p *AccountEventPublisher) PublishConnect(ctx context.Context, accountID, userID string) {
	p.publish(ctx, SubjectAccountConnect+"."+accountID, &AccountEvent{
		AccountID: accountID, UserID: userID, Timestamp: time.Now(),
	})
}

// PublishDisconnect publishes an account.disconnect.<accountID> event.
func (p *AccountEventPublisher) PublishDisconnect(ctx context.Context, accountID, userID string) {
	p.publish(ctx, SubjectAccountDisconnect+"."+accountID, &AccountEvent{
		AccountID: accountID, UserID: userID, Timestamp: time.Now(),
	})
}

// PublishReconnect publishes an account.reconnect.<accountID> event.
func (p *AccountEventPublisher) PublishReconnect(ctx context.Context, accountID, userID string) {
	p.publish(ctx, SubjectAccountReconnect+"."+accountID, &AccountEvent{
		AccountID: accountID, UserID: userID, Timestamp: time.Now(),
	})
}
