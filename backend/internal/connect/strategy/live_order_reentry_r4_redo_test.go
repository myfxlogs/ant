// live_order_reentry_r4_redo_test.go — R4 复审阻断解决测试（2026-08-26）.
package strategy

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mdgateway/adapter/mt4"
	"alphaforge/internal/mdgateway/adapter/mt5"
	"alphaforge/internal/mthub"
	mt4pb "alphaforge/mt4"
	mt5pb "alphaforge/mt5"
)

// TestLIVE_ORDER_REENTRY_1_R4_OpenMutationWithTicket_NoRecovery verifies that
// when an open mutation gets a valid ticket from the broker RPC but confirmation
// times out (outcome unknown), the recovery goroutine is NOT started.
// Adversarial proof: remove the `spec.action != actionOpen` guard -> recovery
// goroutine starts -> barrier transitions out of outcomeUnknown -> RED.
func TestLIVE_ORDER_REENTRY_1_R4_OpenMutationWithTicket_NoRecovery(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			return 77, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return nil, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	conf := confirmationConfig{
		pushWait:              50 * time.Millisecond,
		readAfterWriteTimeout: 500 * time.Millisecond,
		mutationRPCTimeout:    5 * time.Second,
		recoveryDelay:         50 * time.Millisecond,
	}

	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.coordinateMutation(context.Background(), cfg, sess, mutationSpec{
		action:         actionOpen,
		clientID:       "open_77",
		expectedMagic:  strategyMagic(cfg.ScheduleID),
		expectedTicket: 0,
		brokerCall: func(brokerCtx context.Context) (int64, error) {
			return exec.PlaceOrder(brokerCtx, &mthub.OrderRequest{})
		},
		verifyReadAfterWrite: nil,
	}, "buy", sig, conf)

	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("OpenMutationWithTicket: state=%s, want outcome_unknown", state)
	}
	if !sess.IsCircuitOpen() {
		t.Fatal("OpenMutationWithTicket: circuit breaker should be open")
	}

	// 等待 recoveryDelay + readAfterWriteTimeout + 余量，确认 barrier 保持 outcomeUnknown。
	// WaitState(ctx, barrierIdle) 阻塞直到 barrier 变为 barrierIdle（recovery 释放）或 ctx 超时。
	// 正确行为：open mutation 不启动 recovery → ctx 超时 → 返回当前状态 barrierOutcomeUnknown。
	// 错误行为：若 recovery 错误启动 → barrier 变为 barrierIdle → WaitState 提前返回 → 断言失败。
	// 对抗证明：突变 mutation_coordinator.go:266 `if spec.action != actionOpen` → `if true`
	// → recovery 启动 → barrier 变为 barrierIdle → finalState != barrierOutcomeUnknown → RED。
	noRecoveryCtx, noRecoveryCancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	finalState := sess.barrier.WaitState(noRecoveryCtx, barrierIdle)
	noRecoveryCancel()
	if finalState != barrierOutcomeUnknown {
		t.Fatalf("OpenMutationWithTicket: recovery ran unexpectedly, state=%s (should stay outcome_unknown for open)", finalState)
	}
	if !sess.IsCircuitOpen() {
		t.Fatal("OpenMutationWithTicket: circuit breaker should stay open (no recovery)")
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_RealParse_MT4 verifies that
// the real MT4 adapter parseMt4OrderUpdate correctly maps a PendingOpen
// UpdateAction to the "pending_open" label, and that this label flows through
// the barrier confirmation logic.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_RealParse_MT4(t *testing.T) {
	summary := &mt4pb.OrderUpdateSummary{
		Update: &mt4pb.OrderUpdateEventArgs{
			Action: mt4pb.UpdateAction_UpdateAction_PendingOpen,
			Order: &mt4pb.Order{
				Ticket:      42,
				Symbol:      "EURUSD",
				MagicNumber: 12345,
				Lots:        0.1,
				OpenPrice:   1.0850,
				Type:        mt4pb.Op_Op_Buy,
			},
		},
	}

	update := mt4.ParseMt4OrderUpdateForTest(summary, "acct-1")
	if update == nil {
		t.Fatal("ParseMt4OrderUpdateForTest returned nil")
	}
	if update.UpdateType != "pending_open" {
		t.Fatalf("RealParse_MT4: UpdateType=%q, want %q", update.UpdateType, "pending_open")
	}
	if update.UpdateTicket != 42 {
		t.Fatalf("RealParse_MT4: UpdateTicket=%d, want 42", update.UpdateTicket)
	}
	if update.UpdateMagic != 12345 {
		t.Fatalf("RealParse_MT4: UpdateMagic=%d, want 12345", update.UpdateMagic)
	}

	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, update.UpdateType)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("RealParse_MT4: barrier state=%s, want confirmed (pending_open label should confirm open action)", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_RealParse_MT5 verifies that
// the real MT5 adapter parseMt5OrderUpdate correctly maps a MarketOpen
// UpdateType to the "open" label, and that this label flows through the
// barrier confirmation logic.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_RealParse_MT5(t *testing.T) {
	summary := &mt5pb.OrderUpdateSummary{
		Update: &mt5pb.OrderUpdate{
			Type: mt5pb.UpdateType_UpdateType_MarketOpen,
			Order: &mt5pb.Order{
				Ticket:    88,
				Symbol:    "EURUSD",
				ExpertId:  12345,
				Lots:      0.1,
				OpenPrice: 1.0850,
			},
		},
	}

	update := mt5.ParseMt5OrderUpdateForTest(summary, "acct-1")
	if update == nil {
		t.Fatal("ParseMt5OrderUpdateForTest returned nil")
	}
	if update.UpdateType != "open" {
		t.Fatalf("RealParse_MT5: UpdateType=%q, want %q", update.UpdateType, "open")
	}
	if update.UpdateTicket != 88 {
		t.Fatalf("RealParse_MT5: UpdateTicket=%d, want 88", update.UpdateTicket)
	}

	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(88)
	b.NotifyConfirmationEvent(88, 12345, update.UpdateType)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("RealParse_MT5: barrier state=%s, want confirmed (open label should confirm open action)", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_RealParse_FullPath_MT4
// verifies the complete pipeline: real adapter parseMt4OrderUpdate ->
// publishOrderUpdate (real broker) -> confirmation listener -> barrier.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_RealParse_FullPath_MT4(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			return 42, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{{Ticket: 42, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, broker := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	summary := &mt4pb.OrderUpdateSummary{
		Update: &mt4pb.OrderUpdateEventArgs{
			Action: mt4pb.UpdateAction_UpdateAction_PendingOpen,
			Order: &mt4pb.Order{
				Ticket:      42,
				Symbol:      "EURUSD",
				MagicNumber: strategyMagic(cfg.ScheduleID),
				Lots:        0.1,
				OpenPrice:   1.0850,
				Type:        mt4pb.Op_Op_Buy,
			},
		},
	}
	update := mt4.ParseMt4OrderUpdateForTest(summary, cfg.AccountID)

	go func() {
		// 确定性等待 barrier 进入 submitting（R4 S3: 用 cond.Wait 同步，禁止轮询睡眠）。
		// WaitState 阻塞直到 barrier 变为 barrierSubmitting 或 ctx 超时。
		// dispatchLiveSignal 是同步阻塞调用，主 goroutine 在其内部会驱动 barrier
		// 状态变化（acquire → submitting），WaitState 的 cond.Wait() 会被唤醒。
		waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
		sess.barrier.WaitState(waitCtx, barrierSubmitting)
		waitCancel()
		publishOrderUpdate(broker, cfg.AccountID, 42, strategyMagic(cfg.ScheduleID), update.UpdateType)
	}()

	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("RealParse_FullPath_MT4: barrier state=%s, want idle (confirmed+released)", state)
	}
	if got := exec.placeCount.Load(); got != 1 {
		t.Fatalf("RealParse_FullPath_MT4: PlaceOrder called %d times, want 1", got)
	}
}
