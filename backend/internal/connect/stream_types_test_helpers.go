package connect

import (
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
)

func (a *AccountStream) SetProfitSnapshotForTest(ev *v1.ProfitUpdateEvent) {
	a.setProfitSnapshot(ev)
}

func (a *AccountStream) UpsertOpenedOrderForTest(ev *v1.OrderUpdateEvent) {
	a.upsertOpenedOrder(ev)
}

func (s *StreamService) GetOrCreateAccountStreamForTest(accountID string) *AccountStream {
	if s == nil || accountID == "" {
		return nil
	}
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	if s.accountStreams == nil {
		s.accountStreams = make(map[string]*AccountStream)
	}
	if as, ok := s.accountStreams[accountID]; ok && as != nil {
		return as
	}
	ctx, cancel := context.WithCancel(context.Background())
	as := &AccountStream{
		AccountID:    accountID,
		Subscribers: make(map[string]*StreamSubscriber),
		Ctx:         ctx,
		Cancel:      cancel,
	}
	s.accountStreams[accountID] = as
	return as
}

func (s *StreamService) BuildSnapshotEventsForTest(accountID string) []*v1.StreamEvent {
	if s == nil || accountID == "" {
		return nil
	}
	as, ok := s.getAccountStream(accountID)
	if !ok || as == nil {
		return nil
	}
	out := make([]*v1.StreamEvent, 0, 16)
	if p := as.getProfitSnapshot(); p != nil {
		out = append(out, &v1.StreamEvent{
			Type:      "profit_update",
			AccountId: accountID,
			Timestamp: timestamppb.Now(),
			Payload:   &v1.StreamEvent_ProfitUpdate{ProfitUpdate: p},
		})
	}
	if led := as.GetLedgerSnapshot(); led != nil {
		out = append(out, &v1.StreamEvent{
			Type:      "ledger_entry",
			AccountId: accountID,
			Timestamp: timestamppb.Now(),
			Payload:   &v1.StreamEvent_LedgerEntry{LedgerEntry: led},
		})
	}
	for _, pos := range as.GetPositionsSnapshot() {
		if pos == nil {
			continue
		}
		out = append(out, &v1.StreamEvent{
			Type:      "position_update",
			AccountId: accountID,
			Timestamp: timestamppb.Now(),
			Payload:   &v1.StreamEvent_PositionUpdate{PositionUpdate: pos},
		})
	}
	for _, d := range as.GetDealsSnapshot() {
		if d == nil {
			continue
		}
		out = append(out, &v1.StreamEvent{
			Type:      "deal_update",
			AccountId: accountID,
			Timestamp: timestamppb.Now(),
			Payload:   &v1.StreamEvent_DealUpdate{DealUpdate: d},
		})
	}
	for _, o := range as.getOpenedOrdersSnapshot() {
		if o == nil {
			continue
		}
		out = append(out, &v1.StreamEvent{
			Type:      "order_update",
			AccountId: accountID,
			Timestamp: timestamppb.Now(),
			Payload:   &v1.StreamEvent_OrderUpdate{OrderUpdate: o},
		})
	}
	return out
}
