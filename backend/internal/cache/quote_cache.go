package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	v1 "anttrader/gen/proto"
)

type QuoteCache interface {
	GetQuote(ctx context.Context, accountID uuid.UUID, symbol string) (*v1.Quote, bool)
	SetQuote(ctx context.Context, accountID uuid.UUID, quote *v1.Quote, ttl time.Duration)
}

type inMemoryQuoteEntry struct {
	quote     *v1.Quote
	expiresAt time.Time
}

// RealtimeQuoteCache is a small L1 (in-memory) + L2 (Redis) cache.
// L1 is optimized for very hot symbols and very short TTL.
// L2 provides cross-process cache when running multiple replicas.
//
// Note: L1 eviction is lazy (checked on read) to keep implementation simple.
// It is safe for concurrent use.
//
// Key strategy:
// - L1 key: accountID + symbol
// - L2 key: quote:{accountID}:{symbol}
// Category: market
//
// The cached value is the protobuf v1.Quote JSON-marshaled by CacheService.
// That is acceptable because cached payload is small and TTL is short.
//
// If you want maximum performance, migrate L2 to protobuf binary encoding.
//
// This cache is optional; if CacheService is nil it behaves like L1-only.
//
// TTL guidance:
// - Quotes: 250ms~2s depending on your quote update frequency.
//
// This implementation intentionally avoids background goroutines.
//
// IMPORTANT: never store pointers that will be mutated after SetQuote.
// v1.Quote is treated as immutable here.

type RealtimeQuoteCache struct {
	l1 sync.Map // map[string]inMemoryQuoteEntry
	l2 *CacheService
}

func NewRealtimeQuoteCache(l2 *CacheService) *RealtimeQuoteCache {
	return &RealtimeQuoteCache{l2: l2}
}

func (c *RealtimeQuoteCache) GetQuote(ctx context.Context, accountID uuid.UUID, symbol string) (*v1.Quote, bool) {
	k := c.l1Key(accountID, symbol)
	if v, ok := c.l1.Load(k); ok {
		entry := v.(inMemoryQuoteEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.quote, true
		}
		c.l1.Delete(k)
	}

	if c.l2 == nil {
		return nil, false
	}

	var q v1.Quote
	if err := c.l2.Get(ctx, c.l2Key(accountID, symbol), &q, MarketCacheOpts); err == nil {
		// keep a tiny L1 TTL to smooth bursts
		c.l1.Store(k, inMemoryQuoteEntry{quote: &q, expiresAt: time.Now().Add(500 * time.Millisecond)})
		return &q, true
	}

	return nil, false
}

func (c *RealtimeQuoteCache) SetQuote(ctx context.Context, accountID uuid.UUID, quote *v1.Quote, ttl time.Duration) {
	if quote == nil {
		return
	}
	k := c.l1Key(accountID, quote.Symbol)
	c.l1.Store(k, inMemoryQuoteEntry{quote: quote, expiresAt: time.Now().Add(ttl)})

	if c.l2 == nil {
		return
	}

	_ = c.l2.Set(ctx, c.l2Key(accountID, quote.Symbol), quote, CacheOptions{TTL: ttl, Category: "market"})
}

func (c *RealtimeQuoteCache) l1Key(accountID uuid.UUID, symbol string) string {
	return fmt.Sprintf("%s:%s", accountID.String(), symbol)
}

func (c *RealtimeQuoteCache) l2Key(accountID uuid.UUID, symbol string) string {
	return fmt.Sprintf("quote:%s:%s", accountID.String(), symbol)
}
