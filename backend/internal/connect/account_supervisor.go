package connect

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

// AccountSupervisor implements an MT-official-like persistent session per account.
// It owns the lifecycle of profit/order stream workers and restarts them with backoff on failures.
// The supervisor is intentionally decoupled from frontend subscription churn.
//
// NOTE: We keep it in package connect so it can reuse existing stream worker implementations
// (startProfitStreamWithCtx/startOrderStreamWithCtx) with minimal disruption.

type AccountSupervisor struct {
	s          *StreamService
	accountID  string
	accountUID uuid.UUID

	ctx    context.Context
	cancel context.CancelFunc
}

func newAccountSupervisor(s *StreamService, accountID string) (*AccountSupervisor, error) {
	uid, err := uuid.Parse(accountID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &AccountSupervisor{s: s, accountID: accountID, accountUID: uid, ctx: ctx, cancel: cancel}, nil
}

func (sup *AccountSupervisor) Stop() {
	if sup == nil {
		return
	}
	sup.cancel()
}

func (sup *AccountSupervisor) Run() {
	if sup == nil {
		return
	}

	// backoff: start small, cap at 30s. Add jitter to avoid thundering herd.
	retry := 500 * time.Millisecond
	for {
		select {
		case <-sup.ctx.Done():
			return
		default:
		}

		account, err := sup.s.accountRepo.GetByID(context.Background(), sup.accountUID)
		if err != nil {
			logger.Warn("supervisor: failed to load account, retrying",
				zap.String("account_id", sup.accountID),
				zap.Error(err))
			sup.sleepBackoff(&retry)
			continue
		}

		// Ensure a persistent account stream exists for routing to subscribers.
		_ = sup.s.ensureAccountStream(context.Background(), account)
		accountStream, ok := sup.s.getAccountStream(sup.accountID)
		if !ok || accountStream == nil {
			logger.Warn("supervisor: account stream missing after ensure, retrying",
				zap.String("account_id", sup.accountID))
			sup.sleepBackoff(&retry)
			continue
		}

		// Ensure MT connection is active before starting stream workers.
		if sup.s.connManager == nil {
			logger.Warn("supervisor: connection manager is nil; cannot start session",
				zap.String("account_id", sup.accountID))
			sup.sleepBackoff(&retry)
			continue
		}
		cctx, ccancel := context.WithTimeout(sup.ctx, 30*time.Second)
		err = sup.s.connManager.Connect(cctx, account)
		ccancel()
		if err != nil {
			logger.Warn("supervisor: failed to ensure MT connection; retrying",
				zap.String("account_id", sup.accountID),
				zap.Error(err))
			sup.sleepBackoff(&retry)
			continue
		}

		// A session context lets us stop both workers when one exits.
		sessionCtx, sessionCancel := context.WithCancel(sup.ctx)

		done := make(chan string, 2)

		// Session watchdog: detect "stuck" streams (no recv) even when they don't error.
		// Market-closed-safe policy:
		// - Only hard-restart when BOTH profit+order streams are stale.
		// - Only hard-restart when there are active frontend subscribers for this account.
		// - Use conservative thresholds to avoid storms during weekends/holidays.
		const (
			watchdogTick = 5 * time.Second
			softStale    = 30 * time.Second
			hardStale    = 3 * time.Minute
		)

		if _, err := sup.s.goroutineMgr.Spawn("supervisor-watchdog-"+sup.accountID, func(gctx context.Context) error {
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
					// continue
				}

				// Only restart when there are active frontend subscribers.
				as, ok := sup.s.getAccountStream(sup.accountID)
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
				if sup.s.connManager != nil {
					if account.MTType == "MT4" {
						if c, err := sup.s.connManager.GetMT4Connection(sup.accountUID); err == nil && c != nil {
							profitAt = c.LastProfitRecvAt()
							orderAt = c.LastOrderRecvAt()
						}
					} else {
						if c, err := sup.s.connManager.GetMT5Connection(sup.accountUID); err == nil && c != nil {
							profitAt = c.LastProfitRecvAt()
							orderAt = c.LastOrderRecvAt()
						}
					}
				}

				// If we haven't received anything yet, don't treat it as stale.
				if profitAt.IsZero() || orderAt.IsZero() {
					continue
				}

				profitStale := now.Sub(profitAt)
				orderStale := now.Sub(orderAt)
				bothSoft := profitStale >= softStale && orderStale >= softStale
				bothHard := profitStale >= hardStale && orderStale >= hardStale

				if bothSoft && now.Sub(lastSoftWarn) >= softStale {
					logger.Warn("supervisor: stream appears stale (soft)",
						zap.String("account_id", sup.accountID),
						zap.String("mt_type", account.MTType),
						zap.Duration("profit_stale", profitStale),
						zap.Duration("order_stale", orderStale))
					lastSoftWarn = now
				}

				if bothHard {
					logger.Warn("supervisor: stream stale (hard), restarting session",
						zap.String("account_id", sup.accountID),
						zap.String("mt_type", account.MTType),
						zap.Duration("profit_stale", profitStale),
						zap.Duration("order_stale", orderStale))

					// Force underlying gRPC streams to break by disconnecting the account.
					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						_ = sup.s.connManager.Disconnect(ctx, sup.accountUID)
					}()

					// Stop this session; supervisor main loop will restart with backoff.
					sessionCancel()
					return nil
				}
			}
		}); err != nil {
			logger.Warn("supervisor: failed to spawn watchdog",
				zap.String("account_id", sup.accountID),
				zap.Error(err))
			sessionCancel()
			sup.sleepBackoff(&retry)
			continue
		}

		if sup.s.goroutineMgr == nil {
			logger.Warn("supervisor: goroutine manager is nil; cannot start workers",
				zap.String("account_id", sup.accountID))
			sessionCancel()
			sup.sleepBackoff(&retry)
			continue
		}

		if _, err := sup.s.goroutineMgr.Spawn("supervisor-profit-"+sup.accountID, func(gctx context.Context) error {
			ctx, cancel := context.WithCancel(sessionCtx)
			go func() {
				select {
				case <-gctx.Done():
					cancel()
				case <-ctx.Done():
				}
			}()
			sup.s.startProfitStreamWithCtx(ctx, accountStream, account)
			select {
			case done <- "profit":
			default:
			}
			return ctx.Err()
		}); err != nil {
			logger.Warn("supervisor: failed to spawn profit worker",
				zap.String("account_id", sup.accountID),
				zap.Error(err))
			sessionCancel()
			sup.sleepBackoff(&retry)
			continue
		}

		if _, err := sup.s.goroutineMgr.Spawn("supervisor-order-"+sup.accountID, func(gctx context.Context) error {
			ctx, cancel := context.WithCancel(sessionCtx)
			go func() {
				select {
				case <-gctx.Done():
					cancel()
				case <-ctx.Done():
				}
			}()
			sup.s.startOrderStreamWithCtx(ctx, accountStream, account)
			select {
			case done <- "order":
			default:
			}
			return ctx.Err()
		}); err != nil {
			logger.Warn("supervisor: failed to spawn order worker",
				zap.String("account_id", sup.accountID),
				zap.Error(err))
			sessionCancel()
			sup.sleepBackoff(&retry)
			continue
		}

		// Wait until one of the workers exits, then restart both.
		select {
		case <-sup.ctx.Done():
			sessionCancel()
			return
		case who := <-done:
			logger.Warn("supervisor: session worker exited; restarting",
				zap.String("account_id", sup.accountID),
				zap.String("worker", who))
			sessionCancel()
			// drain second exit (best-effort) to avoid goroutine leak
			select {
			case <-done:
			case <-time.After(2 * time.Second):
			}
		}

		// Ensure this session context is fully released before backoff/restart.
		sessionCancel()

		sup.sleepBackoff(&retry)
	}
}

func (sup *AccountSupervisor) sleepBackoff(retry *time.Duration) {
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
	case <-sup.ctx.Done():
		return
	case <-time.After(sleep):
	}

	// Exponential increase.
	*retry = *retry * 2
	if *retry > capDur {
		*retry = capDur
	}
}
