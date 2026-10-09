package risk

import (
	"context"
	"testing"
	"time"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func TestDuplicateProtection_Allow(t *testing.T) {
	r := &DuplicateProtection{DedupWindow: 5 * time.Second}
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestDuplicateProtection_Block(t *testing.T) {
	r := &DuplicateProtection{DedupWindow: time.Hour}
	r.Check(context.Background(), intentBuy("0.1"), nil)
	// Same intent within dedup window.
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if result.Allowed {
		t.Error("expected duplicate blocked")
	}
}

func TestDuplicateProtection_DifferentVolAllowed(t *testing.T) {
	r := &DuplicateProtection{DedupWindow: time.Hour}
	r.Check(context.Background(), intentBuy("0.1"), nil)
	result := r.Check(context.Background(), intentBuy("0.2"), nil)
	if !result.Allowed {
		t.Errorf("different volume should be allowed")
	}
}

// ── Task 3 (DEDUP-5S-THROTTLE): dedup key includes AccountID + Magic ──

// TestTask3_DifferentTicket_Close_Allowed verifies that two close intents
// with different tickets (Magic) from the same account within 5s are both
// allowed. Revert to old key (without Magic) → second close rejected → RED.

// TestTask3_DifferentTicket_Close_Allowed verifies that two close intents
// with different tickets (Magic) from the same account within 5s are both
// allowed. Revert to old key (without Magic) → second close rejected → RED.
func TestTask3_DifferentTicket_Close_Allowed(t *testing.T) {
	r := &DuplicateProtection{DedupWindow: 5 * time.Second}
	intent1 := &antv1.OrderIntent{
		AccountId: "acct-1", Symbol: "", Side: "", Volume: "0",
		Type: "close", Price: "", Magic: 100,
		Source: antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
	}
	intent2 := &antv1.OrderIntent{
		AccountId: "acct-1", Symbol: "", Side: "", Volume: "0",
		Type: "close", Price: "", Magic: 200,
		Source: antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
	}
	r.Check(context.Background(), intent1, nil)
	result := r.Check(context.Background(), intent2, nil)
	if !result.Allowed {
		t.Fatalf("different ticket close should be allowed, got: %s", result.Reason)
	}
}

// TestTask3_DifferentAccount_Allowed verifies that same params from different
// accounts are both allowed. Revert to old key (without AccountID) → RED.

// TestTask3_DifferentAccount_Allowed verifies that same params from different
// accounts are both allowed. Revert to old key (without AccountID) → RED.
func TestTask3_DifferentAccount_Allowed(t *testing.T) {
	r := &DuplicateProtection{DedupWindow: 5 * time.Second}
	intent1 := intentBuy("0.1")
	intent1.AccountId = "acct-1"
	intent2 := intentBuy("0.1")
	intent2.AccountId = "acct-2"
	r.Check(context.Background(), intent1, nil)
	result := r.Check(context.Background(), intent2, nil)
	if !result.Allowed {
		t.Fatalf("different account should be allowed, got: %s", result.Reason)
	}
}

// TestTask3_SameKey_StillBlocked verifies that true duplicates (same account,
// same magic, same params) are still blocked — prevents over-fixing.

// TestTask3_SameKey_StillBlocked verifies that true duplicates (same account,
// same magic, same params) are still blocked — prevents over-fixing.
func TestTask3_SameKey_StillBlocked(t *testing.T) {
	r := &DuplicateProtection{DedupWindow: 5 * time.Second}
	r.Check(context.Background(), intentBuy("0.1"), nil)
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if result.Allowed {
		t.Fatal("true duplicate (same account+magic+params) should be blocked")
	}
}

// ── R9: Margin Pre-Check ──────────────────────────────────────────────

// RISK-DEDUP-KEY-1 (gate layer): two same-param orders with distinct comments
// are distinct intents (grid/pyramid strategies place same-price orders with
// different comments); an identical retry (same comment) must still block.
// Adversarial: remove Comment from the key → first case fails → RED.
func TestDuplicateProtection_DistinctComment_Allowed(t *testing.T) {
	r := &DuplicateProtection{DedupWindow: 5 * time.Second}
	base := func(comment string) *antv1.OrderIntent {
		return &antv1.OrderIntent{
			AccountId: "acct-1", Symbol: "BTCUSDm", Side: "buy", Volume: "0.01",
			Type: "market", Price: "81347.02", Magic: 71001, Comment: comment,
			Source: antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
		}
	}
	if res := r.Check(context.Background(), base("probe4_v1"), nil); !res.Allowed {
		t.Fatalf("first order blocked: %s", res.Reason)
	}
	if res := r.Check(context.Background(), base("probe4_v2"), nil); !res.Allowed {
		t.Fatalf("distinct comment must be a distinct intent, got: %s", res.Reason)
	}
	if res := r.Check(context.Background(), base("probe4_v1"), nil); res.Allowed {
		t.Fatal("identical retry (same comment) must still be deduped")
	}
}
