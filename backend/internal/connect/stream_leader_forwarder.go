package connect

import (
	"context"
	"encoding/json"

	"anttrader/internal/coordination"
	"google.golang.org/protobuf/encoding/protojson"

	v1 "anttrader/gen/proto"
)

func (s *StreamService) startForwarder(accountID string) {
	if s == nil || s.eventBus == nil || accountID == "" {
		return
	}

	s.forwardersMu.Lock()
	if _, ok := s.forwarders[accountID]; ok {
		s.forwardersMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.forwarders[accountID] = cancel
	s.forwardersMu.Unlock()

	run := func(gctx context.Context) error {
		ps, ch, err := s.eventBus.Subscribe(gctx, accountID)
		if err != nil {
			return err
		}
		defer ps.Close()
		for {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case msg, ok := <-ch:
				if !ok || msg == nil {
					continue
				}
				var env coordination.Envelope
				if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
					continue
				}
				switch env.Type {
				case coordination.EventTypeProfit:
					var ev v1.ProfitUpdateEvent
					if err := protojson.Unmarshal(env.Data, &ev); err != nil {
						continue
					}
					as, ok := s.getAccountStream(accountID)
					if !ok || as == nil {
						continue
					}
					as.setProfitSnapshot(&ev)
				case coordination.EventTypeOrder:
					var ev v1.OrderUpdateEvent
					if err := protojson.Unmarshal(env.Data, &ev); err != nil {
						continue
					}
					as, ok := s.getAccountStream(accountID)
					if !ok || as == nil {
						continue
					}
					as.upsertOpenedOrder(&ev)
				case coordination.EventTypeAccountStatus:
					var ev v1.AccountStatusEvent
					if err := protojson.Unmarshal(env.Data, &ev); err != nil {
						continue
					}
					_ = &ev
				}
			}
		}
	}

	if s.goroutineMgr != nil {
		_, _ = s.goroutineMgr.Spawn("redis-forwarder-"+accountID, func(gctx context.Context) error {
			return run(gctx)
		})
	} else {
		go func() { _ = run(ctx) }()
	}
}

func (s *StreamService) stopForwarder(accountID string) {
	if s == nil || accountID == "" {
		return
	}
	s.forwardersMu.Lock()
	cancel := s.forwarders[accountID]
	delete(s.forwarders, accountID)
	s.forwardersMu.Unlock()
	if cancel != nil {
		cancel()
	}
}
