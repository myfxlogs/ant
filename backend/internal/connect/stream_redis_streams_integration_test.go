package connect

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/encoding/protojson"

	v1 "anttrader/gen/proto"
)

func TestReadAccountStreamLoop_Resume(t *testing.T) {
	addr := "localhost:6379"
	rc := redis.NewClient(&redis.Options{Addr: addr})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rc.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available at %s: %v", addr, err)
	}

	s := &StreamService{redisClient: rc}
	accountID := "it-account"
	key := s.eventStreamKey(accountID)
	_ = rc.Del(ctx, key).Err()

	mkBlob := func(typ string) string {
		sev := &v1.StreamEvent{Type: typ, AccountId: accountID}
		b, err := protojson.Marshal(sev)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		return string(b)
	}

	id1, err := rc.XAdd(ctx, &redis.XAddArgs{Stream: key, ID: "*", Values: map[string]any{"type": "t1", "account_id": accountID, "ts": "1700000000000", "event": mkBlob("t1")}}).Result()
	if err != nil {
		t.Fatalf("xadd1: %v", err)
	}
	id2, err := rc.XAdd(ctx, &redis.XAddArgs{Stream: key, ID: "*", Values: map[string]any{"type": "t2", "account_id": accountID, "ts": "1700000000001", "event": mkBlob("t2")}}).Result()
	if err != nil {
		t.Fatalf("xadd2: %v", err)
	}
	id3, err := rc.XAdd(ctx, &redis.XAddArgs{Stream: key, ID: "*", Values: map[string]any{"type": "t3", "account_id": accountID, "ts": "1700000000002", "event": mkBlob("t3")}}).Result()
	if err != nil {
		t.Fatalf("xadd3: %v", err)
	}

	out := make(chan *v1.StreamEvent, 16)
	readCtx, readCancel := context.WithCancel(context.Background())
	defer readCancel()
	go func() { _ = s.readAccountStreamLoop(readCtx, accountID, "0-0", out) }()

	seen := map[string]bool{}
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for len(seen) < 3 {
		select {
		case ev := <-out:
			if ev != nil {
				seen[ev.EventId] = true
			}
		case <-deadline.C:
			t.Fatalf("timeout waiting initial events, seen=%v", seen)
		}
	}
	if !seen[id1] || !seen[id2] || !seen[id3] {
		t.Fatalf("missing events, expected ids=%q,%q,%q seen=%v", id1, id2, id3, seen)
	}

	readCancel()

	out2 := make(chan *v1.StreamEvent, 16)
	readCtx2, readCancel2 := context.WithCancel(context.Background())
	defer readCancel2()
	go func() { _ = s.readAccountStreamLoop(readCtx2, accountID, id2, out2) }()

	select {
	case ev := <-out2:
		if ev == nil {
			t.Fatalf("expected event")
		}
		if ev.EventId != id3 {
			t.Fatalf("expected to resume after %q and get %q, got %q", id2, id3, ev.EventId)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting resume event")
	}
}
