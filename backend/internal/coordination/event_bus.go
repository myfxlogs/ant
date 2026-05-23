package coordination

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type EventType string

const (
	EventTypeProfit       EventType = "profit"
	EventTypeOrder        EventType = "order"
	EventTypeAccountStatus EventType = "account_status"
)

type Envelope struct {
	Type EventType        `json:"type"`
	Data json.RawMessage  `json:"data"`
	At   int64            `json:"at"`
}

type EventBus struct {
	rdb    *redis.Client
	prefix string
}

func NewEventBus(rdb *redis.Client) *EventBus {
	return &EventBus{rdb: rdb, prefix: "antrader:eventbus:"}
}

func (b *EventBus) SetPrefix(prefix string) {
	if b == nil {
		return
	}
	if prefix != "" {
		b.prefix = prefix
	}
}

func (b *EventBus) ChannelForAccount(accountID string) string {
	return b.prefix + "account:" + accountID
}

func (b *EventBus) Publish(ctx context.Context, accountID string, typ EventType, data json.RawMessage) error {
	if b == nil || b.rdb == nil {
		return fmt.Errorf("redis event bus not configured")
	}
	if accountID == "" {
		return fmt.Errorf("accountID required")
	}

	env := Envelope{Type: typ, Data: data, At: time.Now().UnixMilli()}
	payload, err := json.Marshal(&env)
	if err != nil {
		return err
	}

	return b.rdb.Publish(ctx, b.ChannelForAccount(accountID), payload).Err()
}

func (b *EventBus) Subscribe(ctx context.Context, accountID string) (*redis.PubSub, <-chan *redis.Message, error) {
	if b == nil || b.rdb == nil {
		return nil, nil, fmt.Errorf("redis event bus not configured")
	}
	if accountID == "" {
		return nil, nil, fmt.Errorf("accountID required")
	}
	ps := b.rdb.Subscribe(ctx, b.ChannelForAccount(accountID))
	// ensure subscription established
	if _, err := ps.Receive(ctx); err != nil {
		_ = ps.Close()
		return nil, nil, err
	}
	return ps, ps.Channel(), nil
}
