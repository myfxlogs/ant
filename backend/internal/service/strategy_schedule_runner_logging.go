package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/pkg/logger"
)

func retryHFSubscribeMT4(ctx context.Context, op string, scheduleID uuid.UUID, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= mt4HFSubscribeRetryAttempts; attempt++ {
		if err := fn(); err == nil {
			if attempt > 1 {
				incrementScheduleMetric(scheduleID, "reconnect_recoveries")
			}
			return nil
		} else {
			lastErr = err
			if !shouldRetryMT4HFSubscribe(err) {
				return err
			}
			if attempt < mt4HFSubscribeRetryAttempts {
				throttledScheduleRunnerWarn("mt4.hf_subscribe_retry."+op+"."+scheduleID.String(), scheduleLogThrottleWindow(),
					"schedule v2: hf subscribe retry", zap.String("op", op), zap.String("schedule_id", scheduleID.String()), zap.Int("attempt", attempt), zap.Error(err))
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(mt4HFSubscribeRetryDelay):
				}
			}
		}
	}
	return lastErr
}

func retryHFSubscribeMT5(ctx context.Context, op string, scheduleID uuid.UUID, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= mt5HFSubscribeRetryAttempts; attempt++ {
		if err := fn(); err == nil {
			if attempt > 1 {
				incrementScheduleMetric(scheduleID, "reconnect_recoveries")
			}
			return nil
		} else {
			lastErr = err
			if !shouldRetryMT5HFSubscribe(err) {
				return err
			}
			if attempt < mt5HFSubscribeRetryAttempts {
				throttledScheduleRunnerWarn("mt5.hf_subscribe_retry."+op+"."+scheduleID.String(), scheduleLogThrottleWindow(),
					"schedule v2: hf subscribe retry", zap.String("op", op), zap.String("schedule_id", scheduleID.String()), zap.Int("attempt", attempt), zap.Error(err))
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(mt5HFSubscribeRetryDelay):
				}
			}
		}
	}
	return lastErr
}

func shouldRetryMT4HFSubscribe(err error) bool {
	if err == nil {
		return false
	}
	s := stringsToUpperTrim(err.Error())
	if containsAny(s, []string{
		"INVALID_TOKEN",
		"INVALID TOKEN",
		"INVALID_ACCOUNT",
		"ACCOUNT_DISABLED",
		"NOT_ENOUGH_RIGHTS",
		"INVALID_PARAM",
	}) {
		return false
	}
	return true
}

func shouldRetryMT5HFSubscribe(err error) bool {
	if err == nil {
		return false
	}
	s := stringsToUpperTrim(err.Error())
	if containsAny(s, []string{
		"INVALID_TOKEN",
		"INVALID TOKEN",
		"INVALID_ACCOUNT",
		"ACCOUNT_DISABLED",
		"INVALID_SYMBOL",
		"INVALID_PARAM",
		"INVALID_REQUEST",
		"NOT_PERMISSION",
	}) {
		return false
	}
	return true
}

func containsAny(s string, parts []string) bool {
	for _, p := range parts {
		if p != "" && stringsContains(s, p) {
			return true
		}
	}
	return false
}

func stringsContains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func shouldLogScheduleRunner(key string, window time.Duration) (bool, int) {
	now := time.Now()
	scheduleRunnerLogThrottleMu.Lock()
	defer scheduleRunnerLogThrottleMu.Unlock()
	if last, ok := scheduleRunnerLogThrottleLast[key]; ok && now.Sub(last) < window {
		scheduleRunnerLogSuppressed[key]++
		return false, 0
	}
	suppressed := scheduleRunnerLogSuppressed[key]
	delete(scheduleRunnerLogSuppressed, key)
	scheduleRunnerLogThrottleLast[key] = now
	cutoff := now.Add(-2 * window)
	for k, ts := range scheduleRunnerLogThrottleLast {
		if ts.Before(cutoff) {
			delete(scheduleRunnerLogThrottleLast, k)
			delete(scheduleRunnerLogSuppressed, k)
		}
	}
	return true, suppressed
}

func throttledScheduleRunnerWarn(key string, window time.Duration, msg string, fields ...zap.Field) {
	if ok, suppressed := shouldLogScheduleRunner(key, window); ok {
		if suppressed > 0 {
			fields = append(fields, zap.Int("suppressed_count", suppressed))
		}
		logger.Warn(msg, fields...)
	}
}

func throttledScheduleRunnerInfo(key string, window time.Duration, msg string, fields ...zap.Field) {
	if ok, suppressed := shouldLogScheduleRunner(key, window); ok {
		if suppressed > 0 {
			fields = append(fields, zap.Int("suppressed_count", suppressed))
		}
		logger.Info(msg, fields...)
	}
}

func incrementScheduleMetric(scheduleID uuid.UUID, metric string) int64 {
	if scheduleID == uuid.Nil || metric == "" {
		return 0
	}
	key := scheduleID.String() + "|" + metric
	scheduleRunnerLogThrottleMu.Lock()
	defer scheduleRunnerLogThrottleMu.Unlock()
	scheduleRunnerLogSuppressed[key]++
	return int64(scheduleRunnerLogSuppressed[key])
}

func snapshotScheduleMetrics(scheduleID uuid.UUID) map[string]int64 {
	out := map[string]int64{}
	if scheduleID == uuid.Nil {
		return out
	}
	prefix := scheduleID.String() + "|"
	scheduleRunnerLogThrottleMu.Lock()
	defer scheduleRunnerLogThrottleMu.Unlock()
	for k, v := range scheduleRunnerLogSuppressed {
		if strings.HasPrefix(k, prefix) {
			out[strings.TrimPrefix(k, prefix)] = int64(v)
		}
	}
	return out
}
