package connect

import (
	"context"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	v1 "anttrader/gen/proto"
)

func (s *TradingService) getCachedProto(ctx context.Context, key string, msg proto.Message) (bool, error) {
	if s == nil || s.streamSvc == nil || s.streamSvc.redisClient == nil || key == "" || msg == nil {
		return false, nil
	}
	b, err := s.streamSvc.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return false, nil
	}
	if err := protojson.Unmarshal(b, msg); err != nil {
		return false, err
	}
	return true, nil
}

func (s *TradingService) setCachedProto(ctx context.Context, key string, msg proto.Message, ttl time.Duration) {
	if s == nil || s.streamSvc == nil || s.streamSvc.redisClient == nil || key == "" || msg == nil {
		return
	}
	b, err := protojson.Marshal(msg)
	if err != nil {
		return
	}
	_ = s.streamSvc.redisClient.Set(ctx, key, b, ttl).Err()
}

func (s *TradingService) publishTradeCommand(accountID string, ev *v1.TradeCommandEvent) {
	if s == nil || s.streamSvc == nil || ev == nil {
		return
	}
	s.streamSvc.publishTradeCommandEvent(accountID, ev)
}

func (s *TradingService) publishTradeReceipt(accountID string, ev *v1.TradeReceiptEvent) {
	if s == nil || s.streamSvc == nil || ev == nil {
		return
	}
	s.streamSvc.publishTradeReceiptEvent(accountID, ev)
}
