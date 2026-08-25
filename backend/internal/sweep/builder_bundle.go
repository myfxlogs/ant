// builder_bundle.go — BuildUnsignedBundle and BuildUndelegateOnlyBundle extracted from builder.go.
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
)

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
