package connect

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	v1 "anttrader/gen/proto"
	"anttrader/pkg/logger"
	"go.uber.org/zap"
)

func (s *StreamService) readHistoryEventsFromRedis(ctx context.Context, accountID string, startAfter string, limit int) ([]*v1.StreamEvent, string, error) {
	if s == nil || s.redisClient == nil {
		return nil, startAfter, nil
	}
	if accountID == "" || limit <= 0 {
		return nil, startAfter, nil
	}

	cursor := startAfter
	if cursor == "" {
		cursor = "0-0"
	}

	res := make([]*v1.StreamEvent, 0, limit)
	for len(res) < limit {
		count := 128
		remain := limit - len(res)
		if remain < count {
			count = remain
		}
		minID := "-"
		if cursor != "" && cursor != "0-0" {
			minID = "(" + cursor
		}
		msgs, err := s.redisClient.XRangeN(ctx, s.eventStreamKey(accountID), minID, "+", int64(count)).Result()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return res, cursor, err
			}
			return res, cursor, err
		}
		if len(msgs) == 0 {
			return res, cursor, nil
		}
		for _, msg := range msgs {
			cursor = msg.ID
			sev, perr := parseStreamEventFromValues(msg.ID, msg.Values)
			if perr != nil || sev == nil {
				continue
			}
			res = append(res, sev)
			if len(res) >= limit {
				break
			}
		}
	}
	return res, cursor, nil
}

func (s *StreamService) ReadHistoryEventsForTest(ctx context.Context, accountID string, startAfter string, limit int) ([]*v1.StreamEvent, string, error) {
	return s.readHistoryEventsFromRedis(ctx, accountID, startAfter, limit)
}

func (s *StreamService) SubscribeHistory(ctx context.Context, req *connect.Request[v1.SubscribeHistoryRequest], stream *connect.ServerStream[v1.StreamEvent]) error {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	releaseSlot, err := s.acquireUserStreamSlot(userID.String())
	if err != nil {
		return connect.NewError(connect.CodeResourceExhausted, err)
	}
	defer releaseSlot()

	accountIDs := req.Msg.GetAccountIds()
	if len(accountIDs) == 0 {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("errors.stream.account_ids_required"))
	}

	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 1000
	}

	curByAccount := make(map[string]string)
	for k, v := range req.Msg.GetLastEventIds() {
		if k == "" {
			continue
		}
		curByAccount[k] = v
	}

	// Authorization + ensure streams
	for _, accountIDStr := range accountIDs {
		_, aerr := s.authorizeAccountStream(ctx, userID, accountIDStr)
		if aerr != nil {
			return aerr
		}
	}

	sent := 0
	for _, accountIDStr := range accountIDs {
		if sent >= limit {
			break
		}
		startAfter := curByAccount[accountIDStr]
		batch, _, berr := s.readHistoryEventsFromRedis(ctx, accountIDStr, startAfter, limit-sent)
		if berr != nil {
			if errors.Is(berr, context.Canceled) {
				return berr
			}
			return connect.NewError(connect.CodeUnavailable, fmt.Errorf("redis read history error: %w", berr))
		}
		for _, sev := range batch {
			if sev == nil {
				continue
			}
			if stream == nil {
				return connect.NewError(connect.CodeInternal, errors.New("nil stream"))
			}
			if err := stream.Send(sev); err != nil {
				if isCanceledErr(err) {
					return err
				}
				logger.Warn("SubscribeHistory send error", zap.Error(err))
				return err
			}
			sent++
			if sent >= limit {
				break
			}
		}
		// small yield to avoid long single-account starvation (best-effort)
		time.Sleep(1 * time.Millisecond)
	}

	return nil
}
