package connect

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/service"
)

func convertOrderResponse(order *service.OrderResponse, accountID string) *v1.Order {
	o := &v1.Order{
		Ticket:      order.Ticket,
		Symbol:      order.Symbol,
		Type:        order.Type,
		Volume:      order.Volume,
		OpenPrice:   order.Price,
		StopLoss:    order.StopLoss,
		TakeProfit:  order.TakeProfit,
		Profit:      order.Profit,
		Comment:     order.Comment,
		MagicNumber: order.Magic,
		AccountId:   accountID,
	}

	if order.OpenTime != "" {
		if t, err := parseTime(order.OpenTime); err == nil {
			o.OpenTime = timestamppb.New(t)
		}
	}

	return o
}

func convertPositionResponse(pos *service.PositionResponse, accountID string) *v1.Order {
	o := &v1.Order{
		Ticket:      pos.Ticket,
		Symbol:      pos.Symbol,
		Type:        pos.Type,
		Volume:      pos.Volume,
		OpenPrice:   pos.OpenPrice,
		ClosePrice:  pos.CurrentPrice,
		StopLoss:    pos.StopLoss,
		TakeProfit:  pos.TakeProfit,
		Profit:      pos.Profit,
		Swap:        pos.Swap,
		Commission:  pos.Commission,
		Comment:     pos.Comment,
		MagicNumber: pos.Magic,
		AccountId:   accountID,
	}

	if pos.OpenTime != "" {
		if t, err := parseTime(pos.OpenTime); err == nil {
			o.OpenTime = timestamppb.New(t)
		}
	}

	return o
}

func convertHistoryOrderResponse(order *service.HistoryOrderResponse, accountID string) *v1.Order {
	o := &v1.Order{
		Ticket:      order.Ticket,
		Symbol:      order.Symbol,
		Type:        order.Type,
		Volume:      order.Volume,
		OpenPrice:   order.OpenPrice,
		ClosePrice:  order.ClosePrice,
		StopLoss:    order.StopLoss,
		TakeProfit:  order.TakeProfit,
		Profit:      order.Profit,
		Swap:        order.Swap,
		Commission:  order.Commission,
		Comment:     order.Comment,
		MagicNumber: order.Magic,
		AccountId:   accountID,
	}

	if order.OpenTime != "" {
		if t, err := parseTime(order.OpenTime); err == nil {
			o.OpenTime = timestamppb.New(t)
		}
	}
	if order.CloseTime != "" {
		if t, err := parseTime(order.CloseTime); err == nil {
			o.CloseTime = timestamppb.New(t)
		}
	}

	return o
}

func parseTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05Z", s)
}
