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
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNew(t *testing.T) {
	t.Parallel()
	cfg := mdtick.AccountConfig{AccountID: "acct-5", Platform: "mt5"}
	gw := New(cfg, zap.NewNop())
	if gw == nil {
		t.Fatal("New returned nil")
	}
	if gw.Platform() != "mt5" {
		t.Errorf("Platform() = %q, want %q", gw.Platform(), "mt5")
	}
	if gw.AccountID() != "acct-5" {
		t.Errorf("AccountID() = %q, want %q", gw.AccountID(), "acct-5")
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

func TestMT5Client_Nil(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	if gw.MT5Client() != nil {
		t.Error("MT5Client() should be nil when not connected")
	}
}

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

func TestPlaceOrder_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.PlaceOrder(context.Background(), &mthub.OrderRequest{})
	if err == nil {
		t.Error("PlaceOrder should fail when not connected")
	}
}

func TestCloseOrder_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.CloseOrder(context.Background(), 0, decimal.Decimal{})
	if err == nil {
		t.Error("CloseOrder should fail when not connected")
	}
}

func TestModifyOrder_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	err := gw.ModifyOrder(context.Background(), 0, decimal.Decimal{}, decimal.Decimal{}, decimal.Decimal{})
	if err == nil {
		t.Error("ModifyOrder should fail when not connected")
	}
}

func TestFetchSymbolParams_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.FetchSymbolParams(context.Background(), nil)
	if err == nil {
		t.Error("FetchSymbolParams should fail when not connected")
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

func TestFetchOrderHistory_Stub(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	// FetchOrderHistory now checks connection state; unconnected gateway returns error.
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

func TestConvertMT5Bars_Empty(t *testing.T) {
	t.Parallel()
	bars := convertMT5Bars(nil, "acct-5", "1h")
	if len(bars) != 0 {
		t.Errorf("expected 0 bars from nil, got %d", len(bars))
	}
	bars = convertMT5Bars([]*pb.Bar{}, "acct-5", "1h")
	if len(bars) != 0 {
		t.Errorf("expected 0 bars from empty, got %d", len(bars))
	}
}

func TestConvertMT5Bars_WithData(t *testing.T) {
	t.Parallel()
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	pbBars := []*pb.Bar{
		{Time: ts, OpenPrice: 1.1000, HighPrice: 1.1050, LowPrice: 1.0990, ClosePrice: 1.1020, Volume: 100, TickVolume: 50},
		{Time: ts, OpenPrice: 1.1020, HighPrice: 1.1080, LowPrice: 1.1010, ClosePrice: 1.1060, Volume: 200, TickVolume: 80},
	}
	bars := convertMT5Bars(pbBars, "acct-5", "1h")
	if len(bars) != 2 {
		t.Fatalf("expected 2 bars, got %d", len(bars))
	}
	if bars[0].AccountID != "acct-5" {
		t.Errorf("AccountID = %q, want acct-5", bars[0].AccountID)
	}
	if bars[0].Period != "1h" {
		t.Errorf("Period = %q, want 1h", bars[0].Period)
	}
	if !bars[0].Open.Equal(decimal.NewFromFloat(1.1000)) {
		t.Errorf("Open = %s, want 1.1000", bars[0].Open)
	}
	if bars[0].Volume != 100 {
		t.Errorf("Volume = %f, want 100", bars[0].Volume)
	}
	if bars[0].TickCount != 50 {
		t.Errorf("TickCount = %d, want 50", bars[0].TickCount)
	}
}

// TestBARALIGN_ConvertMT5Bars_SubSecondAlignment verifies BAR-ALIGN:
// mtapi bar Time with sub-second precision must be floored to period boundary.
//
// Adversarial proof: Remove the `openMs -= openMs % pm` alignment line →
// openMs retains sub-second offset → open_ts % periodMs != 0 (RED).
func TestBARALIGN_ConvertMT5Bars_SubSecondAlignment(t *testing.T) {
	t.Parallel()
	// 2026-01-15 10:00:00.385 UTC → UnixMilli = 1784978400385 (not 5m-aligned, offset 385ms)
	ts := timestamppb.New(time.Date(2026, 1, 15, 10, 0, 0, 385_000_000, time.UTC))
	pbBars := []*pb.Bar{
		{Time: ts, OpenPrice: 1.1000, HighPrice: 1.1050, LowPrice: 1.0990, ClosePrice: 1.1020, Volume: 100, TickVolume: 50},
	}
	bars := convertMT5Bars(pbBars, "acct-5", "5m")
	if len(bars) != 1 {
		t.Fatalf("expected 1 bar, got %d", len(bars))
	}
	pm := mdtick.PeriodMs("5m") // 300000
	if bars[0].OpenTsUnixMs%pm != 0 {
		t.Fatalf("BAR-ALIGN: open_ts %d not aligned to 5m boundary (remainder %d) — RED: sub-second offset not floored",
			bars[0].OpenTsUnixMs, bars[0].OpenTsUnixMs%pm)
	}
	wantOpen := int64(1768471200000) // 2026-01-15 10:00:00.000 UTC, floored to 5m
	if bars[0].OpenTsUnixMs != wantOpen {
		t.Fatalf("open_ts = %d, want %d (floored to 5m)", bars[0].OpenTsUnixMs, wantOpen)
	}
	if bars[0].CloseTsUnixMs != wantOpen+pm {
		t.Fatalf("close_ts = %d, want %d (open+periodMs)", bars[0].CloseTsUnixMs, wantOpen+pm)
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

func TestGetPriceHistory_NotConnected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	_, err := gw.GetPriceHistory(context.Background(), "acct-5", "EURUSD", "1h", 0, 3600_000)
	if err == nil {
		t.Error("GetPriceHistory should fail when not connected")
	}
}

func TestDisconnect_NilConn(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	if err := gw.Disconnect(context.Background()); err != nil {
		t.Errorf("Disconnect on nil conn should not error, got %v", err)
	}
}

func TestDisconnect_FullState(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{Login: "123", Password: "p", BrokerHost: "h"}, zap.NewNop())
	gw.client = &mockMT5Client{}
	gw.connCli = &mockMT5ConnCli{}
	gw.streamCli = &mt5MockStreamsClient{}
	gw.subCli = &mockMT5SubCli{}
	gw.tradingCli = &mockTradingClient{}
	gw.qhCli = &mockQHClient{}
	gw.sessionID = "sid"
	ctx1, c1 := context.WithCancel(context.Background())
	ctx2, c2 := context.WithCancel(context.Background())
	ctx3, c3 := context.WithCancel(context.Background())
	gw.cancelSub = c1
	gw.cancelProfitSub = c2
	gw.cancelOrderUpdateSub = c3
	if err := gw.Disconnect(context.Background()); err != nil {
		t.Errorf("Disconnect should not error: %v", err)
	}
	checkCancelled := func(name string, ctx context.Context) {
		select {
		case <-ctx.Done():
		default:
			t.Errorf("%s should be cancelled", name)
		}
	}
	checkCancelled("cancelSub", ctx1)
	checkCancelled("cancelProfitSub", ctx2)
	checkCancelled("cancelOrderUpdateSub", ctx3)
	if gw.client != nil {
		t.Error("client should be nil after Disconnect")
	}
	if gw.sessionID != "" {
		t.Error("sessionID should be empty after Disconnect")
	}
}

func TestEnsureConnected_AlreadySet(t *testing.T) {
	t.Parallel()
	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{}, zap.NewNop())
	gw.conn = &cc
	bo := 100 * time.Millisecond
	if err := gw.ensureConnected(context.Background(), &bo, time.Second); err != nil {
		t.Errorf("ensureConnected should succeed when conn is set: %v", err)
	}
}
