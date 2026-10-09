package mt5

import (
	"context"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
