package risksvc

import (
	"context"
	"testing"
)

func TestCountryCode_NilResolver(t *testing.T) {
	t.Parallel()
	// nil resolver should return ErrGeoIPUnavailable.
	_, err := (*MaxMindGeoIPResolver)(nil).CountryCode("1.2.3.4")
	if err == nil {
		t.Fatal("expected error for nil resolver")
	}
}

func TestCountryCode_EmptyDBPath(t *testing.T) {
	t.Parallel()
	r := &MaxMindGeoIPResolver{dbPath: ""}
	_, err := r.CountryCode("1.2.3.4")
	if err == nil {
		t.Fatal("expected error for empty dbPath")
	}
}

func TestPipeline_Process_NilSizer(t *testing.T) {
	t.Parallel()
	capStore := NewCapabilityStore()
	capStore.Set(&Capability{UserID: "u1", Tier: Tier2LiveLimited})

	p := NewSignalPipeline(PipelineConfig{
		CapStore: capStore,
		// Sizer is nil — should reject at size stage.
	})

	sig := &SignalRequest{
		UserID: "u1", AccountID: "a1", Symbol: "EURUSD", Side: "buy",
		SignalStrength: 1.0, Price: decF(1.085),
	}
	result := p.Process(context.Background(), sig)
	if result.Allowed {
		t.Fatal("expected rejection without sizer")
	}
	if result.Stage != "sizer" {
		t.Fatalf("expected stage sizer, got %s", result.Stage)
	}
}

func TestCapabilityStore_Overwrite(t *testing.T) {
	t.Parallel()
	store := NewCapabilityStore()
	store.Set(&Capability{UserID: "u1", Tier: Tier2LiveLimited})
	c := store.Get("u1")
	if c == nil || c.Tier != Tier2LiveLimited {
		t.Fatalf("expected Tier2LiveLimited, got %v", c)
	}
	// Overwrite.
	store.Set(&Capability{UserID: "u1", Tier: Tier0ViewOnly})
	c = store.Get("u1")
	if c == nil || c.Tier != Tier0ViewOnly {
		t.Fatalf("expected Tier0ViewOnly after overwrite, got %v", c)
	}
}
