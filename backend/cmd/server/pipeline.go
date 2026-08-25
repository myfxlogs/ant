package main

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/marketplace"
	"alphaforge/internal/mdgateway"
	"alphaforge/internal/mdgateway/adapter"
	"alphaforge/internal/mdgateway/adapter/brokersearch"
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
	"alphaforge/internal/notifier"
	"alphaforge/internal/repository"
	"alphaforge/internal/risksvc"
	"alphaforge/internal/secrets"
	"alphaforge/internal/service"
)

type mdGatewayPipelineDeps struct {
	pipelineCtx       context.Context
	log               *zap.Logger
	pool              *pgxpool.Pool
	store             repository.MarketDataStore
	nc                *nats.Conn
	rdb               *goredis.Client
	spillDir          string
	secClient         secrets.Client
	hub               *mthub.Hub
	accountSvc        *service.AccountService
	mthubSvc          *mthub.MtHubService
	accountSyncSvc    *service.AccountSyncService
	tradeRecordRepo   *repository.TradeRecordRepository
	snapshotBroker    *mthub.PositionSnapshotBroker
	accountBroker     *mthub.AccountProfitBroker
	barBroker         *mthub.BarBroker
	eventStore        *mthub.TradeEventStore
	emailNotifier     **notifier.EmailNotifier
	platformAgg       **risksvc.PlatformAggregator
	reconLoop         **mthub.ReconciliationLoop
	brokerReg         *adapter.BrokerRegistry
	livePerfCollector *marketplace.LivePerformanceCollector
}

func startMdGatewayPipeline(d mdGatewayPipelineDeps) error {
	pst := newPipelineState(d.pool, d.log)

	pst.loadMarginThresholds()
	d.log.Info("mdgateway pipeline starting", zap.String("spill_dir", d.spillDir))
	d.mthubSvc.SetBarBroker(d.barBroker)

	deps := mdgateway.RunnerDeps{
		Log:                 d.log,
		PG:                  d.pool,
		Store:               d.store,
		NATSConn:            d.nc,
		RedisClient:         d.rdb,
		SpillDir:            d.spillDir,
		Secrets:             d.secClient,
		Hub:                 d.hub,
		BrokerRegistry:      d.brokerReg,
		Searcher:            brokersearch.New("", ""),
		OnAccountProfit:     pst.makeOnAccountProfit(d.accountSvc, d.mthubSvc, d.accountSyncSvc, d.eventStore, d.emailNotifier, d.livePerfCollector, d.snapshotBroker),
		OnOrderUpdate:       buildOnOrderUpdate(d.log, d.snapshotBroker, d.tradeRecordRepo, d.mthubSvc),
		OnAccountDisconnect: makeOnAccountDisconnect(d.log, d.pool, d.accountSvc, d.accountSyncSvc, d.platformAgg, d.hub, d.mthubSvc),
		OnBrokerInfo:        pst.makeOnBrokerInfo(d.accountSvc, d.accountSyncSvc, d.mthubSvc, d.snapshotBroker, d.reconLoop),
		OnBreakerTrip: func(accountID, userID, status, message string) {
			d.mthubSvc.PublishAccountStatus(&mthub.AccountStatusEvent{
				AccountID: accountID, UserID: userID, Status: status,
				Message: message, Timestamp: time.Now(),
			})
		},
		OnAccountStatus: makeOnAccountStatus(d.log, d.pool, d.mthubSvc),
		OnBar: func(b *mdtick.Bar) {
			d.mthubSvc.PublishBar(&mthub.BarUpdate{
				AccountID: b.AccountID,
				Symbol:    b.Canonical,
				Period:    b.Period,
				OpenTime:  b.OpenTsUnixMs,
				Open:      b.Open,
				High:      b.High,
				Low:       b.Low,
				Close:     b.Close,
				Bid:       b.Bid,
				Ask:       b.Ask,
				Volume:    b.Volume,
				Closed:    b.IsClosed,
			})
		},
		OnTick: func(t *mdtick.Tick) {
			d.mthubSvc.PublishTick(&mthub.TickUpdate{
				AccountID: t.AccountID,
				Symbol:    t.Canonical,
				Bid:       t.Bid,
				Ask:       t.Ask,
			})
		},
	}

	if err := mdgateway.Run(d.pipelineCtx, deps); err != nil {
		d.log.Error("mdgateway pipeline exited with error", zap.Error(err))
		return err
	}
	return nil
}

type pipelineState struct {
	pool                 *pgxpool.Pool
	log                  *zap.Logger
	marginCallMu         sync.Mutex
	marginCallLastSent   map[string]map[int]time.Time
	marginCallThresholds map[string]decimal.Decimal
	thresholdMu          sync.RWMutex
	lastSnapshot         map[string]time.Time
	snapshotMu           sync.Mutex
	lastMetricsWrite     map[string]time.Time
	metricsMu            sync.Mutex
}

func newPipelineState(pool *pgxpool.Pool, log *zap.Logger) *pipelineState {
	return &pipelineState{
		pool:                 pool,
		log:                  log,
		marginCallLastSent:   make(map[string]map[int]time.Time),
		marginCallThresholds: make(map[string]decimal.Decimal),
		lastSnapshot:         make(map[string]time.Time),
		lastMetricsWrite:     make(map[string]time.Time),
	}
}

func (p *pipelineState) loadMarginThresholds() {
	rows, err := p.pool.Query(context.Background(), `SELECT id, broker_margin_call_pct FROM mt_accounts WHERE deleted_at IS NULL`)
	if err != nil {
		p.log.Warn("B-2.3: failed to load margin thresholds, using defaults", zap.Error(err))
		return
	}
	defer rows.Close()
	for rows.Next() {
		var aid string
		var pct decimal.Decimal
		if err := rows.Scan(&aid, &pct); err != nil {
			p.log.Warn("B-2.3: scan margin threshold failed", zap.Error(err))
			continue
		}
		p.marginCallThresholds[aid] = pct
	}
}

func (p *pipelineState) makeOnAccountProfit(
	accountSvc *service.AccountService,
	mthubSvc *mthub.MtHubService,
	accountSyncSvc *service.AccountSyncService,
	eventStore *mthub.TradeEventStore,
	emailNotifier **notifier.EmailNotifier,
	livePerfCollector *marketplace.LivePerformanceCollector,
	snapshotBroker *mthub.PositionSnapshotBroker,
) func(accountID, userID string, pr *mdtick.ProfitUpdate) {
	return func(accountID, userID string, pr *mdtick.ProfitUpdate) {
		writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		userUID, err := uuid.Parse(userID)
		if err != nil {
			p.log.Warn("OnAccountProfit: invalid user UUID", zap.String("userID", userID), zap.Error(err))
		}
		writeNow := false
		func() {
			p.metricsMu.Lock()
			defer p.metricsMu.Unlock()
			if time.Since(p.lastMetricsWrite[accountID]) > 5*time.Second {
				p.lastMetricsWrite[accountID] = time.Now()
				writeNow = true
			}
		}()
		if writeNow && userUID != uuid.Nil {
			if err := accountSvc.UpdateAccountMetrics(writeCtx, userUID, accountID, pr.Balance, pr.Equity, pr.Credit, pr.Margin, pr.FreeMargin, pr.MarginLevel); err != nil {
				p.log.Warn("OnAccountProfit: pg update failed", zap.String("account", accountID), zap.Error(err))
			}
		}
		func() {
			p.snapshotMu.Lock()
			last, exists := p.lastSnapshot[accountID]
			if exists && time.Since(last) < time.Hour {
				p.snapshotMu.Unlock()
				return
			}
			p.lastSnapshot[accountID] = time.Now()
			p.snapshotMu.Unlock()
			if err := accountSvc.RecordBalanceSnapshot(writeCtx, accountID, userID, pr.Balance, pr.Equity, pr.Margin, pr.FreeMargin); err != nil {
				// Warn, not Debug: this insert failed 100% of the time for 28 days
				// (wrong target table) and the Debug level hid it while every
				// analytics reader of account_balance_history starved.
				p.log.Warn("OnAccountProfit: snapshot insert failed", zap.String("account", accountID), zap.Error(err))
			}
		}()
		accountSvc.UpdateSummaryCache(userID, accountID, pr.Balance, pr.Equity, "connected")
		mthubSvc.PublishAccountProfit(&mthub.AccountProfitEvent{
			AccountID: accountID, UserID: userID, Platform: pr.Platform,
			Balance: pr.Balance, Credit: pr.Credit, Equity: pr.Equity,
			Margin: pr.Margin, FreeMargin: pr.FreeMargin, MarginLevel: pr.MarginLevel,
			Profit: pr.Profit, ProfitPercent: pr.ProfitPercent,
			Status: "connected", Timestamp: time.Now(),
			Positions: convertProfitPositions(pr.Positions),
		})
		// OnOrderUpdate/OpenedOrders are unreliable for some sessions; OnOrderProfit
		// always carries the full opened-orders list. Publish a position snapshot from
		// the profit data so the frontend displays the correct open positions.
		publishProfitPositionSnapshot(snapshotBroker, accountID, userID, pr)
		if pr.MarginLevel.GreaterThan(decimal.Zero) {
			p.thresholdMu.RLock()
			callPct := p.marginCallThresholds[accountID]
			p.thresholdMu.RUnlock()
			if !callPct.GreaterThan(decimal.Zero) {
				callPct = decimal.NewFromInt(100)
			}
			accountSyncSvc.CheckMarginCall(accountID, userID, pr.MarginLevel, pr.Margin, pr.Equity, callPct, &p.marginCallMu, p.marginCallLastSent, eventStore, *emailNotifier)
		}
		if livePerfCollector != nil {
			livePerfCollector.OnProfitUpdate(accountID, pr.Equity, pr.Balance)
		}
	}
}
