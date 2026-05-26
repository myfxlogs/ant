package risksvc

import (
	"context"
	"sort"
)

// AllocAccount represents an account available for block trade allocation.
type AllocAccount struct {
	AccountID string
	Equity    float64
	FreeMargin float64
	Priority  int // lower = higher priority for FIFO
}

// BlockAllocator distributes a total volume across accounts.
type BlockAllocator interface {
	Name() string
	Allocate(ctx context.Context, totalVolume float64, accounts []AllocAccount) map[string]float64
}

// ProRataAllocator allocates volume proportional to each account's equity.
type ProRataAllocator struct{}

func (a *ProRataAllocator) Name() string { return "pro_rata" }

func (a *ProRataAllocator) Allocate(_ context.Context, totalVolume float64, accounts []AllocAccount) map[string]float64 {
	result := make(map[string]float64, len(accounts))
	totalEquity := 0.0
	for _, acc := range accounts {
		if acc.Equity > 0 {
			totalEquity += acc.Equity
		}
	}
	if totalEquity <= 0 || totalVolume <= 0 {
		return result
	}
	remaining := totalVolume
	sorted := make([]AllocAccount, len(accounts))
	copy(sorted, accounts)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Equity > sorted[j].Equity })
	for i, acc := range sorted {
		if acc.Equity <= 0 {
			continue
		}
		share := totalVolume * (acc.Equity / totalEquity)
		if share > acc.FreeMargin && acc.FreeMargin > 0 {
			share = acc.FreeMargin
		}
		if i == len(sorted)-1 {
			share = remaining
		}
		if share > remaining {
			share = remaining
		}
		result[acc.AccountID] = share
		remaining -= share
	}
	return result
}

// FIFOAllocator allocates volume in priority order (lowest priority first).
type FIFOAllocator struct{}

func (a *FIFOAllocator) Name() string { return "fifo" }

func (a *FIFOAllocator) Allocate(_ context.Context, totalVolume float64, accounts []AllocAccount) map[string]float64 {
	result := make(map[string]float64, len(accounts))
	if totalVolume <= 0 || len(accounts) == 0 {
		return result
	}
	sorted := make([]AllocAccount, len(accounts))
	copy(sorted, accounts)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Priority < sorted[j].Priority })
	remaining := totalVolume
	for i, acc := range sorted {
		share := remaining
		if i < len(sorted)-1 {
			if acc.FreeMargin > 0 && acc.FreeMargin < share {
				share = acc.FreeMargin
			}
		}
		if share <= 0 {
			continue
		}
		if share > remaining {
			share = remaining
		}
		result[acc.AccountID] = share
		remaining -= share
		if remaining <= 0 {
			break
		}
	}
	return result
}

// VWAPAllocator allocates volume weighted by account capacity (free margin).
type VWAPAllocator struct{}

func (a *VWAPAllocator) Name() string { return "vwap" }

func (a *VWAPAllocator) Allocate(_ context.Context, totalVolume float64, accounts []AllocAccount) map[string]float64 {
	result := make(map[string]float64, len(accounts))
	totalCapacity := 0.0
	for _, acc := range accounts {
		if acc.FreeMargin > 0 {
			totalCapacity += acc.FreeMargin
		}
	}
	if totalCapacity <= 0 || totalVolume <= 0 {
		return result
	}
	remaining := totalVolume
	sorted := make([]AllocAccount, len(accounts))
	copy(sorted, accounts)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].FreeMargin > sorted[j].FreeMargin })
	for i, acc := range sorted {
		if acc.FreeMargin <= 0 {
			continue
		}
		share := totalVolume * (acc.FreeMargin / totalCapacity)
		if share > acc.FreeMargin {
			share = acc.FreeMargin
		}
		if i == len(sorted)-1 {
			share = remaining
		}
		if share > remaining {
			share = remaining
		}
		result[acc.AccountID] = share
		remaining -= share
	}
	return result
}
