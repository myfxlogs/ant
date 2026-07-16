package risksvc

import (
	"context"
	"sort"

	"github.com/shopspring/decimal"
)

// AllocAccount represents an account available for block trade allocation.
type AllocAccount struct {
	AccountID  string
	Equity     decimal.Decimal
	FreeMargin decimal.Decimal
	Priority   int // lower = higher priority for FIFO
}

// BlockAllocator distributes a total volume across accounts.
type BlockAllocator interface {
	Name() string
	Allocate(ctx context.Context, totalVolume decimal.Decimal, accounts []AllocAccount) map[string]decimal.Decimal
}

// ProRataAllocator allocates volume proportional to each account's equity.
type ProRataAllocator struct{}

func (a *ProRataAllocator) Name() string { return "pro_rata" }

func (a *ProRataAllocator) Allocate(_ context.Context, totalVolume decimal.Decimal, accounts []AllocAccount) map[string]decimal.Decimal {
	result := make(map[string]decimal.Decimal, len(accounts))
	totalEquity := decimal.Zero
	for _, acc := range accounts {
		if acc.Equity.GreaterThan(decimal.Zero) {
			totalEquity = totalEquity.Add(acc.Equity)
		}
	}
	if totalEquity.LessThanOrEqual(decimal.Zero) || totalVolume.LessThanOrEqual(decimal.Zero) {
		return result
	}
	remaining := totalVolume
	sorted := make([]AllocAccount, len(accounts))
	copy(sorted, accounts)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Equity.GreaterThan(sorted[j].Equity) })
	for i, acc := range sorted {
		if acc.Equity.LessThanOrEqual(decimal.Zero) {
			continue
		}
		share := totalVolume.Mul(acc.Equity).Div(totalEquity)
		if share.GreaterThan(acc.FreeMargin) && acc.FreeMargin.GreaterThan(decimal.Zero) {
			share = acc.FreeMargin
		}
		if i == len(sorted)-1 {
			share = remaining
		}
		if share.GreaterThan(remaining) {
			share = remaining
		}
		result[acc.AccountID] = share
		remaining = remaining.Sub(share)
	}
	return result
}

// FIFOAllocator allocates volume in priority order (lowest priority first).
type FIFOAllocator struct{}

func (a *FIFOAllocator) Name() string { return "fifo" }

func (a *FIFOAllocator) Allocate(_ context.Context, totalVolume decimal.Decimal, accounts []AllocAccount) map[string]decimal.Decimal {
	result := make(map[string]decimal.Decimal, len(accounts))
	if totalVolume.LessThanOrEqual(decimal.Zero) || len(accounts) == 0 {
		return result
	}
	sorted := make([]AllocAccount, len(accounts))
	copy(sorted, accounts)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Priority < sorted[j].Priority })
	remaining := totalVolume
	for i, acc := range sorted {
		share := remaining
		if i < len(sorted)-1 {
			if acc.FreeMargin.GreaterThan(decimal.Zero) && acc.FreeMargin.LessThan(share) {
				share = acc.FreeMargin
			}
		}
		if share.LessThanOrEqual(decimal.Zero) {
			continue
		}
		if share.GreaterThan(remaining) {
			share = remaining
		}
		result[acc.AccountID] = share
		remaining = remaining.Sub(share)
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
	}
	return result
}

// VWAPAllocator allocates volume weighted by account capacity (free margin).
type VWAPAllocator struct{}

func (a *VWAPAllocator) Name() string { return "vwap" }

func (a *VWAPAllocator) Allocate(_ context.Context, totalVolume decimal.Decimal, accounts []AllocAccount) map[string]decimal.Decimal {
	result := make(map[string]decimal.Decimal, len(accounts))
	totalCapacity := decimal.Zero
	for _, acc := range accounts {
		if acc.FreeMargin.GreaterThan(decimal.Zero) {
			totalCapacity = totalCapacity.Add(acc.FreeMargin)
		}
	}
	if totalCapacity.LessThanOrEqual(decimal.Zero) || totalVolume.LessThanOrEqual(decimal.Zero) {
		return result
	}
	remaining := totalVolume
	sorted := make([]AllocAccount, len(accounts))
	copy(sorted, accounts)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].FreeMargin.GreaterThan(sorted[j].FreeMargin) })
	for i, acc := range sorted {
		if acc.FreeMargin.LessThanOrEqual(decimal.Zero) {
			continue
		}
		share := totalVolume.Mul(acc.FreeMargin).Div(totalCapacity)
		if share.GreaterThan(acc.FreeMargin) {
			share = acc.FreeMargin
		}
		if i == len(sorted)-1 {
			share = remaining
		}
		if share.GreaterThan(remaining) {
			share = remaining
		}
		result[acc.AccountID] = share
		remaining = remaining.Sub(share)
	}
	return result
}
