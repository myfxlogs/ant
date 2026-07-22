package sweep

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
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
	tron           *TronClient
	addrRepo       *repository.DepositAddressRepository
	adminRepo      *repository.AdminRepository
	log            *zap.Logger
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
func (b *Builder) BuildUnsignedBundle(ctx context.Context, addr *model.DepositAddress, amount string) (*antv1.UnsignedSweepBundle, error) {
	cfg, err := b.loadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	energyAmount, err := calculateEnergy(addr.HasReceivedUSDT, cfg.DEMFactor, cfg.EnergyBufferPercent, cfg.BaseEnergyFirst, cfg.BaseEnergySubsequent)
	if err != nil {
		return nil, err
	}
	// Convert energy units to TRX (SUN) for DelegateResource.
	delegateTRX, err := energyToTRX(energyAmount, cfg.EnergyTRXRate)
	if err != nil {
		return nil, err
	}

	// Parse amount as decimal (DB stores human-readable USDT, e.g. "1.500000").
	amountDec, err := decimal.NewFromString(amount)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: parse amount %q: %w", amount, err)
	}
	// USDT has 6 decimals — convert to smallest unit (integer) for TRC20 transfer.
	usdtDecimals := decimal.NewFromInt(1_000_000)
	amountSmallest := amountDec.Mul(usdtDecimals).BigInt()

	expiryMs := time.Now().Add(time.Duration(cfg.RawTxExpiryHours) * time.Hour).UnixMilli()
	batchID := uuid.New().String()

	// Leg 0: DelegateResource (energy_account → deposit_address)
	// delegateBalance is in SUN (frozen TRX amount), not energy units.
	delegateRaw, delegateTxID, err := b.tron.BuildDelegateResource(ctx, cfg.EnergyAccountAddress, addr.Address, delegateTRX)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: build delegate: %w", err)
	}
	delegateRaw, err = SetTxExpiry(delegateRaw, expiryMs)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: set delegate expiry: %w", err)
	}

	// Leg 1: TRC20 Transfer (deposit_address → cold_wallet)
	transferRaw, transferTxID, err := b.tron.BuildTRC20Transfer(ctx, addr.Address, cfg.ColdWalletAddress, cfg.USDTContractAddress, amountSmallest, cfg.FeeLimit)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: build transfer: %w", err)
	}
	transferRaw, err = SetTxExpiry(transferRaw, expiryMs)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: set transfer expiry: %w", err)
	}

	// Leg 2: UndelegateResource (deposit_address → energy_account)
	// Reclaim the same TRX amount that was delegated.
	undelegateRaw, undelegateTxID, err := b.tron.BuildUndelegateResource(ctx, cfg.EnergyAccountAddress, addr.Address, delegateTRX)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: build undelegate: %w", err)
	}
	undelegateRaw, err = SetTxExpiry(undelegateRaw, expiryMs)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: set undelegate expiry: %w", err)
	}

	bundle := &antv1.UnsignedSweepBundle{
		BundleId:        batchID,
		BuiltAtMs:       time.Now().UnixMilli(),
		XpubFingerprint: b.xpubFingerprint, // ADR-0026 R5: coldsign verifies this matches its own derivation
		Txs: []*antv1.UnsignedTx{
			{
				Kind:            antv1.TxKind_TX_KIND_DELEGATE,
				FromAddress:     cfg.EnergyAccountAddress,
				ToAddress:       addr.Address,
				Amount:          strconv.FormatInt(delegateTRX, 10),
				RawTx:           delegateRaw,
				ExpiryMs:        expiryMs,
				ExpectedTxid:    delegateTxID,
				DerivationIndex: 0,
				Tx: &antv1.UnsignedTx_Delegate{
					Delegate: &antv1.DelegateTx{
						EnergyAccount: cfg.EnergyAccountAddress,
						Resource:      "ENERGY",
					},
				},
			},
			{
				Kind:            antv1.TxKind_TX_KIND_TRANSFER,
				FromAddress:     addr.Address,
				ToAddress:       cfg.ColdWalletAddress,
				Amount:          amount,
				RawTx:           transferRaw,
				ExpiryMs:        expiryMs,
				ExpectedTxid:    transferTxID,
				DerivationIndex: uint32(addr.DerivationIndex),
				Tx: &antv1.UnsignedTx_Transfer{
					Transfer: &antv1.TransferTx{
						TokenContract: cfg.USDTContractAddress,
					},
				},
			},
			{
				Kind:            antv1.TxKind_TX_KIND_UNDELEGATE,
				FromAddress:     cfg.EnergyAccountAddress,
				ToAddress:       addr.Address,
				Amount:          strconv.FormatInt(delegateTRX, 10),
				RawTx:           undelegateRaw,
				ExpiryMs:        expiryMs,
				ExpectedTxid:    undelegateTxID,
				DerivationIndex: 0,
				Tx: &antv1.UnsignedTx_Undelegate{
					Undelegate: &antv1.UndelegateTx{
						EnergyAccount: cfg.EnergyAccountAddress,
						Resource:      "ENERGY",
					},
				},
			},
		},
	}

	b.log.Info("sweep bundle built",
		zap.String("batch_id", batchID),
		zap.String("address", addr.Address),
		zap.String("amount", amount),
		zap.Int64("energy", energyAmount),
		zap.Bool("first_sweep", !addr.HasReceivedUSDT))

	return bundle, nil
}

// BuildUndelegateOnlyBundle constructs an unsigned bundle containing only
// undelegate transactions for the given addresses (C5: stuck energy recovery).
// Used when a sweep bundle is stuck in MANUAL_REVIEW after delegate succeeded
// but transfer failed — the operator can recover the delegated energy by
// signing and broadcasting just the undelegate legs.
// The deposit addresses' USDT remains safe — only frozen TRX (energy) is reclaimed.
func (b *Builder) BuildUndelegateOnlyBundle(ctx context.Context, entries []BatchSweepEntry) (*antv1.UnsignedSweepBundle, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("sweep builder: undelegate-only: empty entries")
	}

	cfg, err := b.loadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: undelegate-only: load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("sweep builder: undelegate-only: %w", err)
	}

	expiryMs := time.Now().Add(time.Duration(cfg.RawTxExpiryHours) * time.Hour).UnixMilli()
	batchID := uuid.New().String()

	var txs []*antv1.UnsignedTx
	for _, e := range entries {
		// Use the same energy calculation to know how much TRX to undelegate.
		energyAmount, err := calculateEnergy(e.Addr.HasReceivedUSDT, cfg.DEMFactor, cfg.EnergyBufferPercent, cfg.BaseEnergyFirst, cfg.BaseEnergySubsequent)
		if err != nil {
			return nil, fmt.Errorf("sweep builder: undelegate-only: calculate energy: %w", err)
		}
		delegateTRX, err := energyToTRX(energyAmount, cfg.EnergyTRXRate)
		if err != nil {
			return nil, fmt.Errorf("sweep builder: undelegate-only: energy to trx: %w", err)
		}

		undelegateRaw, undelegateTxID, err := b.tron.BuildUndelegateResource(ctx, cfg.EnergyAccountAddress, e.Addr.Address, delegateTRX)
		if err != nil {
			return nil, fmt.Errorf("sweep builder: undelegate-only: build %s: %w", e.Addr.Address, err)
		}
		undelegateRaw, err = SetTxExpiry(undelegateRaw, expiryMs)
		if err != nil {
			return nil, fmt.Errorf("sweep builder: undelegate-only: expiry: %w", err)
		}

		txs = append(txs, &antv1.UnsignedTx{
			Kind:            antv1.TxKind_TX_KIND_UNDELEGATE,
			FromAddress:     cfg.EnergyAccountAddress,
			ToAddress:       e.Addr.Address,
			Amount:          strconv.FormatInt(delegateTRX, 10),
			RawTx:           undelegateRaw,
			ExpiryMs:        expiryMs,
			ExpectedTxid:    undelegateTxID,
			DerivationIndex: 0,
			Tx: &antv1.UnsignedTx_Undelegate{
				Undelegate: &antv1.UndelegateTx{
					EnergyAccount: cfg.EnergyAccountAddress,
					Resource:      "ENERGY",
				},
			},
		})
	}

	bundle := &antv1.UnsignedSweepBundle{
		BundleId:        batchID,
		BuiltAtMs:       time.Now().UnixMilli(),
		XpubFingerprint: b.xpubFingerprint,
		Txs:             txs,
	}

	b.log.Info("sweep undelegate-only bundle built",
		zap.String("batch_id", batchID),
		zap.Int("addresses", len(entries)),
		zap.Int("txs", len(txs)))

	return bundle, nil
}
