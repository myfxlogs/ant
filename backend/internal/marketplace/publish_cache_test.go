package marketplace

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// T1: ListPublished cache hit returns real total (not -1).
// Pre-populate cache, call ListPublished with nil pg (cache hit avoids DB).
// Adversarial: revert cache get to return -1 → total=-1 → assertion fails → red.
func TestListPublished_CacheHitReturnsTotal(t *testing.T) {
	s := &Service{
		pg:       nil, // safe: cache hit returns before touching pg
		pubCache: newPublishedCache(),
		log:      zap.NewNop(),
	}

	// Pre-populate cache with known data and total.
	data := []PublishedStrategy{{PublishID: "p1", Title: "Test Strategy"}}
	expectedTotal := 42
	cacheKey := s.pubCache.key("", "", "", "", "", 10, 0)
	s.pubCache.set(cacheKey, data, expectedTotal)

	// Call ListPublished with same params — should hit cache, return total=42.
	got, total, err := s.ListPublished(context.Background(), "", 10, 0, "", "", "", "")
	if err != nil {
		t.Fatalf("ListPublished error: %v", err)
	}
	if total != expectedTotal {
		t.Errorf("total = %d, want %d (cache hit must return real total, not -1)", total, expectedTotal)
	}
	if len(got) != 1 || got[0].PublishID != "p1" {
		t.Errorf("data mismatch: %+v", got)
	}
}

// T1b: Cache miss returns total=0 and ok=false.
func TestPublishedCache_MissReturnsZero(t *testing.T) {
	c := newPublishedCache()
	_, total, ok := c.get("nonexistent")
	if ok {
		t.Fatal("expected cache miss")
	}
	if total != 0 {
		t.Errorf("total = %d, want 0 on miss", total)
	}
}

// T2: buildPublishedCountQuery includes price_filter in SQL.
// Adversarial: delete priceFilter clause → query identical for all filters → red.
func TestBuildPublishedCountQuery_PriceFilterInSQL(t *testing.T) {
	freeQuery, freeArgs := buildPublishedCountQuery("", "", "", "free")
	paidQuery, paidArgs := buildPublishedCountQuery("", "", "", "paid")
	allQuery, _ := buildPublishedCountQuery("", "", "", "")

	if freeQuery == allQuery {
		t.Error("free filter query identical to no-filter query — priceFilter not in SQL")
	}
	if paidQuery == allQuery {
		t.Error("paid filter query identical to no-filter query — priceFilter not in SQL")
	}
	if freeQuery == paidQuery {
		t.Error("free and paid queries are identical — priceFilter not differentiated")
	}
	if len(freeArgs) != 0 || len(paidArgs) != 0 {
		t.Logf("freeArgs=%d paidArgs=%d (static clauses expected 0 args)", len(freeArgs), len(paidArgs))
	}
	if !contains(freeQuery, "price_amount IS NULL") {
		t.Error("free query missing 'price_amount IS NULL' clause")
	}
	if !contains(paidQuery, "price_amount::numeric > 0") {
		t.Error("paid query missing 'price_amount::numeric > 0' clause")
	}
}

// T1c: Cache entry expires after TTL.
func TestPublishedCache_Expiry(t *testing.T) {
	c := newPublishedCache()
	c.set("k_exp", []PublishedStrategy{}, 5)

	c.mu.Lock()
	e := c.m["k_exp"]
	e.expiresAt = time.Now().Add(-1 * time.Second)
	c.m["k_exp"] = e
	c.mu.Unlock()

	_, _, ok := c.get("k_exp")
	if ok {
		t.Fatal("expected cache miss after expiry")
	}
}
