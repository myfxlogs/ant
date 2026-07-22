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
	// B-2.3: Per-broker 3-level margin call detection.
	// Level 1 (预警): margin_level <= call_pct * 1.5 → SSE only
	// Level 2 (警告): margin_level <= call_pct → SSE + Email
	// Level 3 (危急): margin_level <= call_pct * 0.7 → SSE + Email (1min cooldown)
	var marginCallMu sync.Mutex
	marginCallLastSent := make(map[string]map[int]time.Time)
	// Per-account broker thresholds loaded from mt_accounts.
	marginCallThresholds := make(map[string]decimal.Decimal) // accountID → broker_margin_call_pct
	var thresholdMu sync.RWMutex
	lastSnapshot := make(map[string]time.Time) // throttle: 1 snapshot/hour/account
	var snapshotMu sync.Mutex
	lastMetricsWrite := make(map[string]time.Time) // throttle: 5s between PG metric writes
	var metricsMu sync.Mutex

	// Load per-account broker margin call thresholds (default 100.0 from migration 122).
	func() {
		rows, err := pool.Query(context.Background(), `SELECT id, broker_margin_call_pct FROM mt_accounts WHERE deleted_at IS NULL`)
		if err != nil {
			log.Warn("B-2.3: failed to load margin thresholds, using defaults", zap.Error(err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var aid string
			var pct decimal.Decimal
			if err := rows.Scan(&aid, &pct); err != nil {
				log.Warn("B-2.3: scan margin threshold failed", zap.Error(err))
				continue
			}
			marginCallThresholds[aid] = pct
		}
	}()

	log.Info("mdgateway pipeline starting", zap.String("spill_dir", spillDir))
	// Wire bar broker so live_runner subscriptions receive bar events (D6-A / paper canary).
	mthubSvc.SetBarBroker(barBroker)

	if err := mdgateway.Run(pipelineCtx, mdgateway.RunnerDeps{
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
		OnAccountProfit: func(accountID, userID string, p *mdtick.ProfitUpdate) {
			// Write latest balance/equity to PG via AccountService.
			writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			userUID, err := uuid.Parse(userID)
			if err != nil {
				log.Warn("OnAccountProfit: invalid user UUID", zap.String("userID", userID), zap.Error(err))
			}
			// Throttle PG metric writes to at most once per 5s per account.
			writeNow := false
			func() {
				metricsMu.Lock()
				defer metricsMu.Unlock()
				if time.Since(lastMetricsWrite[accountID]) > 5*time.Second {
					lastMetricsWrite[accountID] = time.Now()
					writeNow = true
				}
			}()
			if writeNow && userUID != uuid.Nil {
				if err := accountSvc.UpdateAccountMetrics(writeCtx, userUID, accountID, p.Balance, p.Equity, p.Credit, p.Margin, p.FreeMargin, p.MarginLevel); err != nil {
					log.Warn("OnAccountProfit: pg update failed", zap.String("account", accountID), zap.Error(err))
				}
			}
			// Record hourly equity snapshot (throttled).
			func() {
				snapshotMu.Lock()
				last, exists := lastSnapshot[accountID]
				if exists && time.Since(last) < time.Hour {
					snapshotMu.Unlock()
					return
				}
				lastSnapshot[accountID] = time.Now()
				snapshotMu.Unlock()
				if err := accountSvc.RecordBalanceSnapshot(writeCtx, accountID, userID, p.Balance, p.Equity, p.Margin, p.FreeMargin); err != nil {
					log.Debug("OnAccountProfit: snapshot insert failed", zap.String("account", accountID), zap.Error(err))
				}
			}()
			// Publish to mthub for real-time SSE streaming.
			mthubSvc.PublishAccountProfit(&mthub.AccountProfitEvent{
				AccountID: accountID, UserID: userID, Platform: p.Platform,
				Balance: p.Balance, Credit: p.Credit, Equity: p.Equity,
				Margin: p.Margin, FreeMargin: p.FreeMargin, MarginLevel: p.MarginLevel,
				Profit: p.Profit, ProfitPercent: p.ProfitPercent,
				Status: "connected", Timestamp: time.Now(),
					Positions:     convertProfitPositions(p.Positions),
			})
			// Update in-memory summary cache so SSE SubscribeUserSummary avoids a full DB scan.
			accountSvc.UpdateSummaryCache(userID, accountID, p.Balance, p.Equity, "connected")
			// B-2.3: 3-level margin call detection with per-broker thresholds.
			if p.MarginLevel.GreaterThan(decimal.Zero) {
				thresholdMu.RLock()
				callPct := marginCallThresholds[accountID]
				thresholdMu.RUnlock()
				if !callPct.GreaterThan(decimal.Zero) {
					callPct = decimal.NewFromInt(100)
				}
				accountSyncSvc.CheckMarginCall(accountID, userID, p.MarginLevel, p.Margin, p.Equity, callPct, &marginCallMu, marginCallLastSent, eventStore, *emailNotifier)
			}
			// Live performance tracking for marketplace strategies.
			if livePerfCollector != nil {
				livePerfCollector.OnProfitUpdate(accountID, p.Equity, p.Balance)
			}
		},
		OnOrderUpdate: buildOnOrderUpdate(log, snapshotBroker, tradeRecordRepo),
		OnAccountDisconnect: func(accountID string) {
			var uid string
			if userID, err := getUserIDFromPool(context.Background(), pool, accountID); err == nil {
				uid = userID
				// D4: Run sync in background to avoid blocking the disconnect handler.
				go accountSyncSvc.SyncAccountHistory(accountID, userID)
			}
			(*platformAgg).ClearAccount(accountID)
			hub.RemoveSession(accountID) // BUG-2: clean Hub executors map on disconnect
			// Update DB status so frontend doesn't keep showing stale "connected" state.
			writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := accountSvc.DisconnectAccountByID(writeCtx, accountID); err != nil {
				log.Warn("OnAccountDisconnect: failed to update account_status", zap.String("account", accountID), zap.Error(err))
			} else if uid != "" {
				accountSvc.InvalidateSummaryCache(uid)
			}
			// Push real-time status update to SSE subscribers.
			mthubSvc.PublishAccountStatus(&mthub.AccountStatusEvent{
				AccountID: accountID, UserID: uid, Status: string(service.StatusDisconnected),
				Message: "health monitor: account dead after reconnect failure",
				Timestamp: time.Now(),
			})
		},
		OnBrokerInfo: func(accountID, platform, broker string, info *mdtick.BrokerInfo) {
			if userID, err := getUserIDFromPool(context.Background(), pool, accountID); err == nil {
				accountSyncSvc.SyncAccountHistory(accountID, userID)
			}
			// H17: Trigger reconciliation on broker reconnect so ant-side state
			// stays consistent with broker-side reality (ADR-0013).
			if *reconLoop != nil {
				(*reconLoop).ReconcileAccount(context.Background(), accountID)
			}
				// Publish initial position snapshot so frontend has data on first load.
				go func() {
					sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					orders, err := mthubSvc.OpenedOrders(sctx, accountID)
					if err != nil {
						log.Warn("OnBrokerInfo: OpenedOrders failed, initial position snapshot skipped",
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
			// Zero values mean "proto doesn't expose these yet" — keep the schema DEFAULTs.
			if pct > 0 || stop > 0 {
				pctDec := decimal.NewFromFloat(pct)
				stopDec := decimal.NewFromFloat(stop)
				if err := accountSvc.UpdateBrokerThresholds(ctx, accountID, pctDec, stopDec); err != nil {
					log.Warn("failed to persist broker margin info",
						zap.String("account", accountID), zap.Error(err))
				} else {
					// D1: Update in-memory thresholds so margin call detection uses fresh values.
					thresholdMu.Lock()
					marginCallThresholds[accountID] = pctDec
					thresholdMu.Unlock()
					log.Info("broker margin thresholds updated",
						zap.String("account", accountID),
						zap.String("margin_call_pct", pctDec.String()),
						zap.String("stop_out_pct", stopDec.String()))
				}
			}
		},
		OnAccountStatus: func(accountID, userID, status, message string) {
			// "reconnecting" is a transient partial state — don't persist or push SSE.
			// Stream recovery is handled by startGatewayForAccount (→ "connected").
			// Stream loss is handled by the health monitor (→ DisconnectAccountByID).
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
		},
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
	}); err != nil {
		log.Error("mdgateway pipeline exited with error", zap.Error(err))
		return err
	}
	return nil
}

func getUserIDFromPool(ctx context.Context, pool *pgxpool.Pool, accountID string) (string, error) {
	var userID string
	err := pool.QueryRow(ctx, "SELECT user_id::text FROM mt_accounts WHERE deleted_at IS NULL AND id = $1", accountID).Scan(&userID)
	return userID, err
}
