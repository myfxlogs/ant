package mt5

import (
	"context"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
	pb "alphaforge/mt5"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestStrToUint64(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  uint64
	}{
		{"digits only", "12345", 12345},
		{"with letters", "abc999def", 999},
		{"empty", "", 0},
		{"no digits", "abc", 0},
		{"mixed", "user42", 42},
		{"zero", "0", 0},
		{"max uint64", "18446744073709551615", 18446744073709551615},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strToUint64(tt.input)
			if got != tt.want {
				t.Errorf("strToUint64(%q) = %d, want %d", tt.input, got, tt.want)
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
}

func TestPfloat64(t *testing.T) {
	t.Parallel()
	zero := pfloat64(decimal.Decimal{})
	if zero != nil {
		t.Error("pfloat64(zero) should be nil")
	}
	val := decimal.NewFromFloat(1.2345)
	p := pfloat64(val)
	if p == nil {
		t.Error("pfloat64(non-zero) should not be nil")
		return
	}
	if *p != 1.2345 {
		t.Errorf("pfloat64(non-zero) = %f, want 1.2345", *p)
	}
}

func TestPInt64(t *testing.T) {
	t.Parallel()
	nilPtr := pInt64(0)
	if nilPtr != nil {
		t.Error("pInt64(0) should be nil")
	}
	p := pInt64(42)
	if p == nil {
		t.Error("pInt64(42) should not be nil")
		return
	}
	if *p != 42 {
		t.Errorf("pInt64(42) = %d, want 42", *p)
	}
}

func TestMT5OrderType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		side mthub.Side
		ot   mthub.OrderType
		want pb.OrderType
	}{
		{"buy market", mthub.SideBuy, mthub.OrderMarket, pb.OrderType_OrderType_Buy},
		{"sell market", mthub.SideSell, mthub.OrderMarket, pb.OrderType_OrderType_Sell},
		{"buy limit", mthub.SideBuy, mthub.OrderLimit, pb.OrderType_OrderType_BuyLimit},
		{"sell limit", mthub.SideSell, mthub.OrderLimit, pb.OrderType_OrderType_SellLimit},
		{"buy stop", mthub.SideBuy, mthub.OrderStop, pb.OrderType_OrderType_BuyStop},
		{"sell stop", mthub.SideSell, mthub.OrderStop, pb.OrderType_OrderType_SellStop},
		{"buy stop limit", mthub.SideBuy, mthub.OrderStopLimit, pb.OrderType_OrderType_BuyStopLimit},
		{"sell stop limit", mthub.SideSell, mthub.OrderStopLimit, pb.OrderType_OrderType_SellStopLimit},
		{"default (unknown)", mthub.SideBuy, mthub.OrderType(99), pb.OrderType_OrderType_Buy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mt5OrderType(tt.side, tt.ot); got != tt.want {
				t.Errorf("mt5OrderType(%v, %v) = %v, want %v", tt.side, tt.ot, got, tt.want)
			}
		})
	}
}

func TestMT5UpdateTypeLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tp   pb.UpdateType
		want string
	}{
		{"market open", pb.UpdateType_UpdateType_MarketOpen, "open"},
		{"market close", pb.UpdateType_UpdateType_MarketClose, "close"},
		{"partial close", pb.UpdateType_UpdateType_PartialClose, "close"},
		{"pending open", pb.UpdateType_UpdateType_PendingOpen, "pending_open"},
		{"pending close", pb.UpdateType_UpdateType_PendingClose, "pending_close"},
		{"market modify", pb.UpdateType_UpdateType_MarketModify, "modify"},
		{"pending modify", pb.UpdateType_UpdateType_PendingModify, "modify"},
		{"unknown (default)", pb.UpdateType_UpdateType_Unknown, "unknown"},
		{"started (default)", pb.UpdateType_UpdateType_Started, "unknown"},
		{"expired (default)", pb.UpdateType_UpdateType_Expired, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Mt5UpdateTypeLabel(tt.tp); got != tt.want {
				t.Errorf("Mt5UpdateTypeLabel(%v) = %q, want %q", tt.tp, got, tt.want)
			}
		})
	}
}

func TestMT5OrderTypeLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ot   pb.OrderType
		want string
	}{
		{"buy", pb.OrderType_OrderType_Buy, "buy"},
		{"sell", pb.OrderType_OrderType_Sell, "sell"},
		{"buy limit", pb.OrderType_OrderType_BuyLimit, "buy_limit"},
		{"sell limit", pb.OrderType_OrderType_SellLimit, "sell_limit"},
		{"buy stop", pb.OrderType_OrderType_BuyStop, "buy_stop"},
		{"sell stop", pb.OrderType_OrderType_SellStop, "sell_stop"},
		{"buy stop limit", pb.OrderType_OrderType_BuyStopLimit, "buy_stop_limit"},
		{"sell stop limit", pb.OrderType_OrderType_SellStopLimit, "sell_stop_limit"},
		{"balance", pb.OrderType_OrderType_Balance, "balance"},
		{"credit", pb.OrderType_OrderType_Credit, "credit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mt5OrderTypeLabel(tt.ot); got != tt.want {
				t.Errorf("mt5OrderTypeLabel(%v) = %q, want %q", tt.ot, got, tt.want)
			}
		})
	}
}

func TestMT5PeriodToTimeframe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		period string
		want   int32
	}{
		{"1m", 1},
		{"5m", 5},
		{"15m", 15},
		{"30m", 30},
		{"1h", 60},
		{"4h", 240},
		{"1d", 1440},
		{"1w", 10080},
		{"unknown", 60},
		{"", 60},
	}
	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			got := mt5PeriodToTimeframe(tt.period)
			if got != tt.want {
				t.Errorf("mt5PeriodToTimeframe(%q) = %d, want %d", tt.period, got, tt.want)
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
		{"30m", 1_800_000},
		{"1h", 3_600_000},
		{"4h", 14_400_000},
		{"1d", 86_400_000},
		{"1w", 604_800_000},
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

func FuzzStrToUint64(f *testing.F) {
	f.Add("12345")
	f.Add("abc999def")
	f.Add("")
	f.Add("user42")
	f.Add("18446744073709551615")
	f.Fuzz(func(t *testing.T, s string) {
		v := strToUint64(s)
		_ = v
	})
}

func BenchmarkStrToUint64(b *testing.B) {
	const s = "mt5_login_123456789_abc"
	b.ReportAllocs()
	for b.Loop() {
		strToUint64(s)
	}
}

// T5 (VM-ERR-CODE-COLLAPSE-1): MT5 broker rejections carry the platform
// retcode (100xx namespace) verbatim via *mthub.BrokerRejectError.
// Adversarial: reverting to fmt.Errorf loses errors.As → RED.
