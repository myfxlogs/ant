package connect

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

type quoteKey struct {
	accountID string
	symbol    string
}

type quoteRecord struct {
	bid  float64
	ask  float64
	time string
}

var (
	quoteWorkerLogThrottleMu   sync.Mutex
	quoteWorkerLogThrottleLast = make(map[string]time.Time)
	quoteWorkerLogSuppressed   = make(map[string]int)
)

const (
	mt4QuoteSubscribeRetryAttempts = 3
	mt4QuoteSubscribeRetryDelay    = 500 * time.Millisecond
	mt5QuoteSubscribeRetryAttempts = 5
	mt5QuoteSubscribeRetryDelay    = 700 * time.Millisecond
	quoteConnectRetryDelay         = 2 * time.Second
	defaultQuoteLogThrottleWindow  = 12 * time.Second
)

func quotePublishInterval() time.Duration {
	// Keep default reasonably low-latency for strategy usage, but prevent event storms.
	// Env format: time.ParseDuration, e.g. 200ms, 1s.
	v := strings.TrimSpace(os.Getenv("ANTRADER_QUOTE_PUBLISH_INTERVAL"))
	if v == "" {
		return 200 * time.Millisecond
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 200 * time.Millisecond
	}
	return d
}

func quoteLogThrottleWindow() time.Duration {
	v := strings.TrimSpace(os.Getenv("ANTRADER_QUOTE_LOG_THROTTLE_WINDOW"))
	if v == "" {
		return defaultQuoteLogThrottleWindow
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return defaultQuoteLogThrottleWindow
	}
	return d
}

func (s *StreamService) startQuoteStreamWithCtx(ctx context.Context, accountStream *AccountStream, account *model.MTAccount) {
	if s == nil || accountStream == nil || account == nil {
		return
	}
	if accountStream.AccountID == "" {
		return
	}
	if !s.isLeaderForAccount(accountStream.AccountID) {
		return
	}

	interval := quotePublishInterval()
	logWindow := quoteLogThrottleWindow()
	var (
		mu     sync.Mutex
		latest = make(map[quoteKey]quoteRecord)
	)

	flush := func() {
		mu.Lock()
		copy := make(map[quoteKey]quoteRecord, len(latest))
		for k, v := range latest {
			copy[k] = v
		}
		latest = make(map[quoteKey]quoteRecord)
		mu.Unlock()

		for k, v := range copy {
			ev := &v1.QuoteEvent{Symbol: k.symbol, Bid: v.bid, Ask: v.ask, Time: v.time}
			s.publishQuoteEvent(k.accountID, ev)
		}
	}

	tick := time.NewTicker(interval)
	defer tick.Stop()

	// Subscribe + read raw quote stream; aggregate latest by symbol.
	switch strings.ToUpper(account.MTType) {
	case "MT4":
		for {
			if ctx.Err() != nil {
				flush()
				return
			}
			conn, err := s.connManager.GetMT4Connection(account.ID)
			if err != nil {
				throttledQuoteWorkerWarn("mt4.get_connection_failed."+accountStream.AccountID, logWindow,
					"quote worker: get MT4 connection failed, retrying",
					zap.String("account_id", accountStream.AccountID),
					zap.Error(err))
				if !sleepWithContext(ctx, quoteConnectRetryDelay) {
					flush()
					return
				}
				continue
			}
			if err := retryQuoteSubscribeMT4(ctx, "mt4.subscribe_quote_stream", accountStream.AccountID, func() error {
				return conn.SubscribeQuoteStream(ctx)
			}); err != nil {
				throttledQuoteWorkerWarn("mt4.subscribe_quote_stream_failed."+accountStream.AccountID, logWindow,
					"quote worker: MT4 subscribe quote stream failed after retries",
					zap.String("account_id", accountStream.AccountID),
					zap.Error(err))
				if !sleepWithContext(ctx, quoteConnectRetryDelay) {
					flush()
					return
				}
				continue
			}
			ch := conn.GetQuoteChannel()
			if ch == nil {
				throttledQuoteWorkerWarn("mt4.quote_channel_nil."+accountStream.AccountID, logWindow,
					"quote worker: MT4 quote channel nil, retrying",
					zap.String("account_id", accountStream.AccountID))
				if !sleepWithContext(ctx, quoteConnectRetryDelay) {
					flush()
					return
				}
				continue
			}
			restart := false
			for !restart {
				select {
				case <-ctx.Done():
					flush()
					return
				case <-tick.C:
					flush()
				case q, ok := <-ch:
					if !ok {
						restart = true
						break
					}
					if q == nil || q.Symbol == "" {
						continue
					}
					ts := ""
					if q.Time != nil {
						ts = q.Time.AsTime().UTC().Format(time.RFC3339Nano)
					}
					mu.Lock()
					latest[quoteKey{accountID: accountStream.AccountID, symbol: q.Symbol}] = quoteRecord{bid: q.Bid, ask: q.Ask, time: ts}
					mu.Unlock()
				}
			}
			throttledQuoteWorkerWarn("mt4.quote_channel_closed."+accountStream.AccountID, logWindow,
				"quote worker: MT4 quote channel closed, resubscribing",
				zap.String("account_id", accountStream.AccountID))
		}

	case "MT5":
		for {
			if ctx.Err() != nil {
				flush()
				return
			}
			conn, err := s.connManager.GetMT5Connection(account.ID)
			if err != nil {
				throttledQuoteWorkerWarn("mt5.get_connection_failed."+accountStream.AccountID, logWindow,
					"quote worker: get MT5 connection failed, retrying",
					zap.String("account_id", accountStream.AccountID),
					zap.Error(err))
				if !sleepWithContext(ctx, quoteConnectRetryDelay) {
					flush()
					return
				}
				continue
			}
			if err := retryQuoteSubscribeMT5(ctx, "mt5.subscribe_quote_stream", accountStream.AccountID, func() error {
				return conn.SubscribeQuoteStream(ctx)
			}); err != nil {
				throttledQuoteWorkerWarn("mt5.subscribe_quote_stream_failed."+accountStream.AccountID, logWindow,
					"quote worker: MT5 subscribe quote stream failed after retries",
					zap.String("account_id", accountStream.AccountID),
					zap.Error(err))
				if !sleepWithContext(ctx, quoteConnectRetryDelay) {
					flush()
					return
				}
				continue
			}
			ch := conn.GetQuoteChannel()
			if ch == nil {
				throttledQuoteWorkerWarn("mt5.quote_channel_nil."+accountStream.AccountID, logWindow,
					"quote worker: MT5 quote channel nil, retrying",
					zap.String("account_id", accountStream.AccountID))
				if !sleepWithContext(ctx, quoteConnectRetryDelay) {
					flush()
					return
				}
				continue
			}
			restart := false
			for !restart {
				select {
				case <-ctx.Done():
					flush()
					return
				case <-tick.C:
					flush()
				case q, ok := <-ch:
					if !ok {
						restart = true
						break
					}
					if q == nil || q.Symbol == "" {
						continue
					}
					ts := ""
					if q.Time != nil {
						ts = q.Time.AsTime().UTC().Format(time.RFC3339Nano)
					}
					mu.Lock()
					latest[quoteKey{accountID: accountStream.AccountID, symbol: q.Symbol}] = quoteRecord{bid: q.Bid, ask: q.Ask, time: ts}
					mu.Unlock()
				}
			}
			throttledQuoteWorkerWarn("mt5.quote_channel_closed."+accountStream.AccountID, logWindow,
				"quote worker: MT5 quote channel closed, resubscribing",
				zap.String("account_id", accountStream.AccountID))
		}
	}
}

func retryQuoteSubscribeMT4(ctx context.Context, op string, accountID string, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= mt4QuoteSubscribeRetryAttempts; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if !shouldRetryMT4Subscribe(err) {
				return err
			}
			if attempt < mt4QuoteSubscribeRetryAttempts {
				throttledQuoteWorkerWarn("mt4.subscribe_retry."+op+"."+accountID, quoteLogThrottleWindow(),
					"quote worker: subscribe retry",
					zap.String("op", op),
					zap.String("account_id", accountID),
					zap.Int("attempt", attempt),
					zap.Error(err))
				if !sleepWithContext(ctx, mt4QuoteSubscribeRetryDelay) {
					return ctx.Err()
				}
			}
		}
	}
	return lastErr
}

func retryQuoteSubscribeMT5(ctx context.Context, op string, accountID string, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= mt5QuoteSubscribeRetryAttempts; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if !shouldRetryMT5Subscribe(err) {
				return err
			}
			if attempt < mt5QuoteSubscribeRetryAttempts {
				throttledQuoteWorkerWarn("mt5.subscribe_retry."+op+"."+accountID, quoteLogThrottleWindow(),
					"quote worker: subscribe retry",
					zap.String("op", op),
					zap.String("account_id", accountID),
					zap.Int("attempt", attempt),
					zap.Error(err))
				if !sleepWithContext(ctx, mt5QuoteSubscribeRetryDelay) {
					return ctx.Err()
				}
			}
		}
	}
	return lastErr
}

func shouldRetryMT4Subscribe(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	// MT4 proto fatal classes: invalid token/account/rights/params do not recover by retry.
	if strings.Contains(s, "invalid_token") ||
		strings.Contains(s, "invalid token") ||
		strings.Contains(s, "invalid_account") ||
		strings.Contains(s, "account_disabled") ||
		strings.Contains(s, "not_enough_rights") ||
		strings.Contains(s, "invalid_param") {
		return false
	}
	return true
}

func shouldRetryMT5Subscribe(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	// MT5 proto fatal classes: auth/token/account/symbol/request validation problems.
	if strings.Contains(s, "invalid_token") ||
		strings.Contains(s, "invalid token") ||
		strings.Contains(s, "invalid_account") ||
		strings.Contains(s, "account_disabled") ||
		strings.Contains(s, "invalid_symbol") ||
		strings.Contains(s, "invalid_param") ||
		strings.Contains(s, "invalid_request") ||
		strings.Contains(s, "not_permission") {
		return false
	}
	return true
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

func shouldLogQuoteWorker(key string, window time.Duration) (bool, int) {
	now := time.Now()
	quoteWorkerLogThrottleMu.Lock()
	defer quoteWorkerLogThrottleMu.Unlock()
	if last, ok := quoteWorkerLogThrottleLast[key]; ok && now.Sub(last) < window {
		quoteWorkerLogSuppressed[key]++
		return false, 0
	}
	suppressed := quoteWorkerLogSuppressed[key]
	delete(quoteWorkerLogSuppressed, key)
	quoteWorkerLogThrottleLast[key] = now
	cutoff := now.Add(-2 * window)
	for k, ts := range quoteWorkerLogThrottleLast {
		if ts.Before(cutoff) {
			delete(quoteWorkerLogThrottleLast, k)
			delete(quoteWorkerLogSuppressed, k)
		}
	}
	return true, suppressed
}

func throttledQuoteWorkerWarn(key string, window time.Duration, msg string, fields ...zap.Field) {
	if ok, suppressed := shouldLogQuoteWorker(key, window); ok {
		if suppressed > 0 {
			fields = append(fields, zap.Int("suppressed_count", suppressed))
		}
		logger.Warn(msg, fields...)
	}
}
