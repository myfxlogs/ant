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

// BatchSweepEntry represents a single address in a batch sweep.
type BatchSweepEntry struct {
	Addr   *model.DepositAddress
	Amount string
}

// BuildBatchUnsignedBundle constructs a batch unsigned bundle for multiple
// addresses in a single cold-signing session (ADR §2.7: "批量归集可一次委托整批、统一收回").
//
// TRON Stake 2.0 DelegateResource is per-receiver: each deposit address needs
// its own delegate + undelegate transaction. The batch optimization does NOT
// reduce on-chain tx count (always 3N: delegate, transfer, undelegate per address).
// The benefit is operational: one USB round-trip to cold-sign all N×3 txs at once,
// instead of N separate cold-sign sessions. This is critical for throughput when
// sweeping many addresses — the air-gapped signing step is the bottleneck.
func (b *Builder) BuildBatchUnsignedBundle(ctx context.Context, entries []BatchSweepEntry) (*antv1.UnsignedSweepBundle, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("sweep builder: batch: empty entries")
	}

	cfg, err := b.loadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: batch: load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("sweep builder: batch: %w", err)
	}

	totalEnergy := int64(0)
	for _, e := range entries {
		energyAmount, err := calculateEnergy(e.Addr.HasReceivedUSDT, cfg.DEMFactor, cfg.EnergyBufferPercent)
		if err != nil {
			return nil, fmt.Errorf("sweep builder: batch: calculate energy: %w", err)
		}
		totalEnergy += energyAmount
	}

	delegateTRX, err := energyToTRX(totalEnergy, cfg.EnergyTRXRate)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: batch: energy to trx: %w", err)
	}

	expiryMs := time.Now().Add(23 * time.Hour).UnixMilli()
	batchID := uuid.New().String()

	var txs []*antv1.UnsignedTx

	for _, e := range entries {
		addrTxs, err := b.buildBatchAddrTxs(ctx, e, cfg, expiryMs)
		if err != nil {
			return nil, err
		}
		txs = append(txs, addrTxs...)
	}

	bundle := &antv1.UnsignedSweepBundle{
		BundleId:        batchID,
		BuiltAtMs:       time.Now().UnixMilli(),
		XpubFingerprint: "",
		Txs:             txs,
	}

	b.log.Info("sweep batch bundle built",
		zap.String("batch_id", batchID),
		zap.Int("addresses", len(entries)),
		zap.Int("txs", len(txs)),
		zap.Int64("total_energy", totalEnergy),
		zap.Int64("total_delegate_trx", delegateTRX))

	return bundle, nil
}

// buildBatchAddrTxs builds the 3 legs (delegate, transfer, undelegate) for a single
// address within a batch bundle.
func (b *Builder) buildBatchAddrTxs(ctx context.Context, e BatchSweepEntry, cfg SweepConfig, expiryMs int64) ([]*antv1.UnsignedTx, error) {
	addrEnergy, err := calculateEnergy(e.Addr.HasReceivedUSDT, cfg.DEMFactor, cfg.EnergyBufferPercent)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: batch: per-addr energy: %w", err)
	}
	addrDelegateTRX, err := energyToTRX(addrEnergy, cfg.EnergyTRXRate)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: batch: per-addr trx: %w", err)
	}

	delegateRaw, delegateTxID, err := b.tron.BuildDelegateResource(ctx, cfg.EnergyAccountAddress, e.Addr.Address, addrDelegateTRX)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: batch: delegate %s: %w", e.Addr.Address, err)
	}
	delegateRaw, err = SetTxExpiry(delegateRaw, expiryMs)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: batch: delegate expiry: %w", err)
	}

	amountDec, err := decimal.NewFromString(e.Amount)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: batch: parse amount %q: %w", e.Amount, err)
	}
	usdtDecimals := decimal.NewFromInt(1_000_000)
	amountSmallest := amountDec.Mul(usdtDecimals).BigInt()

	transferRaw, transferTxID, err := b.tron.BuildTRC20Transfer(ctx, e.Addr.Address, cfg.ColdWalletAddress, cfg.USDTContractAddress, amountSmallest, cfg.FeeLimit)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: batch: transfer %s: %w", e.Addr.Address, err)
	}
	transferRaw, err = SetTxExpiry(transferRaw, expiryMs)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: batch: transfer expiry: %w", err)
	}

	undelegateRaw, undelegateTxID, err := b.tron.BuildUndelegateResource(ctx, cfg.EnergyAccountAddress, e.Addr.Address, addrDelegateTRX)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: batch: undelegate %s: %w", e.Addr.Address, err)
	}
	undelegateRaw, err = SetTxExpiry(undelegateRaw, expiryMs)
	if err != nil {
		return nil, fmt.Errorf("sweep builder: batch: undelegate expiry: %w", err)
	}

	return []*antv1.UnsignedTx{
		{
			Kind:            antv1.TxKind_TX_KIND_DELEGATE,
			FromAddress:     cfg.EnergyAccountAddress,
			ToAddress:       e.Addr.Address,
			Amount:          strconv.FormatInt(addrDelegateTRX, 10),
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
			FromAddress:     e.Addr.Address,
			ToAddress:       cfg.ColdWalletAddress,
			Amount:          e.Amount,
			RawTx:           transferRaw,
			ExpiryMs:        expiryMs,
			ExpectedTxid:    transferTxID,
			DerivationIndex: uint32(e.Addr.DerivationIndex),
			Tx: &antv1.UnsignedTx_Transfer{
				Transfer: &antv1.TransferTx{
					TokenContract: cfg.USDTContractAddress,
				},
			},
		},
		{
			Kind:            antv1.TxKind_TX_KIND_UNDELEGATE,
			FromAddress:     cfg.EnergyAccountAddress,
			ToAddress:       e.Addr.Address,
			Amount:          strconv.FormatInt(addrDelegateTRX, 10),
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
	}, nil
}
