package connect

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"anttrader/internal/coordination"
	v1 "anttrader/gen/proto"
)

func TestRedisStreams_LeaderFailover_NoDualWrite(t *testing.T) {
	addr := "localhost:6379"
	rc := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rc.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available at %s: %v", addr, err)
	}

	accountID := "it-account-leader"
	streamKey := "antrader:events:account:" + accountID
	_ = rc.Del(ctx, streamKey).Err()

	mkSvc := func(instanceID string) *StreamService {
		return &StreamService{
			redisClient:  rc,
			sessionLeader: coordination.NewSessionLeader(rc, instanceID),
			leaderLeases: make(map[string]*coordination.Lease),
			instanceID:   instanceID,
		}
	}

	s1 := mkSvc("i1")
	s2 := mkSvc("i2")

	ev1 := &v1.StreamEvent{Type: "profit_update", AccountId: accountID}
	ev2 := &v1.StreamEvent{Type: "profit_update", AccountId: accountID}

	lease1, ok1, err := s1.sessionLeader.TryAcquire(ctx, accountID)
	if err != nil {
		t.Fatalf("try acquire s1: %v", err)
	}
	if !ok1 {
		t.Fatalf("expected s1 acquired")
	}
	s1.leaderLeases[accountID] = lease1
	defer s1.sessionLeader.Release(context.Background(), lease1)

	id, err := s1.appendStreamEvent(ctx, accountID, "profit_update", ev1)
	if err != nil {
		t.Fatalf("s1 append: %v", err)
	}
	if id == "" {
		t.Fatalf("expected non-empty id")
	}

	id2, err := s2.appendStreamEvent(ctx, accountID, "profit_update", ev1)
	if err != nil {
		t.Fatalf("s2 append: %v", err)
	}
	if id2 != "" {
		t.Fatalf("expected s2 not to write as non-leader, got id=%q", id2)
	}

	s1.sessionLeader.Release(context.Background(), lease1)
	s1.leaderLeases[accountID] = nil
	delete(s1.leaderLeases, accountID)

	lease2, ok2, err := s2.sessionLeader.TryAcquire(ctx, accountID)
	if err != nil {
		t.Fatalf("try acquire s2: %v", err)
	}
	if !ok2 {
		t.Fatalf("expected s2 acquired")
	}
	s2.leaderLeases[accountID] = lease2
	defer s2.sessionLeader.Release(context.Background(), lease2)

	id3, err := s2.appendStreamEvent(ctx, accountID, "profit_update", ev2)
	if err != nil {
		t.Fatalf("s2 append2: %v", err)
	}
	if id3 == "" {
		t.Fatalf("expected non-empty id3")
	}

	// Verify exactly 2 entries exist.
	res, err := rc.XRangeN(ctx, streamKey, "-", "+", 10).Result()
	if err != nil {
		t.Fatalf("xrange: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 stream entries, got=%d", len(res))
	}
}

func TestRedisStreams_LeaderFailover_NoEventLoss(t *testing.T) {
	addr := "localhost:6379"
	rc := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err := rc.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available at %s: %v", addr, err)
	}

	accountID := "it-account-failover"
	streamKey := "antrader:events:account:" + accountID
	_ = rc.Del(ctx, streamKey).Err()

	mkSvc := func(instanceID string) *StreamService {
		return &StreamService{
			redisClient:  rc,
			sessionLeader: coordination.NewSessionLeader(rc, instanceID),
			leaderLeases: make(map[string]*coordination.Lease),
			instanceID:   instanceID,
		}
	}

	s1 := mkSvc("i1")
	s2 := mkSvc("i2")

	lease1, ok1, err := s1.sessionLeader.TryAcquire(ctx, accountID)
	if err != nil {
		t.Fatalf("try acquire s1: %v", err)
	}
	if !ok1 {
		t.Fatalf("expected s1 acquired")
	}
	s1.leaderLeases[accountID] = lease1
	defer s1.sessionLeader.Release(context.Background(), lease1)

	for i := 0; i < 3; i++ {
		sev := &v1.StreamEvent{Type: "profit_update", AccountId: accountID}
		id, err := s1.appendStreamEvent(ctx, accountID, "profit_update", sev)
		if err != nil {
			t.Fatalf("s1 append %d: %v", i, err)
		}
		if id == "" {
			t.Fatalf("expected non-empty id")
		}
	}

	// Failover: release lease1, acquire lease2.
	s1.sessionLeader.Release(context.Background(), lease1)
	delete(s1.leaderLeases, accountID)

	lease2, ok2, err := s2.sessionLeader.TryAcquire(ctx, accountID)
	if err != nil {
		t.Fatalf("try acquire s2: %v", err)
	}
	if !ok2 {
		t.Fatalf("expected s2 acquired")
	}
	s2.leaderLeases[accountID] = lease2
	defer s2.sessionLeader.Release(context.Background(), lease2)

	for i := 0; i < 2; i++ {
		sev := &v1.StreamEvent{Type: "profit_update", AccountId: accountID}
		id, err := s2.appendStreamEvent(ctx, accountID, "profit_update", sev)
		if err != nil {
			t.Fatalf("s2 append %d: %v", i, err)
		}
		if id == "" {
			t.Fatalf("expected non-empty id")
		}
	}

	res, err := rc.XRangeN(ctx, streamKey, "-", "+", 10).Result()
	if err != nil {
		t.Fatalf("xrange: %v", err)
	}
	if len(res) != 5 {
		t.Fatalf("expected 5 stream entries, got=%d", len(res))
	}
}

func TestRedisStreams_SubscribeLikeReader_Failover_NoLossNoDup(t *testing.T) {
	addr := "localhost:6379"
	rc := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	if err := rc.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available at %s: %v", addr, err)
	}

	accountID := "it-account-failover-reader"
	streamKey := "antrader:events:account:" + accountID
	_ = rc.Del(ctx, streamKey).Err()

	mkSvc := func(instanceID string) *StreamService {
		return &StreamService{
			redisClient:   rc,
			sessionLeader: coordination.NewSessionLeader(rc, instanceID),
			leaderLeases:  make(map[string]*coordination.Lease),
			instanceID:    instanceID,
		}
	}

	s1 := mkSvc("i1")
	s2 := mkSvc("i2")

	outCh := make(chan *v1.StreamEvent, 64)
	readCtx, readCancel := context.WithCancel(ctx)
	defer readCancel()

	var readWG sync.WaitGroup
	readWG.Add(1)
	go func() {
		defer readWG.Done()
		_ = s1.readAccountStreamLoop(readCtx, accountID, "0", outCh)
	}()

	lease1, ok1, err := s1.sessionLeader.TryAcquire(ctx, accountID)
	if err != nil {
		t.Fatalf("try acquire s1: %v", err)
	}
	if !ok1 {
		t.Fatalf("expected s1 acquired")
	}
	s1.leaderLeases[accountID] = lease1

	for i := 0; i < 3; i++ {
		sev := &v1.StreamEvent{Type: "profit_update", AccountId: accountID}
		id, err := s1.appendStreamEvent(ctx, accountID, "profit_update", sev)
		if err != nil {
			t.Fatalf("s1 append %d: %v", i, err)
		}
		if id == "" {
			t.Fatalf("expected non-empty id")
		}
	}

	// Failover: release lease1, acquire lease2.
	s1.sessionLeader.Release(context.Background(), lease1)
	delete(s1.leaderLeases, accountID)

	lease2, ok2, err := s2.sessionLeader.TryAcquire(ctx, accountID)
	if err != nil {
		t.Fatalf("try acquire s2: %v", err)
	}
	if !ok2 {
		t.Fatalf("expected s2 acquired")
	}
	s2.leaderLeases[accountID] = lease2
	defer s2.sessionLeader.Release(context.Background(), lease2)

	for i := 0; i < 2; i++ {
		sev := &v1.StreamEvent{Type: "profit_update", AccountId: accountID}
		id, err := s2.appendStreamEvent(ctx, accountID, "profit_update", sev)
		if err != nil {
			t.Fatalf("s2 append %d: %v", i, err)
		}
		if id == "" {
			t.Fatalf("expected non-empty id")
		}
	}

	seen := make(map[string]struct{}, 8)
	ids := make([]string, 0, 5)
	for len(ids) < 5 {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for reader events, got=%d", len(ids))
		case ev := <-outCh:
			if ev == nil || ev.EventId == "" {
				continue
			}
			if _, ok := seen[ev.EventId]; ok {
				t.Fatalf("duplicate event_id observed: %s", ev.EventId)
			}
			seen[ev.EventId] = struct{}{}
			ids = append(ids, ev.EventId)
		}
	}

	// Stream IDs must be strictly increasing in read order.
	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Fatalf("expected increasing event_id order, ids[%d]=%s ids[%d]=%s", i-1, ids[i-1], i, ids[i])
		}
	}

	_ = streamKey
}
