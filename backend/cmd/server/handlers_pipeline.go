package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"anttrader/internal/config"
	"anttrader/internal/costsvc"
	"anttrader/internal/mthub"
	"anttrader/internal/risksvc"
	"anttrader/internal/usermgr"
)

// initRiskPipeline creates and wires the SignalPipeline for pre-trade risk checks.
// Returns the pipeline, platform aggregator, and sets the pipeline/cost estimator/
// OMS writer/rate limiter on mthubSvc.
func initRiskPipeline(
	pool *pgxpool.Pool,
	log *zap.Logger,
	mthubSvc *mthub.MtHubService,
	eventStore *mthub.TradeEventStore,
	cfg *config.Config,
) (*risksvc.SignalPipeline, *risksvc.PlatformAggregator) {
	jurisStore := risksvc.NewPgJurisdictionStore(pool)
	geoipResolver := risksvc.NewMaxMindGeoIPResolver(cfg.GeoIPDBPath)
	jurisGate := &risksvc.JurisdictionGate{
		Store:               jurisStore,
		GeoIP:               geoipResolver,
		RequireKYC:           cfg.RequireKYC,
		RequireDisclaimer:    cfg.RequireDisclaimer,
		RequireQuestionnaire: cfg.RequireQuestionnaire,
	}

	capStore := risksvc.NewCapabilityStore()
	rows, err := pool.Query(context.Background(),
		`SELECT user_id, COALESCE(capability_tier, 0),
		        COALESCE(order_types_allowed, '{}'),
		        lot_per_order_max, daily_order_max, leverage_max,
		        COALESCE(symbol_whitelist, '{}'),
		        COALESCE(killswitch_enabled, false)
		 FROM user_risk_profiles`)
	if err != nil {
		log.Error("capability LoadFromPG query failed, using defaults", zap.Error(err))
	} else {
		if err := capStore.LoadFromPG(context.Background(), rows); err != nil {
			log.Error("capability LoadFromPG scan failed, using defaults", zap.Error(err))
		}
		log.Info("capability store loaded", zap.Int("users", capStore.Count()))
	}

	hardLimit := risksvc.NewHardLimitEvaluator(&risksvc.KycJurisdictionRule{Gate: jurisGate})
	platformAgg := risksvc.NewPlatformAggregator()
	platformAgg.StartRefreshLoop(5 * time.Second)
	platformLimits := risksvc.DefaultPlatformLimits()
	riskEngine := risksvc.NewEngine(
		&risksvc.MaxPosition{Max: 20},
		&risksvc.Margin{MinLevel: 1.5},
	)
	sizer := &risksvc.VolTargetSizer{RiskBudgetPct: 0.01}
	allocator := &risksvc.ProRataAllocator{}

	pipeline := risksvc.NewSignalPipeline(risksvc.PipelineConfig{
		CapStore:  capStore,
		HardLimit: hardLimit,
		Platform:  platformAgg,
		Limits:    platformLimits,
		Engine:    riskEngine,
		Sizer:     sizer,
		Allocator: allocator,
	})
	mthubSvc.SetRiskPipeline(pipeline)

	// Account state provider queries PG for balance/equity/margin per account.
	mthubSvc.SetAccountStateProvider(func(ctx context.Context, accountID string) (*mthub.AccountState, error) {
		var state mthub.AccountState
		var positions int64
		err := pool.QueryRow(ctx,
			`SELECT balance, equity, free_margin, COALESCE(margin, 0)::float8,
			        COALESCE((SELECT count(*) FROM positions WHERE mt_account_id = $1), 0)::int
			 FROM mt_accounts WHERE id = $1::uuid`,
			accountID,
		).Scan(&state.Balance, &state.Equity, &state.FreeMargin, &state.Margin, &positions)
		if err != nil {
			return nil, err
		}
		state.Positions = int(positions)
		return &state, nil
	})

	limiter := usermgr.NewUserLimiter(usermgr.DefaultConfig())
	mthubSvc.SetUserLimiter(limiter)

	eurtusdModel := &costsvc.CostModel{
		Symbol:           "EURUSD",
		SpreadPips:       1.0,
		PipSize:          0.0001,
		PipValue:         10.0,
		CommissionPerLot: 7.0,
	}
	estimator := &costsvc.StaticEstimator{Model: eurtusdModel}
	mthubSvc.SetCostEstimator(estimator)

	omsWriter := mthub.NewOmsWriter(pool, eventStore)
	mthubSvc.SetOmsWriter(omsWriter)

	return pipeline, platformAgg
}
