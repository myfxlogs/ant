// Package backfiller fills missing historical bars via mtapi GetPriceHistory.
// Spec: docs/spec/18-backfiller.md; ADR: docs/adr/0009.
package backfiller

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// Source fetches historical bars from the MT API.
type Source interface {
	FetchBars(ctx context.Context, req FetchReq) ([]*mdtick.Bar, error)
}

// FetchReq is a single GetPriceHistory request.
type FetchReq struct {
	AccountID string
	Broker    string
	Canonical string
	SymbolRaw string
	Period    string // "1m" | "1h" | "1d"
	From, To  int64  // unix ms
	Limit     int    // default 5000
}

// Target ingests bars through bar_aggregator finality check.
type Target interface {
	IngestBar(ctx context.Context, bar *mdtick.Bar) error
}

// ActiveAccount is a minimal account view for backfill scheduling.
type ActiveAccount struct {
	AccountID string
	Broker    string
	Symbols   []string // canonical symbols
}

// CHMaxCloseTs queries ClickHouse for the maximum close_ts_unix_ms.
type CHMaxCloseTs interface {
	MaxCloseTs(ctx context.Context, broker, canonical, period string) (int64, error)
}

// PGActiveAccounts loads accounts with active subscriptions.
type PGActiveAccounts interface {
	ActiveAccounts(ctx context.Context) ([]ActiveAccount, error)
}

// Backfiller fills historical bar gaps for all active accounts.
type Backfiller struct {
	src             Source
	tgt             Target
	chMax           CHMaxCloseTs
	pgActive        PGActiveAccounts
	accountLimiters map[string]*rate.Limiter // per-account: 6 req/min each
	globalLimiter   *rate.Limiter            // global: 60 req/s ceiling
	metrics         *Metrics
}

// New creates a Backfiller.
func New(src Source, tgt Target, chMax CHMaxCloseTs, pgActive PGActiveAccounts) *Backfiller {
	return &Backfiller{
		src:             src,
		tgt:             tgt,
		chMax:           chMax,
		pgActive:        pgActive,
		accountLimiters: make(map[string]*rate.Limiter),
		globalLimiter:   rate.NewLimiter(rate.Limit(60), 1), // 60 req/s global ceiling
		metrics:         &Metrics{},
	}
}

// getLimiter returns the per-account limiter, creating one if needed.
func (b *Backfiller) getLimiter(accountID string) *rate.Limiter {
	if l, ok := b.accountLimiters[accountID]; ok {
		return l
	}
	l := rate.NewLimiter(rate.Every(10*time.Second), 1) // 6 req/min/account
	b.accountLimiters[accountID] = l
	return l
}

// Run executes a complete backfill scan for all active accounts.
// Called once at startup, then every 6h (cron-like).
func (b *Backfiller) Run(ctx context.Context) error {
	accounts, err := b.pgActive.ActiveAccounts(ctx)
	if err != nil {
		return fmt.Errorf("load active accounts: %w", err)
	}
	b.metrics.started.Add(int64(len(accounts)))

	for _, acc := range accounts {
		if err := b.backfillAccount(ctx, acc); err != nil {
			b.metrics.errors.Add(1)
			// Continue with next account; single-account failure is not fatal.
		}
	}
	return nil
}

// BackfillAccount backfills all periods for a single account's symbols.
func (b *Backfiller) BackfillAccount(ctx context.Context, accountID string) error {
	accounts, err := b.pgActive.ActiveAccounts(ctx)
	if err != nil {
		return fmt.Errorf("load active accounts for single-account backfill: %w", err)
	}
	for _, acc := range accounts {
		if acc.AccountID == accountID {
			return b.backfillAccount(ctx, acc)
		}
	}
	return nil
}

// BackfillSymbol backfills all periods for a single (account, canonical) pair.
func (b *Backfiller) BackfillSymbol(ctx context.Context, accountID, canonical string) error {
	accounts, err := b.pgActive.ActiveAccounts(ctx)
	if err != nil {
		return fmt.Errorf("load active accounts for symbol backfill: %w", err)
	}
	for _, acc := range accounts {
		if acc.AccountID == accountID {
			for _, sym := range acc.Symbols {
				if sym == canonical {
					return b.backfillSymbol(ctx, acc, canonical)
				}
			}
		}
	}
	return nil
}

func (b *Backfiller) backfillAccount(ctx context.Context, acc ActiveAccount) error {
	for _, canon := range acc.Symbols {
		if err := b.backfillSymbol(ctx, acc, canon); err != nil {
			return fmt.Errorf("backfill symbol %s: %w", canon, err)
		}
	}
	return nil
}

var defaultPeriods = []struct {
	name     string
	lookback time.Duration
}{
	{"1m", 30 * 24 * time.Hour},
	{"1h", 90 * 24 * time.Hour},
	{"1d", 365 * 24 * time.Hour},
}

func (b *Backfiller) backfillSymbol(ctx context.Context, acc ActiveAccount, canon string) error {
	for _, dp := range defaultPeriods {
		from, err := b.chMax.MaxCloseTs(ctx, acc.Broker, canon, dp.name)
		if err != nil {
			return fmt.Errorf("query max close_ts for %s/%s/%s: %w", acc.Broker, canon, dp.name, err)
		}
		if from == 0 {
			from = Clk.Now().Add(-dp.lookback).UnixMilli()
		}
		to := Clk.Now().UnixMilli()
		if to-from < periodMs(dp.name)*2 {
			continue // gap < 2 periods, not worth the API call
		}

		started := Clk.Now()
		if err := b.backfillRange(ctx, acc, canon, dp.name, from, to); err != nil {
			b.metrics.errors.Add(1)
			continue
		}
		b.metrics.durationNs.Add(time.Since(started).Nanoseconds())
		b.metrics.durationCnt.Add(1)
		b.metrics.barsIngested.Add(1)
	}
	return nil
}

func (b *Backfiller) backfillRange(ctx context.Context, acc ActiveAccount, canon, period string, from, to int64) error {
	accLimiter := b.getLimiter(acc.AccountID)
	for from < to {
		if err := accLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("account rate limiter wait: %w", err)
		}
		if err := b.globalLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("global rate limiter wait: %w", err)
		}
		bars, err := b.src.FetchBars(ctx, FetchReq{
			AccountID: acc.AccountID,
			Broker:    acc.Broker,
			Canonical: canon,
			Period:    period,
			From:      from,
			To:        to,
			Limit:     5000,
		})
		if err != nil {
			return fmt.Errorf("fetch bars: %w", err)
		}
		for _, bar := range bars {
			bar.IsReplay = true
			if err := b.tgt.IngestBar(ctx, bar); err != nil {
				return fmt.Errorf("ingest bar: %w", err)
			}
		}
		if len(bars) == 0 {
			break
		}
		from = bars[len(bars)-1].CloseTsUnixMs + 1
	}
	return nil
}

func periodMs(period string) int64 {
	switch period {
	case "1m":
		return 60_000
	case "5m":
		return 300_000
	case "15m":
		return 900_000
	case "1h":
		return 3_600_000
	case "4h":
		return 14_400_000
	case "1d":
		return 86_400_000
	default:
		return 60_000
	}
}
