package service

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
	"alphaforge/internal/repository"
)

// WithdrawalBuilder constructs UnsignedSweepBundle for pending withdrawals.
// It reads SIGNED_WAITING_BUNDLE withdrawals that have assertions stored,
// builds UnsignedTx with TransferTx.auth=WithdrawalAuth, and persists them
// as PENDING_SIGN bundles for coldsign to verify and sign (R11).
type WithdrawalBuilder struct {
	withdrawRepo *repository.WithdrawalRepository
	bundleSaver  BundleSaver
	adminRepo    *repository.AdminRepository
	tron         TronTxBuilder
	log          *zap.Logger

	cfgMu     sync.Mutex
	cfgCache  *withdrawalConfig
	cfgExpiry time.Time
}

// BundleSaver persists unsigned bundles (implemented by sweep.BundleRepository).
type BundleSaver interface {
	SaveUnsignedBundle(ctx context.Context, batchID, addrID uuid.UUID, unsigned *antv1.UnsignedSweepBundle, builtAtMs int64) error
}

// TronTxBuilder is the interface for building unsigned TRON transactions.
type TronTxBuilder interface {
	BuildTRC20Transfer(ctx context.Context, from, to, contract string, amount *big.Int, feeLimit int64) ([]byte, string, error)
}

// NewWithdrawalBuilder creates a WithdrawalBuilder.
func NewWithdrawalBuilder(
	withdrawRepo *repository.WithdrawalRepository,
	bundleSaver BundleSaver,
	adminRepo *repository.AdminRepository,
	tron TronTxBuilder,
	log *zap.Logger,
) *WithdrawalBuilder {
	return &WithdrawalBuilder{
		withdrawRepo: withdrawRepo,
		bundleSaver:  bundleSaver,
		adminRepo:    adminRepo,
		tron:         tron,
		log:          log,
	}
}

// BuildPendingWithdrawals constructs UnsignedSweepBundle for all withdrawals
// that have assertions stored but no bundle yet (status=SIGNED_WAITING_BUNDLE).
func (b *WithdrawalBuilder) BuildPendingWithdrawals(ctx context.Context) error {
	withdrawals, err := b.withdrawRepo.ListWithdrawalsByStatus(ctx, "SIGNED_WAITING_BUNDLE")
	if err != nil {
		return fmt.Errorf("withdrawal builder: list pending: %w", err)
	}

	cfg, err := b.loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("withdrawal builder: load config: %w", err)
	}

	for _, wr := range withdrawals {
		if err := b.buildOne(ctx, &wr, cfg); err != nil {
			b.log.Warn("withdrawal builder: build failed",
				zap.String("withdrawal_id", wr.ID.String()),
				zap.Error(err))
		}
	}
	return nil
}

func (b *WithdrawalBuilder) loadConfig(ctx context.Context) (*withdrawalConfig, error) {
	b.cfgMu.Lock()
	defer b.cfgMu.Unlock()

	if b.cfgCache != nil && time.Now().Before(b.cfgExpiry) {
		return b.cfgCache, nil
	}

	get := func(key string) string {
		cfg, err := b.adminRepo.GetConfig(ctx, key)
		if err != nil || cfg == nil {
			return ""
		}
		return cfg.Value
	}

	coldWallet := get("cold_wallet_address")
	if coldWallet == "" {
		return nil, fmt.Errorf("cold_wallet_address not configured")
	}
	usdtContract := get("usdt_contract_address")
	if usdtContract == "" {
		return nil, fmt.Errorf("usdt_contract_address not configured")
	}
	xpubFingerprint := get("xpub_fingerprint")

	feeLimitStr := get("fee_limit")
	if feeLimitStr == "" {
		feeLimitStr = "100000000"
	}
	feeLimit, err := decimal.NewFromString(feeLimitStr)
	if err != nil {
		return nil, fmt.Errorf("parse fee_limit: %w", err)
	}

	cfg := &withdrawalConfig{
		coldWalletAddress:   coldWallet,
		usdtContractAddress: usdtContract,
		feeLimit:            feeLimit.IntPart(),
		xpubFingerprint:     xpubFingerprint,
	}
	b.cfgCache = cfg
	b.cfgExpiry = time.Now().Add(5 * time.Minute)
	return cfg, nil
}

type withdrawalConfig struct {
	coldWalletAddress   string
	usdtContractAddress string
	feeLimit            int64
	xpubFingerprint     string
}

func (b *WithdrawalBuilder) buildOne(ctx context.Context, wr *model.WithdrawalRequest, cfg *withdrawalConfig) error {
	amountDec, err := decimal.NewFromString(wr.Amount)
	if err != nil {
		return fmt.Errorf("parse amount: %w", err)
	}
	usdtDecimals := decimal.NewFromInt(1_000_000)
	amountSmallest := amountDec.Mul(usdtDecimals).BigInt()

	expiryMs := time.Now().Add(23 * time.Hour).UnixMilli()
	batchID := uuid.New()

	rawTx, expectedTxid, err := b.tron.BuildTRC20Transfer(ctx,
		cfg.coldWalletAddress, wr.DestAddress, cfg.usdtContractAddress,
		amountSmallest, cfg.feeLimit)
	if err != nil {
		return fmt.Errorf("build trc20 transfer: %w", err)
	}

	auth := &antv1.WithdrawalAuth{
		UserId:       wr.UserID.String(),
		Nonce:        uint64(wr.Nonce),
		CredentialId: wr.CredentialID,
		Assertion:    wr.Assertion,
	}

	unsignedTx := &antv1.UnsignedTx{
		Kind:         antv1.TxKind_TX_KIND_TRANSFER,
		FromAddress:  cfg.coldWalletAddress,
		ToAddress:    wr.DestAddress,
		Amount:       wr.Amount,
		RawTx:        rawTx,
		ExpiryMs:     expiryMs,
		ExpectedTxid: expectedTxid,
		Tx: &antv1.UnsignedTx_Transfer{
			Transfer: &antv1.TransferTx{
				TokenContract: cfg.usdtContractAddress,
				Auth:          auth,
			},
		},
	}

	bundle := &antv1.UnsignedSweepBundle{
		BundleId:        batchID.String(),
		BuiltAtMs:       time.Now().UnixMilli(),
		XpubFingerprint: cfg.xpubFingerprint,
		Txs:             []*antv1.UnsignedTx{unsignedTx},
	}

	if err := b.bundleSaver.SaveUnsignedBundle(ctx, batchID, uuid.Nil, bundle, time.Now().UnixMilli()); err != nil {
		return fmt.Errorf("save bundle: %w", err)
	}

	if err := b.withdrawRepo.UpdateWithdrawalBundle(ctx, wr.ID, batchID); err != nil {
		return fmt.Errorf("link withdrawal to bundle: %w", err)
	}

	b.log.Info("withdrawal bundle built for coldsign",
		zap.String("withdrawal_id", wr.ID.String()),
		zap.String("batch_id", batchID.String()),
		zap.String("dest", wr.DestAddress),
		zap.String("amount", wr.Amount))

	return nil
}
