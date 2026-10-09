package mt4

import (
	"context"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
	pb "alphaforge/mt4"

	"go.uber.org/zap"
)

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

func TestPeriodMs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		period string
		want   int64
	}{
		{"1m", 60_000},
		{"5m", 300_000},
		{"15m", 900_000},
		{"1h", 3_600_000},
		{"4h", 14_400_000},
		{"1d", 86_400_000},
		{"unknown", 60_000},
		{"", 60_000},
	}
	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			if got := mdtick.PeriodMs(tt.period); got != tt.want {
				t.Errorf("periodMs(%q) = %d, want %d", tt.period, got, tt.want)
			}
		})
	}
}

func TestSleep_CtxCancelled(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	gw.sleep(ctx, time.Second)
	if time.Since(start) > 100*time.Millisecond {
		t.Error("sleep should return immediately when ctx is cancelled")
	}
}

func TestMt4Op(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		side    mthub.Side
		ot      mthub.OrderType
		want    pb.Op
		wantErr bool
	}{
		{"buy market", mthub.SideBuy, mthub.OrderMarket, pb.Op_Op_Buy, false},
		{"sell market", mthub.SideSell, mthub.OrderMarket, pb.Op_Op_Sell, false},
		{"buy limit", mthub.SideBuy, mthub.OrderLimit, pb.Op_Op_BuyLimit, false},
		{"sell limit", mthub.SideSell, mthub.OrderLimit, pb.Op_Op_SellLimit, false},
		{"buy stop", mthub.SideBuy, mthub.OrderStop, pb.Op_Op_BuyStop, false},
		{"sell stop", mthub.SideSell, mthub.OrderStop, pb.Op_Op_SellStop, false},
		{"buy stop_limit unsupported", mthub.SideBuy, mthub.OrderStopLimit, 0, true},
		{"sell stop_limit unsupported", mthub.SideSell, mthub.OrderStopLimit, 0, true},
		{"unknown type returns error", mthub.SideBuy, mthub.OrderType(99), 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mt4Op(tt.side, tt.ot)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("mt4Op(%v, %v) expected error, got %v", tt.side, tt.ot, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("mt4Op(%v, %v) unexpected error: %v", tt.side, tt.ot, err)
			}
			if got != tt.want {
				t.Errorf("mt4Op(%v, %v) = %v, want %v", tt.side, tt.ot, got, tt.want)
			}
		})
	}
}

// --- PlaceOrder with mock TradingClient ---
