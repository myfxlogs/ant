// Package reconcile implements two-phase deposit reconciliation:
// Phase 1 (every 6h): internal ledger consistency (no API calls).
// Phase 2 (every 24h): on-chain balance verification via TronGrid.
package reconcile

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/chain"
	"alphaforge/internal/repository"
)

// Reconciler runs periodic reconciliation checks.
type Reconciler struct {
	reconcileRepo *repository.ReconcileRepository
	adminRepo     *repository.AdminRepository
	addrRepo      *repository.DepositAddressRepository
	grid          *chain.TronGridClient
	log           *zap.Logger

	alertThreshold float64
}

func NewReconciler(
	reconcileRepo *repository.ReconcileRepository,
	adminRepo *repository.AdminRepository,
	addrRepo *repository.DepositAddressRepository,
	grid *chain.TronGridClient,
	log *zap.Logger,
) *Reconciler {
	return &Reconciler{
		reconcileRepo: reconcileRepo,
		adminRepo:     adminRepo,
		addrRepo:      addrRepo,
		grid:          grid,
		log:           log,
	}
}

// Run starts the reconciliation loop. It runs until ctx is cancelled.
func (r *Reconciler) Run(ctx context.Context) error {
	if err := r.loadConfig(ctx); err != nil {
		return fmt.Errorf("reconciler: load config: %w", err)
	}

	r.log.Info("reconciler started", zap.Float64("alert_threshold", r.alertThreshold))

	internalTicker := time.NewTicker(6 * time.Hour)
	defer internalTicker.Stop()
	chainTicker := time.NewTicker(24 * time.Hour)
	defer chainTicker.Stop()

	// Run once at startup.
	r.runInternalReconcile(ctx)
	r.runChainReconcile(ctx)

	for {
		select {
		case <-ctx.Done():
			r.log.Info("reconciler stopped")
			return ctx.Err()
		case <-internalTicker.C:
			r.runInternalReconcile(ctx)
		case <-chainTicker.C:
			r.runChainReconcile(ctx)
		}
	}
}

func (r *Reconciler) loadConfig(ctx context.Context) error {
	if cfg, err := r.adminRepo.GetConfig(ctx, "reconcile_alert_threshold"); err == nil {
		if v, err := strconv.ParseFloat(cfg.Value, 64); err == nil {
			r.alertThreshold = v
		}
	}
	if r.alertThreshold == 0 {
		r.alertThreshold = 10.0
	}
	return nil
}

// runInternalReconcile checks that internal ledger is consistent.
// Phase 1: verifies that every confirmed deposit has been credited to a wallet.
// totalDeposits should equal totalDepositCredits in wallet_transactions.
func (r *Reconciler) runInternalReconcile(ctx context.Context) {
	totalDeposits, err := r.reconcileRepo.SumConfirmedDeposits(ctx)
	if err != nil {
		r.log.Error("reconcile: query total deposits", zap.Error(err))
		return
	}

	totalDepositCredits, err := r.reconcileRepo.SumDepositCredits(ctx)
	if err != nil {
		r.log.Error("reconcile: query deposit credits", zap.Error(err))
		return
	}

	deposits, err := decimal.NewFromString(totalDeposits)
	if err != nil {
		r.log.Error("reconcile: parse total deposits", zap.String("value", totalDeposits), zap.Error(err))
		return
	}
	credits, err := decimal.NewFromString(totalDepositCredits)
	if err != nil {
		r.log.Error("reconcile: parse total credits", zap.String("value", totalDepositCredits), zap.Error(err))
		return
	}
	diff := deposits.Sub(credits)

	r.log.Info("internal reconcile",
		zap.String("total_deposits", deposits.String()),
		zap.String("total_deposit_credits", credits.String()),
		zap.String("diff", diff.String()))

	if diff.Abs().GreaterThan(decimal.NewFromFloat(r.alertThreshold)) {
		r.log.Error("reconcile ALERT: internal ledger mismatch",
			zap.String("diff", diff.String()),
			zap.Float64("threshold", r.alertThreshold))
	}
}

// runChainReconcile verifies on-chain balances match internal records.
// Phase 2: queries TronGrid for actual USDT balances of all ASSIGNED deposit addresses,
// then compares with expected: SUM(deposits WHERE status='CONFIRMED') - SUM(sweep_logs WHERE status='DONE').
// The hot wallet is intentionally excluded — its balance includes swept funds plus any
// outflows/initial deposits we don't track, so including it would make diff = hot_wallet_balance.
func (r *Reconciler) runChainReconcile(ctx context.Context) {
	// Get all ASSIGNED addresses with their IDs for on-chain balance queries.
	addrMap, err := r.addrRepo.ListAssignedAddresses(ctx)
	if err != nil {
		r.log.Error("reconcile: list assigned addresses", zap.Error(err))
		return
	}

	// Query on-chain USDT balance for each assigned address.
	totalOnChain := decimal.Zero
	for addrStr := range addrMap {
		balance, err := r.grid.GetTRC20Balance(ctx, addrStr)
		if err != nil {
			r.log.Warn("reconcile: skip address balance query",
				zap.String("addr", addrStr),
				zap.Error(err))
			continue
		}
		balDecimal, perr := decimal.NewFromString(balance)
		if perr != nil {
			r.log.Warn("reconcile: parse on-chain balance", zap.String("addr", addrStr), zap.String("value", balance), zap.Error(perr))
			continue
		}
		totalOnChain = totalOnChain.Add(balDecimal)
	}

	// Get internal expected: confirmed deposits - swept amounts.
	// This represents USDT that should still be sitting in deposit addresses (not yet swept).
	totalDeposits, err := r.reconcileRepo.SumConfirmedDeposits(ctx)
	if err != nil {
		r.log.Error("reconcile: sum confirmed deposits", zap.Error(err))
		return
	}
	totalSwept, err := r.reconcileRepo.SumSweptAmounts(ctx)
	if err != nil {
		r.log.Error("reconcile: sum swept amounts", zap.Error(err))
		return
	}

	deposits, err := decimal.NewFromString(totalDeposits)
	if err != nil {
		r.log.Error("reconcile: parse total deposits for chain", zap.String("value", totalDeposits), zap.Error(err))
		return
	}
	swept, err := decimal.NewFromString(totalSwept)
	if err != nil {
		r.log.Error("reconcile: parse total swept", zap.String("value", totalSwept), zap.Error(err))
		return
	}
	expectedOnChain := deposits.Sub(swept)

	diff := totalOnChain.Sub(expectedOnChain)

	r.log.Info("chain reconcile",
		zap.String("on_chain_total", totalOnChain.String()),
		zap.String("expected_total", expectedOnChain.String()),
		zap.String("diff", diff.String()),
		zap.Int("queried_addresses", len(addrMap)))

	if diff.Abs().GreaterThan(decimal.NewFromFloat(r.alertThreshold)) {
		r.log.Error("reconcile ALERT: on-chain balance mismatch",
			zap.String("diff", diff.String()),
			zap.Float64("threshold", r.alertThreshold),
			zap.String("on_chain", totalOnChain.String()),
			zap.String("expected", expectedOnChain.String()))
	}
}
