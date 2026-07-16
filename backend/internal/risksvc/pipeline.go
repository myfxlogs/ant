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

	"github.com/shopspring/decimal"
)

// SignalRequest represents an incoming trade signal from the strategy/quant engine.
type SignalRequest struct {
	UserID         string
	AccountID      string
	Symbol         string
	Side           string // "buy" / "sell"
	SignalStrength float64 // 0–1, how strong the signal is

	// Market data for sizing
	Price          decimal.Decimal
	ATR            decimal.Decimal
	AnnualVol      float64
	ContractSize   decimal.Decimal
	HoldingDays    float64
	ContractExpiry time.Time // zero if spot

	// Account state
	Balance    decimal.Decimal
	Equity     decimal.Decimal
	FreeMargin decimal.Decimal
	Margin     decimal.Decimal // currently used margin
	Positions  int             // current open position count

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

	Lots        decimal.Decimal
	Allocations map[string]decimal.Decimal // accountID → lots (multi-account only)
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
	if result := p.checkCapability(sig); !result.Allowed { return result }
	if result := p.checkHardLimit(ctx, sig); !result.Allowed { return result }
	if result := p.checkPlatformLimits(); !result.Allowed { return result }
	if result := p.checkRiskEngine(ctx, sig); !result.Allowed { return result }
	return p.sizePosition(ctx, sig)
}

func (p *SignalPipeline) checkCapability(sig *SignalRequest) *SignalResult {
	if p.capStore == nil { return &SignalResult{Allowed: true} }
	cap := p.capStore.Get(sig.UserID)
	pre := cap.TierCheck()
	if !pre.Allowed { return &SignalResult{Allowed: false, Reason: pre.Reason, Stage: "capability"} }
	return &SignalResult{Allowed: true}
}

func (p *SignalPipeline) checkHardLimit(ctx context.Context, sig *SignalRequest) *SignalResult {
	if p.hardLimit == nil { return &SignalResult{Allowed: true} }
	req := &HardLimitRequest{
		UserID: sig.UserID, AccountID: sig.AccountID, Symbol: sig.Symbol,
		Side: sig.Side, Volume: decimal.NewFromFloat(sig.SignalStrength), Price: sig.Price,
		Balance: sig.Balance, Equity: sig.Equity, FreeMargin: sig.FreeMargin,
		ContractExpiry: sig.ContractExpiry, ClientIP: sig.ClientIP,
	}
	if err := p.hardLimit.Evaluate(ctx, req); err != nil {
		return &SignalResult{Allowed: false, Reason: err.Error(), Stage: "hardlimit"}
	}
	return &SignalResult{Allowed: true}
}

func (p *SignalPipeline) checkPlatformLimits() *SignalResult {
	if p.platform == nil || p.limits == nil { return &SignalResult{Allowed: true} }
	exposure := p.platform.GetSnapshot()
	if exposure == nil { return &SignalResult{Allowed: true} }
	result := p.limits.Check(exposure)
	if !result.Allowed { return &SignalResult{Allowed: false, Reason: result.Reason, Stage: "platform_limits"} }
	return &SignalResult{Allowed: true}
}

func (p *SignalPipeline) checkRiskEngine(ctx context.Context, sig *SignalRequest) *SignalResult {
	if p.engine == nil { return &SignalResult{Allowed: true} }
	check := &CheckRequest{
		UserID: sig.UserID, AccountID: sig.AccountID, Symbol: sig.Symbol,
		Side: sig.Side, Volume: decimal.NewFromFloat(sig.SignalStrength), Price: sig.Price,
		Balance: sig.Balance, Equity: sig.Equity, Margin: sig.Margin,
		Positions: sig.Positions,
	}
	result := p.engine.Evaluate(ctx, check)
	if !result.Passed { return &SignalResult{Allowed: false, Reason: result.Reason, Stage: "risk_engine"} }
	return &SignalResult{Allowed: true}
}

func (p *SignalPipeline) sizePosition(ctx context.Context, sig *SignalRequest) *SignalResult {
	if p.sizer == nil {
		return &SignalResult{Allowed: false, Reason: "no sizer configured", Stage: "sizer"}
	}
	sreq := &SizerRequest{
		Symbol: sig.Symbol, Price: sig.Price, ATR: sig.ATR, AnnualVol: sig.AnnualVol,
		ContractSize: sig.ContractSize, HoldingDays: sig.HoldingDays,
		AccountID: sig.AccountID, Balance: sig.Balance, Equity: sig.Equity, FreeMargin: sig.FreeMargin,
	}
	sres, err := p.sizer.Size(ctx, sreq)
	if err != nil { return &SignalResult{Allowed: false, Reason: err.Error(), Stage: "sizer"} }
	if sres.Lots.LessThanOrEqual(decimal.Zero) {
		return &SignalResult{Allowed: true, Reason: "sizer passthrough (manual order)", Stage: "sizer", RiskUsed: sres.RiskUsed, Method: sres.Method}
	}
	if p.allocator != nil && len(sig.TargetAccounts) > 0 {
		allocs := p.allocator.Allocate(ctx, sres.Lots, sig.TargetAccounts)
		return &SignalResult{Allowed: true, Stage: "complete", Lots: sres.Lots, Allocations: allocs, RiskUsed: sres.RiskUsed, Method: sres.Method}
	}
	return &SignalResult{Allowed: true, Stage: "complete", Lots: sres.Lots, RiskUsed: sres.RiskUsed, Method: sres.Method}
}
