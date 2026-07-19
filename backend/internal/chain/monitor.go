package chain

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"

	"github.com/shopspring/decimal"
)

// Monitor scans TronGrid block events for USDT transfers to user deposit addresses.
// It loads ASSIGNED addresses into memory at startup and refreshes periodically.
type Monitor struct {
	grid       *TronGridClient
	scan       *TronScanClient
	addrRepo   *repository.DepositAddressRepository
	adminRepo  *repository.AdminRepository
	depositSvc *service.DepositService
	depositRepo *repository.DepositRepository
	log        *zap.Logger

	mu       sync.RWMutex
	addrMap  map[string]repository.AddressInfo // address → {userID, addrID}

	usdtContract   string
	minConfirms    int
	minDepositAmt  string
	scanInterval   time.Duration
	refreshInterval time.Duration
}

func NewMonitor(
	grid *TronGridClient,
	scan *TronScanClient,
	addrRepo *repository.DepositAddressRepository,
	adminRepo *repository.AdminRepository,
	depositSvc *service.DepositService,
	depositRepo *repository.DepositRepository,
	log *zap.Logger,
) *Monitor {
	return &Monitor{
		grid:           grid,
		scan:           scan,
		addrRepo:       addrRepo,
		adminRepo:      adminRepo,
		depositSvc:     depositSvc,
		depositRepo:    depositRepo,
		log:            log,
		addrMap:        make(map[string]repository.AddressInfo),
		scanInterval:   3 * time.Second,
		refreshInterval: 30 * time.Second,
	}
}

// Run starts the block scanning loop. It runs until ctx is cancelled.
func (m *Monitor) Run(ctx context.Context) error {
	if err := m.loadConfig(ctx); err != nil {
		return fmt.Errorf("chain monitor: load config: %w", err)
	}
	if err := m.loadAddresses(ctx); err != nil {
		return fmt.Errorf("chain monitor: load addresses: %w", err)
	}

	lastBlock, err := m.getCheckpoint(ctx)
	if err != nil {
		return fmt.Errorf("chain monitor: get checkpoint: %w", err)
	}

	m.log.Info("chain monitor started",
		zap.Int64("last_block", lastBlock),
		zap.Int("tracked_addresses", len(m.addrMap)))

	scanTicker := time.NewTicker(m.scanInterval)
	defer scanTicker.Stop()
	refreshTicker := time.NewTicker(m.refreshInterval)
	defer refreshTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.log.Info("chain monitor stopped")
			return ctx.Err()
		case <-refreshTicker.C:
			if err := m.loadAddresses(ctx); err != nil {
				m.log.Error("chain monitor: refresh addresses", zap.Error(err))
			}
		case <-scanTicker.C:
			if err := m.scanBlocks(ctx, &lastBlock); err != nil {
				m.log.Error("chain monitor: scan error", zap.Error(err))
			}
		}
	}
}

// loadConfig reads system_config for USDT contract address, min confirmations, and min deposit amount.
func (m *Monitor) loadConfig(ctx context.Context) error {
	cfg, err := m.adminRepo.GetConfig(ctx, "usdt_contract_address")
	if err != nil {
		return fmt.Errorf("get usdt_contract_address: %w", err)
	}
	m.usdtContract = cfg.Value

	cfg, err = m.adminRepo.GetConfig(ctx, "min_confirmations")
	if err != nil {
		return fmt.Errorf("get min_confirmations: %w", err)
	}
	n, err := strconv.Atoi(cfg.Value)
	if err != nil || n <= 0 {
		n = 20
	}
	m.minConfirms = n

	cfg, err = m.adminRepo.GetConfig(ctx, "min_deposit_amount")
	if err != nil {
		m.minDepositAmt = "1"
	} else {
		m.minDepositAmt = cfg.Value
	}
	if m.minDepositAmt == "" {
		m.minDepositAmt = "1"
	}

	return nil
}

// loadAddresses loads all ASSIGNED deposit addresses into the in-memory map.
func (m *Monitor) loadAddresses(ctx context.Context) error {
	addrs, err := m.addrRepo.ListAssignedAddresses(ctx)
	if err != nil {
		return fmt.Errorf("load addresses: %w", err)
	}
	m.mu.Lock()
	m.addrMap = addrs
	m.mu.Unlock()
	return nil
}

// RegisterAddress adds a single address to the in-memory map immediately.
// Called by DepositService.GetOrClaimAddress after a successful claim,
// eliminating the 30s refresh window where new deposits could be missed.
func (m *Monitor) RegisterAddress(addr string, userID uuid.UUID, addrID uuid.UUID) {
	m.mu.Lock()
	m.addrMap[addr] = repository.AddressInfo{UserID: userID, AddrID: addrID}
	m.mu.Unlock()
}

// getCheckpoint reads last_scanned_block from system_config.
func (m *Monitor) getCheckpoint(ctx context.Context) (int64, error) {
	cfg, err := m.adminRepo.GetConfig(ctx, "last_scanned_block")
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseInt(cfg.Value, 10, 64)
	if err != nil || n < 0 {
		return 0, nil
	}
	return n, nil
}

// saveCheckpoint persists last_scanned_block to system_config.
func (m *Monitor) saveCheckpoint(ctx context.Context, block int64) error {
	return m.adminRepo.SetConfigValue(ctx, "last_scanned_block", strconv.FormatInt(block, 10))
}

// scanBlocks scans one or more blocks after lastBlock, or waits if no new block.
// Only processes blocks that have at least minConfirms confirmations (latest - minConfirms).
// When behind (e.g. after restart), scans up to maxBlocksPerTick blocks per tick to catch up.
func (m *Monitor) scanBlocks(ctx context.Context, lastBlock *int64) error {
	latest, err := m.grid.GetLatestBlock(ctx)
	if err != nil {
		return fmt.Errorf("get latest block: %w", err)
	}

	// Only scan blocks with sufficient confirmations.
	safeLatest := latest - int64(m.minConfirms)
	if safeLatest <= *lastBlock {
		return nil
	}

	const maxBlocksPerTick = 10
	scanned := 0
	for *lastBlock < safeLatest && scanned < maxBlocksPerTick {
		nextBlock := *lastBlock + 1
		events, err := m.grid.GetBlockEvents(ctx, m.usdtContract, nextBlock)
		if err != nil {
			return fmt.Errorf("get block %d events: %w", nextBlock, err)
		}

		for _, evt := range events {
			m.processEvent(ctx, evt)
		}

		if err := m.saveCheckpoint(ctx, nextBlock); err != nil {
			return fmt.Errorf("save checkpoint for block %d: %w", nextBlock, err)
		}
		*lastBlock = nextBlock
		scanned++
	}

	if *lastBlock < safeLatest {
		m.log.Info("chain monitor: catching up",
			zap.Int64("current", *lastBlock),
			zap.Int64("safe_latest", safeLatest),
			zap.Int64("chain_latest", latest),
			zap.Int("scanned_this_tick", scanned))
	}

	return nil
}

// processEvent checks if a Transfer event matches a user deposit address and confirms it.
func (m *Monitor) processEvent(ctx context.Context, evt TransferEvent) {
	m.mu.RLock()
	info, ok := m.addrMap[evt.To]
	m.mu.RUnlock()
	if !ok {
		return
	}

	// Enforce minimum deposit amount using decimal (no float64).
	if evt.AmountString == "" {
		return
	}
	amt, err := decimal.NewFromString(evt.AmountString)
	if err != nil {
		m.log.Error("parse amount", zap.Error(err), zap.String("amount", evt.AmountString))
		return
	}
	minAmt, err := decimal.NewFromString(m.minDepositAmt)
	if err != nil {
		minAmt = decimal.NewFromInt(1)
	}
	if minAmt.GreaterThan(decimal.Zero) && amt.LessThan(minAmt) {
		m.log.Warn("deposit below minimum, ignoring",
			zap.String("amount", evt.AmountString),
			zap.String("min", m.minDepositAmt),
			zap.String("tx_hash", evt.TxHash))
		return
	}

	m.log.Info("deposit detected",
		zap.String("address", evt.To),
		zap.String("user_id", info.UserID.String()),
		zap.String("tx_hash", evt.TxHash),
		zap.String("amount", evt.AmountString),
		zap.Int64("block", evt.BlockNumber))

	verified, err := m.scan.VerifyTransaction(ctx, evt.TxHash, evt.To)
	if err != nil {
		// TronScan API failure (network/timeout) ≠ source confirmed not-found.
		// Retry once; if still failing, proceed with TronGrid-only confirmation.
		m.log.Warn("tronscan verification error, retrying",
			zap.Error(err), zap.String("tx_hash", evt.TxHash))
		verified, err = m.scan.VerifyTransaction(ctx, evt.TxHash, evt.To)
		if err != nil {
			m.log.Warn("tronscan retry failed, proceeding with TronGrid-only confirmation",
				zap.Error(err), zap.String("tx_hash", evt.TxHash))
			verified = true // degrade to single-source on API failure (not not-found)
		}
	}

	if !verified {
		m.log.Warn("multi-source verification: TronScan did not confirm, marking MANUAL_REVIEW",
			zap.String("tx_hash", evt.TxHash))
		d := &model.Deposit{
			ID:               uuid.New(),
			UserID:           info.UserID,
			DepositAddressID: info.AddrID,
			TxHash:           evt.TxHash,
			Amount:           evt.AmountString,
			BlockNumber:      evt.BlockNumber,
			Confirmations:    m.minConfirms,
			Status:           "MANUAL_REVIEW",
		}
		if err := m.depositRepo.Create(ctx, d); err != nil {
			m.log.Error("insert manual review deposit", zap.Error(err))
		}
		return
	}

	if err := m.depositSvc.ConfirmDeposit(ctx, info.UserID, info.AddrID,
		evt.TxHash, evt.AmountString, evt.BlockNumber, m.minConfirms); err != nil {
		m.log.Error("confirm deposit failed — falling back to MANUAL_REVIEW",
			zap.Error(err), zap.String("tx_hash", evt.TxHash))
		d := &model.Deposit{
			ID:               uuid.New(),
			UserID:           info.UserID,
			DepositAddressID: info.AddrID,
			TxHash:           evt.TxHash,
			Amount:           evt.AmountString,
			BlockNumber:      evt.BlockNumber,
			Confirmations:    m.minConfirms,
			Status:           "MANUAL_REVIEW",
		}
		if err := m.depositRepo.Create(ctx, d); err != nil {
			m.log.Error("insert manual review fallback deposit",
				zap.Error(err), zap.String("tx_hash", evt.TxHash))
		}
		return
	}

	if err := m.addrRepo.MarkReceivedUSDT(ctx, info.AddrID); err != nil {
		m.log.Error("mark received usdt", zap.Error(err))
	}
}
