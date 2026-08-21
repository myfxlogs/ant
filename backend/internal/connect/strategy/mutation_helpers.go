// mutation_helpers.go — Helper functions for mutation coordination
// (LIVE-ORDER-REENTRY-1). Extracted from mutation_coordinator.go to keep
// file size within limits.

package strategy

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
)

// publishReadAfterWriteSnapshot publishes the OpenedOrders result into the
// existing PositionSnapshotBroker so PositionCache and all subscribers
// receive the broker's authoritative state.
func (s *StrategyExecutionServer) publishReadAfterWriteSnapshot(
	cfg LiveStrategyConfig, orders []*mthub.OrderRecord,
) {
	if s.mtHub == nil || s.mtHub.SnapshotBroker() == nil {
		return
	}
	snapshot := &mthub.PositionSnapshot{
		AccountID:              cfg.AccountID,
		FinancialsSource:       "read_after_write",
		CapturedAt:             time.Now(),
		PositionsAuthoritative: true,
		PositionsSource:        "opened_orders_confirmation",
		PositionsCapturedAt:    time.Now(),
		Positions:              make([]mthub.PositionSnapshotItem, 0, len(orders)),
		PendingOrders:          make([]mthub.PositionSnapshotItem, 0),
	}
	for _, o := range orders {
		// LIVE-MQL-ORDER-CONTEXT-1: use OrderTypeString for full order type
		// (e.g. "BUY_LIMIT") and split market vs pending.
		typeStr := strings.ToLower(o.OrderTypeString())
		item := mthub.PositionSnapshotItem{
			Ticket: o.Ticket, Symbol: o.Canonical, Type: typeStr, Magic: o.Magic,
			Volume: o.Volume, OpenPrice: o.OpenPrice,
			StopLoss: o.StopLoss, TakeProfit: o.TakeProfit,
			Profit: o.Profit, Swap: o.Swap, Commission: o.Commission,
			Comment: o.Comment, OpenTime: o.OpenTime.Unix(),
		}
		if mdtick.IsPendingOrderType(typeStr) {
			snapshot.PendingOrders = append(snapshot.PendingOrders, item)
		} else {
			snapshot.Positions = append(snapshot.Positions, item)
		}
	}
	s.mtHub.SnapshotBroker().Publish(snapshot)
}

// logOrderLifecycle emits a structured lifecycle log for order events.
func (s *StrategyExecutionServer) logOrderLifecycle(
	activeSess *ActiveSession, cfg LiveStrategyConfig,
	kind, sideStr string, ticket int64, errMsg string,
) {
	fields := []zap.Field{
		zap.String("lifecycle", kind),
		zap.String("account", cfg.AccountID),
		zap.String("symbol", cfg.Symbol),
		zap.String("side", sideStr),
		zap.Int64("ticket", ticket),
	}
	if activeSess != nil {
		fields = append(fields,
			zap.String("run_id", activeSess.RunID.String()),
			zap.String("schedule_id", activeSess.ScheduleID.String()),
			zap.Int32("magic", activeSess.MagicNumber),
		)
		// LIVE-DIAG-TRUTH-1 rework: persist lifecycle + ticket in sessionDiag
		// so diagnostics survive barrier.Release() (which clears transient state).
		if activeSess.diag != nil {
			activeSess.diag.RecordLifecycle(kind, ticket)
		}
	}
	if errMsg != "" {
		fields = append(fields, zap.String("error", errMsg))
	}
	s.log.Info("LiveStrategyRunner: order lifecycle", fields...)
}

// verifyTicketPresent returns true if the given ticket is in the OpenedOrders list.
// Used for open mutations.
func verifyTicketPresent(ticket int64) func([]*mthub.OrderRecord) bool {
	return func(orders []*mthub.OrderRecord) bool {
		for _, o := range orders {
			if o.Ticket == ticket {
				return true
			}
		}
		return false
	}
}

// verifyTicketAbsent returns true if the given ticket is NOT in the OpenedOrders list.
// Used for close and cancel mutations.
func verifyTicketAbsent(ticket int64) func([]*mthub.OrderRecord) bool {
	return func(orders []*mthub.OrderRecord) bool {
		for _, o := range orders {
			if o.Ticket == ticket {
				return false
			}
		}
		return true
	}
}

// verifyTicketModified returns true if the given ticket is in the OpenedOrders
// list AND its SL/TP/price match the requested values. R5: modify read-after-write
// must verify the modification actually took effect, not just ticket presence.
// R5-⑤ (third-round rework): nil pointer = field not provided (don't check);
// non-nil pointer = field explicitly set, including explicit zero (e.g.
// clearing SL/TP to 0). This distinguishes "unspecified" from "set to zero".
func verifyTicketModified(ticket int64, sl, tp, price *decimal.Decimal) func([]*mthub.OrderRecord) bool {
	return func(orders []*mthub.OrderRecord) bool {
		for _, o := range orders {
			if o.Ticket == ticket {
				// R5-⑤: check each field only if it was explicitly provided.
				// A non-nil pointer to decimal.Zero verifies the broker
				// actually cleared the field (SL/TP = 0).
				if sl != nil && !o.StopLoss.Equal(*sl) {
					return false
				}
				if tp != nil && !o.TakeProfit.Equal(*tp) {
					return false
				}
				if price != nil && !o.OpenPrice.Equal(*price) {
					return false
				}
				return true
			}
		}
		return false
	}
}
