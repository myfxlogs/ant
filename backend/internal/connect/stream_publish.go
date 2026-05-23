package connect

import (
	"context"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/coordination"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func (s *StreamService) BroadcastAccountStatus(accountID string, status string, message string) {
	if s == nil {
		return
	}
	if !s.isLeaderForAccount(accountID) {
		return
	}
	if accountID == "" {
		return
	}

	as, ok := s.getAccountStream(accountID)
	if !ok || as == nil {
		return
	}

	// De-dup / throttle status events to avoid spamming streams during flapping.
	// - If status+message unchanged within a short window, skip.
	// - Otherwise publish and update cache.
	now := time.Now()
	const sameWindow = 10 * time.Second
	s.statusMu.Lock()
	if s.lastAccountStatus != nil {
		if last, ok := s.lastAccountStatus[accountID]; ok {
			if last.status == status && last.message == message && !last.at.IsZero() && now.Sub(last.at) < sameWindow {
				s.statusMu.Unlock()
				return
			}
		}
		s.lastAccountStatus[accountID] = accountStatusRecord{status: status, message: message, at: now}
	}
	s.statusMu.Unlock()

	sev := &v1.StreamEvent{
		Type:      "account_status",
		AccountId: accountID,
		Timestamp: timestamppb.Now(),
		Payload: &v1.StreamEvent_AccountStatus{
			AccountStatus: &v1.AccountStatusEvent{
				AccountId: accountID,
				Status:    status,
				Message:   message,
			},
		},
	}
	if id, err := s.appendStreamEvent(context.Background(), accountID, "account_status", sev); err == nil && id != "" {
		sev.EventId = id
	} else if err != nil {
		logger.Warn("stream publish: append account_status failed", zap.String("account_id", accountID), zap.Error(err))
	}
	if s.eventBus != nil {
		if data, err := protojson.Marshal(sev.GetAccountStatus()); err == nil {
			if err := s.eventBus.Publish(context.Background(), accountID, coordination.EventTypeAccountStatus, data); err != nil {
				logger.Warn("stream publish: eventBus account_status publish failed", zap.String("account_id", accountID), zap.Error(err))
			}
		} else {
			logger.Warn("stream publish: marshal account_status failed", zap.String("account_id", accountID), zap.Error(err))
		}
	}
}

func (s *StreamService) publishProfitEvent(accountID string, ev *v1.ProfitUpdateEvent) {
	if s == nil || ev == nil || accountID == "" {
		return
	}
	if !s.isLeaderForAccount(accountID) {
		return
	}
	sev := &v1.StreamEvent{
		Type:      "profit_update",
		AccountId: accountID,
		Timestamp: timestamppb.Now(),
		Payload:   &v1.StreamEvent_ProfitUpdate{ProfitUpdate: ev},
	}
	if id, err := s.appendStreamEvent(context.Background(), accountID, "profit_update", sev); err == nil && id != "" {
		sev.EventId = id
	} else if err != nil {
		logger.Warn("stream publish: append profit_update failed", zap.String("account_id", accountID), zap.Error(err))
	}
	if s.eventBus != nil {
		if data, err := protojson.Marshal(ev); err == nil {
			if err := s.eventBus.Publish(context.Background(), accountID, coordination.EventTypeProfit, data); err != nil {
				logger.Warn("stream publish: eventBus profit publish failed", zap.String("account_id", accountID), zap.Error(err))
			}
		} else {
			logger.Warn("stream publish: marshal profit_update failed", zap.String("account_id", accountID), zap.Error(err))
		}
	}
}

func (s *StreamService) publishQuoteEvent(accountID string, ev *v1.QuoteEvent) {
	if s == nil || ev == nil || accountID == "" {
		return
	}
	if !s.isLeaderForAccount(accountID) {
		return
	}
	sev := &v1.StreamEvent{
		Type:      "quote_tick",
		AccountId: accountID,
		Timestamp: timestamppb.Now(),
		Payload:   &v1.StreamEvent_Quote{Quote: ev},
	}
	if id, err := s.appendStreamEvent(context.Background(), accountID, "quote_tick", sev); err == nil && id != "" {
		sev.EventId = id
	} else if err != nil {
		logger.Warn("stream publish: append quote_tick failed", zap.String("account_id", accountID), zap.Error(err))
	}
}

func (s *StreamService) publishOrderEvent(accountID string, ev *v1.OrderUpdateEvent) {
	if s == nil || ev == nil || accountID == "" {
		return
	}
	if !s.isLeaderForAccount(accountID) {
		return
	}
	sev := &v1.StreamEvent{
		Type:      "order_update",
		AccountId: accountID,
		Timestamp: timestamppb.Now(),
		Payload:   &v1.StreamEvent_OrderUpdate{OrderUpdate: ev},
	}
	if id, err := s.appendStreamEvent(context.Background(), accountID, "order_update", sev); err == nil && id != "" {
		sev.EventId = id
	} else if err != nil {
		logger.Warn("stream publish: append order_update failed", zap.String("account_id", accountID), zap.Error(err))
	}
	if s.eventBus != nil {
		if data, err := protojson.Marshal(ev); err == nil {
			if err := s.eventBus.Publish(context.Background(), accountID, coordination.EventTypeOrder, data); err != nil {
				logger.Warn("stream publish: eventBus order publish failed", zap.String("account_id", accountID), zap.Error(err))
			}
		} else {
			logger.Warn("stream publish: marshal order_update failed", zap.String("account_id", accountID), zap.Error(err))
		}
	}
}

func (s *StreamService) publishSyncEvent(accountID string, ev *v1.SyncEvent) {
	if s == nil || ev == nil || accountID == "" {
		return
	}
	if !s.isLeaderForAccount(accountID) {
		return
	}
	sev := &v1.StreamEvent{
		Type:      "sync",
		AccountId: accountID,
		Timestamp: timestamppb.Now(),
		Payload:   &v1.StreamEvent_Sync{Sync: ev},
	}
	if id, err := s.appendStreamEvent(context.Background(), accountID, "sync", sev); err == nil && id != "" {
		sev.EventId = id
	} else if err != nil {
		logger.Warn("stream publish: append sync failed", zap.String("account_id", accountID), zap.Error(err))
	}
}

func (s *StreamService) publishDealEvent(accountID string, ev *v1.DealUpdateEvent) {
	if s == nil || ev == nil || accountID == "" {
		return
	}
	if !s.isLeaderForAccount(accountID) {
		return
	}
	sev := &v1.StreamEvent{
		Type:      "deal_update",
		AccountId: accountID,
		Timestamp: timestamppb.Now(),
		Payload:   &v1.StreamEvent_DealUpdate{DealUpdate: ev},
	}
	if id, err := s.appendStreamEvent(context.Background(), accountID, "deal_update", sev); err == nil && id != "" {
		sev.EventId = id
	} else if err != nil {
		logger.Warn("stream publish: append deal_update failed", zap.String("account_id", accountID), zap.Error(err))
	}
}

func (s *StreamService) publishPositionEvent(accountID string, ev *v1.PositionUpdateEvent) {
	if s == nil || ev == nil || accountID == "" {
		return
	}
	if !s.isLeaderForAccount(accountID) {
		return
	}
	sev := &v1.StreamEvent{
		Type:      "position_update",
		AccountId: accountID,
		Timestamp: timestamppb.Now(),
		Payload:   &v1.StreamEvent_PositionUpdate{PositionUpdate: ev},
	}
	if id, err := s.appendStreamEvent(context.Background(), accountID, "position_update", sev); err == nil && id != "" {
		sev.EventId = id
	} else if err != nil {
		logger.Warn("stream publish: append position_update failed", zap.String("account_id", accountID), zap.Error(err))
	}
}

func (s *StreamService) publishLedgerEntryEvent(accountID string, ev *v1.LedgerEntryEvent) {
	if s == nil || ev == nil || accountID == "" {
		return
	}
	if !s.isLeaderForAccount(accountID) {
		return
	}
	sev := &v1.StreamEvent{
		Type:      "ledger_entry",
		AccountId: accountID,
		Timestamp: timestamppb.Now(),
		Payload:   &v1.StreamEvent_LedgerEntry{LedgerEntry: ev},
	}
	if id, err := s.appendStreamEvent(context.Background(), accountID, "ledger_entry", sev); err == nil && id != "" {
		sev.EventId = id
	} else if err != nil {
		logger.Warn("stream publish: append ledger_entry failed", zap.String("account_id", accountID), zap.Error(err))
	}
}

func (s *StreamService) publishTradeCommandEvent(accountID string, ev *v1.TradeCommandEvent) {
	if s == nil || ev == nil || accountID == "" {
		return
	}
	if !s.isLeaderForAccount(accountID) {
		return
	}
	sev := &v1.StreamEvent{
		Type:      "trade_command",
		AccountId: accountID,
		Timestamp: timestamppb.Now(),
		Payload:   &v1.StreamEvent_TradeCommand{TradeCommand: ev},
	}
	if id, err := s.appendStreamEvent(context.Background(), accountID, "trade_command", sev); err == nil && id != "" {
		sev.EventId = id
	} else if err != nil {
		logger.Warn("stream publish: append trade_command failed", zap.String("account_id", accountID), zap.Error(err))
	}
}

func (s *StreamService) publishTradeReceiptEvent(accountID string, ev *v1.TradeReceiptEvent) {
	if s == nil || ev == nil || accountID == "" {
		return
	}
	if !s.isLeaderForAccount(accountID) {
		return
	}
	sev := &v1.StreamEvent{
		Type:      "trade_receipt",
		AccountId: accountID,
		Timestamp: timestamppb.Now(),
		Payload:   &v1.StreamEvent_TradeReceipt{TradeReceipt: ev},
	}
	if id, err := s.appendStreamEvent(context.Background(), accountID, "trade_receipt", sev); err == nil && id != "" {
		sev.EventId = id
	} else if err != nil {
		logger.Warn("stream publish: append trade_receipt failed", zap.String("account_id", accountID), zap.Error(err))
	}
}
