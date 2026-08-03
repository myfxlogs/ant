package backtest

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

func d(s string) decimal.Decimal {
	v, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return v
}

func TestSimBroker_PositionModify(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	res, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"),
	})
	ticket := res.Ticket

	// Modify SL/TP.
	res2, err := broker.PositionModify(ticket, d("1.05"), d("1.20"))
	if err != nil {
		t.Fatalf("PositionModify failed: %v", err)
	}
	if res2.RetCode != sdk.RetDone {
		t.Errorf("RetCode = %v, want Done", res2.RetCode)
	}

	// Verify the modification was applied.
	for _, p := range broker.positions {
		if p.Ticket == ticket {
			if !p.StopLoss.Equal(d("1.05")) {
				t.Errorf("SL = %s, want 1.05", p.StopLoss)
			}
			if !p.TakeProfit.Equal(d("1.20")) {
				t.Errorf("TP = %s, want 1.20", p.TakeProfit)
			}
		}
	}

	// Modify non-existent ticket.
	res3, _ := broker.PositionModify(99999, decimal.Zero, decimal.Zero)
	if res3.RetCode != sdk.RetRejected {
		t.Errorf("PositionModify on non-existent ticket should be Rejected")
	}
}

func TestSimBroker_OrderDelete(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	res, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderLimit,
		Volume: d("0.1"), Price: d("1.0"),
	})
	ticket := res.Ticket

	// Delete the pending order.
	res2, err := broker.OrderDelete(ticket)
	if err != nil {
		t.Fatalf("OrderDelete failed: %v", err)
	}
	if res2.RetCode != sdk.RetDone {
		t.Errorf("RetCode = %v, want Done", res2.RetCode)
	}
	if len(broker.pending) != 0 {
		t.Errorf("pending count = %d, want 0", len(broker.pending))
	}
	if len(broker.history) != 1 {
		t.Errorf("history count = %d, want 1 (cancelled order)", len(broker.history))
	}

	// Delete non-existent ticket.
	res3, _ := broker.OrderDelete(99999)
	if res3.RetCode != sdk.RetRejected {
		t.Errorf("OrderDelete on non-existent should be Rejected")
	}
}

func TestSimBroker_Orders(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderLimit,
		Volume: d("0.1"), Price: d("1.0"), Magic: 100,
	})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideSell, Type: sdk.OrderStop,
		Volume: d("0.2"), Price: d("1.5"), Magic: 200,
	})

	all := broker.Orders(0)
	if len(all) != 2 {
		t.Fatalf("Orders(0) = %d, want 2", len(all))
	}

	filtered := broker.Orders(100)
	if len(filtered) != 1 {
		t.Errorf("Orders(100) = %d, want 1", len(filtered))
	}
}

func TestSimBroker_Positions_FilterByMagic(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"), Magic: 100,
	})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideSell, Type: sdk.OrderMarket,
		Volume: d("0.2"), Price: d("1.1"), Magic: 200,
	})

	all := broker.Positions(0)
	if len(all) != 2 {
		t.Fatalf("Positions(0) = %d, want 2", len(all))
	}

	filtered := broker.Positions(100)
	if len(filtered) != 1 {
		t.Errorf("Positions(100) = %d, want 1", len(filtered))
	}
}

func TestSimBroker_ApplyCommission(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: d("100000"),
		Leverage:       100,
		Commission:     d("0.0003"),
		ContractSize:   d("100000"),
	})
	before := broker.balance
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("1.0"), Price: d("1.1"),
	})
	if !broker.balance.LessThan(before) {
		t.Errorf("balance should decrease after commission, got %s (before %s)", broker.balance, before)
	}
}

func TestSimBroker_ApplySwap(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: d("100000"),
		Leverage:       100,
		SwapRate:       d("0.00001"),
		ContractSize:   d("100000"),
	})
	res, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("1.0"), Price: d("1.1"),
	})
	for _, p := range broker.positions {
		if p.Ticket == res.Ticket {
			broker.applySwap(p, 3)
			if !p.Swap.IsPositive() {
				t.Errorf("Swap should be positive after 3 days, got %s", p.Swap)
			}
			break
		}
	}
}

func TestSimBroker_ApplySwap_DefaultRate(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: d("100000"),
		Leverage:       100,
		ContractSize:   d("100000"),
	})
	res, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("1.0"), Price: d("1.1"),
	})
	for _, p := range broker.positions {
		if p.Ticket == res.Ticket {
			broker.applySwap(p, 1)
			if !p.Swap.IsPositive() {
				t.Errorf("Swap should be positive with default rate, got %s", p.Swap)
			}
			break
		}
	}
}

func TestSimBroker_ApplyCommission_DefaultContractSize(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: d("100000"),
		Leverage:       100,
		Commission:     d("0.001"),
		// ContractSize zero — should default to 100000
	})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("0.5"), Price: d("1.2"),
	})
	if !broker.balance.LessThan(d("100000")) {
		t.Errorf("balance should decrease with default contract size, got %s", broker.balance)
	}
}

func TestSimBroker_SymbolInfo(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: d("100000"),
		Leverage:       100,
		Symbol:       "EURUSD",
		SymbolDigits: 5,
		SymbolPoint:  d("0.00001"),
		VolumeMin:    d("0.01"),
		VolumeMax:    d("100"),
		VolumeStep:   d("0.01"),
	})
	si, err := broker.SymbolInfo("EURUSD")
	if err != nil {
		t.Fatalf("SymbolInfo failed: %v", err)
	}
	if si.Digits != 5 {
		t.Errorf("Digits = %d, want 5", si.Digits)
	}
	if !si.Point.Equal(d("0.00001")) {
		t.Errorf("Point = %s, want 0.00001", si.Point)
	}
}

func TestSimBroker_Account(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: d("50000"),
		Leverage:       200,
	})
	info := broker.Account()
	if !info.Balance.Equal(d("50000")) {
		t.Errorf("Balance = %s, want 50000", info.Balance)
	}
	if info.Leverage != 200 {
		t.Errorf("Leverage = %d, want 200", info.Leverage)
	}
}

func TestSimBroker_Deals(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	res, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"),
	})
	for _, p := range broker.positions {
		if p.Ticket == res.Ticket {
			p.ClosePrice = d("1.2")
			break
		}
	}
	broker.PositionClose(res.Ticket, decimal.Zero)

	deals := broker.Deals(0, 0, 0)
	if len(deals) != 1 {
		t.Errorf("Deals() = %d, want 1", len(deals))
	}
}

func TestSimBroker_HistoryOrders(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	res, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"),
	})
	for _, p := range broker.positions {
		if p.Ticket == res.Ticket {
			p.ClosePrice = d("1.2")
			break
		}
	}
	broker.PositionClose(res.Ticket, decimal.Zero)

	hist := broker.HistoryOrders(0, 0)
	if len(hist) != 1 {
		t.Errorf("HistoryOrders() = %d, want 1", len(hist))
	}
}

func TestSimBroker_SetBarPrice(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	broker.SetBar(5)
	broker.SetBarTime(time.Now())
	broker.SetBarPrice(d("1.2345"))
	if !broker.currentPrice.Equal(d("1.2345")) {
		t.Errorf("currentPrice = %s, want 1.2345", broker.currentPrice)
	}
}

// --- Engine tests ---

func TestEngine_CheckPendingOrders_LimitBuy(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderLimit,
		Volume: d("0.1"), Price: d("1.0"),
	})
	if len(broker.pending) != 1 {
		t.Fatalf("pending = %d, want 1", len(broker.pending))
	}

	engine := &Engine{broker: broker, config: Config{}}
	// Bar with low <= limit price → fill.
	engine.checkPendingOrders(sdk.Bar{
		High: d("1.1"), Low: d("0.95"), Close: d("1.05"),
	})
	if len(broker.pending) != 0 {
		t.Errorf("pending should be 0 after fill, got %d", len(broker.pending))
	}
	if len(broker.positions) != 1 {
		t.Errorf("positions should be 1 after fill, got %d", len(broker.positions))
	}
}

func TestEngine_CheckPendingOrders_LimitSell(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideSell, Type: sdk.OrderLimit,
		Volume: d("0.1"), Price: d("1.2"),
	})

	engine := &Engine{broker: broker, config: Config{}}
	// Bar with high >= limit price → fill.
	engine.checkPendingOrders(sdk.Bar{
		High: d("1.25"), Low: d("1.1"), Close: d("1.15"),
	})
	if len(broker.pending) != 0 {
		t.Errorf("pending should be 0 after fill, got %d", len(broker.pending))
	}
}

func TestEngine_CheckPendingOrders_StopBuy(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderStop,
		Volume: d("0.1"), Price: d("1.2"),
	})

	engine := &Engine{broker: broker, config: Config{}}
	// Bar with high >= stop price → fill.
	engine.checkPendingOrders(sdk.Bar{
		High: d("1.25"), Low: d("1.1"), Close: d("1.15"),
	})
	if len(broker.pending) != 0 {
		t.Errorf("pending should be 0 after stop fill, got %d", len(broker.pending))
	}
}

func TestEngine_CheckPendingOrders_StopSell(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideSell, Type: sdk.OrderStop,
		Volume: d("0.1"), Price: d("1.0"),
	})

	engine := &Engine{broker: broker, config: Config{}}
	// Bar with low <= stop price → fill.
	engine.checkPendingOrders(sdk.Bar{
		High: d("1.1"), Low: d("0.95"), Close: d("1.05"),
	})
	if len(broker.pending) != 0 {
		t.Errorf("pending should be 0 after stop fill, got %d", len(broker.pending))
	}
}

func TestEngine_CheckPendingOrders_NoFill(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderLimit,
		Volume: d("0.1"), Price: d("0.9"),
	})

	engine := &Engine{broker: broker, config: Config{}}
	// Bar with low > limit price → no fill.
	engine.checkPendingOrders(sdk.Bar{
		High: d("1.1"), Low: d("1.0"), Close: d("1.05"),
	})
	if len(broker.pending) != 1 {
		t.Errorf("pending should still be 1 (no fill), got %d", len(broker.pending))
	}
}

func TestEngine_CheckSLTP_Buy_TP(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100, ContractSize: d("100000")})
	res, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"),
		TakeProfit: d("1.2"),
	})

	engine := &Engine{broker: broker, config: Config{ContractSize: d("100000")}}
	// Bar with high >= TP → close.
	engine.checkSLTP(sdk.Bar{
		High: d("1.25"), Low: d("1.1"), Close: d("1.15"),
	})
	if len(broker.positions) != 0 {
		t.Errorf("position should be closed at TP, got %d positions", len(broker.positions))
	}
	if len(broker.trades) != 1 {
		t.Errorf("trades should be 1, got %d", len(broker.trades))
	}
	_ = res
}

func TestEngine_CheckSLTP_Buy_SL(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100, ContractSize: d("100000")})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"),
		StopLoss: d("1.05"),
	})

	engine := &Engine{broker: broker, config: Config{ContractSize: d("100000")}}
	// Bar with low <= SL → close.
	engine.checkSLTP(sdk.Bar{
		High: d("1.15"), Low: d("1.0"), Close: d("1.05"),
	})
	if len(broker.positions) != 0 {
		t.Errorf("position should be closed at SL, got %d positions", len(broker.positions))
	}
}

func TestEngine_CheckSLTP_Sell_TP(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100, ContractSize: d("100000")})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideSell, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"),
		TakeProfit: d("1.0"),
	})

	engine := &Engine{broker: broker, config: Config{ContractSize: d("100000")}}
	// For sell: TP triggers when low <= TP.
	engine.checkSLTP(sdk.Bar{
		High: d("1.15"), Low: d("0.95"), Close: d("1.0"),
	})
	if len(broker.positions) != 0 {
		t.Errorf("sell position should be closed at TP, got %d positions", len(broker.positions))
	}
}

func TestEngine_CheckSLTP_Sell_SL(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100, ContractSize: d("100000")})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideSell, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"),
		StopLoss: d("1.2"),
	})

	engine := &Engine{broker: broker, config: Config{ContractSize: d("100000")}}
	// For sell: SL triggers when high >= SL.
	engine.checkSLTP(sdk.Bar{
		High: d("1.25"), Low: d("1.1"), Close: d("1.15"),
	})
	if len(broker.positions) != 0 {
		t.Errorf("sell position should be closed at SL, got %d positions", len(broker.positions))
	}
}

func TestEngine_CheckSLTP_NoHit(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"),
		StopLoss: d("1.0"), TakeProfit: d("1.2"),
	})

	engine := &Engine{broker: broker, config: Config{}}
	// Bar within SL/TP range → no close.
	engine.checkSLTP(sdk.Bar{
		High: d("1.15"), Low: d("1.05"), Close: d("1.1"),
	})
	if len(broker.positions) != 1 {
		t.Errorf("position should remain open, got %d", len(broker.positions))
	}
}

func TestEngine_DispatchSignal_CloseAll(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100, ContractSize: d("100000")})
	r1, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"), Magic: 100,
	})
	r2, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideSell, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"), Magic: 100,
	})
	_ = r1
	_ = r2

	engine := &Engine{broker: broker, config: Config{ContractSize: d("100000")}}
	// Set close prices for all positions.
	for _, p := range broker.positions {
		p.ClosePrice = d("1.15")
	}
	engine.dispatchSignal(&sdk.Signal{
		Action: sdk.ActionCloseAll,
		Magic:  100,
	}, sdk.Bar{Close: d("1.15")})
	if len(broker.positions) != 0 {
		t.Errorf("CloseAll should close all positions, got %d", len(broker.positions))
	}
}

func TestEngine_DispatchSignal_CancelAll(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderLimit,
		Volume: d("0.1"), Price: d("1.0"), Magic: 100,
	})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideSell, Type: sdk.OrderStop,
		Volume: d("0.1"), Price: d("1.5"), Magic: 100,
	})

	engine := &Engine{broker: broker, config: Config{}}
	engine.dispatchSignal(&sdk.Signal{
		Action: sdk.ActionCancelAll,
		Magic:  100,
	}, sdk.Bar{})
	if len(broker.pending) != 0 {
		t.Errorf("CancelAll should cancel all pending orders, got %d", len(broker.pending))
	}
}

func TestEngine_DispatchSignal_BuySell(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	engine := &Engine{broker: broker, config: Config{}}

	engine.dispatchSignal(&sdk.Signal{
		Action: sdk.ActionBuy,
		Symbol: "EURUSD",
		Volume: d("0.1"),
	}, sdk.Bar{Close: d("1.1")})
	if len(broker.positions) != 1 {
		t.Errorf("Buy should open 1 position, got %d", len(broker.positions))
	}

	engine.dispatchSignal(&sdk.Signal{
		Action: sdk.ActionSell,
		Symbol: "EURUSD",
		Volume: d("0.1"),
	}, sdk.Bar{Close: d("1.1")})
	if len(broker.positions) != 2 {
		t.Errorf("Sell should open another position, got %d", len(broker.positions))
	}
}

func TestEngine_DispatchSignal_LimitStop(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	engine := &Engine{broker: broker, config: Config{}}

	engine.dispatchSignal(&sdk.Signal{
		Action: sdk.ActionBuyLimit,
		Symbol: "EURUSD",
		Volume: d("0.1"),
		Price:  d("1.0"),
	}, sdk.Bar{Close: d("1.1")})
	if len(broker.pending) != 1 {
		t.Errorf("BuyLimit should create 1 pending, got %d", len(broker.pending))
	}
	if broker.pending[0].OrderType != sdk.OrderLimit {
		t.Errorf("OrderType = %v, want Limit", broker.pending[0].OrderType)
	}

	engine.dispatchSignal(&sdk.Signal{
		Action: sdk.ActionSellStop,
		Symbol: "EURUSD",
		Volume: d("0.1"),
		Price:  d("1.5"),
	}, sdk.Bar{Close: d("1.1")})
	if len(broker.pending) != 2 {
		t.Errorf("SellStop should create another pending, got %d", len(broker.pending))
	}
	if broker.pending[1].OrderType != sdk.OrderStop {
		t.Errorf("OrderType = %v, want Stop", broker.pending[1].OrderType)
	}
}

func TestEngine_DispatchSignal_Close(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100, ContractSize: d("100000")})
	res, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"),
	})
	for _, p := range broker.positions {
		if p.Ticket == res.Ticket {
			p.ClosePrice = d("1.15")
		}
	}

	engine := &Engine{broker: broker, config: Config{ContractSize: d("100000")}}
	engine.dispatchSignal(&sdk.Signal{
		Action:      sdk.ActionClose,
		OrderTicket: res.Ticket,
	}, sdk.Bar{Close: d("1.15")})
	if len(broker.positions) != 0 {
		t.Errorf("Close should close the position, got %d", len(broker.positions))
	}
}

func TestEngine_DispatchSignal_Cancel(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000"), Leverage: 100})
	res, _ := broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderLimit,
		Volume: d("0.1"), Price: d("1.0"),
	})

	engine := &Engine{broker: broker, config: Config{}}
	engine.dispatchSignal(&sdk.Signal{
		Action:      sdk.ActionCancel,
		OrderTicket: res.Ticket,
	}, sdk.Bar{})
	if len(broker.pending) != 0 {
		t.Errorf("Cancel should delete the pending order, got %d", len(broker.pending))
	}
}

func TestEngine_ComputeEquity(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: d("100000"),
		Leverage:       100,
		ContractSize:   d("100000"),
	})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"),
	})

	engine := &Engine{broker: broker, config: Config{ContractSize: d("100000")}}
	equity := engine.computeEquity(sdk.Bar{Close: d("1.2")})
	// Floating profit: (1.2 - 1.1) * 0.1 * 100000 = 1000
	if !equity.GreaterThan(d("100000")) {
		t.Errorf("equity should be > 100000 with floating profit, got %s", equity)
	}
}

func TestEngine_ComputeEquity_SellPosition(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: d("100000"),
		Leverage:       100,
		ContractSize:   d("100000"),
	})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideSell, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"),
	})

	engine := &Engine{broker: broker, config: Config{ContractSize: d("100000")}}
	// Price drops → sell position profits.
	equity := engine.computeEquity(sdk.Bar{Close: d("1.0")})
	if !equity.GreaterThan(d("100000")) {
		t.Errorf("equity should be > 100000 for profitable sell, got %s", equity)
	}
}

func TestEngine_ComputeEquity_DefaultContractSize(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: d("100000"),
		Leverage:       100,
	})
	broker.OrderSend(sdk.OrderRequest{
		Symbol: "EURUSD", Side: sdk.SideBuy, Type: sdk.OrderMarket,
		Volume: d("0.1"), Price: d("1.1"),
	})

	engine := &Engine{broker: broker, config: Config{}}
	equity := engine.computeEquity(sdk.Bar{Close: d("1.2")})
	if !equity.GreaterThan(d("100000")) {
		t.Errorf("equity should use default contract size, got %s", equity)
	}
}

func TestEngine_AdvanceExtraBars(t *testing.T) {
	engine := &Engine{config: Config{}}
	btCtx := &backtestContext{
		extraBars: map[string][]sdk.Bar{
			"GBPUSD": {
				{Close: d("1.3"), Timestamp: 1000},
				{Close: d("1.4"), Timestamp: 2000},
				{Close: d("1.5"), Timestamp: 3000},
			},
		},
		extraBarIndex: map[string]int{"GBPUSD": 0},
	}
	engine.advanceExtraBars(btCtx, 2500)
	if btCtx.extraBarIndex["GBPUSD"] != 1 {
		t.Errorf("extraBarIndex[GBPUSD] = %d, want 1", btCtx.extraBarIndex["GBPUSD"])
	}
}

// --- backtestContext tests ---

func TestBacktestContext_ParamMethods(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000")})
	ctx := &backtestContext{
		broker: broker,
		symbol: "EURUSD",
		tf:     "M5",
		params: map[string]string{
			"lot":    "0.5",
			"period": "14",
			"name":   "test",
			"flag":   "true",
			"num":    "1",
		},
	}

	if v := ctx.Param("lot", "default"); v != "0.5" {
		t.Errorf("Param(lot) = %v, want 0.5", v)
	}
	if v := ctx.Param("missing", "default"); v != "default" {
		t.Errorf("Param(missing) = %v, want default", v)
	}

	if !ctx.ParamDecimal("lot", decimal.Zero).Equal(d("0.5")) {
		t.Errorf("ParamDecimal(lot) mismatch")
	}
	if !ctx.ParamDecimal("missing", d("1.0")).Equal(d("1.0")) {
		t.Errorf("ParamDecimal(missing) mismatch")
	}

	if ctx.ParamInt("period", 7) != 14 {
		t.Errorf("ParamInt(period) = %d, want 14", ctx.ParamInt("period", 7))
	}
	if ctx.ParamInt("missing", 7) != 7 {
		t.Errorf("ParamInt(missing) = %d, want 7", ctx.ParamInt("missing", 7))
	}

	if ctx.ParamString("name", "default") != "test" {
		t.Errorf("ParamString(name) mismatch")
	}
	if ctx.ParamString("missing", "default") != "default" {
		t.Errorf("ParamString(missing) mismatch")
	}

	if !ctx.ParamBool("flag", false) {
		t.Errorf("ParamBool(flag) = false, want true")
	}
	if !ctx.ParamBool("num", false) {
		t.Errorf("ParamBool(num) = false, want true")
	}
	if ctx.ParamBool("missing", false) {
		t.Errorf("ParamBool(missing) = true, want false")
	}
	if !ctx.ParamBool("missing", true) {
		t.Errorf("ParamBool(missing, true) = false, want true")
	}
}

func TestBacktestContext_ParamInvalidValues(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000")})
	ctx := &backtestContext{
		broker: broker,
		params: map[string]string{"bad_dec": "xyz", "bad_int": "abc"},
	}
	if !ctx.ParamDecimal("bad_dec", d("2.0")).Equal(d("2.0")) {
		t.Errorf("ParamDecimal with invalid value should return default")
	}
	if ctx.ParamInt("bad_int", 5) != 5 {
		t.Errorf("ParamInt with invalid value should return default")
	}
}

func TestBacktestContext_MarketData(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: d("100000"),
		Spread:         d("0.0002"),
	})
	ctx := &backtestContext{
		broker:      broker,
		symbol:      "EURUSD",
		tf:          "M5",
		currentBar:  sdk.Bar{Close: d("1.1"), Timestamp: 1000},
		point:       d("0.00001"),
		digits:      5,
	}

	if !ctx.Point().Equal(d("0.00001")) {
		t.Errorf("Point() = %s, want 0.00001", ctx.Point())
	}
	if !ctx.Pip().Equal(d("0.0001")) {
		t.Errorf("Pip() = %s, want 0.0001", ctx.Pip())
	}
	if ctx.Digits() != 5 {
		t.Errorf("Digits() = %d, want 5", ctx.Digits())
	}
	if !ctx.Ask().Equal(d("1.1002")) {
		t.Errorf("Ask() = %s, want 1.1002", ctx.Ask())
	}
	if !ctx.Bid().Equal(d("1.1")) {
		t.Errorf("Bid() = %s, want 1.1", ctx.Bid())
	}
	if !ctx.Spread().Equal(d("0.0002")) {
		t.Errorf("Spread() = %s, want 0.0002", ctx.Spread())
	}
	if ctx.ServerTime() != 1000 {
		t.Errorf("ServerTime() = %d, want 1000", ctx.ServerTime())
	}
}

func TestBacktestContext_DefaultPointDigits(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("100000")})
	ctx := &backtestContext{
		broker: broker,
		// point and digits unset — should return defaults
	}
	if !ctx.Point().Equal(d("0.00001")) {
		t.Errorf("default Point() = %s, want 0.00001", ctx.Point())
	}
	if ctx.Digits() != 5 {
		t.Errorf("default Digits() = %d, want 5", ctx.Digits())
	}
}

func TestBacktestContext_SpreadFallbackToSlippage(t *testing.T) {
	broker := NewSimBroker(Config{
		InitialCapital: d("100000"),
		Slippage:       d("0.0005"),
		// Spread not set
	})
	ctx := &backtestContext{
		broker:     broker,
		currentBar: sdk.Bar{Close: d("1.1")},
	}
	if !ctx.Spread().Equal(d("0.0005")) {
		t.Errorf("Spread fallback = %s, want 0.0005", ctx.Spread())
	}
	if !ctx.Ask().Equal(d("1.1005")) {
		t.Errorf("Ask with slippage fallback = %s, want 1.1005", ctx.Ask())
	}
}

func TestBacktestContext_BarsAndBarsTF(t *testing.T) {
	ctx := &backtestContext{
		symbol:   "EURUSD",
		tf:       "M5",
		bars:     []sdk.Bar{{Close: d("1.0"), Timestamp: 1000}, {Close: d("1.1"), Timestamp: 2000}},
		barIndex: 1,
	}
	bs := ctx.Bars()
	if bs.Len() != 2 {
		t.Errorf("Bars().Len() = %d, want 2", bs.Len())
	}

	// BarsTF with same tf returns Bars().
	bs2 := ctx.BarsTF("M5")
	if bs2.Len() != 2 {
		t.Errorf("BarsTF(M5).Len() = %d, want 2", bs2.Len())
	}

	// BarsTF with different tf aggregates.
	bs3 := ctx.BarsTF("H1")
	if bs3.Len() < 1 {
		t.Errorf("BarsTF(H1).Len() = %d, should be >= 1", bs3.Len())
	}
}

func TestBacktestContext_BarsForSymbol(t *testing.T) {
	ctx := &backtestContext{
		symbol:   "EURUSD",
		tf:       "M5",
		bars:     []sdk.Bar{{Close: d("1.0"), Timestamp: 1000}},
		barIndex: 0,
		extraBars: map[string][]sdk.Bar{
			"GBPUSD": {{Close: d("1.3"), Timestamp: 1000}},
		},
		extraBarIndex: map[string]int{"GBPUSD": 0},
	}

	// Primary symbol returns Bars().
	bs := ctx.BarsForSymbol("EURUSD", "M5")
	if bs.Len() != 1 {
		t.Errorf("BarsForSymbol(EURUSD, M5) = %d, want 1", bs.Len())
	}

	// Extra symbol returns extra bars.
	bs2 := ctx.BarsForSymbol("GBPUSD", "M5")
	if bs2.Len() != 1 {
		t.Errorf("BarsForSymbol(GBPUSD, M5) = %d, want 1", bs2.Len())
	}

	// Unknown symbol returns empty.
	bs3 := ctx.BarsForSymbol("UNKNOWN", "")
	if bs3.Len() != 0 {
		t.Errorf("BarsForSymbol(UNKNOWN) = %d, want 0", bs3.Len())
	}
}

func TestBacktestContext_BarsForSymbol_NoExtraBars(t *testing.T) {
	ctx := &backtestContext{
		symbol:   "EURUSD",
		tf:       "M5",
		bars:     []sdk.Bar{{Close: d("1.0")}},
		barIndex: 0,
		// extraBars is nil
	}
	bs := ctx.BarsForSymbol("GBPUSD", "")
	if bs.Len() != 0 {
		t.Errorf("BarsForSymbol with nil extraBars = %d, want 0", bs.Len())
	}
}

func TestBacktestContext_BarsForSymbol_EmptySymbol(t *testing.T) {
	ctx := &backtestContext{
		symbol:   "EURUSD",
		tf:       "M5",
		bars:     []sdk.Bar{{Close: d("1.0")}},
		barIndex: 0,
	}
	bs := ctx.BarsForSymbol("", "")
	if bs.Len() != 1 {
		t.Errorf("BarsForSymbol(\"\", \"\") should return primary bars, got %d", bs.Len())
	}
}

func TestBacktestContext_AccountAndMode(t *testing.T) {
	broker := NewSimBroker(Config{InitialCapital: d("50000"), Leverage: 100})
	ctx := &backtestContext{broker: broker}
	info := ctx.Account()
	if !info.Balance.Equal(d("50000")) {
		t.Errorf("Account().Balance = %s, want 50000", info.Balance)
	}
	if ctx.Mode() != sdk.ModeHedging {
		t.Errorf("Mode() = %v, want hedging", ctx.Mode())
	}
}

func TestBacktestContext_GoContext(t *testing.T) {
	ctx := &backtestContext{}
	if ctx.GoContext() == nil {
		t.Error("GoContext() should not be nil")
	}
}
