package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"anttrader/internal/config"
	"anttrader/internal/costsvc"
	"anttrader/internal/mthub"
	"anttrader/internal/risk"
	"anttrader/internal/risksvc"
	"anttrader/internal/usermgr"
)

// initRiskPipeline creates and wires the SignalPipeline for pre-trade risk checks.
// Returns the pipeline, platform aggregator, and sets the pipeline/cost estimator/
// OMS writer/rate limiter on mthubSvc.
func initRiskPipeline(
	pool *pgxpool.Pool, log *zap.Logger, mthubSvc *mthub.MtHubService,
	hub *mthub.Hub, eventStore *mthub.TradeEventStore, cfg *config.Config,
	guard *risk.Guard,
) (*risksvc.JurisdictionGate, *risksvc.CapabilityStore, *risksvc.PlatformAggregator) {
	jurisGate := buildJurisdictionGate(pool, cfg)
	capStore := loadCapabilityStore(pool, log)
	platformAgg := risksvc.NewPlatformAggregator()
	platformAgg.StartRefreshLoop(5 * time.Second)

	// D6-A: risksvc pipeline replaced by risk.Gate (single chokepoint).
	wireMthubServices(pool, log, mthubSvc, hub, eventStore, guard)
	return jurisGate, capStore, platformAgg
}

func buildJurisdictionGate(pool *pgxpool.Pool, cfg *config.Config) *risksvc.JurisdictionGate {
	return &risksvc.JurisdictionGate{
		Store: risksvc.NewPgJurisdictionStore(pool),
		GeoIP: risksvc.NewMaxMindGeoIPResolver(cfg.GeoIPDBPath),
		RequireKYC: cfg.RequireKYC, RequireDisclaimer: cfg.RequireDisclaimer,
		RequireQuestionnaire: cfg.RequireQuestionnaire,
	}
}

func loadCapabilityStore(pool *pgxpool.Pool, log *zap.Logger) *risksvc.CapabilityStore {
	capStore := risksvc.NewCapabilityStore()
	rows, err := pool.Query(context.Background(),
		`SELECT user_id, COALESCE(capability_tier,0), COALESCE(order_types_allowed,'{}'),
		        lot_per_order_max, daily_order_max, leverage_max,
		        COALESCE(symbol_whitelist,'{}'), COALESCE(killswitch_enabled,false)
		 FROM user_risk_profiles`)
	if err != nil {
		log.Error("capability LoadFromPG query failed, using defaults", zap.Error(err))
		return capStore
	}
	if err := capStore.LoadFromPG(context.Background(), rows); err != nil {
		log.Error("capability LoadFromPG scan failed, using defaults", zap.Error(err))
	}
	log.Info("capability store loaded", zap.Int("users", capStore.Count()))
	return capStore
}

func wireMthubServices(pool *pgxpool.Pool, log *zap.Logger, mthubSvc *mthub.MtHubService, hub *mthub.Hub, eventStore *mthub.TradeEventStore, guard *risk.Guard) {
	mthubSvc.SetAccountStateProvider(func(ctx context.Context, accountID string) (*risk.AccountState, error) {
		var balance, equity, freeMargin, margin, positions float64
		err := pool.QueryRow(ctx,
			`SELECT balance, equity, free_margin, COALESCE(margin,0)::float8,
			        COALESCE((SELECT count(*) FROM positions WHERE mt_account_id=$1),0)::int
			 FROM accounts WHERE id=$2`, accountID, accountID).
			Scan(&balance, &equity, &freeMargin, &margin, &positions)
		if err != nil {
			return nil, fmt.Errorf("account state query: %w", err)
		}
		return &risk.AccountState{
			Balance:       decimal.NewFromFloat(balance),
			Equity:        decimal.NewFromFloat(equity),
			FreeMargin:    decimal.NewFromFloat(freeMargin),
			UsedMargin:    decimal.NewFromFloat(margin),
			OpenPositions: int(positions),
		}, nil
	})
	mthubSvc.SetUserLimiter(usermgr.NewUserLimiter(usermgr.DefaultConfig()))
	mthubSvc.SetCostEstimator(mthub.NewHubCostEstimator(hub, &costsvc.CostModel{
		Symbol: "DEFAULT", SpreadPips: 1.0, PipSize: 0.00001, PipValue: 1.0, CommissionPerLot: 0,
	}, log))
	mthubSvc.SetOmsWriter(mthub.NewOmsWriter(pool, eventStore))
	mthubSvc.SetGuard(guard)
}
