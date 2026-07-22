package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/chain"
	"alphaforge/internal/config"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/reconcile"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
	"alphaforge/internal/sweep"

	connectrpc "connectrpc.com/connect"
)

// depositSweepDeps holds the deposit + sweep related dependencies created by setupDepositAndSweep.
type depositSweepDeps struct {
	depositSvc      *service.DepositService
	depositAddrRepo *repository.DepositAddressRepository
	adminRepo       *repository.AdminRepository
	chainMonitor    *chain.Monitor
	reconcilerInst  *reconcile.Reconciler
	sweepWorker     *sweep.Worker
	sweepBundleRepo *sweep.BundleRepository
	sweepTronClient *sweep.TronClient
}

// setupDepositAndSweep wires the USDT deposit service, chain monitor, reconciler,
// and sweep worker (ADR-0026 Phase C). Called from registerHandlers.
func setupDepositAndSweep(
	mux *http.ServeMux,
	pool *pgxpool.Pool,
	cfg *config.Config,
	walletRepo *repository.WalletRepository,
	platformSvc *service.PlatformService,
	log *zap.Logger,
	otelInterceptor connectrpc.Interceptor,
	authInterceptor *interceptor.AuthInterceptor,
) depositSweepDeps {
	depositAddrRepo := repository.NewDepositAddressRepository(pool)
	depositRepo := repository.NewDepositRepository(pool)
	depositSvc := service.NewDepositService(depositAddrRepo, depositRepo, walletRepo, pool, cfg.DepositXpub, log)

	adminRepo := repository.NewAdminRepository(pool)
	tronGrid := chain.NewTronGridClient(cfg.TrongridAPIKey)
	tronScan := chain.NewTronScanClient(cfg.TronscanAPIKey)
	chainMonitor := chain.NewMonitor(tronGrid, tronScan, pool, depositAddrRepo, adminRepo, depositSvc, depositRepo, log)
	depositSvc.OnAddressClaimed = chainMonitor.RegisterAddress

	reconcileRepo := repository.NewReconcileRepository(pool)
	reconcilerInst := reconcile.NewReconciler(reconcileRepo, adminRepo, depositAddrRepo, tronGrid, log)

	sweepLogRepo := repository.NewSweepLogRepository(pool)
	sweepTronClient, err := sweep.NewTronClient(cfg.TronGridGRPCEndpoint, cfg.TrongridAPIKey)
	if err != nil {
		log.Warn("sweep tron client: failed to connect (sweep worker disabled)", zap.Error(err))
	}
	var sweepWorker *sweep.Worker
	var sweepBundleRepo *sweep.BundleRepository
	if sweepTronClient != nil {
		sweepBuilder := sweep.NewBuilder(sweepTronClient, depositAddrRepo, adminRepo, cfg.DepositXpubFingerprint, log)
		sweepBroadcaster := sweep.NewBroadcaster(sweepTronClient, sweepLogRepo, depositAddrRepo, adminRepo, log)
		sweepState := sweep.NewStateMachine(sweepTronClient, sweepLogRepo, tronGrid, adminRepo, depositAddrRepo, log)
		sweepBundleRepo = sweep.NewBundleRepository(pool)
		sweepWorker = sweep.NewWorker(sweepBuilder, sweepBroadcaster, sweepState, sweepBundleRepo,
			sweepLogRepo, depositRepo, depositAddrRepo, adminRepo, pool, log)
	}

	registerDepositHandler(mux, depositSvc, platformSvc, sweepWorker, adminRepo, cfg.DepositXpubFingerprint, log, otelInterceptor, authInterceptor)

	return depositSweepDeps{
		depositSvc:      depositSvc,
		depositAddrRepo: depositAddrRepo,
		adminRepo:       adminRepo,
		chainMonitor:    chainMonitor,
		reconcilerInst:  reconcilerInst,
		sweepWorker:     sweepWorker,
		sweepBundleRepo: sweepBundleRepo,
		sweepTronClient: sweepTronClient,
	}
}
