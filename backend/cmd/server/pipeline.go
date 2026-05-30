package main

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/model"
	"anttrader/internal/mthub"
	"anttrader/internal/notifier"
	"anttrader/internal/repository"
	"anttrader/internal/risksvc"
	"anttrader/internal/secrets"
	"anttrader/internal/service"
)

func startMdGatewayPipeline(
	pipelineCtx context.Context,
	log *zap.Logger,
	pool *pgxpool.Pool,
	ch clickhouse.Conn,
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
	eventStore *mthub.TradeEventStore,
	emailNotifier **notifier.EmailNotifier,
	platformAgg **risksvc.PlatformAggregator,
	reconLoop **mthub.ReconciliationLoop,
) error {
	// B-2.3: Per-broker 3-level margin call detection.
	// Level 1 (预警): margin_level <= call_pct * 1.5 → SSE only
	// Level 2 (警告): margin_level <= call_pct → SSE + Email
	// Level 3 (危急): margin_level <= call_pct * 0.7 → SSE + Email (1min cooldown)
	var marginCallMu sync.Mutex
	marginCallLastSent := make(map[string]map[int]time.Time)
	// Per-account broker thresholds loaded from mt_accounts.
	marginCallThresholds := make(map[string]float64) // accountID → broker_margin_call_pct
	var snapshotMu sync.Mutex
	lastSnapshot := make(map[string]time.Time) // throttle: 1 snapshot/hour/account

	// Load per-account broker margin call thresholds (default 100.0 from migration 122).
	func() {
		rows, err := pool.Query(context.Background(), `SELECT id, broker_margin_call_pct FROM mt_accounts`)
		if err != nil {
			log.Warn("B-2.3: failed to load margin thresholds, using defaults", zap.Error(err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var aid string
			var pct float64
			if err := rows.Scan(&aid, &pct); err != nil {
				log.Warn("B-2.3: scan margin threshold failed", zap.Error(err))
				continue
			}
			marginCallThresholds[aid] = pct
		}
	}()

	log.Info("mdgateway pipeline starting", zap.String("spill_dir", spillDir))
	if err := mdgateway.Run(pipelineCtx, mdgateway.RunnerDeps{
		Log:      log,
		PG:       pool,
		CH:       ch,
		NATSConn: nc,
		SpillDir: spillDir,
		Secrets:  secClient,
		Hub:      hub,
		OnAccountProfit: func(accountID, userID string, p *mdtick.ProfitUpdate) {
			// Write latest balance/equity to PG via AccountService.
			writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			userUID, err := uuid.Parse(userID)
			if err != nil {
				log.Warn("OnAccountProfit: invalid user UUID", zap.String("userID", userID), zap.Error(err))
			}
			if err := accountSvc.UpdateAccountMetrics(writeCtx, userUID, accountID, p.Balance, p.Equity, p.Credit, p.Margin, p.FreeMargin, p.MarginLevel); err != nil {
				log.Warn("OnAccountProfit: pg update failed", zap.String("account", accountID), zap.Error(err))
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
				if err := accountSvc.RecordBalanceSnapshot(writeCtx, accountID, p.Balance, p.Equity, p.Margin, p.FreeMargin); err != nil {
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
			// B-2.3: 3-level margin call detection with per-broker thresholds.
			if p.MarginLevel > 0 {
				callPct := marginCallThresholds[accountID]
				if callPct <= 0 {
					callPct = 100.0
				}
				service.CheckMarginCall(accountID, userID, p.MarginLevel, p.Margin, p.Equity, callPct, &marginCallMu, marginCallLastSent, eventStore, *emailNotifier)
			}
		},
		OnOrderUpdate: func(accountID, userID string, o *mdtick.OrderUpdate) {
			// Update PG with latest account metrics via AccountService.
			writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			userUID, _ := uuid.Parse(userID)
			if err := accountSvc.UpdateAccountMetrics(writeCtx, userUID, accountID, o.Balance, o.Equity, o.Credit, o.Margin, o.FreeMargin, o.MarginLevel); err != nil {
				log.Warn("OnOrderUpdate: pg update failed", zap.String("account", accountID), zap.Error(err))
			}

			// Publish profit update to AccountProfitBroker for SSE streaming.
			accountBroker.Publish(&mthub.AccountProfitEvent{
				AccountID: accountID, UserID: userID, Platform: o.Platform,
				Balance: o.Balance, Credit: o.Credit, Equity: o.Equity,
				Margin: o.Margin, FreeMargin: o.FreeMargin, MarginLevel: o.MarginLevel,
				Profit: o.Profit, ProfitPercent: o.ProfitPercent, Status: "connected", Timestamp: time.Now(),
				Positions: func() []mthub.AccountProfitPosition {
					out := make([]mthub.AccountProfitPosition, 0, len(o.Positions))
					for _, pos := range o.Positions {
						out = append(out, mthub.AccountProfitPosition{
							Ticket: pos.Ticket, Symbol: pos.Symbol,
							Profit: pos.Profit, Volume: pos.Volume,
							CurrentPrice: pos.CurrentPrice,
						})
					}
					return out
				}(),
			})

			// Publish FULL position snapshot — ticket, symbol, lots, prices,
			// profit all in one event (matching proto OpenedOrders model).
			snapshot := &mthub.PositionSnapshot{
				AccountID:   accountID,
				UserID:      userID,
				Platform:    o.Platform,
				Balance:     o.Balance,
				Credit:      o.Credit,
				Equity:      o.Equity,
				Margin:      o.Margin,
				FreeMargin:  o.FreeMargin,
				MarginLevel: o.MarginLevel,
				Profit:      o.Profit,
				Positions:   make([]mthub.PositionSnapshotItem, 0, len(o.Positions)),
			}
			for _, pos := range o.Positions {
				snapshot.Positions = append(snapshot.Positions, mthub.PositionSnapshotItem{
					Ticket:       pos.Ticket,
					Symbol:       pos.Symbol,
					Type:         pos.Type,
					Volume:       pos.Volume,
					OpenPrice:    pos.OpenPrice,
					CurrentPrice: pos.CurrentPrice,
					StopLoss:     pos.StopLoss,
					TakeProfit:   pos.TakeProfit,
					Profit:       pos.Profit,
					Swap:         pos.Swap,
					Commission:   pos.Commission,
					Comment:      pos.Comment,
					OpenTime:     pos.OpenTime,
				})
			}
			snapshotBroker.Publish(snapshot)

			// B-1.2: Feed PlatformAggregator for risk pipeline Stage 3 (platform_limits).
			(*platformAgg).ClearAccount(accountID)
			for _, pos := range o.Positions {
				netVol := pos.Volume
				if pos.Type == "sell" {
					netVol = -netVol
				}
				(*platformAgg).UpdatePosition(accountID, &risksvc.AggregatorPosition{
					Canonical: pos.Symbol,
					NetVolume: netVol,
					Notional:  pos.Volume * pos.CurrentPrice * 100000,
					Margin:    0,
				})
			}

				// Auto-write closed orders to trade_records.
				if strings.EqualFold(o.UpdateType, "close") && o.UpdateCloseTime > 0 {
					uid, err := uuid.Parse(accountID)
					if err == nil {
						rec := &model.TradeRecord{
							AccountID:    uid,
							Ticket:       o.UpdateTicket,
							Symbol:       o.UpdateSymbol,
							OrderType:    o.UpdateOrderType,
							Volume:       o.UpdateVolume,
							OpenPrice:    o.UpdateOpenPrice,
							ClosePrice:   o.UpdateClosePrice,
							Profit:       o.UpdateProfit,
							Swap:         o.UpdateSwap,
							Commission:   o.UpdateCommission,
							OpenTime:     time.Unix(o.UpdateOpenTime, 0),
							CloseTime:    time.Unix(o.UpdateCloseTime, 0),
							StopLoss:     o.UpdateSL,
							TakeProfit:   o.UpdateTP,
							OrderComment: o.UpdateComment,
							Platform:     o.Platform,
						}
						if err := tradeRecordRepo.Create(writeCtx, rec); err != nil {
							log.Warn("OnOrderUpdate: write closed trade failed", zap.String("account", accountID), zap.Int64("ticket", o.UpdateTicket), zap.Error(err))
						}
					}
				}
		},
		OnAccountDisconnect: func(accountID string) {
			accountSyncSvc.SyncAccountHistory(accountID)
			(*platformAgg).ClearAccount(accountID)
			// Update DB status so frontend doesn't keep showing stale "connected" state.
			writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := accountSvc.DisconnectAccountByID(writeCtx, accountID); err != nil {
				log.Warn("OnAccountDisconnect: failed to update account_status", zap.String("account", accountID), zap.Error(err))
			}
		},
		OnBrokerInfo: func(accountID, platform, broker string, info *mdtick.BrokerInfo) {
			accountSyncSvc.SyncAccountHistory(accountID)
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
							Ticket: o.Ticket, Symbol: o.SymbolRaw, Type: service.MapSideToString(o.Side), Volume: o.Volume.InexactFloat64(),
							OpenPrice: o.OpenPrice.InexactFloat64(), Profit: o.Profit.InexactFloat64(),
							Swap: o.Swap.InexactFloat64(), Commission: o.Commission.InexactFloat64(), Comment: o.Comment,
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
				if err := accountSvc.UpdateBrokerThresholds(ctx, accountID, pct, stop); err != nil {
					log.Warn("failed to persist broker margin info",
						zap.String("account", accountID), zap.Error(err))
				} else {
					log.Info("broker margin thresholds updated",
						zap.String("account", accountID),
						zap.Float64("margin_call_pct", pct),
						zap.Float64("stop_out_pct", stop))
				}
			}
		},
	}); err != nil {
		log.Error("mdgateway pipeline exited with error", zap.Error(err))
		return err
	}
	return nil
}

// convertProfitPositions converts mdtick.ProfitPosition slice to mthub.AccountProfitPosition slice.
func convertProfitPositions(positions []mdtick.ProfitPosition) []mthub.AccountProfitPosition {
	out := make([]mthub.AccountProfitPosition, 0, len(positions))
	for _, pos := range positions {
		out = append(out, mthub.AccountProfitPosition{
			Ticket:       pos.Ticket,
			Symbol:       pos.Symbol,
			Profit:       pos.Profit,
			Volume:       pos.Volume,
			CurrentPrice: pos.CurrentPrice,
		})
	}
	return out
}
