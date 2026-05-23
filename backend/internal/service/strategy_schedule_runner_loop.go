package service

import (
	"context"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"anttrader/internal/model"
	"anttrader/pkg/logger"
)

const (
	triggerModeStableKline           = "stable_kline"
	triggerModeHFQuoteStream         = "hf_quote_stream"
	streamTypeStrategyRuntime        = "strategy"
	mt4HFSubscribeRetryAttempts      = 3
	mt4HFSubscribeRetryDelay         = 500 * time.Millisecond
	mt5HFSubscribeRetryAttempts      = 5
	mt5HFSubscribeRetryDelay         = 700 * time.Millisecond
	defaultScheduleLogThrottleWindow = 12 * time.Second
	defaultScheduleWatchdogInterval  = 30 * time.Second
)

func scheduleLogThrottleWindow() time.Duration {
	v := strings.TrimSpace(os.Getenv("ANTRADER_SCHEDULE_LOG_THROTTLE_WINDOW"))
	if v == "" {
		return defaultScheduleLogThrottleWindow
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return defaultScheduleLogThrottleWindow
	}
	return d
}

func scheduleEventDrivenEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("ANTRADER_SCHEDULE_EVENT_DRIVEN_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func scheduleWatchdogInterval() time.Duration {
	v := strings.TrimSpace(os.Getenv("ANTRADER_SCHEDULE_WATCHDOG_INTERVAL"))
	if v == "" {
		return defaultScheduleWatchdogInterval
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return defaultScheduleWatchdogInterval
	}
	return d
}

func resolveTriggerMode(schedule *model.StrategySchedule, conf map[string]interface{}) string {
	mode := triggerModeStableKline
	if stringsToUpperTrim(schedule.ScheduleType) == stringsToUpperTrim(model.ScheduleTypeEvent) {
		mode = triggerModeHFQuoteStream
	}
	if conf != nil {
		if v, ok := conf["trigger_mode"]; ok {
			if s, ok := v.(string); ok && s != "" {
				mode = s
			}
		}
	}
	return mode
}

func (r *StrategyScheduleRunner) runSchedule(ctx context.Context, schedule *model.StrategySchedule) {
	if r == nil || schedule == nil {
		return
	}

	if r.connMgr != nil {
		r.connMgr.AddSubscription(schedule.AccountID, streamTypeStrategyRuntime)
		defer r.connMgr.RemoveSubscription(schedule.AccountID, streamTypeStrategyRuntime)
	}

	conf, _ := schedule.GetScheduleConfig()
	mode := resolveTriggerMode(schedule, conf)
	logger.Info("schedule runtime started",
		zap.String("schedule_id", schedule.ID.String()),
		zap.String("trigger_mode", mode),
		zap.Bool("event_driven_enabled", scheduleEventDrivenEnabled()),
	)

	switch mode {
	case triggerModeHFQuoteStream:
		// Run one immediate evaluation on startup so newly-enabled schedules
		// do not need to wait for the first cooldown/quote tick.
		r.evalOnceWithSource(ctx, schedule, "startup")
		r.loopHFQuote(ctx, schedule)
	default:
		// Run one immediate evaluation on startup so newly-enabled schedules
		// do not need to wait for a full timeframe interval.
		r.evalOnceWithSource(ctx, schedule, "startup")
		if scheduleEventDrivenEnabled() {
			r.loopStableEventDriven(ctx, schedule)
		} else {
			r.loopStable(ctx, schedule)
		}
	}

	r.stateMu.Lock()
	delete(r.states, schedule.ID)
	store := r.stateStore
	r.stateMu.Unlock()
	if store != nil {
		if err := store.Delete(context.Background(), schedule.ID); err != nil {
			logger.Warn("schedule runtime state delete failed", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
		}
	}

}

func timeframeToDuration(tf string) time.Duration {
	switch stringsToUpperTrim(tf) {
	case "M1":
		return 1 * time.Minute
	case "M5":
		return 5 * time.Minute
	case "M15":
		return 15 * time.Minute
	case "M30":
		return 30 * time.Minute
	case "H1":
		return 1 * time.Hour
	case "H4":
		return 4 * time.Hour
	case "D1":
		return 24 * time.Hour
	default:
		return 5 * time.Minute
	}
}

func stringsToUpperTrim(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] = b[i] - 32
		}
	}
	// trim spaces
	start := 0
	for start < len(b) && (b[start] == ' ' || b[start] == '\t' || b[start] == '\n' || b[start] == '\r') {
		start++
	}
	end := len(b)
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t' || b[end-1] == '\n' || b[end-1] == '\r') {
		end--
	}
	return string(b[start:end])
}

func (r *StrategyScheduleRunner) loopStable(ctx context.Context, schedule *model.StrategySchedule) {
	interval := timeframeToDuration(schedule.Timeframe)
	conf, _ := schedule.GetScheduleConfig()
	if conf != nil {
		if v, ok := conf["stable_override_interval_ms"]; ok {
			ms := toInt64Local(v)
			if ms > 0 {
				interval = time.Duration(ms) * time.Millisecond
			}
		}
	}
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	watchdog := scheduleWatchdogInterval()
	if watchdog < interval {
		watchdog = interval
	}
	ticker := time.NewTicker(watchdog)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.scheduleRepo.UpdateNextRunAt(ctx, schedule.ID, time.Now().Add(interval)); err != nil {
				logger.Warn("schedule next_run_at update failed", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
			}
			r.evalOnceWithSource(ctx, schedule, "watchdog")
		}
	}
}

func (r *StrategyScheduleRunner) loopStableEventDriven(ctx context.Context, schedule *model.StrategySchedule) {
	interval := timeframeToDuration(schedule.Timeframe)
	conf, _ := schedule.GetScheduleConfig()
	if conf != nil {
		if v, ok := conf["stable_override_interval_ms"]; ok {
			ms := toInt64Local(v)
			if ms > 0 {
				interval = time.Duration(ms) * time.Millisecond
			}
		}
	}
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	account, err := r.accountRepo.GetByID(ctx, schedule.AccountID)
	if err != nil || account == nil {
		logger.Warn("schedule v2: stable-event: account not found", zap.String("schedule_id", schedule.ID.String()))
		return
	}

	nextBoundary := func(now time.Time) time.Time {
		base := now.UTC().Truncate(interval)
		return base.Add(interval)
	}

	lastBucket := time.Now().UTC().Truncate(interval)
	watchdog := time.NewTicker(scheduleWatchdogInterval())
	defer watchdog.Stop()
	reconnectDelay := 2 * time.Second

	updateNextRunAt := func() {
		if err := r.scheduleRepo.UpdateNextRunAt(ctx, schedule.ID, nextBoundary(time.Now())); err != nil {
			logger.Warn("schedule next_run_at update failed", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
		}
	}

	mtType := stringsToUpperTrim(account.MTType)
	logWindow := scheduleLogThrottleWindow()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		switch mtType {
		case "MT4":
			conn, err := r.connMgr.GetMT4Connection(account.ID)
			if err != nil {
				throttledScheduleRunnerWarn("mt4.connection_unavailable."+schedule.ID.String(), logWindow,
					"schedule v2: stable-event: MT4 connection unavailable", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
				select {
				case <-ctx.Done():
					return
				case <-time.After(reconnectDelay):
					continue
				}
			}
			if err := retryHFSubscribeMT4(ctx, "mt4.subscribe_symbol", schedule.ID, func() error {
				return conn.Subscribe(ctx, schedule.Symbol)
			}); err != nil {
				throttledScheduleRunnerWarn("mt4.subscribe_symbol_failed."+schedule.ID.String(), logWindow,
					"schedule v2: stable-event: MT4 subscribe symbol still failing after retries", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
				select {
				case <-ctx.Done():
					return
				case <-time.After(reconnectDelay):
					continue
				}
			}
			if err := retryHFSubscribeMT4(ctx, "mt4.subscribe_quote_stream", schedule.ID, func() error {
				return conn.SubscribeQuoteStream(ctx)
			}); err != nil {
				throttledScheduleRunnerWarn("mt4.subscribe_quote_stream_failed."+schedule.ID.String(), logWindow,
					"schedule v2: stable-event: MT4 subscribe quote stream still failing after retries", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
				select {
				case <-ctx.Done():
					return
				case <-time.After(reconnectDelay):
					continue
				}
			}
			ch := conn.GetQuoteChannel()
			if ch == nil {
				throttledScheduleRunnerWarn("mt4.quote_channel_nil."+schedule.ID.String(), logWindow,
					"schedule v2: stable-event: MT4 quote channel nil", zap.String("schedule_id", schedule.ID.String()))
				select {
				case <-ctx.Done():
					return
				case <-time.After(reconnectDelay):
					continue
				}
			}
			restart := false
			for !restart {
				select {
				case <-ctx.Done():
					return
				case <-watchdog.C:
					updateNextRunAt()
					r.evalOnceWithSource(ctx, schedule, "watchdog")
				case q, ok := <-ch:
					if !ok {
						restart = true
						break
					}
					if q == nil || stringsToUpperTrim(q.Symbol) != stringsToUpperTrim(schedule.Symbol) {
						continue
					}
					currentBucket := time.Now().UTC().Truncate(interval)
					if currentBucket.After(lastBucket) {
						lastBucket = currentBucket
						updateNextRunAt()
						r.evalOnceWithSource(ctx, schedule, "bar_close")
					}
				}
			}
		case "MT5":
			conn, err := r.connMgr.GetMT5Connection(account.ID)
			if err != nil {
				throttledScheduleRunnerWarn("mt5.connection_unavailable."+schedule.ID.String(), logWindow,
					"schedule v2: stable-event: MT5 connection unavailable", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
				select {
				case <-ctx.Done():
					return
				case <-time.After(reconnectDelay):
					continue
				}
			}
			if err := retryHFSubscribeMT5(ctx, "mt5.subscribe_symbol", schedule.ID, func() error {
				return conn.Subscribe(ctx, schedule.Symbol, 0)
			}); err != nil {
				throttledScheduleRunnerWarn("mt5.subscribe_symbol_failed."+schedule.ID.String(), logWindow,
					"schedule v2: stable-event: MT5 subscribe symbol still failing after retries", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
				select {
				case <-ctx.Done():
					return
				case <-time.After(reconnectDelay):
					continue
				}
			}
			if err := retryHFSubscribeMT5(ctx, "mt5.subscribe_quote_stream", schedule.ID, func() error {
				return conn.SubscribeQuoteStream(ctx)
			}); err != nil {
				throttledScheduleRunnerWarn("mt5.subscribe_quote_stream_failed."+schedule.ID.String(), logWindow,
					"schedule v2: stable-event: MT5 subscribe quote stream still failing after retries", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
				select {
				case <-ctx.Done():
					return
				case <-time.After(reconnectDelay):
					continue
				}
			}
			ch := conn.GetQuoteChannel()
			if ch == nil {
				throttledScheduleRunnerWarn("mt5.quote_channel_nil."+schedule.ID.String(), logWindow,
					"schedule v2: stable-event: MT5 quote channel nil", zap.String("schedule_id", schedule.ID.String()))
				select {
				case <-ctx.Done():
					return
				case <-time.After(reconnectDelay):
					continue
				}
			}
			restart := false
			for !restart {
				select {
				case <-ctx.Done():
					return
				case <-watchdog.C:
					updateNextRunAt()
					r.evalOnceWithSource(ctx, schedule, "watchdog")
				case q, ok := <-ch:
					if !ok {
						restart = true
						break
					}
					if q == nil || stringsToUpperTrim(q.Symbol) != stringsToUpperTrim(schedule.Symbol) {
						continue
					}
					currentBucket := time.Now().UTC().Truncate(interval)
					if currentBucket.After(lastBucket) {
						lastBucket = currentBucket
						updateNextRunAt()
						r.evalOnceWithSource(ctx, schedule, "bar_close")
					}
				}
			}
		default:
			logger.Warn("schedule v2: stable-event: unsupported mt_type", zap.String("mt_type", account.MTType))
			return
		}
	}
}

func (r *StrategyScheduleRunner) loopHFQuote(ctx context.Context, schedule *model.StrategySchedule) {
	cooldown := 1 * time.Second
	conf, _ := schedule.GetScheduleConfig()
	if conf != nil {
		if v, ok := conf["hf_cooldown_ms"]; ok {
			ms := toInt64Local(v)
			if ms > 0 {
				cooldown = time.Duration(ms) * time.Millisecond
			}
		}
	}

	lastEval := time.Time{}

	account, err := r.accountRepo.GetByID(ctx, schedule.AccountID)
	if err != nil || account == nil {
		logger.Warn("schedule v2: hf: account not found", zap.String("schedule_id", schedule.ID.String()))
		return
	}

	mtType := stringsToUpperTrim(account.MTType)
	logWindow := scheduleLogThrottleWindow()
	switch mtType {
	case "MT4":
		conn, err := r.connMgr.GetMT4Connection(account.ID)
		if err != nil {
			throttledScheduleRunnerWarn("mt4.connection_unavailable."+schedule.ID.String(), logWindow,
				"schedule v2: hf: MT4 connection unavailable", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
			return
		}
		// Retry subscription to reduce transient MT bridge/network failures.
		if err := retryHFSubscribeMT4(ctx, "mt4.subscribe_symbol", schedule.ID, func() error {
			return conn.Subscribe(ctx, schedule.Symbol)
		}); err != nil {
			throttledScheduleRunnerWarn("mt4.subscribe_symbol_failed."+schedule.ID.String(), logWindow,
				"schedule v2: hf: MT4 subscribe symbol still failing after retries", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
		}
		if err := retryHFSubscribeMT4(ctx, "mt4.subscribe_quote_stream", schedule.ID, func() error {
			return conn.SubscribeQuoteStream(ctx)
		}); err != nil {
			throttledScheduleRunnerWarn("mt4.subscribe_quote_stream_failed."+schedule.ID.String(), logWindow,
				"schedule v2: hf: MT4 subscribe quote stream still failing after retries", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
		}
		ch := conn.GetQuoteChannel()
		if ch == nil {
			throttledScheduleRunnerWarn("mt4.quote_channel_nil."+schedule.ID.String(), logWindow,
				"schedule v2: hf: MT4 quote channel nil", zap.String("schedule_id", schedule.ID.String()))
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case q, ok := <-ch:
				if !ok {
					return
				}
				if q == nil || q.Symbol != schedule.Symbol {
					continue
				}
				if time.Since(lastEval) < cooldown {
					continue
				}
				lastEval = time.Now()
				r.evalOnceWithSource(ctx, schedule, "quote_stream")
			}
		}
	case "MT5":
		conn, err := r.connMgr.GetMT5Connection(account.ID)
		if err != nil {
			throttledScheduleRunnerWarn("mt5.connection_unavailable."+schedule.ID.String(), logWindow,
				"schedule v2: hf: MT5 connection unavailable", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
			return
		}
		if err := retryHFSubscribeMT5(ctx, "mt5.subscribe_symbol", schedule.ID, func() error {
			return conn.Subscribe(ctx, schedule.Symbol, 0)
		}); err != nil {
			throttledScheduleRunnerWarn("mt5.subscribe_symbol_failed."+schedule.ID.String(), logWindow,
				"schedule v2: hf: MT5 subscribe symbol still failing after retries", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
		}
		if err := retryHFSubscribeMT5(ctx, "mt5.subscribe_quote_stream", schedule.ID, func() error {
			return conn.SubscribeQuoteStream(ctx)
		}); err != nil {
			throttledScheduleRunnerWarn("mt5.subscribe_quote_stream_failed."+schedule.ID.String(), logWindow,
				"schedule v2: hf: MT5 subscribe quote stream still failing after retries", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
		}
		ch := conn.GetQuoteChannel()
		if ch == nil {
			throttledScheduleRunnerWarn("mt5.quote_channel_nil."+schedule.ID.String(), logWindow,
				"schedule v2: hf: MT5 quote channel nil", zap.String("schedule_id", schedule.ID.String()))
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case q, ok := <-ch:
				if !ok {
					return
				}
				if q == nil || q.Symbol != schedule.Symbol {
					continue
				}
				if time.Since(lastEval) < cooldown {
					continue
				}
				lastEval = time.Now()
				r.evalOnceWithSource(ctx, schedule, "quote_stream")
			}
		}
	default:
		logger.Warn("schedule v2: hf: unsupported mt_type", zap.String("mt_type", account.MTType))
		return
	}
}
