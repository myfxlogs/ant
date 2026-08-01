package strategy

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

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
	*bars = append(*bars, liveBar{
		open:     bar.Open.String(),
		high:     bar.High.String(),
		low:      bar.Low.String(),
		close:    bar.Close.String(),
		volume:   strconv.FormatFloat(bar.Volume, 'f', -1, 64),
		openTime: bar.OpenTime,
	})
	if len(*bars) > maxContextBars {
		*bars = (*bars)[len(*bars)-maxContextBars:]
	}

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

	var lctx *antv1.LiveStrategyContext
	if *firstBar {
		lctx = s.buildLiveContext(ctx, cfg, *bars, extraBars)
	} else {
		lctx = s.buildDeltaContext(ctx, cfg, *bars, extraBars)
	}

	req := &antv1.ExecuteLiveRequest{
		StrategyCode: cfg.Code,
		StrategyId:   cfg.StrategyID,
		RequestType:  antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:   lctx,
	}
	reqBytes, marshalErr := proto.Marshal(req)
	if marshalErr != nil {
		s.log.Error("LiveStrategyRunner: proto marshal failed", zap.Error(marshalErr))
		return
	}

	var respBytes []byte
	var err error
	if *firstBar {
		vmSess, vmErr := s.initVMSession(ctx, cfg, activeSess)
		if vmErr != nil {
			return
		}
		*session = vmSess
		respBytes, err = (*session).Start(ctx, reqBytes)
		*firstBar = false
	} else {
		if *session == nil {
			s.log.Error("LiveStrategyRunner: session lost before bar event")
			return
		}
		respBytes, err = (*session).SendEvent(ctx, reqBytes)
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
	s.dispatchFromBytes(ctx, cfg, bar, respBytes, activeSess)
}

func (s *StrategyExecutionServer) initVMSession(ctx context.Context, cfg LiveStrategyConfig, activeSess *ActiveSession) (Session, error) {
	var cachedBytecode []byte
	if cfg.StrategyID != "" && s.importedRepo != nil {
		if sid, parseErr := uuid.Parse(cfg.StrategyID); parseErr == nil {
			cachedBytecode, _ = s.importedRepo.GetBytecode(ctx, sid)
		}
	}
	vmSess, vmErr := NewVMLiveSessionCached(cfg.Code, cachedBytecode)
	if vmErr != nil {
		s.log.Error("LiveStrategyRunner: compile MQL failed", zap.Error(vmErr))
		if activeSess != nil {
			activeSess.RecordError("compile MQL: " + vmErr.Error())
		}
		return nil, vmErr
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
	tctx := s.buildTickContext(ctx, cfg, tick)

	req := &antv1.ExecuteLiveRequest{
		StrategyCode: cfg.Code,
		StrategyId:   cfg.StrategyID,
		RequestType:  antv1.RequestType_REQUEST_TYPE_TICK,
		TickContext:  tctx,
	}
	reqBytes, marshalErr := proto.Marshal(req); if marshalErr != nil { s.log.Warn("LiveStrategyRunner: tick proto marshal failed", zap.Error(marshalErr)); return }
	respBytes, err := (*session).SendEvent(ctx, reqBytes)
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
	s.dispatchFromBytes(ctx, cfg, nil, respBytes, activeSess)
}

func (s *StrategyExecutionServer) handleTrade(
	ctx context.Context, cfg LiveStrategyConfig,
	evt *mthub.BrokerTradeEvent, session *Session, firstBar *bool, activeSess *ActiveSession,
) {
	if *session == nil {
		return
	}
	tctx := s.buildTradeContext(ctx, cfg, evt)

	req := &antv1.ExecuteLiveRequest{
		StrategyCode: cfg.Code,
		StrategyId:   cfg.StrategyID,
		RequestType:  antv1.RequestType_REQUEST_TYPE_TRADE,
		TradeContext: tctx,
	}
	reqBytes, marshalErr := proto.Marshal(req); if marshalErr != nil { s.log.Warn("LiveStrategyRunner: trade proto marshal failed", zap.Error(marshalErr)); return }
	respBytes, err := (*session).SendEvent(ctx, reqBytes)
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
	s.dispatchFromBytes(ctx, cfg, nil, respBytes, activeSess)
}
