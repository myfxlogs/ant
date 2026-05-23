package connect

import (
	"context"
	"encoding/json"

	"anttrader/internal/coordination"

	"github.com/redis/go-redis/v9"
)

func (s *StreamService) SetSessionLeader(rdb *redis.Client, instanceID string) {
	if s == nil {
		return
	}
	if rdb == nil || instanceID == "" {
		s.sessionLeader = nil
		s.eventBus = nil
		s.redisClient = nil
		return
	}
	s.sessionLeader = coordination.NewSessionLeader(rdb, instanceID)
	s.eventBus = coordination.NewEventBus(rdb)
	s.redisClient = rdb
	s.instanceID = instanceID

	// Subscribe wakeup channel once per process: when any instance receives a subscription
	// on a non-leader, it can notify the cluster to attempt acquiring leadership.
	s.wakeupOnce.Do(func() {
		if s.goroutineMgr != nil {
			_, _ = s.goroutineMgr.Spawn("leader-wakeup-listener", func(gctx context.Context) error {
				ps := rdb.Subscribe(gctx, "antrader:session_leader:wakeup")
				defer ps.Close()
				if _, err := ps.Receive(gctx); err != nil {
					return err
				}
				ch := ps.Channel()
				for {
					select {
					case <-gctx.Done():
						return gctx.Err()
					case msg, ok := <-ch:
						if !ok || msg == nil {
							continue
						}
						var payload struct {
							AccountID string `json:"account_id"`
						}
						if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
							continue
						}
						if payload.AccountID == "" {
							continue
						}
						s.ensureSupervisor(payload.AccountID)
					}
				}
			})
		} else {
			go func() {
				ps := rdb.Subscribe(context.Background(), "antrader:session_leader:wakeup")
				defer ps.Close()
				_, _ = ps.Receive(context.Background())
				ch := ps.Channel()
				for msg := range ch {
					var payload struct {
						AccountID string `json:"account_id"`
					}
					if msg == nil {
						continue
					}
					if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
						continue
					}
					if payload.AccountID == "" {
						continue
					}
					s.ensureSupervisor(payload.AccountID)
				}
			}()
		}
	})
}

func (s *StreamService) publishWakeup(accountID string) {
	if s == nil || s.redisClient == nil {
		return
	}
	payload, err := json.Marshal(map[string]string{"account_id": accountID})
	if err != nil {
		return
	}
	// best-effort; ignore errors
	_ = s.redisClient.Publish(context.Background(), "antrader:session_leader:wakeup", payload).Err()
}
