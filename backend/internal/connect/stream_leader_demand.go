package connect

import (
	"context"
	"time"
)

func (s *StreamService) demandKey(accountID string) string {
	return "antrader:session_leader:demand:account:" + accountID + ":instance:" + s.instanceID
}

func (s *StreamService) setDemand(accountID string, on bool) {
	if s == nil || s.redisClient == nil || accountID == "" {
		return
	}
	if s.instanceID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if on {
		// TTL makes it self-heal if an instance crashes without cleanup.
		_ = s.redisClient.Set(ctx, s.demandKey(accountID), "1", 3*time.Minute).Err()
		return
	}
	_ = s.redisClient.Del(ctx, s.demandKey(accountID)).Err()
}

func (s *StreamService) hasDemand(accountID string) bool {
	if s == nil || s.redisClient == nil || accountID == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pattern := "antrader:session_leader:demand:account:" + accountID + ":instance:*"
	iter := s.redisClient.Scan(ctx, 0, pattern, 1).Iterator()
	for iter.Next(ctx) {
		return true
	}
	return false
}
