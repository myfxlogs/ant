package marketplace

import (
	"testing"
	"time"
)

// T1: Cache hit returns correct total (not -1).
// Adversarial: delete `entry.total` return → total=0 → red.
func TestPublishedCache_HitReturnsTotal(t *testing.T) {
	c := newPublishedCache()
	data := []PublishedStrategy{{PublishID: "p1"}}
	c.set("k1", data, 42)

	got, total, ok := c.get("k1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if total != 42 {
		t.Errorf("total = %d, want 42 (cache must return real total, not -1 or 0)", total)
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

	// free and paid must differ from all (no filter)
	if freeQuery == allQuery {
		t.Error("free filter query identical to no-filter query — priceFilter not in SQL")
	}
	if paidQuery == allQuery {
		t.Error("paid filter query identical to no-filter query — priceFilter not in SQL")
	}
	if freeQuery == paidQuery {
		t.Error("free and paid queries are identical — priceFilter not differentiated")
	}
	// free has no args (clause is static), paid has no args either
	if len(freeArgs) != 0 || len(paidArgs) != 0 {
		t.Logf("freeArgs=%d paidArgs=%d (static clauses expected 0 args)", len(freeArgs), len(paidArgs))
	}
	// Verify the actual SQL contains the filter keywords
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

	// Manually expire
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
