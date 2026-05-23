package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/service"
)

type EconomicDataService struct {
	svc *service.EconomicCalendarService
}

func NewEconomicDataService(svc *service.EconomicCalendarService) *EconomicDataService {
	return &EconomicDataService{svc: svc}
}

func (s *EconomicDataService) ListEconomicCalendarEvents(ctx context.Context, req *connect.Request[v1.ListEconomicCalendarEventsRequest]) (*connect.Response[v1.ListEconomicCalendarEventsResponse], error) {
	if s.svc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("economic calendar service not available"))
	}
	events, err := s.svc.GetCalendar(ctx, &service.EconomicCalendarQuery{
		From: req.Msg.From, To: req.Msg.To, Country: req.Msg.Country,
		Symbol: req.Msg.Symbol, Importance: req.Msg.Importance, Lang: req.Msg.Lang,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&v1.ListEconomicCalendarEventsResponse{Events: economicCalendarEvents(events)}), nil
}

func (s *EconomicDataService) ListEconomicIndicators(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.ListEconomicIndicatorsResponse], error) {
	_ = req
	if s.svc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("economic calendar service not available"))
	}
	indicators, err := s.svc.GetKeyIndicators(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&v1.ListEconomicIndicatorsResponse{Indicators: economicIndicators(indicators)}), nil
}

func economicCalendarEvents(events []*service.EconomicCalendarEvent) []*v1.EconomicCalendarEvent {
	out := make([]*v1.EconomicCalendarEvent, 0, len(events))
	for _, e := range events {
		out = append(out, &v1.EconomicCalendarEvent{Date: e.Date, Time: e.Time, Country: e.Country, Event: e.Event, LocalizedEvent: e.LocalizedEvent, Impact: e.Impact, Actual: e.Actual, Previous: e.Previous, Estimate: e.Estimate, Unit: e.Unit, Currency: e.Currency, Timestamp: e.Timestamp})
	}
	return out
}

func economicIndicators(indicators []*service.KeyIndicator) []*v1.EconomicIndicator {
	out := make([]*v1.EconomicIndicator, 0, len(indicators))
	for _, ind := range indicators {
		item := &v1.EconomicIndicator{Code: ind.Code, SeriesId: ind.SeriesID, Name: ind.Name, Units: ind.Units, Frequency: ind.Frequency, LatestDate: ind.LatestDate, LatestValue: ind.LatestValue}
		for _, h := range ind.History {
			item.History = append(item.History, &v1.EconomicIndicatorPoint{Date: h.Date, Value: h.Value})
		}
		out = append(out, item)
	}
	return out
}
