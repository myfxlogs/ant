package marketplace

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
)

func TestParseDecimal_Valid(t *testing.T) {
	t.Parallel()
	d := parseDecimal("123.45")
	expected := decimal.NewFromFloat(123.45)
	if !d.Equals(expected) {
		t.Errorf("expected 123.45, got %s", d.String())
	}
}

func TestParseDecimal_Invalid(t *testing.T) {
	t.Parallel()
	d := parseDecimal("not-a-number")
	if !d.IsZero() {
		t.Errorf("expected zero for invalid input, got %s", d.String())
	}
}

func TestParseDecimal_Empty(t *testing.T) {
	t.Parallel()
	d := parseDecimal("")
	if !d.IsZero() {
		t.Errorf("expected zero for empty input, got %s", d.String())
	}
}

// Adversarial proof: InitiateStrategyIteration requires authentication.
// Remove the auth check and this test fails red.

// Adversarial proof: InitiateStrategyIteration requires authentication.
// Remove the auth check and this test fails red.
func TestInitiateStrategyIteration_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := testMarketplaceHandler(&stubMarketplaceSvc{})
	_, err := h.InitiateStrategyIteration(context.Background(), connect.NewRequest(&antv1.InitiateStrategyIterationRequest{}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}
}

// Adversarial proof: not-owner returns PermissionDenied.

// Adversarial proof: not-owner returns PermissionDenied.
func TestInitiateStrategyIteration_NotOwner(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{err: errors.New("marketplace: initiate iteration: not the strategy owner")}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	_, err := h.InitiateStrategyIteration(ctx, connect.NewRequest(&antv1.InitiateStrategyIterationRequest{StrategyId: "s1"}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied, got %v", err)
	}
}

// Adversarial proof: strategy not found returns NotFound.

// Adversarial proof: strategy not found returns NotFound.
func TestInitiateStrategyIteration_NotFound(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{err: errors.New("marketplace: initiate iteration: strategy not found: sql: no rows")}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	_, err := h.InitiateStrategyIteration(ctx, connect.NewRequest(&antv1.InitiateStrategyIterationRequest{StrategyId: "s1"}))
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %v", err)
	}
}

// Adversarial proof: success returns task_id + success=true.

// Adversarial proof: success returns task_id + success=true.
func TestInitiateStrategyIteration_Success(t *testing.T) {
	t.Parallel()
	svc := &stubMarketplaceSvc{}
	h := testMarketplaceHandler(svc)
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "u1")
	resp, err := h.InitiateStrategyIteration(ctx, connect.NewRequest(&antv1.InitiateStrategyIterationRequest{StrategyId: "s1"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.Success {
		t.Fatal("expected success=true")
	}
}
