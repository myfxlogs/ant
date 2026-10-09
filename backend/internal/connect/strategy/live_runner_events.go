package strategy

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go"
)

func (s *StrategyExecutionServer) handleBar(
	ctx context.Context, cfg LiveStrategyConfig,
	bar *mthub.BarUpdate, bars *[]liveBar,
	session *Session, firstBar *bool, activeSess *ActiveSession,
	extraBars map[string][]liveBar,
) {
	appendDedupBar(bars, liveBar{
		open:     bar.Open.String(),
		high:     bar.High.String(),
		low:      bar.Low.String(),
		close:    bar.Close.String(),
		volume:   strconv.FormatFloat(bar.Volume, 'f', -1, 64),
		openTime: bar.OpenTime,
	})

	if cfg.ShadowVerifier != nil {
		cfg.ShadowVerifier.RecordBar(sdk.Bar{
			Open:      bar.Open,
			High:      bar.High,
			Low:       bar.Low,
			Close:     bar.Close,
			Volume:    int64(bar.Volume),
			Timestamp: bar.OpenTime,
		})
	}

	lctx, err := s.buildLiveContext(ctx, cfg, *bars, extraBars)
	if err != nil {
		s.log.Warn("LiveStrategyRunner: bar skipped", zap.Error(err))
		if activeSess != nil {
			activeSess.RecordError(err.Error())
		}
		return
	}

	if activeSess != nil && activeSess.diag != nil {
		activeSess.diag.RecordWindow(len(*bars))
	}

	req := &antv1.ExecuteLiveRequest{
		StrategyCode: cfg.Code,
		StrategyId:   cfg.StrategyID,
		RequestType:  antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:   lctx,
	}

	var resp *antv1.ExecuteLiveResponse
	if *firstBar {
		vmSess, vmErr := s.initVMSession(ctx, cfg, activeSess)
		if vmErr != nil {
			return
		}
		*session = vmSess
		s.wireSyncDispatch(ctx, cfg, bar, *session, activeSess)
		resp, err = (*session).Start(ctx, req)
		*firstBar = false
	} else {
		if *session == nil {
			s.log.Error("LiveStrategyRunner: session lost before bar event")
			return
		}
		s.wireSyncDispatch(ctx, cfg, bar, *session, activeSess)
		resp, err = (*session).SendEvent(ctx, req)
	}
	if err != nil {
		s.log.Error("LiveStrategyRunner: bar request failed", zap.Error(err))
		if activeSess != nil {
			activeSess.RecordError(err.Error())
		}
		if *session != nil {
			_ = (*session).Close()
		}
		*session = nil
		*firstBar = true
		return
	}
	s.dispatchResponse(ctx, cfg, bar, resp, activeSess, sessionSyncDispatched(*session))
}

// wireSyncDispatch installs the synchronous broker-mutation dispatcher on
// live VM sessions before each event (VM-LIVE-SYNC-DISPATCH-1, R1). The
// closure captures this event's bar so open-order ClientIDs stay
// bar-scoped. Signal-mode trade builtins then execute mutations inside
// the VM event and receive real broker outcomes (ticket/rejection).
func (s *StrategyExecutionServer) wireSyncDispatch(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, sess Session, activeSess *ActiveSession) {
	if cfg.Mode != modeLive {
		return
	}
	vmSess, ok := sess.(*VMLiveSession)
	if !ok || vmSess.strategy == nil {
		return
	}
	vmSess.SetSyncDispatcher(func(sig *sdk.Signal) (int64, error) {
		return s.dispatchSignalSync(ctx, cfg, bar, sig, activeSess, vmSess.runner)
	})
}

// sessionSyncDispatched reports whether the session's signals were already
// executed synchronously inside the VM event — dispatchResponse must then
// skip the async broker dispatch (VM-LIVE-SYNC-DISPATCH-1: no double-submit).
func sessionSyncDispatched(sess Session) bool {
	if vmSess, ok := sess.(*VMLiveSession); ok {
		return vmSess.SyncDispatched()
	}
	return false
}

func (s *StrategyExecutionServer) initVMSession(ctx context.Context, cfg LiveStrategyConfig, activeSess *ActiveSession) (Session, error) {
	cachedBytecode := s.cachedBytecodeFor(ctx, cfg.StrategyID)
	vmSess, err := s.compileVMLive(cfg, cachedBytecode, activeSess)
	if err != nil {
		return nil, err
	}
	if activeSess != nil {
		vmSess.SetDiag(activeSess.diag)
	}
	s.saveVMBytecodeCache(ctx, cfg, vmSess)
	s.attachHistoryProvider(cfg, vmSess)
	return vmSess, nil
}

// cachedBytecodeFor loads the previously persisted bytecode for the strategy,
// so compile can reuse it instead of recompiling from source.
func (s *StrategyExecutionServer) cachedBytecodeFor(ctx context.Context, strategyID string) []byte {
	if strategyID == "" || s.importedRepo == nil {
		return nil
	}
	sid, parseErr := uuid.Parse(strategyID)
	if parseErr != nil {
		return nil
	}
	bc, _ := s.importedRepo.GetBytecode(ctx, sid)
	return bc
}

// compileVMLive builds the VM session (Python or MQL by code shape) and
// records the compile failure on the active session before returning it.
func (s *StrategyExecutionServer) compileVMLive(cfg LiveStrategyConfig, cachedBytecode []byte, activeSess *ActiveSession) (*VMLiveSession, error) {
	if sdk.IsPython(cfg.Code) {
		vmSess, vmErr := NewPythonVMLiveSessionCached(cfg.Code, cachedBytecode)
		if vmErr != nil {
			s.log.Error("LiveStrategyRunner: compile Python failed", zap.Error(vmErr))
			if activeSess != nil {
				activeSess.RecordError("compile Python: " + vmErr.Error())
			}
			return nil, vmErr
		}
		return vmSess, nil
	}
	vmSess, vmErr := NewVMLiveSessionCached(cfg.Code, cachedBytecode)
	if vmErr != nil {
		s.log.Error("LiveStrategyRunner: compile MQL failed", zap.Error(vmErr))
		if activeSess != nil {
			activeSess.RecordError("compile MQL: " + vmErr.Error())
		}
		return nil, vmErr
	}
	return vmSess, nil
}

// saveVMBytecodeCache persists the freshly compiled bytecode so subsequent
// boots can skip recompilation (best-effort: failures only warn).
func (s *StrategyExecutionServer) saveVMBytecodeCache(ctx context.Context, cfg LiveStrategyConfig, vmSess *VMLiveSession) {
	if cfg.StrategyID == "" || s.importedRepo == nil {
		return
	}
	sid, parseErr := uuid.Parse(cfg.StrategyID)
	if parseErr != nil || sid == uuid.Nil {
		return
	}
	bcData, mErr := mql2go.MarshalBytecode(vmSess.strategy.Bytecode())
	if mErr != nil {
		return
	}
	if saveErr := s.importedRepo.SaveBytecode(ctx, sid, bcData); saveErr != nil {
		s.log.Warn("LiveStrategyRunner: save bytecode cache failed", zap.Error(saveErr))
	}
}

// attachHistoryProvider stages the broker order-history provider on the
// session (LIVE-HISTORY-POOL-1) so OrdersHistoryTotal / OrderSelect
// MODE_HISTORY see real closed orders in live mode (applied to the runner
// inside Start).
func (s *StrategyExecutionServer) attachHistoryProvider(cfg LiveStrategyConfig, vmSess *VMLiveSession) {
	if cfg.Mode != modeLive || s.mtHub == nil {
		return
	}
	vmSess.SetHistoryProvider(func(hctx context.Context, from, to int64) ([]sdk.Position, error) {
		fromT := time.Unix(from, 0)
		toT := time.Unix(to, 0)
		if from <= 0 {
			fromT = time.Unix(0, 0)
		}
		if to <= 0 {
			toT = time.Now()
		}
		recs, err := s.mtHub.OrderHistory(hctx, cfg.AccountID, fromT, toT)
		if err != nil {
			return nil, err
		}
		out := make([]sdk.Position, 0, len(recs))
		for _, rec := range recs {
			if rec == nil {
				continue
			}
			p := orderRecordToPosition(rec)
			p.ClosePrice = rec.ClosePrice
			p.CloseTime = rec.CloseTime
			out = append(out, p)
		}
		return out, nil
	})
}

func (s *StrategyExecutionServer) handleTick(
	ctx context.Context, cfg LiveStrategyConfig,
	tick *mthub.TickUpdate, session *Session, firstBar *bool, activeSess *ActiveSession,
) {
	if *session == nil {
		return
	}
	tctx, err := s.buildTickContext(ctx, cfg, tick)
	if err != nil {
		s.log.Warn("LiveStrategyRunner: tick skipped", zap.Error(err))
		if activeSess != nil {
			activeSess.RecordError(err.Error())
		}
		return
	}

	req := &antv1.ExecuteLiveRequest{
		StrategyCode: cfg.Code,
		StrategyId:   cfg.StrategyID,
		RequestType:  antv1.RequestType_REQUEST_TYPE_TICK,
		TickContext:  tctx,
	}
	s.wireSyncDispatch(ctx, cfg, nil, *session, activeSess)
	resp, err := (*session).SendEvent(ctx, req)
	if err != nil {
		s.log.Warn("LiveStrategyRunner: tick request failed", zap.Error(err))
		if activeSess != nil {
			activeSess.RecordError(err.Error())
		}
		_ = (*session).Close()
		*session = nil
		*firstBar = true
		return
	}
	s.dispatchResponse(ctx, cfg, nil, resp, activeSess, sessionSyncDispatched(*session))
}

func (s *StrategyExecutionServer) handleTrade(
	ctx context.Context, cfg LiveStrategyConfig,
	evt *mthub.BrokerTradeEvent, session *Session, firstBar *bool, activeSess *ActiveSession,
) {
	if *session == nil {
		return
	}
	tctx, err := s.buildTradeContext(ctx, cfg, evt)
	if err != nil {
		s.log.Warn("LiveStrategyRunner: trade event skipped", zap.Error(err))
		if activeSess != nil {
			activeSess.RecordError(err.Error())
		}
		return
	}

	req := &antv1.ExecuteLiveRequest{
		StrategyCode: cfg.Code,
		StrategyId:   cfg.StrategyID,
		RequestType:  antv1.RequestType_REQUEST_TYPE_TRADE,
		TradeContext: tctx,
	}
	s.wireSyncDispatch(ctx, cfg, nil, *session, activeSess)
	resp, err := (*session).SendEvent(ctx, req)
	if err != nil {
		s.log.Warn("LiveStrategyRunner: trade request failed", zap.Error(err))
		if activeSess != nil {
			activeSess.RecordError(err.Error())
		}
		_ = (*session).Close()
		*session = nil
		*firstBar = true
		return
	}
	s.dispatchResponse(ctx, cfg, nil, resp, activeSess, sessionSyncDispatched(*session))
}
