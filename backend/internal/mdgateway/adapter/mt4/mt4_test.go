package mt4

import (
	"context"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt4"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	t.Parallel()
	cfg := mdtick.AccountConfig{AccountID: "acct-1", Platform: "mt4"}
	gw := New(cfg, zap.NewNop())
	if gw == nil {
		t.Fatal("New returned nil")
	}
	if gw.Platform() != "mt4" {
		t.Errorf("Platform() = %q, want %q", gw.Platform(), "mt4")
	}
	if gw.AccountID() != "acct-1" {
		t.Errorf("AccountID() = %q, want %q", gw.AccountID(), "acct-1")
	}
}

func TestHealthCheck_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	if err := gw.HealthCheck(context.Background()); err == nil {
		t.Error("HealthCheck should fail when not connected")
	}
}

func TestSessionID_Empty(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	if gw.SessionID() != "" {
		t.Errorf("SessionID() = %q, want empty", gw.SessionID())
	}
}

func TestMT4Client_Nil(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	if gw.MT4Client() != nil {
		t.Error("MT4Client() should be nil when not connected")
	}
}

func TestStrToInt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"digits only", "12345", 12345},
		{"with letters", "abc999def", 999},
		{"empty", "", 0},
		{"no digits", "abc", 0},
		{"mixed", "user42", 42},
		{"zero", "0", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strToInt(tt.input)
			if got != tt.want {
				t.Errorf("strToInt(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestMinDuration(t *testing.T) {
	t.Parallel()
	if got := minDuration(1*time.Second, 5*time.Minute); got != 1*time.Second {
		t.Errorf("minDuration(1s, 5m) = %v, want 1s", got)
	}
	if got := minDuration(10*time.Second, 2*time.Second); got != 2*time.Second {
		t.Errorf("minDuration(10s, 2s) = %v, want 2s", got)
	}
	if got := minDuration(1*time.Second, 1*time.Second); got != 1*time.Second {
		t.Errorf("minDuration(1s, 1s) = %v, want 1s", got)
	}
}

func TestPlaceOrder_Stub(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.PlaceOrder(context.Background(), nil)
	if err == nil {
		t.Error("PlaceOrder should return error")
	}
}

func TestCloseOrder_Stub(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.CloseOrder(context.Background(), 0, decimal.Decimal{})
	if err == nil {
		t.Error("CloseOrder should return error")
	}
}

func TestModifyOrder_Stub(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.ModifyOrder(context.Background(), 0, decimal.Decimal{}, decimal.Decimal{}, decimal.Decimal{})
	if err == nil {
		t.Error("ModifyOrder should return error")
	}
}

func TestFetchSymbolParams_Stub(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.FetchSymbolParams(context.Background(), nil)
	if err == nil {
		t.Error("FetchSymbolParams should return error")
	}
}

func TestSubscribeOrderEvents_Stub(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.SubscribeOrderEvents(context.Background(), nil)
	if err == nil {
		t.Error("SubscribeOrderEvents should return error")
	}
}

func TestFetchOpenedOrders_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.FetchOpenedOrders(context.Background())
	if err == nil {
		t.Error("FetchOpenedOrders should fail when not connected")
	}
}

func TestFetchOrderHistory_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.FetchOrderHistory(context.Background(), time.Now(), time.Now())
	if err == nil {
		t.Error("FetchOrderHistory should fail when not connected")
	}
}

func TestSubscribe_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.Subscribe(context.Background(), nil, nil)
	if err == nil {
		t.Error("Subscribe should fail when not connected")
	}
}

func TestSubscribeProfit_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.SubscribeProfit(context.Background(), nil)
	if err == nil {
		t.Error("SubscribeProfit should fail when not connected")
	}
}

func TestSubscribeOrderUpdate_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.SubscribeOrderUpdate(context.Background(), nil)
	if err == nil {
		t.Error("SubscribeOrderUpdate should fail when not connected")
	}
}

func TestMT4UpdateActionLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		action pb.UpdateAction
		want   string
	}{
		{"position open", pb.UpdateAction_UpdateAction_PositionOpen, "open"},
		{"position close", pb.UpdateAction_UpdateAction_PositionClose, "close"},
		{"position modify", pb.UpdateAction_UpdateAction_PositionModify, "modify"},
		{"pending open", pb.UpdateAction_UpdateAction_PendingOpen, "pending_open"},
		{"pending close", pb.UpdateAction_UpdateAction_PendingClose, "pending_close"},
		{"pending modify", pb.UpdateAction_UpdateAction_PendingModify, "pending_modify"},
		{"pending fill", pb.UpdateAction_UpdateAction_PendingFill, "open"},
		{"balance", pb.UpdateAction_UpdateAction_Balance, "balance"},
		{"credit", pb.UpdateAction_UpdateAction_Credit, "credit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Mt4UpdateActionLabel(tt.action); got != tt.want {
				t.Errorf("Mt4UpdateActionLabel(%v) = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}

func TestMT4OrderOpLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		op   pb.Op
		want string
	}{
		{"buy", pb.Op_Op_Buy, "buy"},
		{"sell", pb.Op_Op_Sell, "sell"},
		{"buy limit", pb.Op_Op_BuyLimit, "buy_limit"},
		{"sell limit", pb.Op_Op_SellLimit, "sell_limit"},
		{"buy stop", pb.Op_Op_BuyStop, "buy_stop"},
		{"sell stop", pb.Op_Op_SellStop, "sell_stop"},
		{"balance", pb.Op_Op_Balance, "balance"},
		{"credit", pb.Op_Op_Credit, "credit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mt4OrderOpLabel(tt.op); got != tt.want {
				t.Errorf("mt4OrderOpLabel(%v) = %q, want %q", tt.op, got, tt.want)
			}
		})
	}
}

func TestMT4PeriodToTimeframe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		period string
		want   pb.Timeframe
		ok     bool
	}{
		{"1m", pb.Timeframe_Timeframe_M1, true},
		{"5m", pb.Timeframe_Timeframe_M5, true},
		{"15m", pb.Timeframe_Timeframe_M15, true},
		{"30m", pb.Timeframe_Timeframe_M30, true},
		{"1h", pb.Timeframe_Timeframe_H1, true},
		{"4h", pb.Timeframe_Timeframe_H4, true},
		{"1d", pb.Timeframe_Timeframe_D1, true},
		{"2m", 0, false},
		{"1w", pb.Timeframe_Timeframe_W1, true},
		{"", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			got, ok := mt4PeriodToTimeframe(tt.period)
			if ok != tt.ok {
				t.Errorf("mt4PeriodToTimeframe(%q) ok=%v, want %v", tt.period, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("mt4PeriodToTimeframe(%q) = %v, want %v", tt.period, got, tt.want)
			}
		})
	}
}
