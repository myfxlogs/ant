package strategy

import (
	"context"
	"strconv"

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
		resp, err = (*session).Start(ctx, req)
		*firstBar = false
	} else {
		if *session == nil {
			s.log.Error("LiveStrategyRunner: session lost before bar event")
			return
		}
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
	s.dispatchResponse(ctx, cfg, bar, resp, activeSess)
}

func (s *StrategyExecutionServer) initVMSession(ctx context.Context, cfg LiveStrategyConfig, activeSess *ActiveSession) (Session, error) {
	var cachedBytecode []byte
	if cfg.StrategyID != "" && s.importedRepo != nil {
		if sid, parseErr := uuid.Parse(cfg.StrategyID); parseErr == nil {
			cachedBytecode, _ = s.importedRepo.GetBytecode(ctx, sid)
		}
	}
	var vmSess *VMLiveSession
	var vmErr error
	if sdk.IsPython(cfg.Code) {
		vmSess, vmErr = NewPythonVMLiveSessionCached(cfg.Code, cachedBytecode)
		if vmErr != nil {
			s.log.Error("LiveStrategyRunner: compile Python failed", zap.Error(vmErr))
			if activeSess != nil {
				activeSess.RecordError("compile Python: " + vmErr.Error())
			}
			return nil, vmErr
		}
	} else {
		vmSess, vmErr = NewVMLiveSessionCached(cfg.Code, cachedBytecode)
		if vmErr != nil {
			s.log.Error("LiveStrategyRunner: compile MQL failed", zap.Error(vmErr))
			if activeSess != nil {
				activeSess.RecordError("compile MQL: " + vmErr.Error())
			}
			return nil, vmErr
		}
	}
	if activeSess != nil {
		vmSess.SetDiag(activeSess.diag)
	}
	if cfg.StrategyID != "" && s.importedRepo != nil {
		if sid, parseErr := uuid.Parse(cfg.StrategyID); parseErr == nil && sid != uuid.Nil {
			if bcData, mErr := mql2go.MarshalBytecode(vmSess.strategy.Bytecode()); mErr == nil {
				if saveErr := s.importedRepo.SaveBytecode(ctx, sid, bcData); saveErr != nil {
					s.log.Warn("LiveStrategyRunner: save bytecode cache failed", zap.Error(saveErr))
				}
			}
		}
	}
	return vmSess, nil
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
	s.dispatchResponse(ctx, cfg, nil, resp, activeSess)
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
	s.dispatchResponse(ctx, cfg, nil, resp, activeSess)
}
