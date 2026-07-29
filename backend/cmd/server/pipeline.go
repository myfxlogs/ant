package main

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/mdgateway"
	"alphaforge/internal/mdgateway/adapter"
	"alphaforge/internal/mdgateway/adapter/brokersearch"
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/marketplace"
	"alphaforge/internal/mthub"
	"alphaforge/internal/notifier"
	"alphaforge/internal/repository"
	"alphaforge/internal/risksvc"
	"alphaforge/internal/secrets"
	"alphaforge/internal/service"
)

func startMdGatewayPipeline(
	pipelineCtx context.Context,
	log *zap.Logger,
	pool *pgxpool.Pool,
	store repository.MarketDataStore,
	chStore repository.MarketDataStore,
	nc *nats.Conn,
	spillDir string,
	secClient secrets.Client,
	hub *mthub.Hub,
	accountSvc *service.AccountService,
	mthubSvc *mthub.MtHubService,
	accountSyncSvc *service.AccountSyncService,
	tradeRecordRepo *repository.TradeRecordRepository,
	snapshotBroker *mthub.PositionSnapshotBroker,
	accountBroker *mthub.AccountProfitBroker,
	barBroker *mthub.BarBroker,
	eventStore *mthub.TradeEventStore,
	emailNotifier **notifier.EmailNotifier,
	platformAgg **risksvc.PlatformAggregator,
	reconLoop **mthub.ReconciliationLoop,
	brokerReg *adapter.BrokerRegistry,
	factorPusher func(bar *mdtick.Bar),
	livePerfCollector *marketplace.LivePerformanceCollector,
) error {
	pst := newPipelineState(pool, log)

	pst.loadMarginThresholds()
	log.Info("mdgateway pipeline starting", zap.String("spill_dir", spillDir))
	mthubSvc.SetBarBroker(barBroker)

	deps := mdgateway.RunnerDeps{
		Log:      log,
		PG:       pool,
		Store:    store,
		ChStore:  chStore,
		NATSConn: nc,
		SpillDir: spillDir,
		Secrets:  secClient,
		Hub:            hub,
		BrokerRegistry: brokerReg,
		FactorPusher:   factorPusher,
		Searcher:       brokersearch.New("", ""),
		OnAccountProfit: pst.makeOnAccountProfit(accountSvc, mthubSvc, accountSyncSvc, eventStore, emailNotifier, livePerfCollector),
		OnOrderUpdate:   buildOnOrderUpdate(log, snapshotBroker, tradeRecordRepo),
		OnAccountDisconnect: makeOnAccountDisconnect(log, pool, accountSvc, accountSyncSvc, platformAgg, hub, mthubSvc),
		OnBrokerInfo:         pst.makeOnBrokerInfo(accountSvc, accountSyncSvc, mthubSvc, snapshotBroker, reconLoop),
		OnBreakerTrip: func(accountID, userID, status, message string) {
			mthubSvc.PublishAccountStatus(&mthub.AccountStatusEvent{
				AccountID: accountID, UserID: userID, Status: status,
				Message: message, Timestamp: time.Now(),
			})
		},
		OnAccountStatus: makeOnAccountStatus(log, pool, mthubSvc),
		OnBar: func(b *mdtick.Bar) {
			mthubSvc.PublishBar(&mthub.BarUpdate{
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
	}

	if err := mdgateway.Run(pipelineCtx, deps); err != nil {
		log.Error("mdgateway pipeline exited with error", zap.Error(err))
		return err
	}
	return nil
}

type pipelineState struct {
	pool                *pgxpool.Pool
	log                *zap.Logger
	marginCallMu       sync.Mutex
	marginCallLastSent map[string]map[int]time.Time
	marginCallThresholds map[string]decimal.Decimal
	thresholdMu        sync.RWMutex
	lastSnapshot       map[string]time.Time
	snapshotMu         sync.Mutex
	lastMetricsWrite   map[string]time.Time
	metricsMu          sync.Mutex
}

func newPipelineState(pool *pgxpool.Pool, log *zap.Logger) *pipelineState {
	return &pipelineState{
		pool:                pool,
		log:                log,
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
				p.log.Debug("OnAccountProfit: snapshot insert failed", zap.String("account", accountID), zap.Error(err))
			}
		}()
		mthubSvc.PublishAccountProfit(&mthub.AccountProfitEvent{
			AccountID: accountID, UserID: userID, Platform: pr.Platform,
			Balance: pr.Balance, Credit: pr.Credit, Equity: pr.Equity,
			Margin: pr.Margin, FreeMargin: pr.FreeMargin, MarginLevel: pr.MarginLevel,
			Profit: pr.Profit, ProfitPercent: pr.ProfitPercent,
			Status: "connected", Timestamp: time.Now(),
			Positions: convertProfitPositions(pr.Positions),
		})
		accountSvc.UpdateSummaryCache(userID, accountID, pr.Balance, pr.Equity, "connected")
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

func (p *pipelineState) makeOnBrokerInfo(
	accountSvc *service.AccountService,
	accountSyncSvc *service.AccountSyncService,
	mthubSvc *mthub.MtHubService,
	snapshotBroker *mthub.PositionSnapshotBroker,
	reconLoop **mthub.ReconciliationLoop,
) func(accountID, platform, broker string, info *mdtick.BrokerInfo) {
	return func(accountID, platform, broker string, info *mdtick.BrokerInfo) {
		if userID, err := getUserIDFromPool(context.Background(), p.pool, accountID); err == nil {
			accountSyncSvc.SyncAccountHistory(accountID, userID)
		}
		if *reconLoop != nil {
			(*reconLoop).ReconcileAccount(context.Background(), accountID)
		}
		go func() {
			sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			orders, err := mthubSvc.OpenedOrders(sctx, accountID)
			if err != nil {
				p.log.Warn("OnBrokerInfo: OpenedOrders failed, initial position snapshot skipped",
					zap.String("account", accountID), zap.String("platform", platform), zap.Error(err))
				return
			}
			snapshot := &mthub.PositionSnapshot{AccountID: accountID, Positions: make([]mthub.PositionSnapshotItem, 0, len(orders))}
			for _, o := range orders {
				snapshot.Positions = append(snapshot.Positions, mthub.PositionSnapshotItem{
					Ticket: o.Ticket, Symbol: o.SymbolRaw, Type: service.MapSideToString(o.Side), Volume: o.Volume,
					OpenPrice: o.OpenPrice, Profit: o.Profit,
					Swap: o.Swap, Commission: o.Commission, Comment: o.Comment,
					OpenTime: o.OpenTime.Unix(),
				})
			}
			snapshotBroker.Publish(snapshot)
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		pct := info.MarginCallPct
		stop := info.StopOutPct
		if pct > 0 || stop > 0 {
			pctDec := decimal.NewFromFloat(pct)
			stopDec := decimal.NewFromFloat(stop)
			if err := accountSvc.UpdateBrokerThresholds(ctx, accountID, pctDec, stopDec); err != nil {
				p.log.Warn("failed to persist broker margin info",
					zap.String("account", accountID), zap.Error(err))
			} else {
				p.thresholdMu.Lock()
				p.marginCallThresholds[accountID] = pctDec
				p.thresholdMu.Unlock()
				p.log.Info("broker margin thresholds updated",
					zap.String("account", accountID),
					zap.String("margin_call_pct", pctDec.String()),
					zap.String("stop_out_pct", stopDec.String()))
			}
		}
	}
}

func makeOnAccountDisconnect(
	log *zap.Logger,
	pool *pgxpool.Pool,
	accountSvc *service.AccountService,
	accountSyncSvc *service.AccountSyncService,
	platformAgg **risksvc.PlatformAggregator,
	hub *mthub.Hub,
	mthubSvc *mthub.MtHubService,
) func(accountID string) {
	return func(accountID string) {
		var uid string
		if userID, err := getUserIDFromPool(context.Background(), pool, accountID); err == nil {
			uid = userID
			go accountSyncSvc.SyncAccountHistory(accountID, userID)
		}
		(*platformAgg).ClearAccount(accountID)
		hub.RemoveSession(accountID)
		writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := accountSvc.DisconnectAccountByID(writeCtx, accountID); err != nil {
			log.Warn("OnAccountDisconnect: failed to update account_status", zap.String("account", accountID), zap.Error(err))
		} else if uid != "" {
			accountSvc.InvalidateSummaryCache(uid)
		}
		mthubSvc.PublishAccountStatus(&mthub.AccountStatusEvent{
			AccountID: accountID, UserID: uid, Status: string(service.StatusDisconnected),
			Message:  "health monitor: account dead after reconnect failure",
			Timestamp: time.Now(),
		})
	}
}

func makeOnAccountStatus(log *zap.Logger, pool *pgxpool.Pool, mthubSvc *mthub.MtHubService) func(accountID, userID, status, message string) {
	return func(accountID, userID, status, message string) {
		if status == "reconnecting" {
			return
		}
		writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if status == "connected" {
			if _, err := pool.Exec(writeCtx,
				`UPDATE mt_accounts SET account_status = $1, last_error = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND deleted_at IS NULL`,
				status, accountID); err != nil {
				log.Warn("OnAccountStatus: update failed", zap.String("account", accountID), zap.Error(err))
			}
		} else {
			msg := message
			if len(msg) > 512 {
				msg = msg[:512]
			}
			if _, err := pool.Exec(writeCtx,
				`UPDATE mt_accounts SET account_status = $1, last_error = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`,
				status, msg, accountID); err != nil {
				log.Warn("OnAccountStatus: update failed", zap.String("account", accountID), zap.Error(err))
			}
		}
		mthubSvc.PublishAccountStatus(&mthub.AccountStatusEvent{
			AccountID: accountID, UserID: userID, Status: status,
			Message: message, Timestamp: time.Now(),
		})
	}
}

func getUserIDFromPool(ctx context.Context, pool *pgxpool.Pool, accountID string) (string, error) {
	var userID string
	err := pool.QueryRow(ctx, "SELECT user_id::text FROM mt_accounts WHERE deleted_at IS NULL AND id = $1", accountID).Scan(&userID)
	return userID, err
}
