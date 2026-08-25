package mthub

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// lossyFloat64 converts a decimal to float64 for MT API proto boundaries.
// Precision loss is detected but not rejected — the MT proto requires float64.
func (s *MtHubService) omsTransition(ctx context.Context, orderID, accountID string, from, to OMSState) {
	if s.omsWriter == nil || orderID == "" {
		return
	}
	if err := s.omsWriter.Transition(ctx, orderID, accountID, from, to); err != nil {
		if s.logger != nil {
			s.logger.Error("oms transition failed",
				zap.Error(err),
				zap.String("orderID", orderID),
				zap.String("from", string(from)),
				zap.String("to", string(to)))
		}
	}
}

// TransitionOrderByTicket looks up the order by broker ticket and transitions
// its OMS state. Used by OnOrderUpdate callback to move orders from SUBMITTED
// to terminal states (FILLED/CANCELLED/FAILED) when broker fill/reject events arrive.
//
// Task 2 (SUBMIT-STUCK-RACE): If the ticket is not yet backfilled in OMS
// (broker fill event arrived before PlaceOrder RPC returned), a delayed retry
// covers the ms-level race. If still not found after retry, triggers
// reconciliation as a fallback (REUSE: ReconciliationLoop.TriggerReconcile).
func (s *MtHubService) TransitionOrderByTicket(ctx context.Context, accountID string, ticket int64, to OMSState) {
	if s.omsWriter == nil {
		return
	}
	orderID, currentState, err := s.omsWriter.OrderIDByTicket(ctx, accountID, ticket)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("oms: order not found by ticket, scheduling retry",
				zap.String("accountID", accountID), zap.Int64("ticket", ticket), zap.Error(err))
		}
		// Delayed retry in a detached goroutine — covers ms-level backfill race.
		go s.retryTransitionByTicket(accountID, ticket, to)
		return
	}
	s.omsTransition(ctx, orderID, accountID, OMSState(currentState), to)
}

// retryTransitionByTicket retries the ticket lookup after a short delay.
// If the ticket is now found, transitions the order. If still not found,
// triggers reconciliation as a fallback (if reconcileTrigger is configured).
func (s *MtHubService) retryTransitionByTicket(accountID string, ticket int64, to OMSState) {
	time.AfterFunc(2*time.Second, func() {
		retryCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		orderID, currentState, err := s.omsWriter.OrderIDByTicket(retryCtx, accountID, ticket)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn("oms: order still not found by ticket after retry, triggering reconciliation",
					zap.String("accountID", accountID), zap.Int64("ticket", ticket))
			}
			if s.reconcileTrigger != nil {
				s.reconcileTrigger(accountID)
			}
			return
		}
		s.omsTransition(retryCtx, orderID, accountID, OMSState(currentState), to)
	})
}

// PublishTradeEventFromUpdate publishes a BrokerTradeEvent derived from an
// OnOrderUpdate event. This bridges broker fill/close events to the TradeBroker,
// enabling strategy OnTrade callbacks (trailing stops, martingale, etc.).
func (s *MtHubService) PublishTradeEventFromUpdate(
	accountID, updateType, orderType, symbol string,
	ticket int64,
	volume, closePrice, sl, tp, profit, commission, swap decimal.Decimal,
) {
	if s.tradeBroker == nil {
		return
	}
	var eventType BrokerTradeEventType
	switch strings.ToLower(updateType) {
	case "close", "pending_close":
		eventType = BrokerTradeClosed
	case "modify":
		eventType = BrokerTradeModified
	case "delete":
		eventType = BrokerTradeCancelled
	default:
		eventType = BrokerTradeFilled
	}
	s.tradeBroker.Publish(&BrokerTradeEvent{
		AccountID:  accountID,
		Ticket:     ticket,
		Symbol:     symbol,
		EventType:  eventType,
		Side:       orderType,
		Volume:     volume,
		Price:      closePrice,
		StopLoss:   sl,
		TakeProfit: tp,
		Profit:     profit,
		Commission: commission,
		Swap:       swap,
	})
}
