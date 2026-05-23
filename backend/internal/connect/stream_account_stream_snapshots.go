package connect

import (
	v1 "anttrader/gen/proto"
)

func (a *AccountStream) UpsertPosition(ev *v1.PositionUpdateEvent) {
	if a == nil || ev == nil {
		return
	}
	a.snapshotMu.Lock()
	if a.positions == nil {
		a.positions = make(map[int64]*v1.PositionUpdateEvent)
	}
	// convention: if close_time is set (>0) consider it closed and remove from snapshot.
	if ev.CloseTime > 0 {
		delete(a.positions, ev.PositionTicket)
	} else {
		a.positions[ev.PositionTicket] = ev
	}
	a.snapshotMu.Unlock()
}

func (a *AccountStream) GetPositionsSnapshot() []*v1.PositionUpdateEvent {
	if a == nil {
		return nil
	}
	a.snapshotMu.RLock()
	if len(a.positions) == 0 {
		a.snapshotMu.RUnlock()
		return nil
	}
	res := make([]*v1.PositionUpdateEvent, 0, len(a.positions))
	for _, v := range a.positions {
		res = append(res, v)
	}
	a.snapshotMu.RUnlock()
	return res
}

func (a *AccountStream) UpsertDeal(ev *v1.DealUpdateEvent) {
	if a == nil || ev == nil {
		return
	}
	a.snapshotMu.Lock()
	if a.deals == nil {
		a.deals = make(map[int64]*v1.DealUpdateEvent)
	}
	a.deals[ev.DealTicket] = ev
	a.snapshotMu.Unlock()
}

func (a *AccountStream) GetDealsSnapshot() []*v1.DealUpdateEvent {
	if a == nil {
		return nil
	}
	a.snapshotMu.RLock()
	if len(a.deals) == 0 {
		a.snapshotMu.RUnlock()
		return nil
	}
	res := make([]*v1.DealUpdateEvent, 0, len(a.deals))
	for _, v := range a.deals {
		res = append(res, v)
	}
	a.snapshotMu.RUnlock()
	return res
}

func (a *AccountStream) SetLedgerSnapshot(ev *v1.LedgerEntryEvent) {
	a.setLedgerSnapshot(ev)
}

func (a *AccountStream) GetLedgerSnapshot() *v1.LedgerEntryEvent {
	return a.getLedgerSnapshot()
}

func (a *AccountStream) setProfitSnapshot(ev *v1.ProfitUpdateEvent) {
	if a == nil {
		return
	}
	a.snapshotMu.Lock()
	a.lastProfit = ev
	a.profitVer.Add(1)
	notify := a.profitNotify
	a.snapshotMu.Unlock()
	if notify == nil {
		return
	}
	select {
	case notify <- struct{}{}:
	default:
		// latest-wins notify
	}
}

func (a *AccountStream) profitNotifyCh() <-chan struct{} {
	if a == nil {
		return nil
	}
	a.snapshotMu.RLock()
	ch := a.profitNotify
	a.snapshotMu.RUnlock()
	return ch
}

func (a *AccountStream) getProfitSnapshot() *v1.ProfitUpdateEvent {
	if a == nil {
		return nil
	}
	a.snapshotMu.RLock()
	ev := a.lastProfit
	a.snapshotMu.RUnlock()
	return ev
}

func (a *AccountStream) setLedgerSnapshot(ev *v1.LedgerEntryEvent) {
	if a == nil {
		return
	}
	a.snapshotMu.Lock()
	a.lastLedger = ev
	a.snapshotMu.Unlock()
}

func (a *AccountStream) getLedgerSnapshot() *v1.LedgerEntryEvent {
	if a == nil {
		return nil
	}
	a.snapshotMu.RLock()
	ev := a.lastLedger
	a.snapshotMu.RUnlock()
	return ev
}

func (a *AccountStream) upsertOpenedOrder(ev *v1.OrderUpdateEvent) {
	if a == nil || ev == nil {
		return
	}
	a.snapshotMu.Lock()
	if a.openedOrders == nil {
		a.openedOrders = make(map[int64]*v1.OrderUpdateEvent)
	}
	// convention: if close_time is set (>0) consider it closed and remove from snapshot.
	if ev.CloseTime > 0 {
		delete(a.openedOrders, ev.Ticket)
	} else {
		a.openedOrders[ev.Ticket] = ev
	}
	a.lastOrderDelta = ev
	a.orderVer.Add(1)
	notify := a.orderNotify
	a.snapshotMu.Unlock()
	if notify == nil {
		return
	}
	select {
	case notify <- struct{}{}:
	default:
	}
}

func (a *AccountStream) orderNotifyCh() <-chan struct{} {
	if a == nil {
		return nil
	}
	a.snapshotMu.RLock()
	ch := a.orderNotify
	a.snapshotMu.RUnlock()
	return ch
}

func (a *AccountStream) getLastOrderDelta() (*v1.OrderUpdateEvent, uint64) {
	if a == nil {
		return nil, 0
	}
	a.snapshotMu.RLock()
	ev := a.lastOrderDelta
	ver := a.orderVer.Load()
	a.snapshotMu.RUnlock()
	return ev, ver
}

func (a *AccountStream) getOpenedOrdersSnapshot() []*v1.OrderUpdateEvent {
	if a == nil {
		return nil
	}
	a.snapshotMu.RLock()
	if len(a.openedOrders) == 0 {
		a.snapshotMu.RUnlock()
		return nil
	}
	res := make([]*v1.OrderUpdateEvent, 0, len(a.openedOrders))
	for _, v := range a.openedOrders {
		res = append(res, v)
	}
	a.snapshotMu.RUnlock()
	return res
}
