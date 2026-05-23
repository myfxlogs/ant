package connect

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/pkg/logger"
	"go.uber.org/zap"
)

func (s *StreamService) eventStreamKey(accountID string) string {
	return "antrader:events:account:" + accountID
}

func (s *StreamService) EventStreamKey(accountID string) string {
	return s.eventStreamKey(accountID)
}

func (s *StreamService) latestStreamEventID(ctx context.Context, accountID string) string {
	if s == nil || s.redisClient == nil || accountID == "" {
		return ""
	}
	// XREVRANGE <key> + - COUNT 1
	res, err := s.redisClient.XRevRangeN(ctx, s.eventStreamKey(accountID), "+", "-", 1).Result()
	if err != nil {
		return ""
	}
	if len(res) == 0 {
		return ""
	}
	return res[0].ID
}

func (s *StreamService) LatestStreamEventID(ctx context.Context, accountID string) string {
	return s.latestStreamEventID(ctx, accountID)
}

func (s *StreamService) currentFence(accountID string) int64 {
	if s == nil || accountID == "" {
		return 0
	}
	if s.sessionLeader == nil {
		return 0
	}
	s.supervisorsMu.Lock()
	lease := s.leaderLeases[accountID]
	s.supervisorsMu.Unlock()
	if lease == nil {
		return 0
	}
	return lease.Fence
}

func (s *StreamService) appendStreamEvent(ctx context.Context, accountID string, eventType string, sev *v1.StreamEvent) (string, error) {
	if s == nil || s.redisClient == nil || accountID == "" || sev == nil {
		return "", nil
	}
	if !s.isLeaderForAccount(accountID) {
		return "", nil
	}
	if sev.Timestamp == nil {
		sev.Timestamp = timestamppb.Now()
	}
	if sev.AccountId == "" {
		sev.AccountId = accountID
	}
	if sev.Type == "" {
		sev.Type = eventType
	}

	blob, err := protojson.Marshal(sev)
	if err != nil {
		return "", err
	}

	fence := s.currentFence(accountID)
	ts := time.Now().UnixMilli()
	args := &redis.XAddArgs{
		Stream: s.eventStreamKey(accountID),
		ID:     "*",
		Values: map[string]any{
			"type":       eventType,
			"account_id": accountID,
			"fence":      strconv.FormatInt(fence, 10),
			"ts":         strconv.FormatInt(ts, 10),
			"event":      string(blob),
		},
	}

	id, err := s.redisClient.XAdd(ctx, args).Result()
	if err != nil {
		return "", err
	}
	return id, nil
}

func streamCursorForXRead(lastID string) string {
	// XREAD returns entries with IDs greater than the provided ID.
	// If no cursor, use "$" to read only new events.
	if lastID == "" {
		return "$"
	}
	return lastID
}

func parseStreamEventFromValues(entryID string, values map[string]any) (*v1.StreamEvent, error) {
	if len(values) == 0 {
		return nil, nil
	}
	raw, ok := values["event"]
	if !ok {
		return nil, nil
	}
	blob, ok := raw.(string)
	if !ok {
		return nil, nil
	}
	sev := &v1.StreamEvent{}
	if err := protojson.Unmarshal([]byte(blob), sev); err != nil {
		return nil, err
	}
	if sev.EventId == "" {
		sev.EventId = entryID
	}
	if sev.Timestamp == nil {
		if v, ok := values["ts"].(string); ok {
			if ms, err := strconv.ParseInt(v, 10, 64); err == nil && ms > 0 {
				sev.Timestamp = timestamppb.New(time.UnixMilli(ms))
			}
		}
	}
	if sev.AccountId == "" {
		if v, ok := values["account_id"].(string); ok {
			sev.AccountId = v
		}
	}
	if sev.Type == "" {
		if v, ok := values["type"].(string); ok {
			sev.Type = v
		}
	}
	return sev, nil
}

func (s *StreamService) readAccountStreamLoop(ctx context.Context, accountID string, startAfter string, out chan<- *v1.StreamEvent) error {
	if s == nil || s.redisClient == nil {
		return nil
	}
	if accountID == "" {
		return nil
	}

	cursor := streamCursorForXRead(startAfter)

	lastErrLog := time.Time{}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		res, err := s.redisClient.XRead(ctx, &redis.XReadArgs{
			Streams: []string{s.eventStreamKey(accountID), cursor},
			Count:   128,
			Block:   5 * time.Second,
		}).Result()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			if errors.Is(err, redis.Nil) {
				continue
			}
			now := time.Now()
			if lastErrLog.IsZero() || now.Sub(lastErrLog) >= 5*time.Second {
				lastErrLog = now
				logger.Warn("redis streams XREAD error",
					zap.String("account_id", accountID),
					zap.String("cursor", cursor),
					zap.Error(err))
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
			continue
		}

		for _, sr := range res {
			for _, msg := range sr.Messages {
				sev, perr := parseStreamEventFromValues(msg.ID, msg.Values)
				if perr != nil || sev == nil {
					cursor = msg.ID
					continue
				}
				cursor = msg.ID
				select {
				case <-ctx.Done():
					return ctx.Err()
				case out <- sev:
				default:
					// Backpressure: best-effort drop to keep loop alive.
				}
			}
		}
	}
}
