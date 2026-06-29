package strategy

import (
	"context"
	"strconv"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/mthub"
	"anttrader/tools/mql2go"
	"anttrader/tools/mql2go/interp"
)

func (s *StrategyExecutionServer) handleBar(
	ctx context.Context, cfg LiveStrategyConfig, wasm *WasmExecutor,
	bar *mthub.BarUpdate, bars *[]liveBar,
	session **LiveSession, firstBar *bool, activeSess *ActiveSession,
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

	var lctx *antv1.LiveStrategyContext
	if *firstBar {
		lctx = s.buildLiveContext(ctx, cfg, *bars)
	} else {
		lctx = s.buildDeltaContext(ctx, cfg, *bars)
	}

	req := &antv1.ExecuteLiveRequest{
		StrategyCode: cfg.Code,
		RequestType:  antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:   lctx,
	}
	reqBytes, _ := proto.Marshal(req)

	var respBytes []byte
	var err error
	if *firstBar {
		if isMQLStrategy(cfg.Code) {
			ir, irErr := mql2go.CompileToIR(cfg.Code)
			if irErr != nil {
				s.log.Error("LiveStrategyRunner: compile MQL to IR failed", zap.Error(irErr))
				return
			}
			*session = NewInterpLiveSession(wasm, interp.SerializeIR(ir), s.log)
		} else {
			*session = NewLiveSession(wasm, cfg.Code, s.log)
		}
		respBytes, err = (*session).Start(ctx, reqBytes)
		*firstBar = false
	} else {
		if *session == nil {
			s.log.Error("LiveStrategyRunner: session lost before bar event")
			return
		}
		respBytes, err = (*session).SendBar(reqBytes)
	}
	if err != nil {
		s.log.Error("LiveStrategyRunner: bar request failed", zap.Error(err))
		if activeSess != nil {
			activeSess.RecordError(err.Error())
		}
		if *session != nil {
			(*session).Close()
		}
		*session = nil
		*firstBar = true
		return
	}
	s.dispatchFromBytes(ctx, cfg, bar, respBytes, activeSess)
}

func (s *StrategyExecutionServer) handleTick(
	ctx context.Context, cfg LiveStrategyConfig,
	tick *mthub.TickUpdate, session **LiveSession, firstBar *bool, activeSess *ActiveSession,
) {
	if *session == nil {
		return
	}
	tctx := s.buildTickContext(ctx, cfg, tick)

	req := &antv1.ExecuteLiveRequest{
		StrategyCode: cfg.Code,
		RequestType:  antv1.RequestType_REQUEST_TYPE_TICK,
		TickContext:  tctx,
	}
	reqBytes, _ := proto.Marshal(req)
	respBytes, err := (*session).SendBar(reqBytes)
	if err != nil {
		s.log.Warn("LiveStrategyRunner: tick request failed", zap.Error(err))
		if activeSess != nil {
			activeSess.RecordError(err.Error())
		}
		(*session).Close()
		*session = nil
		*firstBar = true
		return
	}
	s.dispatchFromBytes(ctx, cfg, nil, respBytes, activeSess)
}

func (s *StrategyExecutionServer) handleTrade(
	ctx context.Context, cfg LiveStrategyConfig,
	evt *mthub.BrokerTradeEvent, session **LiveSession, firstBar *bool, activeSess *ActiveSession,
) {
	if *session == nil {
		return
	}
	tctx := s.buildTradeContext(ctx, cfg, evt)

	req := &antv1.ExecuteLiveRequest{
		StrategyCode:  cfg.Code,
		RequestType:   antv1.RequestType_REQUEST_TYPE_TRADE,
		TradeContext:  tctx,
	}
	reqBytes, _ := proto.Marshal(req)
	respBytes, err := (*session).SendBar(reqBytes)
	if err != nil {
		s.log.Warn("LiveStrategyRunner: trade request failed", zap.Error(err))
		if activeSess != nil {
			activeSess.RecordError(err.Error())
		}
		(*session).Close()
		*session = nil
		*firstBar = true
		return
	}
	s.dispatchFromBytes(ctx, cfg, nil, respBytes, activeSess)
}
