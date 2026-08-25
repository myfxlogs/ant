// pipeline_state_callbacks.go — Broker info, disconnect, status callbacks and helper
// extracted from pipeline.go (H1 file) for file-lines compliance.
package main

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
	"alphaforge/internal/risksvc"
	"alphaforge/internal/service"
)

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
