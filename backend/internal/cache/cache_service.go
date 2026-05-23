package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"anttrader/internal/config"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

type CacheService struct {
	redisClient *redis.Client
	config      *config.RedisConfig
}

type CacheOptions struct {
	TTL      time.Duration
	Category string // 用于分类管理缓存
}

func NewCacheService(cfg *config.RedisConfig) (*CacheService, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &CacheService{
		redisClient: rdb,
		config:      cfg,
	}, nil
}

func NewCacheServiceWithClient(rdb *redis.Client, cfg *config.RedisConfig) *CacheService {
	return &CacheService{
		redisClient: rdb,
		config:      cfg,
	}
}

func (cs *CacheService) Client() *redis.Client {
	return cs.redisClient
}

func (cs *CacheService) Set(ctx context.Context, key string, value interface{}, opts CacheOptions) error {
	// 构建完整的键名
	fullKey := cs.buildKey(key, opts.Category)
	
	// 序列化值
	data, err := json.Marshal(value)
	if err != nil {
		logger.Error("Failed to marshal cache value", 
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	// 设置缓存
	if err := cs.redisClient.Set(ctx, fullKey, data, opts.TTL).Err(); err != nil {
		logger.Error("Failed to set cache",
			zap.String("key", key),
			zap.Duration("ttl", opts.TTL),
			zap.Error(err))
		return err
	}

	return nil
}

func (cs *CacheService) Get(ctx context.Context, key string, dest interface{}, opts CacheOptions) error {
	fullKey := cs.buildKey(key, opts.Category)
	
	data, err := cs.redisClient.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache miss")
		}
		logger.Error("Failed to get cache", zap.String("key", key), zap.Error(err))
		return err
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		logger.Error("Failed to unmarshal cache value",
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	return nil
}

func (cs *CacheService) Delete(ctx context.Context, key string, opts CacheOptions) error {
	fullKey := cs.buildKey(key, opts.Category)
	
	if err := cs.redisClient.Del(ctx, fullKey).Err(); err != nil {
		logger.Error("Failed to delete cache", zap.String("key", key), zap.Error(err))
		return err
	}

	return nil
}

func (cs *CacheService) DeletePattern(ctx context.Context, pattern string, opts CacheOptions) error {
	fullPattern := cs.buildKey(pattern, opts.Category)
	
	keys, err := cs.redisClient.Keys(ctx, fullPattern).Result()
	if err != nil {
		logger.Error("Failed to get keys for pattern", 
			zap.String("pattern", pattern),
			zap.Error(err))
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	if err := cs.redisClient.Del(ctx, keys...).Err(); err != nil {
		logger.Error("Failed to delete keys by pattern",
			zap.String("pattern", pattern),
			zap.Int("count", len(keys)),
			zap.Error(err))
		return err
	}

	return nil
}

func (cs *CacheService) Exists(ctx context.Context, key string, opts CacheOptions) (bool, error) {
	fullKey := cs.buildKey(key, opts.Category)
	
	count, err := cs.redisClient.Exists(ctx, fullKey).Result()
	if err != nil {
		logger.Error("Failed to check cache existence", zap.String("key", key), zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

func (cs *CacheService) SetWithTags(ctx context.Context, key string, value interface{}, opts CacheOptions, tags []string) error {
	// 设置缓存
	if err := cs.Set(ctx, key, value, opts); err != nil {
		return err
	}

	// 为每个标签添加键的引用
	for _, tag := range tags {
		tagKey := fmt.Sprintf("tag:%s", tag)
		if err := cs.redisClient.SAdd(ctx, tagKey, cs.buildKey(key, opts.Category)).Err(); err != nil {
			logger.Error("Failed to add tag reference",
				zap.String("key", key),
				zap.String("tag", tag),
				zap.Error(err))
			return err
		}
		
		// 设置标签的过期时间
		cs.redisClient.Expire(ctx, tagKey, opts.TTL)
	}

	return nil
}

func (cs *CacheService) InvalidateByTag(ctx context.Context, tag string) error {
	tagKey := fmt.Sprintf("tag:%s", tag)
	
	keys, err := cs.redisClient.SMembers(ctx, tagKey).Result()
	if err != nil {
		logger.Error("Failed to get tagged keys",
			zap.String("tag", tag),
			zap.Error(err))
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	// 删除所有标记的键
	if err := cs.redisClient.Del(ctx, keys...).Err(); err != nil {
		logger.Error("Failed to invalidate tagged keys",
			zap.String("tag", tag),
			zap.Int("count", len(keys)),
			zap.Error(err))
		return err
	}

	// 删除标签集合
	cs.redisClient.Del(ctx, tagKey)

	return nil
}

// 缓存统计
func (cs *CacheService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := cs.redisClient.Info(ctx).Result()
	if err != nil {
		return nil, err
	}

	// 解析Redis信息
	stats := make(map[string]interface{})
	stats["redis_info"] = info
	
	// 获取内存使用
	memInfo := cs.redisClient.Info(ctx, "memory").Val()
	stats["memory_info"] = memInfo

	return stats, nil
}

// 预热缓存
func (cs *CacheService) Warmup(ctx context.Context, warmupFunc func(ctx context.Context, cs *CacheService) error) error {
	if err := warmupFunc(ctx, cs); err != nil {
		logger.Error("Cache warmup failed", zap.Error(err))
		return err
	}

	return nil
}

// 清理过期缓存
func (cs *CacheService) Cleanup(ctx context.Context) error {
	// Redis会自动清理过期的键，这里主要是清理标签引用
	pattern := "tag:*"
	iter := cs.redisClient.Scan(ctx, 0, pattern, 100).Iterator()
	
	cleaned := 0
	for iter.Next(ctx) {
		tagKey := iter.Val()
		members, err := cs.redisClient.SMembers(ctx, tagKey).Result()
		if err != nil {
			continue
		}

		// 检查每个键是否还存在
		validKeys := make([]string, 0)
		for _, key := range members {
			exists, _ := cs.redisClient.Exists(ctx, key).Result()
			if exists > 0 {
				validKeys = append(validKeys, key)
			}
		}

		if len(validKeys) == 0 {
			// 删除空的标签集合
			cs.redisClient.Del(ctx, tagKey)
			cleaned++
		} else if len(validKeys) != len(members) {
			// 更新标签集合，移除无效的键
			cs.redisClient.Del(ctx, tagKey)
			if len(validKeys) > 0 {
				args := make([]interface{}, 0, len(validKeys))
				for _, k := range validKeys {
					args = append(args, k)
				}
				cs.redisClient.SAdd(ctx, tagKey, args...)
			}
		}
	}

	return nil
}

func (cs *CacheService) Close() error {
	return cs.redisClient.Close()
}

func (cs *CacheService) buildKey(key, category string) string {
	if category != "" {
		return fmt.Sprintf("antrader:cache:%s:%s", category, key)
	}
	return fmt.Sprintf("antrader:cache:%s", key)
}

// 预定义的缓存选项
var (
	// 用户相关缓存
	UserCacheOpts = CacheOptions{
		TTL:      time.Minute * 30,
		Category: "user",
	}

	// 账户相关缓存
	AccountCacheOpts = CacheOptions{
		TTL:      time.Minute * 10,
		Category: "account",
	}

	// 市场数据缓存
	MarketCacheOpts = CacheOptions{
		TTL:      time.Second * 30,
		Category: "market",
	}

	// K线数据缓存
	KlineCacheOpts = CacheOptions{
		TTL:      time.Minute * 5,
		Category: "kline",
	}

	// 分析数据缓存
	AnalyticsCacheOpts = CacheOptions{
		TTL:      time.Minute * 15,
		Category: "analytics",
	}

	// 系统配置缓存
	ConfigCacheOpts = CacheOptions{
		TTL:      time.Hour,
		Category: "config",
	}
)
