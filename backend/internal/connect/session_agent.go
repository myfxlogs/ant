package connect

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

type SessionAgent struct {
	s          *StreamService
	accountID  string
	accountUID uuid.UUID

	ctx    context.Context
	cancel context.CancelFunc
}

func newSessionAgent(s *StreamService, accountID string) (*SessionAgent, error) {
	uid, err := uuid.Parse(accountID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &SessionAgent{s: s, accountID: accountID, accountUID: uid, ctx: ctx, cancel: cancel}, nil
}

func (a *SessionAgent) Stop() {
	if a == nil {
		return
	}
	a.cancel()
}

func (a *SessionAgent) Run() {
	if a == nil {
		return
	}

	retry := 500 * time.Millisecond
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
		}

		account, err := a.s.accountRepo.GetByID(context.Background(), a.accountUID)
		if err != nil {
			logger.Warn("session-agent: failed to load account, retrying",
				zap.String("account_id", a.accountID),
				zap.Error(err))
			a.s.BroadcastAccountStatus(a.accountID, SessionStatusDisconnected, "load account failed")
			a.sleepBackoff(&retry)
			continue
		}

		_ = a.s.ensureAccountStream(context.Background(), account)
		accountStream, ok := a.s.getAccountStream(a.accountID)
		if !ok || accountStream == nil {
			logger.Warn("session-agent: account stream missing after ensure, retrying",
				zap.String("account_id", a.accountID))
			a.s.BroadcastAccountStatus(a.accountID, SessionStatusDisconnected, "account stream missing")
			a.sleepBackoff(&retry)
			continue
		}

		if a.s.connManager == nil {
			logger.Warn("session-agent: connection manager is nil; cannot start session",
				zap.String("account_id", a.accountID))
			a.s.BroadcastAccountStatus(a.accountID, SessionStatusDisconnected, "connection manager nil")
			a.sleepBackoff(&retry)
			continue
		}

		a.s.BroadcastAccountStatus(a.accountID, SessionStatusConnecting, "connecting to MT")

		cctx, ccancel := context.WithTimeout(a.ctx, 30*time.Second)
		err = a.s.connManager.Connect(cctx, account)
		ccancel()
		if err != nil {
			logger.Warn("session-agent: failed to ensure MT connection; retrying",
				zap.String("account_id", a.accountID),
				zap.Error(err))
			a.s.BroadcastAccountStatus(a.accountID, SessionStatusDisconnected, "connect failed")
			a.sleepBackoff(&retry)
			continue
		}

		// Workers will perform initial channel reads and then keep streaming.
		a.s.BroadcastAccountStatus(a.accountID, SessionStatusSyncing, "connected")

		sessionCtx, sessionCancel := context.WithCancel(a.ctx)
		enableQuoteWorker := strings.TrimSpace(strings.ToLower(os.Getenv("ANTRADER_ENABLE_QUOTE_WORKER"))) == "true"
		doneCap := 2
		if enableQuoteWorker {
			doneCap = 3
		}
		done := make(chan string, doneCap)

		const (
			watchdogTick = 5 * time.Second
			softStale    = 30 * time.Second
			hardStale    = 3 * time.Minute
		)

		if _, err := a.s.goroutineMgr.Spawn("session-agent-watchdog-"+a.accountID, func(gctx context.Context) error {
			tick := time.NewTicker(watchdogTick)
			defer tick.Stop()
			lastSoftWarn := time.Time{}

			for {
				select {
				case <-gctx.Done():
					return gctx.Err()
				case <-sessionCtx.Done():
					return sessionCtx.Err()
				case <-tick.C:
				}

				as, ok := a.s.getAccountStream(a.accountID)
				if !ok || as == nil {
					continue
				}
				as.mu.RLock()
				hasSubs := len(as.Subscribers) > 0
				as.mu.RUnlock()
				if !hasSubs {
					continue
				}

				now := time.Now()
				var profitAt, orderAt time.Time
				if a.s.connManager != nil {
					if account.MTType == "MT4" {
						if c, err := a.s.connManager.GetMT4Connection(a.accountUID); err == nil && c != nil {
							profitAt = c.LastProfitRecvAt()
							orderAt = c.LastOrderRecvAt()
						}
					} else {
						if c, err := a.s.connManager.GetMT5Connection(a.accountUID); err == nil && c != nil {
							profitAt = c.LastProfitRecvAt()
							orderAt = c.LastOrderRecvAt()
						}
					}
				}

				if profitAt.IsZero() || orderAt.IsZero() {
					continue
				}

				profitStale := now.Sub(profitAt)
				orderStale := now.Sub(orderAt)
				bothSoft := profitStale >= softStale && orderStale >= softStale
				bothHard := profitStale >= hardStale && orderStale >= hardStale

				if bothSoft && now.Sub(lastSoftWarn) >= softStale {
					logger.Warn("session-agent: stream appears stale (soft)",
						zap.String("account_id", a.accountID),
						zap.String("mt_type", account.MTType),
						zap.Duration("profit_stale", profitStale),
						zap.Duration("order_stale", orderStale))
					a.s.BroadcastAccountStatus(a.accountID, SessionStatusDegraded, "stream stale (soft)")
					lastSoftWarn = now
				}

				if bothHard {
					logger.Warn("session-agent: stream stale (hard), restarting session",
						zap.String("account_id", a.accountID),
						zap.String("mt_type", account.MTType),
						zap.Duration("profit_stale", profitStale),
						zap.Duration("order_stale", orderStale))

					// Market-closed-safe policy: avoid restart storms on weekends/holidays,
					// but don't suppress forever (real disconnect could happen).
					if shouldSuppressHardRestart(now, profitStale, orderStale) {
						ps := profitStale.Truncate(time.Minute)
						os := orderStale.Truncate(time.Minute)
						msg := fmt.Sprintf("no updates; probable market closed; profit_stale=%s order_stale=%s", ps, os)
						a.s.BroadcastAccountStatus(a.accountID, SessionStatusMarketClosed, msg)
						continue
					}

					a.s.BroadcastAccountStatus(a.accountID, SessionStatusDisconnected, "stream stale (hard)")

					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						_ = a.s.connManager.Disconnect(ctx, a.accountUID)
					}()

					sessionCancel()
					return nil
				}
			}
		}); err != nil {
			logger.Warn("session-agent: failed to spawn watchdog",
				zap.String("account_id", a.accountID),
				zap.Error(err))
			a.s.BroadcastAccountStatus(a.accountID, SessionStatusDisconnected, "watchdog spawn failed")
			sessionCancel()
			a.sleepBackoff(&retry)
			continue
		}

		if a.s.goroutineMgr == nil {
			logger.Warn("session-agent: goroutine manager is nil; cannot start workers",
				zap.String("account_id", a.accountID))
			a.s.BroadcastAccountStatus(a.accountID, SessionStatusDisconnected, "goroutine manager nil")
			sessionCancel()
			a.sleepBackoff(&retry)
			continue
		}

		if _, err := a.s.goroutineMgr.Spawn("session-agent-profit-"+a.accountID, func(gctx context.Context) error {
			ctx, cancel := context.WithCancel(sessionCtx)
			go func() {
				select {
				case <-gctx.Done():
					cancel()
				case <-ctx.Done():
				}
			}()
			a.s.startProfitStreamWithCtx(ctx, accountStream, account)
			select {
			case done <- "profit":
			default:
			}
			return ctx.Err()
		}); err != nil {
			logger.Warn("session-agent: failed to spawn profit worker",
				zap.String("account_id", a.accountID),
				zap.Error(err))
			a.s.BroadcastAccountStatus(a.accountID, SessionStatusDisconnected, "profit worker spawn failed")
			sessionCancel()
			a.sleepBackoff(&retry)
			continue
		}

		if _, err := a.s.goroutineMgr.Spawn("session-agent-order-"+a.accountID, func(gctx context.Context) error {
			ctx, cancel := context.WithCancel(sessionCtx)
			go func() {
				select {
				case <-gctx.Done():
					cancel()
				case <-ctx.Done():
				}
			}()
			a.s.startOrderStreamWithCtx(ctx, accountStream, account)
			select {
			case done <- "order":
			default:
			}
			return ctx.Err()
		}); err != nil {
			logger.Warn("session-agent: failed to spawn order worker",
				zap.String("account_id", a.accountID),
				zap.Error(err))
			a.s.BroadcastAccountStatus(a.accountID, SessionStatusDisconnected, "order worker spawn failed")
			sessionCancel()
			a.sleepBackoff(&retry)
			continue
		}

		if enableQuoteWorker {
			if _, err := a.s.goroutineMgr.Spawn("session-agent-quote-"+a.accountID, func(gctx context.Context) error {
				ctx, cancel := context.WithCancel(sessionCtx)
				go func() {
					select {
					case <-gctx.Done():
						cancel()
					case <-ctx.Done():
					}
				}()
				a.s.startQuoteStreamWithCtx(ctx, accountStream, account)
				select {
				case done <- "quote":
				default:
				}
				return ctx.Err()
			}); err != nil {
				logger.Warn("session-agent: failed to spawn quote worker",
					zap.String("account_id", a.accountID),
					zap.Error(err))
				a.s.BroadcastAccountStatus(a.accountID, SessionStatusDisconnected, "quote worker spawn failed")
				sessionCancel()
				a.sleepBackoff(&retry)
				continue
			}
		}

		a.s.BroadcastAccountStatus(a.accountID, SessionStatusRunning, "streaming")

		select {
		case <-a.ctx.Done():
			sessionCancel()
			return
		case who := <-done:
			logger.Warn("session-agent: session worker exited; restarting",
				zap.String("account_id", a.accountID),
				zap.String("worker", who))
			a.s.BroadcastAccountStatus(a.accountID, SessionStatusDisconnected, "worker exited")
			sessionCancel()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
			}
		}

		sessionCancel()
		a.sleepBackoff(&retry)
	}
}

func (a *SessionAgent) sleepBackoff(retry *time.Duration) {
	if retry == nil {
		t := 500 * time.Millisecond
		retry = &t
	}
	capDur := 30 * time.Second
	jitter := time.Duration(rand.Int63n(int64(*retry/3 + 1)))
	sleep := *retry + jitter
	if sleep > capDur {
		sleep = capDur
	}
	select {
	case <-a.ctx.Done():
		return
	case <-time.After(sleep):
	}

	*retry = *retry * 2
	if *retry > capDur {
		*retry = capDur
	}
}
