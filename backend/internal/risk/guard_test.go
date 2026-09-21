package risk

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func guardReq(account, symbol, side, orderType, comment string, magic int32, volume, price string) *GuardRequest {
	v, _ := decimal.NewFromString(volume)
	p, _ := decimal.NewFromString(price)
	return &GuardRequest{
		AccountID: account, Symbol: symbol, Side: side,
		Volume: v, OrderType: orderType, Price: p,
		Comment: comment, Magic: magic,
	}
}

// RISK-DEDUP-KEY-1: the Guard-layer dedup key must carry account|magic|type
// (same shape as the Gate-layer DuplicateProtection rule). Two distinct
// same-tick orders — different magic or comment — are different intents and
// must pass; only a true identical resubmission is a duplicate.
//
// Adversarial (M1): revert the key to symbol|side|volume|type|price → the
// different-magic / different-comment cases turn RED (the probe failure).
func TestGuard_Dedup_DistinctIntentAllowed(t *testing.T) {
	g := NewGuard(&GuardConfig{DedupWindow: time.Hour})
	ctx := context.Background()

	if r := g.Check(ctx, guardReq("acct-1", "BTCUSDm", "buy", "market", "x1", 791, "0.01", "81300")); !r.Allowed {
		t.Fatalf("first order blocked: %s", r.Reason)
	}
	// Same params, different magic+comment — the live-probe failure case.
	if r := g.Check(ctx, guardReq("acct-1", "BTCUSDm", "buy", "market", "x2", 792, "0.01", "81300")); !r.Allowed {
		t.Fatalf("distinct-intent same-tick order wrongly deduped: %s", r.Reason)
	}
	// Same magic, different comment — still a distinct strategy intent.
	if r := g.Check(ctx, guardReq("acct-1", "BTCUSDm", "buy", "market", "x3", 791, "0.01", "81300")); !r.Allowed {
		t.Fatalf("same-magic different-comment order wrongly deduped: %s", r.Reason)
	}
	// Same everything except order type — limit vs market must not collide.
	if r := g.Check(ctx, guardReq("acct-1", "BTCUSDm", "buy", "limit", "x1", 791, "0.01", "81300")); !r.Allowed {
		t.Fatalf("different order type wrongly deduped: %s", r.Reason)
	}
	// Different account — must never cross-contaminate.
	if r := g.Check(ctx, guardReq("acct-2", "BTCUSDm", "buy", "market", "x1", 791, "0.01", "81300")); !r.Allowed {
		t.Fatalf("different account wrongly deduped: %s", r.Reason)
	}
}

func TestGuard_Dedup_TrueDuplicateStillBlocked(t *testing.T) {
	g := NewGuard(&GuardConfig{DedupWindow: time.Hour})
	ctx := context.Background()

	req := guardReq("acct-1", "BTCUSDm", "buy", "market", "retry-me", 791, "0.01", "81300")
	if r := g.Check(ctx, req); !r.Allowed {
		t.Fatalf("first order blocked: %s", r.Reason)
	}
	// Byte-identical resubmission (same account+magic+comment+params) — the
	// retry-safety property the dedup exists for.
	if r := g.Check(ctx, req); r.Allowed {
		t.Fatal("true duplicate (identical account+magic+comment+params) should be blocked")
	}
}
