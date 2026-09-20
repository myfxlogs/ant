package mthub

// TRADE-RECORDS-DUP-1 T1: the ledger-write guard matrix.
//
// Broker history carries rows that must never enter the closed-trade ledger:
// open/pending rows (State != Closed, epoch close time after verbatim
// mapping) and balance/credit cash events. SyncableClosedTrade is the single
// source both sync sites call.
//
// Mutations: M1 drop the State branch → Open/Cancelled cases write → RED.
// M2 drop the CloseTime.IsZero() clause → zero-time closed row writes → RED
// (double-layer discrimination: State alone is not sufficient).

import (
	"testing"
	"time"
)

func TestSyncableClosedTrade_GuardMatrix(t *testing.T) {
	realClose := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		state OrderState
		typ   OrderType
		close time.Time
		want  bool
	}{
		{"closed_trade", OrderStateClosed, OrderMarket, realClose, true},
		{"open_state_epoch_close", OrderStateOpen, OrderMarket, realClose, false},
		{"closed_but_zero_close_time", OrderStateClosed, OrderMarket, time.Time{}, false},
		{"balance_cash_event", OrderStateClosed, OrderBalance, realClose, false},
		{"credit_cash_event", OrderStateClosed, OrderCredit, realClose, false},
		{"pending_limit", OrderStatePending, OrderLimit, realClose, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &OrderRecord{State: tc.state, OrderType: tc.typ, CloseTime: tc.close}
			if got := r.SyncableClosedTrade(); got != tc.want {
				t.Errorf("SyncableClosedTrade(State=%v, Type=%v, CloseTime zero=%v) = %v, want %v",
					tc.state, tc.typ, tc.close.IsZero(), got, tc.want)
			}
		})
	}
}
