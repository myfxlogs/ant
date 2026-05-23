package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type ExecutionIdempotencyStore interface {
	GetOrderResponse(ctx context.Context, key string) (*OrderResponse, bool, error)
	SetOrderResponse(ctx context.Context, key string, resp *OrderResponse, ttl time.Duration) error
}

type RedisExecutionIdempotencyStore struct {
	client *redis.Client
}

func NewRedisExecutionIdempotencyStore(client *redis.Client) *RedisExecutionIdempotencyStore {
	return &RedisExecutionIdempotencyStore{client: client}
}

func (s *RedisExecutionIdempotencyStore) GetOrderResponse(ctx context.Context, key string) (*OrderResponse, bool, error) {
	if s == nil || s.client == nil || key == "" {
		return nil, false, nil
	}
	b, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false, nil
	}
	var resp OrderResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, false, err
	}
	return &resp, true, nil
}

func (s *RedisExecutionIdempotencyStore) SetOrderResponse(ctx context.Context, key string, resp *OrderResponse, ttl time.Duration) error {
	if s == nil || s.client == nil || key == "" {
		return nil
	}
	if resp == nil {
		return errors.New("nil response")
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, b, ttl).Err()
}
