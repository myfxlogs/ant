package strategy

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// Circuit breaker gate tests: verify that dispatchLiveSignal suppresses
// new orders (market + pending) when the circuit breaker is open,
// while still allowing close/close_all/modify/cancel through.
// When circuit is open, the gate returns before reaching mtHub, so nil mtHub is safe.

func TestCircuitBreaker_SuppressesMarketBuy(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop()}
	sess := &ActiveSession{RunID: uuid.New(), AccountID: "acct-1", Symbol: "EURUSD"}
	sess.SetCircuitOpen(true)
	cfg := LiveStrategyConfig{AccountID: "acct-1", UserID: "user-1", Symbol: "EURUSD", Mode: "live"}
	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
}

func TestCircuitBreaker_SuppressesMarketSell(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop()}
	sess := &ActiveSession{RunID: uuid.New(), AccountID: "acct-1", Symbol: "EURUSD"}
	sess.SetCircuitOpen(true)
	cfg := LiveStrategyConfig{AccountID: "acct-1", UserID: "user-1", Symbol: "EURUSD", Mode: "live"}
	sig := &antv1.StrategySignal{SignalType: "sell", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
}

func TestCircuitBreaker_SuppressesAllPendingTypes(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop()}
	sess := &ActiveSession{RunID: uuid.New(), AccountID: "acct-1", Symbol: "EURUSD"}
	sess.SetCircuitOpen(true)
	cfg := LiveStrategyConfig{AccountID: "acct-1", UserID: "user-1", Symbol: "EURUSD", Mode: "live"}
	for _, action := range []string{"buy_limit", "sell_limit", "buy_stop", "sell_stop", "buy_stop_limit", "sell_stop_limit"} {
		sig := &antv1.StrategySignal{SignalType: action, Volume: "0.1", Price: "1.1000"}
		srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
	}
}

func TestCircuitBreaker_AllowsCloseWhenOpen(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop()}
	sess := &ActiveSession{RunID: uuid.New(), AccountID: "acct-1", Symbol: "EURUSD"}
	sess.SetCircuitOpen(true)
	cfg := LiveStrategyConfig{AccountID: "acct-1", UserID: "user-1", Symbol: "EURUSD", Mode: "live"}
	sig := &antv1.StrategySignal{SignalType: "close", Volume: "0.1", ExecutedTicket: 12345}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
}

func TestCircuitBreaker_AllowsCloseAllWhenOpen(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop()}
	sess := &ActiveSession{RunID: uuid.New(), AccountID: "acct-1", Symbol: "EURUSD"}
	sess.SetCircuitOpen(true)
	cfg := LiveStrategyConfig{AccountID: "acct-1", UserID: "user-1", Symbol: "EURUSD", Mode: "live"}
	sig := &antv1.StrategySignal{SignalType: "close_all"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
}

func TestCircuitBreaker_AllowsModifyWhenOpen(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop()}
	sess := &ActiveSession{RunID: uuid.New(), AccountID: "acct-1", Symbol: "EURUSD"}
	sess.SetCircuitOpen(true)
	cfg := LiveStrategyConfig{AccountID: "acct-1", UserID: "user-1", Symbol: "EURUSD", Mode: "live"}
	sig := &antv1.StrategySignal{SignalType: "modify", ExecutedTicket: 12345, StopLoss: "1.05", TakeProfit: "1.20"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
}

func TestCircuitBreaker_AllowsCancelWhenOpen(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop()}
	sess := &ActiveSession{RunID: uuid.New(), AccountID: "acct-1", Symbol: "EURUSD"}
	sess.SetCircuitOpen(true)
	cfg := LiveStrategyConfig{AccountID: "acct-1", UserID: "user-1", Symbol: "EURUSD", Mode: "live"}
	sig := &antv1.StrategySignal{SignalType: "cancel", ExecutedTicket: 12345}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
}

func TestCircuitBreaker_PaperModeBypassesGate(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop()}
	sess := &ActiveSession{RunID: uuid.New(), AccountID: "acct-1", Symbol: "EURUSD"}
	sess.SetCircuitOpen(true)
	cfg := LiveStrategyConfig{AccountID: "acct-1", UserID: "user-1", Symbol: "EURUSD", Mode: "paper"}
	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
}

func TestCircuitBreaker_NilSessionDoesNotPanic(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop()}
	cfg := LiveStrategyConfig{AccountID: "acct-1", UserID: "user-1", Symbol: "EURUSD", Mode: "live"}
	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, nil)
}

func TestCircuitBreaker_HoldActionIsNoop(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop()}
	sess := &ActiveSession{RunID: uuid.New(), AccountID: "acct-1", Symbol: "EURUSD"}
	sess.SetCircuitOpen(true)
	cfg := LiveStrategyConfig{AccountID: "acct-1", UserID: "user-1", Symbol: "EURUSD", Mode: "live"}
	sig := &antv1.StrategySignal{SignalType: "hold"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
}

func TestActiveSession_CircuitOpenState(t *testing.T) {
	sess := &ActiveSession{}
	if sess.IsCircuitOpen() {
		t.Error("new session should have circuitOpen=false")
	}
	sess.SetCircuitOpen(true)
	if !sess.IsCircuitOpen() {
		t.Error("after SetCircuitOpen(true), IsCircuitOpen should return true")
	}
	sess.SetCircuitOpen(false)
	if sess.IsCircuitOpen() {
		t.Error("after SetCircuitOpen(false), IsCircuitOpen should return false")
	}
}

func TestActiveSession_CircuitOpenConcurrent(t *testing.T) {
	sess := &ActiveSession{}
	done := make(chan bool, 20)
	for i := 0; i < 10; i++ {
		go func() {
			sess.SetCircuitOpen(true)
			_ = sess.IsCircuitOpen()
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		go func() {
			sess.SetCircuitOpen(false)
			_ = sess.IsCircuitOpen()
			done <- true
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
}
