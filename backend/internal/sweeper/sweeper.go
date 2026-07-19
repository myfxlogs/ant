// Package sweeper implements fund consolidation (归集) from user deposit addresses
// to the platform hot wallet using Energy delegation.
package sweeper

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/repository"
	"alphaforge/internal/secrets"
)

// Sweeper periodically sweeps USDT from deposit addresses to the hot wallet.
type Sweeper struct {
	sweepRepo   *repository.SweepLogRepository
	addrRepo    *repository.DepositAddressRepository
	adminRepo   *repository.AdminRepository
	depositRepo *repository.DepositRepository
	tron        *TronClient
	secrets     secrets.Client
	log         *zap.Logger

	hotWalletAddr  string
	hotWalletKey   []byte // encrypted hot wallet private key (from wallet_secrets)
	usdtContract   string
	demFactor      decimal.Decimal
	energyBuffer   decimal.Decimal
	sweepThreshold string
	batchSize      int
	scanInterval   time.Duration

	baseEnergyNew    int64 // energy for first-time USDT transfer (no prior USDT on address)
	baseEnergyRepeat int64 // energy for subsequent USDT transfers
}

func NewSweeper(
	sweepRepo *repository.SweepLogRepository,
	addrRepo *repository.DepositAddressRepository,
	adminRepo *repository.AdminRepository,
	depositRepo *repository.DepositRepository,
	tron *TronClient,
	secretsCli secrets.Client,
	log *zap.Logger,
) *Sweeper {
	return &Sweeper{
		sweepRepo:   sweepRepo,
		addrRepo:    addrRepo,
		adminRepo:   adminRepo,
		depositRepo: depositRepo,
		tron:        tron,
		secrets:     secretsCli,
		log:         log,
		scanInterval: 60 * time.Second,
	}
}

// Run starts the sweep loop. It runs until ctx is cancelled.
func (s *Sweeper) Run(ctx context.Context) error {
	if err := s.loadConfig(ctx); err != nil {
		return fmt.Errorf("sweeper: load config: %w", err)
	}

	s.log.Info("sweeper started",
		zap.String("hot_wallet", s.hotWalletAddr),
		zap.String("sweep_threshold", s.sweepThreshold))

	ticker := time.NewTicker(s.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("sweeper stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := s.scanAndSweep(ctx); err != nil {
				s.log.Error("sweeper: scan error", zap.Error(err))
			}
		}
	}
}

// loadConfig reads system_config for sweep-related settings.
func (s *Sweeper) loadConfig(ctx context.Context) error {
	keys := map[string]*string{
		"hot_wallet_address":    &s.hotWalletAddr,
		"usdt_contract_address": &s.usdtContract,
		"sweep_threshold":       &s.sweepThreshold,
	}
	for key, ptr := range keys {
		cfg, err := s.adminRepo.GetConfig(ctx, key)
		if err != nil {
			return fmt.Errorf("get %s: %w", key, err)
		}
		*ptr = cfg.Value
	}

	if cfg, err := s.adminRepo.GetConfig(ctx, "dem_factor"); err == nil {
		if v, err := decimal.NewFromString(cfg.Value); err == nil {
			s.demFactor = v
		}
	}
	if s.demFactor.IsZero() {
		s.demFactor = decimal.NewFromFloat(1.3)
	}

	if cfg, err := s.adminRepo.GetConfig(ctx, "energy_buffer_percent"); err == nil {
		if v, err := decimal.NewFromString(cfg.Value); err == nil {
			s.energyBuffer = v.Div(decimal.NewFromInt(100))
		}
	}
	if s.energyBuffer.IsZero() {
		s.energyBuffer = decimal.NewFromFloat(0.1)
	}

	if cfg, err := s.adminRepo.GetConfig(ctx, "sweep_batch_size"); err == nil {
		if v, err := strconv.Atoi(cfg.Value); err == nil {
			s.batchSize = v
		}
	}
	if s.batchSize == 0 {
		s.batchSize = 10
	}

	s.baseEnergyNew = 130000
	s.baseEnergyRepeat = 65000
	if cfg, err := s.adminRepo.GetConfig(ctx, "sweep_base_energy_new"); err == nil {
		if v, err := strconv.ParseInt(cfg.Value, 10, 64); err == nil && v > 0 {
			s.baseEnergyNew = v
		}
	}
	if cfg, err := s.adminRepo.GetConfig(ctx, "sweep_base_energy_repeat"); err == nil {
		if v, err := strconv.ParseInt(cfg.Value, 10, 64); err == nil && v > 0 {
			s.baseEnergyRepeat = v
		}
	}

	if s.sweepThreshold == "" {
		s.sweepThreshold = "1"
	}

	// Validate config values — fail fast before entering sweep loop.
	if s.hotWalletAddr == "" {
		return fmt.Errorf("sweeper: hot_wallet_address not configured")
	}
	if s.usdtContract == "" {
		return fmt.Errorf("sweeper: usdt_contract_address not configured")
	}
	if s.demFactor.IsNegative() {
		return fmt.Errorf("sweeper: dem_factor must be positive, got %s", s.demFactor.String())
	}
	if s.energyBuffer.IsNegative() {
		return fmt.Errorf("sweeper: energy_buffer_percent must be >= 0, got %s", s.energyBuffer.String())
	}
	if s.batchSize <= 0 {
		return fmt.Errorf("sweeper: sweep_batch_size must be > 0, got %d", s.batchSize)
	}
	thresholdDec, err := decimal.NewFromString(s.sweepThreshold)
	if err != nil {
		return fmt.Errorf("sweeper: sweep_threshold invalid: %w", err)
	}
	if thresholdDec.IsNegative() || thresholdDec.IsZero() {
		return fmt.Errorf("sweeper: sweep_threshold must be > 0, got %s", s.sweepThreshold)
	}

	// Load encrypted hot wallet private key from wallet_secrets table.
	hotKey, err := s.adminRepo.GetHotWalletKey(ctx)
	if err != nil {
		return fmt.Errorf("sweeper: load hot wallet key: %w", err)
	}
	s.hotWalletKey = hotKey

	return nil
}

// scanAndSweep finds deposit addresses with unswept confirmed deposits and creates sweep tasks.
// Uses ListUnsweptAddresses which correctly tracks (total_deposits - total_swept) per address,
// supporting multiple deposits to the same address across multiple sweep cycles.
func (s *Sweeper) scanAndSweep(ctx context.Context) error {
	// Recover sweep_logs stuck in SWEEPING or PENDING state from previous process crashes.
	if stuck, err := s.sweepRepo.MarkStuckSweepingAsFailed(ctx, 5*time.Minute); err != nil {
		s.log.Error("sweeper: mark stuck", zap.Error(err))
	} else if stuck > 0 {
		s.log.Warn("sweeper: recovered stuck sweep_logs", zap.Int64("count", stuck))
	}

	unswept, err := s.depositRepo.ListUnsweptAddresses(ctx, s.sweepThreshold, s.batchSize)
	if err != nil {
		return fmt.Errorf("sweeper: list unswept: %w", err)
	}

	for _, ps := range unswept {
		sweepLog, err := s.sweepRepo.CreatePending(ctx, ps.AddrID, ps.Amount)
		if err != nil {
			s.log.Error("sweeper: create pending", zap.Error(err), zap.String("addr_id", ps.AddrID.String()))
			continue
		}
		s.log.Info("sweeper: created sweep task",
			zap.String("sweep_id", sweepLog.ID.String()),
			zap.String("addr_id", ps.AddrID.String()),
			zap.String("amount", ps.Amount))

		sweepCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		sweepErr := s.executeSweep(sweepCtx, sweepLog, ps.AddrID)
		if sweepErr != nil {
			if err := s.sweepRepo.UpdateToFailed(ctx, sweepLog.ID, sweepErr.Error()); err != nil {
				s.log.Error("sweeper: update to failed", zap.Error(err))
			}
			s.log.Error("sweeper: execute failed", zap.Error(sweepErr), zap.String("sweep_id", sweepLog.ID.String()))
			cancel()
			continue
		}
		cancel()
	}

	return nil
}

// executeSweep performs the actual sweep: decrypt key → delegate energy → transfer → cleanup.
// Steps:
// 1. Decrypt deposit address private key via envelope encryption
// 2. Decrypt hot wallet private key for energy delegation
// 3. Delegate energy from hot wallet to deposit address
// 4. Trigger TRC20 transfer(deposit_addr → hot_wallet, amount)
// 5. Broadcast and wait for confirmation
// 6. Undelegate energy
// 7. Zero private key bytes from memory
// 8. Update sweep_log to DONE
func (s *Sweeper) executeSweep(ctx context.Context, sweepLog *model.SweepLog, addrID uuid.UUID) error {
	addr, err := s.addrRepo.GetByID(ctx, addrID)
	if err != nil {
		return fmt.Errorf("get address by id: %w", err)
	}
	if addr == nil {
		return fmt.Errorf("address not found: %s", addrID)
	}

	// Step 1: Decrypt deposit address private key.
	depositPrivKey, err := s.secrets.Decrypt(ctx, secrets.PurposeDepositPrivKey, addr.EncryptedPrivkey)
	if err != nil {
		return fmt.Errorf("decrypt deposit privkey: %w", err)
	}
	defer cryptoZero(depositPrivKey)

	// Step 2: Decrypt hot wallet private key for energy delegation.
	if len(s.hotWalletKey) == 0 {
		return fmt.Errorf("hot wallet encrypted key not loaded")
	}
	hotPrivKey, err := s.secrets.Decrypt(ctx, secrets.PurposeHotWalletKey, s.hotWalletKey)
	if err != nil {
		return fmt.Errorf("decrypt hot wallet privkey: %w", err)
	}
	defer cryptoZero(hotPrivKey)

	// Step 3: Calculate energy needed.
	baseEnergy := s.baseEnergyRepeat
	if !addr.HasReceivedUSDT {
		baseEnergy = s.baseEnergyNew
	}
	energyNeeded := decimal.NewFromInt(baseEnergy).Mul(s.demFactor).Mul(decimal.NewFromInt(1).Add(s.energyBuffer)).IntPart()
	if energyNeeded <= 0 {
		return fmt.Errorf("invalid energy calculation: %d (check dem_factor/energy_buffer config)", energyNeeded)
	}

	s.log.Info("sweeper: executing sweep",
		zap.String("sweep_id", sweepLog.ID.String()),
		zap.Int64("energy_needed", energyNeeded),
		zap.String("amount", sweepLog.Amount),
		zap.String("from_addr", addr.Address),
		zap.String("to_addr", s.hotWalletAddr))

	// Step 4: Delegate energy from hot wallet to deposit address.
	delegateTxHash, err := s.tron.DelegateEnergy(ctx, s.hotWalletAddr, addr.Address, hotPrivKey, energyNeeded)
	if err != nil {
		return fmt.Errorf("delegate energy: %w", err)
	}
	s.log.Info("sweeper: energy delegated",
		zap.String("delegate_tx", delegateTxHash),
		zap.Int64("energy", energyNeeded))

	// Ensure undelegate runs even if transfer fails.
	defer func() {
		undelegateCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := s.tron.UndelegateEnergy(undelegateCtx, s.hotWalletAddr, addr.Address, hotPrivKey, energyNeeded); err != nil {
			s.log.Error("sweeper: undelegate energy failed", zap.Error(err))
		}
	}()

	// Step 5: Convert amount from USDT string to smallest unit (sun, 10^-6).
	amountDecimal, err := decimal.NewFromString(sweepLog.Amount)
	if err != nil {
		return fmt.Errorf("parse sweep amount: %w", err)
	}
	amountSun := amountDecimal.Mul(decimal.NewFromInt(1000000)).RoundDown(0).IntPart()
	if amountSun <= 0 {
		return fmt.Errorf("sweep amount too small: %s USDT = %d sun", sweepLog.Amount, amountSun)
	}

	// Step 6: Update sweep_log to SWEEPING before broadcasting.
	if err := s.sweepRepo.UpdateToSweeping(ctx, sweepLog.ID, "", energyNeeded); err != nil {
		return fmt.Errorf("update to sweeping: %w", err)
	}

	// Step 7: Execute TRC20 transfer(deposit_addr → hot_wallet, amount).
	transferTxHash, err := s.tron.TransferTRC20(ctx, addr.Address, s.hotWalletAddr, s.usdtContract, depositPrivKey, amountSun)
	if err != nil {
		return fmt.Errorf("transfer trc20: %w", err)
	}

	s.log.Info("sweeper: transfer broadcast",
		zap.String("transfer_tx", transferTxHash))

	// Step 8a: Persist tx_hash immediately — even if confirmation fails,
	// we need a trace of the broadcast transaction to avoid losing funds.
	if err := s.sweepRepo.UpdateTxHash(ctx, sweepLog.ID, transferTxHash); err != nil {
		s.log.Error("sweeper: update tx_hash after broadcast", zap.Error(err))
	}

	// Step 8b: Wait for confirmation (up to 60s).
	// If this times out, the tx was still broadcast successfully — the funds have
	// moved on-chain. We mark DONE with a warning rather than FAILED, because:
	//   - FAILED would cause ListUnsweptAddresses to re-sweep the same address
	//   - The re-sweep would waste energy on an already-empty address
	//   - The on-chain tx will eventually confirm regardless of our polling
	if err := s.tron.WaitForConfirmation(ctx, transferTxHash, 60*time.Second); err != nil {
		s.log.Warn("sweeper: confirmation timeout — tx was broadcast, marking DONE",
			zap.String("sweep_id", sweepLog.ID.String()),
			zap.String("transfer_tx", transferTxHash),
			zap.Error(err))
	}

	// Step 9: Update sweep_log to DONE.
	if err := s.sweepRepo.UpdateToDone(ctx, sweepLog.ID); err != nil {
		return fmt.Errorf("update to done: %w", err)
	}

	s.log.Info("sweeper: sweep completed",
		zap.String("sweep_id", sweepLog.ID.String()),
		zap.String("transfer_tx", transferTxHash),
		zap.String("amount", sweepLog.Amount))

	return nil
}

// cryptoZero securely zeroes a byte slice in memory.
func cryptoZero(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}
