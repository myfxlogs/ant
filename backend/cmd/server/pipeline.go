package main

import (
	"context"
	"os"
	"strings"
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
	scheduleResolver  mthub.ScheduleResolver
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
		Searcher:            brokersearch.NewFromConfig(os.Getenv("MTAPI_MT4_HOST"), os.Getenv("MTAPI_MT5_HOST")),
		OnAccountProfit:     pst.makeOnAccountProfit(d.accountSvc, d.mthubSvc, d.accountSyncSvc, d.eventStore, d.emailNotifier, d.livePerfCollector, d.snapshotBroker),
		OnOrderUpdate:       buildOnOrderUpdate(d.log, d.snapshotBroker, d.tradeRecordRepo, d.mthubSvc, d.scheduleResolver),
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

func (p *pipelineState) makeOnBrokerInfo(
	accountSvc *service.AccountService,
	accountSyncSvc *service.AccountSyncService,
	mthubSvc *mthub.MtHubService,
	snapshotBroker *mthub.PositionSnapshotBroker,
	reconLoop **mthub.ReconciliationLoop,
) func(accountID, platform, broker string, info *mdtick.BrokerInfo) {
	return func(accountID, platform, broker string, info *mdtick.BrokerInfo) {
		if userID, err := getUserIDFromPool(context.Background(), p.pool, accountID); err == nil {
			go accountSyncSvc.SyncAccountHistory(accountID, userID)
		}
		if *reconLoop != nil {
			(*reconLoop).ReconcileAccount(context.Background(), accountID)
		}
		// Publish authoritative broker financials on every connect/reconnect.
		// This corrects stale DB/frontend values when OnOrderProfit is silent
		// (e.g. no open positions, or mtapi stream stuck).
		if info.HasAccountSummary {
			userID, _ := getUserIDFromPool(context.Background(), p.pool, accountID)
			var profitPercent float64
			if info.Balance.GreaterThan(decimal.Zero) {
				pp, _ := info.Profit.Div(info.Balance).Mul(decimal.NewFromInt(100)).Float64()
				profitPercent = pp
			}
			mthubSvc.PublishAccountProfit(&mthub.AccountProfitEvent{
				AccountID:     accountID,
				UserID:        userID,
				Platform:      platform,
				Balance:       info.Balance,
				Credit:        info.Credit,
				Equity:        info.Equity,
				Margin:        info.Margin,
				FreeMargin:    info.FreeMargin,
				MarginLevel:   info.MarginLevel,
				Profit:        info.Profit,
				ProfitPercent: profitPercent,
				Status:        "connected",
				Timestamp:     time.Now(),
			})
			if uid, err := uuid.Parse(userID); err == nil {
				mctx, mcancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = accountSvc.UpdateAccountMetrics(mctx, uid, accountID,
					info.Balance, info.Equity, info.Credit,
					info.Margin, info.FreeMargin, info.MarginLevel)
				if err := accountSvc.RecordBalanceSnapshot(mctx, accountID, userID,
					info.Balance, info.Equity, info.Margin, info.FreeMargin); err != nil {
					p.log.Warn("OnBrokerInfo: snapshot insert failed", zap.String("account", accountID), zap.Error(err))
				}
				if info.AccountType != "" {
					if err := accountSvc.UpdateAccountType(mctx, accountID, info.AccountType); err != nil {
						p.log.Warn("OnBrokerInfo: update account_type failed",
							zap.String("account", accountID),
							zap.String("account_type", info.AccountType),
							zap.Error(err))
					}
				}
				mcancel()
			}
			p.log.Info("OnBrokerInfo: published authoritative account summary from broker",
				zap.String("account", accountID), zap.String("platform", platform),
				zap.String("balance", info.Balance.String()),
				zap.String("equity", info.Equity.String()),
				zap.String("margin", info.Margin.String()))
		}
		go func() {
			// mtapi.io session needs a moment to load account state after Connect.
			// Reconciliation (30s timeout) usually succeeds; the initial OpenedOrders
			// call can race the session warm-up. Retry errors before publishing the
			// broker-confirmed position list, including a legitimate empty list.
			var orders []*mthub.OrderRecord
			var err error
			for attempt := 0; attempt < 3; attempt++ {
				if attempt > 0 {
					time.Sleep(2 * time.Second)
				}
				sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				orders, err = mthubSvc.OpenedOrders(sctx, accountID)
				cancel()
				if err != nil {
					p.log.Warn("OnBrokerInfo: OpenedOrders failed, retrying",
						zap.String("account", accountID), zap.String("platform", platform),
						zap.Int("attempt", attempt+1), zap.Error(err))
					continue
				}
				break
			}
			if err != nil {
				p.log.Warn("OnBrokerInfo: OpenedOrders failed, initial position snapshot skipped",
					zap.String("account", accountID), zap.String("platform", platform), zap.Error(err))
				return
			}
			p.log.Info("OnBrokerInfo: publishing initial position snapshot",
				zap.String("account", accountID), zap.String("platform", platform), zap.Int("positions", len(orders)))
			userID, _ := getUserIDFromPool(context.Background(), p.pool, accountID)
			snapshot := &mthub.PositionSnapshot{
				AccountID: accountID, UserID: userID, Platform: platform,
				Balance: info.Balance, Credit: info.Credit, Equity: info.Equity,
				Margin: info.Margin, FreeMargin: info.FreeMargin, MarginLevel: info.MarginLevel,
				Profit: info.Profit, Leverage: info.Leverage, FinancialsAuthoritative: info.HasAccountSummary,
				FinancialsSource: mdtick.FinancialsSourceAccountSummary, CapturedAt: info.CapturedAt,
				PositionsAuthoritative: true,
				// B6: positions provenance for initial OpenedOrders.
				PositionsCapturedAt: info.CapturedAt,
				PositionsSource:     "opened_orders_initial",
				Positions:           make([]mthub.PositionSnapshotItem, 0, len(orders)),
				PendingOrders:       make([]mthub.PositionSnapshotItem, 0),
			}
			for _, o := range orders {
				// LIVE-MQL-ORDER-CONTEXT-1: use OrderTypeString (handles side +
				// order type, e.g. "BUY_LIMIT") and lowercase to match adapter
				// labels. Also set Magic/SL/TP which were previously missing.
				item := mthub.PositionSnapshotItem{
					Ticket: o.Ticket, Symbol: o.SymbolRaw,
					Type:   strings.ToLower(o.OrderTypeString()),
					Magic:  o.Magic,
					Volume: o.Volume, OpenPrice: o.OpenPrice, Profit: o.Profit,
					Swap: o.Swap, Commission: o.Commission, Comment: o.Comment,
					StopLoss:   o.StopLoss,
					TakeProfit: o.TakeProfit,
					OpenTime:   o.OpenTime.Unix(),
				}
				if mdtick.IsPendingOrderType(item.Type) {
					snapshot.PendingOrders = append(snapshot.PendingOrders, item)
				} else {
					snapshot.Positions = append(snapshot.Positions, item)
				}
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
			accountSyncSvc.SyncAccountHistory(accountID, userID)
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
			Message:   "health monitor: account dead after reconnect failure",
			Timestamp: time.Now(),
		})
	}
}

func makeOnAccountStatus(log *zap.Logger, pool *pgxpool.Pool, mthubSvc *mthub.MtHubService) func(accountID, userID, status, message string) {
	return func(accountID, userID, status, message string) {
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
