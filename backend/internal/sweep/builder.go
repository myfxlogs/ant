package sweep

import (
	"context"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
)

// SweepConfig holds the runtime configuration for the sweep builder.
type SweepConfig struct {
	ColdWalletAddress    string
	EnergyAccountAddress string
	USDTContractAddress  string
	DEMFactor            string // e.g. "1.3"
	EnergyBufferPercent  string // e.g. "10"
	EnergyTRXRate        string // TRX (SUN) per energy unit, e.g. "1" = 1 SUN per energy
	FeeLimit             int64  // TRC20 transfer fee limit in SUN
	BaseEnergyFirst      int64  // energy for first sweep (has_received_usdt=false)
	BaseEnergySubsequent int64  // energy for subsequent sweeps
	RawTxExpiryHours     int    // raw tx expiry window in hours
}

// Validate checks that required config values are set.
func (c SweepConfig) Validate() error {
	if c.ColdWalletAddress == "" {
		return fmt.Errorf("sweep builder: cold_wallet_address not configured")
	}
	if c.EnergyAccountAddress == "" {
		return fmt.Errorf("sweep builder: energy_account_address not configured")
	}
	if c.USDTContractAddress == "" {
		return fmt.Errorf("sweep builder: usdt_contract_address not configured")
	}
	return nil
}

// Builder constructs unsigned sweep bundles for cold signing.
// It uses the Tron gRPC client to build raw transactions without signing (R1: watch-only).
type Builder struct {
	tron            *TronClient
	addrRepo        *repository.DepositAddressRepository
	adminRepo       *repository.AdminRepository
	log             *zap.Logger
	xpubFingerprint string // ADR-0026 R5: set on every bundle for coldsign verification
}

func NewBuilder(tron *TronClient, addrRepo *repository.DepositAddressRepository, adminRepo *repository.AdminRepository, xpubFingerprint string, log *zap.Logger) *Builder {
	return &Builder{tron: tron, addrRepo: addrRepo, adminRepo: adminRepo, xpubFingerprint: xpubFingerprint, log: log}
}

// loadConfig reads sweep-related config from system_config.
func (b *Builder) loadConfig(ctx context.Context) (SweepConfig, error) {
	get := func(key string) string {
		cfg, err := b.adminRepo.GetConfig(ctx, key)
		if err != nil || cfg == nil {
			return ""
		}
		return cfg.Value
	}

	demFactor := get("dem_factor")
	if demFactor == "" {
		demFactor = "1.3"
	}
	energyBuffer := get("energy_buffer_percent")
	if energyBuffer == "" {
		energyBuffer = "10"
	}
	energyTRXRate := get("energy_trx_rate")
	if energyTRXRate == "" {
		energyTRXRate = "1" // 1 SUN per energy unit (conservative default)
	}

	feeLimit := int64(1_000_000_000) // 1000 TRX in SUN — generous for USDT transfers
	if v := get("sweep_fee_limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			feeLimit = n
		}
	}
	baseEnergyFirst := int64(130_000)
	if v := get("sweep_base_energy_first"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			baseEnergyFirst = n
		}
	}
	baseEnergySubsequent := int64(65_000)
	if v := get("sweep_base_energy_subsequent"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			baseEnergySubsequent = n
		}
	}
	rawTxExpiryHours := 23
	if v := get("sweep_raw_tx_expiry_hours"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rawTxExpiryHours = n
		}
	}

	return SweepConfig{
		ColdWalletAddress:    get("cold_wallet_address"),
		EnergyAccountAddress: get("energy_account_address"),
		USDTContractAddress:  get("usdt_contract_address"),
		DEMFactor:            demFactor,
		EnergyBufferPercent:  energyBuffer,
		EnergyTRXRate:        energyTRXRate,
		FeeLimit:             feeLimit,
		BaseEnergyFirst:      baseEnergyFirst,
		BaseEnergySubsequent: baseEnergySubsequent,
		RawTxExpiryHours:     rawTxExpiryHours,
	}, nil
}

// calculateEnergy computes the energy amount for delegate/undelegate legs.
// First sweep (has_received_usdt=false): 130k × dem_factor + buffer%.
// Subsequent sweeps (has_received_usdt=true): 65k × dem_factor + buffer%.
func calculateEnergy(hasReceivedUSDT bool, demFactor, bufferPercent string, baseFirst, baseSubsequent int64) (int64, error) {
	baseEnergy := decimal.NewFromInt(baseSubsequent)
	if !hasReceivedUSDT {
		baseEnergy = decimal.NewFromInt(baseFirst)
	}

	dem, err := decimal.NewFromString(demFactor)
	if err != nil {
		return 0, fmt.Errorf("sweep builder: parse dem_factor: %w", err)
	}
	buffer, err := decimal.NewFromString(bufferPercent)
	if err != nil {
		return 0, fmt.Errorf("sweep builder: parse energy_buffer_percent: %w", err)
	}

	energy := baseEnergy.Mul(dem).Mul(decimal.NewFromInt(1).Add(buffer.Div(decimal.NewFromInt(100))))
	return energy.IntPart(), nil
}

// energyToTRX converts energy units to TRX amount in SUN for DelegateResource.
// TRON Stake 2.0 DelegateResource takes delegateBalance in SUN (frozen TRX),
// not energy units. The conversion rate is configurable (energy_trx_rate).
func energyToTRX(energy int64, rateStr string) (int64, error) {
	rate, err := decimal.NewFromString(rateStr)
	if err != nil {
		return 0, fmt.Errorf("sweep builder: parse energy_trx_rate: %w", err)
	}
	if rate.LessThanOrEqual(decimal.Zero) {
		return 0, fmt.Errorf("sweep builder: energy_trx_rate must be positive")
	}
	return decimal.NewFromInt(energy).Mul(rate).IntPart(), nil
}

// BuildUnsignedBundle constructs an UnsignedSweepBundle for a single address.
// Each address produces 3 unsigned transactions: delegate, transfer, undelegate.
// Raw transactions have ~24h expiry for crash recovery window (Q3).
