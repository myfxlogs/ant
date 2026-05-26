// Package risksvc provides the SignalPipeline (M10-BASE-C6).
//
// SignalPipeline connects the full pre-trade flow:
//
//	Signal → Capability → HardLimit → PlatformLimits → PreCheck → Sizer → (BlockAlloc) → Result
//
// Each stage can independently block or modify the order before it reaches the broker.

package risksvc

import (
	"context"
	"time"
)

// SignalRequest represents an incoming trade signal from the strategy/quant engine.
type SignalRequest struct {
	UserID         string
	AccountID      string
	Symbol         string
	Side           string // "buy" / "sell"
	SignalStrength float64 // 0–1, how strong the signal is

	// Market data for sizing
	Price         float64
	ATR           float64
	AnnualVol     float64
	ContractSize  float64
	HoldingDays   float64
	ContractExpiry time.Time // zero if spot

	// Account state
	Balance    float64
	Equity     float64
	FreeMargin float64
	Margin     float64 // currently used margin
	Positions  int     // current open position count

	// Multi-account block trade (optional)
	TargetAccounts []AllocAccount

	// ClientIP is extracted from the incoming request for GeoIP jurisdiction checks.
	ClientIP string
}

// SignalResult is the outcome of the signal pipeline.
type SignalResult struct {
	Allowed bool
	Reason  string
	Stage   string // which stage produced the result

	Lots        float64
	Allocations map[string]float64 // accountID → lots (multi-account only)
	RiskUsed    float64
	Method      string // sizer name
}

// SignalPipeline orchestrates the pre-trade risk and sizing flow.
type SignalPipeline struct {
	capStore   *CapabilityStore
	hardLimit  *HardLimitEvaluator
	platform   *PlatformAggregator
	limits     *PlatformLimits
	engine     *Engine
	sizer      PositionSizer
	allocator  BlockAllocator
}

// PipelineConfig bundles optional pipeline components.
type PipelineConfig struct {
	CapStore  *CapabilityStore
	HardLimit *HardLimitEvaluator
	Platform  *PlatformAggregator
	Limits    *PlatformLimits
	Engine    *Engine
	Sizer     PositionSizer
	Allocator BlockAllocator
}

// NewSignalPipeline creates a signal pipeline from the given config.
func NewSignalPipeline(cfg PipelineConfig) *SignalPipeline {
	return &SignalPipeline{
		capStore:  cfg.CapStore,
		hardLimit: cfg.HardLimit,
		platform:  cfg.Platform,
		limits:    cfg.Limits,
		engine:    cfg.Engine,
		sizer:     cfg.Sizer,
		allocator: cfg.Allocator,
	}
}

// Process runs the full signal-to-decision pipeline.
func (p *SignalPipeline) Process(ctx context.Context, sig *SignalRequest) *SignalResult {
	// Stage 1: Capability tier check
	if p.capStore != nil {
		cap := p.capStore.Get(sig.UserID)
		pre := cap.TierCheck()
		if !pre.Allowed {
			return &SignalResult{Allowed: false, Reason: pre.Reason, Stage: "capability"}
		}
	}

	// Stage 2: HardLimit evaluation (binary deny)
	if p.hardLimit != nil {
		req := &HardLimitRequest{
			UserID:         sig.UserID,
			AccountID:      sig.AccountID,
			Symbol:         sig.Symbol,
			Side:           sig.Side,
			Volume:         sig.SignalStrength,
			Price:          sig.Price,
			Balance:        sig.Balance,
			Equity:         sig.Equity,
			FreeMargin:     sig.FreeMargin,
			ContractExpiry: sig.ContractExpiry,
			ClientIP:       sig.ClientIP,
		}
		if err := p.hardLimit.Evaluate(ctx, req); err != nil {
			return &SignalResult{Allowed: false, Reason: err.Error(), Stage: "hardlimit"}
		}
	}

	// Stage 3: Platform limits check
	if p.platform != nil && p.limits != nil {
		exposure := p.platform.GetSnapshot()
		if exposure != nil {
			result := p.limits.Check(exposure)
			if !result.Allowed {
				return &SignalResult{Allowed: false, Reason: result.Reason, Stage: "platform_limits"}
			}
		}
	}

	// Stage 4: Risk engine pre-check
	if p.engine != nil {
		check := &CheckRequest{
			UserID:    sig.UserID,
			AccountID: sig.AccountID,
			Symbol:    sig.Symbol,
			Side:      sig.Side,
			Volume:    sig.SignalStrength,
			Price:     sig.Price,
			Balance:   sig.Balance,
			Equity:    sig.Equity,
			Margin:    sig.Margin,
			Positions: sig.Positions,
		}
		result := p.engine.Evaluate(ctx, check)
		if !result.Passed {
			return &SignalResult{Allowed: false, Reason: result.Reason, Stage: "risk_engine"}
		}
	}

	// Stage 5: Position sizing
	if p.sizer == nil {
		return &SignalResult{Allowed: false, Reason: "no sizer configured", Stage: "sizer"}
	}
	sreq := &SizerRequest{
		Symbol:       sig.Symbol,
		Price:        sig.Price,
		ATR:          sig.ATR,
		AnnualVol:    sig.AnnualVol,
		ContractSize: sig.ContractSize,
		HoldingDays:  sig.HoldingDays,
		AccountID:    sig.AccountID,
		Balance:      sig.Balance,
		Equity:       sig.Equity,
		FreeMargin:   sig.FreeMargin,
	}
	sres, err := p.sizer.Size(ctx, sreq)
	if err != nil {
		return &SignalResult{Allowed: false, Reason: err.Error(), Stage: "sizer"}
	}
	if sres.Lots <= 0 {
		return &SignalResult{Allowed: false, Reason: "sizer returned zero lots", Stage: "sizer", RiskUsed: sres.RiskUsed, Method: sres.Method}
	}

	// Stage 6: Block allocation (multi-account)
	if p.allocator != nil && len(sig.TargetAccounts) > 0 {
		allocs := p.allocator.Allocate(ctx, sres.Lots, sig.TargetAccounts)
		return &SignalResult{
			Allowed:     true,
			Stage:       "complete",
			Lots:        sres.Lots,
			Allocations: allocs,
			RiskUsed:    sres.RiskUsed,
			Method:      sres.Method,
		}
	}

	return &SignalResult{
		Allowed:  true,
		Stage:    "complete",
		Lots:     sres.Lots,
		RiskUsed: sres.RiskUsed,
		Method:   sres.Method,
	}
}
