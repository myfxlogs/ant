package marketplace

import (
	"context"
	"testing"
)

// ── Price model whitelist validation tests ──

func TestPublish_InvalidPriceModel(t *testing.T) {
	s := &Service{}
	_, err := s.Publish(context.Background(), PublishParams{PriceModel: "monthly"})
	if err == nil {
		t.Fatal("expected error for invalid price model 'monthly'")
	}
}

func TestSetPricing_InvalidPriceModel(t *testing.T) {
	s := &Service{}
	err := s.SetPricing(context.Background(), "00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002", "monthly", "10", "0.1")
	if err == nil {
		t.Fatal("expected error for invalid price model 'monthly'")
	}
}
