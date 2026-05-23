package service

import (
	"context"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DynamicConfigService struct {
	db    *pgxpool.Pool
	cache sync.Map
}

func NewDynamicConfigService(db *pgxpool.Pool) *DynamicConfigService {
	return &DynamicConfigService{db: db}
}

type ConfigValue struct {
	Value   string
	Enabled bool
}

func (s *DynamicConfigService) GetConfig(ctx context.Context, key string) (*ConfigValue, error) {
	if cached, ok := s.cache.Load(key); ok {
		return cached.(*ConfigValue), nil
	}

	return s.loadConfig(ctx, key)
}

func (s *DynamicConfigService) loadConfig(ctx context.Context, key string) (*ConfigValue, error) {
	query := `SELECT value, COALESCE(enabled, true) as enabled FROM system_config WHERE key = $1`

	var value string
	var enabled bool
	err := s.db.QueryRow(ctx, query, key).Scan(&value, &enabled)
	if err != nil {
		return nil, err
	}

	cv := &ConfigValue{Value: value, Enabled: enabled}
	s.cache.Store(key, cv)
	return cv, nil
}

func (s *DynamicConfigService) GetInt(ctx context.Context, key string, defaultValue int) (int, bool, error) {
	cv, err := s.GetConfig(ctx, key)
	if err != nil {
		return defaultValue, true, nil
	}

	if !cv.Enabled {
		return defaultValue, false, nil
	}

	val, err := strconv.Atoi(cv.Value)
	if err != nil {
		return defaultValue, true, nil
	}
	return val, true, nil
}

func (s *DynamicConfigService) GetInt64(ctx context.Context, key string, defaultValue int64) (int64, bool, error) {
	cv, err := s.GetConfig(ctx, key)
	if err != nil {
		return defaultValue, true, nil
	}

	if !cv.Enabled {
		return defaultValue, false, nil
	}

	val, err := strconv.ParseInt(cv.Value, 10, 64)
	if err != nil {
		return defaultValue, true, nil
	}
	return val, true, nil
}

func (s *DynamicConfigService) GetFloat64(ctx context.Context, key string, defaultValue float64) (float64, bool, error) {
	cv, err := s.GetConfig(ctx, key)
	if err != nil {
		return defaultValue, true, nil
	}

	if !cv.Enabled {
		return defaultValue, false, nil
	}

	val, err := strconv.ParseFloat(cv.Value, 64)
	if err != nil {
		return defaultValue, true, nil
	}
	return val, true, nil
}

func (s *DynamicConfigService) GetString(ctx context.Context, key string, defaultValue string) (string, bool, error) {
	cv, err := s.GetConfig(ctx, key)
	if err != nil {
		return defaultValue, true, nil
	}

	if !cv.Enabled {
		return defaultValue, false, nil
	}

	return cv.Value, true, nil
}

func (s *DynamicConfigService) InvalidateCache(key string) {
	s.cache.Delete(key)
}

func (s *DynamicConfigService) InvalidateAll() {
	s.cache = sync.Map{}
}

func (s *DynamicConfigService) MaxAccountsPerUser(ctx context.Context) (int, bool) {
	val, enabled, _ := s.GetInt(ctx, "max_accounts_per_user", 10)
	return val, enabled
}
